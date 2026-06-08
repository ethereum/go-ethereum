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
)

// handleBlobs dispatches POST /engine/v2/blobs/v{1,2,3,4}.
// suffix is the path remainder after "/blobs/".
func (rt *Router) handleBlobs(w http.ResponseWriter, r *http.Request, suffix string) {
	if r.Method != http.MethodPost {
		writeProblem(w, http.StatusNotFound, ErrMethodNotFound, "")
		return
	}
	switch suffix {
	case "v1":
		rt.handleBlobsV1(w, r)
	case "v2":
		rt.handleBlobsV2(w, r, false)
	case "v3":
		rt.handleBlobsV2(w, r, true)
	case "v4":
		rt.handleBlobsV4(w, r)
	default:
		writeProblem(w, http.StatusNotFound, ErrMethodNotFound, "")
	}
}

func (rt *Router) handleBlobsV1(w http.ResponseWriter, r *http.Request) {
	req := new(sszt.BlobsVersionedHashesRequest)
	if !readSSZRequest(w, r, req, maxPayloadBytes) {
		return
	}
	if len(req.VersionedHashes) > sszt.MaxBlobsRequest {
		writeProblem(w, http.StatusRequestEntityTooLarge, ErrRequestTooLarge, "")
		return
	}
	_, v1, err := rt.backend.GetBlobs(req.VersionedHashes, false)
	if err != nil {
		mapBackendErr(w, err)
		return
	}
	// Backend returns nil entries for misses; map to available=false.
	resp := &sszt.BlobsV1Response{Entries: make([]*sszt.BlobV1Entry, len(v1))}
	for i, bp := range v1 {
		e := &sszt.BlobV1Entry{Contents: emptyBlobAndProofV1()}
		if bp != nil {
			e.Available = true
			e.Contents = &sszt.BlobAndProofV1{Blob: &sszt.Blob{Bytes: bp.Blob}}
			copy(e.Contents.Proof[:], bp.Proof)
		}
		resp.Entries[i] = e
	}
	writeSSZResponse(w, resp)
}

// handleBlobsV2 serves both /v2 (all-or-nothing) and /v3 (partial). The
// allowPartial flag selects between them.
func (rt *Router) handleBlobsV2(w http.ResponseWriter, r *http.Request, allowPartial bool) {
	req := new(sszt.BlobsVersionedHashesRequest)
	if !readSSZRequest(w, r, req, maxPayloadBytes) {
		return
	}
	if len(req.VersionedHashes) > sszt.MaxBlobsRequest {
		writeProblem(w, http.StatusRequestEntityTooLarge, ErrRequestTooLarge, "")
		return
	}
	v2, _, err := rt.backend.GetBlobs(req.VersionedHashes, true)
	if err != nil {
		mapBackendErr(w, err)
		return
	}
	if !allowPartial {
		// V2 is all-or-nothing: missing entries collapse the response to 204.
		for _, bp := range v2 {
			if bp == nil {
				w.WriteHeader(http.StatusNoContent)
				return
			}
		}
	}
	resp := &sszt.BlobsV2Response{Entries: make([]*sszt.BlobV2Entry, len(v2))}
	for i, bp := range v2 {
		e := &sszt.BlobV2Entry{Contents: emptyBlobAndProofV2()}
		if bp != nil {
			e.Available = true
			e.Contents = blobAndProofV2FromEngine(bp)
		}
		resp.Entries[i] = e
	}
	writeSSZResponse(w, resp)
}

// handleBlobsV4 implements POST /engine/v2/blobs/v4 (cell-range selection).
// The custody/cell-range logic is not yet plumbed into the txpool; we return
// 204 to signal the EL cannot serve it. The handler still validates the
// request body and the indices_bitarray length to keep the wire contract live.
func (rt *Router) handleBlobsV4(w http.ResponseWriter, r *http.Request) {
	req := new(sszt.BlobsV4Request)
	if !readSSZRequest(w, r, req, maxPayloadBytes) {
		return
	}
	if req.IndicesBitarray == nil || len(req.IndicesBitarray.Bytes) != sszt.CellsPerExtBlob/8 {
		writeProblem(w, http.StatusBadRequest, ErrInvalidBody, "indices_bitarray length")
		return
	}
	if len(req.VersionedHashes) > sszt.MaxBlobsRequest {
		writeProblem(w, http.StatusRequestEntityTooLarge, ErrRequestTooLarge, "")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func emptyBlobAndProofV1() *sszt.BlobAndProofV1 {
	return &sszt.BlobAndProofV1{Blob: &sszt.Blob{Bytes: make([]byte, sszt.BytesPerBlob)}}
}

func emptyBlobAndProofV2() *sszt.BlobAndProofV2 {
	return &sszt.BlobAndProofV2{Blob: &sszt.Blob{Bytes: make([]byte, sszt.BytesPerBlob)}}
}

func blobAndProofV2FromEngine(bp *engine.BlobAndProofV2) *sszt.BlobAndProofV2 {
	out := &sszt.BlobAndProofV2{
		Blob:   &sszt.Blob{Bytes: append([]byte(nil), bp.Blob...)},
		Proofs: make([][48]byte, len(bp.CellProofs)),
	}
	for i, p := range bp.CellProofs {
		copy(out.Proofs[i][:], p)
	}
	return out
}
