package balance

import (
	"checkout-api/internal/domain"
	balanceRepo "checkout-api/internal/repository/balance"
	"context"
	"errors"
	"fmt"
)

type BalanceStore interface {
	EnsureBalance(ctx context.Context, userID int) error
	GetBalance(ctx context.Context, userID int) (int64, error)
	AddBalance(ctx context.Context, userID int, amount int64) (int64, error)
	DebitBalance(ctx context.Context, userID int, amount int64) (int64, error)
	GetTransactions(ctx context.Context, userID int) ([]*domain.BalanceTransaction, error)
}

type BalanceService struct {
	store BalanceStore
}

func NewBalanceService(store BalanceStore) *BalanceService {
	return &BalanceService{store: store}
}

const maxAllowedAmount int64 = 1_000_000_000

func validateAmount(amount int64) error {
	if amount <= 0 || amount > maxAllowedAmount {
		return ErrInvalidAmount
	}
	return nil
}

func (s *BalanceService) GetBalance(ctx context.Context, userID int) (int64, error) {
	if err := s.store.EnsureBalance(ctx, userID); err != nil {
		return 0, fmt.Errorf("get balance: %w", err)
	}

	balance, err := s.store.GetBalance(ctx, userID)
	if err != nil {
		if errors.Is(err, balanceRepo.ErrUserBalanceNotFound) {
			return 0, nil
		}
		return 0, fmt.Errorf("get balance: %w", err)
	}
	return balance, nil
}

func (s *BalanceService) TopUp(ctx context.Context, userID int, amount int64) (int64, error) {
	if err := validateAmount(amount); err != nil {
		return 0, err
	}
	if err := s.store.EnsureBalance(ctx, userID); err != nil {
		return 0, fmt.Errorf("top up: %w", err)
	}
	balance, err := s.store.AddBalance(ctx, userID, amount)
	if err != nil {
		return 0, fmt.Errorf("top up: %w", err)
	}
	return balance, nil
}

func (s *BalanceService) GetTransactions(ctx context.Context, userID int) ([]*domain.BalanceTransaction, error) {
	if err := s.store.EnsureBalance(ctx, userID); err != nil {
		return nil, fmt.Errorf("get transactions: %w", err)
	}
	transactions, err := s.store.GetTransactions(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("get transactions: %w", err)
	}
	return transactions, nil
}

func (s *BalanceService) DebitForPurchase(ctx context.Context, userID int, amount int64) (int64, error) {
	if amount <= 0 || amount > maxAllowedAmount {
		return 0, ErrInvalidAmount
	}
	if err := s.store.EnsureBalance(ctx, userID); err != nil {
		return 0, fmt.Errorf("purchase debit: %w", err)
	}
	balance, err := s.store.DebitBalance(ctx, userID, amount)
	if err != nil {
		if errors.Is(err, balanceRepo.ErrInsufficientBalance) {
			return 0, ErrInsufficientBalance
		}
		return 0, fmt.Errorf("purchase debit: %w", err)
	}
	return balance, nil
}
