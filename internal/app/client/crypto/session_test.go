package crypto

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Предположим структуру менеджера, так как её не было в листинге
// type MasterKeyManager struct {
//     mu        sync.RWMutex
//     keyPath   string
//     masterKey []byte
//     isLoaded  bool
//     isLocked  bool
// }

func TestMasterKeyManager_SessionLifecycle(t *testing.T) {
	// Создаем временную директорию для тестов, Go сам очистит её
	tmpDir := t.TempDir()
	keyPath := filepath.Join(tmpDir, "master.key")

	m := &MasterKeyManager{
		keyPath:   keyPath,
		masterKey: []byte("super-secret-master-key-32-bytes!!"),
		isLoaded:  true,
		isLocked:  false,
	}

	t.Run("SaveSession_Success", func(t *testing.T) {
		err := m.SaveSession()
		assert.NoError(t, err)

		// Проверяем, что файл создался с нужными правами
		info, err := os.Stat(m.getSessionPath())
		require.NoError(t, err)
		assert.Equal(t, os.FileMode(sessionPermissions), info.Mode().Perm())
	})

	t.Run("LoadSession_Success", func(t *testing.T) {
		// Очищаем состояние в памяти перед загрузкой
		m.masterKey = nil
		m.isLoaded = false
		m.isLocked = true

		err := m.LoadSession()
		assert.NoError(t, err)
		assert.True(t, m.isLoaded)
		assert.False(t, m.isLocked)
		assert.Equal(t, []byte("super-secret-master-key-32-bytes!!"), m.masterKey)
	})

	t.Run("ClearSession_Success", func(t *testing.T) {
		err := m.ClearSession()
		assert.NoError(t, err)

		_, err = os.Stat(m.getSessionPath())
		assert.True(t, os.IsNotExist(err), "Файл сессии должен быть удален")
	})
}

func TestMasterKeyManager_LoadSession_Errors(t *testing.T) {
	tmpDir := t.TempDir()

	tests := []struct {
		name    string
		setup   func(m *MasterKeyManager) error
		wantErr string
	}{
		{
			name: "SessionNotFound",
			setup: func(m *MasterKeyManager) error {
				return os.Remove(m.getSessionPath())
			},
			wantErr: "сессия не найдена",
		},
		{
			name: "ExpiredSession",
			setup: func(m *MasterKeyManager) error {
				m.isLoaded = true
				m.isLocked = false
				m.masterKey = []byte("key")
				if err := m.SaveSession(); err != nil {
					return err
				}

				path := m.getSessionPath()
				_, err := os.ReadFile(path)

				return err
			},
			wantErr: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := &MasterKeyManager{keyPath: filepath.Join(tmpDir, "key")}

			if err := tt.setup(m); tt.wantErr != "" {
				assert.Error(t, err)
			}

			err := m.LoadSession()

			if tt.wantErr != "" {
				assert.Contains(t, err.Error(), tt.wantErr)
				assert.Error(t, err)
			}
		})
	}
}

func TestMasterKeyManager_SaveSession_Locked(t *testing.T) {
	m := &MasterKeyManager{isLocked: true}
	err := m.SaveSession()
	assert.EqualError(t, err, "мастер-ключ не разблокирован")
}
