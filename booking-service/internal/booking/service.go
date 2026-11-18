package booking

import (
	"context"
	"errors"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/mirskov/booking-microservices/booking-service/internal/clients"
)

type Service struct {
	repo         Repository
	userClient   *clients.UserClient
	notifyClient *clients.NotificationClient
}

func NewService(
	repo Repository,
	userClient *clients.UserClient,
	notifyClient *clients.NotificationClient,
) *Service {
	return &Service{
		repo:         repo,
		userClient:   userClient,
		notifyClient: notifyClient,
	}
}

func (s *Service) Create(
	ctx context.Context,
	req CreateBookingRequest,
) (*Booking, error) {
	if err := validateCreateRequest(req); err != nil {
		return nil, err
	}

	req.Title = strings.TrimSpace(req.Title)
	req.Description = strings.TrimSpace(req.Description)

	// Проверяем существование организатора.
	organizer, err := s.userClient.GetUser(ctx, req.OrganizerID)
	if err != nil {
		if errors.Is(err, clients.ErrUserNotFound) {
			return nil, ErrUserNotFound
		}

		return nil, fmt.Errorf("get organizer: %w", err)
	}

	// Проверяем существование приглашённого.
	invitee, err := s.userClient.GetUser(ctx, req.InviteeID)
	if err != nil {
		if errors.Is(err, clients.ErrUserNotFound) {
			return nil, ErrUserNotFound
		}

		return nil, fmt.Errorf("get invitee: %w", err)
	}

	// Проверяем расписание обоих участников.
	if err := s.checkAvailability(
		ctx,
		organizer,
		req.StartsAt,
		req.EndsAt,
	); err != nil {
		return nil, err
	}

	if err := s.checkAvailability(
		ctx,
		invitee,
		req.StartsAt,
		req.EndsAt,
	); err != nil {
		return nil, err
	}

	// Создаём встречу в PostgreSQL.
	booking, err := s.repo.Create(ctx, req)
	if err != nil {
		if errors.Is(err, ErrConflict) {
			return nil, ErrConflict
		}

		return nil, fmt.Errorf("create booking: %w", err)
	}

	// Уведомляем участников.
	event := clients.NotificationEvent{
		EventID:     uuid.New(),
		BookingID:   booking.ID,
		Type:        "booking.created",
		Title:       booking.Title,
		Description: booking.Description,
		StartsAt:    booking.StartsAt,
		EndsAt:      booking.EndsAt,
		Recipients: []clients.NotificationRecipient{
			{
				UserID: organizer.ID,
				Name:   organizer.Name,
				Email:  organizer.Email,
			},
			{
				UserID: invitee.ID,
				Name:   invitee.Name,
				Email:  invitee.Email,
			},
		},
	}

	if err := s.notifyClient.SendEvent(ctx, event); err != nil {
		// Встреча уже создана. Не возвращаем ошибку создания,
		// иначе клиент может повторить запрос и создать дубль.
		log.Printf(
			"booking %s created, notification delivery failed: %v",
			booking.ID,
			err,
		)
	}

	return booking, nil
}

func (s *Service) checkAvailability(
	ctx context.Context,
	user *clients.User,
	startsAt time.Time,
	endsAt time.Time,
) error {
	location, err := time.LoadLocation(user.Timezone)
	if err != nil {
		return fmt.Errorf(
			"load timezone for user %s: %w",
			user.ID,
			err,
		)
	}

	// Дата определяется в локальном часовом поясе пользователя.
	date := startsAt.In(location).Format("2006-01-02")

	slots, err := s.userClient.GetSlots(ctx, user.ID, date)
	if err != nil {
		if errors.Is(err, clients.ErrUserNotFound) {
			return ErrUserNotFound
		}

		return fmt.Errorf(
			"get availability for user %s: %w",
			user.ID,
			err,
		)
	}

	for _, slot := range slots.Slots {
		if slot.Start.Equal(startsAt) && slot.End.Equal(endsAt) {
			return nil
		}
	}

	return ErrSlotUnavailable
}

func validateCreateRequest(req CreateBookingRequest) error {
	if req.OrganizerID == uuid.Nil {
		return fmt.Errorf("%w: organizer_id is required", ErrInvalidInput)
	}

	if req.InviteeID == uuid.Nil {
		return fmt.Errorf("%w: invitee_id is required", ErrInvalidInput)
	}

	if req.OrganizerID == req.InviteeID {
		return fmt.Errorf(
			"%w: organizer and invitee must be different users",
			ErrInvalidInput,
		)
	}

	if strings.TrimSpace(req.Title) == "" {
		return fmt.Errorf("%w: title is required", ErrInvalidInput)
	}

	if len(req.Title) > 255 {
		return fmt.Errorf("%w: title is too long", ErrInvalidInput)
	}

	if len(req.Description) > 5000 {
		return fmt.Errorf("%w: description is too long", ErrInvalidInput)
	}

	if req.StartsAt.IsZero() || req.EndsAt.IsZero() {
		return fmt.Errorf("%w: start and end time are required", ErrInvalidInput)
	}

	if !req.StartsAt.Before(req.EndsAt) {
		return fmt.Errorf("%w: starts_at must be before ends_at", ErrInvalidInput)
	}

	if req.StartsAt.Before(time.Now()) {
		return fmt.Errorf("%w: booking cannot be in the past", ErrInvalidInput)
	}

	return nil
}

func (s *Service) GetByID(
	ctx context.Context,
	id uuid.UUID,
) (*Booking, error) {
	if id == uuid.Nil {
		return nil, fmt.Errorf("%w: invalid booking ID", ErrInvalidInput)
	}

	booking, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("get booking: %w", err)
	}

	return booking, nil
}

func (s *Service) ListByUser(
	ctx context.Context,
	userID uuid.UUID,
	limit int,
	offset int,
) ([]Booking, error) {
	if userID == uuid.Nil {
		return nil, fmt.Errorf("%w: invalid user ID", ErrInvalidInput)
	}

	if limit == 0 {
		limit = 20
	}

	if limit < 1 || limit > 100 {
		return nil, fmt.Errorf("%w: limit must be between 1 and 100", ErrInvalidInput)
	}

	if offset < 0 {
		return nil, fmt.Errorf("%w: offset cannot be negative", ErrInvalidInput)
	}

	bookings, err := s.repo.ListByUser(ctx, userID, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("list bookings: %w", err)
	}

	return bookings, nil
}

func (s *Service) Cancel(
	ctx context.Context,
	bookingID uuid.UUID,
	requesterID uuid.UUID,
) (*Booking, error) {
	if bookingID == uuid.Nil || requesterID == uuid.Nil {
		return nil, fmt.Errorf("%w: invalid ID", ErrInvalidInput)
	}

	// Получаем встречу до отмены.
	existing, err := s.repo.GetByID(ctx, bookingID)
	if err != nil {
		return nil, fmt.Errorf("get booking before cancellation: %w", err)
	}

	// Проверяем права на отмену.
	if requesterID != existing.OrganizerID &&
		requesterID != existing.InviteeID {
		return nil, ErrForbidden
	}

	// Повторная отмена не отправляет повторное уведомление.
	if existing.Status == StatusCancelled {
		return existing, nil
	}

	// Отменяем встречу.
	cancelled, err := s.repo.Cancel(ctx, bookingID)
	if err != nil {
		return nil, fmt.Errorf("cancel booking: %w", err)
	}

	// Отправляем уведомления best effort.
	s.notifyCancellation(ctx, cancelled)

	return cancelled, nil
}

func (s *Service) notifyCancellation(
	ctx context.Context,
	booking *Booking,
) {
	organizer, err := s.userClient.GetUser(ctx, booking.OrganizerID)
	if err != nil {
		log.Printf(
			"booking %s cancelled, failed to get organizer: %v",
			booking.ID,
			err,
		)
		return
	}

	invitee, err := s.userClient.GetUser(ctx, booking.InviteeID)
	if err != nil {
		log.Printf(
			"booking %s cancelled, failed to get invitee: %v",
			booking.ID,
			err,
		)
		return
	}

	event := clients.NotificationEvent{
		EventID:     uuid.New(),
		BookingID:   booking.ID,
		Type:        "booking.cancelled",
		Title:       booking.Title,
		Description: booking.Description,
		StartsAt:    booking.StartsAt,
		EndsAt:      booking.EndsAt,
		Recipients: []clients.NotificationRecipient{
			{
				UserID: organizer.ID,
				Name:   organizer.Name,
				Email:  organizer.Email,
			},
			{
				UserID: invitee.ID,
				Name:   invitee.Name,
				Email:  invitee.Email,
			},
		},
	}

	if err := s.notifyClient.SendEvent(ctx, event); err != nil {
		log.Printf(
			"booking %s cancelled, notification delivery failed: %v",
			booking.ID,
			err,
		)
	}
}
