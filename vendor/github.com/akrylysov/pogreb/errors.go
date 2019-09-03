package pogreb

import (
	"errors"
)

var (
	errKeyEmpty      = errors.New("key is empty")
	errKeyTooLarge   = errors.New("key is too large")
	errValueTooLarge = errors.New("value is too large")
	errFull          = errors.New("database is full")
	errCorrupted     = errors.New("database is corrupted")
	errLocked        = errors.New("database is locked")
)
