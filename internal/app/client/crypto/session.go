// internal/app/client/crypto/session.go
package crypto

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"
)

const (
	sessionTimeout     = 15 * time.Minute // Таймаут сессии
	sessionPermissions = 0600
)

// Session хранит информацию о разблокированной сессии
type Session struct {
	SessionKey []byte    `json:"session_key"` // Зашифрованный мастер-ключ
	ExpiresAt  time.Time `json:"expires_at"`
	CreatedAt  time.Time `json:"created_at"`
}

// SaveSession сохраняет сессию с разблокированным ключом
func (m *MasterKeyManager) SaveSession() error {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if !m.isLoaded || m.isLocked || len(m.masterKey) == 0 {
		return fmt.Errorf("мастер-ключ не разблокирован")
	}

	// Генерируем случайный ключ сессии для шифрования
	sessionKey := make([]byte, 32)
	if _, err := io.ReadFull(rand.Reader, sessionKey); err != nil {
		return fmt.Errorf("ошибка генерации ключа сессии: %w", err)
	}

	// Шифруем мастер-ключ ключом сессии
	encryptedMasterKey, err := encryptWithKey(sessionKey, m.masterKey)
	if err != nil {
		return fmt.Errorf("ошибка шифрования мастер-ключа: %w", err)
	}

	session := Session{
		SessionKey: encryptedMasterKey,
		ExpiresAt:  time.Now().Add(sessionTimeout),
		CreatedAt:  time.Now(),
	}

	// Сериализуем сессию
	data, err := json.MarshalIndent(session, "", "  ")
	if err != nil {
		return fmt.Errorf("ошибка сериализации сессии: %w", err)
	}

	// Шифруем данные сессии ключом сессии
	encryptedSession, err := encryptWithKey(sessionKey, data)
	if err != nil {
		return fmt.Errorf("ошибка шифрования сессии: %w", err)
	}

	// Сохраняем ключ сессии и зашифрованную сессию
	sessionData := struct {
		Key  string `json:"key"`
		Data string `json:"data"`
	}{
		Key:  hex.EncodeToString(sessionKey),
		Data: hex.EncodeToString(encryptedSession),
	}

	sessionJSON, err := json.MarshalIndent(sessionData, "", "  ")
	if err != nil {
		return fmt.Errorf("ошибка сериализации: %w", err)
	}

	sessionPath := m.getSessionPath()
	if err := os.WriteFile(sessionPath, sessionJSON, sessionPermissions); err != nil {
		return fmt.Errorf("ошибка сохранения сессии: %w", err)
	}

	return nil
}

// LoadSession загружает сессию и восстанавливает мастер-ключ
func (m *MasterKeyManager) LoadSession() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	sessionPath := m.getSessionPath()

	// Проверяем существование файла сессии
	if _, err := os.Stat(sessionPath); os.IsNotExist(err) {
		return fmt.Errorf("сессия не найдена")
	}

	// Читаем файл сессии
	sessionJSON, err := os.ReadFile(sessionPath)
	if err != nil {
		return fmt.Errorf("ошибка чтения сессии: %w", err)
	}

	var sessionData struct {
		Key  string `json:"key"`
		Data string `json:"data"`
	}

	if err := json.Unmarshal(sessionJSON, &sessionData); err != nil {
		return fmt.Errorf("ошибка декодирования сессии: %w", err)
	}

	// Декодируем ключ сессии
	sessionKey, err := hex.DecodeString(sessionData.Key)
	if err != nil {
		return fmt.Errorf("ошибка декодирования ключа сессии: %w", err)
	}

	// Декодируем зашифрованную сессию
	encryptedSession, err := hex.DecodeString(sessionData.Data)
	if err != nil {
		return fmt.Errorf("ошибка декодирования данных сессии: %w", err)
	}

	// Расшифровываем сессию
	sessionBytes, err := decryptWithKey(sessionKey, encryptedSession)
	if err != nil {
		// Сессия повреждена, удаляем её
		os.Remove(sessionPath)
		return fmt.Errorf("ошибка расшифровки сессии: %w", err)
	}

	var session Session
	if err := json.Unmarshal(sessionBytes, &session); err != nil {
		os.Remove(sessionPath)
		return fmt.Errorf("ошибка декодирования сессии: %w", err)
	}

	// Проверяем срок действия сессии
	if time.Now().After(session.ExpiresAt) {
		os.Remove(sessionPath)
		return fmt.Errorf("сессия истекла")
	}

	// Расшифровываем мастер-ключ
	masterKey, err := decryptWithKey(sessionKey, session.SessionKey)
	if err != nil {
		os.Remove(sessionPath)
		return fmt.Errorf("ошибка расшифровки мастер-ключа: %w", err)
	}

	// Восстанавливаем мастер-ключ в памяти
	m.masterKey = masterKey
	m.isLoaded = true
	m.isLocked = false

	return nil
}

// ClearSession удаляет файл сессии
func (m *MasterKeyManager) ClearSession() error {
	sessionPath := m.getSessionPath()
	if err := os.Remove(sessionPath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("ошибка удаления сессии: %w", err)
	}
	return nil
}

// getSessionPath возвращает путь к файлу сессии
func (m *MasterKeyManager) getSessionPath() string {
	dir := filepath.Dir(m.keyPath)
	return filepath.Join(dir, ".session")
}
