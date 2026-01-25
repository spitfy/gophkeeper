package record

import (
	"context"
	"time"
)

// Repository расширенный интерфейс репозитория
type Repository interface {
	// Базовые CRUD операции
	List(ctx context.Context, userID int) ([]Record, error)
	Get(ctx context.Context, userID, recordID int) (*Record, error)
	GetByChecksum(ctx context.Context, userID int, checksum string) (*Record, error)
	Create(ctx context.Context, record *Record) (int, error)
	Update(ctx context.Context, record *Record) error
	Delete(ctx context.Context, userID, recordID int) error
	SoftDelete(ctx context.Context, userID, recordID int) error

	// Поиск и фильтрация
	Search(ctx context.Context, userID int, criteria SearchCriteria) ([]Record, error)
	GetByType(ctx context.Context, userID int, recordType string) ([]Record, error)
	GetModifiedSince(ctx context.Context, userID int, since time.Time) ([]Record, error)

	// Статистика
	GetStats(ctx context.Context, userID int) (map[string]interface{}, error)

	// Вспомогательные методы
	SaveVersion(ctx context.Context, version *Version) error
	GetVersions(ctx context.Context, recordID int) ([]Version, error)
}
