package sync

import "errors"

var (
	ErrDeviceNotFound = errors.New("device not found")
	ErrRecordNotFound = errors.New("record not found")
)
