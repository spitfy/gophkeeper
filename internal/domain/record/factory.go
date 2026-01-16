package record

import (
	"fmt"
)

// RecordFactory - фабрика для создания моделей записей
type RecordFactory struct{}

// NewRecordFactory создает новую фабрику
func NewRecordFactory() *RecordFactory {
	return &RecordFactory{}
}

// CreateData создает структуру данных для указанного типа
func (f *RecordFactory) CreateData(typ RecType) (RecordData, error) {
	switch typ {
	case RecTypeLogin:
		return &LoginData{}, nil
	case RecTypeText:
		return &TextData{}, nil
	case RecTypeBinary:
		return &BinaryData{}, nil
	case RecTypeCard:
		return &CardData{}, nil
	default:
		return nil, fmt.Errorf("unsupported record type: %s", typ)
	}
}

// CreateMeta создает структуру метаданных для указанного типа
func (f *RecordFactory) CreateMeta(typ RecType) (MetaData, error) {
	switch typ {
	case RecTypeLogin:
		return &LoginMeta{}, nil
	case RecTypeText:
		return &TextMeta{}, nil
	case RecTypeBinary:
		return &BinaryMeta{}, nil
	case RecTypeCard:
		return &CardMeta{}, nil
	default:
		return nil, fmt.Errorf("unsupported record type: %s", typ)
	}
}

// ParseMeta парсит метаданные из JSON
func (f *RecordFactory) ParseMeta(typ RecType, data []byte) (MetaData, error) {
	meta, err := f.CreateMeta(typ)
	if err != nil {
		return nil, err
	}

	if err := meta.FromJSON(data); err != nil {
		return nil, fmt.Errorf("failed to parse meta for type %s: %w", typ, err)
	}

	return meta, nil
}

// ParseData парсит данные из JSON
func (f *RecordFactory) ParseData(typ RecType, data []byte) (RecordData, error) {
	recordData, err := f.CreateData(typ)
	if err != nil {
		return nil, err
	}

	if err := recordData.FromJSON(data); err != nil {
		return nil, fmt.Errorf("failed to parse data for type %s: %w", typ, err)
	}

	return recordData, nil
}

// ValidateRecordData валидирует данные записи
func (f *RecordFactory) ValidateRecordData(typ RecType, data []byte) error {
	recordData, err := f.ParseData(typ, data)
	if err != nil {
		return err
	}

	return recordData.Validate()
}

// ValidateMetaData валидирует метаданные
func (f *RecordFactory) ValidateMetaData(typ RecType, data []byte) error {
	meta, err := f.ParseMeta(typ, data)
	if err != nil {
		return err
	}

	return meta.Validate()
}

// PrepareRecord подготавливает запись к сохранению
func (f *RecordFactory) PrepareRecord(typ RecType, data RecordData, meta MetaData) (*Record, error) {
	// Валидация данных
	if err := data.Validate(); err != nil {
		return nil, fmt.Errorf("data validation failed: %w", err)
	}

	if err := meta.Validate(); err != nil {
		return nil, fmt.Errorf("meta validation failed: %w", err)
	}

	// Преобразование данных в JSON
	dataJSON, err := data.ToJSON()
	if err != nil {
		return nil, fmt.Errorf("failed to marshal data: %w", err)
	}

	metaJSON, err := meta.ToJSON()
	if err != nil {
		return nil, fmt.Errorf("failed to marshal meta: %w", err)
	}

	record := &Record{
		Type:          typ,
		EncryptedData: string(dataJSON), // В реальности здесь будет зашифрованная строка
		Meta:          metaJSON,
		Version:       1,
	}

	return record, nil
}

// GetDefaultMeta возвращает метаданные по умолчанию для типа
func (f *RecordFactory) GetDefaultMeta(typ RecType) (MetaData, error) {
	meta, err := f.CreateMeta(typ)
	if err != nil {
		return nil, err
	}

	// Устанавливаем значения по умолчанию
	switch m := meta.(type) {
	case *LoginMeta:
		m.Category = "Логины"
		m.AutoLogin = false
		m.TwoFA = false
	case *TextMeta:
		m.Category = "Тексты"
		m.Format = "plain"
		m.IsSensitive = false
	case *BinaryMeta:
		m.Category = "Файлы"
	case *CardMeta:
		m.Category = "Карты"
		m.IsActive = true
		m.IsVirtual = false
	}

	return meta, nil
}
