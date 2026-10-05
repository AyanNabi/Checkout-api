package domain

import (
	pageview "checkout-api/internal/helper/page"
	"time"
)

type Item struct {
	ID          int       `json:"id"`
	Name        string    `json:"name"`
	Category    string    `json:"category_id"`
	Description string    `json:"description"`
	Price       int       `json:"price"`
	Stock       int       `json:"stock"`
	XsollaSKU   string    `json:"xsolla_sku,omitempty"`
	CreatedAt   time.Time `json:"created_at"`
}
type ItemPage = pageview.Page[Item]

func (i Item) Validate() {

}
