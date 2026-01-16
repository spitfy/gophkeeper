package record

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

// TextData - текстовые данные (до шифрования)
type TextData struct {
	Content string `json:"content"`
}

func (t *TextData) GetType() RecType {
	return RecTypeText
}

func (t *TextData) Validate() error {
	if strings.TrimSpace(t.Content) == "" {
		return fmt.Errorf("content is required")
	}
	// Ограничение на размер текста
	if len(t.Content) > 10*1024*1024 { // 10 MB
		return fmt.Errorf("text content too large (max 10MB)")
	}
	return nil
}

func (t *TextData) ToJSON() ([]byte, error) {
	return json.Marshal(t)
}

func (t *TextData) FromJSON(data []byte) error {
	return json.Unmarshal(data, t)
}

// TextMeta - метаданные текста
type TextMeta struct {
	Title       string          `json:"title"`
	Category    string          `json:"category,omitempty"`
	Tags        []string        `json:"tags,omitempty"`
	Format      string          `json:"format,omitempty"`   // plain, markdown, html, json, xml
	Language    string          `json:"language,omitempty"` // ru, en, etc.
	IsSensitive bool            `json:"is_sensitive,omitempty"`
	ExpiresAt   *time.Time      `json:"expires_at,omitempty"`
	WordCount   int             `json:"word_count,omitempty"`
	CharsCount  int             `json:"chars_count,omitempty"`
	Preview     string          `json:"preview,omitempty"` // Первые 100 символов
	CustomData  json.RawMessage `json:"custom_data,omitempty"`
}

func (m *TextMeta) Validate() error {
	if strings.TrimSpace(m.Title) == "" {
		return fmt.Errorf("title is required")
	}

	// Валидация формата
	validFormats := map[string]bool{
		"plain": true, "markdown": true, "html": true,
		"json": true, "xml": true, "yaml": true,
	}

	if m.Format != "" && !validFormats[m.Format] {
		return fmt.Errorf("invalid format. Allowed: plain, markdown, html, json, xml, yaml")
	}

	return nil
}

func (m *TextMeta) ToJSON() ([]byte, error) {
	// Автоматически считаем статистику если не заполнено
	if m.WordCount == 0 && m.Preview == "" {
		// Эти поля можно заполнить позже при обработке
	}
	return json.Marshal(m)
}

func (m *TextMeta) FromJSON(data []byte) error {
	return json.Unmarshal(data, m)
}
