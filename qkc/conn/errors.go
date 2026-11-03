// Copyright 2026-2027 The quarkchain Authors
// This file is part of the quarkchain library.
//
// The quarkchain library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The quarkchain library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the quarkchain library. If not, see <http://www.gnu.org/licenses/>.

package conn

import "errors"

var (
	// ErrConnectionClosed is returned when an operation is attempted on a
	// connection that has already been closed, or wrapped when a write failure
	// closes the connection. Use errors.Is to check for it; write failures also
	// preserve the underlying transport error.
	ErrConnectionClosed = errors.New("connection closed")

	// ErrNotActive is returned when an RPC is attempted on a connection that
	// has not been started (state != ACTIVE).
	ErrNotActive = errors.New("connection not active")
)
