package sync

import (
	"time"
)

// Request/Response структуры для GetChanges
type getChangesInput struct {
	Body GetChangesRequest
}

type getChangesOutput struct {
	Body GetChangesResponse
}

type GetChangesRequest struct {
	LastSyncTime time.Time `json:"last_sync_time" example:"2024-01-01T12:00:00Z" format:"date-time"`
	Limit        int       `json:"limit" minimum:"1" maximum:"1000" default:"100"`
	Offset       int       `json:"offset" minimum:"0" default:"0"`
}

type GetChangesResponse struct {
	Status      string          `json:"status"`
	Error       string          `json:"error,omitempty"`
	Records     []SyncRecord    `json:"records,omitempty"`
	HasMore     bool            `json:"has_more,omitempty"`
	ServerTime  time.Time       `json:"server_time,omitempty"`
	SyncVersion int64           `json:"sync_version,omitempty"`
	Stats       *SyncStatsBrief `json:"stats,omitempty"`
}

// Request/Response для BatchSync
type batchSyncInput struct {
	Body BatchSyncRequest
}

type batchSyncOutput struct {
	Body BatchSyncResponse
}

type BatchSyncRequest struct {
	Records []SyncRecord `json:"records"`
}

type BatchSyncResponse struct {
	Status    string   `json:"status"`
	Error     string   `json:"error,omitempty"`
	Processed int      `json:"processed,omitempty"`
	Failed    int      `json:"failed,omitempty"`
	Errors    []string `json:"errors,omitempty"`
}

// Request/Response для GetStatus
type getStatusInput struct {
}

type getStatusOutput struct {
	Body GetStatusResponse
}

type GetStatusResponse struct {
	Status string      `json:"status"`
	Error  string      `json:"error,omitempty"`
	Data   *SyncStatus `json:"data,omitempty"`
}

// Request/Response для GetConflicts
type getConflictsInput struct {
}

type getConflictsOutput struct {
	Body GetConflictsResponse
}

type GetConflictsResponse struct {
	Status string         `json:"status"`
	Error  string         `json:"error,omitempty"`
	Data   []ConflictInfo `json:"data,omitempty"`
}

// Request/Response для ResolveConflict
type resolveConflictInput struct {
	ID   string `path:"id"`
	Body ResolveConflictRequest
}

type resolveConflictOutput struct {
	Body ResolveConflictResponse
}

type ResolveConflictRequest struct {
	Resolution   string      `json:"resolution" enum:"client,server,merged"`
	ResolvedData *SyncRecord `json:"resolved_data,omitempty"`
}

type ResolveConflictResponse struct {
	Status  string `json:"status"`
	Error   string `json:"error,omitempty"`
	Message string `json:"message,omitempty"`
}

// Request/Response для GetDevices
type getDevicesInput struct {
}

type getDevicesOutput struct {
	Body GetDevicesResponse
}

type GetDevicesResponse struct {
	Status string       `json:"status"`
	Error  string       `json:"error,omitempty"`
	Data   []DeviceInfo `json:"data,omitempty"`
}

// Request/Response для RemoveDevice
type removeDeviceInput struct {
	ID string `path:"id"`
}

type removeDeviceOutput struct {
	Body RemoveDeviceResponse
}

type RemoveDeviceResponse struct {
	Status  string `json:"status"`
	Error   string `json:"error,omitempty"`
	Message string `json:"message,omitempty"`
}

// Общие структуры данных
type SyncRecord struct {
	ID        string            `json:"id"`
	Type      string            `json:"type" enum:"password,note,card,file"`
	Metadata  map[string]string `json:"metadata"`
	Data      []byte            `json:"data"`
	Version   int               `json:"version"`
	Deleted   bool              `json:"deleted,omitempty"`
	CreatedAt time.Time         `json:"created_at"`
	UpdatedAt time.Time         `json:"updated_at"`
}

type SyncStatus struct {
	LastSyncTime time.Time `json:"last_sync_time"`
	TotalRecords int       `json:"total_records"`
	DeviceCount  int       `json:"device_count"`
	StorageUsed  int64     `json:"storage_used"`
	StorageLimit int64     `json:"storage_limit"`
	SyncVersion  int64     `json:"sync_version"`
	UserID       string    `json:"user_id"`
}

type ConflictInfo struct {
	ID           string      `json:"id"`
	RecordID     string      `json:"record_id"`
	ConflictType string      `json:"conflict_type" enum:"edit-edit,delete-edit,edit-delete"`
	LocalRecord  *SyncRecord `json:"local_record,omitempty"`
	ServerRecord *SyncRecord `json:"server_record,omitempty"`
	CreatedAt    time.Time   `json:"created_at"`
}

type DeviceInfo struct {
	ID           string    `json:"id"`
	Name         string    `json:"name"`
	Type         string    `json:"type" enum:"desktop,mobile,web"`
	LastSyncTime time.Time `json:"last_sync_time"`
	CreatedAt    time.Time `json:"created_at"`
	IPAddress    string    `json:"ip_address,omitempty"`
}

type SyncStatsBrief struct {
	TotalSyncs      int     `json:"total_syncs"`
	LastSuccessful  string  `json:"last_successful,omitempty"`
	AvgSyncDuration float64 `json:"avg_sync_duration"`
	TotalConflicts  int     `json:"total_conflicts"`
	TotalResolved   int     `json:"total_resolved"`
}
