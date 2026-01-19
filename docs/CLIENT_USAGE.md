# GophKeeper Client - Руководство пользователя

## Обзор

GophKeeper Client — это клиентское приложение для безопасного хранения паролей, заметок, данных банковских карт и файлов. Все данные шифруются на стороне клиента с использованием мастер-ключа и синхронизируются с сервером.

## Установка

```bash
cd cmd/client
go build -o gophkeeper
```

## Конфигурация

Клиент использует следующие переменные окружения (можно задать в `.env` файле):

```bash
# Адрес сервера
SERVER_ADDRESS=localhost:8080

# Уровень логирования (debug, info, warn, error)
LOG_LEVEL=info

# Окружение (local, dev, prod)
APP_ENV=local

# Путь к мастер-ключу (по умолчанию ~/.gophkeeper/.master.key)
MASTER_KEY_PATH=~/.gophkeeper/.master.key

# Директория конфигурации (по умолчанию ~/.gophkeeper)
CONFIG_DIR=~/.gophkeeper

# Интервал синхронизации в секундах
SYNC_INTERVAL_SECONDS=30

# Использовать TLS
ENABLE_TLS=false

# Путь к CA сертификату (если используется самоподписанный)
CA_CERT_PATH=
```

## Основные команды

### Аутентификация

#### Регистрация нового пользователя

```bash
gophkeeper auth register
```

Интерактивно запросит email и пароль.

#### Вход в систему

```bash
gophkeeper auth login
```

После успешного входа токен сохраняется локально и автоматически запускается синхронизация.

Опции:
- `--remember` / `-r` - сохранить токен для последующих сессий

### Работа с записями

#### Создание записи

```bash
# Интерактивное создание
gophkeeper record create

# Создание пароля с параметрами
gophkeeper record create --type password --name "GitHub" --username "user@example.com" --password "secret123" --url "https://github.com"

# Создание текстовой заметки
gophkeeper record create --type note --name "Важная заметка" --content "Текст заметки"

# Создание записи карты
gophkeeper record create --type card --name "Основная карта" --card-number "1234567890123456" --card-holder "IVAN IVANOV" --expiry "12/25" --cvv "123"

# Создание файла
gophkeeper record create --type file --name "Документ" --file "/path/to/file.pdf"
```

Поддерживаемые типы записей:
- `password` - логин и пароль
- `note` - текстовая заметка
- `card` - данные банковской карты
- `file` - бинарный файл

#### Просмотр списка записей

```bash
# Все записи
gophkeeper record list

# Фильтр по типу
gophkeeper record list --type password

# Показать удаленные
gophkeeper record list --deleted

# Разные форматы вывода
gophkeeper record list --format table
gophkeeper record list --format json
gophkeeper record list --format csv

# Пагинация
gophkeeper record list --limit 10 --offset 20
```

#### Просмотр записи

```bash
# Просмотр записи по ID
gophkeeper record get 123

# Показать пароли
gophkeeper record get 123 --show-password

# Вывод в JSON
gophkeeper record get 123 --output json
```

### Синхронизация

#### Запуск синхронизации

```bash
# Обычная синхронизация
gophkeeper sync

# Принудительная синхронизация
gophkeeper sync --force
```

#### Статус синхронизации

```bash
gophkeeper sync --status
```

Показывает:
- Статистику синхронизаций
- Количество загруженных/скачанных записей
- Конфликты
- Состояние соединения с сервером

#### Просмотр конфликтов

```bash
gophkeeper sync --conflicts
```

#### Сброс статистики

```bash
gophkeeper sync --reset
```

## Безопасность

### Мастер-ключ

При первом запуске клиент создает мастер-ключ на основе вашего пароля. Этот ключ используется для шифрования всех данных локально.

**ВАЖНО**: 
- Мастер-ключ никогда не покидает ваше устройство
- Сервер получает только зашифрованные данные
- Если вы потеряете мастер-пароль, восстановить данные будет невозможно

### Шифрование

- Используется AES-256-GCM для шифрования данных
- PBKDF2-SHA256 для генерации ключа из пароля (100,000 итераций)
- Каждая запись шифруется отдельно с уникальным nonce

### TLS

Для продакшн окружения настоятельно рекомендуется использовать TLS:

```bash
export ENABLE_TLS=true
export SERVER_ADDRESS=gophkeeper.example.com:443
```

## Синхронизация

### Как работает синхронизация

1. **Автоматическая синхронизация**: Запускается каждые N секунд (настраивается через `SYNC_INTERVAL_SECONDS`)
2. **Двусторонняя**: Изменения отправляются на сервер и загружаются с сервера
3. **Конфликты**: Автоматически разрешаются по стратегии (по умолчанию - выбирается более новая версия)

### Стратегии разрешения конфликтов

Настраиваются в файле `~/.gophkeeper/sync_config.json`:

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

Доступные стратегии:
- `client` - всегда выбирать локальную версию
- `server` - всегда выбирать серверную версию
- `newer` - выбирать более новую версию (по умолчанию)
- `manual` - требовать ручного разрешения

## Офлайн режим

Клиент полностью функционален в офлайн режиме:
- Все записи сохраняются локально в SQLite
- При восстановлении соединения автоматически синхронизируются
- Конфликты разрешаются автоматически или вручную

## Примеры использования

### Первый запуск

```bash
# 1. Регистрация
gophkeeper auth register
# Email: user@example.com
# Password: ********

# 2. Вход
gophkeeper auth login
# Email: user@example.com
# Password: ********
# Мастер-пароль: ********

# 3. Создание первой записи
gophkeeper record create --type password --name "Gmail" --username "user@gmail.com"
# Пароль будет сгенерирован автоматически

# 4. Просмотр записей
gophkeeper record list
```

### Работа с паролями

```bash
# Создание пароля с автогенерацией
gophkeeper record create --type password --name "GitHub"
# Логин/Email: user@example.com
# Пароль (оставьте пустым для генерации): 
# Сгенерирован пароль: aB3$xY9#mK2@

# Просмотр пароля
gophkeeper record get 1 --show-password
```

### Работа с заметками

```bash
# Создание заметки
gophkeeper record create --type note --name "Важная информация" --content "Секретная информация"

# Или интерактивно
gophkeeper record create --type note --name "Заметка"
# Введите текст заметки (Ctrl+D для завершения):
# Строка 1
# Строка 2
# ^D
```

### Работа с картами

```bash
gophkeeper record create --type card \
  --name "Основная карта" \
  --card-number "1234567890123456" \
  --card-holder "IVAN IVANOV" \
  --expiry "12/25" \
  --cvv "123"
```

### Работа с файлами

```bash
# Загрузка файла
gophkeeper record create --type file --name "Паспорт" --file "/path/to/passport.pdf"
```

## Устранение неполадок

### Сервер недоступен

```bash
# Проверка соединения
gophkeeper sync --status
```

Если сервер недоступен, клиент продолжит работать в офлайн режиме.

### Забыт мастер-пароль

К сожалению, если вы забыли мастер-пароль, восстановить данные невозможно. Это сделано для обеспечения максимальной безопасности.

### Конфликты синхронизации

```bash
# Просмотр конфликтов
gophkeeper sync --conflicts

# Принудительная синхронизация
gophkeeper sync --force
```

### Сброс клиента

```bash
# Удаление всех локальных данных
rm -rf ~/.gophkeeper

# Повторная инициализация
gophkeeper auth login
```

## Архитектура клиента

### Компоненты

1. **HTTP Client** ([`internal/app/client/http_client.go`](../internal/app/client/http_client.go))
   - Взаимодействие с сервером через REST API
   - Автоматические retry при сетевых ошибках
   - Обработка токенов аутентификации

2. **Crypto** ([`internal/app/client/crypto/`](../internal/app/client/crypto/))
   - Управление мастер-ключом
   - Шифрование/расшифровка записей
   - AES-256-GCM + PBKDF2-SHA256

3. **Storage** ([`internal/app/client/storage_sqlite.go`](../internal/app/client/storage_sqlite.go))
   - Локальное хранилище на SQLite
   - Поддержка офлайн режима
   - Отслеживание синхронизации

4. **Sync Service** ([`internal/app/client/sync.go`](../internal/app/client/sync.go))
   - Двусторонняя синхронизация
   - Обнаружение и разрешение конфликтов
   - Пакетная обработка изменений

### Поток данных

```
Пользователь → CLI → App → Crypto (шифрование) → Storage (локально)
                                                ↓
                                          HTTP Client → Сервер
                                                ↓
                                          Sync Service (синхронизация)
```

## API Endpoints

Клиент взаимодействует со следующими эндпоинтами сервера:

### Аутентификация
- `POST /user/register` - регистрация
- `POST /user/login` - вход

### Записи
- `GET /api/records` - список записей
- `POST /api/records` - создание записи (generic)
- `GET /api/records/{id}` - получение записи
- `PUT /api/records/{id}` - обновление записи
- `DELETE /api/records/{id}` - удаление записи

### Типизированное создание
- `POST /api/records/login` - создание логина
- `POST /api/records/text` - создание текста
- `POST /api/records/card` - создание карты
- `POST /api/records/binary` - создание файла

### Синхронизация
- `POST /api/sync/changes` - получение изменений
- `POST /api/sync/batch` - пакетная синхронизация
- `GET /api/sync/status` - статус синхронизации
- `GET /api/sync/conflicts` - список конфликтов
- `POST /api/sync/conflicts/{id}/resolve` - разрешение конфликта
- `GET /api/sync/devices` - список устройств
- `DELETE /api/sync/devices/{id}` - удаление устройства

## Разработка

### Структура проекта

```
cmd/client/
├── main.go                 # Точка входа
└── cmd/                    # CLI команды
    ├── root.go            # Корневая команда
    ├── init.go            # Инициализация
    ├── auth/              # Команды аутентификации
    ├── record/            # Команды работы с записями
    └── sync/              # Команды синхронизации

internal/app/client/
├── client.go              # Основное приложение
├── http_client.go         # HTTP клиент
├── storage_sqlite.go      # Локальное хранилище
├── sync.go                # Сервис синхронизации
├── models.go              # Модели данных
├── config/                # Конфигурация
└── crypto/                # Криптография
    ├── master_key.go      # Управление мастер-ключом
    ├── encryption.go      # Шифрование записей
    └── utils.go           # Утилиты
```

### Добавление новой команды

1. Создайте файл в `cmd/client/cmd/<category>/`
2. Определите команду с помощью `cobra.Command`
3. Зарегистрируйте в `init.go`

Пример:

```go
package record

import (
    "github.com/spf13/cobra"
    "gophkeeper/internal/app/client"
)

var ExportCmd = &cobra.Command{
    Use:   "export [id] [output]",
    Short: "Экспортировать запись",
    RunE: func(cmd *cobra.Command, args []string) error {
        app := cmd.Context().Value("app").(*client.App)
        // Ваша логика
        return nil
    },
}
```

### Тестирование

```bash
# Запуск тестов
go test ./internal/app/client/...

# Тесты с покрытием
go test -cover ./internal/app/client/...

# Тесты криптографии
go test ./internal/app/client/crypto/...
```

## Известные ограничения

1. **Смена пароля**: Функция временно недоступна, требуется реализация на сервере
2. **Экспорт файлов**: Команда export для сохранения бинарных файлов не реализована
3. **Ручное разрешение конфликтов**: Интерактивное разрешение конфликтов будет добавлено в будущих версиях

## Roadmap

- [ ] Реализация команды export для файлов
- [ ] Интерактивное разрешение конфликтов
- [ ] Поддержка множественных профилей
- [ ] Импорт/экспорт всей базы данных
- [ ] Поиск по записям
- [ ] Теги и категории
- [ ] История изменений записей
- [ ] Двухфакторная аутентификация
