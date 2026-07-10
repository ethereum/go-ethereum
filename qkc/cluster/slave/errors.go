// Copyright 2026-2027, QuarkChain.

package slave

import (
	"errors"
)

var (
	// ErrConnectionClosed is returned when an operation is attempted on a
	// connection that has already been closed.
	ErrConnectionClosed = errors.New("connection closed")

	// ErrNotActive is returned when an RPC is attempted on a connection that
	// has not been started (state != ACTIVE).
	ErrNotActive = errors.New("connection not active")
)
