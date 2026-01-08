package client

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	_ "github.com/mattn/go-sqlite3"
	"gophkeeper/internal/domain/record"
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
			id TEXT PRIMARY KEY,
			type TEXT NOT NULL,
			metadata TEXT NOT NULL,
			data BLOB NOT NULL,
			created_at DATETIME NOT NULL,
			updated_at DATETIME NOT NULL,
			version INTEGER NOT NULL DEFAULT 1,
			synced BOOLEAN NOT NULL DEFAULT 0,
			deleted BOOLEAN NOT NULL DEFAULT 0,
			sync_version INTEGER NOT NULL DEFAULT 0
		);
		
		CREATE INDEX IF NOT EXISTS idx_records_type ON records(type);
		CREATE INDEX IF NOT EXISTS idx_records_deleted ON records(deleted);
		CREATE INDEX IF NOT EXISTS idx_records_updated ON records(updated_at);
	`)

	return err
}

func (s *SQLiteStorage) SaveRecord(rec *record.Record) error {
	metadataJSON, err := json.Marshal(rec.Meta)
	if err != nil {
		return fmt.Errorf("ошибка сериализации метаданных: %w", err)
	}

	// Проверяем, существует ли запись
	var exists bool
	err = s.db.QueryRow("SELECT EXISTS(SELECT 1 FROM records WHERE id = ?)", rec.ID).Scan(&exists)
	if err != nil {
		return fmt.Errorf("ошибка проверки существования записи: %w", err)
	}

	if exists {
		// Обновляем
		_, err = s.db.Exec(`
			UPDATE records 
			SET type = ?, metadata = ?, data = ?, updated_at = ?, version = ?, 
			    synced = ?, deleted = ?, sync_version = ?
			WHERE id = ?
		`, rec.Type, metadataJSON, rec.EncryptedData, rec.LastModified, rec.Version,
			rec.Synced, rec.DeletedAt, rec.Version, rec.ID)
	} else {
		// Вставляем новую
		_, err = s.db.Exec(`
			INSERT INTO records (id, type, metadata, data, created_at, updated_at, 
			                     version, synced, deleted, sync_version)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		`, rec.ID, rec.Type, metadataJSON, rec.Data, rec.CreatedAt, rec.UpdatedAt,
			rec.Version, rec.Synced, rec.Deleted, rec.SyncVersion)
	}

	if err != nil {
		return fmt.Errorf("ошибка сохранения записи: %w", err)
	}

	return nil
}

func (s *SQLiteStorage) GetRecord(id string) (*record.Record, error) {
	var rec Record
	var metadataJSON string
	var createdAt, updatedAt string

	err := s.db.QueryRow(`
		SELECT id, type, metadata, data, created_at, updated_at, 
		       version, synced, deleted, sync_version
		FROM records 
		WHERE id = ? AND deleted = 0
	`, id).Scan(&rec.ID, &rec.Type, &metadataJSON, &rec.Data,
		&createdAt, &updatedAt, &rec.Version, &rec.Synced,
		&rec.Deleted, &rec.SyncVersion)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("запись не найдена: %s", id)
	}
	if err != nil {
		return nil, fmt.Errorf("ошибка получения записи: %w", err)
	}

	// Парсим метаданные
	if err := json.Unmarshal([]byte(metadataJSON), &rec.Metadata); err != nil {
		return nil, fmt.Errorf("ошибка парсинга метаданных: %w", err)
	}

	// Парсим временные метки
	rec.CreatedAt, _ = time.Parse(time.RFC3339, createdAt)
	rec.UpdatedAt, _ = time.Parse(time.RFC3339, updatedAt)

	return &rec, nil
}

func (s *SQLiteStorage) ListRecords(filter *record.RecordFilter) ([]*record.Record, error) {
	query := "SELECT id, type, metadata, data, created_at, updated_at, version, synced, deleted, sync_version FROM records WHERE 1=1"
	args := []interface{}{}

	if !filter.ShowDeleted {
		query += " AND deleted = 0"
	}

	if filter.Type != "" {
		query += " AND type = ?"
		args = append(args, filter.Type)
	}

	query += " ORDER BY updated_at DESC"

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

	var records []*record.Record
	for rows.Next() {
		var rec Record
		var metadataJSON string
		var createdAt, updatedAt string

		if err := rows.Scan(&rec.ID, &rec.Type, &metadataJSON, &rec.Data,
			&createdAt, &updatedAt, &rec.Version, &rec.Synced,
			&rec.Deleted, &rec.SyncVersion); err != nil {
			return nil, fmt.Errorf("ошибка сканирования записи: %w", err)
		}

		// Парсим метаданные
		if err := json.Unmarshal([]byte(metadataJSON), &rec.Metadata); err != nil {
			return nil, fmt.Errorf("ошибка парсинга метаданных: %w", err)
		}

		// Парсим временные метки
		rec.CreatedAt, _ = time.Parse(time.RFC3339, createdAt)
		rec.UpdatedAt, _ = time.Parse(time.RFC3339, updatedAt)

		records = append(records, &rec)
	}

	return records, nil
}

func (s *SQLiteStorage) UpdateRecord(rec *record.Record) error {
	metadataJSON, err := json.Marshal(rec.Metadata)
	if err != nil {
		return fmt.Errorf("ошибка сериализации метаданных: %w", err)
	}

	_, err = s.db.Exec(`
		UPDATE records 
		SET type = ?, metadata = ?, data = ?, updated_at = ?, version = ?, 
		    synced = ?, deleted = ?, sync_version = ?
		WHERE id = ?
	`, rec.Type, metadataJSON, rec.Data, rec.UpdatedAt, rec.Version,
		rec.Synced, rec.Deleted, rec.SyncVersion, rec.ID)

	if err != nil {
		return fmt.Errorf("ошибка обновления записи: %w", err)
	}

	return nil
}

func (s *SQLiteStorage) DeleteRecord(id string) error {
	_, err := s.db.Exec("DELETE FROM records WHERE id = ?", id)
	if err != nil {
		return fmt.Errorf("ошибка удаления записи: %w", err)
	}

	return nil
}

func (s *SQLiteStorage) CountRecords() (int, error) {
	var count int
	err := s.db.QueryRow("SELECT COUNT(*) FROM records WHERE deleted = 0").Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("ошибка подсчета записей: %w", err)
	}

	return count, nil
}

func (s *SQLiteStorage) Close() error {
	return s.db.Close()
}
