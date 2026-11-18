package booking

import (
	"time"

	"github.com/google/uuid"
)

type BookingStatus string

const (
	StatusConfirmed BookingStatus = "confirmed"
	StatusCancelled BookingStatus = "cancelled"
)

type ParticipantRole string

const (
	RoleOrganizer ParticipantRole = "organizer"
	RoleInvitee   ParticipantRole = "invitee"
)

type Booking struct {
	ID          uuid.UUID `json:"id"`
	Title       string    `json:"title"`
	Description string    `json:"description"`

	StartsAt time.Time `json:"starts_at"`
	EndsAt   time.Time `json:"ends_at"`

	Status BookingStatus `json:"status"`

	OrganizerID uuid.UUID `json:"organizer_id"`
	InviteeID   uuid.UUID `json:"invitee_id"`

	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type CreateBookingRequest struct {
	OrganizerID uuid.UUID `json:"organizer_id"`
	InviteeID   uuid.UUID `json:"invitee_id"`

	Title       string `json:"title"`
	Description string `json:"description"`

	StartsAt time.Time `json:"starts_at"`
	EndsAt   time.Time `json:"ends_at"`
}

type BookingParticipant struct {
	BookingID uuid.UUID       `json:"booking_id"`
	UserID    uuid.UUID       `json:"user_id"`
	Role      ParticipantRole `json:"role"`
}

type BookingListResponse struct {
	Bookings []Booking `json:"bookings"`
}
