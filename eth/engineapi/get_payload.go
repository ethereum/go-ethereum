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

// handleGetPayload implements GET /engine/v1/payloads/{payloadId} (fork via Eth-Execution-Version).
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

	// allowedForks is the header fork's era: the named fork plus any BPO forks
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
// the SSZ BuiltPayload shape for the header fork. BlockValue/Requests come straight
// across; the inner payload goes through the SSZ converter. The blobs bundle is
// emitted as V1 (Cancun/Prague) or V2 (Osaka+) per the fork; pre-Cancun forks
// carry no bundle (and no should_override_builder), so those fields are left
// nil and the codec drops them.
func buildBuiltPayloadAmsterdam(env *engine.ExecutionPayloadEnvelope, sf ssz.Fork) *sszt.BuiltPayloadAmsterdam {
	out := &sszt.BuiltPayloadAmsterdam{
		Payload:    sszt.ExecutionPayloadFromEngine(env.ExecutionPayload, sf),
		BlockValue: new(uint256.Int),
	}
	if env.BlockValue != nil {
		out.BlockValue.SetFromBig(env.BlockValue)
	}
	// execution_requests and should_override_builder exist from Prague and
	// Cancun respectively; the codec gates them, so it is safe to always set
	// the values the EL produced — the gated-off forks simply ignore them.
	if sszt.AtLeast(sf, forks.Prague) {
		out.ExecutionRequests = env.Requests
	}
	if sszt.AtLeast(sf, forks.Cancun) {
		override := env.Override
		out.ShouldOverrideBuilder = &override
	}
	// Select the bundle revision. Cancun/Prague use V1 (one proof per blob);
	// Osaka+ uses V2 (cell proofs). Pre-Cancun forks have no bundle.
	switch {
	case sszt.AtLeast(sf, forks.Osaka):
		if env.BlobsBundle != nil {
			out.BlobsBundleV2 = convertBlobsBundleV2(env.BlobsBundle)
		} else {
			out.BlobsBundleV2 = new(sszt.BlobsBundleV2)
		}
	case sszt.AtLeast(sf, forks.Cancun):
		if env.BlobsBundle != nil {
			out.BlobsBundleV1 = convertBlobsBundleV1(env.BlobsBundle)
		} else {
			out.BlobsBundleV1 = new(sszt.BlobsBundleV1)
		}
	}
	return out
}

// convertBlobsBundleV2 copies the JSON BlobsBundle into the SSZ V2 (cell-proof)
// layout. Inputs are length-validated by the caller's miner pipeline.
func convertBlobsBundleV2(b *engine.BlobsBundle) *sszt.BlobsBundleV2 {
	out := &sszt.BlobsBundleV2{
		Commitments: make([][48]byte, len(b.Commitments)),
		Proofs:      make([][48]byte, len(b.Proofs)),
		Blobs:       make([]*sszt.Blob, len(b.Blobs)),
	}
	fillBundle(out.Commitments, out.Proofs, out.Blobs, b)
	return out
}

// convertBlobsBundleV1 copies the JSON BlobsBundle into the SSZ V1 (single-proof)
// layout used by Cancun/Prague.
func convertBlobsBundleV1(b *engine.BlobsBundle) *sszt.BlobsBundleV1 {
	out := &sszt.BlobsBundleV1{
		Commitments: make([][48]byte, len(b.Commitments)),
		Proofs:      make([][48]byte, len(b.Proofs)),
		Blobs:       make([]*sszt.Blob, len(b.Blobs)),
	}
	fillBundle(out.Commitments, out.Proofs, out.Blobs, b)
	return out
}

// fillBundle copies the commitments/proofs/blobs from a JSON BlobsBundle into
// the destination SSZ slices (shared by the V1 and V2 converters, whose wire
// layout is identical).
func fillBundle(commitments, proofs [][48]byte, blobs []*sszt.Blob, b *engine.BlobsBundle) {
	for i, c := range b.Commitments {
		copy(commitments[i][:], c)
	}
	for i, p := range b.Proofs {
		copy(proofs[i][:], p)
	}
	for i, blob := range b.Blobs {
		blobs[i] = &sszt.Blob{Bytes: append([]byte(nil), blob...)}
	}
}
