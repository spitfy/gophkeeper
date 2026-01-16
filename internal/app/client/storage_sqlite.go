package client

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"gophkeeper/internal/domain/record"

	_ "github.com/mattn/go-sqlite3"
)

type SQLiteStorage struct {
	db *sql.DB
}

func NewSQLiteStorage(path string) (*SQLiteStorage, error) {
	db, err := sql.Open("sqlite3", path+"?_foreign_keys=on&_journal_mode=WAL")
	if err != nil {
		return nil, fmt.Errorf("ошибка открытия базы данных: %w", err)
	}

	storage := &SQLiteStorage{db: db}

	// Создаем таблицы
	if err := storage.initTables(); err != nil {
		db.Close()
		return nil, fmt.Errorf("ошибка инициализации таблиц: %w", err)
	}

	return storage, nil
}

func (s *SQLiteStorage) initTables() error {
	// Таблица записей
	_, err := s.db.Exec(`
		CREATE TABLE IF NOT EXISTS records (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			server_id INTEGER DEFAULT 0,
			user_id INTEGER DEFAULT 0,
			type TEXT NOT NULL,
			encrypted_data TEXT,
			meta TEXT,
			version INTEGER NOT NULL DEFAULT 1,
			last_modified DATETIME NOT NULL,
			deleted_at DATETIME,
			checksum TEXT,
			device_id TEXT,
			synced BOOLEAN NOT NULL DEFAULT 0,
			sync_version INTEGER NOT NULL DEFAULT 0,
			created_at DATETIME NOT NULL
		);
		
		CREATE INDEX IF NOT EXISTS idx_records_type ON records(type);
		CREATE INDEX IF NOT EXISTS idx_records_deleted ON records(deleted_at);
		CREATE INDEX IF NOT EXISTS idx_records_synced ON records(synced);
		CREATE INDEX IF NOT EXISTS idx_records_server_id ON records(server_id);
		CREATE INDEX IF NOT EXISTS idx_records_last_modified ON records(last_modified);
	`)

	return err
}

func (s *SQLiteStorage) SaveRecord(rec *LocalRecord) error {
	metaJSON := string(rec.Meta)
	if rec.Meta == nil {
		metaJSON = "{}"
	}

	var deletedAt sql.NullTime
	if rec.DeletedAt != nil {
		deletedAt = sql.NullTime{Time: *rec.DeletedAt, Valid: true}
	}

	if rec.ID == 0 {
		// Вставляем новую запись
		result, err := s.db.Exec(`
			INSERT INTO records (server_id, user_id, type, encrypted_data, meta, version, 
			                     last_modified, deleted_at, checksum, device_id, synced, 
			                     sync_version, created_at)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		`, rec.ServerID, rec.UserID, rec.Type, rec.EncryptedData, metaJSON, rec.Version,
			rec.LastModified, deletedAt, rec.Checksum, rec.DeviceID, rec.Synced,
			rec.SyncVersion, rec.CreatedAt)
		if err != nil {
			return fmt.Errorf("ошибка вставки записи: %w", err)
		}

		id, err := result.LastInsertId()
		if err != nil {
			return fmt.Errorf("ошибка получения ID: %w", err)
		}
		rec.ID = int(id)
	} else {
		// Обновляем существующую запись
		_, err := s.db.Exec(`
			UPDATE records 
			SET server_id = ?, user_id = ?, type = ?, encrypted_data = ?, meta = ?, 
			    version = ?, last_modified = ?, deleted_at = ?, checksum = ?, 
			    device_id = ?, synced = ?, sync_version = ?
			WHERE id = ?
		`, rec.ServerID, rec.UserID, rec.Type, rec.EncryptedData, metaJSON, rec.Version,
			rec.LastModified, deletedAt, rec.Checksum, rec.DeviceID, rec.Synced,
			rec.SyncVersion, rec.ID)
		if err != nil {
			return fmt.Errorf("ошибка обновления записи: %w", err)
		}
	}

	return nil
}

func (s *SQLiteStorage) GetRecord(id int) (*LocalRecord, error) {
	var rec LocalRecord
	var metaJSON string
	var deletedAt sql.NullTime

	err := s.db.QueryRow(`
		SELECT id, server_id, user_id, type, encrypted_data, meta, version, 
		       last_modified, deleted_at, checksum, device_id, synced, 
		       sync_version, created_at
		FROM records 
		WHERE id = ?
	`, id).Scan(&rec.ID, &rec.ServerID, &rec.UserID, &rec.Type, &rec.EncryptedData,
		&metaJSON, &rec.Version, &rec.LastModified, &deletedAt, &rec.Checksum,
		&rec.DeviceID, &rec.Synced, &rec.SyncVersion, &rec.CreatedAt)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("запись не найдена: %d", id)
	}
	if err != nil {
		return nil, fmt.Errorf("ошибка получения записи: %w", err)
	}

	rec.Meta = json.RawMessage(metaJSON)
	if deletedAt.Valid {
		rec.DeletedAt = &deletedAt.Time
	}

	return &rec, nil
}

func (s *SQLiteStorage) GetRecordByServerID(serverID int) (*LocalRecord, error) {
	var rec LocalRecord
	var metaJSON string
	var deletedAt sql.NullTime

	err := s.db.QueryRow(`
		SELECT id, server_id, user_id, type, encrypted_data, meta, version, 
		       last_modified, deleted_at, checksum, device_id, synced, 
		       sync_version, created_at
		FROM records 
		WHERE server_id = ?
	`, serverID).Scan(&rec.ID, &rec.ServerID, &rec.UserID, &rec.Type, &rec.EncryptedData,
		&metaJSON, &rec.Version, &rec.LastModified, &deletedAt, &rec.Checksum,
		&rec.DeviceID, &rec.Synced, &rec.SyncVersion, &rec.CreatedAt)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("запись не найдена по server_id: %d", serverID)
	}
	if err != nil {
		return nil, fmt.Errorf("ошибка получения записи: %w", err)
	}

	rec.Meta = json.RawMessage(metaJSON)
	if deletedAt.Valid {
		rec.DeletedAt = &deletedAt.Time
	}

	return &rec, nil
}

func (s *SQLiteStorage) ListRecords(filter *RecordFilter) ([]*LocalRecord, error) {
	query := `SELECT id, server_id, user_id, type, encrypted_data, meta, version, 
	                 last_modified, deleted_at, checksum, device_id, synced, 
	                 sync_version, created_at 
	          FROM records WHERE 1=1`
	args := []interface{}{}

	if !filter.ShowDeleted {
		query += " AND deleted_at IS NULL"
	}

	if filter.Type != "" {
		query += " AND type = ?"
		args = append(args, filter.Type)
	}

	query += " ORDER BY last_modified DESC"

	if filter.Limit > 0 {
		query += " LIMIT ?"
		args = append(args, filter.Limit)
	}

	if filter.Offset > 0 {
		query += " OFFSET ?"
		args = append(args, filter.Offset)
	}

	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("ошибка выполнения запроса: %w", err)
	}
	defer rows.Close()

	var records []*LocalRecord
	for rows.Next() {
		var rec LocalRecord
		var metaJSON string
		var deletedAt sql.NullTime

		if err := rows.Scan(&rec.ID, &rec.ServerID, &rec.UserID, &rec.Type, &rec.EncryptedData,
			&metaJSON, &rec.Version, &rec.LastModified, &deletedAt, &rec.Checksum,
			&rec.DeviceID, &rec.Synced, &rec.SyncVersion, &rec.CreatedAt); err != nil {
			return nil, fmt.Errorf("ошибка сканирования записи: %w", err)
		}

		rec.Meta = json.RawMessage(metaJSON)
		if deletedAt.Valid {
			rec.DeletedAt = &deletedAt.Time
		}

		records = append(records, &rec)
	}

	return records, nil
}

func (s *SQLiteStorage) UpdateRecord(rec *LocalRecord) error {
	return s.SaveRecord(rec)
}

func (s *SQLiteStorage) DeleteRecord(id int) error {
	now := time.Now()
	_, err := s.db.Exec(`
		UPDATE records 
		SET deleted_at = ?, last_modified = ?, synced = 0
		WHERE id = ?
	`, now, now, id)
	if err != nil {
		return fmt.Errorf("ошибка удаления записи: %w", err)
	}

	return nil
}

func (s *SQLiteStorage) HardDeleteRecord(id int) error {
	_, err := s.db.Exec("DELETE FROM records WHERE id = ?", id)
	if err != nil {
		return fmt.Errorf("ошибка удаления записи: %w", err)
	}

	return nil
}

func (s *SQLiteStorage) CountRecords() (int, error) {
	var count int
	err := s.db.QueryRow("SELECT COUNT(*) FROM records WHERE deleted_at IS NULL").Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("ошибка подсчета записей: %w", err)
	}

	return count, nil
}

func (s *SQLiteStorage) GetUnsyncedRecords() ([]*LocalRecord, error) {
	return s.ListRecords(&RecordFilter{
		ShowDeleted: true,
	})
}

func (s *SQLiteStorage) GetRecordsModifiedAfter(since time.Time, limit int) ([]*LocalRecord, error) {
	query := `SELECT id, server_id, user_id, type, encrypted_data, meta, version, 
	                 last_modified, deleted_at, checksum, device_id, synced, 
	                 sync_version, created_at 
	          FROM records 
	          WHERE (synced = 0 OR last_modified > ?)
	          ORDER BY last_modified ASC
	          LIMIT ?`

	rows, err := s.db.Query(query, since, limit)
	if err != nil {
		return nil, fmt.Errorf("ошибка выполнения запроса: %w", err)
	}
	defer rows.Close()

	var records []*LocalRecord
	for rows.Next() {
		var rec LocalRecord
		var metaJSON string
		var deletedAt sql.NullTime

		if err := rows.Scan(&rec.ID, &rec.ServerID, &rec.UserID, &rec.Type, &rec.EncryptedData,
			&metaJSON, &rec.Version, &rec.LastModified, &deletedAt, &rec.Checksum,
			&rec.DeviceID, &rec.Synced, &rec.SyncVersion, &rec.CreatedAt); err != nil {
			return nil, fmt.Errorf("ошибка сканирования записи: %w", err)
		}

		rec.Meta = json.RawMessage(metaJSON)
		if deletedAt.Valid {
			rec.DeletedAt = &deletedAt.Time
		}

		records = append(records, &rec)
	}

	return records, nil
}

func (s *SQLiteStorage) MarkAsSynced(id int, serverID int, syncVersion int64) error {
	_, err := s.db.Exec(`
		UPDATE records 
		SET synced = 1, server_id = ?, sync_version = ?
		WHERE id = ?
	`, serverID, syncVersion, id)
	if err != nil {
		return fmt.Errorf("ошибка обновления статуса синхронизации: %w", err)
	}

	return nil
}

// GetDB возвращает подключение к базе данных (для sync service)
func (s *SQLiteStorage) GetDB() *sql.DB {
	return s.db
}

func (s *SQLiteStorage) Close() error {
	return s.db.Close()
}

// Storage интерфейс для локального хранилища
type Storage interface {
	SaveRecord(rec *LocalRecord) error
	GetRecord(id int) (*LocalRecord, error)
	GetRecordByServerID(serverID int) (*LocalRecord, error)
	ListRecords(filter *RecordFilter) ([]*LocalRecord, error)
	UpdateRecord(rec *LocalRecord) error
	DeleteRecord(id int) error
	HardDeleteRecord(id int) error
	CountRecords() (int, error)
	GetUnsyncedRecords() ([]*LocalRecord, error)
	GetRecordsModifiedAfter(since time.Time, limit int) ([]*LocalRecord, error)
	MarkAsSynced(id int, serverID int, syncVersion int64) error
	Close() error
}

// Убедимся, что SQLiteStorage и MemoryStorage реализуют интерфейс Storage
var _ Storage = (*SQLiteStorage)(nil)
var _ Storage = (*MemoryStorage)(nil)

// Добавляем недостающие методы в MemoryStorage

func (m *MemoryStorage) GetRecordsModifiedAfter(since time.Time, limit int) ([]*LocalRecord, error) {
	var records []*LocalRecord
	for _, rec := range m.records {
		if !rec.Synced || rec.LastModified.After(since) {
			records = append(records, rec)
			if len(records) >= limit {
				break
			}
		}
	}
	return records, nil
}

func (m *MemoryStorage) MarkAsSynced(id int, serverID int, syncVersion int64) error {
	rec, exists := m.records[id]
	if !exists {
		return fmt.Errorf("запись не найдена: %d", id)
	}
	rec.Synced = true
	rec.ServerID = serverID
	rec.SyncVersion = syncVersion
	m.serverMap[serverID] = id
	return nil
}

// Unused import fix
var _ = record.RecTypeLogin
