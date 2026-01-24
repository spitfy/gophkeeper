package postgres

import (
	"context"
	"database/sql"
	"encoding/hex"
	"fmt"
	"github.com/jackc/pgx/v5"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"golang.org/x/exp/slog"

	"gophkeeper/internal/domain/sync"
)

// SyncRepository реализация репозитория синхронизации для PostgreSQL
type SyncRepository struct {
	pool *pgxpool.Pool
	log  *slog.Logger
}

// NewSyncRepository создает новый репозиторий синхронизации
func NewSyncRepository(pool *pgxpool.Pool, log *slog.Logger) *SyncRepository {
	return &SyncRepository{
		pool: pool,
		log:  log,
	}
}

// GetSyncStatus возвращает статус синхронизации пользователя из VIEW
func (r *SyncRepository) GetSyncStatus(ctx context.Context, userID int) (*sync.SyncStatus, error) {
	query := `
		SELECT user_id, last_sync_time, total_records, device_count, 
		       storage_used, storage_limit, sync_version
		FROM sync_status_view
		WHERE user_id = $1
	`

	var status sync.SyncStatus
	var lastSyncTime sql.NullTime

	err := r.pool.QueryRow(ctx, query, userID).Scan(
		&status.UserID,
		&lastSyncTime,
		&status.TotalRecords,
		&status.DeviceCount,
		&status.StorageUsed,
		&status.StorageLimit,
		&status.SyncVersion,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			// Возвращаем пустой статус для нового пользователя
			return &sync.SyncStatus{
				UserID:       userID,
				LastSyncTime: time.Time{},
				TotalRecords: 0,
				DeviceCount:  0,
				StorageUsed:  0,
				StorageLimit: 104857600, // 100 MB
				SyncVersion:  0,
			}, nil
		}
		return nil, fmt.Errorf("failed to get sync status: %w", err)
	}

	if lastSyncTime.Valid {
		status.LastSyncTime = lastSyncTime.Time
	}

	return &status, nil
}

// UpdateSyncStatus обновляет статус синхронизации (теперь это no-op, т.к. используется VIEW)
func (r *SyncRepository) UpdateSyncStatus(_ context.Context, status *sync.SyncStatus) error {
	// VIEW автоматически вычисляет данные, ничего обновлять не нужно
	r.log.Debug("UpdateSyncStatus called but using VIEW, no action needed", "user_id", status.UserID)
	return nil
}

// GetDeviceInfo возвращает информацию об устройстве
func (r *SyncRepository) GetDeviceInfo(ctx context.Context, deviceID int) (*sync.DeviceInfo, error) {
	query := `
		SELECT id, user_id, name, type, last_sync_time, created_at, updated_at, ip_address, user_agent
		FROM devices
		WHERE id = $1
	`

	var device sync.DeviceInfo
	var lastSyncTime sql.NullTime

	err := r.pool.QueryRow(ctx, query, deviceID).Scan(
		&device.ID,
		&device.UserID,
		&device.Name,
		&device.Type,
		&lastSyncTime,
		&device.CreatedAt,
		&device.UpdatedAt,
		&device.IPAddress,
		&device.UserAgent,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, sync.ErrDeviceNotFound
		}
		return nil, fmt.Errorf("failed to get device info: %w", err)
	}

	if lastSyncTime.Valid {
		device.LastSyncTime = lastSyncTime.Time
	}

	return &device, nil
}

// RegisterDevice регистрирует новое устройство
func (r *SyncRepository) RegisterDevice(ctx context.Context, device *sync.DeviceInfo) error {
	query := `
		INSERT INTO devices (id, user_id, name, type, last_sync_time, created_at, updated_at, ip_address, user_agent)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		ON CONFLICT (id) DO UPDATE SET
			name = EXCLUDED.name,
			type = EXCLUDED.type,
			last_sync_time = EXCLUDED.last_sync_time,
			updated_at = EXCLUDED.updated_at,
			ip_address = EXCLUDED.ip_address,
			user_agent = EXCLUDED.user_agent
	`

	_, err := r.pool.Exec(ctx, query,
		device.ID,
		device.UserID,
		device.Name,
		device.Type,
		device.LastSyncTime,
		device.CreatedAt,
		device.UpdatedAt,
		device.IPAddress,
		device.UserAgent,
	)

	if err != nil {
		return fmt.Errorf("failed to register device: %w", err)
	}

	return nil
}

// UpdateDeviceSyncTime обновляет время синхронизации устройства
func (r *SyncRepository) UpdateDeviceSyncTime(ctx context.Context, deviceID int, syncTime time.Time) error {
	query := `
		UPDATE devices
		SET last_sync_time = $1, updated_at = $2
		WHERE id = $3
	`

	_, err := r.pool.Exec(ctx, query, syncTime, time.Now(), deviceID)
	if err != nil {
		return fmt.Errorf("failed to update device sync time: %w", err)
	}

	return nil
}

// ListUserDevices возвращает список устройств пользователя
func (r *SyncRepository) ListUserDevices(ctx context.Context, userID int) ([]*sync.DeviceInfo, error) {
	query := `
		SELECT id, user_id, name, type, last_sync_time, created_at, updated_at
		FROM devices
		WHERE user_id = $1
		ORDER BY last_sync_time DESC
	`

	rows, err := r.pool.Query(ctx, query, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to query user devices: %w", err)
	}
	defer rows.Close()

	var devices []*sync.DeviceInfo
	for rows.Next() {
		var device sync.DeviceInfo
		var lastSyncTime sql.NullTime

		err := rows.Scan(
			&device.ID,
			&device.UserID,
			&device.Name,
			&device.Type,
			&lastSyncTime,
			&device.CreatedAt,
			&device.UpdatedAt,
		)

		if err != nil {
			return nil, fmt.Errorf("failed to scan device: %w", err)
		}

		if lastSyncTime.Valid {
			device.LastSyncTime = lastSyncTime.Time
		}

		devices = append(devices, &device)
	}

	return devices, nil
}

// DeleteDevice удаляет устройство
func (r *SyncRepository) DeleteDevice(ctx context.Context, deviceID int) error {
	query := `DELETE FROM devices WHERE id = $1`

	_, err := r.pool.Exec(ctx, query, deviceID)
	if err != nil {
		return fmt.Errorf("failed to delete device: %w", err)
	}

	return nil
}

// GetRecordsForSync возвращает записи для синхронизации (используем реальную схему records)
func (r *SyncRepository) GetRecordsForSync(ctx context.Context, userID int, lastSyncTime time.Time, limit, offset int) ([]*sync.RecordSync, error) {
	query := `
		SELECT id, user_id, type, encrypted_data, meta, version, last_modified, 
		       deleted_at, checksum, device_id
		FROM records
		WHERE user_id = $1 
			AND last_modified > $2
		ORDER BY last_modified ASC
		LIMIT $3 OFFSET $4
	`

	rows, err := r.pool.Query(ctx, query, userID, lastSyncTime, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to query records for sync: %w", err)
	}
	defer rows.Close()

	var records []*sync.RecordSync
	for rows.Next() {
		rec, err := r.scanRecordSync(rows)
		if err != nil {
			return nil, fmt.Errorf("failed to scan record: %w", err)
		}
		records = append(records, rec)
	}

	return records, nil
}

// GetRecordByID возвращает запись по ID
func (r *SyncRepository) GetRecordByID(ctx context.Context, recordID int) (*sync.RecordSync, error) {
	query := `
		SELECT id, user_id, type, encrypted_data, meta, version, last_modified, 
		       deleted_at, checksum, device_id
		FROM records
		WHERE id = $1
	`

	row := r.pool.QueryRow(ctx, query, recordID)
	rec, err := r.scanRecordSync(row)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, sync.ErrRecordNotFound
		}
		return nil, fmt.Errorf("failed to get record: %w", err)
	}

	return rec, nil
}

// GetRecordVersions возвращает версии записи из record_versions
func (r *SyncRepository) GetRecordVersions(ctx context.Context, recordID int, limit int) ([]*sync.RecordSync, error) {
	query := `
		SELECT rv.id, r.user_id, r.type, rv.encrypted_data, rv.meta, rv.version, 
		       rv.created_at as last_modified, NULL as deleted_at, rv.checksum, r.device_id
		FROM record_versions rv
		JOIN records r ON rv.record_id = r.id
		WHERE rv.record_id = $1
		ORDER BY rv.version DESC
		LIMIT $2
	`

	rows, err := r.pool.Query(ctx, query, recordID, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to query record versions: %w", err)
	}
	defer rows.Close()

	var versions []*sync.RecordSync
	for rows.Next() {
		rec, err := r.scanRecordSync(rows)
		if err != nil {
			return nil, fmt.Errorf("failed to scan record version: %w", err)
		}
		versions = append(versions, rec)
	}

	return versions, nil
}

// SaveRecord сохраняет запись (используем реальную схему records)
func (r *SyncRepository) SaveRecord(ctx context.Context, record *sync.RecordSync) error {
	data, err := hex.DecodeString(record.EncryptedData)
	if err != nil {
		return fmt.Errorf("failed to decode encrypted data: %w", err)
	}

	query := `
		INSERT INTO records (user_id, type, encrypted_data, meta, version, checksum, device_id)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		ON CONFLICT (user_id, type, encrypted_data) 
		WHERE deleted_at IS NULL
		DO UPDATE SET
			meta = EXCLUDED.meta,
			version = EXCLUDED.version,
			checksum = EXCLUDED.checksum,
			device_id = EXCLUDED.device_id,
			last_modified = NOW()
		WHERE records.version < EXCLUDED.version
		RETURNING id, version, last_modified
	`

	err = r.pool.QueryRow(ctx, query,
		record.UserID,
		record.Type,
		data,
		record.Meta,
		record.Version,
		record.Checksum,
		record.DeviceID,
	).Scan(&record.ID, &record.Version, &record.LastModified)

	if err != nil {
		return fmt.Errorf("failed to save record: %w", err)
	}

	return nil
}

// GetSyncConflicts возвращает конфликты синхронизации
func (r *SyncRepository) GetSyncConflicts(ctx context.Context, userID int) ([]*sync.Conflict, error) {
	query := `
		SELECT id, record_id, user_id, device_id, local_data, server_data, 
		       conflict_type, resolved, resolution, resolved_at, created_at, updated_at
		FROM sync_conflicts
		WHERE user_id = $1 AND resolved = false
		ORDER BY created_at DESC
	`

	rows, err := r.pool.Query(ctx, query, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to query sync conflicts: %w", err)
	}
	defer rows.Close()

	var conflicts []*sync.Conflict
	for rows.Next() {
		var conflict sync.Conflict
		var resolvedAt sql.NullTime

		err := rows.Scan(
			&conflict.ID,
			&conflict.RecordID,
			&conflict.UserID,
			&conflict.DeviceID,
			&conflict.LocalData,
			&conflict.ServerData,
			&conflict.ConflictType,
			&conflict.Resolved,
			&conflict.Resolution,
			&resolvedAt,
			&conflict.CreatedAt,
			&conflict.UpdatedAt,
		)

		if err != nil {
			return nil, fmt.Errorf("failed to scan conflict: %w", err)
		}

		if resolvedAt.Valid {
			conflict.ResolvedAt = resolvedAt.Time
		}

		conflicts = append(conflicts, &conflict)
	}

	return conflicts, nil
}

// GetConflictByID возвращает конфликт по ID
func (r *SyncRepository) GetConflictByID(ctx context.Context, conflictID int) (*sync.Conflict, error) {
	query := `
		SELECT id, record_id, user_id, device_id, local_data, server_data, 
		       conflict_type, resolved, resolution, resolved_at, created_at, updated_at
		FROM sync_conflicts
		WHERE id = $1
	`

	var conflict sync.Conflict
	var resolvedAt sql.NullTime

	err := r.pool.QueryRow(ctx, query, conflictID).Scan(
		&conflict.ID,
		&conflict.RecordID,
		&conflict.UserID,
		&conflict.DeviceID,
		&conflict.LocalData,
		&conflict.ServerData,
		&conflict.ConflictType,
		&conflict.Resolved,
		&conflict.Resolution,
		&resolvedAt,
		&conflict.CreatedAt,
		&conflict.UpdatedAt,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, sync.ErrRecordNotFound
		}
		return nil, fmt.Errorf("failed to get conflict: %w", err)
	}

	if resolvedAt.Valid {
		conflict.ResolvedAt = resolvedAt.Time
	}

	return &conflict, nil
}

// SaveConflict сохраняет конфликт
func (r *SyncRepository) SaveConflict(ctx context.Context, conflict *sync.Conflict) error {
	query := `
		INSERT INTO sync_conflicts 
			(record_id, user_id, device_id, local_data, server_data, 
			 conflict_type, resolved, resolution, resolved_at, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
		ON CONFLICT (id) DO UPDATE SET
			local_data = EXCLUDED.local_data,
			server_data = EXCLUDED.server_data,
			resolved = EXCLUDED.resolved,
			resolution = EXCLUDED.resolution,
			resolved_at = EXCLUDED.resolved_at,
			updated_at = EXCLUDED.updated_at
		RETURNING id
	`

	err := r.pool.QueryRow(ctx, query,
		conflict.RecordID,
		conflict.UserID,
		conflict.DeviceID,
		conflict.LocalData,
		conflict.ServerData,
		conflict.ConflictType,
		conflict.Resolved,
		conflict.Resolution,
		conflict.ResolvedAt,
		conflict.CreatedAt,
		conflict.UpdatedAt,
	).Scan(&conflict.ID)

	if err != nil {
		return fmt.Errorf("failed to save conflict: %w", err)
	}

	return nil
}

// ResolveConflict разрешает конфликт
func (r *SyncRepository) ResolveConflict(ctx context.Context, conflictID int, resolution string, resolvedData []byte) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer func(tx pgx.Tx, ctx context.Context) {
		_ = tx.Rollback(ctx)
	}(tx, ctx)

	// Получаем информацию о конфликте
	var recordID int
	var userID int
	var conflictType string
	err = tx.QueryRow(ctx, `
		SELECT record_id, user_id, conflict_type 
		FROM sync_conflicts 
		WHERE id = $1
	`, conflictID).Scan(&recordID, &userID, &conflictType)
	if err != nil {
		return fmt.Errorf("failed to get conflict info: %w", err)
	}

	// Обновляем запись, если нужно
	if conflictType == "version_mismatch" && resolvedData != nil {
		_, err := tx.Exec(ctx, `
			UPDATE records 
			SET encrypted_data = $1, version = version + 1, last_modified = NOW()
			WHERE id = $2 AND user_id = $3
		`, resolvedData, recordID, userID)
		if err != nil {
			return fmt.Errorf("failed to update record: %w", err)
		}
	}

	// Помечаем конфликт как разрешенный
	_, err = tx.Exec(ctx, `
		UPDATE sync_conflicts
		SET resolved = true, resolution = $1, resolved_at = $2, updated_at = $3
		WHERE id = $4
	`, resolution, time.Now(), time.Now(), conflictID)
	if err != nil {
		return fmt.Errorf("failed to resolve conflict: %w", err)
	}

	return tx.Commit(ctx)
}

// BatchUpsertRecords массовое обновление/вставка записей
func (r *SyncRepository) BatchUpsertRecords(ctx context.Context, records []*sync.RecordSync) (int, []int, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return 0, nil, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer func(tx pgx.Tx, ctx context.Context) {
		_ = tx.Rollback(ctx)
	}(tx, ctx)

	var processed int
	var failedIDs []int

	for _, rec := range records {
		data, err := hex.DecodeString(rec.EncryptedData)
		if err != nil {
			failedIDs = append(failedIDs, rec.ID)
			continue
		}

		_, err = tx.Exec(ctx, `
		INSERT INTO records (user_id, type, encrypted_data, meta, version, checksum, device_id)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		ON CONFLICT (user_id, type, encrypted_data) 
		WHERE deleted_at IS NULL
		DO UPDATE SET
			meta = EXCLUDED.meta,
			version = EXCLUDED.version,
			checksum = EXCLUDED.checksum,
			device_id = EXCLUDED.device_id,
			last_modified = NOW()
		WHERE records.version < EXCLUDED.version
		`, rec.UserID, rec.Type, data, rec.Meta, rec.Version, rec.Checksum, rec.DeviceID)

		if err != nil {
			failedIDs = append(failedIDs, rec.ID)
			continue
		}

		processed++
	}

	if err := tx.Commit(ctx); err != nil {
		return 0, getAllIDs(records), fmt.Errorf("failed to commit transaction: %w", err)
	}

	return processed, failedIDs, nil
}

func getAllIDs(records []*sync.RecordSync) []int {
	ids := make([]int, len(records))
	for i, rec := range records {
		ids[i] = rec.ID
	}
	return ids
}

// BatchDeleteRecords массовое удаление записей (soft delete)
func (r *SyncRepository) BatchDeleteRecords(ctx context.Context, recordIDs []int, userID int) error {
	if len(recordIDs) == 0 {
		return nil
	}

	// Создаем плейсхолдеры для IN clause
	placeholders := make([]string, len(recordIDs))
	args := make([]interface{}, len(recordIDs)+2)
	args[0] = userID
	args[1] = time.Now()

	for i, id := range recordIDs {
		placeholders[i] = fmt.Sprintf("$%d", i+3)
		args[i+2] = id
	}

	query := fmt.Sprintf(`
		UPDATE records 
		SET deleted_at = $2, last_modified = $2
		WHERE user_id = $1 AND id IN (%s)
	`, strings.Join(placeholders, ","))

	_, err := r.pool.Exec(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("failed to batch delete records: %w", err)
	}

	return nil
}

// GetSyncStats возвращает статистику синхронизации (заглушка, т.к. таблица удалена)
func (r *SyncRepository) GetSyncStats(_ context.Context, userID int) (*sync.SyncStats, error) {
	// Возвращаем пустую статистику
	return &sync.SyncStats{
		UserID:          userID,
		TotalSyncs:      0,
		LastSync:        time.Time{},
		TotalUploads:    0,
		TotalDownloads:  0,
		TotalConflicts:  0,
		TotalResolved:   0,
		AvgSyncDuration: 0,
		UpdatedAt:       time.Now(),
	}, nil
}

// IncrementSyncStats увеличивает статистику синхронизации (заглушка)
func (r *SyncRepository) IncrementSyncStats(_ context.Context, userID int, _, _ int64) error {
	r.log.Debug("IncrementSyncStats called but sync_stats table removed", "user_id", userID)
	return nil
}

// RecordSyncDuration записывает время синхронизации (заглушка)
func (r *SyncRepository) RecordSyncDuration(_ context.Context, userID int, _ time.Duration) error {
	// Таблица удалена, ничего не делаем
	r.log.Debug("RecordSyncDuration called but sync_stats table removed", "user_id", userID)
	return nil
}

// Вспомогательные методы

// scanRecordSync сканирует RecordSync из row
func (r *SyncRepository) scanRecordSync(row interface {
	Scan(dest ...interface{}) error
}) (*sync.RecordSync, error) {
	var rec sync.RecordSync
	var data []byte
	var deletedAt sql.NullTime

	err := row.Scan(
		&rec.ID,
		&rec.UserID,
		&rec.Type,
		&data,
		&rec.Meta,
		&rec.Version,
		&rec.LastModified,
		&deletedAt,
		&rec.Checksum,
		&rec.DeviceID,
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
