package service

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/ove4lo/ship-cargo-service/internal/model"
	"github.com/ove4lo/ship-cargo-service/internal/metrics"
)

var (
	// ErrVoyageNotAvailable is returned when a voyage is not in a mockable or loadable status.
	ErrVoyageNotAvailable = errors.New("voyage isn't available for booking")
	// ErrNoCapacity is returned when none of the requested cargo items can fit into the remaining voyage capacity.
	ErrNoCapacity = errors.New("no capacity available")
	// ErrVoyageLocked is returned when another process is concurrently executing a booking operation on the same voyage.
	ErrVoyageLocked = errors.New("voyage is being booked by another user")
)

// VoyageProvider defines the data layer interactions required to read and 
// modify voyage records during a cargo booking process.
type VoyageProvider interface {
	GetByIDForUpdate(ctx context.Context, tx model.Tx, id string) (*model.Voyage, error)
	UpdateReservedTx(ctx context.Context, tx model.Tx, id string, weightKg, volumeM3 float64) error
}

// BookingCreator defines the persistence operations needed to validate idempotency 
// and store booking aggregates within a transaction.
type BookingCreator interface {
	GetByIdempotencyKey(ctx context.Context, key string) (*model.Booking, error)
	BeginTx(ctx context.Context) (model.Tx, error)
	CreateTx(ctx context.Context, tx model.Tx, item *model.Booking) error
	CreateItemTx(ctx context.Context, tx model.Tx, item *model.BookingItem) error
}

// Locker defines the behavioral contract for managing distributed locks 
// to prevent concurrent race conditions on shared domain resources.
type Locker interface {
	Acquire(ctx context.Context, key string, ttl time.Duration) (string, error)
	Release(ctx context.Context, key string, token string) error
}

// EventPublisher defines the contract for broadcasting booking-related domain events 
// to external message brokers like RabbitMQ.
type EventPublisher interface {
	PublishBookingEvent(ctx context.Context, event BookingEvent) error 
}

// BookingEvent encapsulates the domain event metadata compiled immediately 
// after a cargo booking transaction is successfully committed.
type BookingEvent struct {
	BookingID  string
	VoyageID   string
	UserID     string
	Status     string
	ItemCount  int
	OccurredAt time.Time
}

// BookingService orchestrates the domain business logic for cargo space allocation.
type BookingService struct {
	bookingRepo BookingCreator
	voyageRepo  VoyageProvider
	lock        Locker
	publisher   EventPublisher
}

// NewBookingService constructs a new BookingService with required repositories and lock dependencies.
func NewBookingService(
	bookingRepo BookingCreator, 
	voyageRepo VoyageProvider,
	lock Locker,
	publisher EventPublisher,
) *BookingService {
	return &BookingService{
		bookingRepo: bookingRepo,
		voyageRepo:  voyageRepo,
		lock:        lock,
		publisher:   publisher,
	}
}

// BookingRequest defines the boundaries of incoming data required to create a cargo booking.
type BookingRequest struct {
	VoyageID       string
	UserID         string
	Priority       model.BookingPriority
	IdempotencyKey string
	Items          []BookingItemRequest
}

// BookingItemRequest encapsulates details for an individual cargo piece inside a larger booking request.
type BookingItemRequest struct {
	Description string
	WeightKg    float64
	VolumeM3    float64
}

// CreateBooking safely checks idempotency, acquires a distributed lock, assesses remaining space, 
// and stores both the booking record and item records within a database transaction.
func (s *BookingService) CreateBooking(ctx context.Context, req BookingRequest) (*model.Booking, error) {
	// 1. Input parameter validation
	if req.VoyageID == "" || req.UserID == "" || len(req.Items) == 0 {
		return nil, fmt.Errorf("invalid booking request: missing required fields or items")
	}

	// 2. Idempotency check.
	existing, err := s.bookingRepo.GetByIdempotencyKey(ctx, req.IdempotencyKey)
	if err == nil {
		return existing, nil
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return nil, fmt.Errorf("check idempotency: %w", err)
	}

	// 3. Track the total duration of the booking operation upon function exit.
	start := time.Now()
	defer func() {
		metrics.BookingDuration.Observe(time.Since(start).Seconds())
	}()

	// 4. Locking in the voyage.
	lockKey := "lock:voyage:" + req.VoyageID
	token, err := s.lock.Acquire(ctx, lockKey, 5*time.Second)
	if err != nil {
		metrics.LockConflicts.Inc()
		return nil, ErrVoyageLocked
	}
	defer s.lock.Release(ctx, lockKey, token)

	// 5. Starting the transaction.
	tx, err := s.bookingRepo.BeginTx(ctx)
	if err != nil {
		return nil, fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback()

	// 6. Receive a flight with a block.
	voyage, err := s.voyageRepo.GetByIDForUpdate(ctx, tx, req.VoyageID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, fmt.Errorf("voyage not found")
		}
		return nil, fmt.Errorf("get voyage: %w", err)
	}

	// 7. Checking flight status.
	if voyage.Status != model.VoyageStatusPlanned && voyage.Status != model.VoyageStatusLoading {
		return nil, ErrVoyageNotAvailable
	}
	
	// 8. Placing positions.
	freeWeight := voyage.FreeWeightKg()
	freeVolume := voyage.FreeVolumeM3()
	var addedWeight, addedVolume float64
	var placedCount int

	items := make([]model.BookingItem, len(req.Items))
	for i, ri := range req.Items {
		items[i] = model.BookingItem{
			Description: ri.Description,
			WeightKg:    ri.WeightKg,
			VolumeM3:    ri.VolumeM3,
		}

		fitsWeight := addedWeight+ri.WeightKg <= freeWeight
		fitsVolume := addedVolume+ri.VolumeM3 <= freeVolume

		if fitsWeight && fitsVolume {
			items[i].Status = model.ItemStatusPlaced
			addedWeight += ri.WeightKg
			addedVolume += ri.VolumeM3
			placedCount++
		} else {
			items[i].Status = model.ItemStatusWaitlisted
		}
	}

	// 9. Determine the booking status.
	var bookingStatus model.BookingStatus
	switch {
	case placedCount == 0:
		return nil, ErrNoCapacity
	case placedCount == len(items):
		bookingStatus = model.BookingStatusConfirmed
	default:
		bookingStatus = model.BookingStatusPartial
	}

	metrics.BookingsTotal.WithLabelValues(string(bookingStatus)).Inc()

	// 10. Save the reservation.
	booking := &model.Booking{
		VoyageID:       req.VoyageID,
		UserID:         req.UserID,
		Priority:       req.Priority,
		Status:         bookingStatus,
		IdempotencyKey: req.IdempotencyKey,
	}

	if err := s.bookingRepo.CreateTx(ctx, tx, booking); err != nil {
		return nil, fmt.Errorf("create booking: %w", err)
	}

	// 11. Save positions.
	for i := range items {
		items[i].BookingID = booking.ID
		if err := s.bookingRepo.CreateItemTx(ctx, tx, &items[i]); err != nil {
			return nil, fmt.Errorf("create item: %w", err)
		}
	}

	// 12. Update flight capacity.
	if err := s.voyageRepo.UpdateReservedTx(ctx, tx,
		voyage.ID,
		voyage.ReservedWeightKg+addedWeight,
		voyage.ReservedVolumeM3+addedVolume,
	); err != nil {
		return nil, fmt.Errorf("update reserved: %w", err)
	}

	// 13. Commit the transaction.
	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("commit: %w", err)
	}

	// 14. Publish the event (after the commit—if it fails, the reservation is already saved).
	_ = s.publisher.PublishBookingEvent(ctx, BookingEvent{
		BookingID:  booking.ID,
		VoyageID:   booking.VoyageID,
		UserID:     booking.UserID,
		Status:     string(booking.Status),
		ItemCount:  len(items),
		OccurredAt: booking.CreatedAt,
	})

	booking.Items = items
	return booking, nil
}