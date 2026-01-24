package client

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/signal"
	gosync "sync"
	"syscall"
	"time"

	"golang.org/x/exp/slog"

	"gophkeeper/internal/app/client/config"
	"gophkeeper/internal/app/client/crypto"
	"gophkeeper/internal/domain/record"
	"gophkeeper/internal/domain/sync"
	"gophkeeper/internal/domain/user"
)

type App struct {
	config         *config.Config
	log            *slog.Logger
	crypto         *crypto.MasterKeyManager
	encryptor      *crypto.RecordEncryptor
	httpClient     *httpClient
	storage        Storage
	syncService    *SyncService
	state          *AppState
	masterKeyReady bool
	authenticated  bool
	wg             gosync.WaitGroup
	cancel         context.CancelFunc
	mu             gosync.RWMutex
}

// AppState хранит состояние приложения
type AppState struct {
	Initialized   bool      `json:"initialized"`
	UserLogin     string    `json:"user_login"`
	LastSync      time.Time `json:"last_sync"`
	RecordsCount  int       `json:"records_count"`
	MasterKeyHash string    `json:"master_key_hash"`
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
	var storage Storage
	sqliteStorage, err := NewSQLiteStorage(cfg.DataPath)
	if err != nil {
		log.Warn("Не удалось инициализировать SQLite, используем память", "error", err)
		storage = NewMemoryStorage()
	} else {
		storage = sqliteStorage
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
	app.syncService = NewSyncService(app)

	// Загружаем токен если он есть
	if token, err := app.GetToken(); err == nil && token != "" {
		httpCl.SetToken(token)
		app.mu.Lock()
		app.authenticated = true
		app.mu.Unlock()
		log.Debug("Токен загружен из файла")
	}

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

	go a.handleSignals()

	a.wg.Add(1)
	go func() {
		defer a.wg.Done()
		a.startSync(ctx)
	}()

	a.log.Info("Клиент запущен",
		"server", a.config.ServerAddress,
		"env", a.config.Env,
	)

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
	if err := a.crypto.GenerateMasterKey(password); err != nil {
		return fmt.Errorf("ошибка генерации мастер-ключа: %w", err)
	}

	// Сохраняем хэш мастер-ключа для проверки в будущем
	keyHash, err := a.crypto.GetKeyHash()
	if err != nil {
		return fmt.Errorf("ошибка получения хэша ключа: %w", err)
	}

	a.mu.Lock()
	a.state.MasterKeyHash = keyHash
	a.masterKeyReady = true
	a.state.Initialized = true

	if err := a.saveAppState(); err != nil {
		a.mu.Unlock()
		return fmt.Errorf("ошибка сохранения состояния: %w", err)
	}
	a.mu.Unlock()

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
	// Проверяем, что таблицы созданы
	count, err := a.storage.CountRecords()
	if err != nil {
		return fmt.Errorf("ошибка инициализации хранилища: %w", err)
	}
	a.state.RecordsCount = count

	return nil
}

// UnlockMasterKey разблокирует мастер-ключ
func (a *App) UnlockMasterKey(password string) error {
	if err := a.crypto.UnlockMasterKey(password); err != nil {
		return fmt.Errorf("неверный мастер-пароль: %w", err)
	}

	a.mu.Lock()
	a.masterKeyReady = true
	a.mu.Unlock()

	return nil
}

// IsMasterKeyUnlocked проверяет, разблокирован ли мастер-ключ
func (a *App) IsMasterKeyUnlocked() bool {
	return !a.crypto.IsLocked()
}

// LockMasterKey блокирует мастер-ключ
func (a *App) LockMasterKey() {
	a.mu.Lock()
	defer a.mu.Unlock()

	a.crypto.Lock()
	a.mu.Lock()
	a.masterKeyReady = false
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
			if _, err := a.syncService.Sync(ctx); err != nil {
				a.log.Error("Ошибка синхронизации", "error", err)
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

// IsAuthenticated проверяет, аутентифицирован ли пользователь
func (a *App) IsAuthenticated() bool {
	a.mu.Lock()
	defer a.mu.Unlock()

	if !a.authenticated {
		token, err := a.GetToken()
		if err == nil && token != "" {
			a.authenticated = true
		}
	}

	return a.authenticated
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

// SaveToken сохраняет токен аутентификации
func (a *App) SaveToken(token string) error {
	if err := os.WriteFile(a.config.TokenPath, []byte(token), 0600); err != nil {
		return fmt.Errorf("ошибка сохранения токена: %w", err)
	}

	a.httpClient.SetToken(token)

	return nil
}

// ClearToken удаляет токен
func (a *App) ClearToken() error {
	a.mu.Lock()
	a.authenticated = false
	a.state.UserLogin = ""

	if err := os.Remove(a.config.TokenPath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("ошибка удаления токена: %w", err)
	}

	if err := a.saveAppState(); err != nil {
		a.mu.Unlock()
		return fmt.Errorf("ошибка сохранения состояния: %w", err)
	}
	a.mu.Unlock()

	return nil
}

// Register регистрирует нового пользователя
func (a *App) Register(ctx context.Context, req user.BaseRequest) error {
	if err := a.httpClient.Register(ctx, req.Login, req.Password); err != nil {
		return err
	}

	a.log.Info("Пользователь успешно зарегистрирован", "login", req.Login)
	return nil
}

// Login выполняет вход пользователя
func (a *App) Login(ctx context.Context, req user.BaseRequest) (string, error) {
	token, err := a.httpClient.Login(ctx, req.Login, req.Password)
	if err != nil {
		return "", err
	}

	if err = a.SaveToken(token); err != nil {
		return "", fmt.Errorf("ошибка сохранения токена: %w", err)
	}

	a.mu.Lock()
	a.authenticated = true
	a.state.UserLogin = req.Login

	if err = a.saveAppState(); err != nil {
		a.mu.Unlock()
		a.log.Warn("Не удалось сохранить состояние", "error", err)
	}
	a.mu.Unlock()

	a.log.Info("Вход выполнен успешно", "login", req.Login)
	return token, nil
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
		decryptedData, err := a.crypto.DecryptData([]byte(rec.EncryptedData))
		if err != nil {
			return fmt.Errorf("ошибка расшифровки данных записи %d: %w", rec.ID, err)
		}

		// Шифруем новым ключом
		encryptedData, err := a.crypto.EncryptDataWithPassword(decryptedData, newMasterPassword)
		if err != nil {
			return fmt.Errorf("ошибка шифровки данных записи %d: %w", rec.ID, err)
		}

		// Обновляем запись
		rec.EncryptedData = string(encryptedData)
		if err := a.storage.UpdateRecord(rec); err != nil {
			return fmt.Errorf("ошибка обновления записи %d: %w", rec.ID, err)
		}
	}

	return nil
}

// ==================== Record Operations ====================

// CreateLoginRecord создает запись логина с шифрованием
func (a *App) CreateLoginRecord(ctx context.Context, req CreateLoginRequest) (int, error) {
	if !a.IsAuthenticated() {
		return 0, fmt.Errorf("требуется аутентификация. Выполните: gophkeeper auth login")
	}

	if !a.IsMasterKeyUnlocked() {
		return 0, fmt.Errorf("мастер-ключ заблокирован. Выполните: gophkeeper unlock")
	}

	// Подготавливаем метаданные (не шифруем для поиска)
	meta := map[string]interface{}{
		"title":    req.Title,
		"resource": req.Resource,
		"category": req.Category,
		"tags":     req.Tags,
	}
	metaJSON, _ := json.Marshal(meta)

	// Подготавливаем зашифрованную запись
	encryptedReq, err := a.prepareEncryptedRecord(record.RecTypeLogin, req, metaJSON)
	if err != nil {
		return 0, fmt.Errorf("ошибка подготовки зашифрованной записи: %w", err)
	}

	// Отправляем на сервер через generic API
	serverID, err := a.httpClient.CreateRecord(ctx, encryptedReq)
	if err != nil {
		a.log.Warn("Не удалось создать запись на сервере, сохраняем локально", "error", err)
		return a.saveLocalRecord(record.RecTypeLogin, req)
	}

	// Сохраняем локально
	localRec := &LocalRecord{
		ServerID:      serverID,
		Type:          record.RecTypeLogin,
		EncryptedData: encryptedReq.Data,
		Meta:          metaJSON,
		Version:       1,
		LastModified:  time.Now(),
		CreatedAt:     time.Now(),
		Synced:        true,
		DeviceID:      req.DeviceID,
	}

	if err := a.storage.SaveRecord(localRec); err != nil {
		a.log.Warn("Не удалось сохранить запись локально", "error", err)
	}

	a.mu.Lock()
	a.state.RecordsCount++
	if err = a.saveAppState(); err != nil {
		a.mu.Unlock()
		return 0, fmt.Errorf("ошибка сохранения состояния: %w", err)
	}
	a.mu.Unlock()

	return serverID, nil
}

// CreateTextRecord создает текстовую запись с шифрованием
func (a *App) CreateTextRecord(ctx context.Context, req CreateTextRequest) (int, error) {
	if !a.IsAuthenticated() {
		return 0, fmt.Errorf("требуется аутентификация. Выполните: gophkeeper auth login")
	}

	if !a.IsMasterKeyUnlocked() {
		return 0, fmt.Errorf("мастер-ключ заблокирован. Выполните: gophkeeper unlock")
	}

	// Подготавливаем метаданные
	meta := map[string]interface{}{
		"title":    req.Title,
		"category": req.Category,
		"tags":     req.Tags,
		"format":   req.Format,
	}
	metaJSON, _ := json.Marshal(meta)

	// Подготавливаем зашифрованную запись
	encryptedReq, err := a.prepareEncryptedRecord(record.RecTypeText, req, metaJSON)
	if err != nil {
		return 0, fmt.Errorf("ошибка подготовки зашифрованной записи: %w", err)
	}

	// Отправляем на сервер
	serverID, err := a.httpClient.CreateRecord(ctx, encryptedReq)
	if err != nil {
		a.log.Warn("Не удалось создать запись на сервере, сохраняем локально", "error", err)
		return a.saveLocalRecord(record.RecTypeText, req)
	}

	// Сохраняем локально
	localRec := &LocalRecord{
		ServerID:      serverID,
		Type:          record.RecTypeText,
		EncryptedData: encryptedReq.Data,
		Meta:          metaJSON,
		Version:       1,
		LastModified:  time.Now(),
		CreatedAt:     time.Now(),
		Synced:        true,
		DeviceID:      req.DeviceID,
	}

	if err := a.storage.SaveRecord(localRec); err != nil {
		a.log.Warn("Не удалось сохранить запись локально", "error", err)
	}

	a.mu.Lock()
	a.state.RecordsCount++
	if err = a.saveAppState(); err != nil {
		a.mu.Unlock()
		return 0, fmt.Errorf("ошибка сохранения состояния: %w", err)
	}
	a.mu.Unlock()

	return serverID, nil
}

// CreateCardRecord создает запись карты с шифрованием
func (a *App) CreateCardRecord(ctx context.Context, req CreateCardRequest) (int, error) {
	if !a.IsAuthenticated() {
		return 0, fmt.Errorf("требуется аутентификация. Выполните: gophkeeper auth login")
	}

	if !a.IsMasterKeyUnlocked() {
		return 0, fmt.Errorf("мастер-ключ заблокирован. Выполните: gophkeeper unlock")
	}

	// Подготавливаем метаданные
	meta := map[string]interface{}{
		"title":     req.Title,
		"bank_name": req.BankName,
		"category":  req.Category,
		"tags":      req.Tags,
	}
	metaJSON, _ := json.Marshal(meta)

	// Подготавливаем зашифрованную запись
	encryptedReq, err := a.prepareEncryptedRecord(record.RecTypeCard, req, metaJSON)
	if err != nil {
		return 0, fmt.Errorf("ошибка подготовки зашифрованной записи: %w", err)
	}

	// Отправляем на сервер
	serverID, err := a.httpClient.CreateRecord(ctx, encryptedReq)
	if err != nil {
		a.log.Warn("Не удалось создать запись на сервере, сохраняем локально", "error", err)
		return a.saveLocalRecord(record.RecTypeCard, req)
	}

	// Сохраняем локально
	localRec := &LocalRecord{
		ServerID:      serverID,
		Type:          record.RecTypeCard,
		EncryptedData: encryptedReq.Data,
		Meta:          metaJSON,
		Version:       1,
		LastModified:  time.Now(),
		CreatedAt:     time.Now(),
		Synced:        true,
		DeviceID:      req.DeviceID,
	}

	if err := a.storage.SaveRecord(localRec); err != nil {
		a.log.Warn("Не удалось сохранить запись локально", "error", err)
	}

	a.mu.Lock()
	a.state.RecordsCount++
	if err = a.saveAppState(); err != nil {
		a.mu.Unlock()
		return 0, fmt.Errorf("ошибка сохранения состояния: %w", err)
	}
	a.mu.Unlock()

	return serverID, nil
}

// CreateBinaryRecord создает бинарную запись с шифрованием
func (a *App) CreateBinaryRecord(ctx context.Context, req CreateBinaryRequest) (int, error) {
	if !a.IsAuthenticated() {
		return 0, fmt.Errorf("требуется аутентификация. Выполните: gophkeeper auth login")
	}

	if !a.IsMasterKeyUnlocked() {
		return 0, fmt.Errorf("мастер-ключ заблокирован. Выполните: gophkeeper unlock")
	}

	// Подготавливаем метаданные
	meta := map[string]interface{}{
		"title":       req.Title,
		"filename":    req.Filename,
		"category":    req.Category,
		"tags":        req.Tags,
		"description": req.Description,
	}
	metaJSON, _ := json.Marshal(meta)

	// Подготавливаем зашифрованную запись
	encryptedReq, err := a.prepareEncryptedRecord(record.RecTypeBinary, req, metaJSON)
	if err != nil {
		return 0, fmt.Errorf("ошибка подготовки зашифрованной записи: %w", err)
	}

	// Отправляем на сервер
	serverID, err := a.httpClient.CreateRecord(ctx, encryptedReq)
	if err != nil {
		a.log.Warn("Не удалось создать запись на сервере, сохраняем локально", "error", err)
		return a.saveLocalRecord(record.RecTypeBinary, req)
	}

	// Сохраняем локально
	localRec := &LocalRecord{
		ServerID:      serverID,
		Type:          record.RecTypeBinary,
		EncryptedData: encryptedReq.Data,
		Meta:          metaJSON,
		Version:       1,
		LastModified:  time.Now(),
		CreatedAt:     time.Now(),
		Synced:        true,
		DeviceID:      req.DeviceID,
	}

	if err := a.storage.SaveRecord(localRec); err != nil {
		a.log.Warn("Не удалось сохранить запись локально", "error", err)
	}

	a.mu.Lock()
	a.state.RecordsCount++
	if err = a.saveAppState(); err != nil {
		a.mu.Unlock()
		return 0, fmt.Errorf("ошибка сохранения состояния: %w", err)
	}
	a.mu.Unlock()

	return serverID, nil
}

// saveLocalRecord сохраняет запись локально без синхронизации
func (a *App) saveLocalRecord(recType record.RecType, data interface{}) (int, error) {
	dataJSON, err := json.Marshal(data)
	if err != nil {
		return 0, fmt.Errorf("ошибка сериализации данных: %w", err)
	}

	// Шифруем данные если мастер-ключ готов
	var encryptedData string
	if a.masterKeyReady {
		encrypted, err := a.encryptor.EncryptRecord(dataJSON)
		if err != nil {
			return 0, fmt.Errorf("ошибка шифрования данных: %w", err)
		}
		encryptedData = string(encrypted)
	} else {
		encryptedData = string(dataJSON)
	}

	localRec := &LocalRecord{
		Type:          recType,
		EncryptedData: encryptedData,
		Version:       1,
		LastModified:  time.Now(),
		CreatedAt:     time.Now(),
		Synced:        false,
	}

	if err := a.storage.SaveRecord(localRec); err != nil {
		return 0, fmt.Errorf("ошибка сохранения записи: %w", err)
	}

	a.mu.Lock()
	a.state.RecordsCount++
	if err = a.saveAppState(); err != nil {
		a.mu.Unlock()
		return 0, fmt.Errorf("ошибка сохранения состояния: %w", err)
	}
	a.mu.Unlock()

	return localRec.ID, nil
}

// GetRecord возвращает запись по ID с расшифровкой
func (a *App) GetRecord(ctx context.Context, id int) (*LocalRecord, error) {
	localRec, err := a.storage.GetRecord(id)
	if err != nil {
		// Если нет локально, пробуем получить с сервера
		if a.IsAuthenticated() {
			serverRec, err := a.httpClient.GetRecord(ctx, id)
			if err != nil {
				return nil, fmt.Errorf("запись не найдена: %w", err)
			}

			localRec = FromServerRecord(serverRec)
			if err := a.storage.SaveRecord(localRec); err != nil {
				a.log.Warn("Не удалось сохранить запись локально", "error", err)
			}

			return localRec, nil
		}
		return nil, fmt.Errorf("запись не найдена: %w", err)
	}

	return localRec, nil
}

// GetDecryptedRecord возвращает расшифрованную запись
func (a *App) GetDecryptedRecord(ctx context.Context, id int) (interface{}, error) {
	localRec, err := a.GetRecord(ctx, id)
	if err != nil {
		return nil, err
	}

	var decryptedData interface{}
	if err := a.decryptRecordData(localRec.EncryptedData, &decryptedData); err != nil {
		return nil, fmt.Errorf("ошибка расшифровки данных: %w", err)
	}

	return decryptedData, nil
}

// ListRecords возвращает список записей
func (a *App) ListRecords(ctx context.Context, filter *RecordFilter) ([]*LocalRecord, error) {
	records, err := a.storage.ListRecords(filter)
	if err != nil {
		return nil, fmt.Errorf("ошибка получения локальных записей: %w", err)
	}

	if a.IsAuthenticated() {
		go func() {
			if _, err := a.syncService.Sync(ctx); err != nil {
				a.log.Warn("Ошибка синхронизации", "error", err)
			}
		}()

		if len(records) == 0 {
			serverRecords, err := a.httpClient.ListRecords(ctx)
			if err == nil && len(serverRecords.Records) > 0 {
				for _, item := range serverRecords.Records {
					// Получаем полную запись с сервера
					serverRec, err := a.httpClient.GetRecord(ctx, item.ID)
					if err != nil {
						a.log.Warn("Не удалось получить запись с сервера", "error", err, "record_id", item.ID)
						continue
					}
					localRec := FromServerRecord(serverRec)
					if err := a.storage.SaveRecord(localRec); err != nil {
						a.log.Warn("Не удалось сохранить запись локально", "error", err, "record_id", item.ID)
					}
					records = append(records, localRec)
				}
			}
		}
	}

	return records, nil
}

// UpdateRecord обновляет запись
func (a *App) UpdateRecord(ctx context.Context, id int, req GenericRecordRequest) error {
	// Получаем существующую запись
	existingRec, err := a.storage.GetRecord(id)
	if err != nil {
		return fmt.Errorf("запись не найдена: %w", err)
	}

	// Обновляем поля
	existingRec.Type = req.Type
	existingRec.Meta = req.Meta
	existingRec.EncryptedData = req.Data
	existingRec.LastModified = time.Now()
	existingRec.Version++
	existingRec.Synced = false

	// Сохраняем локально
	if err := a.storage.UpdateRecord(existingRec); err != nil {
		return fmt.Errorf("ошибка обновления записи: %w", err)
	}

	// Синхронизируем с сервером
	if a.IsAuthenticated() && existingRec.ServerID > 0 {
		if err := a.httpClient.UpdateRecord(ctx, existingRec.ServerID, req); err != nil {
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
func (a *App) DeleteRecord(ctx context.Context, id int, permanent bool) error {
	// Получаем запись
	rec, err := a.storage.GetRecord(id)
	if err != nil {
		return fmt.Errorf("запись не найдена: %w", err)
	}

	if permanent {
		if err := a.storage.HardDeleteRecord(id); err != nil {
			return fmt.Errorf("ошибка удаления записи: %w", err)
		}
		a.state.RecordsCount--
	} else {
		if err := a.storage.DeleteRecord(id); err != nil {
			return fmt.Errorf("ошибка удаления записи: %w", err)
		}
	}

	if a.IsAuthenticated() && rec.ServerID > 0 {
		if err := a.httpClient.DeleteRecord(ctx, rec.ServerID); err != nil {
			a.log.Warn("Не удалось синхронизировать удаление с сервером", "error", err, "record_id", id)
		}
	}

	a.mu.Lock()
	if err := a.saveAppState(); err != nil {
		a.mu.Unlock()
		a.log.Warn("Не удалось сохранить состояние", "error", err)
	}
	a.mu.Unlock()

	return nil
}

// Sync запускает синхронизацию
func (a *App) Sync(ctx context.Context) (*SyncResult, error) {
	return a.syncService.Sync(ctx)
}

// GetSyncService возвращает сервис синхронизации
func (a *App) GetSyncService() *SyncService {
	return a.syncService
}

// GetSyncStatus получает статус синхронизации
func (a *App) GetSyncStatus(ctx context.Context) (*sync.SyncStatus, error) {
	return a.httpClient.GetSyncStatus(ctx)
}

// GetSyncConflicts получает конфликты синхронизации
func (a *App) GetSyncConflicts(ctx context.Context) ([]sync.Conflict, error) {
	return a.httpClient.GetSyncConflicts(ctx)
}

// ResolveConflict разрешает конфликт
func (a *App) ResolveConflict(ctx context.Context, conflictID int, req sync.ResolveConflictRequest) error {
	return a.httpClient.ResolveConflict(ctx, conflictID, req)
}

// GetDevices получает список устройств
func (a *App) GetDevices(ctx context.Context) ([]sync.DeviceInfo, error) {
	return a.httpClient.GetDevices(ctx)
}

// RemoveDevice удаляет устройство
func (a *App) RemoveDevice(ctx context.Context, deviceID int) error {
	return a.httpClient.RemoveDevice(ctx, deviceID)
}
