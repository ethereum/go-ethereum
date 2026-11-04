// Copyright 2026 The quarkchain Authors
// This file is part of the quarkchain library.
//
// The quarkchain library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The quarkchain library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the quarkchain library. If not, see <http://www.gnu.org/licenses/>.

package shard

import (
	"context"
	"math/rand"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
)

// simulatedSearch seals work without hashing, like pyquarkchain's POW_SIMULATE
// consensus. It waits for an exponentially distributed delay after the
// candidate's timestamp, as PoW block intervals are, and returns a random
// nonce. Chains mined this way must not verify seals.
//
// TODO: In the New/Init PR, add ConsensusType to params.QuarkChainConfig and have
// ShardBackend.New instantiate posw.New or posw.NewSimulate accordingly, defaulting
// to PoSW. Nodes on the same chain must agree because this changes seal validity.
type simulatedSearch struct {
	blockTime time.Duration
}

// NewSimulatedSearch returns a search sealing work blockTime after its
// timestamp on average.
func NewSimulatedSearch(blockTime time.Duration) NonceSearch {
	return simulatedSearch{blockTime: blockTime}
}

func (s simulatedSearch) Search(ctx context.Context, work *MiningWork) (types.BlockNonce, common.Hash, bool) {
	delay := time.Duration(rand.ExpFloat64() * float64(s.blockTime))
	timer := time.NewTimer(time.Until(time.Unix(int64(work.Time), 0).Add(delay)))
	defer timer.Stop()
	select {
	case <-timer.C:
		return types.EncodeNonce(rand.Uint64()), common.Hash{}, true
	case <-ctx.Done():
		return types.BlockNonce{}, common.Hash{}, false
	}
}
