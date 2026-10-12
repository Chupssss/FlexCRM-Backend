package auth

// Ошибки, которые могут вернуться в хэндлер

import (
	"errors"
)

var (
	ErrEmailAlreadyExists = errors.New("email already exists")
	ErrUserNotFound       = errors.New("user not found")
)
