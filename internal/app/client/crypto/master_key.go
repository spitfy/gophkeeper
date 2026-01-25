// internal/app/client/crypto/master_key.go
package crypto

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync"
	"time"

	"golang.org/x/crypto/argon2"
	"golang.org/x/crypto/pbkdf2"
)

const (
	// Константы для PBKDF2
	pbkdf2Iterations = 100000
	pbkdf2KeyLength  = 32 // 256 бит для AES-256
	pbkdf2SaltLength = 16

	// Константы для Argon2 (более безопасная альтернатива)
	argon2Time    = 1
	argon2Memory  = 64 * 1024 // 64 MB
	argon2Threads = 4
	argon2KeyLen  = 32

	// Константы для шифрования
	keyVersion = 1

	// Константы для файла мастер-ключа
	masterKeyPermissions = 0600
)

// MasterKeyHeader содержит метаданные мастер-ключа
type MasterKeyHeader struct {
	Version      int       `json:"version"`
	KeyAlgorithm string    `json:"key_algorithm"`
	Salt         string    `json:"salt"` // base64 encoded salt
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
	KeyHash      string    `json:"key_hash"`   // SHA256 хэш ключа для проверки
	Iterations   int       `json:"iterations"` // Для PBKDF2
}

// MasterKeyManager управляет мастер-ключом
type MasterKeyManager struct {
	masterKey []byte          // Загруженный мастер-ключ в памяти
	header    MasterKeyHeader // Заголовок с метаданными
	keyPath   string          // Путь к файлу мастер-ключа
	isLoaded  bool            // Загружен ли ключ в память
	isLocked  bool            // Заблокирован ли ключ (очищен из памяти)
	mu        sync.RWMutex
}

// NewMasterKeyManager создает новый менеджер мастер-ключа
func NewMasterKeyManager(keyPath string) (*MasterKeyManager, error) {
	// Нормализуем путь
	absPath, err := filepath.Abs(keyPath)
	if err != nil {
		return nil, fmt.Errorf("ошибка определения пути: %w", err)
	}

	manager := &MasterKeyManager{
		keyPath:  absPath,
		isLoaded: false,
		isLocked: true,
	}

	// Если файл существует, загружаем заголовок
	if _, err := os.Stat(absPath); err == nil {
		if err := manager.loadHeader(); err != nil {
			return nil, fmt.Errorf("ошибка загрузки заголовка ключа: %w", err)
		}
	}

	// Пытаемся загрузить активную сессию
	_ = manager.LoadSession()

	return manager, nil
}

// GenerateMasterKey генерирует новый мастер-ключ
func (m *MasterKeyManager) GenerateMasterKey(password string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Генерируем соль
	salt := make([]byte, pbkdf2SaltLength)
	if _, err := io.ReadFull(rand.Reader, salt); err != nil {
		return fmt.Errorf("ошибка генерации соли: %w", err)
	}

	// Генерируем ключ из пароля с помощью PBKDF2
	key := pbkdf2.Key([]byte(password), salt, pbkdf2Iterations, pbkdf2KeyLength, sha256.New)

	// Вычисляем хэш ключа для будущей проверки
	keyHash := sha256.Sum256(key)

	// Создаем заголовок
	m.header = MasterKeyHeader{
		Version:      keyVersion,
		KeyAlgorithm: "PBKDF2-SHA256",
		Salt:         hex.EncodeToString(salt),
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
		KeyHash:      hex.EncodeToString(keyHash[:]),
		Iterations:   pbkdf2Iterations,
	}

	// Сохраняем ключ в память
	m.masterKey = key
	m.isLoaded = true
	m.isLocked = false

	// Сохраняем ключ в файл (зашифрованный)
	if err := m.saveMasterKey(); err != nil {
		m.clearKey()
		return fmt.Errorf("ошибка сохранения мастер-ключа: %w", err)
	}

	return nil
}

// UnlockMasterKey загружает мастер-ключ из файла с использованием пароля
func (m *MasterKeyManager) UnlockMasterKey(password string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.isLoaded && !m.isLocked {
		return nil
	}

	encryptedData, err := os.ReadFile(m.keyPath)
	if err != nil {
		return fmt.Errorf("ошибка чтения файла ключа: %w", err)
	}

	var container struct {
		Header MasterKeyHeader `json:"header"`
		Data   string          `json:"data"` // base64 encoded encrypted key
	}

	if err := json.Unmarshal(encryptedData, &container); err != nil {
		return fmt.Errorf("ошибка декодирования файла ключа: %w", err)
	}

	m.header = container.Header

	// Декодируем соль
	salt, err := hex.DecodeString(m.header.Salt)
	if err != nil {
		return fmt.Errorf("ошибка декодирования соли: %w", err)
	}

	// Восстанавливаем ключ из пароля
	var key []byte
	switch m.header.KeyAlgorithm {
	case "PBKDF2-SHA256":
		key = pbkdf2.Key([]byte(password), salt, m.header.Iterations, pbkdf2KeyLength, sha256.New)
	case "Argon2id":
		// Для будущей поддержки Argon2
		salt = salt[:16] // Argon2 требует 16 байт соли
		key = argon2.IDKey([]byte(password), salt, argon2Time, argon2Memory, argon2Threads, argon2KeyLen)
	default:
		return fmt.Errorf("неподдерживаемый алгоритм: %s", m.header.KeyAlgorithm)
	}

	// Проверяем хэш ключа
	keyHash := sha256.Sum256(key)
	if hex.EncodeToString(keyHash[:]) != m.header.KeyHash {
		return fmt.Errorf("неверный пароль")
	}

	// Декодируем и расшифровываем мастер-ключ
	encryptedKey, err := hex.DecodeString(container.Data)
	if err != nil {
		return fmt.Errorf("ошибка декодирования зашифрованного ключа: %w", err)
	}

	// Для первого раза ключ еще не зашифрован (генерируется из пароля)
	if len(encryptedKey) == 0 {
		m.masterKey = key
	} else {
		// Расшифровываем мастер-ключ
		decryptedKey, err := decryptWithKey(key, encryptedKey)
		if err != nil {
			return fmt.Errorf("ошибка расшифровки мастер-ключа: %w", err)
		}
		m.masterKey = decryptedKey
	}

	m.isLoaded = true
	m.isLocked = false

	m.mu.Unlock()
	_ = m.SaveSession()
	m.mu.Lock()

	return nil
}

// saveMasterKey сохраняет мастер-ключ в файл
func (m *MasterKeyManager) saveMasterKey() error {
	// Для первого сохранения используем ключ, полученный из пароля
	// (он же и будет мастер-ключом)
	var encryptedData string

	// Если у нас уже есть мастер-ключ в памяти, шифруем его самим собой
	if len(m.masterKey) > 0 {
		// Шифруем мастер-ключ самим собой
		encryptedKey, err := encryptWithKey(m.masterKey, m.masterKey)
		if err != nil {
			return fmt.Errorf("ошибка шифрования мастер-ключа: %w", err)
		}
		encryptedData = hex.EncodeToString(encryptedKey)
	}

	container := struct {
		Header MasterKeyHeader `json:"header"`
		Data   string          `json:"data"`
	}{
		Header: m.header,
		Data:   encryptedData,
	}

	// Сериализуем в JSON
	data, err := json.MarshalIndent(container, "", "  ")
	if err != nil {
		return fmt.Errorf("ошибка сериализации: %w", err)
	}

	// Сохраняем в файл
	if err := os.WriteFile(m.keyPath, data, masterKeyPermissions); err != nil {
		return fmt.Errorf("ошибка записи файла: %w", err)
	}

	return nil
}

// loadHeader загружает только заголовок мастер-ключа
func (m *MasterKeyManager) loadHeader() error {
	data, err := os.ReadFile(m.keyPath)
	if err != nil {
		return fmt.Errorf("ошибка чтения файла ключа: %w", err)
	}

	var container struct {
		Header MasterKeyHeader `json:"header"`
	}

	if err := json.Unmarshal(data, &container); err != nil {
		return fmt.Errorf("ошибка декодирования файла ключа: %w", err)
	}

	m.header = container.Header
	return nil
}

// EncryptData шифрует данные с использованием мастер-ключа
func (m *MasterKeyManager) EncryptData(plaintext []byte) ([]byte, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if !m.isLoaded || m.isLocked {
		return nil, fmt.Errorf("мастер-ключ не загружен или заблокирован")
	}

	return encryptWithKey(m.masterKey, plaintext)
}

// DecryptData расшифровывает данные с использованием мастер-ключа
func (m *MasterKeyManager) DecryptData(ciphertext []byte) ([]byte, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if !m.isLoaded || m.isLocked {
		return nil, fmt.Errorf("мастер-ключ не загружен или заблокирован")
	}

	return decryptWithKey(m.masterKey, ciphertext)
}

// EncryptDataWithPassword шифрует данные с использованием пароля напрямую
func (m *MasterKeyManager) EncryptDataWithPassword(plaintext []byte, password string) ([]byte, error) {
	// Генерируем ключ из пароля для этого шифрования
	salt := make([]byte, pbkdf2SaltLength)
	if _, err := io.ReadFull(rand.Reader, salt); err != nil {
		return nil, fmt.Errorf("ошибка генерации соли: %w", err)
	}

	key := pbkdf2.Key([]byte(password), salt, pbkdf2Iterations, pbkdf2KeyLength, sha256.New)

	// Шифруем данные
	ciphertext, err := encryptWithKey(key, plaintext)
	if err != nil {
		return nil, err
	}

	// Добавляем соль в начало шифротекста для возможности расшифровки
	result := make([]byte, len(salt)+len(ciphertext))
	copy(result[:len(salt)], salt)
	copy(result[len(salt):], ciphertext)

	return result, nil
}

// DecryptDataWithPassword расшифровывает данные с использованием пароля
func (m *MasterKeyManager) DecryptDataWithPassword(ciphertext []byte, password string) ([]byte, error) {
	// Извлекаем соль из шифротекста
	if len(ciphertext) < pbkdf2SaltLength {
		return nil, fmt.Errorf("неверный формат шифротекста")
	}

	salt := ciphertext[:pbkdf2SaltLength]
	encryptedData := ciphertext[pbkdf2SaltLength:]

	// Генерируем ключ из пароля
	key := pbkdf2.Key([]byte(password), salt, pbkdf2Iterations, pbkdf2KeyLength, sha256.New)

	// Расшифровываем данные
	return decryptWithKey(key, encryptedData)
}

// GetKeyHash возвращает хэш текущего мастер-ключа
func (m *MasterKeyManager) GetKeyHash() (string, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if !m.isLoaded || m.isLocked {
		return "", fmt.Errorf("мастер-ключ не загружен или заблокирован")
	}

	keyHash := sha256.Sum256(m.masterKey)
	return hex.EncodeToString(keyHash[:]), nil
}

// ChangeMasterPassword изменяет мастер-пароль
func (m *MasterKeyManager) ChangeMasterPassword(oldPassword, newPassword string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Проверяем старый пароль
	if err := m.verifyPassword(oldPassword); err != nil {
		return fmt.Errorf("неверный старый пароль: %w", err)
	}

	// Генерируем новую соль
	newSalt := make([]byte, pbkdf2SaltLength)
	if _, err := io.ReadFull(rand.Reader, newSalt); err != nil {
		return fmt.Errorf("ошибка генерации новой соли: %w", err)
	}

	// Генерируем новый ключ из пароля
	newKey := pbkdf2.Key([]byte(newPassword), newSalt, pbkdf2Iterations, pbkdf2KeyLength, sha256.New)
	newKeyHash := sha256.Sum256(newKey)

	// Обновляем заголовок
	m.header.Salt = hex.EncodeToString(newSalt)
	m.header.KeyHash = hex.EncodeToString(newKeyHash[:])
	m.header.UpdatedAt = time.Now()

	// Шифруем текущий мастер-ключ новым ключом
	encryptedMasterKey, err := encryptWithKey(newKey, m.masterKey)
	if err != nil {
		return fmt.Errorf("ошибка шифрования нового мастер-ключа: %w", err)
	}

	// Сохраняем изменения
	container := struct {
		Header MasterKeyHeader `json:"header"`
		Data   string          `json:"data"`
	}{
		Header: m.header,
		Data:   hex.EncodeToString(encryptedMasterKey),
	}

	data, err := json.MarshalIndent(container, "", "  ")
	if err != nil {
		return fmt.Errorf("ошибка сериализации: %w", err)
	}

	if err := os.WriteFile(m.keyPath, data, masterKeyPermissions); err != nil {
		return fmt.Errorf("ошибка записи файла: %w", err)
	}

	return nil
}

// Lock блокирует мастер-ключ (очищает из памяти)
func (m *MasterKeyManager) Lock() {
	m.clearKey()
	m.isLocked = true
	_ = m.ClearSession()
}

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

// verifyPassword проверяет пароль без разблокировки ключа
func (m *MasterKeyManager) verifyPassword(password string) error {
	salt, err := hex.DecodeString(m.header.Salt)
	if err != nil {
		return fmt.Errorf("ошибка декодирования соли: %w", err)
	}

	var key []byte
	switch m.header.KeyAlgorithm {
	case "PBKDF2-SHA256":
		key = pbkdf2.Key([]byte(password), salt, m.header.Iterations, pbkdf2KeyLength, sha256.New)
	default:
		return fmt.Errorf("неподдерживаемый алгоритм: %s", m.header.KeyAlgorithm)
	}

	keyHash := sha256.Sum256(key)
	if hex.EncodeToString(keyHash[:]) != m.header.KeyHash {
		return fmt.Errorf("неверный пароль")
	}

	return nil
}

// clearKey безопасно очищает ключ из памяти
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

// encryptWithKey шифрует данные с использованием AES-GCM
func encryptWithKey(key, plaintext []byte) ([]byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("ошибка создания cipher: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("ошибка создания GCM: %w", err)
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, fmt.Errorf("ошибка генерации nonce: %w", err)
	}

	ciphertext := gcm.Seal(nonce, nonce, plaintext, nil)
	return ciphertext, nil
}

// decryptWithKey расшифровывает данные с использованием AES-GCM
func decryptWithKey(key, ciphertext []byte) ([]byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("ошибка создания cipher: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("ошибка создания GCM: %w", err)
	}

	nonceSize := gcm.NonceSize()
	if len(ciphertext) < nonceSize {
		return nil, fmt.Errorf("шифротекст слишком короткий")
	}

	nonce, ciphertext := ciphertext[:nonceSize], ciphertext[nonceSize:]
	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return nil, fmt.Errorf("ошибка расшифровки: %w", err)
	}

	return plaintext, nil
}
