package sync

import (
	"context"
	"time"
)

// Repository интерфейс для работы с синхронизацией
type Repository interface {
	// SyncMetadata методы
	GetSyncStatus(ctx context.Context, userID int) (*SyncStatus, error)
	UpdateSyncStatus(ctx context.Context, status *SyncStatus) error
	GetDeviceInfo(ctx context.Context, deviceID string) (*DeviceInfo, error)
	RegisterDevice(ctx context.Context, device *DeviceInfo) error
	UpdateDeviceSyncTime(ctx context.Context, deviceID string, syncTime time.Time) error
	ListUserDevices(ctx context.Context, userID int) ([]*DeviceInfo, error)

	// Sync methods
	GetRecordsForSync(ctx context.Context, userID int, lastSyncTime time.Time, limit, offset int) ([]*RecordSync, error)
	GetRecordByID(ctx context.Context, recordID string) (*RecordSync, error)
	GetRecordVersions(ctx context.Context, recordID string, limit int) ([]*RecordSync, error)
	GetSyncConflicts(ctx context.Context, userID int) ([]*Conflict, error)
	GetConflictByID(ctx context.Context, conflictID string) (*Conflict, error)
	ResolveConflict(ctx context.Context, conflictID, resolution string, resolvedData []byte) error

	// Batch operations
	SaveRecord(ctx context.Context, record *RecordSync) error
	SaveConflict(ctx context.Context, conflict *Conflict) error
	BatchUpsertRecords(ctx context.Context, records []*RecordSync) (int, []string, error)
	BatchDeleteRecords(ctx context.Context, recordIDs []string, userID int) error
	DeleteDevice(ctx context.Context, deviceID string) error

	// Statistics
	GetSyncStats(ctx context.Context, userID int) (*SyncStats, error)
	IncrementSyncStats(ctx context.Context, userID int, uploads, downloads int64) error
	RecordSyncDuration(ctx context.Context, userID int, duration time.Duration) error
}
