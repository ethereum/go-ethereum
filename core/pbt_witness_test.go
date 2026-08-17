// Copyright 2024 The go-ethereum Authors
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
	"encoding/binary"
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/tracing"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/core/types/bal"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/params"
	"github.com/ethereum/go-ethereum/triedb"
)

// Despite the filename this covers the EIP-2935 history contract; witnesses
// are pinned in pbt_capabilities_test.go, block processing by the EEST
// fixtures.

func TestProcessParentBlockHash(t *testing.T) {
	// This test uses blocks where,
	// block 1 parent hash is 0x0100....
	// block 2 parent hash is 0x0200....
	// etc
	checkBlockHashes := func(statedb *state.StateDB, isPBT bool) {
		statedb.SetNonce(params.HistoryStorageAddress, 1, tracing.NonceChangeUnspecified)
		statedb.SetCode(params.HistoryStorageAddress, params.HistoryStorageCode, tracing.CodeChangeUnspecified)
		// Process n blocks, from 1 .. num
		var num = 2
		for i := 1; i <= num; i++ {
			header := &types.Header{ParentHash: common.Hash{byte(i)}, Number: big.NewInt(int64(i)), Difficulty: new(big.Int)}
			chainConfig := params.MergedTestChainConfig
			if isPBT {
				chainConfig = testPBTChainConfig
			}
			vmContext := NewEVMBlockContext(header, &BlockChain{chainConfig: chainConfig}, new(common.Address))
			evm := vm.NewEVM(vmContext, statedb, chainConfig, vm.Config{})
			ProcessParentBlockHash(header.ParentHash, evm, bal.NewConstructionBlockAccessList())
		}
		// Read block hashes for block 0 .. num-1
		for i := 0; i < num; i++ {
			have, want := getContractStoredBlockHash(statedb, uint64(i)), common.Hash{byte(i + 1)}
			if have != want {
				t.Errorf("block %d, pbt=%v, have parent hash %v, want %v", i, isPBT, have, want)
			}
		}
	}
	t.Run("MPT", func(t *testing.T) {
		statedb, _ := state.New(types.EmptyRootHash, state.NewDatabaseForTesting())
		checkBlockHashes(statedb, false)
	})
	t.Run("PBT", func(t *testing.T) {
		db := rawdb.NewMemoryDatabase()
		cacheConfig := DefaultConfig().WithStateScheme(rawdb.PathScheme)
		cacheConfig.SnapshotLimit = 0
		tdbConfig, err := cacheConfig.triedbConfig(true)
		if err != nil {
			t.Fatal(err)
		}
		triedb := triedb.NewDatabase(db, tdbConfig)
		statedb, _ := state.New(types.EmptyBinaryHash, state.NewDatabase(triedb, nil))
		checkBlockHashes(statedb, true)
	})
}

// getContractStoredBlockHash is a utility method which reads the stored parent blockhash for block 'number'
func getContractStoredBlockHash(statedb *state.StateDB, number uint64) common.Hash {
	ringIndex := number % params.HistoryServeWindow
	var key common.Hash
	binary.BigEndian.PutUint64(key[24:], ringIndex)
	return statedb.GetState(params.HistoryStorageAddress, key)
}
