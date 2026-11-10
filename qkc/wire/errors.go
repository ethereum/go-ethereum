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

package wire

import (
	"errors"

	"github.com/ethereum/go-ethereum/qkc/comm"
)

// ErrInvalidClusterFrame identifies a frame rejected before any bytes are written.
var ErrInvalidClusterFrame = errors.New("invalid cluster frame")

// ErrorCode converts a business rejection to its wire ErrorCode. Other errors
// are returned unchanged for the connection layer to treat as handler failures.
func ErrorCode(err error) (uint32, error) {
	if err == nil {
		return 0, nil
	}
	var remoteErr *comm.RemoteError
	if errors.As(err, &remoteErr) {
		return remoteErr.Code, nil
	}
	return 0, err
}
