package crypto

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
)

// ClearMemory затирает чувствительные данные из памяти (публичный алиас)
func ClearMemory(data []byte) {
	clearMemory(data)
}

// GenerateRandomBytes генерирует криптографически безопасные случайные байты
func GenerateRandomBytes(size int) ([]byte, error) {
	bytes := make([]byte, size)
	if _, err := io.ReadFull(rand.Reader, bytes); err != nil {
		return nil, fmt.Errorf("failed to generate random bytes: %w", err)
	}
	return bytes, nil
}

// GenerateRandomHex генерирует случайную hex строку
func GenerateRandomHex(size int) (string, error) {
	bytes, err := GenerateRandomBytes(size)
	if err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes), nil
}

// HashPassword создает безопасный хеш пароля (альтернатива для хранения паролей приложений)
func HashPassword(password []byte) (string, error) {
	salt, err := GenerateRandomBytes(16)
	if err != nil {
		return "", err
	}

	params := DefaultMasterKeyParams()
	params.Salt = salt
	params.Time = 1 // Меньше итераций для проверки паролей приложений

	hash, err := DeriveKeyFromPassword(password, params)
	if err != nil {
		return "", err
	}

	// Сохраняем соль и хеш вместе
	result := make([]byte, len(salt)+len(hash))
	copy(result, salt)
	copy(result[len(salt):], hash)

	return base64.StdEncoding.EncodeToString(result), nil
}

// VerifyPasswordHash проверяет пароль против хеша
func VerifyPasswordHash(password []byte, hash string) (bool, error) {
	decoded, err := base64.StdEncoding.DecodeString(hash)
	if err != nil {
		return false, err
	}

	if len(decoded) < 48 { // 16 байт соль + 32 байта хеш
		return false, errors.New("invalid hash format")
	}

	salt := decoded[:16]
	storedHash := decoded[16:]

	params := DefaultMasterKeyParams()
	params.Salt = salt
	params.Time = 1

	computedHash, err := DeriveKeyFromPassword(password, params)
	if err != nil {
		return false, err
	}

	// Сравнение с постоянным временем выполнения
	if len(computedHash) != len(storedHash) {
		return false, nil
	}

	equal := true
	for i := 0; i < len(computedHash); i++ {
		equal = equal && (computedHash[i] == storedHash[i])
	}

	return equal, nil
}
