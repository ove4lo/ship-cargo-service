package repository

import (
	"context"
	"database/sql"

	"github.com/ove4lo/ship-cargo-service/internal/model"
)

// VoyageRepository represents layer between business logic and the database for voyage.
type VoyageRepository struct {
	db *sql.DB
}

// NewVoyageRepository creates an instance of VoyageRepository and passes a dependency (database connection) to it.
func NewVoyageRepository(db *sql.DB) *VoyageRepository {
	return &VoyageRepository{db: db}
}

// Create inserts a new voyage record and returns generated fields like ID, status, and timestamps.
func (r *VoyageRepository) Create(ctx context.Context, voyage *model.Voyage) error {
	return r.db.QueryRowContext(ctx,
		`INSERT INTO voyages (vessel_id, route, departure_date)
		VALUES ($1, $2, $3)
		RETURNING id, status, reserved_weight_kg, reserved_volume_m3, created_at`,
		voyage.VesselID, voyage.Route, voyage.DepartureDate,
	).Scan(&voyage.ID, &voyage.Status, &voyage.ReservedWeightKg, &voyage.ReservedVolumeM3, &voyage.CreatedAt)
}

// GetAll retrieves all voyages ordered by departure date, including details of the assigned vessel.
func (r *VoyageRepository) GetAll(ctx context.Context) ([]model.Voyage, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT v.id, v.vessel_id, v.route, v.departure_date, v.status,
		        v.reserved_weight_kg, v.reserved_volume_m3, v.created_at,
		        ve.name, ve.max_weight_kg, ve.max_volume_m3
		 FROM voyages v
		 JOIN vessels ve ON ve.id = v.vessel_id
		 ORDER BY v.departure_date ASC`,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var voyages []model.Voyage
	for rows.Next() {
		var voy model.Voyage
		if err := rows.Scan(
			&voy.ID, &voy.VesselID, &voy.Route, &voy.DepartureDate, &voy.Status,
			&voy.ReservedWeightKg, &voy.ReservedVolumeM3, &voy.CreatedAt,
			&voy.VesselName, &voy.MaxWeightKg, &voy.MaxVolumeM3,
		); err != nil {
			return nil, err
		}
		voyages = append(voyages, voy)
	}

	return voyages, rows.Err()
}

// GetByID fetches a single voyage by its unique identifier along with its associated vessel information.
func (r *VoyageRepository) GetByID(ctx context.Context, id string) (*model.Voyage, error) {
	var voy model.Voyage
	err := r.db.QueryRowContext(ctx,
		`SELECT v.id, v.vessel_id, v.route, v.departure_date, v.status,
			v.reserved_weight_kg, v.reserved_volume_m3, v.created_at,
			ve.name, ve.max_weight_kg, ve.max_volume_m3
		FROM voyages v
		JOIN vessels ve ON ve.id = v.vessel_id
		WHERE v.id = $1`,
		id,
	).Scan(
		&voy.ID, &voy.VesselID, &voy.Route, &voy.DepartureDate, &voy.Status,
		&voy.ReservedWeightKg, &voy.ReservedVolumeM3, &voy.CreatedAt,
		&voy.VesselName, &voy.MaxWeightKg, &voy.MaxVolumeM3,
	)
	if err != nil {
		return nil, err
	}
	return &voy, nil
}

// GetByIDForUpdate fetches a single voyage within a transaction and locks the row using FOR UPDATE.
func (r *VoyageRepository) GetByIDForUpdate(ctx context.Context, tx *sql.Tx, id string) (*model.Voyage, error) {
	var voy model.Voyage
	err := tx.QueryRowContext(ctx,
		`SELECT v.id, v.vessel_id, v.route, v.departure_date, v.status,
				v.reserved_weight_kg, v.reserved_volume_m3, v.created_at,
				ve.name, ve.max_weight_kg, ve.max_volume_m3
		FROM voyages v
		JOIN vessels ve ON ve.id = v.vessel_id
		WHERE v.id = $1
		FOR UPDATE`,
		id,
	).Scan(
		&voy.ID, &voy.VesselID, &voy.Route, &voy.DepartureDate, &voy.Status,
		&voy.ReservedWeightKg, &voy.ReservedVolumeM3, &voy.CreatedAt,
		&voy.VesselName, &voy.MaxWeightKg, &voy.MaxVolumeM3,
	)
	if err != nil {
		return nil, err
	}

	return &voy, nil
}

// UpdateReservedTx updates the absolute values of reserved capacity for a specific voyage within a transaction.
func (r *VoyageRepository) UpdateReservedTx(ctx context.Context, tx *sql.Tx, id string, weightKg, volumeM3 float64) error {
	_, err := tx.ExecContext(ctx,
		`UPDATE voyages SET reserved_weight_kg = $1, reserved_volume_m3 = $2 WHERE id = $3`,
		weightKg, volumeM3, id,
	)
	return err
}