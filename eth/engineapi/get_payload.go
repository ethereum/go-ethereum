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
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/params/forks"
	"github.com/holiman/uint256"
	"github.com/karalabe/ssz"
)

// handleGetPayload implements GET /engine/v2/{fork}/payloads/{payloadId}.
func (rt *Router) handleGetPayload(w http.ResponseWriter, r *http.Request, fork forks.Fork, idHex string) {
	sf, ok := resolveFork(w, fork, forks.Paris)
	if !ok {
		return
	}
	raw, err := hexutil.Decode(idHex)
	if err != nil || len(raw) != 8 {
		writeProblem(w, http.StatusBadRequest, ErrInvalidRequest, "malformed payloadId")
		return
	}
	var id engine.PayloadID
	copy(id[:], raw)

	// allowedForks is the URL fork's era: the named fork plus any BPO forks
	// that layer on it. The catalyst layer derives the cached payload's fork
	// via LatestFork, which can yield a BPO fork.
	env, err := rt.backend.GetPayload(id, eraForks(fork))
	if err != nil {
		mapBackendErr(w, err)
		return
	}

	out := buildBuiltPayloadAmsterdam(env, sf)
	w.Header().Set("Cache-Control", "no-store")
	writeSSZResponse(w, out, sf)
}

// buildBuiltPayloadAmsterdam packages an engine.ExecutionPayloadEnvelope into
// the SSZ BuiltPayload shape. BlockValue/BlobsBundle/Requests come straight
// across; the inner payload goes through the SSZ converter.
func buildBuiltPayloadAmsterdam(env *engine.ExecutionPayloadEnvelope, sf ssz.Fork) *sszt.BuiltPayloadAmsterdam {
	out := &sszt.BuiltPayloadAmsterdam{
		Payload:               sszt.ExecutionPayloadFromEngine(env.ExecutionPayload, sf),
		BlockValue:            new(uint256.Int),
		ExecutionRequests:     env.Requests,
		ShouldOverrideBuilder: env.Override,
	}
	if env.BlockValue != nil {
		out.BlockValue.SetFromBig(env.BlockValue)
	}
	if env.BlobsBundle != nil {
		out.BlobsBundle = convertBlobsBundle(env.BlobsBundle)
	} else {
		out.BlobsBundle = new(sszt.BlobsBundleV2)
	}
	return out
}

// convertBlobsBundle copies the JSON BlobsBundle into the SSZ V2 layout.
// Inputs are length-validated by the caller's miner pipeline.
func convertBlobsBundle(b *engine.BlobsBundle) *sszt.BlobsBundleV2 {
	out := &sszt.BlobsBundleV2{
		Commitments: make([][48]byte, len(b.Commitments)),
		Proofs:      make([][48]byte, len(b.Proofs)),
		Blobs:       make([]*sszt.Blob, len(b.Blobs)),
	}
	for i, c := range b.Commitments {
		copy(out.Commitments[i][:], c)
	}
	for i, p := range b.Proofs {
		copy(out.Proofs[i][:], p)
	}
	for i, blob := range b.Blobs {
		out.Blobs[i] = &sszt.Blob{Bytes: append([]byte(nil), blob...)}
	}
	return out
}
