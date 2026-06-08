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

	sszt "github.com/ethereum/go-ethereum/beacon/engine/ssz"
)

// capabilitiesResponse mirrors the structured shape in refactor.md §
// "Capabilities format".
type capabilitiesResponse struct {
	SupportedForks         []string            `json:"supported_forks"`
	ForkScopedEndpoints    []string            `json:"fork_scoped_endpoints"`
	IndependentlyVersioned map[string][]string `json:"independently_versioned"`
	UnscopedEndpoints      []string            `json:"unscoped_endpoints"`
	Limits                 map[string]uint64   `json:"limits"`
}

// handleCapabilities implements GET /engine/v2/capabilities.
func (rt *Router) handleCapabilities(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeProblem(w, http.StatusNotFound, ErrMethodNotFound, "")
		return
	}
	resp := capabilitiesResponse{
		SupportedForks:      []string{"amsterdam"},
		ForkScopedEndpoints: []string{"payloads", "forkchoice", "bodies"},
		IndependentlyVersioned: map[string][]string{
			"blobs": {"v1", "v2", "v3", "v4"},
		},
		UnscopedEndpoints: []string{"capabilities", "identity"},
		Limits: map[string]uint64{
			"bodies.max_count":           sszt.MaxBodiesRequest,
			"blobs.max_versioned_hashes": sszt.MaxBlobsRequest,
			"payload.max_bytes":          maxPayloadBytes,
		},
	}
	w.Header().Set("Content-Type", jsonContentType)
	_ = json.NewEncoder(w).Encode(resp)
}
