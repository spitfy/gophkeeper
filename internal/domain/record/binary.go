package record

import (
	"encoding/json"
	"fmt"
	"mime"
	"path/filepath"
	"strings"
)

// BinaryData - бинарные данные (до шифрования)
// Внимание: сами бинарные данные будут храниться как []byte
type BinaryData struct {
	Data        []byte `json:"-"`
	Filename    string `json:"filename"`
	ContentType string `json:"content_type"`
	Size        int64  `json:"size"`
}

func (b *BinaryData) GetType() RecType {
	return RecTypeBinary
}

func (b *BinaryData) Validate() error {
	if len(b.Data) == 0 {
		return fmt.Errorf("data is required")
	}

	if strings.TrimSpace(b.Filename) == "" {
		return fmt.Errorf("filename is required")
	}

	// Проверка размера файла
	if b.Size <= 0 {
		b.Size = int64(len(b.Data))
	}

	if b.Size > 100*1024*1024 { // 100 MB
		return fmt.Errorf("file too large (max 100MB)")
	}

	// Определяем ContentType если не указан
	if b.ContentType == "" {
		ext := strings.ToLower(filepath.Ext(b.Filename))
		b.ContentType = mime.TypeByExtension(ext)
		if b.ContentType == "" {
			b.ContentType = "application/octet-stream"
		}
	}

	return nil
}

func (b *BinaryData) ToJSON() ([]byte, error) {
	// Не сериализуем сами данные в JSON, они будут храниться отдельно
	return json.Marshal(map[string]interface{}{
		"filename":     b.Filename,
		"content_type": b.ContentType,
		"size":         b.Size,
	})
}

func (b *BinaryData) FromJSON(data []byte) error {
	var temp struct {
		Filename    string `json:"filename"`
		ContentType string `json:"content_type"`
		Size        int64  `json:"size"`
	}
	if err := json.Unmarshal(data, &temp); err != nil {
		return err
	}
	b.Filename = temp.Filename
	b.ContentType = temp.ContentType
	b.Size = temp.Size
	return nil
}

// BinaryMeta - метаданные бинарного файла
type BinaryMeta struct {
	Title        string           `json:"title"`
	Category     string           `json:"category,omitempty"`
	Tags         []string         `json:"tags,omitempty"`
	Description  string           `json:"description,omitempty"`
	OriginalHash string           `json:"original_hash,omitempty"` // SHA256 оригинального файла
	Compression  CompressionInfo  `json:"compression,omitempty"`
	Encryption   EncryptionInfo   `json:"encryption,omitempty"`
	Dimensions   *ImageDimensions `json:"dimensions,omitempty"` // Для изображений
	Duration     *float64         `json:"duration,omitempty"`   // Для аудио/видео (в секундах)
	CustomData   json.RawMessage  `json:"custom_data,omitempty"`
}

type CompressionInfo struct {
	Algorithm string  `json:"algorithm,omitempty"` // gzip, deflate, etc.
	Level     int     `json:"level,omitempty"`
	Ratio     float64 `json:"ratio,omitempty"` // Коэффициент сжатия
}

type EncryptionInfo struct {
	Algorithm string `json:"algorithm,omitempty"` // AES-256-GCM, ChaCha20-Poly1305
	KeyID     string `json:"key_id,omitempty"`
	IV        string `json:"iv,omitempty"`
}

type ImageDimensions struct {
	Width  int `json:"width"`
	Height int `json:"height"`
}

func (m *BinaryMeta) Validate() error {
	if strings.TrimSpace(m.Title) == "" {
		return fmt.Errorf("title is required")
	}

	// Валидация для изображений
	if m.Dimensions != nil {
		if m.Dimensions.Width <= 0 || m.Dimensions.Height <= 0 {
			return fmt.Errorf("invalid image dimensions")
		}
	}

	return nil
}

func (m *BinaryMeta) ToJSON() ([]byte, error) {
	return json.Marshal(m)
}

func (m *BinaryMeta) FromJSON(data []byte) error {
	return json.Unmarshal(data, m)
}
