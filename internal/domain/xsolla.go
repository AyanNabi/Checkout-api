package domain

import "time"

type XsollaPurchase struct {
	ID                  string     `json:"id"`
	OrderID             int        `json:"order_id,omitempty"`
	UserID              int        `json:"user_id"`
	ItemID              int        `json:"item_id"`
	SKU                 string     `json:"sku"`
	Quantity            int        `json:"quantity"`
	XsollaTransactionID string     `json:"xsolla_transaction_id,omitempty"`
	Status              string     `json:"status"`
	CreatedAt           time.Time  `json:"created_at"`
	PaidAt              *time.Time `json:"paid_at,omitempty"`
}

type XsollaPaymentRequest struct {
	ItemID   int `json:"item_id"`
	Quantity int `json:"quantity"`
}

type XsollaPaymentResponse struct {
	PurchaseID    string `json:"purchase_id"`
	OrderID       int    `json:"order_id,omitempty"`
	Total         int    `json:"total,omitempty"`
	Token         string `json:"token"`
	PayStationURL string `json:"pay_station_url"`
}

type XsollaPurchaseView struct {
	ID                  string     `json:"id"`
	OrderID             int        `json:"order_id,omitempty"`
	ItemID              int        `json:"item_id,omitempty"`
	ItemName            string     `json:"item_name,omitempty"`
	XsollaSKU           string     `json:"xsolla_sku,omitempty"`
	Quantity            int        `json:"quantity,omitempty"`
	Status              string     `json:"status"`
	XsollaTransactionID string     `json:"xsolla_transaction_id,omitempty"`
	CreatedAt           time.Time  `json:"created_at"`
	PaidAt              *time.Time `json:"paid_at,omitempty"`
}
