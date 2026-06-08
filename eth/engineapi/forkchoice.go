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

	"github.com/ethereum/go-ethereum/beacon/engine"
	sszt "github.com/ethereum/go-ethereum/beacon/engine/ssz"
	"github.com/ethereum/go-ethereum/params/forks"
)

// handleForkchoice implements POST /engine/v2/{fork}/forkchoice.
func (rt *Router) handleForkchoice(w http.ResponseWriter, r *http.Request, fork forks.Fork) {
	if fork != forks.Amsterdam {
		writeProblem(w, http.StatusBadRequest, ErrUnsupportedFork, "")
		return
	}
	fcu := new(sszt.ForkchoiceUpdateAmsterdam)
	if !readSSZRequest(w, r, fcu, maxPayloadBytes) {
		return
	}
	if err := fcu.Validate(); err != nil {
		writeProblem(w, http.StatusBadRequest, ErrInvalidBody, err.Error())
		return
	}
	state := sszt.ForkchoiceStateToV1(fcu.ForkchoiceState)
	var attrs *engine.PayloadAttributes
	if len(fcu.PayloadAttributes) == 1 {
		ssz := fcu.PayloadAttributes[0]
		// If PayloadAttributes is present the URL fork MUST match the fork
		// the new payload would belong to. Today only Amsterdam URL exists
		// in this implementation so the timestamp check is implicit; we
		// keep an explicit guard for future fork URLs.
		if rt.backend.ForkFromTimestamp(ssz.Timestamp) != fork {
			writeProblem(w, http.StatusBadRequest, ErrUnsupportedFork,
				"payload_attributes timestamp does not match URL fork")
			return
		}
		attrs = sszt.PayloadAttributesAmsterdamToEngine(ssz)
		// target_gas_limit and custody_columns are not yet plumbed into
		// the JSON-RPC engine API. Custody is parsed-but-stubbed per
		// agreed scope; target_gas_limit will be picked up when the
		// underlying miner gains the corresponding setting.
	}

	resp, err := rt.backend.ForkchoiceUpdated(r.Context(), state, attrs, engine.PayloadV4)
	if err != nil {
		mapBackendErr(w, err)
		return
	}
	// /forkchoice MUST NOT return ACCEPTED.
	if resp.PayloadStatus.Status == engine.ACCEPTED {
		resp.PayloadStatus.Status = engine.INVALID
	}
	out := &sszt.ForkchoiceUpdateResponseAmsterdam{
		PayloadStatus: sszt.PayloadStatusFromV1(&resp.PayloadStatus),
	}
	if resp.PayloadID != nil {
		out.PayloadID = [][8]byte{[8]byte(*resp.PayloadID)}
	}
	writeSSZResponse(w, out)
}
