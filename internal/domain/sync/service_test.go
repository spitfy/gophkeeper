package sync

import (
	"context"
	"errors"
	"testing"
	"time"

	"gophkeeper/internal/app/server/api/http/middleware/auth"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"golang.org/x/exp/slog"
)

// MockRepository is a mock implementation of the Repository interface for testing
type MockRepository struct {
	mock.Mock
}

func (m *MockRepository) GetSyncStatus(ctx context.Context, userID int) (*SyncStatus, error) {
	args := m.Called(ctx, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*SyncStatus), args.Error(1)
}

func (m *MockRepository) UpdateSyncStatus(ctx context.Context, status *SyncStatus) error {
	args := m.Called(ctx, status)
	return args.Error(0)
}

func (m *MockRepository) GetDeviceInfo(ctx context.Context, deviceID int) (*DeviceInfo, error) {
	args := m.Called(ctx, deviceID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*DeviceInfo), args.Error(1)
}

func (m *MockRepository) RegisterDevice(ctx context.Context, device *DeviceInfo) error {
	args := m.Called(ctx, device)
	return args.Error(0)
}

func (m *MockRepository) UpdateDeviceSyncTime(ctx context.Context, deviceID int, syncTime time.Time) error {
	args := m.Called(ctx, deviceID, syncTime)
	return args.Error(0)
}

func (m *MockRepository) ListUserDevices(ctx context.Context, userID int) ([]*DeviceInfo, error) {
	args := m.Called(ctx, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*DeviceInfo), args.Error(1)
}

func (m *MockRepository) DeleteDevice(ctx context.Context, deviceID int) error {
	args := m.Called(ctx, deviceID)
	return args.Error(0)
}

func (m *MockRepository) GetRecordsForSync(ctx context.Context, userID int, lastSyncTime time.Time, limit, offset int) ([]*RecordSync, error) {
	args := m.Called(ctx, userID, lastSyncTime, limit, offset)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*RecordSync), args.Error(1)
}

func (m *MockRepository) GetRecordByID(ctx context.Context, recordID int) (*RecordSync, error) {
	args := m.Called(ctx, recordID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*RecordSync), args.Error(1)
}

func (m *MockRepository) GetRecordVersions(ctx context.Context, recordID int, limit int) ([]*RecordSync, error) {
	args := m.Called(ctx, recordID, limit)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*RecordSync), args.Error(1)
}

func (m *MockRepository) GetSyncConflicts(ctx context.Context, userID int) ([]*Conflict, error) {
	args := m.Called(ctx, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*Conflict), args.Error(1)
}

func (m *MockRepository) GetConflictByID(ctx context.Context, conflictID int) (*Conflict, error) {
	args := m.Called(ctx, conflictID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*Conflict), args.Error(1)
}

func (m *MockRepository) ResolveConflict(ctx context.Context, conflictID int, resolution string, resolvedData []byte) error {
	args := m.Called(ctx, conflictID, resolution, resolvedData)
	return args.Error(0)
}

func (m *MockRepository) SaveRecord(ctx context.Context, record *RecordSync) error {
	args := m.Called(ctx, record)
	return args.Error(0)
}

func (m *MockRepository) SaveConflict(ctx context.Context, conflict *Conflict) error {
	args := m.Called(ctx, conflict)
	return args.Error(0)
}

func (m *MockRepository) BatchUpsertRecords(ctx context.Context, records []*RecordSync) (int, []int, error) {
	args := m.Called(ctx, records)
	return args.Int(0), args.Get(1).([]int), args.Error(2)
}

func (m *MockRepository) BatchDeleteRecords(ctx context.Context, recordIDs []int, userID int) error {
	args := m.Called(ctx, recordIDs, userID)
	return args.Error(0)
}

func (m *MockRepository) GetSyncStats(ctx context.Context, userID int) (*SyncStats, error) {
	args := m.Called(ctx, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*SyncStats), args.Error(1)
}

func (m *MockRepository) IncrementSyncStats(ctx context.Context, userID int, uploads, downloads int64) error {
	args := m.Called(ctx, userID, uploads, downloads)
	return args.Error(0)
}

func (m *MockRepository) RecordSyncDuration(ctx context.Context, userID int, duration time.Duration) error {
	args := m.Called(ctx, userID, duration)
	return args.Error(0)
}

// createContextWithUserID creates a context with userID set for testing
func createContextWithUserID(userID int) context.Context {
	return context.WithValue(context.Background(), auth.UserIDKey, userID)
}

func TestService_GetChanges(t *testing.T) {
	mockRepo := new(MockRepository)
	logger := slog.Default()
	config := &ServiceConfig{
		BatchSize:      100,
		MaxSyncRecords: 1000,
		StorageLimit:   100 * 1024 * 1024,
	}
	service := NewService(mockRepo, logger, config)

	userID := 123
	req := GetChangesRequest{
		LastSyncTime: time.Now().Add(-1 * time.Hour),
		Limit:        50,
		Offset:       0,
	}

	// Mock repository calls
	records := []*RecordSync{
		{
			ID:            1,
			UserID:        userID,
			Type:          "login",
			EncryptedData: "encrypted_data_1",
			Version:       1,
			LastModified:  time.Now(),
		},
		{
			ID:            2,
			UserID:        userID,
			Type:          "text",
			EncryptedData: "encrypted_data_2",
			Version:       1,
			LastModified:  time.Now(),
		},
	}

	status := &SyncStatus{
		UserID:       userID,
		StorageUsed:  1000,
		StorageLimit: config.StorageLimit,
	}

	stats := &SyncStats{
		TotalSyncs:      10,
		LastSync:        time.Now(),
		TotalConflicts:  2,
		TotalResolved:   1,
		AvgSyncDuration: 0.5,
	}

	mockRepo.On("GetRecordsForSync", mock.Anything, userID, req.LastSyncTime, req.Limit, req.Offset).Return(records, nil)
	mockRepo.On("GetSyncStatus", mock.Anything, userID).Return(status, nil)
	mockRepo.On("UpdateSyncStatus", mock.Anything, mock.MatchedBy(func(s *SyncStatus) bool {
		return s.UserID == userID && s.SyncVersion > 0
	})).Return(nil)
	mockRepo.On("GetSyncStats", mock.Anything, userID).Return(stats, nil)

	ctx := createContextWithUserID(userID)
	response, err := service.GetChanges(ctx, req)
	assert.NoError(t, err)
	assert.NotNil(t, response)
	assert.Equal(t, "Ok", response.Status)
	assert.Len(t, response.Records, 2)
	assert.Equal(t, records[0].ID, response.Records[0].ID)
	assert.Equal(t, records[1].ID, response.Records[1].ID)
	assert.False(t, response.HasMore) // 2 records < limit of 50, so no more
	assert.NotNil(t, response.Stats)
	assert.Equal(t, 10, response.Stats.TotalSyncs)

	mockRepo.AssertExpectations(t)
}

func TestService_GetChanges_NotAuthenticated(t *testing.T) {
	mockRepo := new(MockRepository)
	logger := slog.Default()
	config := &ServiceConfig{}
	service := NewService(mockRepo, logger, config)

	req := GetChangesRequest{}
	ctx := context.Background() // No userID in context

	_, err := service.GetChanges(ctx, req)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "user not authenticated")
}

func TestService_GetChanges_RepositoryError(t *testing.T) {
	mockRepo := new(MockRepository)
	logger := slog.Default()
	config := &ServiceConfig{}
	service := NewService(mockRepo, logger, config)

	userID := 123
	req := GetChangesRequest{}
	ctx := createContextWithUserID(userID)

	mockRepo.On("GetRecordsForSync", mock.Anything, userID, mock.AnythingOfType("time.Time"), mock.AnythingOfType("int"), mock.AnythingOfType("int")).Return([]*RecordSync{}, errors.New("database error"))

	_, err := service.GetChanges(ctx, req)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "database error")

	mockRepo.AssertExpectations(t)
}

func TestService_ProcessBatch(t *testing.T) {
	mockRepo := new(MockRepository)
	logger := slog.Default()
	config := &ServiceConfig{
		StorageLimit: 100 * 1024 * 1024,
	}
	service := NewService(mockRepo, logger, config)

	userID := 123
	records := []RecordSync{
		{
			ID:            1,
			Type:          "login",
			EncryptedData: "encrypted_data_1",
			Version:       1,
			LastModified:  time.Now(),
		},
		{
			ID:            2,
			Type:          "text",
			EncryptedData: "encrypted_data_2",
			Version:       1,
			LastModified:  time.Now(),
		},
	}

	req := BatchSyncRequest{
		Records: records,
	}

	status := &SyncStatus{
		UserID:       userID,
		StorageUsed:  0,
		StorageLimit: config.StorageLimit,
	}

	mockRepo.On("GetSyncStatus", mock.Anything, userID).Return(status, nil)
	mockRepo.On("GetRecordByID", mock.Anything, 1).Return((*RecordSync)(nil), ErrRecordNotFound)
	mockRepo.On("GetRecordByID", mock.Anything, 2).Return((*RecordSync)(nil), ErrRecordNotFound)
	mockRepo.On("SaveRecord", mock.Anything, mock.MatchedBy(func(r *RecordSync) bool {
		return r.UserID == userID && r.ID == 1
	})).Return(nil)
	mockRepo.On("SaveRecord", mock.Anything, mock.MatchedBy(func(r *RecordSync) bool {
		return r.UserID == userID && r.ID == 2
	})).Return(nil)
	mockRepo.On("UpdateSyncStatus", mock.Anything, mock.MatchedBy(func(s *SyncStatus) bool {
		return s.UserID == userID && s.StorageUsed > 0
	})).Return(nil)
	mockRepo.On("IncrementSyncStats", mock.Anything, userID, int64(2), int64(0)).Return(nil)

	ctx := createContextWithUserID(userID)
	response, err := service.ProcessBatch(ctx, req)
	assert.NoError(t, err)
	assert.NotNil(t, response)
	assert.Equal(t, "Ok", response.Status)
	assert.Equal(t, 2, response.Processed)
	assert.Equal(t, 0, response.Failed)
	assert.Empty(t, response.Errors)

	mockRepo.AssertExpectations(t)
}

func TestService_ProcessBatch_StorageLimitExceeded(t *testing.T) {
	mockRepo := new(MockRepository)
	logger := slog.Default()
	config := &ServiceConfig{
		StorageLimit: 100, // Very small limit for testing
	}
	service := NewService(mockRepo, logger, config)

	userID := 123
	records := []RecordSync{
		{
			ID:            1,
			Type:          "login",
			EncryptedData: "very_long_encrypted_data_that_exceeds_storage_limit",
			Version:       1,
			LastModified:  time.Now(),
		},
	}

	req := BatchSyncRequest{
		Records: records,
	}

	status := &SyncStatus{
		UserID:       userID,
		StorageUsed:  50,
		StorageLimit: config.StorageLimit,
	}

	mockRepo.On("GetSyncStatus", mock.Anything, userID).Return(status, nil)

	ctx := createContextWithUserID(userID)
	_, err := service.ProcessBatch(ctx, req)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "storage limit exceeded")

	mockRepo.AssertExpectations(t)
}

func TestService_GetStatus(t *testing.T) {
	mockRepo := new(MockRepository)
	logger := slog.Default()
	config := &ServiceConfig{}
	service := NewService(mockRepo, logger, config)

	userID := 123
	status := &SyncStatus{
		UserID:       userID,
		LastSyncTime: time.Now(),
		TotalRecords: 10,
		StorageUsed:  1000,
	}

	mockRepo.On("GetSyncStatus", mock.Anything, userID).Return(status, nil)

	ctx := createContextWithUserID(userID)
	response, err := service.GetStatus(ctx)
	assert.NoError(t, err)
	assert.NotNil(t, response)
	assert.Equal(t, "Ok", response.Status)
	assert.Equal(t, status, response.Data)

	mockRepo.AssertExpectations(t)
}

func TestService_GetStatus_NotAuthenticated(t *testing.T) {
	mockRepo := new(MockRepository)
	logger := slog.Default()
	config := &ServiceConfig{}
	service := NewService(mockRepo, logger, config)

	ctx := context.Background()
	_, err := service.GetStatus(ctx)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "user not authenticated")
}

func TestService_GetConflicts(t *testing.T) {
	mockRepo := new(MockRepository)
	logger := slog.Default()
	config := &ServiceConfig{}
	service := NewService(mockRepo, logger, config)

	userID := 123
	conflicts := []*Conflict{
		{
			ID:           1,
			RecordID:     100,
			UserID:       userID,
			ConflictType: "version_mismatch",
			CreatedAt:    time.Now(),
		},
	}

	mockRepo.On("GetSyncConflicts", mock.Anything, userID).Return(conflicts, nil)

	ctx := createContextWithUserID(userID)
	response, err := service.GetConflicts(ctx)
	assert.NoError(t, err)
	assert.NotNil(t, response)
	assert.Equal(t, "Ok", response.Status)
	assert.Len(t, response.Data, 1)
	assert.Equal(t, conflicts[0].ID, response.Data[0].ID)

	mockRepo.AssertExpectations(t)
}

func TestService_ResolveConflict(t *testing.T) {
	mockRepo := new(MockRepository)
	logger := slog.Default()
	config := &ServiceConfig{}
	service := NewService(mockRepo, logger, config)

	userID := 123
	conflictID := 1
	conflict := &Conflict{
		ID:       conflictID,
		UserID:   userID,
		RecordID: 100,
	}

	req := ResolveConflictRequest{
		Resolution: "client",
		ResolvedData: &RecordSync{
			ID:            100,
			EncryptedData: "resolved_encrypted_data",
		},
	}

	mockRepo.On("GetConflictByID", mock.Anything, conflictID).Return(conflict, nil)
	mockRepo.On("ResolveConflict", mock.Anything, conflictID, "client", mock.AnythingOfType("[]uint8")).Return(nil)

	ctx := createContextWithUserID(userID)
	response, err := service.ResolveConflict(ctx, conflictID, req)
	assert.NoError(t, err)
	assert.NotNil(t, response)
	assert.Equal(t, "Ok", response.Status)
	assert.Equal(t, "Conflict resolved successfully", response.Message)

	mockRepo.AssertExpectations(t)
}

func TestService_ResolveConflict_WrongUser(t *testing.T) {
	mockRepo := new(MockRepository)
	logger := slog.Default()
	config := &ServiceConfig{}
	service := NewService(mockRepo, logger, config)

	userID := 123
	conflictID := 1
	conflict := &Conflict{
		ID:       conflictID,
		UserID:   456, // Different user
		RecordID: 100,
	}

	req := ResolveConflictRequest{
		Resolution: "client",
	}

	mockRepo.On("GetConflictByID", mock.Anything, conflictID).Return(conflict, nil)

	ctx := createContextWithUserID(userID)
	_, err := service.ResolveConflict(ctx, conflictID, req)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "conflict does not belong to user")

	mockRepo.AssertExpectations(t)
}

func TestService_GetDevices(t *testing.T) {
	mockRepo := new(MockRepository)
	logger := slog.Default()
	config := &ServiceConfig{}
	service := NewService(mockRepo, logger, config)

	userID := 123
	devices := []*DeviceInfo{
		{
			ID:           1,
			UserID:       userID,
			Name:         "Test Device",
			Type:         "desktop",
			LastSyncTime: time.Now(),
		},
	}

	mockRepo.On("ListUserDevices", mock.Anything, userID).Return(devices, nil)

	ctx := createContextWithUserID(userID)
	result, err := service.GetDevices(ctx)
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Len(t, result, 1)
	assert.Equal(t, devices[0].ID, result[0].ID)

	mockRepo.AssertExpectations(t)
}

func TestService_RemoveDevice(t *testing.T) {
	mockRepo := new(MockRepository)
	logger := slog.Default()
	config := &ServiceConfig{}
	service := NewService(mockRepo, logger, config)

	userID := 123
	deviceID := 1
	device := &DeviceInfo{
		ID:     deviceID,
		UserID: userID,
		Name:   "Test Device",
	}

	mockRepo.On("GetDeviceInfo", mock.Anything, deviceID).Return(device, nil)
	mockRepo.On("DeleteDevice", mock.Anything, deviceID).Return(nil)

	ctx := createContextWithUserID(userID)
	response, err := service.RemoveDevice(ctx, deviceID)
	assert.NoError(t, err)
	assert.NotNil(t, response)
	assert.Equal(t, "Ok", response.Status)
	assert.Equal(t, "Device removed successfully", response.Message)

	mockRepo.AssertExpectations(t)
}

func TestService_RemoveDevice_WrongUser(t *testing.T) {
	mockRepo := new(MockRepository)
	logger := slog.Default()
	config := &ServiceConfig{}
	service := NewService(mockRepo, logger, config)

	userID := 123
	deviceID := 1
	device := &DeviceInfo{
		ID:     deviceID,
		UserID: 456, // Different user
		Name:   "Test Device",
	}

	mockRepo.On("GetDeviceInfo", mock.Anything, deviceID).Return(device, nil)

	ctx := createContextWithUserID(userID)
	_, err := service.RemoveDevice(ctx, deviceID)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "device does not belong to user")

	mockRepo.AssertExpectations(t)
}

// Test edge cases for GetChanges with different limit values
func TestService_GetChanges_LimitValidation(t *testing.T) {
	mockRepo := new(MockRepository)
	logger := slog.Default()
	config := &ServiceConfig{
		BatchSize:      50,
		MaxSyncRecords: 200,
	}
	service := NewService(mockRepo, logger, config)

	userID := 123
	records := []*RecordSync{
		{
			ID:            1,
			UserID:        userID,
			Type:          "login",
			EncryptedData: "encrypted_data_1",
			Version:       1,
			LastModified:  time.Now(),
		},
	}

	status := &SyncStatus{
		UserID: userID,
	}

	stats := &SyncStats{}

	tests := []struct {
		name            string
		req             GetChangesRequest
		expectedLimit   int
		expectedHasMore bool
	}{
		{
			name: "Zero limit should use default batch size",
			req: GetChangesRequest{
				LastSyncTime: time.Now().Add(-1 * time.Hour),
				Limit:        0,
				Offset:       0,
			},
			expectedLimit:   50,
			expectedHasMore: false,
		},
		{
			name: "Limit above max should be capped",
			req: GetChangesRequest{
				LastSyncTime: time.Now().Add(-1 * time.Hour),
				Limit:        500,
				Offset:       0,
			},
			expectedLimit:   200,
			expectedHasMore: false,
		},
		{
			name: "Valid limit should be used as-is",
			req: GetChangesRequest{
				LastSyncTime: time.Now().Add(-1 * time.Hour),
				Limit:        25,
				Offset:       0,
			},
			expectedLimit:   25,
			expectedHasMore: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockRepo.On("GetRecordsForSync", mock.Anything, userID, tt.req.LastSyncTime, tt.expectedLimit, tt.req.Offset).Return(records, nil)
			mockRepo.On("GetSyncStatus", mock.Anything, userID).Return(status, nil)
			mockRepo.On("UpdateSyncStatus", mock.Anything, mock.AnythingOfType("*sync.SyncStatus")).Return(nil)
			mockRepo.On("GetSyncStats", mock.Anything, userID).Return(stats, nil)

			ctx := createContextWithUserID(userID)
			response, err := service.GetChanges(ctx, tt.req)
			assert.NoError(t, err)
			assert.NotNil(t, response)
			assert.Equal(t, tt.expectedHasMore, response.HasMore)

			mockRepo.AssertExpectations(t)
			mockRepo.ExpectedCalls = nil // Clear expectations for next test
		})
	}
}
