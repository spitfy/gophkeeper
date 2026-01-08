package crypto

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
)

// EncryptData шифрует данные с использованием мастер-ключа
func EncryptData(plaintext []byte, masterKey []byte) (string, error) {
	// Создаем новый cipher block
	block, err := aes.NewCipher(masterKey)
	if err != nil {
		return "", fmt.Errorf("failed to create cipher: %w", err)
	}

	// Генерируем случайный nonce для GCM
	nonce := make([]byte, 12)
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", fmt.Errorf("failed to generate nonce: %w", err)
	}

	// Создаем GCM
	aesgcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", fmt.Errorf("failed to create GCM: %w", err)
	}

	// Шифруем данные
	ciphertext := aesgcm.Seal(nil, nonce, plaintext, nil)

	// Объединяем nonce и ciphertext в один массив
	result := make([]byte, len(nonce)+len(ciphertext))
	copy(result, nonce)
	copy(result[len(nonce):], ciphertext)

	// Кодируем в base64 для удобства хранения
	return base64.StdEncoding.EncodeToString(result), nil
}

// DecryptData расшифровывает данные с использованием мастер-ключа
func DecryptData(encryptedData string, masterKey []byte) ([]byte, error) {
	// Декодируем из base64
	data, err := base64.StdEncoding.DecodeString(encryptedData)
	if err != nil {
		return nil, fmt.Errorf("failed to decode base64: %w", err)
	}

	// Проверяем минимальную длину
	if len(data) < 13 { // 12 байт nonce + минимум 1 байт данных
		return nil, errors.New("invalid encrypted data length")
	}

	// Извлекаем nonce и ciphertext
	nonce := data[:12]
	ciphertext := data[12:]

	// Создаем cipher block
	block, err := aes.NewCipher(masterKey)
	if err != nil {
		return nil, fmt.Errorf("failed to create cipher: %w", err)
	}

	// Создаем GCM
	aesgcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("failed to create GCM: %w", err)
	}

	// Расшифровываем данные
	plaintext, err := aesgcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to decrypt: %w", err)
	}

	return plaintext, nil
}
