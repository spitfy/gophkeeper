package record

import (
	"errors"
)

var (
	ErrNotFound        = errors.New("record not found")
	ErrInvalidData     = errors.New("invalid record data")
	ErrVersionConflict = errors.New("record version conflict")
	ErrRecordDeleted   = errors.New("record was deleted")
)
