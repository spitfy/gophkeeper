// internal/app/client/crypto/encryption.go
package crypto

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
)

// RecordEncryptor отвечает за шифрование данных записей
type RecordEncryptor struct {
	masterKeyManager *MasterKeyManager
}

// NewRecordEncryptor создает новый шифровальщик записей
func NewRecordEncryptor(masterKeyManager *MasterKeyManager) *RecordEncryptor {
	return &RecordEncryptor{
		masterKeyManager: masterKeyManager,
	}
}

// EncryptRecord шифрует данные записи
func (e *RecordEncryptor) EncryptRecord(plaintext []byte) ([]byte, error) {
	if e.masterKeyManager == nil {
		return nil, fmt.Errorf("мастер-ключ не инициализирован")
	}

	return e.masterKeyManager.EncryptData(plaintext)
}

// DecryptRecord расшифровывает данные записи
func (e *RecordEncryptor) DecryptRecord(ciphertext []byte) ([]byte, error) {
	if e.masterKeyManager == nil {
		return nil, fmt.Errorf("мастер-ключ не инициализирован")
	}

	return e.masterKeyManager.DecryptData(ciphertext)
}

// EncryptField шифрует отдельное поле записи
func (e *RecordEncryptor) EncryptField(fieldName string, value string) (string, error) {
	if e.masterKeyManager == nil {
		return "", fmt.Errorf("мастер-ключ не инициализирован")
	}

	encrypted, err := e.masterKeyManager.EncryptData([]byte(value))
	if err != nil {
		return "", err
	}

	return hex.EncodeToString(encrypted), nil
}

// DecryptField расшифровывает отдельное поле записи
func (e *RecordEncryptor) DecryptField(fieldName string, encryptedHex string) (string, error) {
	if e.masterKeyManager == nil {
		return "", fmt.Errorf("мастер-ключ не инициализирован")
	}

	encrypted, err := hex.DecodeString(encryptedHex)
	if err != nil {
		return "", fmt.Errorf("ошибка декодирования hex: %w", err)
	}

	decrypted, err := e.masterKeyManager.DecryptData(encrypted)
	if err != nil {
		return "", err
	}

	return string(decrypted), nil
}

// GenerateHMAC создает HMAC для проверки целостности данных
func (e *RecordEncryptor) GenerateHMAC(data []byte) (string, error) {
	if e.masterKeyManager == nil {
		return "", fmt.Errorf("мастер-ключ не инициализирован")
	}

	// Используем мастер-ключ для создания HMAC
	mac := hmac.New(sha256.New, e.masterKeyManager.getRawKey())
	mac.Write(data)
	return hex.EncodeToString(mac.Sum(nil)), nil
}

// VerifyHMAC проверяет HMAC для данных
func (e *RecordEncryptor) VerifyHMAC(data []byte, expectedHMAC string) (bool, error) {
	if e.masterKeyManager == nil {
		return false, fmt.Errorf("мастер-ключ не инициализирован")
	}

	actualHMAC, err := e.GenerateHMAC(data)
	if err != nil {
		return false, err
	}

	return hmac.Equal([]byte(actualHMAC), []byte(expectedHMAC)), nil
}

// EncryptMetadata шифрует метаданные записи
func (e *RecordEncryptor) EncryptMetadata(metadata map[string]string) (map[string]string, error) {
	if e.masterKeyManager == nil {
		return nil, fmt.Errorf("мастер-ключ не инициализирован")
	}

	encryptedMetadata := make(map[string]string)
	for key, value := range metadata {
		encrypted, err := e.EncryptField(key, value)
		if err != nil {
			return nil, fmt.Errorf("ошибка шифрования поля %s: %w", key, err)
		}
		encryptedMetadata[key] = encrypted
	}

	return encryptedMetadata, nil
}

// DecryptMetadata расшифровывает метаданные записи
func (e *RecordEncryptor) DecryptMetadata(encryptedMetadata map[string]string) (map[string]string, error) {
	if e.masterKeyManager == nil {
		return nil, fmt.Errorf("мастер-ключ не инициализирован")
	}

	metadata := make(map[string]string)
	for key, encryptedValue := range encryptedMetadata {
		decrypted, err := e.DecryptField(key, encryptedValue)
		if err != nil {
			return nil, fmt.Errorf("ошибка расшифровки поля %s: %w", key, err)
		}
		metadata[key] = decrypted
	}

	return metadata, nil
}

// getRawKey возвращает сырой мастер-ключ (только для внутреннего использования)
func (m *MasterKeyManager) getRawKey() []byte {
	m.mu.RLock()
	defer m.mu.RUnlock()

	// Создаем копию ключа для безопасности
	keyCopy := make([]byte, len(m.masterKey))
	copy(keyCopy, m.masterKey)
	return keyCopy
}
