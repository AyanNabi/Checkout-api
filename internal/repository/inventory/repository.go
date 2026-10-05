package inventory

import (
	"checkout-api/internal/domain"
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
)

type PostgresStore struct {
	conn *pgxpool.Pool
}

func NewPostgresStore(conn *pgxpool.Pool) *PostgresStore {
	return &PostgresStore{conn: conn}
}

func (s *PostgresStore) GetInventory(ctx context.Context, userID int) ([]*domain.InventoryItem, error) {
	rows, err := s.conn.Query(ctx, `
		SELECT i.id, i.name, ui.quantity, ui.updated_at
		FROM user_inventory ui
		JOIN items i ON i.id = ui.item_id
		WHERE ui.user_id = $1
		ORDER BY i.id
	`, userID)
	if err != nil {
		return nil, fmt.Errorf("get inventory: %w", err)
	}
	defer rows.Close()

	items := make([]*domain.InventoryItem, 0)
	for rows.Next() {
		var item domain.InventoryItem
		if err := rows.Scan(&item.ItemID, &item.Name, &item.Quantity, &item.UpdatedAt); err != nil {
			return nil, fmt.Errorf("get inventory: scan: %w", err)
		}
		items = append(items, &item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("get inventory: rows: %w", err)
	}
	return items, nil
}
