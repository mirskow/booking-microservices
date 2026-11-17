package user

import (
	"context"
	"errors"
	"fmt"
	"net/mail"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

var (
	ErrNotFound     = errors.New("profile not found")
	ErrEmailExists  = errors.New("email already exists")
	ErrInvalidInput = errors.New("invalid profile data")
)

type Service struct {
	repo Repository
}

func NewService(repo Repository) *Service {
	return &Service{
		repo: repo,
	}
}

func (s *Service) Create(
	ctx context.Context,
	req CreateProfileRequest,
) (*Profile, error) {
	req.Name = strings.TrimSpace(req.Name)
	req.Email = strings.TrimSpace(req.Email)
	req.Timezone = strings.TrimSpace(req.Timezone)

	if err := validateProfile(req.Name, req.Email, req.Timezone); err != nil {
		return nil, err
	}

	profile, err := s.repo.Create(ctx, req)
	if err != nil {
		return nil, mapRepositoryError(err)
	}

	return profile, nil
}

func (s *Service) GetByID(
	ctx context.Context,
	id uuid.UUID,
) (*Profile, error) {
	profile, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, mapRepositoryError(err)
	}

	return profile, nil
}

func (s *Service) Update(
	ctx context.Context,
	id uuid.UUID,
	req UpdateProfileRequest,
) (*Profile, error) {
	req.Name = strings.TrimSpace(req.Name)
	req.Email = strings.TrimSpace(req.Email)
	req.Timezone = strings.TrimSpace(req.Timezone)

	if err := validateProfile(req.Name, req.Email, req.Timezone); err != nil {
		return nil, err
	}

	profile, err := s.repo.Update(ctx, id, req)
	if err != nil {
		return nil, mapRepositoryError(err)
	}

	return profile, nil
}

func validateProfile(name, email, timezone string) error {
	if name == "" {
		return fmt.Errorf("%w: name is required", ErrInvalidInput)
	}

	if len(name) > 100 {
		return fmt.Errorf("%w: name must not exceed 100 characters", ErrInvalidInput)
	}

	if email == "" {
		return fmt.Errorf("%w: email is required", ErrInvalidInput)
	}

	if len(email) > 255 {
		return fmt.Errorf("%w: email must not exceed 255 characters", ErrInvalidInput)
	}

	if _, err := mail.ParseAddress(email); err != nil {
		return fmt.Errorf("%w: invalid email", ErrInvalidInput)
	}

	if timezone == "" {
		return fmt.Errorf("%w: timezone is required", ErrInvalidInput)
	}

	if _, err := time.LoadLocation(timezone); err != nil {
		return fmt.Errorf("%w: invalid timezone", ErrInvalidInput)
	}

	return nil
}

func mapRepositoryError(err error) error {
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrNotFound
	}

	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		if pgErr.Code == "23505" {
			return ErrEmailExists
		}
	}

	return fmt.Errorf("repository operation failed: %w", err)
}

func (s *Service) GetTimezone(
	ctx context.Context,
	userID uuid.UUID,
) (string, error) {
	user, err := s.repo.GetByID(ctx, userID)
	if err != nil {
		return "", mapRepositoryError(err)
	}

	return user.Timezone, nil
}
