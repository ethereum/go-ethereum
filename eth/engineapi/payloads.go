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
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/params/forks"
)

// handleNewPayload implements POST /engine/v2/{fork}/payloads.
func (rt *Router) handleNewPayload(w http.ResponseWriter, r *http.Request, fork forks.Fork) {
	if fork != forks.Amsterdam {
		writeProblem(w, http.StatusBadRequest, ErrUnsupportedFork, "")
		return
	}
	env := new(sszt.ExecutionPayloadEnvelopeAmsterdam)
	if !readSSZRequest(w, r, env, maxPayloadBytes) {
		return
	}
	data := sszt.ExecutionPayloadAmsterdamToEngine(env.Payload)

	// The spec drops expectedBlobVersionedHashes; we recompute them from the
	// payload's transactions before passing to the EL.
	versionedHashes, err := versionedHashesFromTxs(data.Transactions)
	if err != nil {
		writeProblem(w, http.StatusUnprocessableEntity, ErrInvalidBody, err.Error())
		return
	}
	root := env.ParentBeaconBlockRoot
	status, err := rt.backend.NewPayload(r.Context(), *data, versionedHashes, &root, env.ExecutionRequests)
	if err != nil {
		mapBackendErr(w, err)
		return
	}
	writeSSZResponse(w, sszt.PayloadStatusFromV1(&status))
}

// versionedHashesFromTxs extracts the blob versioned-hash list from the
// payload's RLP-encoded transactions. The CL no longer sends it; the EL
// recomputes for the block-hash check.
func versionedHashesFromTxs(raw [][]byte) ([]common.Hash, error) {
	txs, err := engine.DecodeTransactions(raw)
	if err != nil {
		return nil, err
	}
	var hashes []common.Hash
	for _, tx := range txs {
		hashes = append(hashes, tx.BlobHashes()...)
	}
	return hashes, nil
}

// mapBackendErr translates engine-level errors into the spec's error model.
func mapBackendErr(w http.ResponseWriter, err error) {
	switch err {
	case engine.UnknownPayload:
		writeProblem(w, http.StatusNotFound, ErrUnknownPayload, "")
	case engine.InvalidForkChoiceState:
		writeProblem(w, http.StatusConflict, ErrInvalidForkchoice, "")
	case engine.InvalidPayloadAttributes:
		writeProblem(w, http.StatusUnprocessableEntity, ErrInvalidAttributes, "")
	case engine.UnsupportedFork:
		writeProblem(w, http.StatusBadRequest, ErrUnsupportedFork, "")
	case engine.TooLargeRequest:
		writeProblem(w, http.StatusRequestEntityTooLarge, ErrRequestTooLarge, "")
	case engine.InvalidParams:
		writeProblem(w, http.StatusBadRequest, ErrInvalidRequest, "")
	default:
		writeProblem(w, http.StatusInternalServerError, ErrInternal, err.Error())
	}
}
