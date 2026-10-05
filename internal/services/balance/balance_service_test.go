package balance

import (
	"context"
	"errors"
	"testing"

	"checkout-api/internal/domain"
)

type fakeBalanceStore struct {
	balance int64
	items   []*domain.BalanceTransaction
	err     error
}

func (f *fakeBalanceStore) EnsureBalance(ctx context.Context, userID int) error {
	return nil
}

func (f *fakeBalanceStore) GetBalance(ctx context.Context, userID int) (int64, error) {
	if f.err != nil {
		return 0, f.err
	}
	return f.balance, nil
}

func (f *fakeBalanceStore) AddBalance(ctx context.Context, userID int, amount int64) (int64, error) {
	if f.err != nil {
		return 0, f.err
	}
	f.balance += amount
	f.items = append(f.items, &domain.BalanceTransaction{
		UserID:       userID,
		Type:         domain.TransactionTypeTopUp,
		Amount:       amount,
		BalanceAfter: f.balance,
		Description:  "Mock balance top-up",
	})
	return f.balance, nil
}

func (f *fakeBalanceStore) DebitBalance(ctx context.Context, userID int, amount int64) (int64, error) {
	if f.err != nil {
		return 0, f.err
	}
	if f.balance < amount {
		return 0, errors.New("insufficient balance")
	}
	f.balance -= amount
	f.items = append(f.items, &domain.BalanceTransaction{
		UserID:       userID,
		Type:         domain.TransactionTypePurchase,
		Amount:       amount,
		BalanceAfter: f.balance,
		Description:  "Purchase",
	})
	return f.balance, nil
}

func (f *fakeBalanceStore) GetTransactions(ctx context.Context, userID int) ([]*domain.BalanceTransaction, error) {
	if f.err != nil {
		return nil, f.err
	}
	return f.items, nil
}

func TestBalanceService_TopUp(t *testing.T) {
	store := &fakeBalanceStore{balance: 1000}
	service := NewBalanceService(store)

	got, err := service.TopUp(context.Background(), 7, 2500)
	if err != nil {
		t.Fatalf("TopUp returned error: %v", err)
	}
	if got != 3500 {
		t.Fatalf("TopUp balance = %d, want 3500", got)
	}
}

func TestBalanceService_TopUpRejectsInvalidAmount(t *testing.T) {
	store := &fakeBalanceStore{balance: 1000}
	service := NewBalanceService(store)

	for _, amount := range []int64{0, -1, 1_000_000_001} {
		if _, err := service.TopUp(context.Background(), 7, amount); !errors.Is(err, ErrInvalidAmount) {
			t.Fatalf("TopUp(%d) error = %v, want %v", amount, err, ErrInvalidAmount)
		}
	}
}

func TestBalanceService_GetBalanceAndTransactions(t *testing.T) {
	store := &fakeBalanceStore{balance: 500}
	service := NewBalanceService(store)

	balance, err := service.GetBalance(context.Background(), 2)
	if err != nil {
		t.Fatalf("GetBalance returned error: %v", err)
	}
	if balance != 500 {
		t.Fatalf("GetBalance = %d, want 500", balance)
	}

	_, err = service.TopUp(context.Background(), 2, 1000)
	if err != nil {
		t.Fatalf("TopUp returned error: %v", err)
	}

	transactions, err := service.GetTransactions(context.Background(), 2)
	if err != nil {
		t.Fatalf("GetTransactions returned error: %v", err)
	}
	if len(transactions) != 1 {
		t.Fatalf("len(transactions) = %d, want 1", len(transactions))
	}
	if transactions[0].Type != domain.TransactionTypeTopUp {
		t.Fatalf("transaction type = %s, want %s", transactions[0].Type, domain.TransactionTypeTopUp)
	}
}
