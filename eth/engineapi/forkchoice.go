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

// handleForkchoice implements POST /engine/v1/forkchoice (fork via Eth-Execution-Version).
func (rt *Router) handleForkchoice(w http.ResponseWriter, r *http.Request, fork forks.Fork) {
	// The forkchoice envelope shape is fork-driven by the codec; every fork
	// from Paris on has a valid wire shape.
	sf, ok := resolveFork(w, fork, forks.Paris)
	if !ok {
		return
	}
	fcu := new(sszt.ForkchoiceUpdateAmsterdam)
	if !readSSZRequest(w, r, fcu, sf, maxPayloadBytes) {
		return
	}
	if err := fcu.Validate(); err != nil {
		writeProblem(w, http.StatusBadRequest, ErrInvalidBody, err.Error())
		return
	}
	state := sszt.ForkchoiceStateToV1(fcu.ForkchoiceState)
	var attrs *engine.PayloadAttributes
	if len(fcu.PayloadAttributes) == 1 {
		attr := fcu.PayloadAttributes[0]
		if err := attr.Validate(sf); err != nil {
			writeProblem(w, http.StatusBadRequest, ErrInvalidAttributes, err.Error())
			return
		}
		// If PayloadAttributes is present the Eth-Execution-Version header MUST
		// match the fork the new payload would belong to. ForkFromTimestamp can
		// return a BPO fork (which has no header value of its own); collapse it
		// onto the named fork it layers on before comparing.
		if baseFork(rt.backend.ForkFromTimestamp(attr.Timestamp)) != fork {
			writeProblem(w, http.StatusBadRequest, ErrUnsupportedFork,
				"payload_attributes timestamp does not match Eth-Execution-Version header")
			return
		}
		attrs = sszt.PayloadAttributesToEngine(attr)
		// target_gas_limit and custody_columns are not yet plumbed into
		// the JSON-RPC engine API. Custody is parsed-but-stubbed per
		// agreed scope; target_gas_limit will be picked up when the
		// underlying miner gains the corresponding setting.
	}

	resp, err := rt.backend.ForkchoiceUpdated(r.Context(), state, attrs, payloadVersionFor(fork))
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
	writeSSZResponse(w, out, sf)
}
