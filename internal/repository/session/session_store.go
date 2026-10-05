package session

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
	return &PostgresStore{conn: conn}
}

func (s *PostgresStore) CreateSession(ctx context.Context, session *domain.Session) error {
	_, err := s.conn.Exec(ctx,
		`INSERT INTO sessions (id, user_id, expires_at) VALUES ($1, $2, $3)`,
		session.ID, session.UserID, session.ExpiresAt,
	)
	if err != nil {
		return fmt.Errorf("create session: %w", err)
	}
	return nil
}

func (s *PostgresStore) GetSession(ctx context.Context, id string) (*domain.Session, error) {
	var session domain.Session

	err := s.conn.QueryRow(ctx,
		`SELECT id, user_id, expires_at FROM sessions WHERE id = $1`,
		id,
	).Scan(&session.ID, &session.UserID, &session.ExpiresAt)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("get session: %w", ErrSessionNotFound)
		}
		return nil, fmt.Errorf("get session: %w", err)
	}

	return &session, nil
}

func (s *PostgresStore) DeleteSession(ctx context.Context, id string) error {
	result, err := s.conn.Exec(ctx,
		`DELETE FROM sessions WHERE id = $1`,
		id,
	)

	if err != nil {
		return fmt.Errorf("delete session: %w", err)
	}

	if result.RowsAffected() == 0 {
		return fmt.Errorf("delete session: %w", ErrSessionNotFound)
	}

	return nil
}
