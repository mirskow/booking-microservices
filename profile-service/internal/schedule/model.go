package schedule

import (
	"time"

	"github.com/google/uuid"
)

// AvailabilityRule represents a recurring weekly availability interval.
type AvailabilityRule struct {
	ID        uuid.UUID `json:"id"`
	UserID    uuid.UUID `json:"user_id"`
	Weekday   int       `json:"weekday"`
	StartTime string    `json:"start_time"`
	EndTime   string    `json:"end_time"`
	CreatedAt time.Time `json:"created_at"`
}

// SetScheduleRequest represents a request to replace a user's weekly schedule.
type SetScheduleRequest struct {
	Rules []AvailabilityRuleInput `json:"rules"`
}

// AvailabilityRuleInput represents an availability interval provided by a client.
type AvailabilityRuleInput struct {
	Weekday   int    `json:"weekday"`
	StartTime string `json:"start_time"`
	EndTime   string `json:"end_time"`
}

// Slot represents an available time interval.
type Slot struct {
	Start time.Time `json:"start"`
	End   time.Time `json:"end"`
}

// SlotsResponse represents available slots for a specific date.
type SlotsResponse struct {
	UserID   uuid.UUID `json:"user_id"`
	Date     string    `json:"date"`
	Timezone string    `json:"timezone"`
	Slots    []Slot    `json:"slots"`
}
