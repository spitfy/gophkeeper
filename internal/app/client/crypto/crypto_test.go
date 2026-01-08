package crypto

import (
	"os"
	"testing"
)

func TestMasterKeyManager(t *testing.T) {
	// Тест 1: Генерация мастер-ключа
	mgr, err := NewMasterKeyManager("test_master.key")
	if err != nil {
		t.Fatalf("Ошибка создания менеджера: %v", err)
	}
	defer os.Remove("test_master.key")

	// Тест 2: Генерация ключа
	err = mgr.GenerateMasterKey("testpassword123")
	if err != nil {
		t.Fatalf("Ошибка генерации ключа: %v", err)
	}

	// Тест 3: Проверка инициализации
	if !mgr.IsInitialized() {
		t.Error("Менеджер должен быть инициализирован")
	}

	// Тест 4: Блокировка ключа
	mgr.Lock()
	if !mgr.IsLocked() {
		t.Error("Ключ должен быть заблокирован")
	}

	// Тест 5: Разблокировка ключа
	err = mgr.UnlockMasterKey("testpassword123")
	if err != nil {
		t.Fatalf("Ошибка разблокировки: %v", err)
	}

	if mgr.IsLocked() {
		t.Error("Ключ должен быть разблокирован")
	}

	// Тест 6: Шифрование и расшифровка данных
	plaintext := []byte("Секретные данные для тестирования")
	ciphertext, err := mgr.EncryptData(plaintext)
	if err != nil {
		t.Fatalf("Ошибка шифрования: %v", err)
	}

	decrypted, err := mgr.DecryptData(ciphertext)
	if err != nil {
		t.Fatalf("Ошибка расшифровки: %v", err)
	}

	if string(decrypted) != string(plaintext) {
		t.Error("Расшифрованные данные не совпадают с оригиналом")
	}

	// Тест 7: Смена пароля
	err = mgr.ChangeMasterPassword("testpassword123", "newpassword456")
	if err != nil {
		t.Fatalf("Ошибка смены пароля: %v", err)
	}

	// Тест 8: Разблокировка новым паролем
	mgr.Lock()
	err = mgr.UnlockMasterKey("newpassword456")
	if err != nil {
		t.Fatalf("Ошибка разблокировки новым паролем: %v", err)
	}

	// Тест 9: Проверка шифрования после смены пароля
	decryptedAfterChange, err := mgr.DecryptData(ciphertext)
	if err != nil {
		t.Fatalf("Ошибка расшифровки после смены пароля: %v", err)
	}

	if string(decryptedAfterChange) != string(plaintext) {
		t.Error("Данные не расшифровываются после смены пароля")
	}
}

func TestPasswordStrength(t *testing.T) {
	tests := []struct {
		password string
		expected PasswordStrength
	}{
		{"123", PasswordWeak},
		{"password", PasswordWeak},
		{"Password1", PasswordMedium},
		{"Password123", PasswordMedium},
		{"StrongPass123!", PasswordStrong},
		{"Very$tr0ngP@ssw0rd!", PasswordStrong},
	}

	for _, test := range tests {
		result := CheckPasswordStrength(test.password)
		if result != test.expected {
			t.Errorf("Для пароля '%s' ожидалось %v, получено %v",
				MaskSensitiveData(test.password), test.expected, result)
		}
	}
}

func TestGenerateSecurePassword(t *testing.T) {
	// Тест генерации пароля без символов
	password, err := GenerateSecurePassword(12, false)
	if err != nil {
		t.Fatalf("Ошибка генерации пароля: %v", err)
	}

	if len(password) != 12 {
		t.Errorf("Неправильная длина пароля: ожидалось 12, получено %d", len(password))
	}

	strength := CheckPasswordStrength(password)
	if strength == PasswordWeak {
		t.Error("Сгенерированный пароль должен быть хотя бы средней сложности")
	}

	// Тест генерации пароля с символами
	passwordWithSymbols, err := GenerateSecurePassword(16, true)
	if err != nil {
		t.Fatalf("Ошибка генерации пароля с символами: %v", err)
	}

	if len(passwordWithSymbols) != 16 {
		t.Errorf("Неправильная длина пароля с символами: ожидалось 16, получено %d", len(passwordWithSymbols))
	}
}
