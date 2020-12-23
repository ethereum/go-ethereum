package bigcache

import "errors"

// ErrEntryNotFound is an error type struct which is returned when entry was not found for provided key
var ErrEntryNotFound = errors.New("Entry not found")
