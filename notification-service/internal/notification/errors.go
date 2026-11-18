package notification

import "errors"

var (
	ErrInvalidInput = errors.New("invalid notification data")
	ErrNotFound     = errors.New("notification not found")
	ErrDuplicate    = errors.New("event already processed")
)
