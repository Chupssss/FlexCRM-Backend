package users

import "errors"

var (
	ErrUserNotFound = errors.New("user not found")
)
var ErrInvalidUserData = errors.New("invalid user data")
var ErrAdminProtected = errors.New("admin role and status are protected")
