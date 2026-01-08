package client

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"golang.org/x/exp/slog"
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

func (h *httpClient) CreateRecord(ctx context.Context, rec *record.Record) error {
	req := record.CreateRequest{
		Type:          rec.Type,
		Meta:          rec.Meta,
		EncryptedData: rec.EncryptedData,
	}

	resp, err := h.doRequest(ctx, "POST", "/api/v1/records", req)
	if err != nil {
		return err
	}

	var createResp struct {
		ID int `json:"id"`
	}

	if err := h.parseResponse(resp, &createResp); err != nil {
		return err
	}

	rec.ID = createResp.ID
	return nil
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
