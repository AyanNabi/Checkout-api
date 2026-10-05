package domain

type LineItem struct {
	ItemID   int    `json:"item_id"`
	Name     string `json:"name"`
	Quantity int    `json:"quantity"`
	Price    int    `json:"price"`
	Stock    int    `json:"stock"`
}
