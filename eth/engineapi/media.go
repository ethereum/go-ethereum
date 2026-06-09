// Copyright 2026 The go-ethereum Authors
// This file is part of the go-ethereum library.
//
// The go-ethereum library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The go-ethereum library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the go-ethereum library. If not, see <http://www.gnu.org/licenses/>.

package engineapi

import (
	"io"
	"net/http"
	"strings"

	"github.com/karalabe/ssz"
)

const (
	sszContentType  = "application/octet-stream"
	jsonContentType = "application/json"

	// maxPayloadBytes mirrors the capabilities advertisement
	// limits.payload.max_bytes (64 MiB). Per-endpoint helpers may lower it.
	maxPayloadBytes = 64 * 1024 * 1024
)

// readSSZRequest enforces the SSZ content-type and Content-Length cap, then
// decodes the request body into obj for the given fork. Writes the appropriate
// problem response on failure and returns false; the caller should return
// immediately.
func readSSZRequest(w http.ResponseWriter, r *http.Request, obj ssz.Object, fork ssz.Fork, max int64) bool {
	if !strings.EqualFold(r.Header.Get("Content-Type"), sszContentType) {
		writeProblem(w, http.StatusUnsupportedMediaType, ErrUnsupportedMedia, "expected "+sszContentType)
		return false
	}
	if r.ContentLength > max {
		writeProblem(w, http.StatusRequestEntityTooLarge, ErrRequestTooLarge, "")
		return false
	}
	body, err := io.ReadAll(io.LimitReader(r.Body, max+1))
	if err != nil {
		writeProblem(w, http.StatusBadRequest, ErrParseError, err.Error())
		return false
	}
	if int64(len(body)) > max {
		writeProblem(w, http.StatusRequestEntityTooLarge, ErrRequestTooLarge, "")
		return false
	}
	if err := ssz.DecodeFromBytesOnFork(body, obj, fork); err != nil {
		writeProblem(w, http.StatusBadRequest, ErrSSZDecode, "")
		return false
	}
	return true
}

// writeSSZResponse encodes obj into the response body for the given fork. On
// encode failure it writes an internal-error problem and returns false.
func writeSSZResponse(w http.ResponseWriter, obj ssz.Object, fork ssz.Fork) bool {
	buf := make([]byte, ssz.SizeOnFork(obj, fork))
	if err := ssz.EncodeToBytesOnFork(buf, obj, fork); err != nil {
		writeProblem(w, http.StatusInternalServerError, ErrInternal, err.Error())
		return false
	}
	w.Header().Set("Content-Type", sszContentType)
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(buf)
	return true
}
