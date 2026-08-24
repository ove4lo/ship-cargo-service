package repository

import (
	"context"
	"database/sql"

	"github.com/ove4lo/ship-cargo-service/internal/model"
)

// UserRepository represents layer between business logic and the database for users
type UserRepository struct {
	db *sql.DB
}

// NewUserRepository creates an instance of UserRepository and passes a dependency (database connection) to it
func NewUserRepository(db *sql.DB) *UserRepository {
	return &UserRepository{db :db}
}

// Create inserts the record and retrieves the generated `id` and `created_at` values ​​via `RETURNING`
func (r *UserRepository) Create(ctx context.Context, user *model.User) error {
	return r.db.QueryRowContext(ctx,
		`INSERT INTO users(name, email, password_hash, role)
		VALUES ($1, $2, $3, $4)
		RETURNING id, created_at`,
		user.Name, user.Email, user.PasswordHash, user.Role,
	).Scan(&user.ID, &user.CreatedAt)
}

// GetByEmail looks up the user by email for login
func (r *UserRepository) GetByEmail(ctx context.Context, email string) (*model.User, error) {
	user := &model.User{}
	err := r.db.QueryRowContext(ctx,
		`SELECT id, name, email, password_hash, role, created_at
		FROM users WHERE email = $1`,
		email,
	).Scan(&user.ID, &user.Name, &user.Email, &user.PasswordHash, &user.Role, &user.CreatedAt)
	if err != nil {
		return nil, err
	}

	return user, nil
}