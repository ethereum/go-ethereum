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
	"github.com/karalabe/ssz"
)

// resolveFork maps the header fork onto the ssz.Fork the codec multiplexes on,
// enforcing that it is at least min. On failure it writes the appropriate
// problem response and returns ok=false; the caller should return immediately.
func resolveFork(w http.ResponseWriter, fork, min forks.Fork) (ssz.Fork, bool) {
	if fork < min {
		writeProblem(w, http.StatusBadRequest, ErrUnsupportedFork, "")
		return ssz.ForkUnknown, false
	}
	sf, ok := sszt.ForkFor(fork)
	if !ok {
		writeProblem(w, http.StatusBadRequest, ErrUnsupportedFork, "")
		return ssz.ForkUnknown, false
	}
	return sf, true
}

// baseFork collapses a BPO fork onto the named fork that it layers on. BPO1..5
// sit between Osaka and Amsterdam in params/forks but have no Engine API fork
// header value of their own: a chain in a BPO era still negotiates
// Eth-Execution-Version: osaka. Named forks map to themselves.
func baseFork(f forks.Fork) forks.Fork {
	switch f {
	case forks.BPO1, forks.BPO2, forks.BPO3, forks.BPO4, forks.BPO5:
		return forks.Osaka
	default:
		return f
	}
}

// eraForks returns the set of params/forks values that share a header fork's
// wire shape: the named fork plus any BPO forks that layer on it. checkFork in
// the catalyst layer derives a cached payload's fork via LatestFork, which can
// return a BPO fork, so the allowed set passed to GetPayload must include them.
func eraForks(fork forks.Fork) []forks.Fork {
	if fork == forks.Osaka {
		return []forks.Fork{forks.Osaka, forks.BPO1, forks.BPO2, forks.BPO3, forks.BPO4, forks.BPO5}
	}
	return []forks.Fork{fork}
}

// payloadVersionFor selects the JSON-RPC PayloadVersion that the catalyst layer
// expects for a given header fork, mirroring the per-fork version table on the
// JSON-RPC ForkchoiceUpdatedVx path (Amsterdam -> V4, Cancun..Osaka -> V3,
// Shanghai -> V2, Paris -> V1).
func payloadVersionFor(fork forks.Fork) engine.PayloadVersion {
	switch {
	case fork >= forks.Amsterdam:
		return engine.PayloadV4
	case fork >= forks.Cancun:
		return engine.PayloadV3
	case fork >= forks.Shanghai:
		return engine.PayloadV2
	default:
		return engine.PayloadV1
	}
}
