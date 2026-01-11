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
	GetDeviceInfo(ctx context.Context, deviceID int) (*DeviceInfo, error)
	RegisterDevice(ctx context.Context, device *DeviceInfo) error
	UpdateDeviceSyncTime(ctx context.Context, deviceID int, syncTime time.Time) error
	ListUserDevices(ctx context.Context, userID int) ([]*DeviceInfo, error)
	DeleteDevice(ctx context.Context, deviceID int) error

	// Sync methods
	GetRecordsForSync(ctx context.Context, userID int, lastSyncTime time.Time, limit, offset int) ([]*RecordSync, error)
	GetRecordByID(ctx context.Context, recordID int) (*RecordSync, error)
	GetRecordVersions(ctx context.Context, recordID int, limit int) ([]*RecordSync, error)
	GetSyncConflicts(ctx context.Context, userID int) ([]*Conflict, error)
	GetConflictByID(ctx context.Context, conflictID int) (*Conflict, error)
	ResolveConflict(ctx context.Context, conflictID int, resolution string, resolvedData []byte) error

	// Batch operations
	SaveRecord(ctx context.Context, record *RecordSync) error
	SaveConflict(ctx context.Context, conflict *Conflict) error
	BatchUpsertRecords(ctx context.Context, records []*RecordSync) (int, []int, error)
	BatchDeleteRecords(ctx context.Context, recordIDs []int, userID int) error

	// Statistics
	GetSyncStats(ctx context.Context, userID int) (*SyncStats, error)
	IncrementSyncStats(ctx context.Context, userID int, uploads, downloads int64) error
	RecordSyncDuration(ctx context.Context, userID int, duration time.Duration) error
}
