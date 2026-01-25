package crypto

import (
	"crypto/hmac"
	"crypto/pbkdf2"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"math/big"
	"strings"
	"unicode"

	"golang.org/x/crypto/argon2"
)

// PasswordStrength проверяет сложность пароля
type PasswordStrength int

const (
	PasswordWeak PasswordStrength = iota
	PasswordMedium
	PasswordStrong
)

// GenerateSecurePassword генерирует безопасный пароль
func GenerateSecurePassword(length int, useSymbols bool) (string, error) {
	if length < 8 {
		return "", fmt.Errorf("длина пароля должна быть не менее 8 символов")
	}

	var charset string
	letters := "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"
	digits := "0123456789"
	symbols := "!@#$%^&*()-_=+[]{}|;:,.<>?"

	// Гарантируем наличие разных типов символов
	charset = letters + digits
	if useSymbols {
		charset += symbols
	}

	password := make([]byte, length)

	// Гарантируем хотя бы одну заглавную букву
	password[0] = letters[26+secureRandInt(len(letters)/2)] // Вторая половина - заглавные

	// Гарантируем хотя бы одну цифру
	password[1] = digits[secureRandInt(len(digits))]

	// Гарантируем хотя бы один символ, если нужно
	if useSymbols {
		password[2] = symbols[secureRandInt(len(symbols))]
	}

	// Заполняем остальные позиции случайными символами
	for i := 3; i < length; i++ {
		password[i] = charset[secureRandInt(len(charset))]
	}

	// Перемешиваем пароль
	shufflePassword(password)

	return string(password), nil
}

// CheckPasswordStrength проверяет сложность пароля
func CheckPasswordStrength(password string) PasswordStrength {
	var (
		hasUpper   bool
		hasLower   bool
		hasDigit   bool
		hasSpecial bool
		length     = len(password)
	)

	for _, char := range password {
		switch {
		case unicode.IsUpper(char):
			hasUpper = true
		case unicode.IsLower(char):
			hasLower = true
		case unicode.IsDigit(char):
			hasDigit = true
		case unicode.IsPunct(char) || unicode.IsSymbol(char):
			hasSpecial = true
		}
	}

	score := 0
	if length >= 8 {
		score++
	}
	if length >= 12 {
		score++
	}
	if hasUpper && hasLower {
		score++
	}
	if hasDigit {
		score++
	}
	if hasSpecial {
		score++
	}

	switch {
	case score >= 5:
		return PasswordStrong
	case score >= 3:
		return PasswordMedium
	default:
		return PasswordWeak
	}
}

// GenerateRandomBytes генерирует криптографически безопасные случайные байты
func GenerateRandomBytes(n int) ([]byte, error) {
	b := make([]byte, n)
	_, err := rand.Read(b)
	if err != nil {
		return nil, fmt.Errorf("ошибка генерации случайных байт: %w", err)
	}
	return b, nil
}

// GenerateArgon2Key генерирует ключ с использованием Argon2
func GenerateArgon2Key(password string, salt []byte) []byte {
	return argon2.IDKey([]byte(password), salt, argon2Time, argon2Memory, argon2Threads, argon2KeyLen)
}

// EncodeBase64 кодирует данные в base64
func EncodeBase64(data []byte) string {
	return base64.StdEncoding.EncodeToString(data)
}

// DecodeBase64 декодирует данные из base64
func DecodeBase64(s string) ([]byte, error) {
	return base64.StdEncoding.DecodeString(s)
}

// MaskSensitiveData маскирует чувствительные данные для логов
func MaskSensitiveData(data string) string {
	if len(data) <= 4 {
		return strings.Repeat("*", len(data))
	}
	return data[:2] + strings.Repeat("*", len(data)-4) + data[len(data)-2:]
}

// secureRandInt возвращает криптографически безопасное случайное число
// nolint
func secureRandInt(max int) int {
	n, err := rand.Int(rand.Reader, big.NewInt(int64(max)))
	if err != nil {
		panic(fmt.Sprintf("ошибка генерации случайного числа: %v", err))
	}
	return int(n.Int64())
}

// shufflePassword перемешивает пароль
func shufflePassword(password []byte) {
	for i := len(password) - 1; i > 0; i-- {
		j := secureRandInt(i + 1)
		password[i], password[j] = password[j], password[i]
	}
}

// DeriveKeyFromPassword создает ключ из пароля с солью
func DeriveKeyFromPassword(password string, salt []byte, iterations int, keyLength int) ([]byte, error) {
	return pbkdf2.Key(sha256.New, password, salt, iterations, keyLength)
}

// GenerateSalt генерирует случайную соль
func GenerateSalt(length int) ([]byte, error) {
	salt := make([]byte, length)
	_, err := rand.Read(salt)
	if err != nil {
		return nil, fmt.Errorf("ошибка генерации соли: %w", err)
	}
	return salt, nil
}

// HashPassword создает хэш пароля для проверки
func HashPassword(password string) (string, error) {
	salt, err := GenerateSalt(16)
	if err != nil {
		return "", err
	}

	hash := argon2.IDKey([]byte(password), salt, argon2Time, argon2Memory, argon2Threads, 32)

	// Формат: argon2id$time$memory$threads$salt$hash
	return fmt.Sprintf("argon2id$%d$%d$%d$%s$%s",
		argon2Time,
		argon2Memory,
		argon2Threads,
		base64.RawStdEncoding.EncodeToString(salt),
		base64.RawStdEncoding.EncodeToString(hash),
	), nil
}

// VerifyPassword проверяет пароль с хэшем
func VerifyPassword(password, hashedPassword string) (bool, error) {
	parts := strings.Split(hashedPassword, "$")
	if len(parts) != 6 || parts[0] != "argon2id" {
		return false, fmt.Errorf("неверный формат хэша")
	}

	time := atoi(parts[1])
	memory := atoi(parts[2])
	threads := atoi(parts[3])

	salt, err := base64.RawStdEncoding.DecodeString(parts[4])
	if err != nil {
		return false, fmt.Errorf("ошибка декодирования соли: %w", err)
	}

	expectedHash, err := base64.RawStdEncoding.DecodeString(parts[5])
	if err != nil {
		return false, fmt.Errorf("ошибка декодирования хэша: %w", err)
	}

	actualHash := argon2.IDKey([]byte(password), salt, uint32(time), uint32(memory), uint8(threads), uint32(len(expectedHash)))

	return hmac.Equal(actualHash, expectedHash), nil
}

// Вспомогательная функция для преобразования строки в int
func atoi(s string) int {
	var result int
	for _, ch := range s {
		result = result*10 + int(ch-'0')
	}
	return result
}
