package booking

import "errors"

var (
	ErrInvalidInput    = errors.New("invalid booking data")
	ErrNotFound        = errors.New("booking not found")
	ErrUserNotFound    = errors.New("user not found")
	ErrSlotUnavailable = errors.New("booking slot unavailable")
	ErrConflict        = errors.New("booking conflict")
	ErrForbidden       = errors.New("forbidden")
)
