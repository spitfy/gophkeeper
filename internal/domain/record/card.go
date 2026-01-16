package record

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
	"time"
)

// CardData - данные карты (до шифрования)
type CardData struct {
	CardNumber     string `json:"card_number"`
	CardHolder     string `json:"card_holder"`
	ExpiryMonth    string `json:"expiry_month"`
	ExpiryYear     string `json:"expiry_year"`
	CVV            string `json:"cvv"`
	PIN            string `json:"pin,omitempty"`
	BillingAddress string `json:"billing_address,omitempty"`
}

func (c *CardData) GetType() RecType {
	return RecTypeCard
}

func (c *CardData) Validate() error {
	// Валидация номера карты (Luhn algorithm можно добавить)
	if strings.TrimSpace(c.CardNumber) == "" {
		return fmt.Errorf("card number is required")
	}

	// Очистка номера от пробелов и дефисов
	cleaned := regexp.MustCompile(`[-\s]`).ReplaceAllString(c.CardNumber, "")
	if len(cleaned) < 13 || len(cleaned) > 19 {
		return fmt.Errorf("invalid card number length")
	}

	if strings.TrimSpace(c.CardHolder) == "" {
		return fmt.Errorf("card holder is required")
	}

	// Валидация срока действия
	if !isValidExpiry(c.ExpiryMonth, c.ExpiryYear) {
		return fmt.Errorf("invalid expiry date")
	}

	if strings.TrimSpace(c.CVV) == "" {
		return fmt.Errorf("CVV is required")
	}

	if len(c.CVV) < 3 || len(c.CVV) > 4 {
		return fmt.Errorf("CVV must be 3 or 4 digits")
	}

	return nil
}

func isValidExpiry(month, year string) bool {
	if month == "" || year == "" {
		return false
	}

	// Проверяем что это числа
	matchMonth, _ := regexp.MatchString(`^(0[1-9]|1[0-2])$`, month)
	matchYear, _ := regexp.MatchString(`^20\d{2}$`, year)

	if !matchMonth || !matchYear {
		return false
	}

	// Проверяем что срок не истек
	now := time.Now()
	currentYear := now.Year()
	currentMonth := int(now.Month())

	expYear := 0
	expMonth := 0
	fmt.Sscanf(year, "%d", &expYear)
	fmt.Sscanf(month, "%d", &expMonth)

	if expYear < currentYear {
		return false
	}

	if expYear == currentYear && expMonth < currentMonth {
		return false
	}

	return true
}

func (c *CardData) ToJSON() ([]byte, error) {
	return json.Marshal(c)
}

func (c *CardData) FromJSON(data []byte) error {
	return json.Unmarshal(data, c)
}

// CardMeta - метаданные карты
type CardMeta struct {
	Title         string          `json:"title"`
	BankName      string          `json:"bank_name,omitempty"`
	PaymentSystem string          `json:"payment_system,omitempty"` // visa, mastercard, mir, unionpay
	Category      string          `json:"category,omitempty"`
	Tags          []string        `json:"tags,omitempty"`
	Notes         string          `json:"notes,omitempty"`
	IsVirtual     bool            `json:"is_virtual,omitempty"`
	IsActive      bool            `json:"is_active,omitempty"`
	DailyLimit    *float64        `json:"daily_limit,omitempty"`
	LastUsed      *time.Time      `json:"last_used,omitempty"`
	PhoneNumber   string          `json:"phone_number,omitempty"` // Привязанный телефон
	CustomData    json.RawMessage `json:"custom_data,omitempty"`
}

func (m *CardMeta) Validate() error {
	if strings.TrimSpace(m.Title) == "" {
		return fmt.Errorf("title is required")
	}

	// Валидация платежной системы
	validSystems := map[string]bool{
		"visa": true, "mastercard": true, "mir": true,
		"unionpay": true, "amex": true, "jcb": true,
	}

	if m.PaymentSystem != "" && !validSystems[m.PaymentSystem] {
		return fmt.Errorf("invalid payment system")
	}

	return nil
}

func (m *CardMeta) ToJSON() ([]byte, error) {
	return json.Marshal(m)
}

func (m *CardMeta) FromJSON(data []byte) error {
	return json.Unmarshal(data, m)
}
