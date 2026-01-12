package client

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"golang.org/x/exp/slog"
	"gophkeeper/internal/domain/sync"
	"io"
	"net/http"
	"time"

	"gophkeeper/internal/app/client/config"
	"gophkeeper/internal/domain/record"
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

// CreateRecord создает запись на сервере
func (h *httpClient) CreateRecord(ctx context.Context, req record.CreateRequest) (string, error) {
	resp, err := h.doRequest(ctx, "POST", "/api/v1/records", req)
	if err != nil {
		return "", err
	}

	var createResp struct {
		ID string `json:"id"`
	}

	if err := h.parseResponse(resp, &createResp); err != nil {
		return "", err
	}

	return createResp.ID, nil
}

// UpdateRecord обновляет запись на сервере
func (h *httpClient) UpdateRecord(ctx context.Context, id string, req record.UpdateRequest) error {
	resp, err := h.doRequest(ctx, "PUT", "/api/v1/records/"+id, req)
	if err != nil {
		return err
	}

	return h.parseResponse(resp, nil)
}

// ChangePassword меняет пароль пользователя
func (h *httpClient) ChangePassword(ctx context.Context, req user.ChangePasswordRequest) error {
	resp, err := h.doRequest(ctx, "POST", "/api/v1/auth/change-password", req)
	if err != nil {
		return err
	}

	return h.parseResponse(resp, nil)
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
			Error string `json:"error"`
		}
		if err := json.Unmarshal(body, &errResp); err == nil && errResp.Error != "" {
			return fmt.Errorf("ошибка сервера: %s", errResp.Error)
		}
		return fmt.Errorf("ошибка сервера: статус %d", resp.StatusCode)
	}

	if result != nil {
		if err := json.Unmarshal(body, result); err != nil {
			return fmt.Errorf("ошибка парсинга ответа: %w", err)
		}
	}

	return nil
}

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
		Token string `json:"token"`
	}

	if err := h.parseResponse(resp, &loginResp); err != nil {
		return "", err
	}

	h.setAuthToken(loginResp.Token)
	return loginResp.Token, nil
}

func (h *httpClient) Register(ctx context.Context, login, password string) error {
	req := user.BaseRequest{
		Login:    login,
		Password: password,
	}

	resp, err := h.doRequest(ctx, "POST", "/api/v1/auth/register", req)
	if err != nil {
		return err
	}

	return h.parseResponse(resp, nil)
}

func (h *httpClient) GetRecords(ctx context.Context) ([]*record.Record, error) {
	resp, err := h.doRequest(ctx, "GET", "/api/v1/records", nil)
	if err != nil {
		return nil, err
	}

	var recordsResp struct {
		Records []*record.Record `json:"records"`
	}

	if err := h.parseResponse(resp, &recordsResp); err != nil {
		return nil, err
	}

	return recordsResp.Records, nil
}

func (h *httpClient) SyncRecords(ctx context.Context, records []*record.Record) error {
	req := struct {
		Records []*record.Record `json:"records"`
	}{
		Records: records,
	}

	resp, err := h.doRequest(ctx, "POST", "/api/v1/records/sync", req)
	if err != nil {
		return err
	}

	return h.parseResponse(resp, nil)
}

// GetSyncChanges получает изменения с сервера
func (c *httpClient) GetSyncChanges(ctx context.Context, req sync.GetChangesRequest) (*sync.GetChangesResponse, error) {
	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	request, err := http.NewRequestWithContext(
		ctx,
		"POST",
		fmt.Sprintf("%s/api/sync/changes", c.baseURL),
		bytes.NewBuffer(body),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	request.Header.Set("Content-Type", "application/json")
	if c.token != "" {
		request.Header.Set("Authorization", "Bearer "+c.token)
	}

	response, err := c.client.Do(request)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("server returned status: %d", response.StatusCode)
	}

	var result sync.GetChangesResponse
	if err := json.NewDecoder(response.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	if result.Status == "Error" {
		return nil, fmt.Errorf("server error: %s", result.Error)
	}

	return &result, nil
}

func (c *httpClient) SendBatchSync(ctx context.Context, req sync.BatchSyncRequest) (*sync.BatchSyncResponse, error) {
	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	request, err := http.NewRequestWithContext(
		ctx,
		"POST",
		fmt.Sprintf("%s/api/sync/batch", c.baseURL),
		bytes.NewBuffer(body),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	request.Header.Set("Content-Type", "application/json")
	if c.token != "" {
		request.Header.Set("Authorization", "Bearer "+c.token)
	}

	response, err := c.client.Do(request)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("server returned status: %d", response.StatusCode)
	}

	var result sync.BatchSyncResponse
	if err := json.NewDecoder(response.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	if result.Status == "Error" {
		return nil, fmt.Errorf("server error: %s", result.Error)
	}

	return &result, nil
}

// GetSyncStatus получает статус синхронизации с сервера
func (c *httpClient) GetSyncStatus(ctx context.Context) (*sync.SyncStatus, error) {
	request, err := http.NewRequestWithContext(
		ctx,
		"GET",
		fmt.Sprintf("%s/api/sync/status", c.baseURL),
		nil,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	request.Header.Set("Content-Type", "application/json")
	if c.token != "" {
		request.Header.Set("Authorization", "Bearer "+c.token)
	}

	response, err := c.client.Do(request)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("server returned status: %d", response.StatusCode)
	}

	var result struct {
		Status string           `json:"status"`
		Error  string           `json:"error,omitempty"`
		Data   *sync.SyncStatus `json:"data,omitempty"`
	}

	if err := json.NewDecoder(response.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	if result.Status == "Error" {
		return nil, fmt.Errorf("server error: %s", result.Error)
	}

	return result.Data, nil
}

// GetSyncConflicts получает конфликты с сервера
func (c *httpClient) GetSyncConflicts(ctx context.Context) ([]sync.Conflict, error) {
	request, err := http.NewRequestWithContext(
		ctx,
		"GET",
		fmt.Sprintf("%s/api/sync/conflicts", c.baseURL),
		nil,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	request.Header.Set("Content-Type", "application/json")
	if c.token != "" {
		request.Header.Set("Authorization", "Bearer "+c.token)
	}

	response, err := c.client.Do(request)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("server returned status: %d", response.StatusCode)
	}

	var result struct {
		Status string          `json:"status"`
		Error  string          `json:"error,omitempty"`
		Data   []sync.Conflict `json:"data,omitempty"`
	}

	if err := json.NewDecoder(response.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	if result.Status == "Error" {
		return nil, fmt.Errorf("server error: %s", result.Error)
	}

	return result.Data, nil
}

// ResolveConflict разрешает конфликт на сервере
func (c *httpClient) ResolveConflict(ctx context.Context, conflictID int, resolution string, record *sync.RecordSync) error {
	req := sync.ResolveConflictRequest{
		Resolution:   resolution,
		ResolvedData: record,
	}

	body, err := json.Marshal(req)
	if err != nil {
		return fmt.Errorf("failed to marshal request: %w", err)
	}

	request, err := http.NewRequestWithContext(
		ctx,
		"POST",
		fmt.Sprintf("%s/api/sync/conflicts/%d/resolve", c.baseURL, conflictID),
		bytes.NewBuffer(body),
	)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	request.Header.Set("Content-Type", "application/json")
	if c.token != "" {
		request.Header.Set("Authorization", "Bearer "+c.token)
	}

	response, err := c.client.Do(request)
	if err != nil {
		return fmt.Errorf("failed to send request: %w", err)
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		return fmt.Errorf("server returned status: %d", response.StatusCode)
	}

	var result sync.ResolveConflictResponse

	if err := json.NewDecoder(response.Body).Decode(&result); err != nil {
		return fmt.Errorf("failed to decode response: %w", err)
	}

	if result.Status == "Error" {
		return fmt.Errorf("server error: %s", result.Error)
	}

	return nil
}

// GetDevices получает список устройств с сервера
func (c *httpClient) GetDevices(ctx context.Context) ([]sync.DeviceInfo, error) {
	request, err := http.NewRequestWithContext(
		ctx,
		"GET",
		fmt.Sprintf("%s/api/sync/devices", c.baseURL),
		nil,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	request.Header.Set("Content-Type", "application/json")
	if c.token != "" {
		request.Header.Set("Authorization", "Bearer "+c.token)
	}

	response, err := c.client.Do(request)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("server returned status: %d", response.StatusCode)
	}

	var result struct {
		Status string            `json:"status"`
		Error  string            `json:"error,omitempty"`
		Data   []sync.DeviceInfo `json:"data,omitempty"`
	}

	if err := json.NewDecoder(response.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	if result.Status == "Error" {
		return nil, fmt.Errorf("server error: %s", result.Error)
	}

	return result.Data, nil
}

// RemoveDevice удаляет устройство на сервере
func (c *httpClient) RemoveDevice(ctx context.Context, deviceID int) error {
	request, err := http.NewRequestWithContext(
		ctx,
		"DELETE",
		fmt.Sprintf("%s/api/sync/devices/%d", c.baseURL, deviceID),
		nil,
	)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	request.Header.Set("Content-Type", "application/json")
	if c.token != "" {
		request.Header.Set("Authorization", "Bearer "+c.token)
	}

	response, err := c.client.Do(request)
	if err != nil {
		return fmt.Errorf("failed to send request: %w", err)
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		return fmt.Errorf("server returned status: %d", response.StatusCode)
	}

	var result struct {
		Status  string `json:"status"`
		Error   string `json:"error,omitempty"`
		Message string `json:"message,omitempty"`
	}

	if err := json.NewDecoder(response.Body).Decode(&result); err != nil {
		return fmt.Errorf("failed to decode response: %w", err)
	}

	if result.Status == "Error" {
		return fmt.Errorf("server error: %s", result.Error)
	}

	return nil
}
