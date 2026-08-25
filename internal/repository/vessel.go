package repository

import (
	"context"
	"database/sql"

	"github.com/ove4lo/ship-cargo-service/internal/model"
)

// VesselRepository represents layer between business logic and the database for vessel
type VesselRepository struct {
	db *sql.DB
}

// NewVesselRepository creates an instance of VesselRepository and passes a dependency (database connection) to it
func NewVesselRepository(db *sql.DB) *VesselRepository {
	return &VesselRepository{db: db}
}

// Create adds a new vessel to the databases
func (r *VesselRepository) Create(ctx context.Context, vessel *model.Vessel) error {
	return r.db.QueryRowContext(ctx,
		`INSERT INTO vessels (name, max_weight_kg, max_volume_m3)
		VALUES ($1, $2, $3) RETURNING id, created_at`,
		vessel.Name, vessel.MaxWeightKg, vessel.MaxVolumeM3,
	).Scan(&vessel.ID, &vessel.CreatedAt)
}

// GetALl returns all ships from the database
func (r *VesselRepository) GetAll(ctx context.Context) ([]model.Vessel, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT id, name, max_weight_kg, max_volume_m3, created_at
		FROM vessels ORDER BY created_at DESC`,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var vessels []model.Vessel
	for rows.Next() {
		var v model.Vessel
		if err := rows.Scan(&v.ID, &v.Name, &v.MaxWeightKg, &v.MaxVolumeM3, &v.CreatedAt); err != nil {
			return nil, err
		}
		vessels = append(vessels, v)
	}

	return vessels, rows.Err()
}
