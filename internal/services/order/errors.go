package order

import "errors"

var (
	ErrItemNotFound        = errors.New("item not found")
	ErrInsufficientStock   = errors.New("insufficient stock")
	ErrInsufficientBalance = errors.New("insufficient balance")
	ErrCartEmpty           = errors.New("cart is empty")
	ErrOrderNotFound       = errors.New("order not found")
	ErrInvalidOrderStatus  = errors.New("invalid order status")
)
