package client

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/docker/docker/daemon/logger"
	"golang.org/x/exp/slog"
	"gophkeeper/internal/domain/record"
	"gophkeeper/internal/domain/user"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"gophkeeper/internal/app/client/config"
	"gophkeeper/internal/app/client/crypto"
)

type App struct {
	config         *config.Config
	log            *slog.Logger
	crypto         *crypto.MasterKeyManager
	encryptor      *crypto.RecordEncryptor
	httpClient     *HTTPClient
	storage        Storage
	sync           *SyncService
	state          *AppState
	masterKeyReady bool
	authenticated  bool
	wg             sync.WaitGroup
	cancel         context.CancelFunc
	mu             sync.RWMutex
}

// AppState хранит состояние приложения
type AppState struct {
	Initialized   bool      `json:"initialized"`
	UserEmail     string    `json:"user_email"`
	LastSync      time.Time `json:"last_sync"`
	RecordsCount  int       `json:"records_count"`
	MasterKeyHash string    `json:"master_key_hash"`
}

// Storage интерфейс для локального хранилища
type Storage interface {
	// Методы для локального хранения данных
	SaveRecord(record *record.Record) error
	GetRecord(id string) (*record.Record, error)
	ListRecords() ([]*record.Record, error)
	DeleteRecord(id string) error
	UpdateRecord(record *record.Record) error
	CountRecords() (int, error)
	Close() error
}

type HTTPClient interface {
	// Методы для HTTP взаимодействия с сервером
	Login(ctx context.Context, login, password string) (string, error)
	Register(ctx context.Context, login, password string) error
	//ChangePassword(ctx context.Context, req user.ChangePasswordRequest) error
	CreateRecord(ctx context.Context, record *record.Record) error
	GetRecords(ctx context.Context) ([]*record.Record, error)
	SyncRecords(ctx context.Context, records []*record.Record) error
}

func New(cfg *config.Config, log *slog.Logger) (*App, error) {
	// Инициализируем менеджер мастер-ключа
	state, err := loadAppState(cfg)
	if err != nil {
		log.Warn("Не удалось загрузить состояние приложения", "error", err)
		state = &AppState{}
	}

	masterKey, err := crypto.NewMasterKeyManager(cfg.MasterKeyPath)
	if err != nil {
		return nil, fmt.Errorf("ошибка инициализации мастер-ключа: %w", err)
	}

	encryptor := crypto.NewRecordEncryptor(masterKey)

	// Инициализируем HTTP клиент
	httpCl, err := NewHTTPClient(cfg, log)
	if err != nil {
		return nil, fmt.Errorf("ошибка инициализации HTTP клиента: %w", err)
	}

	// Инициализируем локальное хранилище (используем SQLite)
	storage, err := NewSQLiteStorage(cfg.DataPath)
	if err != nil {
		log.Warn("Не удалось инициализировать SQLite, используем память", "error", err)
		storage = NewMemoryStorage()
	}

	app := &App{
		config:     cfg,
		log:        log,
		crypto:     masterKey,
		encryptor:  encryptor,
		httpClient: httpCl,
		storage:    storage,
		state:      state,
	}

	// Инициализируем сервис синхронизации
	app.sync = NewSyncService(app)

	return app, nil
}

func loadAppState(cfg *config.Config) (*AppState, error) {
	statePath := cfg.ConfigDir + "/state.json"

	if _, err := os.Stat(statePath); os.IsNotExist(err) {
		return &AppState{}, nil
	}

	data, err := os.ReadFile(statePath)
	if err != nil {
		return nil, err
	}

	var state AppState
	if err := json.Unmarshal(data, &state); err != nil {
		return nil, err
	}

	return &state, nil
}

func (a *App) saveAppState() error {
	a.mu.RLock()
	defer a.mu.RUnlock()

	statePath := a.config.ConfigDir + "/state.json"
	data, err := json.MarshalIndent(a.state, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(statePath, data, 0600)
}

func (a *App) Run() error {
	ctx, cancel := context.WithCancel(context.Background())
	a.cancel = cancel

	// Запускаем обработку сигналов
	go a.handleSignals()

	// Запускаем фоновую синхронизацию
	a.wg.Add(1)
	go func() {
		defer a.wg.Done()
		a.startSync(ctx)
	}()

	// Здесь будет запуск CLI интерфейса
	a.log.Info("Клиент запущен",
		"server", a.config.ServerAddress,
		"env", a.config.Env,
	)

	// Блокируем main горутину
	a.wg.Wait()
	return nil
}

// IsInitialized проверяет, инициализирован ли клиент
func (a *App) IsInitialized() bool {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.state.Initialized
}

// InitMasterKey инициализирует мастер-ключ
func (a *App) InitMasterKey(password string) error {
	a.mu.Lock()
	defer a.mu.Unlock()

	// Генерируем мастер-ключ
	if err := a.crypto.GenerateMasterKey(password); err != nil {
		return fmt.Errorf("ошибка генерации мастер-ключа: %w", err)
	}

	// Сохраняем хэш мастер-ключа для проверки в будущем
	keyHash, err := a.crypto.GetKeyHash()
	if err != nil {
		return fmt.Errorf("ошибка получения хэша ключа: %w", err)
	}

	a.state.MasterKeyHash = keyHash
	a.masterKeyReady = true
	a.state.Initialized = true

	if err := a.saveAppState(); err != nil {
		return fmt.Errorf("ошибка сохранения состояния: %w", err)
	}

	return nil
}

// CheckConnection проверяет соединение с сервером
func (a *App) CheckConnection() error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	return a.httpClient.HealthCheck(ctx)
}

// InitStorage инициализирует хранилище
func (a *App) InitStorage() error {
	// Для SQLiteStorage создание таблиц происходит автоматически
	// Для MemoryStorage ничего делать не нужно
	if _, ok := a.storage.(*SQLiteStorage); ok {
		// Проверяем, что таблицы созданы
		count, err := a.storage.CountRecords()
		if err != nil {
			return fmt.Errorf("ошибка инициализации хранилища: %w", err)
		}
		a.state.RecordsCount = count
	}

	return nil
}

// UnlockMasterKey разблокирует мастер-ключ
func (a *App) UnlockMasterKey(password string) error {
	a.mu.Lock()
	defer a.mu.Unlock()

	if err := a.crypto.UnlockMasterKey(password); err != nil {
		return fmt.Errorf("неверный мастер-пароль: %w", err)
	}

	a.masterKeyReady = true
	return nil
}

// HasLocalData проверяет наличие локальных данных
func (a *App) HasLocalData() bool {
	count, err := a.storage.CountRecords()
	if err != nil {
		return false
	}
	return count > 0
}

func (a *App) startSync(ctx context.Context) {
	ticker := time.NewTicker(time.Duration(a.config.SyncInterval) * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			a.log.Info("Синхронизация остановлена")
			return
		case <-ticker.C:
			if err := a.sync.Sync(ctx); err != nil {
				a.log.Error("Ошибка синхронизации", err)
			}
		}
	}
}

func (a *App) handleSignals() {
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)

	sig := <-sigChan
	a.log.Info("Получен сигнал завершения", "signal", sig.String())

	if a.cancel != nil {
		a.cancel()
	}
}

func (a *App) Shutdown() {
	a.log.Info("Завершение работы клиента...")

	if a.cancel != nil {
		a.cancel()
	}

	a.wg.Wait()
	a.log.Info("Клиент завершил работу")
}

// Вспомогательные методы
func (a *App) IsAuthenticated() bool {
	// Проверяем наличие токена
	_, err := os.ReadFile(a.config.TokenPath)
	return err == nil
}

func (a *App) GetToken() (string, error) {
	tokenBytes, err := os.ReadFile(a.config.TokenPath)
	if err != nil {
		return "", fmt.Errorf("ошибка чтения токена: %w", err)
	}
	return string(tokenBytes), nil
}

// Register регистрирует нового пользователя
func (a *App) Register(ctx context.Context, req user.BaseRequest) error {
	if err := a.httpClient.Register(ctx, req); err != nil {
		return err
	}

	a.log.Info("Пользователь успешно зарегистрирован", "email", req.Email)
	return nil
}

// Login выполняет вход пользователя
func (a *App) Login(ctx context.Context, req user.BaseRequest) (string, error) {
	token, err := a.httpClient.Login(ctx, req)
	if err != nil {
		return "", err
	}

	// Сохраняем токен
	if err := a.SaveToken(token); err != nil {
		return "", fmt.Errorf("ошибка сохранения токена: %w", err)
	}

	a.mu.Lock()
	a.authenticated = true
	a.state.UserEmail = req.Email
	a.mu.Unlock()

	// Сохраняем состояние
	if err := a.saveAppState(); err != nil {
		a.log.Warn("Не удалось сохранить состояние", "error", err)
	}

	a.log.Info("Вход выполнен успешно", "email", req.Email)
	return token, nil
}

// ChangePassword изменяет пароль пользователя
func (a *App) ChangePassword(ctx context.Context, req user.ChangePasswordRequest) error {
	// Проверяем, что мастер-ключ разблокирован
	if !a.masterKeyReady {
		return fmt.Errorf("мастер-ключ не разблокирован")
	}

	// Получаем текущий токен
	token, err := a.GetToken()
	if err != nil {
		return fmt.Errorf("токен не найден: %w", err)
	}

	// Устанавливаем токен для HTTP клиента
	a.httpClient.SetToken(token)

	// Отправляем запрос на смену пароля
	if err := a.httpClient.ChangePassword(ctx, req); err != nil {
		return err
	}

	// Если изменился мастер-пароль, нужно перешифровать локальные данные
	if req.MasterPassword != "" {
		if err := a.reencryptLocalData(req.MasterPassword); err != nil {
			return fmt.Errorf("ошибка перешифровки данных: %w", err)
		}
	}

	a.log.Info("Пароль успешно изменен")
	return nil
}

// SaveToken сохраняет токен аутентификации
func (a *App) SaveToken(token string) error {
	if err := os.WriteFile(a.config.TokenPath, []byte(token), 0600); err != nil {
		return fmt.Errorf("ошибка сохранения токена: %w", err)
	}

	// Устанавливаем токен для HTTP клиента
	a.httpClient.SetToken(token)

	return nil
}

// GetToken возвращает сохраненный токен
func (a *App) GetToken() (string, error) {
	tokenBytes, err := os.ReadFile(a.config.TokenPath)
	if err != nil {
		if os.IsNotExist(err) {
			return "", fmt.Errorf("токен не найден. Выполните вход: gophkeeper auth login")
		}
		return "", fmt.Errorf("ошибка чтения токена: %w", err)
	}
	return string(tokenBytes), nil
}

// ClearToken удаляет токен
func (a *App) ClearToken() error {
	a.mu.Lock()
	defer a.mu.Unlock()

	a.authenticated = false
	a.state.UserEmail = ""

	if err := os.Remove(a.config.TokenPath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("ошибка удаления токена: %w", err)
	}

	if err := a.saveAppState(); err != nil {
		return fmt.Errorf("ошибка сохранения состояния: %w", err)
	}

	return nil
}

// IsAuthenticated проверяет, аутентифицирован ли пользователь
func (a *App) IsAuthenticated() bool {
	a.mu.RLock()
	defer a.mu.RUnlock()

	if !a.authenticated {
		// Проверяем наличие токена
		token, err := a.GetToken()
		if err == nil && token != "" {
			a.authenticated = true
		}
	}

	return a.authenticated
}

func (a *App) reencryptLocalData(newMasterPassword string) error {
	// Получаем все записи
	records, err := a.storage.ListRecords(&RecordFilter{})
	if err != nil {
		return fmt.Errorf("ошибка получения записей: %w", err)
	}

	// Перешифровываем каждую запись
	for _, rec := range records {
		// Расшифровываем старым ключом
		decryptedData, err := a.crypto.DecryptData(rec.Data)
		if err != nil {
			return fmt.Errorf("ошибка расшифровки данных записи %s: %w", rec.ID, err)
		}

		// Шифруем новым ключом
		encryptedData, err := a.crypto.EncryptDataWithPassword(decryptedData, newMasterPassword)
		if err != nil {
			return fmt.Errorf("ошибка шифровки данных записи %s: %w", rec.ID, err)
		}

		// Обновляем запись
		rec.Data = encryptedData
		if err := a.storage.UpdateRecord(rec); err != nil {
			return fmt.Errorf("ошибка обновления записи %s: %w", rec.ID, err)
		}
	}

	return nil
}

// CreateRecord создает новую запись
func (a *App) CreateRecord(ctx context.Context, rec *record.Record) error {
	// Проверяем аутентификацию
	if !a.IsAuthenticated() {
		return fmt.Errorf("требуется аутентификация. Выполните: gophkeeper auth login")
	}

	// Проверяем мастер-ключ
	if !a.masterKeyReady {
		return fmt.Errorf("мастер-ключ не разблокирован")
	}

	// Генерируем ID если не задан
	if rec.ID == "" {
		rec.ID = generateID()
	}

	// Устанавливаем временные метки
	now := time.Now()
	rec.CreatedAt = now
	rec.UpdatedAt = now
	rec.Version = 1

	// Шифруем данные
	encryptedData, err := a.encryptor.EncryptRecord(rec.EncryptedData)
	if err != nil {
		return fmt.Errorf("ошибка шифрования данных: %w", err)
	}

	// Создаем локальную запись
	localRec := &Record{
		ID:        rec.ID,
		Type:      rec.Type,
		Metadata:  rec.Metadata,
		Data:      encryptedData,
		CreatedAt: rec.CreatedAt,
		UpdatedAt: rec.UpdatedAt,
		Version:   rec.Version,
		Synced:    false,
		Deleted:   false,
	}

	// Сохраняем локально
	if err := a.storage.SaveRecord(localRec); err != nil {
		return fmt.Errorf("ошибка сохранения записи: %w", err)
	}

	// Пытаемся синхронизировать с сервером
	if a.IsAuthenticated() {
		createReq := record.CreateRequest{
			Type:     rec.Type,
			Metadata: rec.Metadata,
			Data:     rec.Data, // Отправляем незашифрованные данные (сервер сам шифрует)
		}

		serverID, err := a.httpClient.CreateRecord(ctx, createReq)
		if err != nil {
			a.log.Warn("Не удалось синхронизировать запись с сервером", "error", err, "record_id", rec.ID)
			// Продолжаем работу в офлайн-режиме
			return nil
		}

		// Обновляем ID если сервер вернул свой
		if serverID != rec.ID {
			localRec.ID = serverID
			localRec.Synced = true
			if err := a.storage.UpdateRecord(localRec); err != nil {
				return fmt.Errorf("ошибка обновления ID записи: %w", err)
			}
		}
	}

	a.state.RecordsCount++
	if err := a.saveAppState(); err != nil {
		a.log.Warn("Не удалось сохранить состояние", "error", err)
	}

	return nil
}

// GetRecord возвращает запись по ID
func (a *App) GetRecord(ctx context.Context, id string, showPassword bool) (*Record, error) {
	// Пытаемся получить из локального хранилища
	localRec, err := a.storage.GetRecord(id)
	if err != nil {
		// Если нет локально, пробуем получить с сервера
		if a.IsAuthenticated() {
			serverRec, err := a.httpClient.GetRecord(ctx, id)
			if err != nil {
				return nil, fmt.Errorf("запись не найдена: %w", err)
			}

			// Сохраняем локально
			if err := a.storage.SaveRecord(serverRec); err != nil {
				a.log.Warn("Не удалось сохранить запись локально", "error", err)
			}

			return serverRec, nil
		}
		return nil, fmt.Errorf("запись не найдена: %w", err)
	}

	// Расшифровываем данные если нужно
	if showPassword && a.crypto.IsInitialized() && !a.crypto.IsLocked() {
		decryptedData, err := a.crypto.DecryptData(localRec.EncryptedData)
		if err != nil {
			return nil, fmt.Errorf("ошибка расшифровки данных: %w", err)
		}
		localRec.EncryptedData = decryptedData
	}

	return localRec, nil
}

// ListRecords возвращает список записей
func (a *App) ListRecords(ctx context.Context, filter *RecordFilter) ([]*Record, error) {
	// Сначала получаем локальные записи
	records, err := a.storage.ListRecords(filter)
	if err != nil {
		return nil, fmt.Errorf("ошибка получения локальных записей: %w", err)
	}

	// Если аутентифицированы, синхронизируем
	if a.IsAuthenticated() {
		// Фоновая синхронизация
		go func() {
			if err := a.sync.Sync(ctx); err != nil {
				a.log.Warn("Ошибка синхронизации", "error", err)
			}
		}()

		// Если локально нет записей, пробуем получить с сервера
		if len(records) == 0 {
			serverRecords, err := a.httpClient.ListRecords(ctx, record.ListRequest{
				Type:        filter.Type,
				ShowDeleted: filter.ShowDeleted,
				Limit:       filter.Limit,
				Offset:      filter.Offset,
			})
			if err == nil && len(serverRecords) > 0 {
				// Сохраняем локально
				for _, rec := range serverRecords {
					if err := a.storage.SaveRecord(rec); err != nil {
						a.log.Warn("Не удалось сохранить запись локально", "error", err, "record_id", rec.ID)
					}
				}
				return serverRecords, nil
			}
		}
	}

	return records, nil
}

// UpdateRecord обновляет запись
func (a *App) UpdateRecord(ctx context.Context, id string, req record.UpdateRequest) error {
	// Получаем существующую запись
	existingRec, err := a.storage.GetRecord(id)
	if err != nil {
		return fmt.Errorf("запись не найдена: %w", err)
	}

	// Обновляем поля
	if req.Metadata != nil {
		existingRec.Metadata = *req.Metadata
	}

	if req.Data != nil {
		// Шифруем новые данные
		encryptedData, err := a.crypto.EncryptData(req.Data)
		if err != nil {
			return fmt.Errorf("ошибка шифрования данных: %w", err)
		}
		existingRec.Data = encryptedData
	}

	existingRec.UpdatedAt = time.Now()
	existingRec.Version++
	existingRec.Synced = false

	// Сохраняем локально
	if err := a.storage.UpdateRecord(existingRec); err != nil {
		return fmt.Errorf("ошибка обновления записи: %w", err)
	}

	// Синхронизируем с сервером
	if a.IsAuthenticated() {
		if err := a.httpClient.UpdateRecord(ctx, id, req); err != nil {
			a.log.Warn("Не удалось синхронизировать обновление с сервером", "error", err, "record_id", id)
		} else {
			existingRec.Synced = true
			if err := a.storage.UpdateRecord(existingRec); err != nil {
				a.log.Warn("Не удалось обновить статус синхронизации", "error", err)
			}
		}
	}

	return nil
}

// DeleteRecord удаляет запись
func (a *App) DeleteRecord(ctx context.Context, id string, permanent bool) error {
	// Помечаем как удаленную локально
	rec, err := a.storage.GetRecord(id)
	if err != nil {
		return fmt.Errorf("запись не найдена: %w", err)
	}

	if permanent {
		// Полное удаление
		if err := a.storage.DeleteRecord(id); err != nil {
			return fmt.Errorf("ошибка удаления записи: %w", err)
		}
		a.state.RecordsCount--
	} else {
		// Мягкое удаление
		rec.Deleted = true
		rec.UpdatedAt = time.Now()
		rec.Synced = false

		if err := a.storage.UpdateRecord(rec); err != nil {
			return fmt.Errorf("ошибка обновления записи: %w", err)
		}
	}

	// Синхронизируем с сервером
	if a.IsAuthenticated() {
		if permanent {
			if err := a.httpClient.DeleteRecord(ctx, id); err != nil {
				a.log.Warn("Не удалось синхронизировать удаление с сервером", "error", err, "record_id", id)
			}
		} else {
			// Для мягкого удаления отправляем обновление
			req := record.UpdateRequest{
				Deleted: &rec.Deleted,
			}
			if err := a.httpClient.UpdateRecord(ctx, id, req); err != nil {
				a.log.Warn("Не удалось синхронизировать удаление с сервером", "error", err, "record_id", id)
			} else {
				rec.Synced = true
				if err := a.storage.UpdateRecord(rec); err != nil {
					a.log.Warn("Не удалось обновить статус синхронизации", "error", err)
				}
			}
		}
	}

	if err := a.saveAppState(); err != nil {
		a.log.Warn("Не удалось сохранить состояние", "error", err)
	}

	return nil
}

// Sync запускает синхронизацию
func (a *App) Sync(ctx context.Context) error {
	return a.sync.Sync(ctx)
}

// Вспомогательные функции
func generateID() string {
	return fmt.Sprintf("%x", time.Now().UnixNano())
}
