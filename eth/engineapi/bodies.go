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
)

// handleBodiesByHash implements POST /engine/v2/{fork}/bodies/hash.
func (rt *Router) handleBodiesByHash(w http.ResponseWriter, r *http.Request, fork forks.Fork) {
	if fork != forks.Amsterdam {
		writeProblem(w, http.StatusBadRequest, ErrUnsupportedFork, "")
		return
	}
	req := new(sszt.BodiesByHashRequest)
	if !readSSZRequest(w, r, req, maxPayloadBytes) {
		return
	}
	if len(req.BlockHashes) > sszt.MaxBodiesRequest {
		writeProblem(w, http.StatusRequestEntityTooLarge, ErrRequestTooLarge, "")
		return
	}
	bodies, timestamps := rt.backend.BodiesByHash(req.BlockHashes)
	writeSSZResponse(w, buildBodiesResponseAmsterdam(rt.backend, fork, bodies, timestamps))
}

// handleBodiesByRange implements GET /engine/v2/{fork}/bodies?from=&count=.
func (rt *Router) handleBodiesByRange(w http.ResponseWriter, r *http.Request, fork forks.Fork) {
	if fork != forks.Amsterdam {
		writeProblem(w, http.StatusBadRequest, ErrUnsupportedFork, "")
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
	writeSSZResponse(w, buildBodiesResponseAmsterdam(rt.backend, fork, bodies, timestamps))
}

// buildBodiesResponseAmsterdam assembles a BodiesResponseAmsterdam, marking
// out-of-era blocks as available=false per the URL fork window.
func buildBodiesResponseAmsterdam(b Backend, fork forks.Fork, bodies []*types.Body, ts []uint64) *sszt.BodiesResponseAmsterdam {
	out := &sszt.BodiesResponseAmsterdam{
		Entries: make([]*sszt.BodyEntryAmsterdam, len(bodies)),
	}
	for i, body := range bodies {
		entry := &sszt.BodyEntryAmsterdam{Body: new(sszt.ExecutionPayloadBodyAmsterdam)}
		if body != nil && b.ForkFromTimestamp(ts[i]) == fork {
			entry.Available = true
			entry.Body = bodyToAmsterdamSSZ(body)
		}
		out.Entries[i] = entry
	}
	return out
}

// bodyToAmsterdamSSZ flattens a *types.Body into the Amsterdam body shape.
func bodyToAmsterdamSSZ(body *types.Body) *sszt.ExecutionPayloadBodyAmsterdam {
	out := &sszt.ExecutionPayloadBodyAmsterdam{
		Transactions: make([][]byte, len(body.Transactions)),
		Withdrawals:  make([]*sszt.Withdrawal, len(body.Withdrawals)),
	}
	for i, tx := range body.Transactions {
		out.Transactions[i], _ = tx.MarshalBinary()
	}
	for i, w := range body.Withdrawals {
		out.Withdrawals[i] = &sszt.Withdrawal{
			Index:          w.Index,
			ValidatorIndex: w.Validator,
			Address:        w.Address,
			Amount:         w.Amount,
		}
	}
	// BlockAccessList is not yet on types.Body; once wired through, copy here.
	return out
}
