package balance

import "errors"

var (
	ErrUserBalanceNotFound = errors.New("user balance not found")
	ErrInsufficientBalance = errors.New("insufficient balance")
)
