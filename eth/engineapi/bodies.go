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
	"net/http"
	"strconv"

	sszt "github.com/ethereum/go-ethereum/beacon/engine/ssz"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/params/forks"
	"github.com/karalabe/ssz"
)

// handleBodiesByHash implements POST /engine/v2/{fork}/bodies/hash.
func (rt *Router) handleBodiesByHash(w http.ResponseWriter, r *http.Request, fork forks.Fork) {
	sf, ok := resolveFork(w, fork, forks.Paris)
	if !ok {
		return
	}
	req := new(sszt.BodiesByHashRequest)
	if !readSSZRequest(w, r, req, sf, maxPayloadBytes) {
		return
	}
	if len(req.BlockHashes) > sszt.MaxBodiesRequest {
		writeProblem(w, http.StatusRequestEntityTooLarge, ErrRequestTooLarge, "")
		return
	}
	bodies, timestamps := rt.backend.BodiesByHash(req.BlockHashes)
	writeSSZResponse(w, buildBodiesResponse(rt.backend, fork, sf, bodies, timestamps), sf)
}

// handleBodiesByRange implements GET /engine/v2/{fork}/bodies?from=&count=.
func (rt *Router) handleBodiesByRange(w http.ResponseWriter, r *http.Request, fork forks.Fork) {
	sf, ok := resolveFork(w, fork, forks.Paris)
	if !ok {
		return
	}
	q := r.URL.Query()
	from, err := strconv.ParseUint(q.Get("from"), 10, 64)
	if err != nil {
		writeProblem(w, http.StatusBadRequest, ErrInvalidRequest, "missing or bad from")
		return
	}
	count, err := strconv.ParseUint(q.Get("count"), 10, 64)
	if err != nil || count == 0 {
		writeProblem(w, http.StatusBadRequest, ErrInvalidRequest, "missing or bad count")
		return
	}
	if count > sszt.MaxBodiesRequest {
		writeProblem(w, http.StatusRequestEntityTooLarge, ErrRequestTooLarge, "")
		return
	}
	bodies, timestamps := rt.backend.BodiesByRange(from, count)
	writeSSZResponse(w, buildBodiesResponse(rt.backend, fork, sf, bodies, timestamps), sf)
}

// buildBodiesResponse assembles a BodiesResponse for the given fork, marking
// out-of-era blocks as available=false per the URL fork window. The body shape
// is fork-driven by the codec; bodyToSSZ populates the superset and the codec
// emits only the fork's active fields.
func buildBodiesResponse(b Backend, fork forks.Fork, sf ssz.Fork, bodies []*types.Body, ts []uint64) *sszt.BodiesResponse {
	out := &sszt.BodiesResponse{
		Entries: make([]*sszt.BodyEntry, len(bodies)),
	}
	for i, body := range bodies {
		entry := &sszt.BodyEntry{Body: new(sszt.ExecutionPayloadBody)}
		if body != nil && b.ForkFromTimestamp(ts[i]) == fork {
			entry.Available = true
			entry.Body = bodyToSSZ(body, sf)
		}
		out.Entries[i] = entry
	}
	return out
}

// bodyToSSZ flattens a *types.Body into the monolithic body shape. Withdrawals
// are only attached from Shanghai on, matching the fork's wire shape.
func bodyToSSZ(body *types.Body, sf ssz.Fork) *sszt.ExecutionPayloadBody {
	out := &sszt.ExecutionPayloadBody{
		Transactions: make([][]byte, len(body.Transactions)),
	}
	for i, tx := range body.Transactions {
		out.Transactions[i], _ = tx.MarshalBinary()
	}
	if sf >= ssz.ForkShapella {
		out.Withdrawals = make([]*sszt.Withdrawal, len(body.Withdrawals))
		for i, w := range body.Withdrawals {
			out.Withdrawals[i] = &sszt.Withdrawal{
				Index:          w.Index,
				ValidatorIndex: w.Validator,
				Address:        w.Address,
				Amount:         w.Amount,
			}
		}
	}
	// BlockAccessList is not yet on types.Body; once wired through, copy here
	// (gated by sf >= forkAmsterdam).
	return out
}
