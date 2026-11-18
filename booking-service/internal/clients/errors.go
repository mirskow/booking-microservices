package clients

import "errors"

var (
	ErrUserNotFound          = errors.New("user not found")
	ErrDependencyUnavailable = errors.New("dependency unavailable")
)
