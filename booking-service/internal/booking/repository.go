package booking

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Repository interface {
	Create(ctx context.Context, req CreateBookingRequest) (*Booking, error)
	GetByID(ctx context.Context, id uuid.UUID) (*Booking, error)
	ListByUser(ctx context.Context, userID uuid.UUID, limit, offset int) ([]Booking, error)
	Cancel(ctx context.Context, id uuid.UUID) (*Booking, error)
}

type PostgresRepository struct {
	db *pgxpool.Pool
}

func NewRepository(db *pgxpool.Pool) Repository {
	return &PostgresRepository{
		db: db,
	}
}

func (r *PostgresRepository) Create(
	ctx context.Context,
	req CreateBookingRequest,
) (*Booking, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	var bookingID uuid.UUID

	err = tx.QueryRow(ctx, `
		INSERT INTO bookings (
			title,
			description,
			starts_at,
			ends_at,
			status
		)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id
	`,
		req.Title,
		req.Description,
		req.StartsAt,
		req.EndsAt,
		StatusConfirmed,
	).Scan(&bookingID)

	if err != nil {
		return nil, fmt.Errorf("insert booking: %w", err)
	}

	_, err = tx.Exec(ctx, `
		INSERT INTO booking_participants (
			booking_id,
			user_id,
			role,
			starts_at,
			ends_at,
			status
		)
		VALUES
			($1, $2, $3, $5, $6, $7),
			($1, $4, $8, $5, $6, $7)
	`,
		bookingID,
		req.OrganizerID,
		RoleOrganizer,
		req.InviteeID,
		req.StartsAt,
		req.EndsAt,
		StatusConfirmed,
		RoleInvitee,
	)

	if err != nil {
		if isOverlapError(err) {
			return nil, ErrConflict
		}

		return nil, fmt.Errorf("insert booking participants: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		if isOverlapError(err) {
			return nil, ErrConflict
		}

		return nil, fmt.Errorf("commit transaction: %w", err)
	}

	return r.GetByID(ctx, bookingID)
}

type rowScanner interface {
	Scan(dest ...any) error
}

func scanBooking(row rowScanner) (Booking, error) {
	var booking Booking

	err := row.Scan(
		&booking.ID,
		&booking.Title,
		&booking.Description,
		&booking.StartsAt,
		&booking.EndsAt,
		&booking.Status,
		&booking.CreatedAt,
		&booking.UpdatedAt,
	)
	if err != nil {
		return Booking{}, err
	}

	return booking, nil
}

func (r *PostgresRepository) GetByID(
	ctx context.Context,
	id uuid.UUID,
) (*Booking, error) {
	row := r.db.QueryRow(ctx, `
		SELECT
			id,
			title,
			description,
			starts_at,
			ends_at,
			status,
			created_at,
			updated_at
		FROM bookings
		WHERE id = $1
	`, id)

	booking, err := scanBooking(row)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}

		return nil, fmt.Errorf("query booking: %w", err)
	}

	if err := r.loadParticipants(ctx, &booking); err != nil {
		return nil, err
	}

	return &booking, nil
}

func (r *PostgresRepository) loadParticipants(
	ctx context.Context,
	booking *Booking,
) error {
	rows, err := r.db.Query(ctx, `
		SELECT user_id, role
		FROM booking_participants
		WHERE booking_id = $1
	`, booking.ID)
	if err != nil {
		return fmt.Errorf("query booking participants: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var userID uuid.UUID
		var role ParticipantRole

		if err := rows.Scan(&userID, &role); err != nil {
			return fmt.Errorf("scan booking participant: %w", err)
		}

		switch role {
		case RoleOrganizer:
			booking.OrganizerID = userID
		case RoleInvitee:
			booking.InviteeID = userID
		}
	}

	if err := rows.Err(); err != nil {
		return fmt.Errorf("iterate booking participants: %w", err)
	}

	return nil
}

func (r *PostgresRepository) ListByUser(
	ctx context.Context,
	userID uuid.UUID,
	limit int,
	offset int,
) ([]Booking, error) {
	rows, err := r.db.Query(ctx, `
		SELECT
			b.id,
			b.title,
			b.description,
			b.starts_at,
			b.ends_at,
			b.status,
			b.created_at,
			b.updated_at
		FROM bookings b
		WHERE EXISTS (
			SELECT 1
			FROM booking_participants bp
			WHERE bp.booking_id = b.id
			  AND bp.user_id = $1
		)
		ORDER BY b.starts_at DESC
		LIMIT $2 OFFSET $3
	`,
		userID,
		limit,
		offset,
	)
	if err != nil {
		return nil, fmt.Errorf("query user bookings: %w", err)
	}

	bookings := make([]Booking, 0)

	for rows.Next() {
		booking, err := scanBooking(rows)
		if err != nil {
			rows.Close()
			return nil, fmt.Errorf("scan booking: %w", err)
		}

		bookings = append(bookings, booking)
	}

	if err := rows.Err(); err != nil {
		rows.Close()
		return nil, fmt.Errorf("iterate bookings: %w", err)
	}

	rows.Close()

	for i := range bookings {
		if err := r.loadParticipants(ctx, &bookings[i]); err != nil {
			return nil, err
		}
	}

	return bookings, nil
}

func (r *PostgresRepository) Cancel(
	ctx context.Context,
	id uuid.UUID,
) (*Booking, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	var status BookingStatus

	err = tx.QueryRow(ctx, `
		SELECT status
		FROM bookings
		WHERE id = $1
		FOR UPDATE
	`, id).Scan(&status)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}

		return nil, fmt.Errorf("query booking status: %w", err)
	}

	// Повторная отмена не должна приводить к ошибке.
	if status == StatusCancelled {
		if err := tx.Commit(ctx); err != nil {
			return nil, fmt.Errorf("commit transaction: %w", err)
		}

		return r.GetByID(ctx, id)
	}

	_, err = tx.Exec(ctx, `
		UPDATE bookings
		SET
			status = $2,
			updated_at = NOW()
		WHERE id = $1
	`,
		id,
		StatusCancelled,
	)
	if err != nil {
		return nil, fmt.Errorf("cancel booking: %w", err)
	}

	_, err = tx.Exec(ctx, `
		UPDATE booking_participants
		SET status = $2
		WHERE booking_id = $1
	`,
		id,
		StatusCancelled,
	)
	if err != nil {
		return nil, fmt.Errorf("cancel booking participants: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit transaction: %w", err)
	}

	return r.GetByID(ctx, id)
}

func isOverlapError(err error) bool {
	var pgErr *pgconn.PgError

	if !errors.As(err, &pgErr) {
		return false
	}

	// PostgreSQL exclusion constraint violation.
	return pgErr.Code == "23P01"
}
