package auth

// Ошибки, которые могут вернуться в хэндлер

import (
	"errors"
)

var (
	ErrEmailAlreadyExists   = errors.New("email already exists")
	ErrUserNotFound         = errors.New("user not found")
	ErrDifferentPass        = errors.New("password doesnt coincide")
	ErrUserDeactive         = errors.New("user doenst active")
	ErrRefreshTokenNotFound = errors.New("refresh-token not found")
	ErrRefreshTokenExpired  = errors.New("refresh-token expired")
)
