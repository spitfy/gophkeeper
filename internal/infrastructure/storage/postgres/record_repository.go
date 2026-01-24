package postgres

import (
	"context"
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"
	"github.com/jackc/pgx/v5/pgxpool"
	"gophkeeper/internal/domain/record"
	"time"

	"github.com/jackc/pgx/v5"
	"golang.org/x/exp/slog"
)

type RecordRepository struct {
	pool *pgxpool.Pool
	log  *slog.Logger
}

func NewRecordRepository(pool *pgxpool.Pool, log *slog.Logger) *RecordRepository {
	return &RecordRepository{
		pool: pool,
		log:  log.With("component", "record_repository"),
	}
}

func (r *RecordRepository) List(ctx context.Context, userID int) ([]record.Record, error) {
	const query = `
		SELECT id, user_id, type, encrypted_data, meta, version, last_modified, 
		       checksum, device_id, deleted_at
		FROM records 
		WHERE user_id = $1 AND deleted_at IS NULL 
		ORDER BY last_modified DESC`

	rows, err := r.pool.Query(ctx, query, userID)
	if err != nil {
		r.log.Error("failed to list records", "user_id", userID, "error", err)
		return nil, fmt.Errorf("list records: %w", err)
	}
	defer rows.Close()

	return r.scanRecords(rows)
}

func (r *RecordRepository) Get(ctx context.Context, userID, recordID int) (*record.Record, error) {
	const query = `
		SELECT id, user_id, type, encrypted_data, meta, version, last_modified, 
		       checksum, device_id, deleted_at
		FROM records 
		WHERE id = $1 AND user_id = $2 AND deleted_at IS NULL`

	row := r.pool.QueryRow(ctx, query, recordID, userID)

	rec, err := r.scanRecord(row)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, record.ErrNotFound
		}
		r.log.Error("failed to get record",
			"record_id", recordID, "user_id", userID, "error", err)
		return nil, fmt.Errorf("get record: %w", err)
	}

	if rec.DeletedAt != nil {
		return nil, record.ErrNotFound
	}

	return rec, nil
}

func (r *RecordRepository) GetByChecksum(ctx context.Context, userID int, checksum string) (*record.Record, error) {
	const query = `
		SELECT id, user_id, type, encrypted_data, meta, version, last_modified, 
		       checksum, device_id, deleted_at
		FROM records 
		WHERE checksum = $1 AND user_id = $2 AND deleted_at IS NULL`

	row := r.pool.QueryRow(ctx, query, checksum, userID)

	rec, err := r.scanRecord(row)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, record.ErrNotFound
		}
		r.log.Error("failed to get record by checksum",
			"checksum", checksum, "user_id", userID, "error", err)
		return nil, fmt.Errorf("get record by checksum: %w", err)
	}

	return rec, nil
}

func (r *RecordRepository) Create(ctx context.Context, rec *record.Record) (int, error) {
	const query = `
		INSERT INTO records (user_id, type, encrypted_data, meta, checksum, device_id)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id, version, last_modified`

	data, err := hex.DecodeString(rec.EncryptedData)
	if err != nil {
		return 0, fmt.Errorf("%w: %v", record.ErrInvalidData, err)
	}

	err = r.pool.QueryRow(ctx, query,
		rec.UserID, rec.Type, data, rec.Meta, rec.Checksum, rec.DeviceID,
	).Scan(&rec.ID, &rec.Version, &rec.LastModified)

	if err != nil {
		r.log.Error("failed to create record",
			"user_id", rec.UserID, "type", rec.Type, "error", err)
		return 0, fmt.Errorf("create record: %w", err)
	}

	return rec.ID, nil
}

func (r *RecordRepository) Update(ctx context.Context, rec *record.Record) error {
	const query = `
		UPDATE records 
		SET type = $1, encrypted_data = $2, meta = $3, 
			version = version + 1, last_modified = NOW(),
			checksum = $4, device_id = $5
		WHERE id = $6 AND user_id = $7 AND version = $8 AND deleted_at IS NULL
		RETURNING version, last_modified`

	data, err := hex.DecodeString(rec.EncryptedData)
	if err != nil {
		return fmt.Errorf("%w: %v", record.ErrInvalidData, err)
	}

	var newVersion int
	var newLastModified time.Time

	err = r.pool.QueryRow(ctx, query,
		rec.Type, data, rec.Meta, rec.Checksum, rec.DeviceID,
		rec.ID, rec.UserID, rec.Version,
	).Scan(&newVersion, &newLastModified)

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return record.ErrVersionConflict
		}
		r.log.Error("failed to update record",
			"record_id", rec.ID, "user_id", rec.UserID, "error", err)
		return fmt.Errorf("update record: %w", err)
	}

	rec.Version = newVersion
	rec.LastModified = newLastModified
	return nil
}

func (r *RecordRepository) Delete(ctx context.Context, userID, recordID int) error {
	const query = `DELETE FROM records WHERE id = $1 AND user_id = $2`

	result, err := r.pool.Exec(ctx, query, recordID, userID)
	if err != nil {
		r.log.Error("failed to delete record",
			"record_id", recordID, "user_id", userID, "error", err)
		return fmt.Errorf("delete record: %w", err)
	}

	if result.RowsAffected() == 0 {
		return record.ErrNotFound
	}

	return nil
}

func (r *RecordRepository) SoftDelete(ctx context.Context, userID, recordID int) error {
	const query = `
		UPDATE records 
		SET deleted_at = NOW(), version = version + 1
		WHERE id = $1 AND user_id = $2 AND deleted_at IS NULL`

	result, err := r.pool.Exec(ctx, query, recordID, userID)
	if err != nil {
		r.log.Error("failed to soft delete record",
			"record_id", recordID, "user_id", userID, "error", err)
		return fmt.Errorf("soft delete record: %w", err)
	}

	if result.RowsAffected() == 0 {
		return record.ErrNotFound
	}

	return nil
}

func (r *RecordRepository) Search(ctx context.Context, userID int, criteria record.SearchCriteria) ([]record.Record, error) {
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

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		r.log.Error("failed to search records", "criteria", criteria, "error", err)
		return nil, fmt.Errorf("search records: %w", err)
	}
	defer rows.Close()

	return r.scanRecords(rows)
}

func (r *RecordRepository) GetByType(ctx context.Context, userID int, recordType string) ([]record.Record, error) {
	return r.Search(ctx, userID, record.SearchCriteria{Type: recordType})
}

func (r *RecordRepository) GetModifiedSince(ctx context.Context, userID int, since time.Time) ([]record.Record, error) {
	const query = `
		SELECT id, user_id, type, encrypted_data, meta, version, last_modified, 
		       checksum, device_id, deleted_at
		FROM records 
		WHERE user_id = $1 AND last_modified > $2 AND deleted_at IS NULL
		ORDER BY last_modified DESC`

	rows, err := r.pool.Query(ctx, query, userID, since)
	if err != nil {
		r.log.Error("failed to get modified records",
			"user_id", userID, "since", since, "error", err)
		return nil, fmt.Errorf("get modified records: %w", err)
	}
	defer rows.Close()

	return r.scanRecords(rows)
}

func (r *RecordRepository) GetStats(ctx context.Context, userID int) (map[string]interface{}, error) {
	const query = `
		SELECT 
			type,
			COUNT(*) as count,
			COALESCE(SUM(LENGTH(encrypted_data)), 0) as total_size
		FROM records 
		WHERE user_id = $1 AND deleted_at IS NULL
		GROUP BY type`

	rows, err := r.pool.Query(ctx, query, userID)
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

func (r *RecordRepository) SaveVersion(ctx context.Context, version *record.RecordVersion) error {
	const query = `
		INSERT INTO record_versions (record_id, version, encrypted_data, meta, checksum)
		VALUES ($1, $2, $3, $4, $5)`

	data, err := hex.DecodeString(version.EncryptedData)
	if err != nil {
		return fmt.Errorf("decode encrypted data: %w", err)
	}

	_, err = r.pool.Exec(ctx, query,
		version.RecordID, version.Version, data, version.Meta, version.Checksum)

	return err
}

func (r *RecordRepository) GetVersions(ctx context.Context, recordID int) ([]record.RecordVersion, error) {
	const query = `
		SELECT id, record_id, version, encrypted_data, meta, checksum, created_at
		FROM record_versions
		WHERE record_id = $1
		ORDER BY version DESC`

	rows, err := r.pool.Query(ctx, query, recordID)
	if err != nil {
		return nil, fmt.Errorf("get versions: %w", err)
	}
	defer rows.Close()

	var versions []record.RecordVersion
	for rows.Next() {
		var v record.RecordVersion
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
func (r *RecordRepository) scanRecords(rows pgx.Rows) ([]record.Record, error) {
	var records []record.Record

	for rows.Next() {
		rec, err := r.scanRecord(rows)
		if err != nil {
			return nil, err
		}
		records = append(records, *rec)
	}

	return records, rows.Err()
}

func (r *RecordRepository) scanRecord(row interface {
	Scan(dest ...interface{}) error
}) (*record.Record, error) {
	var rec record.Record
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
