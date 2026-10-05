package domain

type Cart struct {
	ID     string     `json:"id"`
	UserID int        `json:"user_id"`
	Items  []LineItem `json:"items"`
}

type LineItemRequest struct {
	ItemID   int `json:"item_id"`
	Quantity int `json:"quantity"`
}

type CreateUserCartRequest struct {
	UserID int               `json:"user_id"`
	Items  []LineItemRequest `json:"items"`
}

type UpdateCartItemRequest struct {
	UserID   int `json:"user_id"`
	Quantity int `json:"quantity"`
}

type RemoveCartItemRequest struct {
	UserID int `json:"user_id"`
}

type AddCartItemRequest struct {
	UserID   int `json:"user_id"`
	ItemID   int `json:"item_id"`
	Quantity int `json:"quantity"`
}
