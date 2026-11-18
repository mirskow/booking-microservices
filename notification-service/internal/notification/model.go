package notification

import (
	"time"

	"github.com/google/uuid"
)

type NotificationType string

const (
	NotificationBookingCreated   NotificationType = "booking.created"
	NotificationBookingCancelled NotificationType = "booking.cancelled"
)

type NotificationStatus string

const (
	StatusPending NotificationStatus = "pending"
	StatusSent    NotificationStatus = "sent"
	StatusFailed  NotificationStatus = "failed"
)

type NotificationChannel string

const (
	ChannelEmail NotificationChannel = "email"
)

// Получатель уведомления.
type Recipient struct {
	UserID uuid.UUID `json:"user_id"`
	Name   string    `json:"name"`
	Email  string    `json:"email"`
}

// Событие, поступающее от booking-service.
type BookingEvent struct {
	EventID     uuid.UUID        `json:"event_id"`
	BookingID   uuid.UUID        `json:"booking_id"`
	Type        NotificationType `json:"type"`
	Title       string           `json:"title"`
	Description string           `json:"description"`
	StartsAt    time.Time        `json:"starts_at"`
	EndsAt      time.Time        `json:"ends_at"`
	Recipients  []Recipient      `json:"recipients"`
}

// Уведомление, хранящееся в БД.
type Notification struct {
	ID        uuid.UUID           `json:"id"`
	EventID   uuid.UUID           `json:"event_id"`
	Recipient Recipient           `json:"recipient"`
	BookingID uuid.UUID           `json:"booking_id"`
	Type      NotificationType    `json:"type"`
	Channel   NotificationChannel `json:"channel"`
	Status    NotificationStatus  `json:"status"`
	Attempts  int                 `json:"attempts"`
	CreatedAt time.Time           `json:"created_at"`
	SentAt    *time.Time          `json:"sent_at,omitempty"`
}

// Ответ на запрос обработки события.
type EventResponse struct {
	EventID       uuid.UUID      `json:"event_id"`
	Notifications []Notification `json:"notifications"`
}
