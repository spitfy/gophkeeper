# Обработка блокировки мастер-ключа в GophKeeper Client

## Обзор

Мастер-ключ в GophKeeper используется для шифрования всех пользовательских данных. Он может находиться в двух состояниях:
- **Разблокирован** (`isLocked: false`) - ключ загружен в память и готов к использованию
- **Заблокирован** (`isLocked: true`) - ключ очищен из памяти для безопасности

## Сценарии блокировки мастер-ключа

### 1. Начальное состояние
При создании [`MasterKeyManager`](../internal/app/client/crypto/master_key.go:63-85) мастер-ключ по умолчанию заблокирован:
```go
manager := &MasterKeyManager{
    keyPath:  absPath,
    isLoaded: false,
    isLocked: true,  // Заблокирован по умолчанию
}
```

### 2. После ручной блокировки
Пользователь может вручную заблокировать ключ вызовом метода [`Lock()`](../internal/app/client/crypto/master_key.go:393-400):
```go
func (m *MasterKeyManager) Lock() {
    m.mu.Lock()
    defer m.mu.Unlock()
    
    m.clearKey()
    m.isLocked = true
}
```

### 3. Не разблокирован после инициализации
После выполнения `gophkeeper init` мастер-ключ остается разблокированным, но при следующем запуске приложения он будет заблокирован.

## Проверки мастер-ключа в командах

### Команды, требующие разблокированный мастер-ключ

#### 1. Синхронизация (`gophkeeper sync`)
**Файл:** [`cmd/client/cmd/sync/sync.go:55-61`](../cmd/client/cmd/sync/sync.go:55-61)

Проверка выполняется перед началом синхронизации:
```go
if !app.IsMasterKeyUnlocked() {
    fmt.Println("❌ Мастер-ключ заблокирован")
    fmt.Println()
    fmt.Println("Для синхронизации необходимо разблокировать мастер-ключ.")
    fmt.Println("Выполните команду: gophkeeper unlock")
    return fmt.Errorf("мастер-ключ заблокирован")
}
```

Также проверка выполняется в [`preSyncChecks()`](../internal/app/client/sync.go:280-283):
```go
// 4. Проверяем мастер-ключ
if s.app.crypto.IsLocked() {
    return fmt.Errorf("мастер-ключ заблокирован")
}
```

#### 2. Создание записей (`gophkeeper record create`)
**Файл:** [`cmd/client/cmd/record/create.go:47-53`](../cmd/client/cmd/record/create.go:47-53)

Проверка выполняется перед созданием любой записи:
```go
if !app.IsMasterKeyUnlocked() {
    fmt.Println("❌ Мастер-ключ заблокирован")
    fmt.Println()
    fmt.Println("Для создания записей необходимо разблокировать мастер-ключ.")
    fmt.Println("Выполните команду: gophkeeper unlock")
    return fmt.Errorf("мастер-ключ заблокирован")
}
```

#### 3. Операции шифрования/дешифрования
**Файл:** [`internal/app/client/crypto/master_key.go`](../internal/app/client/crypto/master_key.go)

Все операции шифрования проверяют состояние ключа:
```go
func (m *MasterKeyManager) EncryptData(plaintext []byte) ([]byte, error) {
    m.mu.RLock()
    defer m.mu.RUnlock()
    
    if !m.isLoaded || m.isLocked {
        return nil, fmt.Errorf("мастер-ключ не загружен или заблокирован")
    }
    
    return encryptWithKey(m.masterKey, plaintext)
}

func (m *MasterKeyManager) DecryptData(ciphertext []byte) ([]byte, error) {
    m.mu.RLock()
    defer m.mu.RUnlock()
    
    if !m.isLoaded || m.isLocked {
        return nil, fmt.Errorf("мастер-ключ не загружен или заблокирован")
    }
    
    return decryptWithKey(m.masterKey, ciphertext)
}
```

### Команды, НЕ требующие разблокированный мастер-ключ

1. **`gophkeeper auth register`** - регистрация на сервере
2. **`gophkeeper auth login`** - вход (автоматически разблокирует ключ)
3. **`gophkeeper record list`** - просмотр списка (только метаданные)
4. **`gophkeeper record get`** - просмотр записи (без расшифровки)

## Разблокировка мастер-ключа

### Автоматическая разблокировка

#### При входе в систему (`gophkeeper auth login`)
**Файл:** [`cmd/client/cmd/auth/login.go:63-75`](../cmd/client/cmd/auth/login.go:63-75)

Команда login автоматически запрашивает мастер-пароль и разблокирует ключ:
```go
if app.IsInitialized() {
    // Разблокируем существующий мастер-ключ
    fmt.Print("Мастер-пароль (для расшифровки данных): ")
    masterPassword, err := term.ReadPassword(int(os.Stdin.Fd()))
    if err != nil {
        return fmt.Errorf("ошибка чтения мастер-пароля: %w", err)
    }
    fmt.Println()
    
    if err := app.UnlockMasterKey(string(masterPassword)); err != nil {
        return fmt.Errorf("неверный мастер-пароль: %w", err)
    }
}
```

### Ручная разблокировка

#### Команда `gophkeeper unlock`
**Файл:** [`cmd/client/cmd/init.go:92-133`](../cmd/client/cmd/init.go:92-133)

Новая команда для явной разблокировки мастер-ключа:
```go
var unlockCmd = &cobra.Command{
    Use:   "unlock",
    Short: "Разблокировать мастер-ключ",
    Long: `Разблокирует мастер-ключ для работы с зашифрованными данными.
    
Мастер-ключ необходим для:
- Создания новых записей
- Просмотра зашифрованных данных
- Синхронизации с сервером`,
    RunE: func(cmd *cobra.Command, args []string) error {
        // Проверяем, инициализирован ли клиент
        if !app.IsInitialized() {
            return fmt.Errorf("клиент не инициализирован. Выполните: gophkeeper init")
        }
        
        // Проверяем, не разблокирован ли уже
        if app.IsMasterKeyUnlocked() {
            fmt.Println("✅ Мастер-ключ уже разблокирован")
            return nil
        }
        
        fmt.Println("=== Разблокировка мастер-ключа ===")
        fmt.Println()
        
        // Запрашиваем мастер-пароль
        fmt.Print("Введите мастер-пароль: ")
        password, err := term.ReadPassword(int(os.Stdin.Fd()))
        if err != nil {
            return fmt.Errorf("ошибка чтения пароля: %w", err)
        }
        fmt.Println()
        
        // Разблокируем мастер-ключ
        if err := app.UnlockMasterKey(string(password)); err != nil {
            return fmt.Errorf("ошибка разблокировки: %w", err)
        }
        
        fmt.Println("✅ Мастер-ключ успешно разблокирован!")
        return nil
    },
}
```

## Вспомогательные методы

### Проверка состояния мастер-ключа

#### В MasterKeyManager
```go
// IsLocked проверяет, заблокирован ли ключ
func (m *MasterKeyManager) IsLocked() bool {
    m.mu.RLock()
    defer m.mu.RUnlock()
    return m.isLocked
}

// IsInitialized проверяет, инициализирован ли мастер-ключ
func (m *MasterKeyManager) IsInitialized() bool {
    m.mu.RLock()
    defer m.mu.RUnlock()
    return m.header.CreatedAt != (time.Time{})
}
```

#### В App
**Файл:** [`internal/app/client/client.go:221-224`](../internal/app/client/client.go:221-224)

```go
// IsMasterKeyUnlocked проверяет, разблокирован ли мастер-ключ
func (a *App) IsMasterKeyUnlocked() bool {
    return !a.crypto.IsLocked()
}
```

## Потокобезопасность

Все операции с мастер-ключом защищены мьютексами:
- `sync.RWMutex` для чтения/записи состояния
- Блокировка при изменении состояния ключа
- Разблокировка только для чтения состояния

```go
type MasterKeyManager struct {
    masterKey []byte          // Загруженный мастер-ключ в памяти
    header    MasterKeyHeader // Заголовок с метаданными
    keyPath   string          // Путь к файлу мастер-ключа
    isLoaded  bool            // Загружен ли ключ в память
    isLocked  bool            // Заблокирован ли ключ
    mu        sync.RWMutex    // Мьютекс для потокобезопасности
}
```

## Безопасность

### Очистка памяти
При блокировке ключ безопасно очищается из памяти:
```go
func (m *MasterKeyManager) clearKey() {
    if m.masterKey != nil {
        // Затираем память нулями перед освобождением
        for i := range m.masterKey {
            m.masterKey[i] = 0
        }
        m.masterKey = nil
    }
    m.isLoaded = false
}
```

## Типичные сценарии использования

### Сценарий 1: Первый запуск
```bash
# 1. Инициализация (создание мастер-ключа)
gophkeeper init
# Мастер-ключ разблокирован после init

# 2. Регистрация на сервере
gophkeeper auth register

# 3. Вход в систему (автоматически разблокирует ключ)
gophkeeper auth login

# 4. Создание записи (ключ уже разблокирован)
gophkeeper record create
```

### Сценарий 2: Повторный запуск
```bash
# 1. Мастер-ключ заблокирован при старте

# 2. Попытка синхронизации
gophkeeper sync
# ❌ Ошибка: мастер-ключ заблокирован

# 3. Разблокировка
gophkeeper unlock
# ✅ Мастер-ключ разблокирован

# 4. Повторная попытка синхронизации
gophkeeper sync
# ✅ Успешно
```

### Сценарий 3: Вход после перезапуска
```bash
# 1. Вход в систему (автоматически разблокирует ключ)
gophkeeper auth login
# Запросит мастер-пароль и разблокирует ключ

# 2. Работа с записями (ключ уже разблокирован)
gophkeeper record create
gophkeeper sync
```

## Обработка ошибок

Все команды, требующие разблокированный мастер-ключ, выводят понятные сообщения об ошибке:

```
❌ Мастер-ключ заблокирован

Для [операции] необходимо разблокировать мастер-ключ.
Выполните команду: gophkeeper unlock
```

Это помогает пользователю понять проблему и знать, как её решить.
