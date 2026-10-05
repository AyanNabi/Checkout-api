package xsolla

import (
	"checkout-api/internal/domain"
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Payment struct {
	PurchaseID    string
	OrderID       int
	TransactionID string
	UserID        int
	ItemID        int
	SKU           string
	Quantity      int
}

type PostgresStore struct {
	conn *pgxpool.Pool
}

func NewPostgresStore(conn *pgxpool.Pool) *PostgresStore {
	return &PostgresStore{conn: conn}
}

func (s *PostgresStore) CreatePurchase(ctx context.Context, purchase domain.XsollaPurchase) error {
	_, err := s.conn.Exec(ctx, `
		INSERT INTO xsolla_purchases
			(id, order_id, user_id, item_id, sku, quantity, status)
		VALUES ($1, NULLIF($2, 0), $3, NULLIF($4, 0), NULLIF($5, ''), NULLIF($6, 0), 'pending')
	`, purchase.ID, purchase.OrderID, purchase.UserID, purchase.ItemID, purchase.SKU, purchase.Quantity)
	if err != nil {
		return fmt.Errorf("create xsolla purchase: %w", err)
	}
	return nil
}

func (s *PostgresStore) MarkPurchaseFailed(ctx context.Context, purchaseID string) error {
	_, err := s.conn.Exec(ctx, `
		UPDATE xsolla_purchases SET status = 'failed' WHERE id = $1 AND status = 'pending'
	`, purchaseID)
	if err != nil {
		return fmt.Errorf("mark xsolla purchase failed: %w", err)
	}
	return nil
}

func (s *PostgresStore) UserExists(ctx context.Context, userID int) (bool, error) {
	var exists bool
	if err := s.conn.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM users WHERE id = $1)`, userID).Scan(&exists); err != nil {
		return false, fmt.Errorf("check xsolla user: %w", err)
	}
	return exists, nil
}

func (s *PostgresStore) GetPurchasesByUserID(ctx context.Context, userID int) ([]domain.XsollaPurchaseView, error) {
	rows, err := s.conn.Query(ctx, `
		SELECT p.id, COALESCE(p.order_id, 0), COALESCE(p.item_id, 0), COALESCE(i.name, ''), COALESCE(p.sku, ''),
			COALESCE(p.quantity, 0), p.status, COALESCE(p.xsolla_transaction_id, ''), p.created_at, p.paid_at
		FROM xsolla_purchases p
		LEFT JOIN items i ON i.id = p.item_id
		WHERE p.user_id = $1
		ORDER BY p.created_at DESC
	`, userID)
	if err != nil {
		return nil, fmt.Errorf("get xsolla purchases: %w", err)
	}
	defer rows.Close()

	purchases := make([]domain.XsollaPurchaseView, 0)
	for rows.Next() {
		var purchase domain.XsollaPurchaseView
		if err := rows.Scan(
			&purchase.ID, &purchase.OrderID, &purchase.ItemID, &purchase.ItemName, &purchase.XsollaSKU,
			&purchase.Quantity, &purchase.Status, &purchase.XsollaTransactionID,
			&purchase.CreatedAt, &purchase.PaidAt,
		); err != nil {
			return nil, fmt.Errorf("scan xsolla purchase: %w", err)
		}
		purchases = append(purchases, purchase)
	}
	if rows.Err() != nil {
		return nil, fmt.Errorf("iterate xsolla purchases: %w", rows.Err())
	}
	return purchases, nil
}

func (s *PostgresStore) ProcessPayment(ctx context.Context, payment Payment) (bool, error) {
	tx, err := s.conn.Begin(ctx)
	if err != nil {
		return false, fmt.Errorf("begin xsolla payment: %w", err)
	}
	defer tx.Rollback(ctx)

	var purchase domain.XsollaPurchase
	err = tx.QueryRow(ctx, `
		SELECT id, COALESCE(order_id, 0), user_id, item_id, sku, quantity, COALESCE(xsolla_transaction_id, ''), status, created_at, paid_at
		FROM xsolla_purchases
		WHERE id = $1
		FOR UPDATE
	`, payment.PurchaseID).Scan(
		&purchase.ID, &purchase.OrderID, &purchase.UserID, &purchase.ItemID, &purchase.SKU,
		&purchase.Quantity, &purchase.XsollaTransactionID, &purchase.Status,
		&purchase.CreatedAt, &purchase.PaidAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		purchase = domain.XsollaPurchase{
			ID: payment.PurchaseID, OrderID: payment.OrderID, UserID: payment.UserID, ItemID: payment.ItemID,
			SKU: payment.SKU, Quantity: payment.Quantity, Status: "pending",
		}
		if _, err := tx.Exec(ctx, `
			INSERT INTO xsolla_purchases (id, order_id, user_id, item_id, sku, quantity, status)
			VALUES ($1, NULLIF($2, 0), $3, $4, $5, $6, 'pending')
		`, purchase.ID, purchase.OrderID, purchase.UserID, purchase.ItemID, purchase.SKU, purchase.Quantity); err != nil {
			return false, fmt.Errorf("create webhook purchase: %w", err)
		}
	} else if err != nil {
		return false, fmt.Errorf("load xsolla purchase: %w", err)
	}

	if purchase.Status == "paid" {
		return false, tx.Commit(ctx)
	}
	if purchase.UserID != payment.UserID || (purchase.OrderID == 0 && (purchase.ItemID != payment.ItemID || purchase.SKU != payment.SKU || purchase.Quantity != payment.Quantity)) {
		return false, fmt.Errorf("xsolla payment does not match purchase")
	}

	var existingPurchaseID string
	err = tx.QueryRow(ctx, `
		SELECT id FROM xsolla_purchases
		WHERE xsolla_transaction_id = $1
		FOR UPDATE
	`, payment.TransactionID).Scan(&existingPurchaseID)
	if err == nil {
		return false, tx.Commit(ctx)
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return false, fmt.Errorf("check xsolla transaction: %w", err)
	}

	if _, err := tx.Exec(ctx, `
		UPDATE xsolla_purchases
		SET xsolla_transaction_id = $1, status = 'paid', paid_at = NOW()
		WHERE id = $2
	`, payment.TransactionID, payment.PurchaseID); err != nil {
		return false, fmt.Errorf("mark xsolla purchase paid: %w", err)
	}

	if purchase.OrderID > 0 {
		rows, queryErr := tx.Query(ctx, `SELECT item_id, quantity FROM order_items WHERE order_id = $1`, purchase.OrderID)
		if queryErr != nil {
			return false, fmt.Errorf("load order items: %w", queryErr)
		}
		defer rows.Close()
		for rows.Next() {
			var itemID, quantity int
			if scanErr := rows.Scan(&itemID, &quantity); scanErr != nil {
				return false, fmt.Errorf("scan order item: %w", scanErr)
			}
			if _, execErr := tx.Exec(ctx, `
				INSERT INTO user_inventory (user_id, item_id, quantity, created_at, updated_at)
				VALUES ($1, $2, $3, NOW(), NOW())
				ON CONFLICT (user_id, item_id)
				DO UPDATE SET quantity = user_inventory.quantity + EXCLUDED.quantity, updated_at = NOW()
			`, payment.UserID, itemID, quantity); execErr != nil {
				return false, fmt.Errorf("credit xsolla inventory: %w", execErr)
			}
		}
		if rows.Err() != nil {
			return false, fmt.Errorf("iterate order items: %w", rows.Err())
		}
	} else if _, err := tx.Exec(ctx, `
		INSERT INTO user_inventory (user_id, item_id, quantity, created_at, updated_at)
		VALUES ($1, $2, $3, NOW(), NOW())
		ON CONFLICT (user_id, item_id)
		DO UPDATE SET quantity = user_inventory.quantity + EXCLUDED.quantity, updated_at = NOW()
	`, payment.UserID, payment.ItemID, payment.Quantity); err != nil {
		return false, fmt.Errorf("credit xsolla inventory: %w", err)
	}

	if purchase.OrderID > 0 {
		if _, err := tx.Exec(ctx, `UPDATE orders SET status = 'paid' WHERE id = $1`, purchase.OrderID); err != nil {
			return false, fmt.Errorf("mark order paid: %w", err)
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return false, fmt.Errorf("commit xsolla payment: %w", err)
	}
	return true, nil
}
