package postgres

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"gophkeeper/internal/domain/sync"
)

// SyncRepository реализация репозитория синхронизации для PostgreSQL
type SyncRepository struct {
	db  *sql.DB
	log *slog.Logger
}

// NewSyncRepository создает новый репозиторий синхронизации
func NewSyncRepository(db *sql.DB, log *slog.Logger) *SyncRepository {
	return &SyncRepository{
		db:  db,
		log: log,
	}
}

// GetSyncStatus возвращает статус синхронизации пользователя
func (r *SyncRepository) GetSyncStatus(ctx context.Context, userID int) (*sync.SyncStatus, error) {
	query := `
		SELECT user_id, last_sync_time, total_records, device_count, 
		       storage_used, storage_limit, sync_version
		FROM sync_status
		WHERE user_id = $1
	`

	var status sync.SyncStatus
	var lastSyncTime sql.NullTime

	err := r.db.QueryRowContext(ctx, query, userID).Scan(
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
			// Создаем начальный статус
			return r.createInitialSyncStatus(ctx, userID)
		}
		return nil, fmt.Errorf("failed to get sync status: %w", err)
	}

	if lastSyncTime.Valid {
		status.LastSyncTime = lastSyncTime.Time
	}

	return &status, nil
}

// UpdateSyncStatus обновляет статус синхронизации
func (r *SyncRepository) UpdateSyncStatus(ctx context.Context, status *sync.SyncStatus) error {
	query := `
		INSERT INTO sync_status 
			(user_id, last_sync_time, total_records, device_count, 
			 storage_used, storage_limit, sync_version, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		ON CONFLICT (user_id) DO UPDATE SET
			last_sync_time = EXCLUDED.last_sync_time,
			total_records = EXCLUDED.total_records,
			device_count = EXCLUDED.device_count,
			storage_used = EXCLUDED.storage_used,
			storage_limit = EXCLUDED.storage_limit,
			sync_version = EXCLUDED.sync_version,
			updated_at = EXCLUDED.updated_at
	`

	_, err := r.db.ExecContext(ctx, query,
		status.UserID,
		status.LastSyncTime,
		status.TotalRecords,
		status.DeviceCount,
		status.StorageUsed,
		status.StorageLimit,
		status.SyncVersion,
		time.Now(),
	)

	if err != nil {
		return fmt.Errorf("failed to update sync status: %w", err)
	}

	return nil
}

// GetDeviceInfo возвращает информацию об устройстве
func (r *SyncRepository) GetDeviceInfo(ctx context.Context, deviceID string) (*sync.DeviceInfo, error) {
	query := `
		SELECT id, user_id, name, type, last_sync_time, created_at, updated_at, ip_address, user_agent
		FROM devices
		WHERE id = $1
	`

	var device sync.DeviceInfo
	var lastSyncTime sql.NullTime

	err := r.db.QueryRowContext(ctx, query, deviceID).Scan(
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

	_, err := r.db.ExecContext(ctx, query,
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

	// Обновляем счетчик устройств в статусе
	status, err := r.GetSyncStatus(ctx, device.UserID)
	if err != nil {
		return fmt.Errorf("failed to get sync status: %w", err)
	}

	status.DeviceCount++
	return r.UpdateSyncStatus(ctx, status)
}

// UpdateDeviceSyncTime обновляет время синхронизации устройства
func (r *SyncRepository) UpdateDeviceSyncTime(ctx context.Context, deviceID string, syncTime time.Time) error {
	query := `
		UPDATE devices
		SET last_sync_time = $1, updated_at = $2
		WHERE id = $3
	`

	_, err := r.db.ExecContext(ctx, query, syncTime, time.Now(), deviceID)
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

	rows, err := r.db.QueryContext(ctx, query, userID)
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
func (r *SyncRepository) DeleteDevice(ctx context.Context, deviceID string) error {
	query := `DELETE FROM devices WHERE id = $1`

	_, err := r.db.ExecContext(ctx, query, deviceID)
	if err != nil {
		return fmt.Errorf("failed to delete device: %w", err)
	}

	return nil
}

// GetRecordsForSync возвращает записи для синхронизации
func (r *SyncRepository) GetRecordsForSync(ctx context.Context, userID int, lastSyncTime time.Time, limit, offset int) ([]*sync.RecordSync, error) {
	query := `
		SELECT id, user_id, type, metadata, data, version, deleted, created_at, updated_at
		FROM records
		WHERE user_id = $1 
			AND (updated_at > $2 OR created_at > $2)
			AND deleted = false
		ORDER BY updated_at ASC
		LIMIT $3 OFFSET $4
	`

	rows, err := r.db.QueryContext(ctx, query, userID, lastSyncTime, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to query records for sync: %w", err)
	}
	defer rows.Close()

	var records []*sync.RecordSync
	for rows.Next() {
		var rec sync.RecordSync
		var metadataJSON string

		err := rows.Scan(
			&rec.ID,
			&rec.UserID,
			&rec.Type,
			&metadataJSON,
			&rec.Data,
			&rec.Version,
			&rec.Deleted,
			&rec.CreatedAt,
			&rec.UpdatedAt,
		)

		if err != nil {
			return nil, fmt.Errorf("failed to scan record: %w", err)
		}

		// Парсим метаданные
		if err := json.Unmarshal([]byte(metadataJSON), &rec.Metadata); err != nil {
			return nil, fmt.Errorf("failed to parse metadata: %w", err)
		}

		records = append(records, &rec)
	}

	return records, nil
}

// GetRecordByID возвращает запись по ID
func (r *SyncRepository) GetRecordByID(ctx context.Context, recordID string) (*sync.RecordSync, error) {
	query := `
		SELECT id, user_id, type, metadata, data, version, deleted, created_at, updated_at
		FROM records
		WHERE id = $1 AND deleted = false
	`

	var rec sync.RecordSync
	var metadataJSON string

	err := r.db.QueryRowContext(ctx, query, recordID).Scan(
		&rec.ID,
		&rec.UserID,
		&rec.Type,
		&metadataJSON,
		&rec.Data,
		&rec.Version,
		&rec.Deleted,
		&rec.CreatedAt,
		&rec.UpdatedAt,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, sync.ErrRecordNotFound
		}
		return nil, fmt.Errorf("failed to get record: %w", err)
	}

	// Парсим метаданные
	if err := json.Unmarshal([]byte(metadataJSON), &rec.Metadata); err != nil {
		return nil, fmt.Errorf("failed to parse metadata: %w", err)
	}

	return &rec, nil
}

// GetRecordVersions возвращает версии записи
func (r *SyncRepository) GetRecordVersions(ctx context.Context, recordID string, limit int) ([]*sync.RecordSync, error) {
	query := `
		SELECT id, user_id, type, metadata, data, version, deleted, created_at, updated_at
		FROM records
		WHERE id = $1
		ORDER BY version DESC
		LIMIT $2
	`

	rows, err := r.db.QueryContext(ctx, query, recordID, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to query record versions: %w", err)
	}
	defer rows.Close()

	var versions []*sync.RecordSync
	for rows.Next() {
		var rec sync.RecordSync
		var metadataJSON string

		err := rows.Scan(
			&rec.ID,
			&rec.UserID,
			&rec.Type,
			&metadataJSON,
			&rec.Data,
			&rec.Version,
			&rec.Deleted,
			&rec.CreatedAt,
			&rec.UpdatedAt,
		)

		if err != nil {
			return nil, fmt.Errorf("failed to scan record version: %w", err)
		}

		// Парсим метаданные
		if err := json.Unmarshal([]byte(metadataJSON), &rec.Metadata); err != nil {
			return nil, fmt.Errorf("failed to parse metadata: %w", err)
		}

		versions = append(versions, &rec)
	}

	return versions, nil
}

// SaveRecord сохраняет запись
func (r *SyncRepository) SaveRecord(ctx context.Context, record *sync.RecordSync) error {
	metadataJSON, err := json.Marshal(record.Metadata)
	if err != nil {
		return fmt.Errorf("failed to marshal metadata: %w", err)
	}

	query := `
		INSERT INTO records (id, user_id, type, metadata, data, version, deleted, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		ON CONFLICT (id) DO UPDATE SET
			type = EXCLUDED.type,
			metadata = EXCLUDED.metadata,
			data = EXCLUDED.data,
			version = EXCLUDED.version,
			deleted = EXCLUDED.deleted,
			updated_at = EXCLUDED.updated_at
		WHERE records.version < EXCLUDED.version
	`

	_, err = r.db.ExecContext(ctx, query,
		record.ID,
		record.UserID,
		record.Type,
		metadataJSON,
		record.Data,
		record.Version,
		record.Deleted,
		record.CreatedAt,
		record.UpdatedAt,
	)

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

	rows, err := r.db.QueryContext(ctx, query, userID)
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
func (r *SyncRepository) GetConflictByID(ctx context.Context, conflictID string) (*sync.Conflict, error) {
	query := `
		SELECT id, record_id, user_id, device_id, local_data, server_data, 
		       conflict_type, resolved, resolution, resolved_at, created_at, updated_at
		FROM sync_conflicts
		WHERE id = $1
	`

	var conflict sync.Conflict
	var resolvedAt sql.NullTime

	err := r.db.QueryRowContext(ctx, query, conflictID).Scan(
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
			(id, record_id, user_id, device_id, local_data, server_data, 
			 conflict_type, resolved, resolution, resolved_at, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
		ON CONFLICT (id) DO UPDATE SET
			local_data = EXCLUDED.local_data,
			server_data = EXCLUDED.server_data,
			resolved = EXCLUDED.resolved,
			resolution = EXCLUDED.resolution,
			resolved_at = EXCLUDED.resolved_at,
			updated_at = EXCLUDED.updated_at
	`

	_, err := r.db.ExecContext(ctx, query,
		conflict.ID,
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
	)

	if err != nil {
		return fmt.Errorf("failed to save conflict: %w", err)
	}

	return nil
}

// ResolveConflict разрешает конфликт
func (r *SyncRepository) ResolveConflict(ctx context.Context, conflictID, resolution string, resolvedData []byte) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	// Получаем информацию о конфликте
	var recordID string
	var userID int
	var conflictType string
	err = tx.QueryRowContext(ctx, `
		SELECT record_id, user_id, conflict_type 
		FROM sync_conflicts 
		WHERE id = $1
	`, conflictID).Scan(&recordID, &userID, &conflictType)
	if err != nil {
		return fmt.Errorf("failed to get conflict info: %w", err)
	}

	// Обновляем запись, если нужно
	if conflictType == "version_mismatch" && resolvedData != nil {
		_, err := tx.ExecContext(ctx, `
			UPDATE records 
			SET data = $1, version = version + 1, updated_at = $2
			WHERE id = $3 AND user_id = $4
		`, resolvedData, time.Now(), recordID, userID)
		if err != nil {
			return fmt.Errorf("failed to update record: %w", err)
		}
	}

	// Помечаем конфликт как разрешенный
	_, err = tx.ExecContext(ctx, `
		UPDATE sync_conflicts
		SET resolved = true, resolution = $1, resolved_at = $2, updated_at = $3
		WHERE id = $4
	`, resolution, time.Now(), time.Now(), conflictID)
	if err != nil {
		return fmt.Errorf("failed to resolve conflict: %w", err)
	}

	return tx.Commit()
}

// BatchUpsertRecords массовое обновление/вставка записей
func (r *SyncRepository) BatchUpsertRecords(ctx context.Context, records []*sync.RecordSync) (int, []string, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return 0, nil, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	var processed int
	var errors []string

	stmt, err := tx.PrepareContext(ctx, `
		INSERT INTO records (id, user_id, type, metadata, data, version, deleted, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		ON CONFLICT (id) DO UPDATE SET
			type = EXCLUDED.type,
			metadata = EXCLUDED.metadata,
			data = EXCLUDED.data,
			version = EXCLUDED.version,
			deleted = EXCLUDED.deleted,
			updated_at = EXCLUDED.updated_at
		WHERE records.version < EXCLUDED.version
	`)
	if err != nil {
		return 0, nil, fmt.Errorf("failed to prepare statement: %w", err)
	}
	defer stmt.Close()

	for _, rec := range records {
		metadataJSON, err := json.Marshal(rec.Metadata)
		if err != nil {
			errors = append(errors, fmt.Sprintf("record %s: failed to marshal metadata: %v", rec.ID, err))
			continue
		}

		_, err = stmt.ExecContext(ctx,
			rec.ID,
			rec.UserID,
			rec.Type,
			metadataJSON,
			rec.Data,
			rec.Version,
			rec.Deleted,
			rec.CreatedAt,
			rec.UpdatedAt,
		)

		if err != nil {
			errors = append(errors, fmt.Sprintf("record %s: failed to upsert: %v", rec.ID, err))
			continue
		}

		processed++
	}

	if err := tx.Commit(); err != nil {
		return 0, errors, fmt.Errorf("failed to commit transaction: %w", err)
	}

	return processed, errors, nil
}

// BatchDeleteRecords массовое удаление записей
func (r *SyncRepository) BatchDeleteRecords(ctx context.Context, recordIDs []string, userID int) error {
	if len(recordIDs) == 0 {
		return nil
	}

	// Создаем плейсхолдеры для IN clause
	placeholders := make([]string, len(recordIDs))
	args := make([]interface{}, len(recordIDs)+1)
	args[0] = userID

	for i, id := range recordIDs {
		placeholders[i] = fmt.Sprintf("$%d", i+2)
		args[i+1] = id
	}

	query := fmt.Sprintf(`
		UPDATE records 
		SET deleted = true, updated_at = $%d 
		WHERE user_id = $1 AND id IN (%s)
	`, len(recordIDs)+2, strings.Join(placeholders, ","))

	args = append(args, time.Now())

	_, err := r.db.ExecContext(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("failed to batch delete records: %w", err)
	}

	return nil
}

// GetSyncStats возвращает статистику синхронизации
func (r *SyncRepository) GetSyncStats(ctx context.Context, userID int) (*sync.SyncStats, error) {
	query := `
		SELECT user_id, total_syncs, last_sync, total_uploads, 
		       total_downloads, total_conflicts, total_resolved, 
		       avg_sync_duration, updated_at
		FROM sync_stats
		WHERE user_id = $1
	`

	var stats sync.SyncStats
	var lastSync sql.NullTime

	err := r.db.QueryRowContext(ctx, query, userID).Scan(
		&stats.UserID,
		&stats.TotalSyncs,
		&lastSync,
		&stats.TotalUploads,
		&stats.TotalDownloads,
		&stats.TotalConflicts,
		&stats.TotalResolved,
		&stats.AvgSyncDuration,
		&stats.UpdatedAt,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			// Создаем начальную статистику
			return r.createInitialSyncStats(ctx, userID)
		}
		return nil, fmt.Errorf("failed to get sync stats: %w", err)
	}

	if lastSync.Valid {
		stats.LastSync = lastSync.Time
	}

	return &stats, nil
}

// IncrementSyncStats увеличивает статистику синхронизации
func (r *SyncRepository) IncrementSyncStats(ctx context.Context, userID int, uploads, downloads int64) error {
	query := `
		INSERT INTO sync_stats (user_id, total_syncs, total_uploads, total_downloads, updated_at)
		VALUES ($1, 1, $2, $3, $4)
		ON CONFLICT (user_id) DO UPDATE SET
			total_syncs = sync_stats.total_syncs + 1,
			total_uploads = sync_stats.total_uploads + EXCLUDED.total_uploads,
			total_downloads = sync_stats.total_downloads + EXCLUDED.total_downloads,
			last_sync = EXCLUDED.updated_at,
			updated_at = EXCLUDED.updated_at
	`

	_, err := r.db.ExecContext(ctx, query,
		userID,
		uploads,
		downloads,
		time.Now(),
	)

	if err != nil {
		return fmt.Errorf("failed to increment sync stats: %w", err)
	}

	return nil
}

// RecordSyncDuration записывает время синхронизации
func (r *SyncRepository) RecordSyncDuration(ctx context.Context, userID int, duration time.Duration) error {
	query := `
		UPDATE sync_stats
		SET avg_sync_duration = 
			CASE 
				WHEN total_syncs = 1 THEN $1
				ELSE (avg_sync_duration * (total_syncs - 1) + $1) / total_syncs
			END,
			updated_at = $2
		WHERE user_id = $3
	`

	_, err := r.db.ExecContext(ctx, query,
		duration.Seconds(),
		time.Now(),
		userID,
	)

	if err != nil {
		return fmt.Errorf("failed to record sync duration: %w", err)
	}

	return nil
}

// Вспомогательные методы

func (r *SyncRepository) createInitialSyncStatus(ctx context.Context, userID int) (*sync.SyncStatus, error) {
	status := &sync.SyncStatus{
		UserID:       userID,
		LastSyncTime: time.Time{},
		TotalRecords: 0,
		DeviceCount:  0,
		StorageUsed:  0,
		StorageLimit: 100 * 1024 * 1024, // 100 MB
		SyncVersion:  0,
	}

	if err := r.UpdateSyncStatus(ctx, status); err != nil {
		return nil, fmt.Errorf("failed to create initial sync status: %w", err)
	}

	return status, nil
}

func (r *SyncRepository) createInitialSyncStats(ctx context.Context, userID int) (*sync.SyncStats, error) {
	stats := &sync.SyncStats{
		UserID:          userID,
		TotalSyncs:      0,
		LastSync:        time.Time{},
		TotalUploads:    0,
		TotalDownloads:  0,
		TotalConflicts:  0,
		TotalResolved:   0,
		AvgSyncDuration: 0,
		UpdatedAt:       time.Now(),
	}

	query := `
		INSERT INTO sync_stats 
		(user_id, total_syncs, last_sync, total_uploads, total_downloads, 
		 total_conflicts, total_resolved, avg_sync_duration, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
	`

	_, err := r.db.ExecContext(ctx, query,
		stats.UserID,
		stats.TotalSyncs,
		stats.LastSync,
		stats.TotalUploads,
		stats.TotalDownloads,
		stats.TotalConflicts,
		stats.TotalResolved,
		stats.AvgSyncDuration,
		stats.UpdatedAt,
	)

	if err != nil {
		return nil, fmt.Errorf("failed to create initial sync stats: %w", err)
	}

	return stats, nil
}
