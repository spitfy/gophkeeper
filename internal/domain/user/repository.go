package user

import (
	"context"
	"fmt"
	"golang.org/x/exp/slog"
	"gophkeeper/internal/infrastructure/storage/postgres"
)

type Repository interface {
	Create(ctx context.Context, login, passwordHash string) (int, error)
	FindByLogin(ctx context.Context, login string) (User, error)
}

func NewRepo(db *postgres.Storage, log *slog.Logger) Repository {
	return &repository{
		db:  db,
		log: log,
	}
}

type repository struct {
	db  *postgres.Storage
	log *slog.Logger
}

func (r *repository) Create(ctx context.Context, login, passwordHash string) (int, error) {
	var userID int
	err := r.db.Pool().QueryRow(ctx,
		`INSERT INTO users (login, password_hash) VALUES ($1, $2) RETURNING id`,
		login, passwordHash).Scan(&userID)
	return userID, err
}

func (r *repository) FindByLogin(ctx context.Context, login string) (User, error) {
	var user User
	err := r.db.Pool().QueryRow(ctx,
		`SELECT id, password_hash FROM users WHERE login = $1`, login).
		Scan(&user.ID, &user.Password)
	if err != nil {
		return user, fmt.Errorf("user not found")
	}

	return user, nil
}
