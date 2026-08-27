package repository

import (
	"context"
	"database/sql"

	"github.com/ove4lo/ship-cargo-service/internal/model"
)

// BookingRepository represents layer between business logic and the database for booking.
type BookingRepository struct {
	db *sql.DB
}

// NewBookingRepository creates an instance of BookingRepository and passes a dependency (database connection) to it.
func NewBookingRepository(db *sql.DB) *BookingRepository {
	return &BookingRepository{db: db}
}

// GetByIdempotencyKey retrieves a booking by its unique idempotency key to prevent duplicate processing.
func (r *BookingRepository) GetByIdempotencyKey(ctx context.Context, key string) (*model.Booking, error) {
	var b model.Booking
	err := r.db.QueryRowContext(ctx,
		`SELECT id, voyage_id, user_id, priority, status, idempotency_key, created_at
		FROM bookings WHERE idempotency_key = $1`,
		key,
	).Scan(&b.ID, &b.VoyageID, &b.UserID, &b.Priority, &b.Status, &b.IdempotencyKey, &b.CreatedAt)
	if err != nil {
		return nil, err
	}
	return &b, nil
}

// CreateTx inserts a new booking record into the database within an active transaction.
func (r *BookingRepository) CreateTx(ctx context.Context, tx model.Tx, booking *model.Booking) error {
	return tx.QueryRowContext(ctx,
		`INSERT INTO bookings (voyage_id, user_id, priority, status, idempotency_key)
		 VALUES ($1, $2, $3, $4, $5)
		 RETURNING id, created_at`,
		booking.VoyageID, booking.UserID, booking.Priority, booking.Status, booking.IdempotencyKey,
	).Scan(&booking.ID, &booking.CreatedAt)
}

// CreateItemTx inserts a specific cargo item tied to a booking within an active database transaction.
func (r *BookingRepository) CreateItemTx(ctx context.Context, tx model.Tx, item *model.BookingItem) error {
	return tx.QueryRowContext(ctx,
		`INSERT INTO booking_items (booking_id, description, weight_kg, volume_m3, status)
		 VALUES ($1, $2, $3, $4, $5)
		 RETURNING id`,
		item.BookingID, item.Description, item.WeightKg, item.VolumeM3, item.Status,
	).Scan(&item.ID)
}

// BeginTx starts a new database transaction for executing atomic booking operations.
func (r *BookingRepository) BeginTx(ctx context.Context) (model.Tx, error) {
	return r.db.BeginTx(ctx, nil)
}