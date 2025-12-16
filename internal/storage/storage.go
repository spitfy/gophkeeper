package storage

import (
	"context"
	"encoding/json"
	"time"
)

type Record struct {
	ID            int             `json:"id"`
	Type          string          `json:"type"`
	EncryptedData string          `json:"data"` // base64
	Meta          json.RawMessage `json:"meta"`
	Version       int             `json:"version"`
	LastModified  time.Time       `json:"last_modified"`
}

type Storage interface {
	// Пользователи
	CreateUser(ctx context.Context, login, passwordHash string) (int, error)
	AuthUser(ctx context.Context, login, password string) (int, string, error)

	// Сессии
	CreateSession(ctx context.Context, userID int, tokenHash string, expiresAt time.Time) error
	ValidateSession(ctx context.Context, tokenHash string) (int, error)

	// Записи
	ListRecords(ctx context.Context, userID int) ([]Record, error)
	CreateRecord(ctx context.Context, userID int, typ, encryptedData string, meta json.RawMessage) (int, error)
	GetRecord(ctx context.Context, userID, recordID int) (*Record, error)
	UpdateRecord(ctx context.Context, userID, recordID int, typ, encryptedData string, meta json.RawMessage) error
	DeleteRecord(ctx context.Context, userID, recordID int) error

	Close() error
}
