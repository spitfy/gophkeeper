package client

import (
	"fmt"
	"time"

	"gophkeeper/internal/domain/record"
)

// Record - локальная модель записи
type Record struct {
	ID        string          `json:"id"`
	Type      record.Type     `json:"type"`
	Metadata  record.Metadata `json:"metadata"`
	Data      []byte          `json:"data"`
	CreatedAt time.Time       `json:"created_at"`
	UpdatedAt time.Time       `json:"updated_at"`
	Version   int             `json:"version"`
	Synced    bool            `json:"synced"`
	Deleted   bool            `json:"deleted"`
}

// MemoryStorage - временное in-memory хранилище
type MemoryStorage struct {
	records map[string]*Record
}

func NewMemoryStorage() *MemoryStorage {
	return &MemoryStorage{
		records: make(map[string]*Record),
	}
}

func (m *MemoryStorage) SaveRecord(record *Record) error {
	m.records[record.ID] = record
	return nil
}

func (m *MemoryStorage) GetRecord(id string) (*Record, error) {
	record, exists := m.records[id]
	if !exists {
		return nil, fmt.Errorf("запись не найдена: %s", id)
	}
	return record, nil
}

func (m *MemoryStorage) ListRecords() ([]*Record, error) {
	records := make([]*Record, 0, len(m.records))
	for _, record := range m.records {
		if !record.Deleted {
			records = append(records, record)
		}
	}
	return records, nil
}

func (m *MemoryStorage) DeleteRecord(id string) error {
	if record, exists := m.records[id]; exists {
		record.Deleted = true
		record.UpdatedAt = time.Now()
	}
	return nil
}
