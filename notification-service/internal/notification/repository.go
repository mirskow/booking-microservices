package notification

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Repository interface {
	CreateForEvent(
		ctx context.Context,
		event BookingEvent,
	) ([]Notification, error)

	GetByID(
		ctx context.Context,
		id uuid.UUID,
	) (*Notification, error)

	List(
		ctx context.Context,
		recipientID uuid.UUID,
		limit int,
		offset int,
	) ([]Notification, error)

	UpdateStatus(
		ctx context.Context,
		id uuid.UUID,
		status NotificationStatus,
	) error
}

type PostgresRepository struct {
	db *pgxpool.Pool
}

func NewRepository(db *pgxpool.Pool) Repository {
	return &PostgresRepository{
		db: db,
	}
}

func (r *PostgresRepository) CreateForEvent(
	ctx context.Context,
	event BookingEvent,
) ([]Notification, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	payload, err := json.Marshal(event)
	if err != nil {
		return nil, fmt.Errorf("marshal event payload: %w", err)
	}

	for _, recipient := range event.Recipients {
		_, err := tx.Exec(ctx, `
			INSERT INTO notifications (
				event_id,
				recipient_id,
				recipient_name,
				recipient_email,
				booking_id,
				type,
				channel,
				status,
				payload
			)
			VALUES (
				$1, $2, $3, $4, $5, $6, $7, $8, $9
			)
			ON CONFLICT (event_id, recipient_id) DO NOTHING
		`,
			event.EventID,
			recipient.UserID,
			recipient.Name,
			recipient.Email,
			event.BookingID,
			event.Type,
			ChannelEmail,
			StatusPending,
			payload,
		)
		if err != nil {
			return nil, fmt.Errorf("insert notification: %w", err)
		}
	}

	notifications, err := getNotificationsByEventID(ctx, tx, event.EventID)
	if err != nil {
		return nil, err
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit transaction: %w", err)
	}

	return notifications, nil
}

func getNotificationsByEventID(
	ctx context.Context,
	tx pgx.Tx,
	eventID uuid.UUID,
) ([]Notification, error) {
	rows, err := tx.Query(ctx, `
		SELECT
			id,
			event_id,
			recipient_id,
			recipient_name,
			recipient_email,
			booking_id,
			type,
			channel,
			status,
			attempts,
			created_at,
			sent_at
		FROM notifications
		WHERE event_id = $1
		ORDER BY created_at
	`, eventID)
	if err != nil {
		return nil, fmt.Errorf("query notifications by event: %w", err)
	}
	defer rows.Close()

	notifications := make([]Notification, 0)

	for rows.Next() {
		notification, err := scanNotification(rows)
		if err != nil {
			return nil, err
		}

		notifications = append(notifications, notification)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate notifications: %w", err)
	}

	return notifications, nil
}

type rowScanner interface {
	Scan(dest ...any) error
}

func scanNotification(row rowScanner) (Notification, error) {
	var notification Notification

	err := row.Scan(
		&notification.ID,
		&notification.EventID,
		&notification.Recipient.UserID,
		&notification.Recipient.Name,
		&notification.Recipient.Email,
		&notification.BookingID,
		&notification.Type,
		&notification.Channel,
		&notification.Status,
		&notification.Attempts,
		&notification.CreatedAt,
		&notification.SentAt,
	)
	if err != nil {
		return Notification{}, fmt.Errorf("scan notification: %w", err)
	}

	return notification, nil
}

func (r *PostgresRepository) GetByID(
	ctx context.Context,
	id uuid.UUID,
) (*Notification, error) {
	row := r.db.QueryRow(ctx, `
		SELECT
			id,
			event_id,
			recipient_id,
			recipient_name,
			recipient_email,
			booking_id,
			type,
			channel,
			status,
			attempts,
			created_at,
			sent_at
		FROM notifications
		WHERE id = $1
	`, id)

	notification, err := scanNotification(row)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}

		return nil, err
	}

	return &notification, nil
}

func (r *PostgresRepository) List(
	ctx context.Context,
	recipientID uuid.UUID,
	limit int,
	offset int,
) ([]Notification, error) {
	rows, err := r.db.Query(ctx, `
		SELECT
			id,
			event_id,
			recipient_id,
			recipient_name,
			recipient_email,
			booking_id,
			type,
			channel,
			status,
			attempts,
			created_at,
			sent_at
		FROM notifications
		WHERE recipient_id = $1
		ORDER BY created_at DESC
		LIMIT $2 OFFSET $3
	`,
		recipientID,
		limit,
		offset,
	)
	if err != nil {
		return nil, fmt.Errorf("query notifications: %w", err)
	}
	defer rows.Close()

	notifications := make([]Notification, 0)

	for rows.Next() {
		notification, err := scanNotification(rows)
		if err != nil {
			return nil, err
		}

		notifications = append(notifications, notification)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate notifications: %w", err)
	}

	return notifications, nil
}

func (r *PostgresRepository) UpdateStatus(
	ctx context.Context,
	id uuid.UUID,
	status NotificationStatus,
) error {
	var sentAt *time.Time

	if status == StatusSent {
		now := time.Now().UTC()
		sentAt = &now
	}

	tag, err := r.db.Exec(ctx, `
		UPDATE notifications
		SET
			status = $2,
			attempts = attempts + 1,
			sent_at = $3
		WHERE id = $1
	`,
		id,
		status,
		sentAt,
	)
	if err != nil {
		return fmt.Errorf("update notification status: %w", err)
	}

	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}

	return nil
}
