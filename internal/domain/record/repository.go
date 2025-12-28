package record

import (
	"context"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/jackc/pgx/v5"
	"time"

	"golang.org/x/exp/slog"
	"gophkeeper/internal/storage/postgres"
)

var (
	ErrNotFound        = errors.New("record not found")
	ErrVersionMismatch = errors.New("version mismatch")
)

// Record представляет запись пользователя
type Record struct {
	ID            int             `json:"id"`
	UserID        int             `json:"user_id"`
	Type          string          `json:"type"`
	EncryptedData string          `json:"encrypted_data,omitempty"`
	Meta          json.RawMessage `json:"meta,omitempty"`
	Version       int             `json:"version"`
	LastModified  time.Time       `json:"last_modified"`
	DeletedAt     *time.Time      `json:"deleted_at,omitempty"`
	Checksum      string          `json:"checksum,omitempty"`
	DeviceID      string          `json:"device_id,omitempty"`
}

type RecordVersion struct {
	ID            int             `json:"id"`
	RecordID      int             `json:"record_id"`
	Version       int             `json:"version"`
	EncryptedData string          `json:"encrypted_data"`
	Meta          json.RawMessage `json:"meta"`
	Checksum      string          `json:"checksum"`
	CreatedAt     time.Time       `json:"created_at"`
}

// RecordWithStatus добавляет статус синхронизации
type RecordWithStatus struct {
	Record
	SyncStatus string `json:"sync_status"`
	DeviceID   string `json:"device_id,omitempty"`
}

// BatchUpdate представляет пакетное обновление записей
type BatchUpdate struct {
	Records []Record `json:"records"`
}

// SearchCriteria критерии поиска записей
type SearchCriteria struct {
	Type      string
	MetaQuery json.RawMessage
	FromDate  *time.Time
	ToDate    *time.Time
	Limit     int
	Offset    int
}

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
	SaveVersion(ctx context.Context, version *RecordVersion) error
	GetVersions(ctx context.Context, recordID int) ([]RecordVersion, error)
}

func NewRepo(db *postgres.Storage, log *slog.Logger) Repository {
	return &repository{
		db:  db,
		log: log.With("component", "record_repository"),
	}
}

type repository struct {
	db  *postgres.Storage
	log *slog.Logger
}

func (r *repository) List(ctx context.Context, userID int) ([]Record, error) {
	const query = `
		SELECT id, user_id, type, encrypted_data, meta, version, last_modified, 
		       checksum, device_id, deleted_at
		FROM records 
		WHERE user_id = $1 AND deleted_at IS NULL 
		ORDER BY last_modified DESC`

	rows, err := r.db.Pool().Query(ctx, query, userID)
	if err != nil {
		r.log.Error("failed to list records", "user_id", userID, "error", err)
		return nil, fmt.Errorf("list records: %w", err)
	}
	defer rows.Close()

	return r.scanRecords(rows)
}

func (r *repository) Get(ctx context.Context, userID, recordID int) (*Record, error) {
	const query = `
		SELECT id, user_id, type, encrypted_data, meta, version, last_modified, 
		       checksum, device_id, deleted_at
		FROM records 
		WHERE id = $1 AND user_id = $2 AND deleted_at IS NULL`

	row := r.db.Pool().QueryRow(ctx, query, recordID, userID)

	record, err := r.scanRecord(row)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		r.log.Error("failed to get record",
			"record_id", recordID, "user_id", userID, "error", err)
		return nil, fmt.Errorf("get record: %w", err)
	}

	if record.DeletedAt != nil {
		return nil, ErrNotFound
	}

	return record, nil
}

func (r *repository) GetByChecksum(ctx context.Context, userID int, checksum string) (*Record, error) {
	const query = `
		SELECT id, user_id, type, encrypted_data, meta, version, last_modified, 
		       checksum, device_id, deleted_at
		FROM records 
		WHERE checksum = $1 AND user_id = $2 AND deleted_at IS NULL`

	row := r.db.Pool().QueryRow(ctx, query, checksum, userID)

	record, err := r.scanRecord(row)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		r.log.Error("failed to get record by checksum",
			"checksum", checksum, "user_id", userID, "error", err)
		return nil, fmt.Errorf("get record by checksum: %w", err)
	}

	return record, nil
}

func (r *repository) Create(ctx context.Context, record *Record) (int, error) {
	const query = `
		INSERT INTO records (user_id, type, encrypted_data, meta, checksum, device_id)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id, version, last_modified`

	data, err := hex.DecodeString(record.EncryptedData)
	if err != nil {
		return 0, fmt.Errorf("%w: %v", ErrInvalidData, err)
	}

	err = r.db.Pool().QueryRow(ctx, query,
		record.UserID, record.Type, data, record.Meta, record.Checksum, record.DeviceID,
	).Scan(&record.ID, &record.Version, &record.LastModified)

	if err != nil {
		r.log.Error("failed to create record",
			"user_id", record.UserID, "type", record.Type, "error", err)
		return 0, fmt.Errorf("create record: %w", err)
	}

	return record.ID, nil
}

func (r *repository) Update(ctx context.Context, record *Record) error {
	const query = `
		UPDATE records 
		SET type = $1, encrypted_data = $2, meta = $3, 
			version = version + 1, last_modified = NOW(),
			checksum = $4, device_id = $5
		WHERE id = $6 AND user_id = $7 AND version = $8 AND deleted_at IS NULL
		RETURNING version, last_modified`

	data, err := hex.DecodeString(record.EncryptedData)
	if err != nil {
		return fmt.Errorf("%w: %v", ErrInvalidData, err)
	}

	var newVersion int
	var newLastModified time.Time

	err = r.db.Pool().QueryRow(ctx, query,
		record.Type, data, record.Meta, record.Checksum, record.DeviceID,
		record.ID, record.UserID, record.Version,
	).Scan(&newVersion, &newLastModified)

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ErrVersionMismatch
		}
		r.log.Error("failed to update record",
			"record_id", record.ID, "user_id", record.UserID, "error", err)
		return fmt.Errorf("update record: %w", err)
	}

	record.Version = newVersion
	record.LastModified = newLastModified
	return nil
}

func (r *repository) Delete(ctx context.Context, userID, recordID int) error {
	const query = `DELETE FROM records WHERE id = $1 AND user_id = $2`

	result, err := r.db.Pool().Exec(ctx, query, recordID, userID)
	if err != nil {
		r.log.Error("failed to delete record",
			"record_id", recordID, "user_id", userID, "error", err)
		return fmt.Errorf("delete record: %w", err)
	}

	if result.RowsAffected() == 0 {
		return ErrNotFound
	}

	return nil
}

func (r *repository) SoftDelete(ctx context.Context, userID, recordID int) error {
	const query = `
		UPDATE records 
		SET deleted_at = NOW(), version = version + 1
		WHERE id = $1 AND user_id = $2 AND deleted_at IS NULL`

	result, err := r.db.Pool().Exec(ctx, query, recordID, userID)
	if err != nil {
		r.log.Error("failed to soft delete record",
			"record_id", recordID, "user_id", userID, "error", err)
		return fmt.Errorf("soft delete record: %w", err)
	}

	if result.RowsAffected() == 0 {
		return ErrNotFound
	}

	return nil
}

func (r *repository) Search(ctx context.Context, userID int, criteria SearchCriteria) ([]Record, error) {
	query := `
		SELECT id, user_id, type, encrypted_data, meta, version, last_modified, 
		       checksum, device_id, deleted_at
		FROM records 
		WHERE user_id = $1 AND deleted_at IS NULL`

	args := []interface{}{userID}
	argIndex := 2

	if criteria.Type != "" {
		query += fmt.Sprintf(" AND type = $%d", argIndex)
		args = append(args, criteria.Type)
		argIndex++
	}

	if criteria.FromDate != nil {
		query += fmt.Sprintf(" AND last_modified >= $%d", argIndex)
		args = append(args, criteria.FromDate)
		argIndex++
	}

	if criteria.ToDate != nil {
		query += fmt.Sprintf(" AND last_modified <= $%d", argIndex)
		args = append(args, criteria.ToDate)
		argIndex++
	}

	query += " ORDER BY last_modified DESC"

	if criteria.Limit > 0 {
		query += fmt.Sprintf(" LIMIT $%d", argIndex)
		args = append(args, criteria.Limit)
		argIndex++

		if criteria.Offset > 0 {
			query += fmt.Sprintf(" OFFSET $%d", argIndex)
			args = append(args, criteria.Offset)
		}
	}

	rows, err := r.db.Pool().Query(ctx, query, args...)
	if err != nil {
		r.log.Error("failed to search records", "criteria", criteria, "error", err)
		return nil, fmt.Errorf("search records: %w", err)
	}
	defer rows.Close()

	return r.scanRecords(rows)
}

func (r *repository) GetByType(ctx context.Context, userID int, recordType string) ([]Record, error) {
	return r.Search(ctx, userID, SearchCriteria{Type: recordType})
}

func (r *repository) GetModifiedSince(ctx context.Context, userID int, since time.Time) ([]Record, error) {
	const query = `
		SELECT id, user_id, type, encrypted_data, meta, version, last_modified, 
		       checksum, device_id, deleted_at
		FROM records 
		WHERE user_id = $1 AND last_modified > $2 AND deleted_at IS NULL
		ORDER BY last_modified DESC`

	rows, err := r.db.Pool().Query(ctx, query, userID, since)
	if err != nil {
		r.log.Error("failed to get modified records",
			"user_id", userID, "since", since, "error", err)
		return nil, fmt.Errorf("get modified records: %w", err)
	}
	defer rows.Close()

	return r.scanRecords(rows)
}

func (r *repository) GetStats(ctx context.Context, userID int) (map[string]interface{}, error) {
	const query = `
		SELECT 
			type,
			COUNT(*) as count,
			COALESCE(SUM(LENGTH(encrypted_data)), 0) as total_size
		FROM records 
		WHERE user_id = $1 AND deleted_at IS NULL
		GROUP BY type`

	rows, err := r.db.Pool().Query(ctx, query, userID)
	if err != nil {
		r.log.Error("failed to get stats", "user_id", userID, "error", err)
		return nil, fmt.Errorf("get stats: %w", err)
	}
	defer rows.Close()

	stats := make(map[string]interface{})
	typeStats := make(map[string]map[string]interface{})
	var totalRecords, totalSize int64

	for rows.Next() {
		var recordType string
		var count int64
		var size int64

		err := rows.Scan(&recordType, &count, &size)
		if err != nil {
			return nil, fmt.Errorf("scan stat: %w", err)
		}

		typeStats[recordType] = map[string]interface{}{
			"count": count,
			"size":  size,
		}
		totalRecords += count
		totalSize += size
	}

	stats["by_type"] = typeStats
	stats["total_records"] = totalRecords
	stats["total_size"] = totalSize
	stats["user_id"] = userID

	return stats, nil
}

func (r *repository) SaveVersion(ctx context.Context, version *RecordVersion) error {
	const query = `
		INSERT INTO record_versions (record_id, version, encrypted_data, meta, checksum)
		VALUES ($1, $2, $3, $4, $5)`

	data, err := hex.DecodeString(version.EncryptedData)
	if err != nil {
		return fmt.Errorf("decode encrypted data: %w", err)
	}

	_, err = r.db.Pool().Exec(ctx, query,
		version.RecordID, version.Version, data, version.Meta, version.Checksum)

	return err
}

func (r *repository) GetVersions(ctx context.Context, recordID int) ([]RecordVersion, error) {
	const query = `
		SELECT id, record_id, version, encrypted_data, meta, checksum, created_at
		FROM record_versions
		WHERE record_id = $1
		ORDER BY version DESC`

	rows, err := r.db.Pool().Query(ctx, query, recordID)
	if err != nil {
		return nil, fmt.Errorf("get versions: %w", err)
	}
	defer rows.Close()

	var versions []RecordVersion
	for rows.Next() {
		var v RecordVersion
		var data []byte

		err := rows.Scan(&v.ID, &v.RecordID, &v.Version, &data, &v.Meta, &v.Checksum, &v.CreatedAt)
		if err != nil {
			return nil, fmt.Errorf("scan version: %w", err)
		}

		v.EncryptedData = hex.EncodeToString(data)
		versions = append(versions, v)
	}

	return versions, nil
}

// Вспомогательные методы
func (r *repository) scanRecords(rows pgx.Rows) ([]Record, error) {
	var records []Record

	for rows.Next() {
		record, err := r.scanRecord(rows)
		if err != nil {
			return nil, err
		}
		records = append(records, *record)
	}

	return records, rows.Err()
}

func (r *repository) scanRecord(row interface {
	Scan(dest ...interface{}) error
}) (*Record, error) {
	var rec Record
	var data []byte
	var deletedAt sql.NullTime

	err := row.Scan(
		&rec.ID, &rec.UserID, &rec.Type, &data,
		&rec.Meta, &rec.Version, &rec.LastModified,
		&rec.Checksum, &rec.DeviceID, &deletedAt,
	)

	if err != nil {
		return nil, err
	}

	rec.EncryptedData = hex.EncodeToString(data)
	if deletedAt.Valid {
		rec.DeletedAt = &deletedAt.Time
	}

	return &rec, nil
}
