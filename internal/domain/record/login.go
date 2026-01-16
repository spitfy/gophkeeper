package record

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
)

// LoginData - данные логина (до шифрования)
type LoginData struct {
	Username string `json:"username"`
	Password string `json:"password"`
	Notes    string `json:"notes,omitempty"`
}

func (l *LoginData) GetType() RecType {
	return RecTypeLogin
}

func (l *LoginData) Validate() error {
	if strings.TrimSpace(l.Username) == "" {
		return fmt.Errorf("username is required")
	}
	if strings.TrimSpace(l.Password) == "" {
		return fmt.Errorf("password is required")
	}
	return nil
}

func (l *LoginData) ToJSON() ([]byte, error) {
	return json.Marshal(l)
}

func (l *LoginData) FromJSON(data []byte) error {
	return json.Unmarshal(data, l)
}

// LoginMeta - метаданные логина
type LoginMeta struct {
	Title      string          `json:"title"`
	Resource   string          `json:"resource"`
	Category   string          `json:"category,omitempty"`
	Tags       []string        `json:"tags,omitempty"`
	Favicon    string          `json:"favicon,omitempty"`
	AutoLogin  bool            `json:"auto_login,omitempty"`
	TwoFA      bool            `json:"two_fa,omitempty"`
	TwoFAType  string          `json:"two_fa_type,omitempty"` // totp, sms, email
	ExpiresAt  *string         `json:"expires_at,omitempty"`  // ISO 8601
	CustomData json.RawMessage `json:"custom_data,omitempty"`
}

func (m *LoginMeta) Validate() error {
	if strings.TrimSpace(m.Title) == "" {
		return fmt.Errorf("title is required")
	}

	if strings.TrimSpace(m.Resource) == "" {
		return fmt.Errorf("resource is required")
	}

	// Валидация URL если есть favicon
	if m.Favicon != "" {
		matched, _ := regexp.MatchString(`^(https?://|data:image/)`, m.Favicon)
		if !matched {
			return fmt.Errorf("favicon must be a valid URL or data URI")
		}
	}

	return nil
}

func (m *LoginMeta) ToJSON() ([]byte, error) {
	return json.Marshal(m)
}

func (m *LoginMeta) FromJSON(data []byte) error {
	return json.Unmarshal(data, m)
}
