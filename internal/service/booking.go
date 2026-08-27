package service

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/ove4lo/ship-cargo-service/internal/model"
	"github.com/ove4lo/ship-cargo-service/internal/repository"
)

var (
	ErrorVoyageNotAvailable = errors.New("voyage isn't available for booking")
	ErrNoCapacity = errors.New("no capacity available")
)

type BookingService struct {
	bookingRepo *repository.BookingRepository
	voyageRepo *repository.VoyageRepository
}

func NewBookingService(bookingRepo *repository.BookingRepository, voyageRepo *repository.VoyageRepository) *BookingService {
	return &BookingService{
		bookingRepo: bookingRepo,
		voyageRepo: voyageRepo,
	}
}

type BookingRequest struct {
	VoyageID string
	UserID string
	Priority model.BookingPriority
	IdempotencyKey string
	Items []BookingItemRequest
}

type BookingItemRequest struct {
	Description string
	WeightKg float64
	VolumeM3 float64
}

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

	// 3. Starting the transaction.
	tx, err := s.bookingRepo.BeginTx(ctx)
	if err != nil {
		return nil, fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback()

	// 4. Receive a flight with a block.
	voyage, err := s.voyageRepo.GetByIDForUpdate(ctx, tx, req.VoyageID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, fmt.Errorf("voyage not found")
		}
		return nil, fmt.Errorf("get voyage: %w", err)
	}

	// 5. Checking flight status.
	if voyage.Status != model.VoyageStatusPlanned && voyage.Status != model.VoyageStatusLoading {
		return nil, ErrorVoyageNotAvailable
	}
	
	// 6. Placing positions.
	freeWeight := voyage.FreeWeightKg()
	freeVolume := voyage.FreeVolumeM3()
	var addedWeight, addedVolume float64
	var placedCount int

	items := make([]model.BookingItem, len(req.Items))
	for i, ri := range req.Items {
		items[i] = model.BookingItem{
			Description: ri.Description,
			WeightKg: ri.WeightKg,
			VolumeM3: ri.VolumeM3,
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

	// 7. Determine the booking status.
	var bookingStatus model.BookingStatus
	switch {
	case placedCount == 0:
		return  nil, ErrNoCapacity
	case placedCount == len(items):
		bookingStatus = model.BookingStatusConfirmed
	default:
		bookingStatus = model.BookingStatusPartial
	}

	// 8. Save the reservation
	booking := &model.Booking{
		VoyageID: req.VoyageID,
		UserID: req.UserID,
		Priority: req.Priority,
		Status: bookingStatus,
		IdempotencyKey: req.IdempotencyKey,
	}

	if err := s.bookingRepo.CreateTx(ctx, tx, booking); err != nil {
		return nil, fmt.Errorf("create booking: %w", err)
	}

	// 9. Save positions
	for i := range items {
		items[i].BookingID = booking.ID
		if err := s.bookingRepo.CreateItemTx(ctx, tx, &items[i]); err != nil {
			return nil, fmt.Errorf("create item: %w", err)
		}
	}

	// 10. Update flight capacity
	if err := s.voyageRepo.UpdateReservedTx(ctx, tx,
		voyage.ID,
		voyage.ReservedWeightKg+addedWeight,
		voyage.ReservedVolumeM3+addedVolume,
	); err != nil {
		return nil, fmt.Errorf("update reserved: %w", err)
	}

	// 11. Commit the transaction
	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("commit: %w", err)
	}

	booking.Items = items
	return booking, nil
}