package session

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"golang.org/x/exp/slog"
	"time"
)

type Servicer interface {
	Create(ctx context.Context, userID int) (string, error)
	Validate(ctx context.Context, token string) (int, error)
}

type Service struct {
	repo Repository
	log  *slog.Logger
}

func NewService(repo Repository, log *slog.Logger) *Service {
	return &Service{
		repo: repo,
		log:  log,
	}
}

func (s *Service) Create(ctx context.Context, userID int) (string, error) {
	// Генерация токена
	tokenBytes := make([]byte, 32)
	if _, err := rand.Read(tokenBytes); err != nil {
		return "", fmt.Errorf("generate token: %w", err)
	}

	token := base64.URLEncoding.EncodeToString(tokenBytes)
	tokenHash := sha256.Sum256([]byte(token))

	expiresAt := time.Now().Add(24 * time.Hour)
	if err := s.repo.Create(ctx, userID, hex.EncodeToString(tokenHash[:]), expiresAt); err != nil {
		return "", fmt.Errorf("save session: %w", err)
	}

	return token, nil
}

func (s *Service) Validate(ctx context.Context, token string) (int, error) {
	tokenHash := sha256.Sum256([]byte(token))

	return s.repo.Validate(ctx, hex.EncodeToString(tokenHash[:]))
}
