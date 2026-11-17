package schedule

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Repository interface {
	ReplaceRules(ctx context.Context, userID uuid.UUID, rules []AvailabilityRuleInput) ([]AvailabilityRule, error)
	GetRules(ctx context.Context, userID uuid.UUID) ([]AvailabilityRule, error)
}

type PostgresRepository struct {
	db *pgxpool.Pool
}

func NewRepository(db *pgxpool.Pool) *PostgresRepository {
	return &PostgresRepository{
		db: db,
	}
}

func (r *PostgresRepository) ReplaceRules(
	ctx context.Context,
	userID uuid.UUID,
	rules []AvailabilityRuleInput,
) ([]AvailabilityRule, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	_, err = tx.Exec(ctx,
		`DELETE FROM availability_rules WHERE user_id = $1`,
		userID,
	)
	if err != nil {
		return nil, fmt.Errorf("delete availability rules: %w", err)
	}

	result := make([]AvailabilityRule, 0, len(rules))

	const query = `
		INSERT INTO availability_rules (
			user_id,
			weekday,
			start_time,
			end_time
		)
		VALUES ($1, $2, $3, $4)
		RETURNING id, user_id, weekday, start_time, end_time, created_at
	`

	for _, rule := range rules {
		var created AvailabilityRule

		err := tx.QueryRow(
			ctx,
			query,
			userID,
			rule.Weekday,
			rule.StartTime,
			rule.EndTime,
		).Scan(
			&created.ID,
			&created.UserID,
			&created.Weekday,
			&created.StartTime,
			&created.EndTime,
			&created.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("insert availability rule: %w", err)
		}

		result = append(result, created)
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit transaction: %w", err)
	}

	return result, nil
}

func (r *PostgresRepository) GetRules(
	ctx context.Context,
	userID uuid.UUID,
) ([]AvailabilityRule, error) {
	const query = `
		SELECT id, user_id, weekday, start_time, end_time, created_at
		FROM availability_rules
		WHERE user_id = $1
		ORDER BY weekday, start_time
	`

	rows, err := r.db.Query(ctx, query, userID)
	if err != nil {
		return nil, fmt.Errorf("query availability rules: %w", err)
	}
	defer rows.Close()

	rules := make([]AvailabilityRule, 0)

	for rows.Next() {
		var rule AvailabilityRule

		if err := rows.Scan(
			&rule.ID,
			&rule.UserID,
			&rule.Weekday,
			&rule.StartTime,
			&rule.EndTime,
			&rule.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan availability rule: %w", err)
		}

		rules = append(rules, rule)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate availability rules: %w", err)
	}

	return rules, nil
}
