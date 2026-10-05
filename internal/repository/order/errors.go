package order

import "errors"

var (
	ErrItemNotFound      = errors.New("item not found")
	ErrOrderNotFound     = errors.New("order not found")
	ErrInsufficientStock = errors.New("insufficient stock")
)
