package user

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
	return &PostgresStore{conn: conn}
}

func (s PostgresStore) CreateUser(
	ctx context.Context,
	user *domain.User,
) error {
	tx, err := s.conn.Begin(ctx)
	if err != nil {
		return fmt.Errorf("create user: begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	query := `
        INSERT INTO users (
            username,
            email,
            password,
            created_at
        )
        VALUES ($1, $2, $3, $4)
        RETURNING id
    `

	err = tx.QueryRow(
		ctx,
		query,
		user.Username,
		user.Email,
		user.Password,
		user.CreatedAt,
	).Scan(&user.ID)

	if err != nil {
		return fmt.Errorf("create user: %w", err)
	}

	_, err = tx.Exec(
		ctx,
		`INSERT INTO user_balances (user_id, balance, created_at, updated_at)
		VALUES ($1, 0, NOW(), NOW())`,
		user.ID,
	)
	if err != nil {
		return fmt.Errorf("create user: create balance: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("create user: commit transaction: %w", err)
	}

	return nil
}

func (s PostgresStore) GetUser(ctx context.Context, id int) (*domain.User, error) {
	var user domain.User

	err := s.conn.QueryRow(ctx,
		`SELECT id, username, email, password, created_at FROM users WHERE id = $1`,
		id,
	).Scan(
		&user.ID, &user.Username, &user.Email,
		&user.Password, &user.CreatedAt,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("get user: %w", ErrUserNotFound)
		}
		return nil, fmt.Errorf("get user: %w", err)
	}

	return &user, nil
}

func (s PostgresStore) GetUserByUsername(ctx context.Context, username string) (*domain.User, error) {
	var user domain.User

	err := s.conn.QueryRow(ctx,
		`SELECT id, username, email, password, created_at
		 FROM users WHERE username = $1`,
		username,
	).Scan(
		&user.ID, &user.Username, &user.Email,
		&user.Password, &user.CreatedAt,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("get user by username: %w", ErrUserNotFound)
		}
		return nil, fmt.Errorf("get user by username: %w", err)
	}

	return &user, nil
}

func (s PostgresStore) GetUserByEmail(ctx context.Context, email string) (*domain.User, error) {
	var user domain.User

	err := s.conn.QueryRow(ctx,
		`SELECT id, username, email, password, created_at
		 FROM users WHERE email = $1`,
		email,
	).Scan(
		&user.ID, &user.Username, &user.Email,
		&user.Password, &user.CreatedAt,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("get user by email: %w", ErrUserNotFound)
		}
		return nil, fmt.Errorf("get user by email: %w", err)
	}

	return &user, nil
}

func (s PostgresStore) UpdateUser(ctx context.Context, user *domain.User) error {
	query := `
		UPDATE users
		SET username = $1, email = $2, password = $3
		WHERE id = $4
	`

	result, err := s.conn.Exec(ctx, query,
		user.Username, user.Email, user.Password, user.ID,
	)

	if err != nil {
		return fmt.Errorf("update user: %w", err)
	}

	if result.RowsAffected() == 0 {
		return fmt.Errorf("update user: %w", ErrUserNotFound)
	}

	return nil
}

func (s PostgresStore) DeleteUser(ctx context.Context, id int) error {
	result, err := s.conn.Exec(ctx,
		`DELETE FROM users WHERE id = $1`,
		id,
	)

	if err != nil {
		return fmt.Errorf("delete user: %w", err)
	}

	if result.RowsAffected() == 0 {
		return fmt.Errorf("delete user: %w", ErrUserNotFound)
	}

	return nil
}

func (s PostgresStore) ListUsers(ctx context.Context, limit int, offset int, cursor string) ([]*domain.User, error) {

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
                username,
                email,
                password,
                created_at
            FROM users
            WHERE id < $1
            ORDER BY id DESC
            LIMIT $2
        `

		rows, err = s.conn.Query(
			ctx,
			query,
			decodedCursor.ID,
			limit+1,
		)

	} else {

		query := `
            SELECT
                id,
                username,
                email,
                password,
                created_at
            FROM users
            ORDER BY id DESC
            LIMIT $1 OFFSET $2
        `

		rows, err = s.conn.Query(
			ctx,
			query,
			limit+1,
			offset,
		)
	}

	if err != nil {
		return nil, fmt.Errorf(
			"list users: %w",
			err,
		)
	}

	defer rows.Close()

	var users []*domain.User

	for rows.Next() {

		var user domain.User

		if err := rows.Scan(
			&user.ID,
			&user.Username,
			&user.Email,
			&user.Password,
			&user.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf(
				"scan user: %w",
				err,
			)
		}

		users = append(users, &user)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf(
			"iterate users: %w",
			err,
		)
	}

	return users, nil
}
