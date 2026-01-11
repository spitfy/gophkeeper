package sync

import "errors"

var (
	ErrSyncConflict        = errors.New("sync conflict detected")
	ErrStorageLimit        = errors.New("storage limit exceeded")
	ErrDeviceNotFound      = errors.New("device not found")
	ErrDeviceNotAuthorized = errors.New("device not authorized")
	ErrInvalidSyncToken    = errors.New("invalid sync token")
	ErrSyncInProgress      = errors.New("sync already in progress")
	ErrRecordNotFound      = errors.New("record not found")
	ErrVersionMismatch     = errors.New("version mismatch")
)

// SyncError расширенная ошибка синхронизации
type SyncError struct {
	Err     error
	Code    string
	Details map[string]interface{}
}

func (e *SyncError) Error() string {
	return e.Err.Error()
}

func (e *SyncError) Unwrap() error {
	return e.Err
}

func NewSyncError(err error, code string, details map[string]interface{}) *SyncError {
	return &SyncError{
		Err:     err,
		Code:    code,
		Details: details,
	}
}
