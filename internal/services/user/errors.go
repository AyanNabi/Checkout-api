package user

import "errors"

var (
	ErrItemNotFound           = errors.New("item not found")
	ErrEmailTaken             = errors.New("email already taken")
	ErrEmailRequired          = errors.New("email is required")
	ErrPasswordRequired       = errors.New("password is required")
	ErrInvalidCredentials     = errors.New("invalid credentials")
	ErrInvalidRefreshToken    = errors.New("invalid refresh token")
	ErrRefreshTokenGeneration = errors.New("failed to generate refresh token")
	ErrUsernameRequired       = errors.New("username is required")
	ErrUserNotFound           = errors.New("user not found")
	ErrUsernameTaken          = errors.New("user already taken ")
)
