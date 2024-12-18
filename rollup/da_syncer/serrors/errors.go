package serrors

import (
	"fmt"
)

const (
	temporary Type = iota
	eof
)

var (
	TemporaryError = NewTemporaryError(nil)
	EOFError       = NewEOFError(nil)
)

type Type uint8

func (t Type) String() string {
	switch t {
	case temporary:
		return "temporary"
	case eof:
		return "EOF"
	default:
		return "unknown"
	}
}

type syncError struct {
	t   Type
	err error
}

func NewTemporaryError(err error) error {
	return &syncError{t: temporary, err: err}
}

func NewEOFError(err error) error {
	return &syncError{t: eof, err: err}
}

func (s *syncError) Error() string {
	return fmt.Sprintf("%s: %v", s.t, s.err)
}

func (s *syncError) Unwrap() error {
	return s.err
}

func (s *syncError) Is(target error) bool {
	if target == nil {
		return s == nil
	}

	targetSyncErr, ok := target.(*syncError)
	if !ok {
		return false
	}

	return s.t == targetSyncErr.t
}
