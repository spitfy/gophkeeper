package sync

import (
	"time"
)

// SyncStatus представляет статус синхронизации пользователя
type SyncStatus struct {
	UserID       int       `json:"user_id"`
	LastSyncTime time.Time `json:"last_sync_time"`
	TotalRecords int       `json:"total_records"`
	DeviceCount  int       `json:"device_count"`
	StorageUsed  int64     `json:"storage_used"`
	StorageLimit int64     `json:"storage_limit"`
	SyncVersion  int64     `json:"sync_version"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

// RecordSync запись для синхронизации
type RecordSync struct {
	ID        int               `json:"id"`
	UserID    int               `json:"user_id"`
	Type      string            `json:"type"`
	Metadata  map[string]string `json:"metadata"`
	Data      []byte            `json:"data"`
	Version   int               `json:"version"`
	Deleted   bool              `json:"deleted"`
	CreatedAt time.Time         `json:"created_at"`
	UpdatedAt time.Time         `json:"updated_at"`
}

// DeviceInfo информация об устройстве
type DeviceInfo struct {
	ID           int       `json:"id"`
	UserID       int       `json:"user_id"`
	Name         string    `json:"name"`
	Type         string    `json:"type"` // mobile, desktop, web
	LastSyncTime time.Time `json:"last_sync_time"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
	IPAddress    string    `json:"ip_address,omitempty"`
	UserAgent    string    `json:"user_agent,omitempty"`
}

// Conflict конфликт синхронизации
type Conflict struct {
	ID           int       `json:"id"`
	RecordID     int       `json:"record_id"`
	UserID       int       `json:"user_id"`
	DeviceID     int       `json:"device_id"`
	LocalData    []byte    `json:"local_data"`
	ServerData   []byte    `json:"server_data"`
	ConflictType string    `json:"conflict_type"`
	Resolved     bool      `json:"resolved"`
	Resolution   string    `json:"resolution"`
	ResolvedAt   time.Time `json:"resolved_at"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

// SyncStats статистика синхронизации
type SyncStats struct {
	UserID          int       `json:"user_id"`
	TotalSyncs      int       `json:"total_syncs"`
	LastSync        time.Time `json:"last_sync"`
	TotalUploads    int64     `json:"total_uploads"`
	TotalDownloads  int64     `json:"total_downloads"`
	TotalConflicts  int       `json:"total_conflicts"`
	TotalResolved   int       `json:"total_resolved"`
	AvgSyncDuration float64   `json:"avg_sync_duration"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
}

// ServiceConfig конфигурация сервиса синхронизации
type ServiceConfig struct {
	BatchSize          int           `json:"batch_size"`
	MaxSyncRecords     int           `json:"max_sync_records"`
	ConflictTTL        time.Duration `json:"conflict_ttl"`
	DeviceSyncInterval time.Duration `json:"device_sync_interval"`
	StorageLimit       int64         `json:"storage_limit"`
}
