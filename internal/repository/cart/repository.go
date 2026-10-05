package cart

import (
	"checkout-api/internal/domain"
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

func (s PostgresStore) CreateUserCart(
	ctx context.Context,
	cart *domain.Cart,
) error {

	tx, err := s.conn.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin create cart transaction: %w", err)
	}

	defer tx.Rollback(ctx)

	var cartID int

	err = tx.QueryRow(
		ctx,
		"INSERT INTO carts (user_id) VALUES ($1) RETURNING id",
		cart.UserID,
	).Scan(&cartID)

	if err != nil {
		return fmt.Errorf("create cart: %w", err)
	}

	for _, item := range cart.Items {

		_, err = tx.Exec(
			ctx,
			`INSERT INTO cart_items
			(cart_id, item_id, quantity, price)
			VALUES ($1, $2, $3, $4)`,
			cartID,
			item.ItemID,
			item.Quantity,
			item.Price,
		)

		if err != nil {
			return fmt.Errorf("insert cart item: %w", err)
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit create cart transaction: %w", err)
	}

	return nil
}

func (s PostgresStore) GetUserCart(
	ctx context.Context,
	userID int,
) (*domain.Cart, error) {

	var cartID int

	err := s.conn.QueryRow(
		ctx,
		"SELECT id FROM carts WHERE user_id = $1",
		userID,
	).Scan(&cartID)

	if err != nil {

		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf(
				"get cart for user %d: %w",
				userID,
				ErrCartNotFound,
			)
		}

		return nil, fmt.Errorf("get cart for user %d: %w", userID, err)
	}

	query := `
		SELECT 
			ci.item_id,
			i.name,
			ci.quantity,
			i.price,
			i.stock
		FROM cart_items ci
		JOIN items i ON ci.item_id = i.id
		WHERE ci.cart_id = $1
		ORDER BY ci.item_id ASC
	`

	rows, err := s.conn.Query(
		ctx,
		query,
		cartID,
	)

	if err != nil {
		return nil, fmt.Errorf("get cart items: %w", err)
	}

	defer rows.Close()

	var items []domain.LineItem

	for rows.Next() {

		var item domain.LineItem

		err := rows.Scan(
			&item.ItemID,
			&item.Name,
			&item.Quantity,
			&item.Price,
			&item.Stock,
		)

		if err != nil {
			return nil, fmt.Errorf("scan cart item: %w", err)
		}

		items = append(items, item)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate cart items: %w", err)
	}

	return &domain.Cart{
		ID:     fmt.Sprintf("%d", cartID),
		UserID: userID,
		Items:  items,
	}, nil
}

func (s PostgresStore) DeleteUserCart(
	ctx context.Context,
	userID int,
) error {

	result, err := s.conn.Exec(
		ctx,
		"DELETE FROM carts WHERE user_id = $1",
		userID,
	)

	if err != nil {
		return fmt.Errorf("delete cart for user %d: %w", userID, err)
	}

	if result.RowsAffected() == 0 {
		return fmt.Errorf(
			"delete cart for user %d: %w",
			userID,
			ErrCartNotFound,
		)
	}

	return nil
}

func (s PostgresStore) UpdateCartItem(
	ctx context.Context,
	userID int,
	itemID int,
	quantity int,
) (bool, error) {

	var cartID int

	err := s.conn.QueryRow(
		ctx,
		"SELECT id FROM carts WHERE user_id = $1",
		userID,
	).Scan(&cartID)

	if err != nil {

		if errors.Is(err, pgx.ErrNoRows) {
			return false, fmt.Errorf(
				"update cart item: %w",
				ErrCartNotFound,
			)
		}

		return false, fmt.Errorf(
			"get cart for update: %w",
			err,
		)
	}

	result, err := s.conn.Exec(
		ctx,
		`UPDATE cart_items
		SET quantity = $1
		WHERE cart_id = $2 AND item_id = $3`,
		quantity,
		cartID,
		itemID,
	)

	if err != nil {
		return false, fmt.Errorf(
			"update cart item %d: %w",
			itemID,
			err,
		)
	}

	if result.RowsAffected() == 0 {
		return false, fmt.Errorf(
			"update cart item %d: %w",
			itemID,
			ErrCartItemNotFound,
		)
	}

	return true, nil
}

func (s PostgresStore) RemoveCartItem(
	ctx context.Context,
	userID int,
	itemID int,
) (bool, error) {

	var cartID int

	err := s.conn.QueryRow(
		ctx,
		"SELECT id FROM carts WHERE user_id = $1",
		userID,
	).Scan(&cartID)

	if err != nil {

		if errors.Is(err, pgx.ErrNoRows) {
			return false, fmt.Errorf(
				"remove cart item: %w",
				ErrCartNotFound,
			)
		}
		return false, fmt.Errorf(
			"get cart for remove item: %w",
			err,
		)
	}

	result, err := s.conn.Exec(
		ctx,
		`DELETE FROM cart_items
		WHERE cart_id = $1 AND item_id = $2`,
		cartID,
		itemID,
	)

	if err != nil {
		return false, fmt.Errorf(
			"remove cart item %d: %w",
			itemID,
			err,
		)
	}

	if result.RowsAffected() == 0 {
		return false, fmt.Errorf(
			"remove cart item %d: %w",
			itemID,
			ErrCartItemNotFound,
		)
	}
	return true, nil
}

func (s PostgresStore) AddToCart(
	ctx context.Context,
	userID int,
	item domain.LineItem,
) error {

	tx, err := s.conn.Begin(ctx)
	if err != nil {
		return fmt.Errorf(
			"begin add to cart transaction: %w",
			err,
		)
	}

	defer tx.Rollback(ctx)

	var cartID int

	err = tx.QueryRow(
		ctx,
		"SELECT id FROM carts WHERE user_id = $1",
		userID,
	).Scan(&cartID)

	if err != nil {
		if !errors.Is(err, pgx.ErrNoRows) {
			return fmt.Errorf(
				"get cart before adding item: %w",
				err,
			)
		}

		err = tx.QueryRow(
			ctx,
			"INSERT INTO carts (user_id) VALUES ($1) RETURNING id",
			userID,
		).Scan(&cartID)

		if err != nil {
			return fmt.Errorf(
				"create cart while adding item: %w",
				err,
			)
		}
	}

	var existingQuantity int

	err = tx.QueryRow(
		ctx,
		`
        SELECT quantity
        FROM cart_items
        WHERE cart_id = $1 AND item_id = $2
        `,
		cartID,
		item.ItemID,
	).Scan(&existingQuantity)

	if err == nil {

		_, err = tx.Exec(
			ctx,
			`
            UPDATE cart_items
            SET quantity = quantity + $1
            WHERE cart_id = $2 AND item_id = $3
            `,
			item.Quantity,
			cartID,
			item.ItemID,
		)

		if err != nil {
			return fmt.Errorf(
				"update existing cart item %d: %w",
				item.ItemID,
				err,
			)
		}

	} else if errors.Is(err, pgx.ErrNoRows) {

		_, err = tx.Exec(
			ctx,
			`
            INSERT INTO cart_items
            (cart_id, item_id, quantity)
            VALUES ($1, $2, $3)
            `,
			cartID,
			item.ItemID,
			item.Quantity,
		)

		if err != nil {
			return fmt.Errorf(
				"insert new cart item %d: %w",
				item.ItemID,
				err,
			)
		}

	} else {
		return fmt.Errorf(
			"check existing cart item %d: %w",
			item.ItemID,
			err,
		)
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf(
			"commit add to cart transaction: %w",
			err,
		)
	}

	return nil
}
