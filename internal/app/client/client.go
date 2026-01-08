package client

import (
	"context"
	"fmt"
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
	config     *config.Config
	log        *slog.Logger
	crypto     *crypto.MasterKeyManager
	httpClient HTTPClient
	storage    Storage
	sync       *SyncService
	wg         sync.WaitGroup
	cancel     context.CancelFunc
}

type Storage interface {
	// Методы для локального хранения данных
	SaveRecord(record *record.Record) error
	GetRecord(id string) (*record.Record, error)
	ListRecords() ([]*record.Record, error)
	DeleteRecord(id string) error
}

type HTTPClient interface {
	// Методы для HTTP взаимодействия с сервером
	Login(ctx context.Context, login, password string) (string, error)
	Register(ctx context.Context, login, password string) error
	CreateRecord(ctx context.Context, record *record.Record) error
	GetRecords(ctx context.Context) ([]*record.Record, error)
	SyncRecords(ctx context.Context, records []*record.Record) error
}

func New(cfg *config.Config, log *slog.Logger) (*App, error) {
	// Инициализируем менеджер мастер-ключа
	masterKey, err := crypto.NewMasterKeyManager(cfg.MasterKeyPath)
	if err != nil {
		return nil, fmt.Errorf("ошибка инициализации мастер-ключа: %w", err)
	}

	// Инициализируем HTTP клиент
	httpClient, err := NewHTTPClient(cfg, log)
	if err != nil {
		return nil, fmt.Errorf("ошибка инициализации HTTP клиента: %w", err)
	}

	// Инициализируем локальное хранилище (пока in-memory, позже добавим SQLite)
	storage := NewMemoryStorage()

	app := &App{
		config:     cfg,
		log:        log,
		crypto:     masterKey,
		httpClient: httpClient,
		storage:    storage,
	}

	// Инициализируем сервис синхронизации
	app.sync = NewSyncService(app)

	return app, nil
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

// IsInitialized проверяет, инициализирован ли мастер-ключ
func (a *App) IsInitialized() bool {
	return a.crypto.IsInitialized()
}

// InitMasterKey инициализирует мастер-ключ с помощью пароля
func (a *App) InitMasterKey(password string) error {
	return a.crypto.Init(password)
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

func (a *App) SaveToken(token string) error {
	if err := os.WriteFile(a.config.TokenPath, []byte(token), 0600); err != nil {
		return fmt.Errorf("ошибка сохранения токена: %w", err)
	}
	return nil
}

func (a *App) ClearToken() error {
	if err := os.Remove(a.config.TokenPath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("ошибка удаления токена: %w", err)
	}
	return nil
}

func (a *App) Register(ctx context.Context, request user.BaseRequest) error {
	return a.httpClient.Register(ctx, request.Login, request.Password)
}
