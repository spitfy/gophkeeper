// internal/app/client/sync.go
package client

import (
	"context"
	"encoding/json"
	"fmt"
	"golang.org/x/exp/slog"
	"os"
	"sync"
	"time"

	"gophkeeper/internal/domain/record"
)

// SyncService управляет синхронизацией данных между клиентом и сервером
type SyncService struct {
	app        *App
	log        *slog.Logger
	config     *SyncConfig
	mu         sync.RWMutex
	lastSync   time.Time
	isSyncing  bool
	syncErrors []SyncError
	stats      *SyncStats
}

// SyncConfig конфигурация синхронизации
type SyncConfig struct {
	Enabled          bool          `json:"enabled"`
	Interval         time.Duration `json:"interval"`
	BatchSize        int           `json:"batch_size"`
	MaxRetries       int           `json:"max_retries"`
	RetryDelay       time.Duration `json:"retry_delay"`
	ConflictStrategy string        `json:"conflict_strategy"` // client, server, newer, manual
	AutoResolve      bool          `json:"auto_resolve"`      // автоматически разрешать конфликты
}

// SyncError ошибка синхронизации
type SyncError struct {
	RecordID  string    `json:"record_id"`
	Error     string    `json:"error"`
	Operation string    `json:"operation"`
	Timestamp time.Time `json:"timestamp"`
	Retry     int       `json:"retry"`
}

// SyncStat статистика синхронизации
type SyncStats struct {
	TotalSyncs      int       `json:"total_syncs"`
	LastSuccessful  time.Time `json:"last_successful"`
	LastFailed      time.Time `json:"last_failed"`
	TotalUploaded   int       `json:"total_uploaded"`
	TotalDownloaded int       `json:"total_downloaded"`
	TotalConflicts  int       `json:"total_conflicts"`
	TotalResolved   int       `json:"total_resolved"`
	TotalErrors     int       `json:"total_errors"`
	AvgSyncDuration float64   `json:"avg_sync_duration"`
}

// SyncResult результат синхронизации
type SyncResult struct {
	Success    bool          `json:"success"`
	Uploaded   int           `json:"uploaded"`
	Downloaded int           `json:"downloaded"`
	Conflicts  int           `json:"conflicts"`
	Resolved   int           `json:"resolved"`
	Errors     []SyncError   `json:"errors"`
	Duration   time.Duration `json:"duration"`
	StartTime  time.Time     `json:"start_time"`
	EndTime    time.Time     `json:"end_time"`
}

// Conflict конфликт синхронизации
type Conflict struct {
	RecordID     string    `json:"record_id"`
	LocalRecord  *Record   `json:"local_record"`
	ServerRecord *Record   `json:"server_record"`
	ConflictType string    `json:"conflict_type"` // edit-edit, delete-edit, etc.
	CreatedAt    time.Time `json:"created_at"`
	Resolved     bool      `json:"resolved"`
	Resolution   string    `json:"resolution"` // local, server, merged, manual
	MergedRecord *Record   `json:"merged_record"`
}

// SyncMetadata метаданные для синхронизации
type SyncMetadata struct {
	ClientID      string    `json:"client_id"`
	LastSyncTime  time.Time `json:"last_sync_time"`
	SyncVersion   int64     `json:"sync_version"`
	DeviceName    string    `json:"device_name"`
	ClientVersion string    `json:"client_version"`
}

// NewSyncService создает новый сервис синхронизации
func NewSyncService(app *App) *SyncService {
	defaultConfig := &SyncConfig{
		Enabled:          true,
		Interval:         30 * time.Second,
		BatchSize:        50,
		MaxRetries:       3,
		RetryDelay:       5 * time.Second,
		ConflictStrategy: "newer", // по умолчанию выбираем новую версию
		AutoResolve:      true,
	}

	// Загружаем конфигурацию из файла если есть
	if config, err := loadSyncConfig(app.config.ConfigDir); err == nil {
		// Объединяем с дефолтными значениями
		mergeConfigs(defaultConfig, config)
	}

	return &SyncService{
		app:    app,
		log:    app.log,
		config: defaultConfig,
		stats:  &SyncStats{},
	}
}

// Sync запускает процесс синхронизации
func (s *SyncService) Sync(ctx context.Context) (*SyncResult, error) {
	s.mu.Lock()
	if s.isSyncing {
		s.mu.Unlock()
		return nil, fmt.Errorf("синхронизация уже выполняется")
	}

	s.isSyncing = true
	s.mu.Unlock()

	defer func() {
		s.mu.Lock()
		s.isSyncing = false
		s.mu.Unlock()
	}()

	result := &SyncResult{
		StartTime: time.Now(),
		Errors:    []SyncError{},
	}

	// Проверяем условия для синхронизации
	if err := s.preSyncChecks(ctx); err != nil {
		result.Errors = append(result.Errors, SyncError{
			Error:     err.Error(),
			Operation: "pre_sync_check",
			Timestamp: time.Now(),
		})
		result.Success = false
		result.EndTime = time.Now()
		result.Duration = result.EndTime.Sub(result.StartTime)
		return result, err
	}

	// Выполняем синхронизацию
	s.log.Info("Начало синхронизации", "start_time", result.StartTime)

	// 1. Получаем метаданные синхронизации
	syncMeta, err := s.getSyncMetadata(ctx)
	if err != nil {
		s.log.Error("Ошибка получения метаданных синхронизации", err)
		result.Errors = append(result.Errors, SyncError{
			Error:     err.Error(),
			Operation: "get_metadata",
			Timestamp: time.Now(),
		})
		result.Success = false
		result.EndTime = time.Now()
		result.Duration = result.EndTime.Sub(result.StartTime)
		return result, err
	}

	// 2. Получаем локальные изменения
	localChanges, err := s.getLocalChanges(ctx, syncMeta)
	if err != nil {
		s.log.Error("Ошибка получения локальных изменений", err)
		result.Errors = append(result.Errors, SyncError{
			Error:     err.Error(),
			Operation: "get_local_changes",
			Timestamp: time.Now(),
		})
	}

	// 3. Получаем изменения с сервера
	serverChanges, err := s.getServerChanges(ctx, syncMeta)
	if err != nil {
		s.log.Error("Ошибка получения изменений с сервера", err)
		result.Errors = append(result.Errors, SyncError{
			Error:     err.Error(),
			Operation: "get_server_changes",
			Timestamp: time.Now(),
		})
	}

	// 4. Обнаруживаем и разрешаем конфликты
	conflicts, err := s.detectConflicts(localChanges, serverChanges)
	if err != nil {
		s.log.Error("Ошибка обнаружения конфликтов", err)
		result.Errors = append(result.Errors, SyncError{
			Error:     err.Error(),
			Operation: "detect_conflicts",
			Timestamp: time.Now(),
		})
	}
	result.Conflicts = len(conflicts)

	// 5. Разрешаем конфликты
	resolvedConflicts, err := s.resolveConflicts(ctx, conflicts)
	if err != nil {
		s.log.Error("Ошибка разрешения конфликтов", err)
		result.Errors = append(result.Errors, SyncError{
			Error:     err.Error(),
			Operation: "resolve_conflicts",
			Timestamp: time.Now(),
		})
	}
	result.Resolved = len(resolvedConflicts)

	// 6. Отправляем изменения на сервер
	if len(localChanges) > 0 {
		uploaded, uploadErrors := s.uploadChanges(ctx, localChanges)
		result.Uploaded = uploaded
		result.Errors = append(result.Errors, uploadErrors...)
	}

	// 7. Применяем изменения с сервера
	if len(serverChanges) > 0 {
		downloaded, downloadErrors := s.applyServerChanges(ctx, serverChanges)
		result.Downloaded = downloaded
		result.Errors = append(result.Errors, downloadErrors...)
	}

	// 8. Обновляем метаданные синхронизации
	if err := s.updateSyncMetadata(ctx); err != nil {
		s.log.Error("Ошибка обновления метаданных синхронизации", err)
		result.Errors = append(result.Errors, SyncError{
			Error:     err.Error(),
			Operation: "update_metadata",
			Timestamp: time.Now(),
		})
	}

	// 9. Обновляем статистику
	s.updateStats(result)

	result.EndTime = time.Now()
	result.Duration = result.EndTime.Sub(result.StartTime)
	result.Success = len(result.Errors) == 0

	if result.Success {
		s.log.Info("Синхронизация успешно завершена",
			"duration", result.Duration,
			"uploaded", result.Uploaded,
			"downloaded", result.Downloaded,
			"conflicts", result.Conflicts,
		)
	} else {
		s.log.Warn("Синхронизация завершена с ошибками",
			"duration", result.Duration,
			"errors", len(result.Errors),
		)
	}

	return result, nil
}

// preSyncChecks проверяет условия для синхронизации
func (s *SyncService) preSyncChecks(ctx context.Context) error {
	// 1. Проверяем, включена ли синхронизация
	if !s.config.Enabled {
		return fmt.Errorf("синхронизация отключена")
	}

	// 2. Проверяем аутентификацию
	if !s.app.IsAuthenticated() {
		return fmt.Errorf("пользователь не аутентифицирован")
	}

	// 3. Проверяем соединение с сервером
	if err := s.app.CheckConnection(); err != nil {
		return fmt.Errorf("сервер недоступен: %w", err)
	}

	// 4. Проверяем мастер-ключ
	if s.app.crypto.IsLocked() {
		return fmt.Errorf("мастер-ключ заблокирован")
	}

	// 5. Проверяем, не слишком ли часто пытаемся синхронизироваться
	if time.Since(s.lastSync) < 5*time.Second {
		return fmt.Errorf("синхронизация выполняется слишком часто")
	}

	return nil
}

// getSyncMetadata получает метаданные синхронизации
func (s *SyncService) getSyncMetadata(ctx context.Context) (*SyncMetadata, error) {
	meta := &SyncMetadata{
		ClientID:      s.app.config.ConfigDir, // Используем директорию конфига как ID клиента
		ClientVersion: "1.0.0",
		DeviceName:    getDeviceName(),
	}

	// Загружаем сохраненные метаданные
	if data, err := s.loadSyncMetadata(); err == nil {
		meta.LastSyncTime = data.LastSyncTime
		meta.SyncVersion = data.SyncVersion
	}

	return meta, nil
}

// getLocalChanges получает локальные изменения
func (s *SyncService) getLocalChanges(ctx context.Context, meta *SyncMetadata) ([]*Record, error) {
	// Получаем записи, которые не синхронизированы или изменились после последней синхронизации
	query := `
		SELECT id, type, metadata, data, created_at, updated_at, version, synced, deleted, sync_version
		FROM records 
		WHERE synced = 0 OR updated_at > ? OR sync_version < ?
		ORDER BY updated_at ASC
		LIMIT ?
	`

	rows, err := s.app.storage.(*SQLiteStorage).db.Query(query,
		meta.LastSyncTime.Format(time.RFC3339),
		meta.SyncVersion,
		s.config.BatchSize)
	if err != nil {
		return nil, fmt.Errorf("ошибка запроса локальных изменений: %w", err)
	}
	defer rows.Close()

	var changes []*Record
	for rows.Next() {
		var rec Record
		var metadataJSON string
		var createdAt, updatedAt string

		if err := rows.Scan(&rec.ID, &rec.Type, &metadataJSON, &rec.Data,
			&createdAt, &updatedAt, &rec.Version, &rec.Synced,
			&rec.Deleted, &rec.SyncVersion); err != nil {
			return nil, fmt.Errorf("ошибка сканирования записи: %w", err)
		}

		// Парсим метаданные
		if err := json.Unmarshal([]byte(metadataJSON), &rec.Metadata); err != nil {
			return nil, fmt.Errorf("ошибка парсинга метаданных: %w", err)
		}

		// Парсим временные метки
		rec.CreatedAt, _ = time.Parse(time.RFC3339, createdAt)
		rec.UpdatedAt, _ = time.Parse(time.RFC3339, updatedAt)

		changes = append(changes, &rec)
	}

	s.log.Debug("Найдены локальные изменения", "count", len(changes))
	return changes, nil
}

// getServerChanges получает изменения с сервера
func (s *SyncService) getServerChanges(ctx context.Context, meta *SyncMetadata) ([]*Record, error) {
	// Используем HTTP клиент для получения изменений с сервера
	req := record.SyncRequest{
		LastSyncTime: meta.LastSyncTime,
		SyncVersion:  meta.SyncVersion,
		DeviceID:     meta.ClientID,
	}

	serverRecords, err := s.app.httpClient.GetSyncChanges(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("ошибка получения изменений с сервера: %w", err)
	}

	s.log.Debug("Получены изменения с сервера", "count", len(serverRecords))
	return serverRecords, nil
}

// detectConflicts обнаруживает конфликты между локальными и серверными изменениями
func (s *SyncService) detectConflicts(localChanges, serverChanges []*Record) ([]*Conflict, error) {
	var conflicts []*Conflict

	// Создаем карту для быстрого поиска записей по ID
	localMap := make(map[string]*Record)
	for _, rec := range localChanges {
		localMap[rec.ID] = rec
	}

	serverMap := make(map[string]*Record)
	for _, rec := range serverChanges {
		serverMap[rec.ID] = rec
	}

	// Проверяем каждую запись на конфликты
	checkedIDs := make(map[string]bool)

	// Проверяем локальные изменения
	for _, localRec := range localChanges {
		if checkedIDs[localRec.ID] {
			continue
		}
		checkedIDs[localRec.ID] = true

		if serverRec, exists := serverMap[localRec.ID]; exists {
			// Обе стороны изменили запись
			conflict, err := s.checkRecordConflict(localRec, serverRec)
			if err != nil {
				return nil, fmt.Errorf("ошибка проверки конфликта записи %s: %w", localRec.ID, err)
			}

			if conflict != nil {
				conflicts = append(conflicts, conflict)
			}
		}
	}

	// Проверяем серверные изменения, которые не были в локальных
	for _, serverRec := range serverChanges {
		if checkedIDs[serverRec.ID] {
			continue
		}

		// Получаем локальную версию записи (если есть)
		localRec, err := s.app.storage.GetRecord(serverRec.ID)
		if err == nil && localRec != nil {
			// Запись существует локально, проверяем конфликт
			conflict, err := s.checkRecordConflict(localRec, serverRec)
			if err != nil {
				return nil, fmt.Errorf("ошибка проверки конфликта записи %s: %w", serverRec.ID, err)
			}

			if conflict != nil {
				conflicts = append(conflicts, conflict)
			}
		}
	}

	s.log.Debug("Обнаружены конфликты", "count", len(conflicts))
	return conflicts, nil
}

// checkRecordConflict проверяет конфликт между двумя версиями записи
func (s *SyncService) checkRecordConflict(localRec, serverRec *Record) (*Conflict, error) {
	// Если обе версии синхронизированы, конфликта нет
	if localRec.Synced && serverRec.Synced {
		return nil, nil
	}

	// Определяем тип конфликта
	conflictType := "edit-edit"

	if localRec.Deleted && !serverRec.Deleted {
		conflictType = "delete-edit"
	} else if !localRec.Deleted && serverRec.Deleted {
		conflictType = "edit-delete"
	} else if localRec.Deleted && serverRec.Deleted {
		// Обе стороны удалили запись - это не конфликт
		return nil, nil
	}

	// Если версии совпадают, конфликта нет
	if localRec.Version == serverRec.Version &&
		localRec.UpdatedAt.Equal(serverRec.UpdatedAt) &&
		localRec.Deleted == serverRec.Deleted {
		return nil, nil
	}

	return &Conflict{
		RecordID:     localRec.ID,
		LocalRecord:  localRec,
		ServerRecord: serverRec,
		ConflictType: conflictType,
		CreatedAt:    time.Now(),
		Resolved:     false,
	}, nil
}

// resolveConflicts разрешает конфликты
func (s *SyncService) resolveConflicts(ctx context.Context, conflicts []*Conflict) ([]*Conflict, error) {
	var resolvedConflicts []*Conflict

	for _, conflict := range conflicts {
		var resolvedConflict *Conflict
		var err error

		if s.config.AutoResolve {
			resolvedConflict, err = s.autoResolveConflict(conflict)
		} else {
			// TODO: В будущем можно добавить интерактивное разрешение конфликтов
			resolvedConflict, err = s.autoResolveConflict(conflict)
		}

		if err != nil {
			s.log.Error("Ошибка разрешения конфликта",
				"record_id", conflict.RecordID,
				"error", err)
			continue
		}

		if resolvedConflict != nil && resolvedConflict.Resolved {
			// Применяем разрешенную версию
			if err := s.applyConflictResolution(resolvedConflict); err != nil {
				s.log.Error("Ошибка применения разрешения конфликта",
					"record_id", conflict.RecordID,
					"error", err)
			} else {
				resolvedConflicts = append(resolvedConflicts, resolvedConflict)
			}
		}
	}

	s.log.Debug("Разрешены конфликты", "count", len(resolvedConflicts))
	return resolvedConflicts, nil
}

// autoResolveConflict автоматически разрешает конфликт
func (s *SyncService) autoResolveConflict(conflict *Conflict) (*Conflict, error) {
	// Копируем конфликт
	resolved := *conflict
	resolved.Resolved = true

	switch s.config.ConflictStrategy {
	case "client":
		// Всегда выбираем локальную версию
		resolved.Resolution = "client"
		resolved.MergedRecord = conflict.LocalRecord

	case "server":
		// Всегда выбираем серверную версию
		resolved.Resolution = "server"
		resolved.MergedRecord = conflict.ServerRecord

	case "newer":
		// Выбираем более новую версию
		if conflict.LocalRecord.UpdatedAt.After(conflict.ServerRecord.UpdatedAt) {
			resolved.Resolution = "client"
			resolved.MergedRecord = conflict.LocalRecord
		} else {
			resolved.Resolution = "server"
			resolved.MergedRecord = conflict.ServerRecord
		}

	case "manual":
		// Требуется ручное разрешение
		resolved.Resolved = false
		return &resolved, nil

	default:
		return nil, fmt.Errorf("неизвестная стратегия разрешения конфликтов: %s", s.config.ConflictStrategy)
	}

	return &resolved, nil
}

// applyConflictResolution применяет разрешенную версию конфликта
func (s *SyncService) applyConflictResolution(conflict *Conflict) error {
	if conflict.MergedRecord == nil {
		return fmt.Errorf("отсутствует объединенная запись")
	}

	// Обновляем запись в локальном хранилище
	conflict.MergedRecord.Synced = false // Помечаем для повторной синхронизации
	conflict.MergedRecord.UpdatedAt = time.Now()

	if conflict.Resolution == "server" {
		// Если выбрана серверная версия, увеличиваем версию
		conflict.MergedRecord.Version = conflict.ServerRecord.Version + 1
	} else {
		// Если выбрана локальная версия, увеличиваем версию
		conflict.MergedRecord.Version = conflict.LocalRecord.Version + 1
	}

	return s.app.storage.UpdateRecord(conflict.MergedRecord)
}

// uploadChanges отправляет локальные изменения на сервер
func (s *SyncService) uploadChanges(ctx context.Context, changes []*Record) (int, []SyncError) {
	var errors []SyncError
	uploaded := 0

	for _, rec := range changes {
		for retry := 0; retry <= s.config.MaxRetries; retry++ {
			if retry > 0 {
				s.log.Debug("Повторная попытка отправки записи",
					"record_id", rec.ID,
					"retry", retry)
				time.Sleep(s.config.RetryDelay)
			}

			var err error
			if rec.Deleted {
				// Отправляем удаление
				err = s.app.httpClient.DeleteRecord(ctx, rec.ID)
			} else if rec.Version == 1 && !rec.Synced {
				// Новая запись
				req := record.CreateRequest{
					Type:     rec.Type,
					Metadata: rec.Metadata,
					Data:     rec.Data,
				}
				_, err = s.app.httpClient.CreateRecord(ctx, req)
			} else {
				// Обновление существующей записи
				req := record.UpdateRequest{
					Metadata: &rec.Metadata,
					Data:     rec.Data,
					Version:  rec.Version,
				}
				err = s.app.httpClient.UpdateRecord(ctx, rec.ID, req)
			}

			if err != nil {
				if retry == s.config.MaxRetries {
					errors = append(errors, SyncError{
						RecordID:  rec.ID,
						Error:     err.Error(),
						Operation: "upload",
						Timestamp: time.Now(),
						Retry:     retry + 1,
					})
					s.log.Error("Ошибка отправки записи после всех попыток",
						"record_id", rec.ID,
						"error", err)
				}
				continue
			}

			// Помечаем запись как синхронизированную
			rec.Synced = true
			if err := s.app.storage.UpdateRecord(rec); err != nil {
				s.log.Warn("Ошибка обновления статуса синхронизации",
					"record_id", rec.ID,
					"error", err)
			}

			uploaded++
			break
		}
	}

	s.log.Debug("Загружено записей на сервер", "count", uploaded, "errors", len(errors))
	return uploaded, errors
}

// applyServerChanges применяет изменения с сервера
func (s *SyncService) applyServerChanges(ctx context.Context, changes []*Record) (int, []SyncError) {
	var errors []SyncError
	downloaded := 0

	for _, serverRec := range changes {
		// Получаем локальную версию записи
		localRec, err := s.app.storage.GetRecord(serverRec.ID)

		if err != nil {
			// Запись не существует локально, создаем новую
			serverRec.Synced = true
			if err := s.app.storage.SaveRecord(serverRec); err != nil {
				errors = append(errors, SyncError{
					RecordID:  serverRec.ID,
					Error:     err.Error(),
					Operation: "download_create",
					Timestamp: time.Now(),
				})
				continue
			}
		} else {
			// Запись существует, обновляем
			// Проверяем, не было ли локальных изменений
			if !localRec.Synced && localRec.UpdatedAt.After(serverRec.UpdatedAt) {
				// Есть локальные изменения, пропускаем это обновление
				// Конфликт уже должен был быть обработан
				continue
			}

			// Обновляем локальную запись
			serverRec.Synced = true
			if err := s.app.storage.UpdateRecord(serverRec); err != nil {
				errors = append(errors, SyncError{
					RecordID:  serverRec.ID,
					Error:     err.Error(),
					Operation: "download_update",
					Timestamp: time.Now(),
				})
				continue
			}
		}

		downloaded++
	}

	s.log.Debug("Загружено записей с сервера", "count", downloaded, "errors", len(errors))
	return downloaded, errors
}

// updateSyncMetadata обновляет метаданные синхронизации
func (s *SyncService) updateSyncMetadata(ctx context.Context) error {
	meta := &SyncMetadata{
		ClientID:      s.app.config.ConfigDir,
		LastSyncTime:  time.Now(),
		SyncVersion:   s.stats.TotalSyncs + 1,
		DeviceName:    getDeviceName(),
		ClientVersion: "1.0.0",
	}

	// Сохраняем метаданные
	if err := s.saveSyncMetadata(meta); err != nil {
		return fmt.Errorf("ошибка сохранения метаданных: %w", err)
	}

	s.lastSync = meta.LastSyncTime
	return nil
}

// updateStats обновляет статистику синхронизации
func (s *SyncService) updateStats(result *SyncResult) {
	s.stats.TotalSyncs++

	if result.Success {
		s.stats.LastSuccessful = time.Now()
	} else {
		s.stats.LastFailed = time.Now()
	}

	s.stats.TotalUploaded += result.Uploaded
	s.stats.TotalDownloaded += result.Downloaded
	s.stats.TotalConflicts += result.Conflicts
	s.stats.TotalResolved += result.Resolved
	s.stats.TotalErrors += len(result.Errors)

	// Обновляем среднюю продолжительность
	if s.stats.AvgSyncDuration == 0 {
		s.stats.AvgSyncDuration = result.Duration.Seconds()
	} else {
		s.stats.AvgSyncDuration = (s.stats.AvgSyncDuration*float64(s.stats.TotalSyncs-1) +
			result.Duration.Seconds()) / float64(s.stats.TotalSyncs)
	}

	// Сохраняем статистику
	s.saveStats()
}

// Вспомогательные методы для работы с файлами

func loadSyncConfig(configDir string) (*SyncConfig, error) {
	configPath := configDir + "/sync_config.json"

	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		return nil, fmt.Errorf("файл конфигурации не найден")
	}

	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("ошибка чтения конфигурации: %w", err)
	}

	var config SyncConfig
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("ошибка парсинга конфигурации: %w", err)
	}

	return &config, nil
}

func (s *SyncService) loadSyncMetadata() (*SyncMetadata, error) {
	metaPath := s.app.config.ConfigDir + "/sync_metadata.json"

	if _, err := os.Stat(metaPath); os.IsNotExist(err) {
		return &SyncMetadata{}, nil
	}

	data, err := os.ReadFile(metaPath)
	if err != nil {
		return nil, fmt.Errorf("ошибка чтения метаданных: %w", err)
	}

	var meta SyncMetadata
	if err := json.Unmarshal(data, &meta); err != nil {
		return nil, fmt.Errorf("ошибка парсинга метаданных: %w", err)
	}

	return &meta, nil
}

func (s *SyncService) saveSyncMetadata(meta *SyncMetadata) error {
	metaPath := s.app.config.ConfigDir + "/sync_metadata.json"

	data, err := json.MarshalIndent(meta, "", "  ")
	if err != nil {
		return fmt.Errorf("ошибка сериализации метаданных: %w", err)
	}

	if err := os.WriteFile(metaPath, data, 0600); err != nil {
		return fmt.Errorf("ошибка записи метаданных: %w", err)
	}

	return nil
}

func (s *SyncService) saveStats() {
	statsPath := s.app.config.ConfigDir + "/sync_stats.json"

	data, err := json.MarshalIndent(s.stats, "", "  ")
	if err != nil {
		s.log.Error("Ошибка сериализации статистики", err)
		return
	}

	if err := os.WriteFile(statsPath, data, 0600); err != nil {
		s.log.Error("Ошибка записи статистики", err)
	}
}

func getDeviceName() string {
	hostname, err := os.Hostname()
	if err != nil {
		return "unknown"
	}
	return hostname
}

func mergeConfigs(defaultConfig, userConfig *SyncConfig) {
	if userConfig.Interval > 0 {
		defaultConfig.Interval = userConfig.Interval
	}
	if userConfig.BatchSize > 0 {
		defaultConfig.BatchSize = userConfig.BatchSize
	}
	if userConfig.MaxRetries > 0 {
		defaultConfig.MaxRetries = userConfig.MaxRetries
	}
	if userConfig.RetryDelay > 0 {
		defaultConfig.RetryDelay = userConfig.RetryDelay
	}
	if userConfig.ConflictStrategy != "" {
		defaultConfig.ConflictStrategy = userConfig.ConflictStrategy
	}
	defaultConfig.AutoResolve = userConfig.AutoResolve
}

// Методы управления синхронизацией

// StartAutoSync запускает автоматическую синхронизацию
func (s *SyncService) StartAutoSync(ctx context.Context) {
	if !s.config.Enabled {
		s.log.Info("Автоматическая синхронизация отключена")
		return
	}

	s.log.Info("Запуск автоматической синхронизации", "interval", s.config.Interval)

	ticker := time.NewTicker(s.config.Interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			s.log.Info("Автоматическая синхронизация остановлена")
			return
		case <-ticker.C:
			if _, err := s.Sync(ctx); err != nil {
				s.log.Error("Ошибка автоматической синхронизации", err)
			}
		}
	}
}

// ForceSync принудительная синхронизация
func (s *SyncService) ForceSync(ctx context.Context) (*SyncResult, error) {
	s.log.Info("Запуск принудительной синхронизации")
	return s.Sync(ctx)
}

// GetStats возвращает статистику синхронизации
func (s *SyncService) GetStats() *SyncStats {
	s.mu.RLock()
	defer s.mu.RUnlock()

	// Возвращаем копию статистики
	statsCopy := *s.stats
	return &statsCopy
}

// GetLastSyncTime возвращает время последней синхронизации
func (s *SyncService) GetLastSyncTime() time.Time {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.lastSync
}

// IsSyncing проверяет, выполняется ли синхронизация
func (s *SyncService) IsSyncing() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.isSyncing
}

// ResetStats сбрасывает статистику синхронизации
func (s *SyncService) ResetStats() {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.stats = &SyncStats{}
	s.saveStats()
}
