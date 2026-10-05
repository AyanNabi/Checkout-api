package domain

import "time"

const (
	TransactionTypeTopUp    = "TOP_UP"
	TransactionTypePurchase = "PURCHASE"
)

type UserBalance struct {
	UserID    int       `json:"user_id"`
	Balance   int64     `json:"balance"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type TopUpRequest struct {
	Amount int64 `json:"amount"`
}

type BalanceResponse struct {
	Balance int64 `json:"balance"`
}

type BalanceTransaction struct {
	ID           int       `json:"id"`
	UserID       int       `json:"user_id"`
	Type         string    `json:"type"`
	Amount       int64     `json:"amount"`
	BalanceAfter int64     `json:"balance_after"`
	Description  string    `json:"description"`
	CreatedAt    time.Time `json:"created_at"`
}

type TransactionsResponse struct {
	Transactions []*BalanceTransaction `json:"transactions"`
}
