package domain

type Order struct {
	ID     int        `json:"id"`
	UserID int        `json:"user_id"`
	Items  []LineItem `json:"items"`
	Total  int        `json:"total"`
	Status string     `json:"status"`
}

type CreateOrderRequest struct {
	UserID int               `json:"user_id"`
	Items  []LineItemRequest `json:"items"`
}

type CreateOrderFromCartRequest struct {
	UserID int `json:"user_id"`
}

type UpdateOrderStatusRequest struct {
	Status string `json:"status"`
}

type PaymentResult struct {
	Success       bool   `json:"success"`
	TransactionID string `json:"transaction_id,omitempty"`
	Error         string `json:"error,omitempty"`
}
