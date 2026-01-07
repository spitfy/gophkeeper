package session

import (
	"context"
	"time"
)

type Repository interface {
	Create(ctx context.Context, userID int, tokenHash string, expiresAt time.Time) error
	Validate(ctx context.Context, tokenHash string) (int, error)
}
