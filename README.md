# GophKeeper

GophKeeper — это безопасный менеджер паролей с клиентским шифрованием и двусторонней синхронизацией между устройствами. Все данные шифруются на стороне клиента с использованием мастер-ключа, и сервер получает только зашифрованные данные.

## Особенности

- **Клиентское шифрование**: Все данные шифруются локально с использованием AES-256-GCM
- **Безопасное хранение**: Поддержка паролей, текстовых заметок, банковских карт и бинарных файлов
- **Двусторонняя синхронизация**: Автоматическая синхронизация между устройствами
- **Офлайн режим**: Полная функциональность без подключения к интернету
- **Конфликтное разрешение**: Автоматическое и ручное разрешение конфликтов синхронизации
- **Открытый исходный код**: Написан на Go с использованием современных библиотек

## Архитектура

- **Клиент**: CLI-приложение на Go с локальным SQLite хранилищем
- **Сервер**: HTTP API на Go с PostgreSQL базой данных
- **Шифрование**: AES-256-GCM + PBKDF2-SHA256 для генерации ключей
- **Аутентификация**: JWT токены с refresh-токенами

## Быстрый старт

### Запуск сервера

```bash
# Сборка сервера
cd cmd/server
go build -o gophkeeper-server

# Запуск сервера
./gophkeeper-server
```

Или с помощью Docker:

```bash
docker-compose up -d
```

### Установка клиента

```bash
# Сборка клиента
cd cmd/client
go build -o gophkeeper

# Или установка глобально
go install ./cmd/client
```

### Базовое использование

```bash
# Регистрация нового пользователя
gophkeeper auth register

# Вход в систему
gophkeeper auth login

# Создание записи
gophkeeper record create --type password --name "GitHub" --username "user@example.com"

# Просмотр записей
gophkeeper record list

# Синхронизация
gophkeeper sync
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
```

## Поддерживаемые типы записей

- **password**: Логин и пароль с поддержкой автогенерации паролей
- **note**: Текстовые заметки с многострочным вводом
- **card**: Данные банковских карт (номер, владелец, срок действия, CVV)
- **file**: Бинарные файлы любого типа

## Синхронизация

Синхронизация работает по следующей схеме:

1. **Автоматическая**: Запускается каждые 30 секунд (настраивается)
2. **Двусторонняя**: Изменения отправляются на сервер и загружаются с сервера
3. **Пакетная**: Изменения группируются для оптимизации трафика
4. **Конфликтное разрешение**: Поддержка стратегий `client`, `server`, `newer`, `manual`

## Безопасность

- **Мастер-ключ**: Никогда не покидает устройство пользователя
- **Шифрование**: AES-256-GCM для данных, PBKDF2-SHA256 для генерации ключей
- **TLS**: Поддержка HTTPS для продакшн окружения
- **JWT**: Безопасная аутентификация с refresh-токенами

## Документация

Подробная документация доступна в директории [`docs/`](docs/):

- [CLIENT_USAGE.md](docs/CLIENT_USAGE.md) - Руководство пользователя
- [CLIENT_SIDE_ENCRYPTION.md](docs/CLIENT_SIDE_ENCRYPTION.md) - Клиентское шифрование
- [MASTER_KEY_HANDLING.md](docs/MASTER_KEY_HANDLING.md) - Управление мастер-ключом
- [DATABASE_REFACTORING.md](docs/DATABASE_REFACTORING.md) - Рефакторинг базы данных
- [HEALTH_CHECK_IMPLEMENTATION.md](docs/HEALTH_CHECK_IMPLEMENTATION.md) - Реализация health check

## Разработка

### Структура проекта

```
cmd/
├── client/          # CLI клиент
└── server/          # HTTP сервер

internal/
├── app/             # Основная логика приложения
│   ├── client/      # Клиентская логика
│   └── server/      # Серверная логика
├── domain/          # Доменные модели
└── infrastructure/  # Инфраструктурные компоненты

docs/                # Документация
migrations/          # Миграции базы данных
```

### Тестирование

```bash
# Запуск всех тестов
go test ./...

# Тесты с покрытием
go test -cover ./...

# Запуск линтера
golangci-lint run
```

### Миграции базы данных

```bash
# Применить миграции
migrate -path migrations -database "postgres://postgres:postgres@localhost:5432/gophkeeper?sslmode=disable" up

# Откатить миграции
migrate -path migrations -database "postgres://postgres:postgres@localhost:5432/gophkeeper?sslmode=disable" down 1
```

## Лицензия

MIT License