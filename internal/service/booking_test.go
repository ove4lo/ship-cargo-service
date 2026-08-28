package service

import (
	"context"
	"database/sql"
	"errors"
	"testing"
	"time"

	"github.com/ove4lo/ship-cargo-service/internal/model"
)

// Fakes

type fakeTx struct{}

func (f *fakeTx) QueryRowContext(_ context.Context, _ string, _ ... any) *sql.Row {
	return nil
}

func (f *fakeTx) ExecContext(_ context.Context, _ string, _ ... any) (sql.Result, error) {
	return nil, nil
}

func (f *fakeTx) Commit() error {
	return nil
}

func (f *fakeTx) Rollback() error {
	return nil
}

// fakeLocker simulates a distributed lock manager using an in-memory boolean flag.
type fakeLocker struct {
	locked bool
}

// Acquire simulates capturing a Redis lock. It returns an error if the lock flag is true.
func (f *fakeLocker) Acquire(_ context.Context, _ string, _ time.Duration) (string, error) {
	if f.locked {
		return "", errors.New("already locked")
	}
	return "fake-token", nil
}

// Release simulates releasing a Redis lock and always returns success in this mock.
func (f *fakeLocker) Release(_ context.Context, _ string, _ string) error {
	return nil
}

// fakeVoyageProvider simulates database operations for voyages using a pre-configured model.
type fakeVoyageProvider struct {
	voyage *model.Voyage
}

// GetByIDForUpdate simulates fetching a voyage with row-locking. Returns sql.ErrNoRows if no voyage is set.
func (f *fakeVoyageProvider) GetByIDForUpdate(_ context.Context, _ model.Tx, _ string) (*model.Voyage, error) {
	if f.voyage == nil {
		return nil, sql.ErrNoRows
	}
	return f.voyage, nil
}

// UpdateReservedTx simulates modifying a voyage's filled capacities directly in memory.
func (f *fakeVoyageProvider) UpdateReservedTx(_ context.Context, _ model.Tx, _ string, weightKg, volumeM3 float64) error {
	f.voyage.ReservedWeightKg = weightKg
	f.voyage.ReservedVolumeM3 = volumeM3
	return nil
}

// fakeBookingCreator simulates transaction management and persistence operations for cargo bookings.
type fakeBookingCreator struct {
	existing *model.Booking
}

// GetByIdempotencyKey simulates an idempotency check, returning a predefined booking if the key matches.
func (f *fakeBookingCreator) GetByIdempotencyKey(_ context.Context, key string) (*model.Booking, error) {
	if f.existing != nil && f.existing.IdempotencyKey == key {
		return f.existing, nil
	}
	return nil, sql.ErrNoRows
}

// BeginTx simulates starting a SQL transaction. It returns nil because in-memory fakes don't require atomic rollbacks.
func (f *fakeBookingCreator) BeginTx(_ context.Context) (model.Tx, error) {
	return &fakeTx{}, nil
}

// CreateTx simulates inserting a booking header, automatically generating a fake ID and timestamp.
func (f *fakeBookingCreator) CreateTx(_ context.Context, _ model.Tx, booking *model.Booking) error {
	booking.ID = "booking-001"
	booking.CreatedAt = time.Now()
	return nil
}

// CreateItemTx simulates inserting an individual cargo item, generating an ID based on its description.
func (f *fakeBookingCreator) CreateItemTx(_ context.Context, _ model.Tx, item *model.BookingItem) error {
	item.ID = "item-" + item.Description
	return nil
}

// fakePublisher simulates broadcasting domain events into an in-memory test black hole.
type fakePublisher struct{}

// PublishBookingEvent simulates a successful event push to RabbitMQ by doing nothing and returning nil.
func (f *fakePublisher) PublishBookingEvent(_ context.Context, _ BookingEvent) error {
	return nil
}

// Tests

// newTestService constructs a BookingService initialized with in-memory fake dependencies for isolated unit testing.
func newTestService(voyage *model.Voyage, existingBooking *model.Booking, locked bool) *BookingService {
	return NewBookingService(
		&fakeBookingCreator{existing: existingBooking},
		&fakeVoyageProvider{voyage: voyage},
		&fakeLocker{locked: locked},
		&fakePublisher{},
	)
}

// baseVoyage creates a template model.Voyage with empty capacities for reuse in test scenarios.
func baseVoyage() *model.Voyage {
	return &model.Voyage{
		ID: "voyage-001",
		Status: model.VoyageStatusPlanned,
		MaxWeightKg: 10000,
		MaxVolumeM3: 100,
		ReservedWeightKg: 0,
		ReservedVolumeM3: 0,
	}
}

// baseRequest generates a mock BookingRequest payload containing a single valid cargo item.
func baseRequest() BookingRequest {
	return BookingRequest{
		VoyageID: "voyage-001",
		UserID: "user-001",
		Priority: model.PriorityNormal,
		IdempotencyKey: "key-001",
		Items: []BookingItemRequest{
			{Description: "Spares", WeightKg: 3000, VolumeM3: 30},
		},
	}
}

// TestCreateBooking_AllItemsPlaced verifies that a booking is successfully confirmed 
// and all items are marked as placed when the voyage has sufficient capacity.
func TestCreateBooking_AllItemsPlaced(t *testing.T) {
	svc := newTestService(baseVoyage(), nil, false)

	booking, err := svc.CreateBooking(context.Background(), baseRequest())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if booking.Status != model.BookingStatusConfirmed {
		t.Errorf("want status %s, got %s", model.BookingStatusConfirmed, booking.Status)
	}

	if len(booking.Items) != 1 {
		t.Fatalf("want 1 item, got %d", len(booking.Items))
	}

	if booking.Items[0].Status != model.ItemStatusPlaced {
		t.Errorf("want item status %s, got %s", model.ItemStatusPlaced, booking.Items[0].Status)
	}
}

// TestCreateBooking_PartialPlacement verifies that when a voyage has limited remaining capacity,
// items that fit are marked as placed, while items exceeding the limits are put on the waitlist.
func TestCreateBooking_PartialPlacement(t *testing.T) {
	voyage := baseVoyage()
	voyage.ReservedWeightKg = 8000 // 2000 kg remaining out of 10000

	req := baseRequest()
	req.Items = []BookingItemRequest{
		{Description: "Light load", WeightKg: 1000, VolumeM3: 10},
		{Description: "Heavy load", WeightKg: 5000, VolumeM3: 20},
	}

	svc := newTestService(voyage, nil, false)
	booking, err := svc.CreateBooking(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if booking.Status != model.BookingStatusPartial {
		t.Errorf("want status %s, got %s", model.BookingStatusPartial, booking.Status)
	}

	if booking.Items[0].Status != model.ItemStatusPlaced {
		t.Errorf("first item: want %s, got %s", model.ItemStatusPlaced, booking.Items[0].Status)
	}

	if booking.Items[1].Status != model.ItemStatusWaitlisted {
		t.Errorf("second item: want %s, got %s", model.ItemStatusWaitlisted, booking.Items[1].Status)
	}
}

// TestCreateBooking_NoCapacity asserts that an error is returned if none of the requested 
// cargo items can fit into the voyage's remaining weight capacity.
func TestCreateBooking_NoCapacity(t *testing.T) {
	voyage := baseVoyage()
	voyage.ReservedWeightKg = 10000 // completely filled

	svc := newTestService(voyage, nil, false)
	_, err := svc.CreateBooking(context.Background(), baseRequest())

	if !errors.Is(err, ErrNoCapacity) {
		t.Errorf("want ErrNoCapacity, got %v", err)
	}
}

// TestCreateBooking_VoyageDeparted ensures that cargo space cannot be allocated 
// if the targeted voyage has already departed.
func TestCreateBooking_VoyageDeparted(t *testing.T) {
	voyage := baseVoyage()
	voyage.Status = model.VoyageStatusDeparted

	svc := newTestService(voyage, nil, false)
	_, err := svc.CreateBooking(context.Background(), baseRequest())

	if !errors.Is(err, ErrVoyageNotAvailable) {
		t.Errorf("want ErrVoyageNotAvailable, got %v", err)
	}
}

// TestCreateBooking_Idempotency verifies that sending a request with a previously processed 
// idempotency key returns the existing booking record without calculating limits again.
func TestCreateBooking_Idempotency(t *testing.T) {
	existing := &model.Booking{
		ID: "existing-001",
		IdempotencyKey: "key-001",
		Status: model.BookingStatusConfirmed,
	}

	svc := newTestService(baseVoyage(), existing, false)
	booking, err := svc.CreateBooking(context.Background(), baseRequest())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if booking.ID != "existing-001" {
		t.Errorf("want existing booking ID, got %s", booking.ID)
	}
}

// TestCreateBooking_VoyageLocked checks that the service rejects requests with an appropriate error 
// if a concurrent booking process holds the distributed lock for the specified voyage.
func TestCreateBooking_VoyageLocked(t *testing.T) {
	svc := newTestService(baseVoyage(), nil, true) // locked = true

	_, err := svc.CreateBooking(context.Background(), baseRequest())

	if !errors.Is(err, ErrVoyageLocked) {
		t.Errorf("want ErrVoyageLocked, got %v", err)
	}
}

// TestCreateBooking_VolumeLimitOnly validates that items exceeding the remaining volume constraints 
// are correctly waitlisted, even if the voyage has ample weight capacity left.
func TestCreateBooking_VolumeLimitOnly(t *testing.T) {
	voyage := baseVoyage()
	voyage.ReservedVolumeM3 = 95 // 5 m^3 remaining

	req := baseRequest()
	req.Items = []BookingItemRequest{
		{Description: "Compact", WeightKg: 100, VolumeM3: 3},
		{Description: "Bulky", WeightKg: 100, VolumeM3: 50},
	}

	svc := newTestService(voyage, nil, false)
	booking, err := svc.CreateBooking(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if booking.Status != model.BookingStatusPartial {
		t.Errorf("want status %s, got %s", model.BookingStatusPartial, booking.Status)
	}

	if booking.Items[0].Status != model.ItemStatusPlaced {
		t.Errorf("compact item: want %s, got %s", model.ItemStatusPlaced, booking.Items[0].Status)
	}

	if booking.Items[1].Status != model.ItemStatusWaitlisted {
		t.Errorf("bulky item: want %s, got %s", model.ItemStatusWaitlisted, booking.Items[1].Status)
	}
}