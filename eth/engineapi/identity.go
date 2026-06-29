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

	"github.com/ethereum/go-ethereum/beacon/engine"
)

// handleIdentity implements GET /engine/v1/identity.
// The CL surfaces itself via the X-Engine-Client-Version request header; the
// EL responds with its own ClientVersion in JSON.
func (rt *Router) handleIdentity(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeProblem(w, http.StatusNotFound, ErrMethodNotFound, "")
		return
	}
	cv := rt.backend.ClientVersion()
	w.Header().Set("Content-Type", jsonContentType)
	_ = json.NewEncoder(w).Encode(struct {
		Versions []engine.ClientVersionV1 `json:"versions"`
	}{Versions: []engine.ClientVersionV1{cv}})
}
