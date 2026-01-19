# GophKeeper Client - Техническая документация

## Обзор изменений

Данный документ описывает доработки клиентской части проекта GophKeeper для полноценной работы с сервером.

## Исправленные проблемы

### 1. Несоответствие путей API эндпоинтов

**Проблема**: Клиент использовал пути `/api/v1/auth/*`, в то время как сервер использует `/user/*`

**Решение**:
- Изменены пути в [`internal/app/client/http_client.go`](../internal/app/client/http_client.go):
  - `Login()`: `/api/v1/auth/login` → `/user/login`
  - `Register()`: `/api/v1/auth/register` → `/user/register`

**Файлы**: 
- [`internal/app/client/http_client.go:166`](../internal/app/client/http_client.go:166)
- [`internal/app/client/http_client.go:196`](../internal/app/client/http_client.go:196)

### 2. Отсутствие эндпоинта смены пароля

**Проблема**: Клиент реализовывал функцию `ChangePassword()`, но на сервере отсутствует соответствующий эндпоинт

**Решение**:
- Закомментирован метод `ChangePassword()` в HTTP клиенте
- Закомментирован метод `App.ChangePassword()` в основном приложении
- CLI команда `change-password` заменена на заглушку с информативным сообщением

**Файлы**:
- [`internal/app/client/http_client.go:220-227`](../internal/app/client/http_client.go:220)
- [`internal/app/client/client.go:370-397`](../internal/app/client/client.go:370)
- [`cmd/client/cmd/auth/change-password.go`](../cmd/client/cmd/auth/change-password.go)

**TODO**: Реализовать эндпоинт `/user/change-password` на сервере

### 3. Доступ к сервису синхронизации

**Проблема**: CLI команды не могли получить доступ к приватному полю `syncService`

**Решение**:
- Добавлен публичный метод `GetSyncService()` в [`internal/app/client/client.go:783`](../internal/app/client/client.go:783)
- Изменена сигнатура метода `Sync()` для возврата результата синхронизации

**Файлы**:
- [`internal/app/client/client.go:778-784`](../internal/app/client/client.go:778)

### 4. Ошибки компиляции в CLI командах

**Проблема**: Множественные ошибки типов и несоответствия интерфейсов

**Решение**:

#### cmd/client/cmd/sync/sync.go
- Заменены обращения к `app.sync` на `app.GetSyncService()`
- Исправлены поля статистики (`TotalUploaded` → `TotalUploads`, `LastSuccessful` → `LastSync`)
- Упрощен вывод конфигурации

#### cmd/client/cmd/auth/login.go
- Исправлен вызов `app.Sync()` для обработки двух возвращаемых значений

#### cmd/client/cmd/record/create.go
- Полностью переписан для использования типизированных методов:
  - `CreateLoginRecord()`
  - `CreateTextRecord()`
  - `CreateCardRecord()`
  - `CreateBinaryRecord()`
- Исправлен импорт `crypto/rand` для генерации паролей
- Упрощена логика создания записей

#### cmd/client/cmd/record/get.go
- Адаптирован для работы с `*client.LocalRecord` вместо `*record.Record`
- Изменен тип параметра ID с `string` на `int`
- Упрощен вывод метаданных

#### cmd/client/cmd/record/list.go
- Адаптирован для работы с `*client.LocalRecord`
- Исправлен фильтр записей (использование `client.RecordFilter`)
- Экспортирована команда `ListCmd`

**Файлы**:
- [`cmd/client/cmd/sync/sync.go`](../cmd/client/cmd/sync/sync.go)
- [`cmd/client/cmd/auth/login.go`](../cmd/client/cmd/auth/login.go)
- [`cmd/client/cmd/record/create.go`](../cmd/client/cmd/record/create.go)
- [`cmd/client/cmd/record/get.go`](../cmd/client/cmd/record/get.go)
- [`cmd/client/cmd/record/list.go`](../cmd/client/cmd/record/list.go)

### 5. Retry логика для HTTP запросов

**Проблема**: Отсутствовала обработка временных сетевых ошибок

**Решение**:
- Добавлена функция `doRequestWithRetry()` с экспоненциальной задержкой
- Реализована логика повторных попыток для серверных ошибок (5xx)
- Клиентские ошибки (4xx) не требуют retry
- Максимум 3 попытки с задержкой от 1 до 10 секунд

**Константы**:
```go
const (
    maxRetries     = 3
    retryDelay     = 1 * time.Second
    maxRetryDelay  = 10 * time.Second
)
```

**Файлы**:
- [`internal/app/client/http_client.go:89-168`](../internal/app/client/http_client.go:89)

## Архитектура клиента

### Компоненты и их взаимодействие

```
┌─────────────────────────────────────────────────────────────┐
│                        CLI Commands                          │
│  (auth, record, sync)                                        │
└────────────────────────┬────────────────────────────────────┘
                         │
                         ▼
┌─────────────────────────────────────────────────────────────┐
│                      App (client.go)                         │
│  - Координация компонентов                                   │
│  - Управление состоянием                                     │
│  - Публичные методы для CLI                                  │
└──┬──────────────┬──────────────┬──────────────┬─────────────┘
   │              │              │              │
   ▼              ▼              ▼              ▼
┌──────────┐ ┌──────────┐ ┌──────────┐ ┌──────────────┐
│  HTTP    │ │  Crypto  │ │ Storage  │ │ SyncService  │
│  Client  │ │          │ │ (SQLite) │ │              │
└──────────┘ └──────────┘ └──────────┘ └──────────────┘
     │            │            │              │
     │            │            │              │
     ▼            ▼            ▼              ▼
  Сервер    Шифрование   Локальная БД   Синхронизация
```

### HTTP Client

**Ответственность**:
- Взаимодействие с REST API сервера
- Управление токенами аутентификации
- Retry логика для сетевых запросов
- Логирование запросов/ответов

**Методы**:
- Auth: `Login()`, `Register()`
- Records: `CreateRecord()`, `GetRecord()`, `ListRecords()`, `UpdateRecord()`, `DeleteRecord()`
- Typed: `CreateLoginRecord()`, `CreateTextRecord()`, `CreateCardRecord()`, `CreateBinaryRecord()`
- Sync: `GetSyncChanges()`, `SendBatchSync()`, `GetSyncStatus()`, `GetSyncConflicts()`, `ResolveConflict()`, `GetDevices()`, `RemoveDevice()`

### Crypto

**Ответственность**:
- Управление мастер-ключом
- Шифрование/расшифровка данных
- Генерация ключей из паролей

**Алгоритмы**:
- **AES-256-GCM** для шифрования данных
- **PBKDF2-SHA256** для генерации ключей (100,000 итераций)
- **SHA-256** для хеширования

**Методы**:
- `GenerateMasterKey(password)` - генерация нового мастер-ключа
- `UnlockMasterKey(password)` - разблокировка существующего ключа
- `EncryptData(plaintext)` - шифрование данных
- `DecryptData(ciphertext)` - расшифровка данных
- `IsLocked()` - проверка блокировки ключа

### Storage (SQLite)

**Ответственность**:
- Локальное хранение записей
- Отслеживание синхронизации
- Поддержка офлайн режима

**Схема таблицы records**:
```sql
CREATE TABLE records (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    server_id INTEGER DEFAULT 0,
    user_id INTEGER DEFAULT 0,
    type TEXT NOT NULL,
    encrypted_data TEXT,
    meta TEXT,
    version INTEGER NOT NULL DEFAULT 1,
    last_modified DATETIME NOT NULL,
    deleted_at DATETIME,
    checksum TEXT,
    device_id TEXT,
    synced BOOLEAN NOT NULL DEFAULT 0,
    sync_version INTEGER NOT NULL DEFAULT 0,
    created_at DATETIME NOT NULL
);
```

**Индексы**:
- `idx_records_type` - по типу записи
- `idx_records_deleted` - по статусу удаления
- `idx_records_synced` - по статусу синхронизации
- `idx_records_server_id` - по ID на сервере
- `idx_records_last_modified` - по времени изменения

**Методы**:
- `SaveRecord()` - сохранение/обновление записи
- `GetRecord(id)` - получение по локальному ID
- `GetRecordByServerID(serverID)` - получение по серверному ID
- `ListRecords(filter)` - список с фильтрацией
- `DeleteRecord(id)` - мягкое удаление
- `HardDeleteRecord(id)` - полное удаление
- `GetRecordsModifiedAfter(since, limit)` - для синхронизации
- `MarkAsSynced(id, serverID, syncVersion)` - пометка синхронизированной

### Sync Service

**Ответственность**:
- Двусторонняя синхронизация данных
- Обнаружение и разрешение конфликтов
- Пакетная обработка изменений
- Статистика синхронизации

**Алгоритм синхронизации**:

1. **Pre-sync checks**:
   - Проверка включенности синхронизации
   - Проверка аутентификации
   - Проверка соединения с сервером
   - Проверка разблокировки мастер-ключа

2. **Получение метаданных**:
   - Загрузка `last_sync_time` из локального хранилища
   - Определение `client_id` и `device_name`

3. **Получение изменений**:
   - Локальные: записи с `synced=false` или измененные после `last_sync_time`
   - Серверные: запрос к `/api/sync/changes` с `last_sync_time`

4. **Обнаружение конфликтов**:
   - Сравнение версий записей
   - Проверка времени изменения
   - Определение типа конфликта (edit-edit, edit-delete, delete-edit)

5. **Разрешение конфликтов**:
   - Автоматическое по стратегии (`client`, `server`, `newer`)
   - Или ручное (в будущих версиях)

6. **Загрузка на сервер**:
   - Пакетная отправка через `/api/sync/batch`
   - Обработка ошибок
   - Пометка записей как синхронизированных

7. **Применение серверных изменений**:
   - Создание новых записей
   - Обновление существующих
   - Обработка удалений

8. **Обновление метаданных**:
   - Сохранение `last_sync_time`
   - Инкремент `sync_version`
   - Обновление статистики

**Конфигурация** (`~/.gophkeeper/sync_config.json`):
```json
{
  "enabled": true,
  "interval": "30s",
  "batch_size": 50,
  "max_retries": 3,
  "retry_delay": "5s",
  "conflict_strategy": "newer",
  "auto_resolve": true
}
```

**Методы**:
- `Sync(ctx)` - основной метод синхронизации
- `GetStats()` - получение статистики
- `ResetStats()` - сброс статистики
- `GetLastSyncTime()` - время последней синхронизации
- `IsSyncing()` - проверка активной синхронизации

## Модели данных

### LocalRecord

Локальная модель записи, расширяющая серверную модель:

```go
type LocalRecord struct {
    ID            int             // Локальный ID
    ServerID      int             // ID на сервере
    UserID        int             // ID пользователя
    Type          record.RecType  // Тип записи
    EncryptedData string          // Зашифрованные данные
    Meta          json.RawMessage // Метаданные (JSON)
    Version       int             // Версия записи
    LastModified  time.Time       // Время изменения
    DeletedAt     *time.Time      // Время удаления (soft delete)
    Checksum      string          // Контрольная сумма
    DeviceID      string          // ID устройства
    
    // Поля для синхронизации
    Synced      bool      // Синхронизирована ли запись
    SyncVersion int64     // Версия синхронизации
    CreatedAt   time.Time // Время создания
}
```

### Типизированные запросы

Клиент поддерживает типизированные запросы для каждого типа записи:

#### CreateLoginRequest
```go
type CreateLoginRequest struct {
    Username  string   // Имя пользователя
    Password  string   // Пароль
    Notes     string   // Заметки
    Title     string   // Название записи
    Resource  string   // URL или название ресурса
    Category  string   // Категория
    Tags      []string // Теги
    TwoFA     bool     // Включена ли 2FA
    TwoFAType string   // Тип 2FA
    DeviceID  string   // ID устройства
}
```

#### CreateTextRequest
```go
type CreateTextRequest struct {
    Content     string   // Текстовое содержимое
    Title       string   // Название записи
    Category    string   // Категория
    Tags        []string // Теги
    Format      string   // Формат (plain, markdown, html, json, xml, yaml)
    Language    string   // Язык
    IsSensitive bool     // Содержит чувствительные данные
    DeviceID    string   // ID устройства
}
```

#### CreateCardRequest
```go
type CreateCardRequest struct {
    CardNumber     string   // Номер карты
    CardHolder     string   // Держатель карты
    ExpiryMonth    string   // Месяц истечения (01-12)
    ExpiryYear     string   // Год истечения (20XX)
    CVV            string   // CVV код
    PIN            string   // PIN код
    BillingAddress string   // Адрес для счетов
    Title          string   // Название записи
    BankName       string   // Название банка
    PaymentSystem  string   // Платежная система
    Category       string   // Категория
    Tags           []string // Теги
    Notes          string   // Заметки
    IsVirtual      bool     // Виртуальная карта
    IsActive       bool     // Карта активна
    DailyLimit     *float64 // Дневной лимит
    PhoneNumber    string   // Привязанный телефон
    DeviceID       string   // ID устройства
}
```

#### CreateBinaryRequest
```go
type CreateBinaryRequest struct {
    Data        string   // Base64-encoded данные
    Filename    string   // Имя файла
    ContentType string   // MIME тип
    Title       string   // Название записи
    Category    string   // Категория
    Tags        []string // Теги
    Description string   // Описание файла
    DeviceID    string   // ID устройства
}
```

## Безопасность

### Шифрование данных

1. **Генерация мастер-ключа**:
   ```
   password → PBKDF2-SHA256 (100k iterations) → master_key (256 bit)
   ```

2. **Шифрование записи**:
   ```
   plaintext → AES-256-GCM (master_key, random nonce) → ciphertext
   ```

3. **Хранение мастер-ключа**:
   ```
   master_key → AES-256-GCM (self-encrypted) → file (~/.gophkeeper/.master.key)
   ```

### Безопасные практики

✅ **Реализовано**:
- Мастер-ключ никогда не покидает клиент
- Пароли не логируются
- Данные шифруются перед отправкой на сервер
- Токены хранятся с правами 0600
- Безопасная очистка ключей из памяти

⚠️ **Рекомендуется**:
- Использовать TLS для соединения с сервером
- Регулярно менять мастер-пароль
- Использовать сильные пароли (минимум 12 символов)

## Обработка ошибок

### Типы ошибок

1. **Сетевые ошибки** - автоматический retry (до 3 попыток)
2. **Ошибки аутентификации** (401) - требуется повторный вход
3. **Ошибки валидации** (422) - некорректные данные
4. **Серверные ошибки** (5xx) - retry с экспоненциальной задержкой
5. **Ошибки шифрования** - критические, требуют вмешательства пользователя

### Graceful Degradation

Клиент продолжает работать при недоступности сервера:
- Все операции выполняются локально
- Данные помечаются как несинхронизированные
- При восстановлении соединения автоматически синхронизируются

## Тестирование

### Unit тесты

```bash
# Тесты криптографии
go test ./internal/app/client/crypto/...

# Тесты синхронизации
go test ./internal/app/client/sync_test.go

# Все тесты клиента
go test ./internal/app/client/...
```

### Интеграционные тесты

```bash
# 1. Запустить сервер
cd cmd/server && go run main.go

# 2. В другом терминале - тесты клиента
cd cmd/client

# Регистрация
./gophkeeper auth register

# Вход
./gophkeeper auth login

# Создание записи
./gophkeeper record create --type password --name "Test" --username "test" --password "test123"

# Список записей
./gophkeeper record list

# Синхронизация
./gophkeeper sync

# Статус
./gophkeeper sync --status
```

## Производительность

### Оптимизации

1. **HTTP Connection Pooling**:
   - `MaxIdleConns: 100`
   - `IdleConnTimeout: 90s`
   - `MaxIdleConnsPerHost: 10`

2. **Пакетная синхронизация**:
   - До 50 записей за один запрос
   - Параллельная обработка (в будущих версиях)

3. **SQLite оптимизации**:
   - WAL режим для лучшей производительности
   - Индексы на часто используемых полях
   - Foreign keys для целостности данных

### Метрики

Статистика синхронизации включает:
- Общее количество синхронизаций
- Количество загруженных/скачанных записей
- Количество конфликтов и разрешений
- Среднее время синхронизации
- Количество ошибок

## Конфигурация

### Файлы конфигурации

Клиент использует следующие файлы в `~/.gophkeeper/`:

- `.master.key` - зашифрованный мастер-ключ
- `token` - токен аутентификации
- `data.json` - локальная база данных (SQLite)
- `state.json` - состояние приложения
- `sync_metadata.json` - метаданные синхронизации
- `sync_stats.json` - статистика синхронизации
- `sync_config.json` - конфигурация синхронизации

### Приоритет конфигурации

1. Флаги командной строки
2. Переменные окружения
3. Конфигурационный файл
4. Значения по умолчанию

## Roadmap

### Краткосрочные задачи

- [ ] Реализовать эндпоинт `/user/change-password` на сервере
- [ ] Добавить команду `export` для сохранения файлов
- [ ] Реализовать интерактивное разрешение конфликтов
- [ ] Добавить команду `update` для обновления записей
- [ ] Добавить команду `delete` для удаления записей

### Среднесрочные задачи

- [ ] Поиск по записям (полнотекстовый)
- [ ] Фильтрация по тегам и категориям
- [ ] История изменений записей
- [ ] Импорт/экспорт базы данных
- [ ] Множественные профили пользователей

### Долгосрочные задачи

- [ ] GUI версия клиента
- [ ] Мобильные приложения (iOS, Android)
- [ ] Браузерное расширение
- [ ] Двухфакторная аутентификация
- [ ] Биометрическая аутентификация
- [ ] Sharing записей между пользователями

## Вклад в проект

При добавлении новых функций следуйте этим принципам:

1. **Безопасность прежде всего**: Все чувствительные данные должны шифроваться
2. **Офлайн-first**: Клиент должен работать без сервера
3. **Graceful degradation**: Обрабатывайте ошибки сети корректно
4. **Логирование**: Используйте структурированное логирование
5. **Тестирование**: Покрывайте код тестами

## Лицензия

См. LICENSE файл в корне проекта.
