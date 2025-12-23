package postgres

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"golang.org/x/crypto/bcrypt"
	"gophkeeper/internal/config"
	"gophkeeper/internal/storage"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

type Storage struct {
	db *pgxpool.Pool
}

func New(cfg *config.Config) (*Storage, error) {
	pool, err := pgxpool.New(context.Background(), cfg.DB.DatabaseURI)
	if err != nil {
		return nil, fmt.Errorf("create pool: %w", err)
	}
	return &Storage{db: pool}, nil
}

func (s *Storage) Close() error {
	s.db.Close()
	return nil
}

func (s *Storage) Pool() *pgxpool.Pool {
	return s.db
}

// === ПОЛЬЗОВАТЕЛИ ===
func (s *Storage) CreateUser(ctx context.Context, login, passwordHash string) (int, error) {
	var userID int
	err := s.db.QueryRow(ctx,
		`INSERT INTO users (login, password_hash) VALUES ($1, $2) RETURNING id`,
		login, passwordHash).Scan(&userID)
	return userID, err
}

func (s *Storage) AuthUser(ctx context.Context, login, password string) (int, string, error) {
	var userID int
	var storedHash string
	err := s.db.QueryRow(ctx,
		`SELECT id, password_hash FROM users WHERE login = $1`, login).
		Scan(&userID, &storedHash)
	if err != nil {
		return 0, "", fmt.Errorf("user not found")
	}

	if err := bcrypt.CompareHashAndPassword([]byte(storedHash), []byte(password)); err != nil {
		return 0, "", fmt.Errorf("invalid password")
	}

	return userID, storedHash, nil
}

// === СЕССИИ ===
func (s *Storage) CreateSession(ctx context.Context, userID int, tokenHash string, expiresAt time.Time) error {
	_, err := s.db.Exec(ctx,
		`INSERT INTO sessions (user_id, token_hash, expires_at) 
         VALUES ($1, decode($2, 'hex'), $3)`,
		userID, tokenHash, expiresAt)
	return err
}

func (s *Storage) ValidateSession(ctx context.Context, tokenHash string) (int, error) {
	var userID int
	err := s.db.QueryRow(ctx,
		`SELECT user_id FROM sessions 
         WHERE token_hash = decode($1, 'hex') AND expires_at > NOW()
         AND deleted_at IS NULL`,
		tokenHash).Scan(&userID)

	if err != nil {
		return 0, fmt.Errorf("invalid session")
	}
	return userID, nil
}

// === ЗАПИСИ ===
func (s *Storage) ListRecords(ctx context.Context, userID int) ([]storage.Record, error) {
	rows, err := s.db.Query(ctx,
		`SELECT id, type, encrypted_data, meta, version, last_modified 
         FROM records WHERE user_id = $1 ORDER BY last_modified DESC`,
		userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var records []storage.Record
	for rows.Next() {
		var r storage.Record
		var data []byte
		err := rows.Scan(&r.ID, &r.Type, &data, &r.Meta, &r.Version, &r.LastModified)
		if err != nil {
			return nil, err
		}
		r.EncryptedData = hex.EncodeToString(data) // или base64
		records = append(records, r)
	}
	return records, nil
}

func (s *Storage) CreateRecord(ctx context.Context, userID int, typ, encryptedData string, meta json.RawMessage) (int, error) {
	data, _ := hex.DecodeString(encryptedData)
	var recordID int
	err := s.db.QueryRow(ctx,
		`INSERT INTO records (user_id, type, encrypted_data, meta) 
         VALUES ($1, $2, $3, $4) RETURNING id`,
		userID, typ, data, meta).Scan(&recordID)
	return recordID, err
}

func (s *Storage) GetRecord(ctx context.Context, userID, recordID int) (*storage.Record, error) {
	row := s.db.QueryRow(ctx,
		`SELECT id, type, encrypted_data, meta, version, last_modified 
         FROM records WHERE user_id = $1 AND id = $2`,
		userID, recordID)

	var r storage.Record
	var data []byte
	err := row.Scan(&r.ID, &r.Type, &data, &r.Meta, &r.Version, &r.LastModified)
	if err != nil {
		return nil, err
	}
	r.EncryptedData = hex.EncodeToString(data)
	return &r, nil
}

func (s *Storage) UpdateRecord(ctx context.Context, userID, recordID int, typ, encryptedData string, meta json.RawMessage) error {
	data, _ := hex.DecodeString(encryptedData)
	_, err := s.db.Exec(ctx,
		`UPDATE records SET type = $1, encrypted_data = $2, meta = $3, 
         version = version + 1, last_modified = NOW()
         WHERE user_id = $4 AND id = $5`,
		typ, data, meta, userID, recordID)
	return err
}

func (s *Storage) DeleteRecord(ctx context.Context, userID, recordID int) error {
	_, err := s.db.Exec(ctx,
		`DELETE FROM records WHERE user_id = $1 AND id = $2`,
		userID, recordID)
	return err
}
