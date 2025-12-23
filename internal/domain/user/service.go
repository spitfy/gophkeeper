package user

import (
	"context"
	"fmt"
	"golang.org/x/crypto/bcrypt"
	"golang.org/x/exp/slog"
)

type Servicer interface {
	Register(ctx context.Context, req baseRequest) (int, error)
	Authenticate(ctx context.Context, req baseRequest) (User, error)
}

type Service struct {
	repo Repository
	log  *slog.Logger
}

func NewService(repo Repository, log *slog.Logger) Servicer {
	return &Service{
		repo: repo,
		log:  log,
	}
}

func (s *Service) Register(ctx context.Context, req baseRequest) (int, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		return 0, fmt.Errorf("hash password: %w", err)
	}

	return s.repo.Create(ctx, req.Login, string(hash))
}

func (s *Service) Authenticate(ctx context.Context, req baseRequest) (User, error) {
	var user User
	user, err := s.repo.FindByLogin(ctx, req.Login)
	if err != nil {
		return user, ErrUserNotFound
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(req.Password)); err != nil {
		return user, ErrInvalidCredentials
	}

	return user, nil
}
