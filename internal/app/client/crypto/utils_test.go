package crypto

import (
	"crypto/hmac"
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCheckPasswordStrength(t *testing.T) {
	tests := []struct {
		password string
		strength PasswordStrength
	}{
		{"weak", PasswordWeak},
		{"Abc1", PasswordWeak},     // короткий
		{"abcdefgh", PasswordWeak}, // нет upper/digit
		{"Abcdefgh123", PasswordMedium},
		{"Abcdefgh123!", PasswordStrong},
		{"A1!23456789", PasswordMedium}, // length 10, upper, digit, special
	}

	for _, tt := range tests {
		t.Run(fmt.Sprintf("password_%s", strings.ReplaceAll(tt.password, string(rune(0)), "_")), func(t *testing.T) {
			got := CheckPasswordStrength(tt.password)
			assert.Equal(t, tt.strength, got)
		})
	}
}

func TestEncodeBase64_DecodeBase64(t *testing.T) {
	data := []byte("hello world")
	encoded := EncodeBase64(data)
	assert.NotEmpty(t, encoded)

	decoded, err := DecodeBase64(encoded)
	require.NoError(t, err)
	assert.Equal(t, data, decoded)

	// Invalid base64
	_, err = DecodeBase64("invalid")
	require.Error(t, err)
}

func TestDeriveKeyFromPassword(t *testing.T) {
	password := "testpass"
	salt := []byte("saltysalt")
	key, err := DeriveKeyFromPassword(password, salt, 1000, 32)
	require.NoError(t, err)
	assert.Len(t, key, 32)

	// Тот же ввод -> тот же ключ
	key2, err := DeriveKeyFromPassword(password, salt, 1000, 32)
	require.NoError(t, err)
	assert.True(t, hmac.Equal(key, key2))
}

func TestHashPassword_VerifyPassword(t *testing.T) {
	password := "StrongPass123!"

	hashed, err := HashPassword(password)
	require.NoError(t, err)

	ok, err := VerifyPassword(password, hashed)
	assert.NoError(t, err)
	assert.True(t, ok)

	// Неправильный пароль
	ok, err = VerifyPassword("wrong", hashed)
	assert.NoError(t, err)
	assert.False(t, ok)

	// Неверный формат
	ok, err = VerifyPassword("test", "invalid$format")
	assert.Error(t, err)
	assert.False(t, ok)
}

func TestAtoi(t *testing.T) {
	assert.Equal(t, 123, atoi("123"))
	assert.Equal(t, 0, atoi("0"))
	assert.Equal(t, 42, atoi("42"))
}
