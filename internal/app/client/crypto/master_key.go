package crypto

import (
	"crypto/rand"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"

	"golang.org/x/crypto/argon2"
	"golang.org/x/crypto/nacl/secretbox"
	"golang.org/x/term"
)

var (
	ErrMasterKeyNotFound = errors.New("master key not found")
	ErrInvalidPassword   = errors.New("invalid password")
	ErrInvalidKeyFile    = errors.New("invalid key file format")
)

// MasterKeyParams параметры для генерации ключа
type MasterKeyParams struct {
	Salt      []byte `json:"salt"`
	Time      uint32 `json:"time"`
	Memory    uint32 `json:"memory"`
	Threads   uint8  `json:"threads"`
	KeyLength uint32 `json:"key_length"`
}

// MasterKey зашифрованный мастер-ключ
type MasterKey struct {
	Params       MasterKeyParams `json:"params"`
	EncryptedKey []byte          `json:"encrypted_key"`
	Nonce        []byte          `json:"nonce"`
}

// DefaultMasterKeyParams возвращает рекомендуемые параметры Argon2id
func DefaultMasterKeyParams() MasterKeyParams {
	return MasterKeyParams{
		Time:      3,                       // 3 итерации
		Memory:    64 * 1024,               // 64 МБ памяти
		Threads:   uint8(runtime.NumCPU()), // Количество потоков = количеству CPU
		KeyLength: 32,                      // 32 байта для AES-256
	}
}

// GenerateSalt генерирует криптографически безопасную соль
func GenerateSalt() ([]byte, error) {
	salt := make([]byte, 16)
	if _, err := io.ReadFull(rand.Reader, salt); err != nil {
		return nil, fmt.Errorf("failed to generate salt: %w", err)
	}
	return salt, nil
}

// DeriveKeyFromPassword создает ключ из пароля и соли с использованием Argon2id
func DeriveKeyFromPassword(password []byte, params MasterKeyParams) ([]byte, error) {
	// Используем Argon2id - рекомендованный алгоритм для хеширования паролей
	key := argon2.IDKey(
		password,
		params.Salt,
		params.Time,
		params.Memory,
		params.Threads,
		params.KeyLength,
	)

	// Затираем пароль из памяти после использования
	clearMemory(password)

	return key, nil
}

// CreateMasterKey создает новый мастер-ключ из пароля
func CreateMasterKey(password []byte) (*MasterKey, error) {
	// Генерируем соль
	salt, err := GenerateSalt()
	if err != nil {
		return nil, err
	}

	// Создаем параметры
	params := DefaultMasterKeyParams()
	params.Salt = salt

	// Генерируем ключ из пароля
	derivedKey, err := DeriveKeyFromPassword(password, params)
	if err != nil {
		return nil, err
	}
	defer clearMemory(derivedKey)

	// Генерируем случайный мастер-ключ
	masterKey := make([]byte, 32)
	if _, err := io.ReadFull(rand.Reader, masterKey); err != nil {
		return nil, fmt.Errorf("failed to generate master key: %w", err)
	}

	// Шифруем мастер-ключ с использованием derivedKey
	nonce := make([]byte, 24)
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, fmt.Errorf("failed to generate nonce: %w", err)
	}

	// Используем secretbox для шифрования
	var nonceArr [24]byte
	var keyArr [32]byte
	copy(nonceArr[:], nonce)
	copy(keyArr[:], derivedKey)

	encryptedKey := secretbox.Seal(nil, masterKey, &nonceArr, &keyArr)

	return &MasterKey{
		Params:       params,
		EncryptedKey: encryptedKey,
		Nonce:        nonce,
	}, nil
}

// DecryptMasterKey расшифровывает мастер-ключ с использованием пароля
func DecryptMasterKey(masterKey *MasterKey, password []byte) ([]byte, error) {
	// Восстанавливаем derivedKey из пароля
	derivedKey, err := DeriveKeyFromPassword(password, masterKey.Params)
	if err != nil {
		return nil, err
	}
	defer clearMemory(derivedKey)

	// Подготавливаем данные для secretbox
	var nonceArr [24]byte
	var keyArr [32]byte

	if len(masterKey.Nonce) != 24 {
		return nil, ErrInvalidKeyFile
	}
	if len(derivedKey) != 32 {
		return nil, ErrInvalidKeyFile
	}

	copy(nonceArr[:], masterKey.Nonce)
	copy(keyArr[:], derivedKey)

	// Расшифровываем мастер-ключ
	decrypted, ok := secretbox.Open(nil, masterKey.EncryptedKey, &nonceArr, &keyArr)
	if !ok {
		return nil, ErrInvalidPassword
	}

	return decrypted, nil
}

// SaveMasterKey сохраняет зашифрованный мастер-ключ в файл
func SaveMasterKey(masterKey *MasterKey, path string) error {
	// Кодируем в формат: версия(2) + params + nonce(24) + encrypted_key
	var data []byte

	// Версия 1
	data = append(data, 0x01, 0x00)

	// Salt (16 байт)
	data = append(data, masterKey.Params.Salt...)

	// Time (4 байта, little-endian)
	timeBytes := make([]byte, 4)
	binary.LittleEndian.PutUint32(timeBytes, masterKey.Params.Time)
	data = append(data, timeBytes...)

	// Memory (4 байта, little-endian)
	memoryBytes := make([]byte, 4)
	binary.LittleEndian.PutUint32(memoryBytes, masterKey.Params.Memory)
	data = append(data, memoryBytes...)

	// Threads (1 байт)
	data = append(data, masterKey.Params.Threads)

	// KeyLength (4 байта, little-endian)
	keyLengthBytes := make([]byte, 4)
	binary.LittleEndian.PutUint32(keyLengthBytes, masterKey.Params.KeyLength)
	data = append(data, keyLengthBytes...)

	// Nonce (24 байта)
	data = append(data, masterKey.Nonce...)

	// Encrypted key
	data = append(data, masterKey.EncryptedKey...)

	// Сохраняем в файл с правами 0400 (только чтение владельцем)
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	return os.WriteFile(path, data, 0400)
}

// LoadMasterKey загружает зашифрованный мастер-ключ из файла
func LoadMasterKey(path string) (*MasterKey, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, ErrMasterKeyNotFound
		}
		return nil, fmt.Errorf("failed to read master key file: %w", err)
	}

	if len(data) < 2 {
		return nil, ErrInvalidKeyFile
	}

	// Проверяем версию
	if data[0] != 0x01 || data[1] != 0x00 {
		return nil, ErrInvalidKeyFile
	}

	data = data[2:]

	// Минимальный размер: salt(16) + time(4) + memory(4) + threads(1) + keylength(4) + nonce(24) = 53 байта
	if len(data) < 53 {
		return nil, ErrInvalidKeyFile
	}

	masterKey := &MasterKey{}

	// Salt
	masterKey.Params.Salt = data[:16]
	data = data[16:]

	// Time
	masterKey.Params.Time = binary.LittleEndian.Uint32(data[:4])
	data = data[4:]

	// Memory
	masterKey.Params.Memory = binary.LittleEndian.Uint32(data[:4])
	data = data[4:]

	// Threads
	masterKey.Params.Threads = data[0]
	data = data[1:]

	// KeyLength
	masterKey.Params.KeyLength = binary.LittleEndian.Uint32(data[:4])
	data = data[4:]

	// Nonce
	masterKey.Nonce = data[:24]
	data = data[24:]

	// Encrypted key (оставшиеся данные)
	masterKey.EncryptedKey = data

	return masterKey, nil
}

// GetPassword интерактивно запрашивает пароль у пользователя
func GetPassword(prompt string) ([]byte, error) {
	fmt.Print(prompt)
	password, err := term.ReadPassword(int(os.Stdin.Fd()))
	fmt.Println() // Переход на новую строку после ввода пароля

	if err != nil {
		return nil, fmt.Errorf("failed to read password: %w", err)
	}

	if len(password) == 0 {
		return nil, errors.New("password cannot be empty")
	}

	return password, nil
}

// VerifyPassword проверяет пароль и возвращает мастер-ключ
func VerifyPassword(keyPath string) ([]byte, error) {
	// Загружаем зашифрованный мастер-ключ
	masterKey, err := LoadMasterKey(keyPath)
	if err != nil {
		return nil, err
	}

	// Запрашиваем пароль
	password, err := GetPassword("Enter master password: ")
	if err != nil {
		return nil, err
	}
	defer clearMemory(password)

	// Расшифровываем мастер-ключ
	decryptedKey, err := DecryptMasterKey(masterKey, password)
	if err != nil {
		return nil, err
	}

	return decryptedKey, nil
}

// ChangePassword меняет мастер-пароль
func ChangePassword(keyPath string) error {
	fmt.Println("Changing master password...")

	// Проверяем старый пароль
	fmt.Println("Verify current password:")
	oldPassword, err := GetPassword("Current password: ")
	if err != nil {
		return err
	}
	defer clearMemory(oldPassword)

	// Загружаем и расшифровываем старый ключ
	masterKey, err := LoadMasterKey(keyPath)
	if err != nil {
		return err
	}

	decryptedKey, err := DecryptMasterKey(masterKey, oldPassword)
	if err != nil {
		return ErrInvalidPassword
	}
	defer clearMemory(decryptedKey)

	// Запрашиваем новый пароль
	fmt.Println("\nEnter new password:")
	newPassword, err := GetPassword("New password: ")
	if err != nil {
		return err
	}
	defer clearMemory(newPassword)

	newPassword2, err := GetPassword("Confirm new password: ")
	if err != nil {
		return err
	}
	defer clearMemory(newPassword2)

	// Проверяем совпадение паролей
	if string(newPassword) != string(newPassword2) {
		return errors.New("passwords do not match")
	}

	// Создаем новый мастер-ключ с тем же ключом, но новым паролем
	salt, err := GenerateSalt()
	if err != nil {
		return err
	}

	params := DefaultMasterKeyParams()
	params.Salt = salt

	derivedKey, err := DeriveKeyFromPassword(newPassword, params)
	if err != nil {
		return err
	}
	defer clearMemory(derivedKey)

	nonce := make([]byte, 24)
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return fmt.Errorf("failed to generate nonce: %w", err)
	}

	var nonceArr [24]byte
	var keyArr [32]byte
	copy(nonceArr[:], nonce)
	copy(keyArr[:], derivedKey)

	encryptedKey := secretbox.Seal(nil, decryptedKey, &nonceArr, &keyArr)

	newMasterKey := &MasterKey{
		Params:       params,
		EncryptedKey: encryptedKey,
		Nonce:        nonce,
	}

	// Сохраняем новый ключ
	return SaveMasterKey(newMasterKey, keyPath)
}

// clearMemory затирает чувствительные данные из памяти
func clearMemory(data []byte) {
	for i := range data {
		data[i] = 0
	}
}

// FormatTimeEstimation возвращает оценку времени для параметров Argon2
func FormatTimeEstimation(params MasterKeyParams) string {
	// Ориентировочная оценка (зависит от оборудования)
	baseTime := float64(params.Time) * float64(params.Memory) / (1024 * 1024) // МБ
	cpuFactor := float64(params.Threads)

	estimatedSeconds := baseTime / cpuFactor * 0.1

	if estimatedSeconds < 1 {
		return "мгновенно"
	} else if estimatedSeconds < 60 {
		return fmt.Sprintf("~%.0f секунд", estimatedSeconds)
	} else {
		return fmt.Sprintf("~%.1f минут", estimatedSeconds/60)
	}
}
