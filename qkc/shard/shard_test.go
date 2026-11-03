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
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"
)

func TestShardCreateBlockToMine(t *testing.T) {
	testPoWMiner(t, func(t *testing.T) {
		env, _, _ := newMiningEnv(t, 1)
		s := NewShard(env.sc)
		coinbase := common.Address{0xaa}
		block, diff, err := s.CreateBlockToMine(context.Background(), coinbase)
		if err != nil {
			t.Fatal(err)
		}
		if block.ParentHash() != env.genesis.Hash() || block.Coinbase() != coinbase || block.Time() != uint64(time.Now().Unix()) {
			t.Fatal("candidate was not built on the head for coinbase at the current time")
		}
		// Without staking, the seal must meet the base difficulty.
		if diff.Sign() <= 0 || diff.Cmp(block.Header().QKCDifficulty()) != 0 {
			t.Fatalf("difficulty %v, want base difficulty %v", diff, block.Header().QKCDifficulty())
		}
		if err := s.InsertMinedBlock(block); err != nil {
			t.Fatal(err)
		}
		if s.CurrentBlock().Hash() != block.Hash() {
			t.Fatal("mined block did not become the head")
		}
		// A candidate built within the same second still follows its parent.
		next, _, err := s.CreateBlockToMine(context.Background(), coinbase)
		if err != nil {
			t.Fatal(err)
		}
		if next.ParentHash() != block.Hash() || next.Time() != block.Time()+1 {
			t.Fatalf("candidate time %d does not follow parent time %d", next.Time(), block.Time())
		}
	})
}
