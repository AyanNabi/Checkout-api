package balance

import (
	"checkout-api/internal/domain"
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type PostgresStore struct {
	conn *pgxpool.Pool
}

func NewPostgresStore(conn *pgxpool.Pool) *PostgresStore {
	return &PostgresStore{conn: conn}
}

func (s PostgresStore) EnsureBalance(ctx context.Context, userID int) error {
	_, err := s.conn.Exec(ctx,
		`INSERT INTO user_balances (user_id, balance, created_at, updated_at)
		VALUES ($1, 0, NOW(), NOW())
		ON CONFLICT (user_id) DO NOTHING`,
		userID,
	)
	if err != nil {
		return fmt.Errorf("ensure balance: %w", err)
	}
	return nil
}

func (s PostgresStore) GetBalance(ctx context.Context, userID int) (int64, error) {
	var balance int64
	err := s.conn.QueryRow(ctx, `SELECT balance FROM user_balances WHERE user_id = $1`, userID).Scan(&balance)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return 0, fmt.Errorf("get balance: %w", ErrUserBalanceNotFound)
		}
		return 0, fmt.Errorf("get balance: %w", err)
	}
	return balance, nil
}

func (s PostgresStore) AddBalance(ctx context.Context, userID int, amount int64) (int64, error) {
	tx, err := s.conn.Begin(ctx)
	if err != nil {
		return 0, fmt.Errorf("add balance: begin tx: %w", err)
	}
	defer tx.Rollback(ctx)

	var balance int64
	if err := tx.QueryRow(ctx,
		`SELECT balance FROM user_balances WHERE user_id = $1 FOR UPDATE`,
		userID,
	).Scan(&balance); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			if _, err := tx.Exec(ctx, `INSERT INTO user_balances (user_id, balance, created_at, updated_at) VALUES ($1, 0, NOW(), NOW())`, userID); err != nil {
				return 0, fmt.Errorf("add balance: create balance: %w", err)
			}
			balance = 0
		} else {
			return 0, fmt.Errorf("add balance: select balance: %w", err)
		}
	}

	balance += amount
	if _, err := tx.Exec(ctx,
		`UPDATE user_balances SET balance = $1, updated_at = NOW() WHERE user_id = $2`,
		balance, userID,
	); err != nil {
		return 0, fmt.Errorf("add balance: update balance: %w", err)
	}

	if _, err := tx.Exec(ctx,
		`INSERT INTO balance_transactions (user_id, type, amount, balance_after, description, created_at)
		VALUES ($1, $2, $3, $4, $5, $6)`,
		userID, domain.TransactionTypeTopUp, amount, balance, "Mock balance top-up", time.Now(),
	); err != nil {
		return 0, fmt.Errorf("add balance: create transaction: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return 0, fmt.Errorf("add balance: commit: %w", err)
	}
	return balance, nil
}

func (s PostgresStore) DebitBalance(ctx context.Context, userID int, amount int64) (int64, error) {
	tx, err := s.conn.Begin(ctx)
	if err != nil {
		return 0, fmt.Errorf("debit balance: begin tx: %w", err)
	}
	defer tx.Rollback(ctx)

	var balance int64
	if err := tx.QueryRow(ctx,
		`SELECT balance FROM user_balances WHERE user_id = $1 FOR UPDATE`,
		userID,
	).Scan(&balance); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return 0, fmt.Errorf("debit balance: %w", ErrUserBalanceNotFound)
		}
		return 0, fmt.Errorf("debit balance: select balance: %w", err)
	}

	if balance < amount {
		return 0, fmt.Errorf("debit balance: %w", ErrInsufficientBalance)
	}

	newBalance := balance - amount
	if _, err := tx.Exec(ctx,
		`UPDATE user_balances SET balance = $1, updated_at = NOW() WHERE user_id = $2`,
		newBalance, userID,
	); err != nil {
		return 0, fmt.Errorf("debit balance: update balance: %w", err)
	}

	if _, err := tx.Exec(ctx,
		`INSERT INTO balance_transactions (user_id, type, amount, balance_after, description, created_at)
		VALUES ($1, $2, $3, $4, $5, $6)`,
		userID, domain.TransactionTypePurchase, amount, newBalance, "Purchase", time.Now(),
	); err != nil {
		return 0, fmt.Errorf("debit balance: create transaction: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return 0, fmt.Errorf("debit balance: commit: %w", err)
	}
	return newBalance, nil
}

func (s PostgresStore) GetTransactions(ctx context.Context, userID int) ([]*domain.BalanceTransaction, error) {
	rows, err := s.conn.Query(ctx,
		`SELECT id, user_id, type, amount, balance_after, description, created_at
		FROM balance_transactions WHERE user_id = $1 ORDER BY id DESC`,
		userID,
	)
	if err != nil {
		return nil, fmt.Errorf("get transactions: %w", err)
	}
	defer rows.Close()

	transactions := make([]*domain.BalanceTransaction, 0)
	for rows.Next() {
		var tx domain.BalanceTransaction
		if err := rows.Scan(&tx.ID, &tx.UserID, &tx.Type, &tx.Amount, &tx.BalanceAfter, &tx.Description, &tx.CreatedAt); err != nil {
			return nil, fmt.Errorf("get transactions: scan: %w", err)
		}
		transactions = append(transactions, &tx)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("get transactions: rows: %w", err)
	}
	return transactions, nil
}
