package client

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"golang.org/x/exp/slog"

	"gophkeeper/internal/app/client/config"
	"gophkeeper/internal/domain/record"
	"gophkeeper/internal/domain/sync"
	"gophkeeper/internal/domain/user"
)

type httpClient struct {
	client    *http.Client
	config    *config.Config
	log       *slog.Logger
	baseURL   string
	token     string
	userAgent string
}

func NewHTTPClient(cfg *config.Config, log *slog.Logger) (*httpClient, error) {
	client := &http.Client{
		Timeout: 30 * time.Second,
		Transport: &http.Transport{
			MaxIdleConns:        100,
			IdleConnTimeout:     90 * time.Second,
			DisableCompression:  false,
			DisableKeepAlives:   false,
			MaxIdleConnsPerHost: 10,
		},
	}

	// Определяем протокол
	scheme := "http://"
	if cfg.EnableTLS {
		scheme = "https://"
	}
	baseURL := scheme + cfg.ServerAddress

	return &httpClient{
		client:    client,
		config:    cfg,
		log:       log,
		baseURL:   baseURL,
		userAgent: "GophKeeper-Client/1.0",
	}, nil
}

// SetToken устанавливает токен аутентификации
func (h *httpClient) SetToken(token string) {
	h.token = token
}

// setAuthToken устанавливает токен аутентификации (alias для SetToken)
func (h *httpClient) setAuthToken(token string) {
	h.token = token
}

// HealthCheck проверяет доступность сервера
func (h *httpClient) HealthCheck(ctx context.Context) error {
	req, err := http.NewRequestWithContext(ctx, "GET", h.baseURL+"/api/v1/health", nil)
	if err != nil {
		return fmt.Errorf("ошибка создания запроса: %w", err)
	}

	req.Header.Set("User-Agent", h.userAgent)

	resp, err := h.client.Do(req)
	if err != nil {
		return fmt.Errorf("сервер недоступен: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("сервер вернул статус: %d", resp.StatusCode)
	}

	return nil
}

func (h *httpClient) doRequest(ctx context.Context, method, path string, body interface{}) (*http.Response, error) {
	var reqBody io.Reader
	if body != nil {
		jsonData, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("ошибка маршалинга тела запроса: %w", err)
		}
		reqBody = bytes.NewBuffer(jsonData)
	}

	req, err := http.NewRequestWithContext(ctx, method, h.baseURL+path, reqBody)
	if err != nil {
		return nil, fmt.Errorf("ошибка создания запроса: %w", err)
	}

	// Добавляем заголовки
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", h.userAgent)
	if h.token != "" {
		req.Header.Set("Authorization", "Bearer "+h.token)
	}

	h.log.Debug("Отправка запроса",
		"method", method,
		"url", req.URL.String(),
	)

	resp, err := h.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("ошибка выполнения запроса: %w", err)
	}

	return resp, nil
}

func (h *httpClient) parseResponse(resp *http.Response, result interface{}) error {
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("ошибка чтения ответа: %w", err)
	}

	h.log.Debug("Получен ответ",
		"status", resp.StatusCode,
		"body", string(body),
	)

	if resp.StatusCode >= 400 {
		var errResp struct {
			Error  string `json:"error"`
			Status string `json:"status"`
		}
		if err := json.Unmarshal(body, &errResp); err == nil && errResp.Error != "" {
			return fmt.Errorf("ошибка сервера: %s", errResp.Error)
		}
		return fmt.Errorf("ошибка сервера: статус %d", resp.StatusCode)
	}

	if result != nil && len(body) > 0 {
		if err := json.Unmarshal(body, result); err != nil {
			return fmt.Errorf("ошибка парсинга ответа: %w", err)
		}
	}

	return nil
}

// ==================== Auth API ====================

// Login выполняет вход пользователя
func (h *httpClient) Login(ctx context.Context, login, password string) (string, error) {
	req := user.BaseRequest{
		Login:    login,
		Password: password,
	}

	resp, err := h.doRequest(ctx, "POST", "/api/v1/auth/login", req)
	if err != nil {
		return "", err
	}

	var loginResp struct {
		Token  string `json:"token"`
		Status string `json:"status"`
		Error  string `json:"error"`
	}

	if err := h.parseResponse(resp, &loginResp); err != nil {
		return "", err
	}

	if loginResp.Status == "Error" {
		return "", fmt.Errorf("ошибка входа: %s", loginResp.Error)
	}

	h.setAuthToken(loginResp.Token)
	return loginResp.Token, nil
}

// Register регистрирует нового пользователя
func (h *httpClient) Register(ctx context.Context, login, password string) error {
	req := user.BaseRequest{
		Login:    login,
		Password: password,
	}

	resp, err := h.doRequest(ctx, "POST", "/api/v1/auth/register", req)
	if err != nil {
		return err
	}

	var registerResp struct {
		ID     int    `json:"user_id"`
		Status string `json:"status"`
		Error  string `json:"error"`
	}

	if err := h.parseResponse(resp, &registerResp); err != nil {
		return err
	}

	if registerResp.Status == "Error" {
		return fmt.Errorf("ошибка регистрации: %s", registerResp.Error)
	}

	return nil
}

// ChangePassword меняет пароль пользователя
func (h *httpClient) ChangePassword(ctx context.Context, req user.ChangePasswordRequest) error {
	resp, err := h.doRequest(ctx, "POST", "/api/v1/auth/change-password", req)
	if err != nil {
		return err
	}

	return h.parseResponse(resp, nil)
}

// ==================== Records API ====================

// RecordResponse - ответ сервера на операции с записями
type RecordResponse struct {
	ID      int    `json:"id,omitempty"`
	Status  string `json:"status"`
	Message string `json:"message,omitempty"`
	Error   string `json:"error,omitempty"`
}

// CreateLoginRequest - запрос на создание записи логина
type CreateLoginRequest struct {
	Username  string   `json:"username"`
	Password  string   `json:"password"`
	Notes     string   `json:"notes,omitempty"`
	Title     string   `json:"title"`
	Resource  string   `json:"resource"`
	Category  string   `json:"category,omitempty"`
	Tags      []string `json:"tags,omitempty"`
	TwoFA     bool     `json:"two_fa,omitempty"`
	TwoFAType string   `json:"two_fa_type,omitempty"`
	DeviceID  string   `json:"device_id,omitempty"`
}

// CreateTextRequest - запрос на создание текстовой записи
type CreateTextRequest struct {
	Content     string   `json:"content"`
	Title       string   `json:"title"`
	Category    string   `json:"category,omitempty"`
	Tags        []string `json:"tags,omitempty"`
	Format      string   `json:"format,omitempty"`
	Language    string   `json:"language,omitempty"`
	IsSensitive bool     `json:"is_sensitive,omitempty"`
	DeviceID    string   `json:"device_id,omitempty"`
}

// CreateCardRequest - запрос на создание записи карты
type CreateCardRequest struct {
	CardNumber     string   `json:"card_number"`
	CardHolder     string   `json:"card_holder"`
	ExpiryMonth    string   `json:"expiry_month"`
	ExpiryYear     string   `json:"expiry_year"`
	CVV            string   `json:"cvv"`
	PIN            string   `json:"pin,omitempty"`
	BillingAddress string   `json:"billing_address,omitempty"`
	Title          string   `json:"title"`
	BankName       string   `json:"bank_name,omitempty"`
	PaymentSystem  string   `json:"payment_system,omitempty"`
	Category       string   `json:"category,omitempty"`
	Tags           []string `json:"tags,omitempty"`
	Notes          string   `json:"notes,omitempty"`
	IsVirtual      bool     `json:"is_virtual,omitempty"`
	IsActive       bool     `json:"is_active,omitempty"`
	DailyLimit     *float64 `json:"daily_limit,omitempty"`
	PhoneNumber    string   `json:"phone_number,omitempty"`
	DeviceID       string   `json:"device_id,omitempty"`
}

// CreateBinaryRequest - запрос на создание бинарной записи
type CreateBinaryRequest struct {
	Data        string   `json:"data"` // base64
	Filename    string   `json:"filename"`
	ContentType string   `json:"content_type,omitempty"`
	Title       string   `json:"title"`
	Category    string   `json:"category,omitempty"`
	Tags        []string `json:"tags,omitempty"`
	Description string   `json:"description,omitempty"`
	DeviceID    string   `json:"device_id,omitempty"`
}

// GenericRecordRequest - generic запрос на создание записи
type GenericRecordRequest struct {
	Type record.RecType  `json:"type"`
	Data string          `json:"data"` // base64 encrypted data
	Meta json.RawMessage `json:"meta"`
}

// CreateRecord создает запись на сервере (generic)
func (h *httpClient) CreateRecord(ctx context.Context, req GenericRecordRequest) (int, error) {
	resp, err := h.doRequest(ctx, "POST", "/api/records", req)
	if err != nil {
		return 0, err
	}

	var createResp RecordResponse
	if err := h.parseResponse(resp, &createResp); err != nil {
		return 0, err
	}

	if createResp.Status == "Error" {
		return 0, fmt.Errorf("ошибка создания записи: %s", createResp.Error)
	}

	return createResp.ID, nil
}

// CreateLoginRecord создает запись логина на сервере
func (h *httpClient) CreateLoginRecord(ctx context.Context, req CreateLoginRequest) (int, error) {
	resp, err := h.doRequest(ctx, "POST", "/api/records/login", req)
	if err != nil {
		return 0, err
	}

	var createResp RecordResponse
	if err := h.parseResponse(resp, &createResp); err != nil {
		return 0, err
	}

	if createResp.Status == "Error" {
		return 0, fmt.Errorf("ошибка создания записи: %s", createResp.Error)
	}

	return createResp.ID, nil
}

// CreateTextRecord создает текстовую запись на сервере
func (h *httpClient) CreateTextRecord(ctx context.Context, req CreateTextRequest) (int, error) {
	resp, err := h.doRequest(ctx, "POST", "/api/records/text", req)
	if err != nil {
		return 0, err
	}

	var createResp RecordResponse
	if err := h.parseResponse(resp, &createResp); err != nil {
		return 0, err
	}

	if createResp.Status == "Error" {
		return 0, fmt.Errorf("ошибка создания записи: %s", createResp.Error)
	}

	return createResp.ID, nil
}

// CreateCardRecord создает запись карты на сервере
func (h *httpClient) CreateCardRecord(ctx context.Context, req CreateCardRequest) (int, error) {
	resp, err := h.doRequest(ctx, "POST", "/api/records/card", req)
	if err != nil {
		return 0, err
	}

	var createResp RecordResponse
	if err := h.parseResponse(resp, &createResp); err != nil {
		return 0, err
	}

	if createResp.Status == "Error" {
		return 0, fmt.Errorf("ошибка создания записи: %s", createResp.Error)
	}

	return createResp.ID, nil
}

// CreateBinaryRecord создает бинарную запись на сервере
func (h *httpClient) CreateBinaryRecord(ctx context.Context, req CreateBinaryRequest) (int, error) {
	resp, err := h.doRequest(ctx, "POST", "/api/records/binary", req)
	if err != nil {
		return 0, err
	}

	var createResp RecordResponse
	if err := h.parseResponse(resp, &createResp); err != nil {
		return 0, err
	}

	if createResp.Status == "Error" {
		return 0, fmt.Errorf("ошибка создания записи: %s", createResp.Error)
	}

	return createResp.ID, nil
}

// UpdateRecord обновляет запись на сервере
func (h *httpClient) UpdateRecord(ctx context.Context, id int, req GenericRecordRequest) error {
	resp, err := h.doRequest(ctx, "PUT", fmt.Sprintf("/api/records/%d", id), req)
	if err != nil {
		return err
	}

	var updateResp RecordResponse
	if err := h.parseResponse(resp, &updateResp); err != nil {
		return err
	}

	if updateResp.Status == "Error" {
		return fmt.Errorf("ошибка обновления записи: %s", updateResp.Error)
	}

	return nil
}

// DeleteRecord удаляет запись на сервере
func (h *httpClient) DeleteRecord(ctx context.Context, id int) error {
	resp, err := h.doRequest(ctx, "DELETE", fmt.Sprintf("/api/records/%d", id), nil)
	if err != nil {
		return err
	}

	var deleteResp RecordResponse
	if err := h.parseResponse(resp, &deleteResp); err != nil {
		return err
	}

	if deleteResp.Status == "Error" {
		return fmt.Errorf("ошибка удаления записи: %s", deleteResp.Error)
	}

	return nil
}

// GetRecord получает запись с сервера
func (h *httpClient) GetRecord(ctx context.Context, id int) (*record.Record, error) {
	resp, err := h.doRequest(ctx, "GET", fmt.Sprintf("/api/records/%d", id), nil)
	if err != nil {
		return nil, err
	}

	var findResp struct {
		Status string         `json:"status"`
		Record *record.Record `json:"record"`
		Error  string         `json:"error,omitempty"`
	}

	if err := h.parseResponse(resp, &findResp); err != nil {
		return nil, err
	}

	if findResp.Status == "Error" {
		return nil, fmt.Errorf("ошибка получения записи: %s", findResp.Error)
	}

	return findResp.Record, nil
}

// ListRecords получает список записей с сервера
func (h *httpClient) ListRecords(ctx context.Context) (*record.ListResponse, error) {
	resp, err := h.doRequest(ctx, "GET", "/api/records", nil)
	if err != nil {
		return nil, err
	}

	var listResp record.ListResponse
	if err := h.parseResponse(resp, &listResp); err != nil {
		return nil, err
	}

	return &listResp, nil
}

// ==================== Sync API ====================

// GetSyncChanges получает изменения с сервера
func (h *httpClient) GetSyncChanges(ctx context.Context, req sync.GetChangesRequest) (*sync.GetChangesResponse, error) {
	resp, err := h.doRequest(ctx, "POST", "/api/sync/changes", req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}

	var result sync.GetChangesResponse
	if err := h.parseResponse(resp, &result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	if result.Status == "Error" {
		return nil, fmt.Errorf("server error: %s", result.Error)
	}

	return &result, nil
}

// SendBatchSync отправляет пакет записей для синхронизации
func (h *httpClient) SendBatchSync(ctx context.Context, req sync.BatchSyncRequest) (*sync.BatchSyncResponse, error) {
	resp, err := h.doRequest(ctx, "POST", "/api/sync/batch", req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}

	var result sync.BatchSyncResponse
	if err := h.parseResponse(resp, &result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	if result.Status == "Error" {
		return nil, fmt.Errorf("server error: %s", result.Error)
	}

	return &result, nil
}

// GetSyncStatus получает статус синхронизации с сервера
func (h *httpClient) GetSyncStatus(ctx context.Context) (*sync.SyncStatus, error) {
	resp, err := h.doRequest(ctx, "GET", "/api/sync/status", nil)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}

	var result sync.GetStatusResponse
	if err := h.parseResponse(resp, &result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	if result.Status == "Error" {
		return nil, fmt.Errorf("server error: %s", result.Error)
	}

	return result.Data, nil
}

// GetSyncConflicts получает конфликты с сервера
func (h *httpClient) GetSyncConflicts(ctx context.Context) ([]sync.Conflict, error) {
	resp, err := h.doRequest(ctx, "GET", "/api/sync/conflicts", nil)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}

	var result sync.GetConflictsResponse
	if err := h.parseResponse(resp, &result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	if result.Status == "Error" {
		return nil, fmt.Errorf("server error: %s", result.Error)
	}

	return result.Data, nil
}

// ResolveConflict разрешает конфликт на сервере
func (h *httpClient) ResolveConflict(ctx context.Context, conflictID int, req sync.ResolveConflictRequest) error {
	resp, err := h.doRequest(ctx, "POST", fmt.Sprintf("/api/sync/conflicts/%d/resolve", conflictID), req)
	if err != nil {
		return fmt.Errorf("failed to send request: %w", err)
	}

	var result sync.ResolveConflictResponse
	if err := h.parseResponse(resp, &result); err != nil {
		return fmt.Errorf("failed to decode response: %w", err)
	}

	if result.Status == "Error" {
		return fmt.Errorf("server error: %s", result.Error)
	}

	return nil
}

// GetDevices получает список устройств с сервера
func (h *httpClient) GetDevices(ctx context.Context) ([]sync.DeviceInfo, error) {
	resp, err := h.doRequest(ctx, "GET", "/api/sync/devices", nil)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}

	var result sync.GetDevicesResponse
	if err := h.parseResponse(resp, &result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	if result.Status == "Error" {
		return nil, fmt.Errorf("server error: %s", result.Error)
	}

	return result.Data, nil
}

// RemoveDevice удаляет устройство на сервере
func (h *httpClient) RemoveDevice(ctx context.Context, deviceID int) error {
	resp, err := h.doRequest(ctx, "DELETE", fmt.Sprintf("/api/sync/devices/%d", deviceID), nil)
	if err != nil {
		return fmt.Errorf("failed to send request: %w", err)
	}

	var result sync.RemoveDeviceResponse
	if err := h.parseResponse(resp, &result); err != nil {
		return fmt.Errorf("failed to decode response: %w", err)
	}

	if result.Status == "Error" {
		return fmt.Errorf("server error: %s", result.Error)
	}

	return nil
}
