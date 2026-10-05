package domain

import "time"

type InventoryItem struct {
	ItemID    int       `json:"item_id"`
	Name      string    `json:"name"`
	Quantity  int       `json:"quantity"`
	UpdatedAt time.Time `json:"updated_at"`
}

type InventoryResponse struct {
	Items []*InventoryItem `json:"items"`
}
