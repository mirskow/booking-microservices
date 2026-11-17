package user

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Repository defines operations for working with profiles.
type Repository interface {
	Create(ctx context.Context, req CreateProfileRequest) (*Profile, error)
	GetByID(ctx context.Context, id uuid.UUID) (*Profile, error)
	Update(ctx context.Context, id uuid.UUID, req UpdateProfileRequest) (*Profile, error)
}

// PostgresRepository implements Repository using PostgreSQL.
type PostgresRepository struct {
	db *pgxpool.Pool
}

func NewRepository(db *pgxpool.Pool) *PostgresRepository {
	return &PostgresRepository{
		db: db,
	}
}

func (r *PostgresRepository) Create(
	ctx context.Context,
	req CreateProfileRequest,
) (*Profile, error) {
	const query = `
		INSERT INTO users (name, email, timezone)
		VALUES ($1, $2, $3)
		RETURNING id, name, email, timezone, created_at, updated_at
	`

	var p Profile

	err := r.db.QueryRow(
		ctx,
		query,
		req.Name,
		req.Email,
		req.Timezone,
	).Scan(
		&p.ID,
		&p.Name,
		&p.Email,
		&p.Timezone,
		&p.CreatedAt,
		&p.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("create profile: %w", err)
	}

	return &p, nil
}

func (r *PostgresRepository) GetByID(
	ctx context.Context,
	id uuid.UUID,
) (*Profile, error) {
	const query = `
		SELECT id, name, email, timezone, created_at, updated_at
		FROM users
		WHERE id = $1
	`

	var p Profile

	err := r.db.QueryRow(ctx, query, id).Scan(
		&p.ID,
		&p.Name,
		&p.Email,
		&p.Timezone,
		&p.CreatedAt,
		&p.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("get profile: %w", err)
	}

	return &p, nil
}

func (r *PostgresRepository) Update(
	ctx context.Context,
	id uuid.UUID,
	req UpdateProfileRequest,
) (*Profile, error) {
	const query = `
		UPDATE users
		SET
			name = $2,
			email = $3,
			timezone = $4,
			updated_at = NOW()
		WHERE id = $1
		RETURNING id, name, email, timezone, created_at, updated_at
	`

	var p Profile

	err := r.db.QueryRow(
		ctx,
		query,
		id,
		req.Name,
		req.Email,
		req.Timezone,
	).Scan(
		&p.ID,
		&p.Name,
		&p.Email,
		&p.Timezone,
		&p.CreatedAt,
		&p.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("update profile: %w", err)
	}

	return &p, nil
}
