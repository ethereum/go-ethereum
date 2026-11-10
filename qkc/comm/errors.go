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

package comm

import "fmt"

// RemoteError reports a nonzero business error code returned by a remote peer.
// Callers receive it as an error so that a rejected request is never mistaken
// for success by code that only checks err. Code keeps the peer's original
// error code available for callers that need to distinguish results.
type RemoteError struct {
	Code uint32
}

func (e *RemoteError) Error() string {
	return fmt.Sprintf("remote error code 0x%x", e.Code)
}
