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

// Tests

// newTestService constructs a BookingService initialized with in-memory fake dependencies for isolated unit testing.
func newTestService(voyage *model.Voyage, existingBooking *model.Booking, locked bool) *BookingService {
	return NewBookingService(
		&fakeBookingCreator{existing: existingBooking},
		&fakeVoyageProvider{voyage: voyage},
		&fakeLocker{locked: locked},
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