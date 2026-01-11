package sync

import (
	"context"
	"fmt"
	"gophkeeper/internal/app/server/api/http/middleware/auth"
	"time"

	"golang.org/x/exp/slog"
)

// Структуры запросов/ответов для сервиса
type GetChangesRequest struct {
	LastSyncTime time.Time `json:"last_sync_time"`
	Limit        int       `json:"limit"`
	Offset       int       `json:"offset"`
}

type GetChangesResponse struct {
	Status      string          `json:"status"`
	Error       string          `json:"error,omitempty"`
	Records     []*RecordSync   `json:"records,omitempty"`
	HasMore     bool            `json:"has_more,omitempty"`
	ServerTime  time.Time       `json:"server_time,omitempty"`
	SyncVersion int64           `json:"sync_version,omitempty"`
	Stats       *SyncStatsBrief `json:"stats,omitempty"`
}

type BatchSyncRequest struct {
	Records []*RecordSync `json:"records"`
}

type BatchSyncResponse struct {
	Status    string   `json:"status"`
	Error     string   `json:"error,omitempty"`
	Processed int      `json:"processed,omitempty"`
	Failed    int      `json:"failed,omitempty"`
	Errors    []string `json:"errors,omitempty"`
}

type GetStatusResponse struct {
	Status string      `json:"status"`
	Error  string      `json:"error,omitempty"`
	Data   *SyncStatus `json:"data,omitempty"`
}

type GetConflictsResponse struct {
	Status string      `json:"status"`
	Error  string      `json:"error,omitempty"`
	Data   []*Conflict `json:"data,omitempty"`
}

type ResolveConflictRequest struct {
	Resolution   string      `json:"resolution"`
	ResolvedData *RecordSync `json:"resolved_data,omitempty"`
}

type ResolveConflictResponse struct {
	Status  string `json:"status"`
	Error   string `json:"error,omitempty"`
	Message string `json:"message,omitempty"`
}

type GetDevicesResponse struct {
	Status string        `json:"status"`
	Error  string        `json:"error,omitempty"`
	Data   []*DeviceInfo `json:"data,omitempty"`
}

type RemoveDeviceResponse struct {
	Status  string `json:"status"`
	Error   string `json:"error,omitempty"`
	Message string `json:"message,omitempty"`
}

type SyncStatsBrief struct {
	TotalSyncs      int     `json:"total_syncs"`
	LastSuccessful  string  `json:"last_successful,omitempty"`
	AvgSyncDuration float64 `json:"avg_sync_duration"`
	TotalConflicts  int     `json:"total_conflicts"`
	TotalResolved   int     `json:"total_resolved"`
}

// Servicer интерфейс сервиса синхронизации
type Servicer interface {
	// GetChanges возвращает изменения после указанного времени
	GetChanges(ctx context.Context, req GetChangesRequest) (*GetChangesResponse, error)

	// ProcessBatch обрабатывает пакет записей для синхронизации
	ProcessBatch(ctx context.Context, req BatchSyncRequest) (*BatchSyncResponse, error)

	// GetStatus возвращает текущий статус синхронизации
	GetStatus(ctx context.Context) (*GetStatusResponse, error)

	// GetConflicts возвращает список неразрешенных конфликтов
	GetConflicts(ctx context.Context) ([]*Conflict, error)

	// ResolveConflict разрешает указанный конфликт
	ResolveConflict(ctx context.Context, conflictID int, req ResolveConflictRequest) (*ResolveConflictResponse, error)

	// GetDevices возвращает список устройств пользователя
	GetDevices(ctx context.Context) ([]*DeviceInfo, error)

	// RemoveDevice удаляет устройство из списка синхронизации
	RemoveDevice(ctx context.Context, deviceID int) (*RemoveDeviceResponse, error)
}

// Service реализация сервиса синхронизации
type Service struct {
	repo   Repository
	log    *slog.Logger
	config *ServiceConfig
}

// NewService создает новый сервис синхронизации
func NewService(repo Repository, log *slog.Logger, config *ServiceConfig) *Service {
	if config == nil {
		config = &ServiceConfig{
			BatchSize:      100,
			MaxSyncRecords: 1000,
			ConflictTTL:    7 * 24 * time.Hour,
			StorageLimit:   100 * 1024 * 1024, // 100 MB
		}
	}

	return &Service{
		repo:   repo,
		log:    log,
		config: config,
	}
}

// GetChanges возвращает изменения после указанного времени
func (s *Service) GetChanges(ctx context.Context, req GetChangesRequest) (*GetChangesResponse, error) {
	// Получаем userID из контекста (устанавливается middleware аутентификации)
	userID, ok := auth.GetUserID(ctx)
	if !ok {
		return nil, fmt.Errorf("user not authenticated")
	}

	// Валидация параметров
	if req.Limit <= 0 {
		req.Limit = s.config.BatchSize
	}
	if req.Limit > s.config.MaxSyncRecords {
		req.Limit = s.config.MaxSyncRecords
	}

	// Получаем записи из репозитория
	records, err := s.repo.GetRecordsForSync(ctx, userID, req.LastSyncTime, req.Limit, req.Offset)
	if err != nil {
		return nil, fmt.Errorf("failed to get records for sync: %w", err)
	}

	// Получаем статус синхронизации
	status, err := s.repo.GetSyncStatus(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get sync status: %w", err)
	}

	// Обновляем время последней синхронизации
	status.LastSyncTime = time.Now()
	status.SyncVersion++
	if err := s.repo.UpdateSyncStatus(ctx, status); err != nil {
		s.log.Warn("Failed to update sync status", "error", err)
	}

	// Проверяем, есть ли еще записи
	hasMore := len(records) >= req.Limit

	// Получаем статистику
	stats, err := s.repo.GetSyncStats(ctx, userID)
	if err != nil {
		s.log.Warn("Failed to get sync stats", "error", err)
	}

	// Формируем ответ
	response := &GetChangesResponse{
		Status:      "Ok",
		Records:     records,
		HasMore:     hasMore,
		ServerTime:  time.Now(),
		SyncVersion: status.SyncVersion,
	}

	// Добавляем статистику, если есть
	if stats != nil {
		response.Stats = &SyncStatsBrief{
			TotalSyncs:      stats.TotalSyncs,
			AvgSyncDuration: stats.AvgSyncDuration,
			TotalConflicts:  stats.TotalConflicts,
			TotalResolved:   stats.TotalResolved,
		}
		if !stats.LastSync.IsZero() {
			response.Stats.LastSuccessful = stats.LastSync.Format(time.RFC3339)
		}
	}

	return response, nil
}

// ProcessBatch обрабатывает пакет записей для синхронизации
func (s *Service) ProcessBatch(ctx context.Context, req BatchSyncRequest) (*BatchSyncResponse, error) {
	userID, ok := auth.GetUserID(ctx)
	if !ok {
		return nil, fmt.Errorf("user not authenticated")
	}

	// Проверяем лимит хранилища
	status, err := s.repo.GetSyncStatus(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get sync status: %w", err)
	}

	// Рассчитываем общий размер добавляемых данных
	var totalSize int64
	for _, rec := range req.Records {
		totalSize += int64(len(rec.Data))
	}

	if status.StorageUsed+totalSize > s.config.StorageLimit {
		return nil, fmt.Errorf("storage limit exceeded")
	}

	// Обрабатываем записи
	processed, failed, errors := s.processBatchRecords(ctx, userID, req.Records)

	// Обновляем статистику хранилища
	status.StorageUsed += totalSize
	if err := s.repo.UpdateSyncStatus(ctx, status); err != nil {
		s.log.Warn("Failed to update storage usage", "error", err)
	}

	// Обновляем статистику синхронизации
	if err := s.repo.IncrementSyncStats(ctx, userID, int64(len(req.Records)), 0); err != nil {
		s.log.Warn("Failed to update sync stats", "error", err)
	}

	return &BatchSyncResponse{
		Status:    "Ok",
		Processed: processed,
		Failed:    failed,
		Errors:    errors,
	}, nil
}

// GetStatus возвращает текущий статус синхронизации
func (s *Service) GetStatus(ctx context.Context) (*GetStatusResponse, error) {
	userID, ok := auth.GetUserID(ctx)
	if !ok {
		return nil, fmt.Errorf("user not authenticated")
	}

	status, err := s.repo.GetSyncStatus(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get sync status: %w", err)
	}

	return &GetStatusResponse{
		Status: "Ok",
		Data:   status,
	}, nil
}

// GetConflicts возвращает список неразрешенных конфликтов
func (s *Service) GetConflicts(ctx context.Context) ([]*Conflict, error) {
	userID, ok := auth.GetUserID(ctx)
	if !ok {
		return nil, fmt.Errorf("user not authenticated")
	}

	conflicts, err := s.repo.GetSyncConflicts(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get conflicts: %w", err)
	}

	return conflicts, nil
}

// ResolveConflict разрешает указанный конфликт
func (s *Service) ResolveConflict(ctx context.Context, conflictID int, req ResolveConflictRequest) (*ResolveConflictResponse, error) {
	userID, ok := auth.GetUserID(ctx)
	if !ok {
		return nil, fmt.Errorf("user not authenticated")
	}

	// Проверяем, что конфликт принадлежит пользователю
	conflict, err := s.repo.GetConflictByID(ctx, conflictID)
	if err != nil {
		return nil, fmt.Errorf("failed to get conflict: %w", err)
	}

	if conflict.UserID != userID {
		return nil, fmt.Errorf("conflict does not belong to user")
	}

	// Разрешаем конфликт
	resolvedData := []byte{}
	if req.ResolvedData != nil {
		resolvedData = req.ResolvedData.Data
	}
	if err := s.repo.ResolveConflict(ctx, conflictID, req.Resolution, resolvedData); err != nil {
		return nil, fmt.Errorf("failed to resolve conflict: %w", err)
	}

	return &ResolveConflictResponse{
		Status:  "Ok",
		Message: "Conflict resolved successfully",
	}, nil
}

// GetDevices возвращает список устройств пользователя
func (s *Service) GetDevices(ctx context.Context) ([]*DeviceInfo, error) {
	userID, ok := auth.GetUserID(ctx)
	if !ok {
		return nil, fmt.Errorf("user not authenticated")
	}

	devices, err := s.repo.ListUserDevices(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get devices: %w", err)
	}

	return devices, nil
}

// RemoveDevice удаляет устройство из списка синхронизации
func (s *Service) RemoveDevice(ctx context.Context, deviceID int) (*RemoveDeviceResponse, error) {
	userID, ok := auth.GetUserID(ctx)
	if !ok {
		return nil, fmt.Errorf("user not authenticated")
	}

	// Проверяем, что устройство принадлежит пользователю
	device, err := s.repo.GetDeviceInfo(ctx, deviceID)
	if err != nil {
		return nil, fmt.Errorf("failed to get device info: %w", err)
	}

	if device.UserID != userID {
		return nil, fmt.Errorf("device does not belong to user")
	}

	// Удаляем устройство
	if err := s.repo.DeleteDevice(ctx, deviceID); err != nil {
		return nil, fmt.Errorf("failed to delete device: %w", err)
	}

	return &RemoveDeviceResponse{
		Status:  "Ok",
		Message: "Device removed successfully",
	}, nil
}

// Вспомогательные методы
func (s *Service) processBatchRecords(ctx context.Context, userID int, records []*RecordSync) (int, int, []string) {
	var processed int
	var errors []string

	for _, rec := range records {
		// Проверяем, что запись принадлежит пользователю
		rec.UserID = userID

		// Проверяем конфликты
		existing, err := s.repo.GetRecordByID(ctx, rec.ID)
		if err == nil && existing != nil {
			// Обнаружен конфликт
			if existing.Version >= rec.Version {
				// Серверная версия новее или равна
				if err := s.handleConflict(ctx, userID, rec, existing); err != nil {
					errors = append(errors, fmt.Sprintf("record %d: conflict handling failed: %v", rec.ID, err))
				}
				continue
			}
		}

		// Сохраняем запись
		if err := s.repo.SaveRecord(ctx, rec); err != nil {
			errors = append(errors, fmt.Sprintf("record %d: %v", rec.ID, err))
			continue
		}

		processed++
	}

	return processed, len(records) - processed, errors
}

func (s *Service) handleConflict(ctx context.Context, userID int, local, server *RecordSync) error {
	// Создаем запись о конфликте
	conflict := &Conflict{
		RecordID:     local.ID,
		UserID:       userID,
		DeviceID:     0, // TODO: нужно получать из контекста устройства
		LocalData:    local.Data,
		ServerData:   server.Data,
		ConflictType: "version_mismatch",
		CreatedAt:    time.Now(),
	}

	// Сохраняем конфликт
	return s.repo.SaveConflict(ctx, conflict)
}
