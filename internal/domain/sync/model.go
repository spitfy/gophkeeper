// internal/domain/sync/model.go
package sync

import (
	"time"
)

// SyncRequest запрос на синхронизацию
type SyncRequest struct {
	LastSyncTime time.Time `json:"last_sync_time"`
	SyncVersion  int64     `json:"sync_version"`
	DeviceID     string    `json:"device_id"`
	Limit        int       `json:"limit"`
}

// SyncResponse ответ на синхронизацию
type SyncResponse struct {
	Records       []RecordSync `json:"records"`
	HasMore       bool         `json:"has_more"`
	NextSyncToken string       `json:"next_sync_token"`
	ServerTime    time.Time    `json:"server_time"`
}

// RecordSync запись для синхронизации
type RecordSync struct {
	ID        string            `json:"id"`
	Type      string            `json:"type"`
	Metadata  map[string]string `json:"metadata"`
	Data      []byte            `json:"data"`
	Version   int               `json:"version"`
	Deleted   bool              `json:"deleted"`
	UpdatedAt time.Time         `json:"updated_at"`
	CreatedAt time.Time         `json:"created_at"`
}

// SyncBatchRequest пакетный запрос на синхронизацию
type SyncBatchRequest struct {
	Records []RecordSync `json:"records"`
}

// SyncBatchResponse пакетный ответ на синхронизацию
type SyncBatchResponse struct {
	Processed int      `json:"processed"`
	Failed    int      `json:"failed"`
	Errors    []string `json:"errors,omitempty"`
}

// DeviceInfo информация об устройстве
type DeviceInfo struct {
	ID           string    `json:"id"`
	Name         string    `json:"name"`
	Type         string    `json:"type"`
	LastSyncTime time.Time `json:"last_sync_time"`
	CreatedAt    time.Time `json:"created_at"`
	IsCurrent    bool      `json:"is_current"`
}
