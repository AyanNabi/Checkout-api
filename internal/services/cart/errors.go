package cart

import "errors"

var (
	ErrCartNotFound      = errors.New("cart not found")
	ErrCartEmpty         = errors.New("cart is empty")
	ErrCartAlreadyExists = errors.New("cart already exists")
	ErrItemNotFound      = errors.New("item not found")
	ErrCartItemNotFound  = errors.New("cart item not found")
	ErrInvalidQuantity   = errors.New("invalid quantity")
	ErrInsufficientStock = errors.New("insufficient stock")
)
