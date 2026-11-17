package user

import (
	"time"

	"github.com/google/uuid"
)

// Profile represents a user's public profile.
type Profile struct {
	ID        uuid.UUID `json:"id"`
	Name      string    `json:"name"`
	Email     string    `json:"email"`
	Timezone  string    `json:"timezone"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// CreateProfileRequest represents a request to create a profile.
type CreateProfileRequest struct {
	Name     string `json:"name"`
	Email    string `json:"email"`
	Timezone string `json:"timezone"`
}

// UpdateProfileRequest represents a request to update a profile.
type UpdateProfileRequest struct {
	Name     string `json:"name"`
	Email    string `json:"email"`
	Timezone string `json:"timezone"`
}
