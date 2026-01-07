package postgres

import (
	"context"
	"fmt"
	"golang.org/x/exp/slog"
	"time"
)

type SessionRepository struct {
	db  *Storage
	log *slog.Logger
}

func NewSessionRepository(db *Storage, log *slog.Logger) *SessionRepository {
	return &SessionRepository{
		db:  db,
		log: log,
	}
}

func (r *SessionRepository) Create(ctx context.Context, userID int, tokenHash string, expiresAt time.Time) error {
	_, err := r.db.Pool().Exec(ctx,
		`INSERT INTO sessions (user_id, token_hash, expires_at) 
         VALUES ($1, decode($2, 'hex'), $3)`,
		userID, tokenHash, expiresAt)
	return err
}

func (r *SessionRepository) Validate(ctx context.Context, tokenHash string) (int, error) {
	var userID int
	err := r.db.Pool().QueryRow(ctx,
		`SELECT user_id FROM sessions 
         WHERE token_hash = decode($1, 'hex') AND expires_at > NOW()`,
		tokenHash).Scan(&userID)

	if err != nil {
		return 0, fmt.Errorf("invalid session")
	}
	return userID, nil
}
