package record

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"golang.org/x/exp/slog"
)

// MockRepository is a mock implementation of the Repository interface for testing
type MockRepository struct {
	mock.Mock
}

func (m *MockRepository) Create(ctx context.Context, record *Record) (int, error) {
	args := m.Called(ctx, record)
	return args.Int(0), args.Error(1)
}

func (m *MockRepository) Get(ctx context.Context, userID, recordID int) (*Record, error) {
	args := m.Called(ctx, userID, recordID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*Record), args.Error(1)
}

func (m *MockRepository) Update(ctx context.Context, record *Record) error {
	args := m.Called(ctx, record)
	return args.Error(0)
}

func (m *MockRepository) Delete(ctx context.Context, userID, recordID int) error {
	args := m.Called(ctx, userID, recordID)
	return args.Error(0)
}

func (m *MockRepository) SoftDelete(ctx context.Context, userID, recordID int) error {
	args := m.Called(ctx, userID, recordID)
	return args.Error(0)
}

func (m *MockRepository) List(ctx context.Context, userID int) ([]Record, error) {
	args := m.Called(ctx, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]Record), args.Error(1)
}

func (m *MockRepository) Search(ctx context.Context, userID int, criteria SearchCriteria) ([]Record, error) {
	args := m.Called(ctx, userID, criteria)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]Record), args.Error(1)
}

func (m *MockRepository) GetStats(ctx context.Context, userID int) (map[string]interface{}, error) {
	args := m.Called(ctx, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(map[string]interface{}), args.Error(1)
}

func (m *MockRepository) GetModifiedSince(ctx context.Context, userID int, since time.Time) ([]Record, error) {
	args := m.Called(ctx, userID, since)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]Record), args.Error(1)
}

func (m *MockRepository) GetByType(ctx context.Context, userID int, recordType string) ([]Record, error) {
	args := m.Called(ctx, userID, recordType)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]Record), args.Error(1)
}

func (m *MockRepository) GetVersions(ctx context.Context, recordID int) ([]Version, error) {
	args := m.Called(ctx, recordID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]Version), args.Error(1)
}

func (m *MockRepository) GetByChecksum(ctx context.Context, userID int, checksum string) (*Record, error) {
	args := m.Called(ctx, userID, checksum)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*Record), args.Error(1)
}

func (m *MockRepository) SaveVersion(ctx context.Context, version *Version) error {
	args := m.Called(ctx, version)
	return args.Error(0)
}

func TestService_List(t *testing.T) {
	mockRepo := new(MockRepository)
	factory := NewFactory()
	logger := slog.Default()
	service := NewService(mockRepo, factory, logger)

	records := []Record{
		{
			ID:           1,
			UserID:       1,
			Type:         RecTypeLogin,
			Version:      1,
			LastModified: time.Now(),
		},
		{
			ID:           2,
			UserID:       1,
			Type:         RecTypeText,
			Version:      1,
			LastModified: time.Now(),
		},
	}

	mockRepo.On("List", mock.Anything, 1).Return(records, nil)

	response, err := service.List(context.Background(), 1)
	assert.NoError(t, err)
	assert.Equal(t, 2, response.Total)
	assert.Len(t, response.Records, 2)
	assert.Equal(t, 1, response.Records[0].ID)
	assert.Equal(t, RecTypeLogin, response.Records[0].Type)

	mockRepo.AssertExpectations(t)
}

func TestService_Create(t *testing.T) {
	mockRepo := new(MockRepository)
	factory := NewFactory()
	logger := slog.Default()
	service := NewService(mockRepo, factory, logger)

	encryptedData := "encrypted_data"
	meta := json.RawMessage(`{"title": "test"}`)

	mockRepo.On("Create", mock.Anything, mock.MatchedBy(func(r *Record) bool {
		return r.UserID == 1 &&
			r.Type == RecTypeLogin &&
			r.EncryptedData == encryptedData &&
			string(r.Meta) == string(meta) &&
			r.Version == 1
	})).Return(123, nil)

	recordID, err := service.Create(context.Background(), 1, RecTypeLogin, encryptedData, meta)
	assert.NoError(t, err)
	assert.Equal(t, 123, recordID)

	mockRepo.AssertExpectations(t)
}

func TestService_Create_InvalidData(t *testing.T) {
	mockRepo := new(MockRepository)
	factory := NewFactory()
	logger := slog.Default()
	service := NewService(mockRepo, factory, logger)

	// Test empty type
	_, err := service.Create(context.Background(), 1, "", "data", nil)
	assert.Error(t, err)
	assert.Equal(t, ErrInvalidData, err)

	// Test empty encrypted data
	_, err = service.Create(context.Background(), 1, RecTypeLogin, "", nil)
	assert.Error(t, err)
	assert.Equal(t, ErrInvalidData, err)
}

func TestService_Find(t *testing.T) {
	mockRepo := new(MockRepository)
	factory := NewFactory()
	logger := slog.Default()
	service := NewService(mockRepo, factory, logger)

	record := &Record{
		ID:           1,
		UserID:       1,
		Type:         RecTypeLogin,
		Version:      1,
		LastModified: time.Now(),
	}

	mockRepo.On("Get", mock.Anything, 1, 1).Return(record, nil)

	found, err := service.Find(context.Background(), 1, 1)
	assert.NoError(t, err)
	assert.Equal(t, record, found)

	mockRepo.AssertExpectations(t)
}

func TestService_Find_NotFound(t *testing.T) {
	mockRepo := new(MockRepository)
	factory := NewFactory()
	logger := slog.Default()
	service := NewService(mockRepo, factory, logger)

	mockRepo.On("Get", mock.Anything, 1, 1).Return((*Record)(nil), ErrNotFound)

	_, err := service.Find(context.Background(), 1, 1)
	assert.Error(t, err)
	assert.Equal(t, ErrNotFound, err)

	mockRepo.AssertExpectations(t)
}

func TestService_Find_Deleted(t *testing.T) {
	mockRepo := new(MockRepository)
	factory := NewFactory()
	logger := slog.Default()
	service := NewService(mockRepo, factory, logger)

	deletedAt := time.Now()
	record := &Record{
		ID:           1,
		UserID:       1,
		Type:         RecTypeLogin,
		Version:      1,
		LastModified: time.Now(),
		DeletedAt:    &deletedAt,
	}

	mockRepo.On("Get", mock.Anything, 1, 1).Return(record, nil)

	_, err := service.Find(context.Background(), 1, 1)
	assert.Error(t, err)
	assert.Equal(t, ErrRecordDeleted, err)

	mockRepo.AssertExpectations(t)
}

func TestService_Update(t *testing.T) {
	mockRepo := new(MockRepository)
	factory := NewFactory()
	logger := slog.Default()
	service := NewService(mockRepo, factory, logger)

	// Current record
	currentRecord := &Record{
		ID:           1,
		UserID:       1,
		Type:         RecTypeLogin,
		Version:      1,
		LastModified: time.Now(),
	}

	// Updated data
	encryptedData := "updated_encrypted_data"
	meta := json.RawMessage(`{"title": "updated"}`)

	mockRepo.On("Get", mock.Anything, 1, 1).Return(currentRecord, nil)
	mockRepo.On("Update", mock.Anything, mock.MatchedBy(func(r *Record) bool {
		return r.ID == 1 &&
			r.UserID == 1 &&
			r.Type == RecTypeLogin &&
			r.EncryptedData == encryptedData &&
			string(r.Meta) == string(meta) &&
			r.Version == 1 // Version should not be incremented in basic Update
	})).Return(nil)

	err := service.Update(context.Background(), 1, 1, RecTypeLogin, encryptedData, meta)
	assert.NoError(t, err)

	mockRepo.AssertExpectations(t)
}

func TestService_Update_NotFound(t *testing.T) {
	mockRepo := new(MockRepository)
	factory := NewFactory()
	logger := slog.Default()
	service := NewService(mockRepo, factory, logger)

	mockRepo.On("Get", mock.Anything, 1, 1).Return((*Record)(nil), ErrNotFound)

	err := service.Update(context.Background(), 1, 1, RecTypeLogin, "data", nil)
	assert.Error(t, err)
	assert.Equal(t, ErrNotFound, err)

	mockRepo.AssertExpectations(t)
}

func TestService_Update_Deleted(t *testing.T) {
	mockRepo := new(MockRepository)
	factory := NewFactory()
	logger := slog.Default()
	service := NewService(mockRepo, factory, logger)

	deletedAt := time.Now()
	record := &Record{
		ID:           1,
		UserID:       1,
		Type:         RecTypeLogin,
		Version:      1,
		LastModified: time.Now(),
		DeletedAt:    &deletedAt,
	}

	mockRepo.On("Get", mock.Anything, 1, 1).Return(record, nil)

	err := service.Update(context.Background(), 1, 1, RecTypeLogin, "data", nil)
	assert.Error(t, err)
	assert.Equal(t, ErrRecordDeleted, err)

	mockRepo.AssertExpectations(t)
}

func TestService_Delete(t *testing.T) {
	mockRepo := new(MockRepository)
	factory := NewFactory()
	logger := slog.Default()
	service := NewService(mockRepo, factory, logger)

	record := &Record{
		ID:           1,
		UserID:       1,
		Type:         RecTypeLogin,
		Version:      1,
		LastModified: time.Now(),
	}

	mockRepo.On("Get", mock.Anything, 1, 1).Return(record, nil)
	mockRepo.On("Delete", mock.Anything, 1, 1).Return(nil)
	mockRepo.On("SaveVersion", mock.Anything, mock.AnythingOfType("*record.Version")).Return(nil)

	err := service.Delete(context.Background(), 1, 1)
	assert.NoError(t, err)

	mockRepo.AssertExpectations(t)
}

func TestService_Delete_NotFound(t *testing.T) {
	mockRepo := new(MockRepository)
	factory := NewFactory()
	logger := slog.Default()
	service := NewService(mockRepo, factory, logger)

	mockRepo.On("Get", mock.Anything, 1, 1).Return((*Record)(nil), ErrNotFound)

	err := service.Delete(context.Background(), 1, 1)
	assert.Error(t, err)
	assert.Equal(t, ErrNotFound, err)

	mockRepo.AssertExpectations(t)
}

func TestService_SoftDelete(t *testing.T) {
	mockRepo := new(MockRepository)
	factory := NewFactory()
	logger := slog.Default()
	service := NewService(mockRepo, factory, logger)

	record := &Record{
		ID:           1,
		UserID:       1,
		Type:         RecTypeLogin,
		Version:      1,
		LastModified: time.Now(),
	}

	mockRepo.On("Get", mock.Anything, 1, 1).Return(record, nil)
	mockRepo.On("SoftDelete", mock.Anything, 1, 1).Return(nil)

	err := service.SoftDelete(context.Background(), 1, 1)
	assert.NoError(t, err)

	mockRepo.AssertExpectations(t)
}

func TestService_SoftDelete_AlreadyDeleted(t *testing.T) {
	mockRepo := new(MockRepository)
	factory := NewFactory()
	logger := slog.Default()
	service := NewService(mockRepo, factory, logger)

	deletedAt := time.Now()
	record := &Record{
		ID:           1,
		UserID:       1,
		Type:         RecTypeLogin,
		Version:      1,
		LastModified: time.Now(),
		DeletedAt:    &deletedAt,
	}

	mockRepo.On("Get", mock.Anything, 1, 1).Return(record, nil)

	err := service.SoftDelete(context.Background(), 1, 1)
	assert.NoError(t, err) // Should not return error if already deleted

	mockRepo.AssertExpectations(t)
}

func TestService_Search(t *testing.T) {
	mockRepo := new(MockRepository)
	factory := NewFactory()
	logger := slog.Default()
	service := NewService(mockRepo, factory, logger)

	records := []Record{
		{
			ID:           1,
			UserID:       1,
			Type:         RecTypeLogin,
			Version:      1,
			LastModified: time.Now(),
		},
	}

	criteria := SearchCriteria{
		Type: "login",
	}

	mockRepo.On("Search", mock.Anything, 1, criteria).Return(records, nil)

	result, err := service.Search(context.Background(), 1, criteria)
	assert.NoError(t, err)
	assert.Equal(t, records, result)

	mockRepo.AssertExpectations(t)
}

func TestService_GetStats(t *testing.T) {
	mockRepo := new(MockRepository)
	factory := NewFactory()
	logger := slog.Default()
	service := NewService(mockRepo, factory, logger)

	stats := map[string]interface{}{
		"total_records": int64(10),
		"total_size":    int64(1000),
		"by_type": map[string]map[string]interface{}{
			"login": {
				"count": int64(5),
				"size":  int64(500),
			},
			"text": {
				"count": int64(5),
				"size":  int64(500),
			},
		},
	}

	mockRepo.On("GetStats", mock.Anything, 1).Return(stats, nil)

	response, err := service.GetStats(context.Background(), 1)
	assert.NoError(t, err)
	assert.Equal(t, int64(10), response.TotalRecords)
	assert.Equal(t, int64(1000), response.TotalSize)
	assert.Equal(t, int64(5), response.ByType["login"].Count)
	assert.Equal(t, int64(500), response.ByType["login"].Size)
	assert.Equal(t, int64(5), response.ByType["text"].Count)
	assert.Equal(t, int64(500), response.ByType["text"].Size)

	mockRepo.AssertExpectations(t)
}

func TestService_GetModifiedSince(t *testing.T) {
	mockRepo := new(MockRepository)
	factory := NewFactory()
	logger := slog.Default()
	service := NewService(mockRepo, factory, logger)

	records := []Record{
		{
			ID:           1,
			UserID:       1,
			Type:         RecTypeLogin,
			Version:      1,
			LastModified: time.Now(),
		},
	}

	since := time.Now().Add(-1 * time.Hour)

	mockRepo.On("GetModifiedSince", mock.Anything, 1, since).Return(records, nil)

	result, err := service.GetModifiedSince(context.Background(), 1, since)
	assert.NoError(t, err)
	assert.Equal(t, records, result)

	mockRepo.AssertExpectations(t)
}

func TestService_BatchCreate(t *testing.T) {
	mockRepo := new(MockRepository)
	factory := NewFactory()
	logger := slog.Default()
	service := NewService(mockRepo, factory, logger)

	requests := []CreateRequest{
		{
			Type:          RecTypeLogin,
			EncryptedData: "data1",
			Meta:          json.RawMessage(`{"title": "test1"}`),
		},
		{
			Type:          RecTypeText,
			EncryptedData: "data2",
			Meta:          json.RawMessage(`{"title": "test2"}`),
		},
	}

	mockRepo.On("Create", mock.Anything, mock.MatchedBy(func(r *Record) bool {
		return r.UserID == 1 && r.Type == RecTypeLogin && r.EncryptedData == "data1"
	})).Return(1, nil)
	mockRepo.On("Create", mock.Anything, mock.MatchedBy(func(r *Record) bool {
		return r.UserID == 1 && r.Type == RecTypeText && r.EncryptedData == "data2"
	})).Return(2, nil)

	response, err := service.BatchCreate(context.Background(), 1, requests)
	assert.NoError(t, err)
	assert.Equal(t, 2, response.SuccessCount)
	assert.Equal(t, 0, response.FailedCount)
	assert.Empty(t, response.Failed)

	mockRepo.AssertExpectations(t)
}

func TestService_BatchCreate_WithErrors(t *testing.T) {
	mockRepo := new(MockRepository)
	factory := NewFactory()
	logger := slog.Default()
	service := NewService(mockRepo, factory, logger)

	requests := []CreateRequest{
		{
			Type:          RecTypeLogin,
			EncryptedData: "data1",
			Meta:          json.RawMessage(`{"title": "test1"}`),
		},
		{
			Type:          RecTypeText,
			EncryptedData: "data2",
			Meta:          json.RawMessage(`{"title": "test2"}`),
		},
	}

	mockRepo.On("Create", mock.Anything, mock.MatchedBy(func(r *Record) bool {
		return r.Type == RecTypeLogin
	})).Return(1, nil)
	mockRepo.On("Create", mock.Anything, mock.MatchedBy(func(r *Record) bool {
		return r.Type == RecTypeText
	})).Return(0, errors.New("database error"))

	response, err := service.BatchCreate(context.Background(), 1, requests)
	assert.NoError(t, err)
	assert.Equal(t, 1, response.SuccessCount)
	assert.Equal(t, 1, response.FailedCount)
	assert.Len(t, response.Failed, 1)
	assert.Equal(t, 1, response.Failed[0].Index)
	assert.Contains(t, response.Failed[0].Error, "database error")

	mockRepo.AssertExpectations(t)
}

func TestService_BatchUpdate(t *testing.T) {
	mockRepo := new(MockRepository)
	factory := NewFactory()
	logger := slog.Default()
	service := NewService(mockRepo, factory, logger)

	currentRecord := &Record{
		ID:           1,
		UserID:       1,
		Type:         RecTypeLogin,
		Version:      1,
		LastModified: time.Now(),
	}

	updates := []UpdateRequest{
		{
			RecordID:      1,
			Type:          RecTypeLogin,
			EncryptedData: "updated_data1",
			Meta:          json.RawMessage(`{"title": "updated1"}`),
			Version:       1,
		},
	}

	mockRepo.On("Get", mock.Anything, 1, 1).Return(currentRecord, nil)
	mockRepo.On("Update", mock.Anything, mock.MatchedBy(func(r *Record) bool {
		return r.ID == 1 && r.EncryptedData == "updated_data1"
	})).Return(nil)

	response, err := service.BatchUpdate(context.Background(), 1, updates)
	assert.NoError(t, err)
	assert.Equal(t, 1, response.SuccessCount)
	assert.Equal(t, 0, response.FailedCount)
	assert.Empty(t, response.Failed)

	mockRepo.AssertExpectations(t)
}

func TestService_BatchUpdate_VersionMismatch(t *testing.T) {
	mockRepo := new(MockRepository)
	factory := NewFactory()
	logger := slog.Default()
	service := NewService(mockRepo, factory, logger)

	currentRecord := &Record{
		ID:           1,
		UserID:       1,
		Type:         RecTypeLogin,
		Version:      2, // Current version is 2
		LastModified: time.Now(),
	}

	updates := []UpdateRequest{
		{
			RecordID:      1,
			Type:          RecTypeLogin,
			EncryptedData: "updated_data1",
			Meta:          json.RawMessage(`{"title": "updated1"}`),
			Version:       1, // But update request has version 1
		},
	}

	mockRepo.On("Get", mock.Anything, 1, 1).Return(currentRecord, nil)

	response, err := service.BatchUpdate(context.Background(), 1, updates)
	assert.NoError(t, err)
	assert.Equal(t, 0, response.SuccessCount)
	assert.Equal(t, 1, response.FailedCount)
	assert.Len(t, response.Failed, 1)
	assert.Equal(t, 0, response.Failed[0].Index)
	assert.Equal(t, 1, response.Failed[0].RecordID)
	assert.Equal(t, "version mismatch", response.Failed[0].Error)

	mockRepo.AssertExpectations(t)
}

func TestService_GenerateChecksum(t *testing.T) {
	mockRepo := new(MockRepository)
	factory := NewFactory()
	logger := slog.Default()
	service := NewService(mockRepo, factory, logger)

	encryptedData := "test_data"
	typ := RecTypeLogin
	meta := json.RawMessage(`{"title": "test"}`)

	checksum1 := service.(*Service).generateChecksum(encryptedData, typ, meta)
	checksum2 := service.(*Service).generateChecksum(encryptedData, typ, meta)

	// Same input should produce same checksum
	assert.Equal(t, checksum1, checksum2)

	// Different input should produce different checksum
	checksum3 := service.(*Service).generateChecksum("different_data", typ, meta)
	assert.NotEqual(t, checksum1, checksum3)
}

func TestService_GetByType(t *testing.T) {
	mockRepo := new(MockRepository)
	factory := NewFactory()
	logger := slog.Default()
	service := NewService(mockRepo, factory, logger)

	records := []Record{
		{
			ID:           1,
			UserID:       1,
			Type:         RecTypeLogin,
			Version:      1,
			LastModified: time.Now(),
		},
	}

	mockRepo.On("GetByType", mock.Anything, 1, "login").Return(records, nil)

	result, err := service.GetByType(context.Background(), 1, "login")
	assert.NoError(t, err)
	assert.Equal(t, records, result)

	mockRepo.AssertExpectations(t)
}

func TestService_GetVersions(t *testing.T) {
	mockRepo := new(MockRepository)
	factory := NewFactory()
	logger := slog.Default()
	service := NewService(mockRepo, factory, logger)

	record := &Record{
		ID:           1,
		UserID:       1,
		Type:         RecTypeLogin,
		Version:      1,
		LastModified: time.Now(),
	}

	versions := []Version{
		{
			ID:            1,
			RecordID:      1,
			Version:       1,
			EncryptedData: "data1",
			Checksum:      "checksum1",
			CreatedAt:     time.Now(),
		},
	}

	mockRepo.On("Get", mock.Anything, 1, 1).Return(record, nil)
	mockRepo.On("GetVersions", mock.Anything, 1).Return(versions, nil)

	result, err := service.GetVersions(context.Background(), 1, 1)
	assert.NoError(t, err)
	assert.Equal(t, versions, result)

	mockRepo.AssertExpectations(t)
}

func TestService_GetVersions_NotFound(t *testing.T) {
	mockRepo := new(MockRepository)
	factory := NewFactory()
	logger := slog.Default()
	service := NewService(mockRepo, factory, logger)

	mockRepo.On("Get", mock.Anything, 1, 1).Return((*Record)(nil), ErrNotFound)

	_, err := service.GetVersions(context.Background(), 1, 1)
	assert.Error(t, err)

	mockRepo.AssertExpectations(t)
}
