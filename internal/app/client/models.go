package client

import (
	"encoding/json"
	"fmt"
	"time"

	"gophkeeper/internal/domain/record"
)

// LocalRecord - локальная модель записи для хранения в SQLite
// Расширяет серверную модель дополнительными полями для синхронизации
type LocalRecord struct {
	ID            int             `json:"id"`
	ServerID      int             `json:"server_id,omitempty"` // ID на сервере (может отличаться от локального)
	UserID        int             `json:"user_id"`
	Type          record.RecType  `json:"type"`
	EncryptedData string          `json:"encrypted_data,omitempty"`
	Meta          json.RawMessage `json:"meta,omitempty"`
	Version       int             `json:"version"`
	LastModified  time.Time       `json:"last_modified"`
	DeletedAt     *time.Time      `json:"deleted_at,omitempty"`
	Checksum      string          `json:"checksum,omitempty"`
	DeviceID      string          `json:"device_id,omitempty"`
	// Локальные поля для синхронизации
	Synced      bool      `json:"synced"`
	SyncVersion int64     `json:"sync_version"`
	CreatedAt   time.Time `json:"created_at"`
}

// ToServerRecord конвертирует локальную запись в серверную модель
func (r *LocalRecord) ToServerRecord() *record.Record {
	return &record.Record{
		ID:            r.ServerID,
		UserID:        r.UserID,
		Type:          r.Type,
		EncryptedData: r.EncryptedData,
		Meta:          r.Meta,
		Version:       r.Version,
		LastModified:  r.LastModified,
		DeletedAt:     r.DeletedAt,
		Checksum:      r.Checksum,
		DeviceID:      r.DeviceID,
	}
}

// FromServerRecord создает локальную запись из серверной модели
func FromServerRecord(r *record.Record) *LocalRecord {
	return &LocalRecord{
		ServerID:      r.ID,
		UserID:        r.UserID,
		Type:          r.Type,
		EncryptedData: r.EncryptedData,
		Meta:          r.Meta,
		Version:       r.Version,
		LastModified:  r.LastModified,
		DeletedAt:     r.DeletedAt,
		Checksum:      r.Checksum,
		DeviceID:      r.DeviceID,
		Synced:        true,
		CreatedAt:     r.LastModified,
	}
}

// RecordFilter фильтр для списка записей
type RecordFilter struct {
	Type        record.RecType
	ShowDeleted bool
	Limit       int
	Offset      int
}

// MemoryStorage - временное in-memory хранилище
type MemoryStorage struct {
	records   map[int]*LocalRecord
	nextID    int
	serverMap map[int]int // serverID -> localID
}

func NewMemoryStorage() *MemoryStorage {
	return &MemoryStorage{
		records:   make(map[int]*LocalRecord),
		nextID:    1,
		serverMap: make(map[int]int),
	}
}

func (m *MemoryStorage) SaveRecord(rec *LocalRecord) error {
	if rec.ID == 0 {
		rec.ID = m.nextID
		m.nextID++
	}
	m.records[rec.ID] = rec
	if rec.ServerID > 0 {
		m.serverMap[rec.ServerID] = rec.ID
	}
	return nil
}

func (m *MemoryStorage) GetRecord(id int) (*LocalRecord, error) {
	rec, exists := m.records[id]
	if !exists {
		return nil, fmt.Errorf("запись не найдена: %d", id)
	}
	return rec, nil
}

func (m *MemoryStorage) GetRecordByServerID(serverID int) (*LocalRecord, error) {
	localID, exists := m.serverMap[serverID]
	if !exists {
		return nil, fmt.Errorf("запись не найдена по server_id: %d", serverID)
	}
	return m.GetRecord(localID)
}

func (m *MemoryStorage) ListRecords(filter *RecordFilter) ([]*LocalRecord, error) {
	records := make([]*LocalRecord, 0, len(m.records))
	for _, rec := range m.records {
		if rec.DeletedAt != nil && !filter.ShowDeleted {
			continue
		}
		if filter.Type != "" && rec.Type != filter.Type {
			continue
		}
		records = append(records, rec)
	}
	return records, nil
}

func (m *MemoryStorage) UpdateRecord(rec *LocalRecord) error {
	if _, exists := m.records[rec.ID]; !exists {
		return fmt.Errorf("запись не найдена: %d", rec.ID)
	}
	m.records[rec.ID] = rec
	if rec.ServerID > 0 {
		m.serverMap[rec.ServerID] = rec.ID
	}
	return nil
}

func (m *MemoryStorage) DeleteRecord(id int) error {
	if rec, exists := m.records[id]; exists {
		now := time.Now()
		rec.DeletedAt = &now
		rec.LastModified = now
		rec.Synced = false
	}
	return nil
}

func (m *MemoryStorage) HardDeleteRecord(id int) error {
	if rec, exists := m.records[id]; exists {
		if rec.ServerID > 0 {
			delete(m.serverMap, rec.ServerID)
		}
		delete(m.records, id)
	}
	return nil
}

func (m *MemoryStorage) CountRecords() (int, error) {
	count := 0
	for _, rec := range m.records {
		if rec.DeletedAt == nil {
			count++
		}
	}
	return count, nil
}

func (m *MemoryStorage) GetUnsyncedRecords() ([]*LocalRecord, error) {
	var records []*LocalRecord
	for _, rec := range m.records {
		if !rec.Synced {
			records = append(records, rec)
		}
	}
	return records, nil
}

func (m *MemoryStorage) Close() error {
	return nil
}
