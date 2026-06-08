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
	"encoding/json"
	"net/http"
)

// Error type URIs from refactor.md § Error model.
const (
	ErrParseError          = "/engine-api/errors/parse-error"
	ErrInvalidRequest      = "/engine-api/errors/invalid-request"
	ErrSSZDecode           = "/engine-api/errors/ssz-decode-error"
	ErrUnsupportedFork     = "/engine-api/errors/unsupported-fork"
	ErrMethodNotFound      = "/engine-api/errors/method-not-found"
	ErrUnknownPayload      = "/engine-api/errors/unknown-payload"
	ErrInvalidForkchoice   = "/engine-api/errors/invalid-forkchoice"
	ErrReorgTooDeep        = "/engine-api/errors/reorg-too-deep"
	ErrRequestTooLarge     = "/engine-api/errors/request-too-large"
	ErrUnsupportedMedia    = "/engine-api/errors/unsupported-media-type"
	ErrInvalidBody         = "/engine-api/errors/invalid-body"
	ErrInvalidAttributes   = "/engine-api/errors/invalid-attributes"
	ErrInternal            = "/engine-api/errors/internal"
)

const problemJSONContentType = "application/problem+json"

// problem is the RFC 7807 body shape. Only type and detail are populated,
// matching the spec's two-field profile.
type problem struct {
	Type   string `json:"type"`
	Detail string `json:"detail,omitempty"`
}

// writeProblem emits an RFC 7807 error body. detail is optional and may be empty.
func writeProblem(w http.ResponseWriter, status int, typeURI, detail string) {
	w.Header().Set("Content-Type", problemJSONContentType)
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(problem{Type: typeURI, Detail: detail})
}
