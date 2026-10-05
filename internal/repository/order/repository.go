package order

import (
	"checkout-api/internal/domain"
	"checkout-api/internal/helper/page"
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type PostgresStore struct {
	conn *pgxpool.Pool
}

func NewPostgresStore(conn *pgxpool.Pool) *PostgresStore {
	return &PostgresStore{
		conn: conn,
	}
}

func (s PostgresStore) CreateOrder(
	ctx context.Context,
	userID int,
	items []domain.LineItem,
	total int,
	status string,
) (*domain.Order, error) {

	tx, err := s.conn.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf(
			"begin create order transaction: %w",
			err,
		)
	}

	defer tx.Rollback(ctx)

	var orderID int

	query := `
		INSERT INTO orders
			(user_id, total, status)
		VALUES
			($1, $2, $3)
		RETURNING id
	`

	err = tx.QueryRow(ctx, query, userID, total, status).Scan(&orderID)

	if err != nil {
		return nil, fmt.Errorf("create order: %w", err)
	}

	for _, item := range items {

		query = `
			INSERT INTO order_items
				(order_id, item_id, quantity, price)
			VALUES
				($1, $2, $3, $4)
		`

		_, err = tx.Exec(ctx, query, orderID, item.ItemID, item.Quantity, item.Price)

		if err != nil {
			return nil, fmt.Errorf("insert order item %d: %w", item.ItemID, err)
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit create order transaction: %w", err)
	}

	return &domain.Order{
		ID:     orderID,
		UserID: userID,
		Items:  items,
		Total:  total,
		Status: status,
	}, nil
}

func (s PostgresStore) GetOrderByID(ctx context.Context, id int) (*domain.Order, error) {

	var order domain.Order

	query := `
		SELECT
			id,
			user_id,
			total,
			status
		FROM orders
		WHERE id = $1
	`

	err := s.conn.QueryRow(
		ctx,
		query,
		id,
	).Scan(
		&order.ID,
		&order.UserID,
		&order.Total,
		&order.Status,
	)

	if err != nil {

		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf(
				"get order %d: %w",
				id,
				ErrOrderNotFound,
			)
		}

		return nil, fmt.Errorf(
			"get order %d: %w",
			id,
			err,
		)
	}

	itemsQuery := `
		SELECT
			oi.item_id,
			COALESCE(i.name, ''),
			oi.quantity,
			oi.price
		FROM order_items oi
		LEFT JOIN items i ON oi.item_id = i.id
		WHERE oi.order_id = $1
	`

	rows, err := s.conn.Query(
		ctx,
		itemsQuery,
		id,
	)

	if err != nil {
		return nil, fmt.Errorf(
			"get items for order %d: %w",
			id,
			err,
		)
	}

	defer rows.Close()

	for rows.Next() {

		var item domain.LineItem

		if err := rows.Scan(
			&item.ItemID,
			&item.Name,
			&item.Quantity,
			&item.Price,
		); err != nil {
			return nil, fmt.Errorf(
				"scan order item for order %d: %w",
				id,
				err,
			)
		}

		order.Items = append(order.Items, item)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf(
			"iterate order items for order %d: %w",
			id,
			err,
		)
	}

	return &order, nil
}
func (s PostgresStore) GetOrdersByUserID(
	ctx context.Context,
	userID int,
	limit int,
	offset int,
	cursor string,
) ([]*domain.Order, error) {

	var rows pgx.Rows
	var err error

	if cursor != "" {

		decodedCursor, err := page.DecodeCursor(cursor)
		if err != nil {
			return nil, fmt.Errorf("decode cursor: %w", err)
		}

		query := `
			SELECT
				id,
				user_id,
				total,
				status
			FROM orders
			WHERE user_id = $1
			  AND id < $2
			ORDER BY id DESC
			LIMIT $3
		`

		rows, err = s.conn.Query(
			ctx,
			query,
			userID,
			decodedCursor.ID,
			limit+1,
		)

	} else {

		query := `
			SELECT
				id,
				user_id,
				total,
				status
			FROM orders
			WHERE user_id = $1
			ORDER BY id DESC
			LIMIT $2 OFFSET $3
		`

		rows, err = s.conn.Query(
			ctx,
			query,
			userID,
			limit+1,
			offset,
		)
	}

	if err != nil {
		return nil, fmt.Errorf(
			"get orders for user %d: %w",
			userID,
			err,
		)
	}

	defer rows.Close()

	var orders []*domain.Order

	for rows.Next() {

		var order domain.Order

		if err := rows.Scan(
			&order.ID,
			&order.UserID,
			&order.Total,
			&order.Status,
		); err != nil {
			return nil, fmt.Errorf(
				"scan order for user %d: %w",
				userID,
				err,
			)
		}

		itemsQuery := `
			SELECT
				oi.item_id,
				COALESCE(i.name, ''),
				oi.quantity,
				oi.price
			FROM order_items oi
			LEFT JOIN items i ON oi.item_id = i.id
			WHERE oi.order_id = $1
		`

		itemsRows, err := s.conn.Query(
			ctx,
			itemsQuery,
			order.ID,
		)

		if err != nil {
			return nil, fmt.Errorf(
				"get items for order %d: %w",
				order.ID,
				err,
			)
		}

		for itemsRows.Next() {

			var item domain.LineItem

			if err := itemsRows.Scan(
				&item.ItemID,
				&item.Name,
				&item.Quantity,
				&item.Price,
			); err != nil {
				itemsRows.Close()

				return nil, fmt.Errorf(
					"scan item for order %d: %w",
					order.ID,
					err,
				)
			}

			order.Items = append(order.Items, item)
		}

		if err := itemsRows.Err(); err != nil {
			itemsRows.Close()

			return nil, fmt.Errorf(
				"iterate items for order %d: %w",
				order.ID,
				err,
			)
		}

		itemsRows.Close()

		orders = append(orders, &order)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf(
			"iterate orders for user %d: %w",
			userID,
			err,
		)
	}

	return orders, nil
}
func (s PostgresStore) UpdateOrderStatus(
	ctx context.Context,
	id int,
	status string,
) error {

	query := `
		UPDATE orders
		SET status = $1
		WHERE id = $2
	`

	result, err := s.conn.Exec(
		ctx,
		query,
		status,
		id,
	)

	if err != nil {
		return fmt.Errorf(
			"update order %d status: %w",
			id,
			err,
		)
	}

	if result.RowsAffected() == 0 {
		return fmt.Errorf(
			"update order %d status: %w",
			id,
			ErrOrderNotFound,
		)
	}

	return nil
}

func (s PostgresStore) DecrementStock(ctx context.Context, itemID int, quantity int) error {

	tx, err := s.conn.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin decrement stock transaction: %w", err)
	}

	defer tx.Rollback(ctx)

	var stock int

	err = tx.QueryRow(ctx, "SELECT stock FROM items WHERE id = $1 FOR UPDATE", itemID).Scan(&stock)

	if err != nil {

		if errors.Is(err, pgx.ErrNoRows) {
			return fmt.Errorf("decrement stock for item %d: %w", itemID, ErrItemNotFound)
		}

		return fmt.Errorf("get stock for item %d: %w", itemID, err)
	}

	if stock < quantity {
		return fmt.Errorf("decrement stock for item %d: %w", itemID, ErrInsufficientStock)
	}

	_, err = tx.Exec(
		ctx, "UPDATE items SET stock = stock - $1 WHERE id = $2", quantity, itemID)

	if err != nil {
		return fmt.Errorf("update stock for item %d: %w", itemID, err)
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit decrement stock transaction: %w", err)
	}

	return nil
}
