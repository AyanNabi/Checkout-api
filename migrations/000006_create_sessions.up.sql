-- +migrate Up

CREATE TABLE sessions (
                          id TEXT PRIMARY KEY,
                          user_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
                          expires_at TIMESTAMP WITH TIME ZONE NOT NULL
);

