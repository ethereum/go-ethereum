package bigcache

import "fmt"

// EntryNotFoundError is an error type struct which is returned when entry was not found for provided key
type EntryNotFoundError struct {
	key string
}

func notFound(key string) error {
	return &EntryNotFoundError{key}
}

// Error returned when entry does not exist.
func (e EntryNotFoundError) Error() string {
	return fmt.Sprintf("Entry %q not found", e.key)
}
