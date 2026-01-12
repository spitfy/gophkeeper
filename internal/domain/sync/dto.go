package sync

import (
	"time"
)

// DTO (Data Transfer Objects) для API синхронизации

// GetChangesRequest запрос на получение изменений
type GetChangesRequest struct {
	LastSyncTime time.Time `json:"last_sync_time" example:"2024-01-01T12:00:00Z" format:"date-time"`
	Limit        int       `json:"limit" minimum:"1" maximum:"1000" default:"100"`
	Offset       int       `json:"offset" minimum:"0" default:"0"`
}

// GetChangesResponse ответ с изменениями
type GetChangesResponse struct {
	Status      string          `json:"status"`
	Error       string          `json:"error,omitempty"`
	Records     []RecordSync    `json:"records,omitempty"`
	HasMore     bool            `json:"has_more,omitempty"`
	ServerTime  time.Time       `json:"server_time,omitempty"`
	SyncVersion int64           `json:"sync_version,omitempty"`
	Stats       *SyncStatsBrief `json:"stats,omitempty"`
}

// BatchSyncRequest запрос на пакетную синхронизацию
type BatchSyncRequest struct {
	Records []RecordSync `json:"records"`
}

// BatchSyncResponse ответ на пакетную синхронизацию
type BatchSyncResponse struct {
	Status    string   `json:"status"`
	Error     string   `json:"error,omitempty"`
	Processed int      `json:"processed,omitempty"`
	Failed    int      `json:"failed,omitempty"`
	Errors    []string `json:"errors,omitempty"`
}

// GetStatusResponse ответ со статусом синхронизации
type GetStatusResponse struct {
	Status string      `json:"status"`
	Error  string      `json:"error,omitempty"`
	Data   *SyncStatus `json:"data,omitempty"`
}

// GetConflictsResponse ответ с конфликтами
type GetConflictsResponse struct {
	Status string     `json:"status"`
	Error  string     `json:"error,omitempty"`
	Data   []Conflict `json:"data,omitempty"`
}

// ResolveConflictRequest запрос на разрешение конфликта
type ResolveConflictRequest struct {
	Resolution   string      `json:"resolution" enum:"client,server,merged"`
	ResolvedData *RecordSync `json:"resolved_data,omitempty"`
}

// ResolveConflictResponse ответ на разрешение конфликта
type ResolveConflictResponse struct {
	Status  string `json:"status"`
	Error   string `json:"error,omitempty"`
	Message string `json:"message,omitempty"`
}

// GetDevicesResponse ответ со списком устройств
type GetDevicesResponse struct {
	Status string       `json:"status"`
	Error  string       `json:"error,omitempty"`
	Data   []DeviceInfo `json:"data,omitempty"`
}

// RemoveDeviceResponse ответ на удаление устройства
type RemoveDeviceResponse struct {
	Status  string `json:"status"`
	Error   string `json:"error,omitempty"`
	Message string `json:"message,omitempty"`
}

// SyncStatsBrief краткая статистика синхронизации
type SyncStatsBrief struct {
	TotalSyncs      int     `json:"total_syncs"`
	LastSuccessful  string  `json:"last_successful,omitempty"`
	AvgSyncDuration float64 `json:"avg_sync_duration"`
	TotalConflicts  int     `json:"total_conflicts"`
	TotalResolved   int     `json:"total_resolved"`
}

// SyncRecordDTO упрощенная версия RecordSync для API
// Note: Вместо создания отдельной структуры, можно использовать RecordSync
// но если нужны отличия, создаем DTO
type SyncRecordDTO struct {
	ID        int               `json:"id"`
	Type      string            `json:"type" enum:"password,note,card,file"`
	Metadata  map[string]string `json:"metadata"`
	Data      []byte            `json:"data"`
	Version   int               `json:"version"`
	Deleted   bool              `json:"deleted,omitempty"`
	CreatedAt time.Time         `json:"created_at"`
	UpdatedAt time.Time         `json:"updated_at"`
}
