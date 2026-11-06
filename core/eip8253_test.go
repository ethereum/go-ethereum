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

package core

import (
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/consensus/beacon"
	"github.com/ethereum/go-ethereum/consensus/ethash"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/params"
)

// TestEIP8253NonceBump checks that the listed accounts get a nonce of one in the
// first Bogota block only, recorded at block access index zero.
func TestEIP8253NonceBump(t *testing.T) {
	var (
		accounts = zeroNonceStorageAccounts[params.MainnetChainConfig.ChainID.Uint64()]
		target   = accounts[0]
		value    = common.HexToHash("0x2a")
	)
	tests := []struct {
		name       string
		chainID    *big.Int
		bogotaTime uint64
		nonce      uint64
	}{
		{"fork block", params.MainnetChainConfig.ChainID, 10, 1},
		{"bogota at genesis", params.MainnetChainConfig.ChainID, 0, 0},
		{"chain without a list", params.SepoliaChainConfig.ChainID, 10, 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// One listed account has the pre-EIP-161 shape, the rest are absent.
			env := newBALTestEnv(types.GenesisAlloc{
				target: {Balance: big.NewInt(1), Storage: map[common.Hash]common.Hash{{}: value}},
			})
			env.cfg.ChainID = tt.chainID
			env.cfg.BogotaTime = &tt.bogotaTime

			// Block 1 is at time 10. Importing it checks its state root and access list.
			engine := beacon.New(ethash.NewFaker())
			_, blocks, _ := GenerateChainWithGenesis(env.gspec, engine, 1, nil)
			assertParallelEquiv(t, env.gspec, engine, blocks[0])

			chain, err := NewBlockChain(rawdb.NewMemoryDatabase(), env.gspec, engine, nil)
			if err != nil {
				t.Fatalf("new blockchain: %v", err)
			}
			defer chain.Stop()
			if _, err := chain.InsertChain(blocks); err != nil {
				t.Fatalf("insert chain: %v", err)
			}
			state, err := chain.State()
			if err != nil {
				t.Fatalf("state: %v", err)
			}
			for _, addr := range accounts {
				if got := state.GetNonce(addr); got != tt.nonce {
					t.Fatalf("nonce of %x: got %d, want %d", addr, got, tt.nonce)
				}
			}
			if got := state.GetState(target, common.Hash{}); got != value {
				t.Fatalf("target storage: got %x, want %x", got, value)
			}
			// The bump shares block access index zero with the system calls.
			if tt.nonce == 1 {
				aa := findAccount(blocks[0].AccessList(), target)
				if aa == nil || len(aa.NonceChanges) != 1 || aa.NonceChanges[0].BlockAccessIndex != 0 {
					t.Fatalf("target nonce change not at index 0\n%s", blocks[0].AccessList().PrettyPrint())
				}
			}
		})
	}
}
