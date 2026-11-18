package notification

import (
	"context"
	"errors"
	"fmt"
	netmail "net/mail"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/mirskov/booking-microservices/notification-service/internal/mail"
)

var (
	ErrDeliveryFailed = errors.New("notification delivery failed")
)

const (
	maxRecipients    = 100
	maxListLimit     = 100
	defaultListLimit = 20
)

type Service struct {
	repo   Repository
	sender mail.Sender
}

func NewService(repo Repository, sender mail.Sender) *Service {
	return &Service{
		repo:   repo,
		sender: sender,
	}
}

func (s *Service) ProcessEvent(
	ctx context.Context,
	event BookingEvent,
) (*EventResponse, error) {
	if err := validateEvent(event); err != nil {
		return nil, err
	}

	notifications, err := s.repo.CreateForEvent(ctx, event)
	if err != nil {
		return nil, fmt.Errorf("create notifications: %w", err)
	}

	var deliveryErrors []error

	for i := range notifications {
		notification := &notifications[i]

		// Уже отправленное уведомление повторно не отправляем.
		if notification.Status == StatusSent {
			continue
		}

		message := buildMessage(*notification, event)

		err := s.sender.Send(ctx, message)
		if err != nil {
			updateErr := s.repo.UpdateStatus(
				ctx,
				notification.ID,
				StatusFailed,
			)

			if updateErr != nil {
				deliveryErrors = append(
					deliveryErrors,
					fmt.Errorf(
						"notification %s: sending failed: %w; updating status failed: %v",
						notification.ID,
						err,
						updateErr,
					),
				)
			} else {
				notification.Status = StatusFailed
				notification.Attempts++

				deliveryErrors = append(
					deliveryErrors,
					fmt.Errorf(
						"notification %s: %w",
						notification.ID,
						err,
					),
				)
			}

			continue
		}

		if err := s.repo.UpdateStatus(
			ctx,
			notification.ID,
			StatusSent,
		); err != nil {
			deliveryErrors = append(
				deliveryErrors,
				fmt.Errorf(
					"notification %s sent but status update failed: %w",
					notification.ID,
					err,
				),
			)

			continue
		}

		notification.Status = StatusSent
		notification.Attempts++
		now := time.Now().UTC()
		notification.SentAt = &now
	}

	response := &EventResponse{
		EventID:       event.EventID,
		Notifications: notifications,
	}

	if len(deliveryErrors) > 0 {
		return response, errors.Join(
			ErrDeliveryFailed,
			errors.Join(deliveryErrors...),
		)
	}

	return response, nil
}

func validateEvent(event BookingEvent) error {
	if event.EventID == uuid.Nil {
		return fmt.Errorf("%w: event_id is required", ErrInvalidInput)
	}

	if event.BookingID == uuid.Nil {
		return fmt.Errorf("%w: booking_id is required", ErrInvalidInput)
	}

	switch event.Type {
	case NotificationBookingCreated, NotificationBookingCancelled:
	default:
		return fmt.Errorf("%w: unsupported event type", ErrInvalidInput)
	}

	if strings.TrimSpace(event.Title) == "" {
		return fmt.Errorf("%w: title is required", ErrInvalidInput)
	}

	if len(event.Title) > 255 {
		return fmt.Errorf("%w: title is too long", ErrInvalidInput)
	}

	if !event.StartsAt.Before(event.EndsAt) {
		return fmt.Errorf("%w: starts_at must be before ends_at", ErrInvalidInput)
	}

	if len(event.Recipients) == 0 {
		return fmt.Errorf("%w: recipients are required", ErrInvalidInput)
	}

	if len(event.Recipients) > maxRecipients {
		return fmt.Errorf(
			"%w: maximum number of recipients is %d",
			ErrInvalidInput,
			maxRecipients,
		)
	}

	seen := make(map[uuid.UUID]struct{}, len(event.Recipients))

	for i, recipient := range event.Recipients {
		if recipient.UserID == uuid.Nil {
			return fmt.Errorf(
				"%w: recipient %d has invalid user_id",
				ErrInvalidInput,
				i,
			)
		}

		if _, exists := seen[recipient.UserID]; exists {
			return fmt.Errorf(
				"%w: duplicate recipient",
				ErrInvalidInput,
			)
		}

		seen[recipient.UserID] = struct{}{}

		if strings.TrimSpace(recipient.Name) == "" {
			return fmt.Errorf(
				"%w: recipient %d has empty name",
				ErrInvalidInput,
				i,
			)
		}

		if _, err := netmail.ParseAddress(recipient.Email); err != nil {
			return fmt.Errorf(
				"%w: recipient %d has invalid email",
				ErrInvalidInput,
				i,
			)
		}
	}

	return nil
}

func buildMessage(
	notification Notification,
	event BookingEvent,
) mail.Message {
	var subject string
	var action string

	switch notification.Type {
	case NotificationBookingCreated:
		subject = "Встреча создана"
		action = "Ваша встреча успешно создана."

	case NotificationBookingCancelled:
		subject = "Встреча отменена"
		action = "Ваша встреча отменена."
	}

	body := fmt.Sprintf(
		`Здравствуйте, %s!

%s

Название встречи: %s
Описание: %s

Начало: %s
Окончание: %s

ID встречи: %s

`,
		notification.Recipient.Name,
		action,
		event.Title,
		event.Description,
		event.StartsAt.Format("02.01.2006 15:04 MST"),
		event.EndsAt.Format("02.01.2006 15:04 MST"),
		notification.BookingID,
	)

	return mail.Message{
		To:      notification.Recipient.Email,
		Subject: subject,
		Body:    body,
	}
}

func (s *Service) GetByID(
	ctx context.Context,
	id uuid.UUID,
) (*Notification, error) {
	if id == uuid.Nil {
		return nil, fmt.Errorf("%w: invalid notification ID", ErrInvalidInput)
	}

	notification, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("get notification: %w", err)
	}

	return notification, nil
}

func (s *Service) List(
	ctx context.Context,
	recipientID uuid.UUID,
	limit int,
	offset int,
) ([]Notification, error) {
	if recipientID == uuid.Nil {
		return nil, fmt.Errorf("%w: invalid recipient ID", ErrInvalidInput)
	}

	if limit == 0 {
		limit = defaultListLimit
	}

	if limit < 0 || limit > maxListLimit {
		return nil, fmt.Errorf(
			"%w: limit must be between 1 and %d",
			ErrInvalidInput,
			maxListLimit,
		)
	}

	if offset < 0 {
		return nil, fmt.Errorf("%w: offset cannot be negative", ErrInvalidInput)
	}

	notifications, err := s.repo.List(
		ctx,
		recipientID,
		limit,
		offset,
	)
	if err != nil {
		return nil, fmt.Errorf("list notifications: %w", err)
	}

	return notifications, nil
}
