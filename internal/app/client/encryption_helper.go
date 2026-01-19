// internal/app/client/encryption_helper.go
package client

import (
	"encoding/base64"
	"encoding/json"
	"fmt"

	"gophkeeper/internal/domain/record"
)

// encryptRecordData шифрует данные записи перед отправкой на сервер
func (a *App) encryptRecordData(data interface{}) (string, error) {
	// Проверяем, что мастер-ключ разблокирован
	if !a.IsMasterKeyUnlocked() {
		return "", fmt.Errorf("мастер-ключ заблокирован")
	}

	// Сериализуем данные в JSON
	dataJSON, err := json.Marshal(data)
	if err != nil {
		return "", fmt.Errorf("ошибка сериализации данных: %w", err)
	}

	// Шифруем данные
	encryptedData, err := a.encryptor.EncryptRecord(dataJSON)
	if err != nil {
		return "", fmt.Errorf("ошибка шифрования данных: %w", err)
	}

	// Кодируем в base64 для передачи
	return base64.StdEncoding.EncodeToString(encryptedData), nil
}

// decryptRecordData расшифровывает данные записи, полученные с сервера
func (a *App) decryptRecordData(encryptedData string, target interface{}) error {
	// Проверяем, что мастер-ключ разблокирован
	if !a.IsMasterKeyUnlocked() {
		return fmt.Errorf("мастер-ключ заблокирован")
	}

	// Декодируем из base64
	encrypted, err := base64.StdEncoding.DecodeString(encryptedData)
	if err != nil {
		return fmt.Errorf("ошибка декодирования base64: %w", err)
	}

	// Расшифровываем данные
	decryptedData, err := a.encryptor.DecryptRecord(encrypted)
	if err != nil {
		return fmt.Errorf("ошибка расшифровки данных: %w", err)
	}

	// Десериализуем JSON в целевую структуру
	if err := json.Unmarshal(decryptedData, target); err != nil {
		return fmt.Errorf("ошибка десериализации данных: %w", err)
	}

	return nil
}

// prepareEncryptedRecord подготавливает зашифрованную запись для отправки на сервер
func (a *App) prepareEncryptedRecord(recType record.RecType, data interface{}, meta json.RawMessage) (GenericRecordRequest, error) {
	// Шифруем данные
	encryptedData, err := a.encryptRecordData(data)
	if err != nil {
		return GenericRecordRequest{}, err
	}

	return GenericRecordRequest{
		Type: recType,
		Data: encryptedData,
		Meta: meta,
	}, nil
}
