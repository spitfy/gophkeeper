package sync

import (
	"gophkeeper/internal/domain/sync"
)

// Request/Response структуры для GetChanges
type getChangesInput struct {
	Body sync.GetChangesRequest
}

type getChangesOutput struct {
	Body sync.GetChangesResponse
}

// Request/Response для BatchSync
type batchSyncInput struct {
	Body sync.BatchSyncRequest
}

type batchSyncOutput struct {
	Body sync.BatchSyncResponse
}

// Request/Response для GetStatus
type getStatusInput struct {
}

type getStatusOutput struct {
	Body sync.GetStatusResponse
}

// Request/Response для GetConflicts
type getConflictsInput struct {
}

type getConflictsOutput struct {
	Body sync.GetConflictsResponse
}

// Request/Response для ResolveConflict
type resolveConflictInput struct {
	ID   int `path:"id"`
	Body sync.ResolveConflictRequest
}

type resolveConflictOutput struct {
	Body sync.ResolveConflictResponse
}

// Request/Response для GetDevices
type getDevicesInput struct {
}

type getDevicesOutput struct {
	Body sync.GetDevicesResponse
}

// Request/Response для RemoveDevice
type removeDeviceInput struct {
	ID int `path:"id"`
}

type removeDeviceOutput struct {
	Body sync.RemoveDeviceResponse
}
