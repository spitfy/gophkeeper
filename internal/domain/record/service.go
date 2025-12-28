package record

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"golang.org/x/exp/slog"
)

// Service errors
var (
	ErrRecordNotFound   = errors.New("record not found")
	ErrInvalidData      = errors.New("invalid record data")
	ErrVersionConflict  = errors.New("record version conflict")
	ErrPermissionDenied = errors.New("permission denied")
	ErrRecordDeleted    = errors.New("record was deleted")
)

// Service defines the business logic for record operations
type Service struct {
	repo Repository
	log  *slog.Logger
}

type Servicer interface {
	List(ctx context.Context, userID int) (ListResponse, error)
	Create(ctx context.Context, userID int, typ, encryptedData string, meta json.RawMessage) (CreateResponse, error)
	Find(ctx context.Context, userID, recordID int) (*Record, error)
	Update(ctx context.Context, userID, recordID int, typ, encryptedData string, meta json.RawMessage) error
	Delete(ctx context.Context, userID, recordID int) error
	SoftDelete(ctx context.Context, userID, recordID int) error
	Search(ctx context.Context, userID int, criteria SearchCriteria) ([]Record, error)
	GetStats(ctx context.Context, userID int) (StatsResponse, error)
	GetModifiedSince(ctx context.Context, userID int, since time.Time) ([]Record, error)
	BatchCreate(ctx context.Context, userID int, records []CreateRequest) (BatchCreateResponse, error)
	BatchUpdate(ctx context.Context, userID int, updates []UpdateRequest) (BatchUpdateResponse, error)
}

// Request/Response types
type ListResponse struct {
	Records []RecordItem `json:"records"`
	Total   int          `json:"total"`
}

type CreateResponse struct {
	ID      int    `json:"id"`
	Status  string `json:"status"`
	Error   string `json:"error,omitempty"`
	Version int    `json:"version"`
}

type CreateRequest struct {
	Type          string          `json:"type"`
	EncryptedData string          `json:"encrypted_data"`
	Meta          json.RawMessage `json:"meta"`
	Checksum      string          `json:"checksum,omitempty"`
	DeviceID      string          `json:"device_id,omitempty"`
}

type UpdateRequest struct {
	RecordID      int             `json:"record_id"`
	Type          string          `json:"type"`
	EncryptedData string          `json:"encrypted_data"`
	Meta          json.RawMessage `json:"meta"`
	Version       int             `json:"version"`
	Checksum      string          `json:"checksum,omitempty"`
	DeviceID      string          `json:"device_id,omitempty"`
}

type BatchCreateResponse struct {
	SuccessCount int               `json:"success_count"`
	FailedCount  int               `json:"failed_count"`
	Failed       []FailedOperation `json:"failed,omitempty"`
}

type BatchUpdateResponse struct {
	SuccessCount int               `json:"success_count"`
	FailedCount  int               `json:"failed_count"`
	Failed       []FailedOperation `json:"failed,omitempty"`
}

type FailedOperation struct {
	Index    int    `json:"index"`
	RecordID int    `json:"record_id,omitempty"`
	Error    string `json:"error"`
}

type StatsResponse struct {
	TotalRecords int64                `json:"total_records"`
	TotalSize    int64                `json:"total_size"`
	ByType       map[string]TypeStats `json:"by_type"`
	ActiveDays   int                  `json:"active_days"`
}

type TypeStats struct {
	Count int64 `json:"count"`
	Size  int64 `json:"size"`
}

// NewService creates a new record service
func NewService(repo Repository, log *slog.Logger) Servicer {
	return &Service{
		repo: repo,
		log:  log.With("component", "record_service"),
	}
}

// List returns all records for a user
func (s *Service) List(ctx context.Context, userID int) (ListResponse, error) {
	records, err := s.repo.List(ctx, userID)
	if err != nil {
		s.log.Error("failed to list records", "user_id", userID, "error", err)
		return ListResponse{}, fmt.Errorf("list records: %w", err)
	}

	items := make([]RecordItem, len(records))
	for i, r := range records {
		items[i] = RecordItem{
			ID:           r.ID,
			Type:         r.Type,
			Meta:         r.Meta,
			Version:      r.Version,
			LastModified: r.LastModified,
		}
	}

	return ListResponse{
		Records: items,
		Total:   len(items),
	}, nil
}

// Create creates a new record
func (s *Service) Create(ctx context.Context, userID int, typ, encryptedData string, meta json.RawMessage) (CreateResponse, error) {
	// Validate input
	if typ == "" || encryptedData == "" {
		return CreateResponse{}, ErrInvalidData
	}

	// Generate checksum if not provided
	checksum := s.generateChecksum(encryptedData, typ, meta)

	record := &Record{
		UserID:        userID,
		Type:          typ,
		EncryptedData: encryptedData,
		Meta:          meta,
		Checksum:      checksum,
		Version:       1,
		LastModified:  time.Now(),
	}

	recordID, err := s.repo.Create(ctx, record)
	if err != nil {
		s.log.Error("failed to create record", "user_id", userID, "type", typ, "error", err)
		return CreateResponse{
			ID:     0,
			Status: "Error",
			Error:  err.Error(),
		}, fmt.Errorf("create record: %w", err)
	}

	s.log.Info("record created successfully", "record_id", recordID, "user_id", userID, "type", typ)

	return CreateResponse{
		ID:      recordID,
		Status:  "Ok",
		Version: record.Version,
	}, nil
}

// Find returns a specific record by ID
func (s *Service) Find(ctx context.Context, userID, recordID int) (*Record, error) {
	record, err := s.repo.Get(ctx, userID, recordID)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			return nil, ErrRecordNotFound
		}
		s.log.Error("failed to find record", "record_id", recordID, "user_id", userID, "error", err)
		return nil, fmt.Errorf("find record: %w", err)
	}

	// Check if record is deleted
	if record.DeletedAt != nil {
		return nil, ErrRecordDeleted
	}

	return record, nil
}

// Update updates an existing record
func (s *Service) Update(ctx context.Context, userID, recordID int, typ, encryptedData string, meta json.RawMessage) error {
	// Get the current record to check permissions and get version
	currentRecord, err := s.repo.Get(ctx, userID, recordID)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			return ErrRecordNotFound
		}
		return fmt.Errorf("get record for update: %w", err)
	}

	// Check if record is deleted
	if currentRecord.DeletedAt != nil {
		return ErrRecordDeleted
	}

	// Generate new checksum
	checksum := s.generateChecksum(encryptedData, typ, meta)

	// Update the record
	updatedRecord := &Record{
		ID:            recordID,
		UserID:        userID,
		Type:          typ,
		EncryptedData: encryptedData,
		Meta:          meta,
		Checksum:      checksum,
		Version:       currentRecord.Version,
	}

	err = s.repo.Update(ctx, updatedRecord)
	if err != nil {
		if errors.Is(err, ErrVersionMismatch) {
			return ErrVersionConflict
		}
		s.log.Error("failed to update record", "record_id", recordID, "user_id", userID, "error", err)
		return fmt.Errorf("update record: %w", err)
	}

	s.log.Info("record updated successfully", "record_id", recordID, "user_id", userID, "new_version", updatedRecord.Version)
	return nil
}

// Delete permanently deletes a record
func (s *Service) Delete(ctx context.Context, userID, recordID int) error {
	// First check if record exists and belongs to user
	record, err := s.repo.Get(ctx, userID, recordID)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			return ErrRecordNotFound
		}
		return fmt.Errorf("get record for delete: %w", err)
	}

	err = s.repo.Delete(ctx, userID, recordID)
	if err != nil {
		s.log.Error("failed to delete record", "record_id", recordID, "user_id", userID, "error", err)
		return fmt.Errorf("delete record: %w", err)
	}

	// Save a version snapshot before deletion
	version := &RecordVersion{
		RecordID:      recordID,
		Version:       record.Version + 1,
		EncryptedData: record.EncryptedData,
		Meta:          record.Meta,
		Checksum:      record.Checksum,
	}
	_ = s.repo.SaveVersion(ctx, version) // Log error but don't fail delete operation

	s.log.Info("record deleted successfully", "record_id", recordID, "user_id", userID)
	return nil
}

// SoftDelete marks a record as deleted without removing it
func (s *Service) SoftDelete(ctx context.Context, userID, recordID int) error {
	// First check if record exists and belongs to user
	record, err := s.repo.Get(ctx, userID, recordID)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			return ErrRecordNotFound
		}
		return fmt.Errorf("get record for soft delete: %w", err)
	}

	if record.DeletedAt != nil {
		// Already deleted
		return nil
	}

	err = s.repo.SoftDelete(ctx, userID, recordID)
	if err != nil {
		s.log.Error("failed to soft delete record", "record_id", recordID, "user_id", userID, "error", err)
		return fmt.Errorf("soft delete record: %w", err)
	}

	s.log.Info("record soft deleted", "record_id", recordID, "user_id", userID)
	return nil
}

// Search searches records with criteria
func (s *Service) Search(ctx context.Context, userID int, criteria SearchCriteria) ([]Record, error) {
	records, err := s.repo.Search(ctx, userID, criteria)
	if err != nil {
		s.log.Error("failed to search records", "user_id", userID, "criteria", criteria, "error", err)
		return nil, fmt.Errorf("search records: %w", err)
	}
	return records, nil
}

// GetStats returns statistics for user's records
func (s *Service) GetStats(ctx context.Context, userID int) (StatsResponse, error) {
	stats, err := s.repo.GetStats(ctx, userID)
	if err != nil {
		s.log.Error("failed to get stats", "user_id", userID, "error", err)
		return StatsResponse{}, fmt.Errorf("get stats: %w", err)
	}

	// Convert to structured response
	response := StatsResponse{
		TotalRecords: 0,
		TotalSize:    0,
		ByType:       make(map[string]TypeStats),
	}

	// Extract data from map
	if totalRecords, ok := stats["total_records"].(int64); ok {
		response.TotalRecords = totalRecords
	}
	if totalSize, ok := stats["total_size"].(int64); ok {
		response.TotalSize = totalSize
	}

	// Extract type stats
	if byType, ok := stats["by_type"].(map[string]map[string]interface{}); ok {
		for typ, typeStats := range byType {
			if count, ok := typeStats["count"].(int64); ok {
				if size, ok := typeStats["size"].(int64); ok {
					response.ByType[typ] = TypeStats{
						Count: count,
						Size:  size,
					}
				}
			}
		}
	}

	return response, nil
}

// GetModifiedSince returns records modified since given time
func (s *Service) GetModifiedSince(ctx context.Context, userID int, since time.Time) ([]Record, error) {
	records, err := s.repo.GetModifiedSince(ctx, userID, since)
	if err != nil {
		s.log.Error("failed to get modified records", "user_id", userID, "since", since, "error", err)
		return nil, fmt.Errorf("get modified records: %w", err)
	}
	return records, nil
}

// BatchCreate creates multiple records in batch
func (s *Service) BatchCreate(ctx context.Context, userID int, requests []CreateRequest) (BatchCreateResponse, error) {
	if len(requests) == 0 {
		return BatchCreateResponse{}, nil
	}

	var failed []FailedOperation
	successCount := 0

	for i, req := range requests {
		record := &Record{
			UserID:        userID,
			Type:          req.Type,
			EncryptedData: req.EncryptedData,
			Meta:          req.Meta,
			Checksum:      req.Checksum,
			DeviceID:      req.DeviceID,
		}

		// Generate checksum if not provided
		if record.Checksum == "" {
			record.Checksum = s.generateChecksum(record.EncryptedData, record.Type, record.Meta)
		}

		_, err := s.repo.Create(ctx, record)
		if err != nil {
			failed = append(failed, FailedOperation{
				Index: i,
				Error: err.Error(),
			})
		} else {
			successCount++
		}
	}

	return BatchCreateResponse{
		SuccessCount: successCount,
		FailedCount:  len(failed),
		Failed:       failed,
	}, nil
}

// BatchUpdate updates multiple records in batch
func (s *Service) BatchUpdate(ctx context.Context, userID int, updates []UpdateRequest) (BatchUpdateResponse, error) {
	if len(updates) == 0 {
		return BatchUpdateResponse{}, nil
	}

	var failed []FailedOperation
	successCount := 0

	for i, update := range updates {
		// Get current record to verify ownership and version
		record, err := s.repo.Get(ctx, userID, update.RecordID)
		if err != nil {
			failed = append(failed, FailedOperation{
				Index:    i,
				RecordID: update.RecordID,
				Error:    err.Error(),
			})
			continue
		}

		// Check version
		if record.Version != update.Version {
			failed = append(failed, FailedOperation{
				Index:    i,
				RecordID: update.RecordID,
				Error:    "version mismatch",
			})
			continue
		}

		// Prepare updated record
		updatedRecord := &Record{
			ID:            update.RecordID,
			UserID:        userID,
			Type:          update.Type,
			EncryptedData: update.EncryptedData,
			Meta:          update.Meta,
			Checksum:      update.Checksum,
			Version:       update.Version,
			DeviceID:      update.DeviceID,
		}

		// Generate checksum if not provided
		if updatedRecord.Checksum == "" {
			updatedRecord.Checksum = s.generateChecksum(updatedRecord.EncryptedData, updatedRecord.Type, updatedRecord.Meta)
		}

		err = s.repo.Update(ctx, updatedRecord)
		if err != nil {
			failed = append(failed, FailedOperation{
				Index:    i,
				RecordID: update.RecordID,
				Error:    err.Error(),
			})
		} else {
			successCount++
		}
	}

	return BatchUpdateResponse{
		SuccessCount: successCount,
		FailedCount:  len(failed),
		Failed:       failed,
	}, nil
}

// Helper method to generate checksum
func (s *Service) generateChecksum(encryptedData, typ string, meta json.RawMessage) string {
	data := encryptedData + typ + string(meta)
	hash := sha256.Sum256([]byte(data))
	return hex.EncodeToString(hash[:])
}

// GetByType returns records of specific type
func (s *Service) GetByType(ctx context.Context, userID int, recordType string) ([]Record, error) {
	return s.repo.GetByType(ctx, userID, recordType)
}

// GetVersions returns version history for a record
func (s *Service) GetVersions(ctx context.Context, userID, recordID int) ([]RecordVersion, error) {
	// First verify ownership
	_, err := s.repo.Get(ctx, userID, recordID)
	if err != nil {
		return nil, fmt.Errorf("verify record ownership: %w", err)
	}

	return s.repo.GetVersions(ctx, recordID)
}
