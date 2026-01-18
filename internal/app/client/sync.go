// internal/app/client/sync.go
package client

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	gosync "sync"
	"time"

	"golang.org/x/exp/slog"

	"gophkeeper/internal/domain/record"
	"gophkeeper/internal/domain/sync"
)

// SyncService управляет синхронизацией данных между клиентом и сервером
type SyncService struct {
	app        *App
	log        *slog.Logger
	config     *SyncConfig
	mu         gosync.RWMutex
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
	RecordID  int       `json:"record_id"`
	Error     string    `json:"error"`
	Operation string    `json:"operation"`
	Timestamp time.Time `json:"timestamp"`
	Retry     int       `json:"retry"`
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

// SyncMetadata метаданные для синхронизации
type SyncMetadata struct {
	ClientID      string    `json:"client_id"`
	LastSyncTime  time.Time `json:"last_sync_time"`
	SyncVersion   int64     `json:"sync_version"`
	DeviceName    string    `json:"device_name"`
	ClientVersion string    `json:"client_version"`
}

// SyncStats статистика синхронизации (локальная версия)
type SyncStats struct {
	TotalSyncs      int       `json:"total_syncs"`
	LastSync        time.Time `json:"last_sync"`
	TotalUploads    int       `json:"total_uploads"`
	TotalDownloads  int       `json:"total_downloads"`
	TotalConflicts  int       `json:"total_conflicts"`
	TotalResolved   int       `json:"total_resolved"`
	TotalErrors     int       `json:"total_errors"`
	AvgSyncDuration float64   `json:"avg_sync_duration"`
}

// LocalConflict конфликт синхронизации (локальная версия с расширенными полями)
type LocalConflict struct {
	sync.Conflict
	LocalRecord  *LocalRecord `json:"local_record,omitempty"`
	ServerRecord *LocalRecord `json:"server_record,omitempty"`
	MergedRecord *LocalRecord `json:"merged_record,omitempty"`
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

	s.log.Info("Начало синхронизации", "start_time", result.StartTime)

	// 1. Получаем метаданные синхронизации
	syncMeta, err := s.getSyncMetadata(ctx)
	if err != nil {
		s.log.Error("Ошибка получения метаданных синхронизации", "error", err)
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
		s.log.Error("Ошибка получения локальных изменений", "error", err)
		result.Errors = append(result.Errors, SyncError{
			Error:     err.Error(),
			Operation: "get_local_changes",
			Timestamp: time.Now(),
		})
	}

	// 3. Получаем изменения с сервера
	serverChanges, err := s.getServerChanges(ctx, syncMeta)
	if err != nil {
		s.log.Error("Ошибка получения изменений с сервера", "error", err)
		result.Errors = append(result.Errors, SyncError{
			Error:     err.Error(),
			Operation: "get_server_changes",
			Timestamp: time.Now(),
		})
	}

	// 4. Обнаруживаем и разрешаем конфликты
	conflicts, err := s.detectConflicts(localChanges, serverChanges)
	if err != nil {
		s.log.Error("Ошибка обнаружения конфликтов", "error", err)
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
		s.log.Error("Ошибка разрешения конфликтов", "error", err)
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
		s.log.Error("Ошибка обновления метаданных синхронизации", "error", err)
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
func (s *SyncService) getLocalChanges(ctx context.Context, meta *SyncMetadata) ([]*LocalRecord, error) {
	// Получаем записи, которые не синхронизированы или изменились после последней синхронизации
	records, err := s.app.storage.GetRecordsModifiedAfter(meta.LastSyncTime, s.config.BatchSize)
	if err != nil {
		return nil, fmt.Errorf("ошибка запроса локальных изменений: %w", err)
	}

	s.log.Debug("Найдены локальные изменения", "count", len(records))
	return records, nil
}

// getServerChanges получает изменения с сервера
func (s *SyncService) getServerChanges(ctx context.Context, meta *SyncMetadata) ([]*LocalRecord, error) {
	// Используем HTTP клиент для получения изменений с сервера
	req := sync.GetChangesRequest{
		LastSyncTime: meta.LastSyncTime,
		Limit:        s.config.BatchSize,
	}

	response, err := s.app.httpClient.GetSyncChanges(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("ошибка получения изменений с сервера: %w", err)
	}

	// Конвертируем серверные записи в локальные
	var records []*LocalRecord
	for _, syncRec := range response.Records {
		localRec := &LocalRecord{
			ServerID:     syncRec.ID,
			UserID:       syncRec.UserID,
			Type:         record.RecType(syncRec.Type),
			Version:      syncRec.Version,
			LastModified: syncRec.UpdatedAt,
			CreatedAt:    syncRec.CreatedAt,
			Synced:       true,
		}
		if syncRec.Deleted {
			now := time.Now()
			localRec.DeletedAt = &now
		}
		// Конвертируем metadata map в JSON
		if syncRec.Metadata != nil {
			metaJSON, _ := json.Marshal(syncRec.Metadata)
			localRec.Meta = metaJSON
		}
		localRec.EncryptedData = string(syncRec.Data)
		records = append(records, localRec)
	}

	s.log.Debug("Получены изменения с сервера", "count", len(records))
	return records, nil
}

// detectConflicts обнаруживает конфликты между локальными и серверными изменениями
func (s *SyncService) detectConflicts(localChanges, serverChanges []*LocalRecord) ([]*LocalConflict, error) {
	var conflicts []*LocalConflict

	// Создаем карту для быстрого поиска записей по ServerID
	localMap := make(map[int]*LocalRecord)
	for _, rec := range localChanges {
		if rec.ServerID > 0 {
			localMap[rec.ServerID] = rec
		}
	}

	serverMap := make(map[int]*LocalRecord)
	for _, rec := range serverChanges {
		serverMap[rec.ServerID] = rec
	}

	// Проверяем каждую запись на конфликты
	checkedIDs := make(map[int]bool)

	// Проверяем локальные изменения
	for _, localRec := range localChanges {
		if localRec.ServerID == 0 {
			continue // Новая локальная запись, не может быть конфликта
		}
		if checkedIDs[localRec.ServerID] {
			continue
		}
		checkedIDs[localRec.ServerID] = true

		if serverRec, exists := serverMap[localRec.ServerID]; exists {
			// Обе стороны изменили запись
			conflict, err := s.checkRecordConflict(localRec, serverRec)
			if err != nil {
				return nil, fmt.Errorf("ошибка проверки конфликта записи %d: %w", localRec.ID, err)
			}

			if conflict != nil {
				conflicts = append(conflicts, conflict)
			}
		}
	}

	// Проверяем серверные изменения, которые не были в локальных
	for _, serverRec := range serverChanges {
		if checkedIDs[serverRec.ServerID] {
			continue
		}

		// Получаем локальную версию записи (если есть)
		localRec, err := s.app.storage.GetRecordByServerID(serverRec.ServerID)
		if err == nil && localRec != nil {
			// Запись существует локально, проверяем конфликт
			conflict, err := s.checkRecordConflict(localRec, serverRec)
			if err != nil {
				return nil, fmt.Errorf("ошибка проверки конфликта записи %d: %w", serverRec.ServerID, err)
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
func (s *SyncService) checkRecordConflict(localRec, serverRec *LocalRecord) (*LocalConflict, error) {
	// Если локальная запись синхронизирована, конфликта нет
	if localRec.Synced {
		return nil, nil
	}

	// Определяем тип конфликта
	conflictType := "edit-edit"

	localDeleted := localRec.DeletedAt != nil
	serverDeleted := serverRec.DeletedAt != nil

	if localDeleted && !serverDeleted {
		conflictType = "delete-edit"
	} else if !localDeleted && serverDeleted {
		conflictType = "edit-delete"
	} else if localDeleted && serverDeleted {
		// Обе стороны удалили запись - это не конфликт
		return nil, nil
	}

	// Если версии совпадают, конфликта нет
	if localRec.Version == serverRec.Version &&
		localRec.LastModified.Equal(serverRec.LastModified) {
		return nil, nil
	}

	return &LocalConflict{
		Conflict: sync.Conflict{
			RecordID:     localRec.ServerID,
			ConflictType: conflictType,
			CreatedAt:    time.Now(),
			Resolved:     false,
		},
		LocalRecord:  localRec,
		ServerRecord: serverRec,
	}, nil
}

// resolveConflicts разрешает конфликты
func (s *SyncService) resolveConflicts(ctx context.Context, conflicts []*LocalConflict) ([]*LocalConflict, error) {
	var resolvedConflicts []*LocalConflict

	for _, conflict := range conflicts {
		var resolvedConflict *LocalConflict
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
func (s *SyncService) autoResolveConflict(conflict *LocalConflict) (*LocalConflict, error) {
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
		if conflict.LocalRecord.LastModified.After(conflict.ServerRecord.LastModified) {
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
func (s *SyncService) applyConflictResolution(conflict *LocalConflict) error {
	if conflict.MergedRecord == nil {
		return fmt.Errorf("отсутствует объединенная запись")
	}

	// Обновляем запись в локальном хранилище
	conflict.MergedRecord.Synced = false // Помечаем для повторной синхронизации
	conflict.MergedRecord.LastModified = time.Now()

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
func (s *SyncService) uploadChanges(ctx context.Context, changes []*LocalRecord) (int, []SyncError) {
	var errors []SyncError
	uploaded := 0

	// Конвертируем локальные записи в формат для batch sync
	var syncRecords []sync.RecordSync
	for _, rec := range changes {
		syncRec := sync.RecordSync{
			ID:        rec.ServerID,
			UserID:    rec.UserID,
			Type:      string(rec.Type),
			Version:   rec.Version,
			Deleted:   rec.DeletedAt != nil,
			CreatedAt: rec.CreatedAt,
			UpdatedAt: rec.LastModified,
			Data:      []byte(rec.EncryptedData),
		}
		// Конвертируем Meta в map
		if rec.Meta != nil {
			var metaMap map[string]string
			if err := json.Unmarshal(rec.Meta, &metaMap); err == nil {
				syncRec.Metadata = metaMap
			}
		}
		syncRecords = append(syncRecords, syncRec)
	}

	// Отправляем batch запрос
	req := sync.BatchSyncRequest{
		Records: syncRecords,
	}

	response, err := s.app.httpClient.SendBatchSync(ctx, req)
	if err != nil {
		errors = append(errors, SyncError{
			Error:     err.Error(),
			Operation: "batch_upload",
			Timestamp: time.Now(),
		})
		return 0, errors
	}

	uploaded = response.Processed

	// Помечаем записи как синхронизированные
	for _, rec := range changes {
		rec.Synced = true
		if err := s.app.storage.UpdateRecord(rec); err != nil {
			s.log.Warn("Ошибка обновления статуса синхронизации",
				"record_id", rec.ID,
				"error", err)
		}
	}

	s.log.Debug("Загружено записей на сервер", "count", uploaded, "errors", len(errors))
	return uploaded, errors
}

// applyServerChanges применяет изменения с сервера
func (s *SyncService) applyServerChanges(ctx context.Context, changes []*LocalRecord) (int, []SyncError) {
	var errors []SyncError
	downloaded := 0

	for _, serverRec := range changes {
		// Получаем локальную версию записи
		localRec, err := s.app.storage.GetRecordByServerID(serverRec.ServerID)

		if err != nil {
			// Запись не существует локально, создаем новую
			serverRec.Synced = true
			if err := s.app.storage.SaveRecord(serverRec); err != nil {
				errors = append(errors, SyncError{
					RecordID:  serverRec.ServerID,
					Error:     err.Error(),
					Operation: "download_create",
					Timestamp: time.Now(),
				})
				continue
			}
		} else {
			// Запись существует, обновляем
			// Проверяем, не было ли локальных изменений
			if !localRec.Synced && localRec.LastModified.After(serverRec.LastModified) {
				// Есть локальные изменения, пропускаем это обновление
				// Конфликт уже должен был быть обработан
				continue
			}

			// Обновляем локальную запись
			serverRec.ID = localRec.ID // Сохраняем локальный ID
			serverRec.Synced = true
			if err := s.app.storage.UpdateRecord(serverRec); err != nil {
				errors = append(errors, SyncError{
					RecordID:  serverRec.ServerID,
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
		SyncVersion:   int64(s.stats.TotalSyncs + 1),
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
	s.stats.LastSync = time.Now()
	s.stats.TotalUploads += result.Uploaded
	s.stats.TotalDownloads += result.Downloaded
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
		s.log.Error("Ошибка сериализации статистики", "error", err)
		return
	}

	if err := os.WriteFile(statsPath, data, 0600); err != nil {
		s.log.Error("Ошибка записи статистики", "error", err)
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
				s.log.Error("Ошибка автоматической синхронизации", "error", err)
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
