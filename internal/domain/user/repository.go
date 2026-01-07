package user

import (
	"context"
)

type Repository interface {
	Create(ctx context.Context, login, passwordHash string) (int, error)
	FindByLogin(ctx context.Context, login string) (User, error)
}
