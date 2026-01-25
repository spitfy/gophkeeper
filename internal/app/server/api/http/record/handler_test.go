package record

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"gophkeeper/internal/app/server/api/http/middleware/auth"
	"gophkeeper/internal/domain/record"
	"testing"
	"time"
)

type MockService struct {
	mock.Mock
}

func (m *MockService) List(ctx context.Context, userID int) (record.ListResponse, error) {
	args := m.Called(ctx, userID)
	return args.Get(0).(record.ListResponse), args.Error(1)
}

func (m *MockService) Create(ctx context.Context, userID int, typ record.RecType, encryptedData string, meta json.RawMessage) (int, error) {
	args := m.Called(ctx, userID, typ, encryptedData, meta)
	return args.Int(0), args.Error(1)
}

func (m *MockService) Find(ctx context.Context, userID, recordID int) (*record.Record, error) {
	args := m.Called(ctx, userID, recordID)
	// Безопасное приведение nil к указателю
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*record.Record), args.Error(1)
}

func (m *MockService) Update(ctx context.Context, userID, recordID int, typ record.RecType, encryptedData string, meta json.RawMessage) error {
	args := m.Called(ctx, userID, recordID, typ, encryptedData, meta)
	return args.Error(0)
}

func (m *MockService) Delete(ctx context.Context, userID, recordID int) error {
	args := m.Called(ctx, userID, recordID)
	return args.Error(0)
}

func (m *MockService) SoftDelete(ctx context.Context, userID, recordID int) error {
	args := m.Called(ctx, userID, recordID)
	return args.Error(0)
}

func (m *MockService) Search(ctx context.Context, userID int, criteria record.SearchCriteria) ([]record.Record, error) {
	args := m.Called(ctx, userID, criteria)
	return args.Get(0).([]record.Record), args.Error(1)
}

func (m *MockService) GetStats(ctx context.Context, userID int) (record.StatsResponse, error) {
	args := m.Called(ctx, userID)
	return args.Get(0).(record.StatsResponse), args.Error(1)
}

func (m *MockService) GetModifiedSince(ctx context.Context, userID int, since time.Time) ([]record.Record, error) {
	args := m.Called(ctx, userID, since)
	return args.Get(0).([]record.Record), args.Error(1)
}

func (m *MockService) BatchCreate(ctx context.Context, userID int, records []record.CreateRequest) (record.BatchCreateResponse, error) {
	args := m.Called(ctx, userID, records)
	return args.Get(0).(record.BatchCreateResponse), args.Error(1)
}

func (m *MockService) BatchUpdate(ctx context.Context, userID int, updates []record.UpdateRequest) (record.BatchUpdateResponse, error) {
	args := m.Called(ctx, userID, updates)
	return args.Get(0).(record.BatchUpdateResponse), args.Error(1)
}

func (m *MockService) GetByType(ctx context.Context, userID int, recordType string) ([]record.Record, error) {
	args := m.Called(ctx, userID, recordType)
	return args.Get(0).([]record.Record), args.Error(1)
}

func (m *MockService) GetVersions(ctx context.Context, userID, recordID int) ([]record.Version, error) {
	args := m.Called(ctx, userID, recordID)
	return args.Get(0).([]record.Version), args.Error(1)
}

func (m *MockService) UpdateWithModels(ctx context.Context, recordID int, userID int, data record.Data, meta record.MetaData, deviceID string) error {
	args := m.Called(ctx, recordID, userID, data, meta, deviceID)
	return args.Error(0)
}

func (m *MockService) GetRecordWithModels(ctx context.Context, recordID, userID int) (record.Data, record.MetaData, error) {
	args := m.Called(ctx, recordID, userID)
	return args.Get(0).(record.Data), args.Get(1).(record.MetaData), args.Error(2)
}

func (m *MockService) CreateWithModels(ctx context.Context, userID int, typ record.RecType, data record.Data, meta record.MetaData, deviceID string) (int, error) {
	args := m.Called(ctx, userID, typ, data, meta, deviceID)
	return args.Int(0), args.Error(1)
}

func TestHandler_CreateBinary(t *testing.T) {
	userID := 123

	// Хелпер для создания контекста с авторизацией
	authCtx := auth.WithUserID(context.Background(), userID)

	t.Run("Success_ValidBase64", func(t *testing.T) {
		svc := new(MockService)
		h := NewHandler(svc, nil, nil)

		rawData := []byte("hello binary")
		encoded := base64.StdEncoding.EncodeToString(rawData)

		input := &createBinaryInput{}
		input.Body.Data = encoded
		input.Body.Filename = "test.bin"
		input.Body.Title = "My File"

		// Ожидаем вызов сервиса с декодированными данными
		svc.On("CreateWithModels",
			mock.Anything,
			userID,
			record.RecTypeBinary,
			mock.MatchedBy(func(d *record.BinaryData) bool {
				return string(d.Data) == "hello binary" && d.Size == 12
			}),
			mock.MatchedBy(func(m *record.BinaryMeta) bool {
				return m.Title == "My File"
			}),
			mock.Anything,
		).Return(123, nil)

		resp, err := h.createBinary(authCtx, input)

		assert.NoError(t, err)
		assert.Equal(t, 123, resp.Body.ID)
		assert.Equal(t, "Ok", resp.Body.Status)
	})

	t.Run("Error_InvalidBase64", func(t *testing.T) {
		h := NewHandler(nil, nil, nil)
		input := &createBinaryInput{}
		input.Body.Data = "!!!не-base64!!!"

		resp, err := h.createBinary(authCtx, input)

		assert.Nil(t, resp)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "Invalid base64")
	})
}

func TestHandler_CreateText_MetadataLogic(t *testing.T) {
	svc := new(MockService)
	h := NewHandler(svc, nil, nil)

	userID := 42
	ctx := auth.WithUserID(context.Background(), userID)

	input := &createTextInput{}
	input.Body.Content = "Go is awesome! 🚀"
	input.Body.Title = "Note"

	svc.On("CreateWithModels",
		mock.Anything, // ctx
		userID,        // В твоем CreateWithModels почему-то userID string?
		record.RecTypeText,
		mock.Anything,
		mock.MatchedBy(func(m *record.TextMeta) bool {
			return m.WordCount == 4
		}),
		mock.Anything,
	).Return(1, nil)

	// Вызываем метод
	_, err := h.createText(ctx, input)

	// Если тут падает Unauthorized, значит auth.GetUserID(ctx) вернул ok=false
	assert.NoError(t, err)
}
