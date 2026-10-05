package item

import (
	"checkout-api/internal/domain"
	filter "checkout-api/internal/helper/filter"
	"checkout-api/internal/helper/page"
	"context"
	"errors"
	"fmt"
	"strings"

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

func (s PostgresStore) GetItems(ctx context.Context, request filter.Request) ([]*domain.Item, error) {
	query := `SELECT id, name, description, category, price, stock, COALESCE(xsolla_sku, ''), created_at FROM items`

	q := filter.NewQueryBuilder()
	//servicede
	if request.Pagination.Cursor != "" {
		cursor, err := page.DecodeCursor(request.Pagination.Cursor)
		if err != nil {
			return nil, fmt.Errorf("decode cursor: %w", err)
		}
		q.Add("id > $%d", cursor.ID)
	}

	if request.Filter.Min > 0 {
		q.Add("price >= $%d", request.Filter.Min)
	}

	if request.Filter.Max > 0 {
		q.Add("price <= $%d", request.Filter.Max)
	}

	if request.Filter.Category != "" {
		q.Add("category = $%d", request.Filter.Category)
	}

	if len(q.Conditions) > 0 {
		query += " WHERE " + strings.Join(q.Conditions, " AND ")
	}

	query += fmt.Sprintf(" ORDER BY id ASC LIMIT $%d", q.Position)
	q.Args = append(q.Args, request.Pagination.Limit+1)

	rows, err := s.conn.Query(ctx, query, q.Args...)
	if err != nil {
		return nil, fmt.Errorf("get items: %w", err)
	}
	defer rows.Close()

	items := make([]*domain.Item, 0, request.Pagination.Limit+1)

	for rows.Next() {
		var item domain.Item

		if err := rows.Scan(
			&item.ID,
			&item.Name,
			&item.Description,
			&item.Category,
			&item.Price,
			&item.Stock,
			&item.XsollaSKU,
			&item.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf("get items: scan: %w", err)
		}

		items = append(items, &item)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("get items: rows: %w", err)
	}

	return items, nil
}

func (s PostgresStore) GetItemByID(
	ctx context.Context,
	id int,
) (*domain.Item, error) {

	var item domain.Item

	err := s.conn.QueryRow(
		ctx,
		`SELECT 
			id,
			name,
			description,
			category,
			price,
			stock,
			COALESCE(xsolla_sku, ''),
			created_at
		FROM items
		WHERE id = $1`,
		id,
	).Scan(
		&item.ID,
		&item.Name,
		&item.Description,
		&item.Category,
		&item.Price,
		&item.Stock,
		&item.XsollaSKU,
		&item.CreatedAt,
	)

	if err != nil {

		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf(
				"get item %d: %w",
				id,
				ErrItemNotFound,
			)
		}

		return nil, fmt.Errorf(
			"get item %d: failed to scan item: %w",
			id,
			err,
		)
	}

	return &item, nil
}

func (s PostgresStore) CreateItem(
	ctx context.Context,
	item *domain.Item,
) error {

	query := `
		INSERT INTO items
			(name, description,category,  price, stock, created_at)
		VALUES
			($1, $2, $3, $4, $5, $6)
		RETURNING id
	`

	err := s.conn.QueryRow(
		ctx,
		query,
		item.Name,
		item.Description,
		&item.Category,
		item.Price,
		item.Stock,
		item.CreatedAt,
	).Scan(&item.ID)

	if err != nil {
		return fmt.Errorf(
			"create item: %w",
			err,
		)
	}

	return nil
}

func (s PostgresStore) UpdateItem(
	ctx context.Context,
	item *domain.Item,
) error {

	query := `
		UPDATE items
		SET
			name = $1,
			description = $2,
			category = $3,
			price = $4,
			stock = $5
		WHERE id = $6
	`

	result, err := s.conn.Exec(
		ctx,
		query,
		item.Name,
		item.Description,
		item.Category,
		item.Price,
		item.Stock,
		item.ID,
	)

	if err != nil {
		return fmt.Errorf(
			"update item %d: %w",
			item.ID,
			err,
		)
	}

	if result.RowsAffected() == 0 {
		return fmt.Errorf(
			"update item %d: %w",
			item.ID,
			ErrItemNotFound,
		)
	}

	return nil
}

func (s PostgresStore) DeleteItem(
	ctx context.Context,
	id int,
) error {

	query := `
		DELETE FROM items
		WHERE id = $1
	`

	result, err := s.conn.Exec(
		ctx,
		query,
		id,
	)

	if err != nil {
		return fmt.Errorf(
			"delete item %d: %w",
			id,
			err,
		)
	}

	if result.RowsAffected() == 0 {
		return fmt.Errorf(
			"delete item %d: %w",
			id,
			ErrItemNotFound,
		)
	}

	return nil
}

//
//func (s PostgresStore) GetItemsByFilter(ctx context.Context, min int, max int, cursor int) ([]*domain.Item, error) {
//
//	var rows pgx.Rows
//	var err error
//
//	if cursor > 0 {
//
//		query := `
//			SELECT
//				id,
//				name,
//				description,
//				price,
//				stock,
//				created_at
//			FROM items
//			WHERE id > $1
//			ORDER BY id ASC
//			LIMIT $2
//		`
//
//		rows, err = s.conn.Query(
//			ctx,
//			query,
//			cursor,
//			limit+1,
//		)
//
//	} else {
//
//		query := `
//			SELECT
//				id,
//				name,
//				description,
//				price,
//				stock,
//				created_at
//			FROM items
//			ORDER BY id ASC
//			LIMIT $1 OFFSET $2
//		`
//
//		rows, err = s.conn.Query(
//			ctx,
//			query,
//			limit+1,
//			offset,
//		)
//	}
//
//	if err != nil {
//		return nil, fmt.Errorf(
//			"get items: failed to execute query: %w",
//			err,
//		)
//	}
//
//	defer rows.Close()
//
//	var items []*domain.Item
//
//	for rows.Next() {
//
//		var item domain.Item
//
//		if err := rows.Scan(
//			&item.ID,
//			&item.Name,
//			&item.Description,
//			&item.Price,
//			&item.Stock,
//			&item.CreatedAt,
//		); err != nil {
//			return nil, fmt.Errorf(
//				"get items: failed to scan item: %w",
//				err,
//			)
//		}
//
//		items = append(items, &item)
//	}
//
//	if err := rows.Err(); err != nil {
//		return nil, fmt.Errorf(
//			"get items: failed while iterating rows: %w",
//			err,
//		)
//	}
//
//	return items, nil
//}
