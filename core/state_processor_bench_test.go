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
	"context"
	"crypto/ecdsa"
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/consensus/beacon"
	"github.com/ethereum/go-ethereum/consensus/ethash"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/params"
)

// benchPureTransferBlock generates a genesis spec and a single block packed
// with as many 21000-gas pure value transfers as fit in the block gas limit.
// All recipients are freshly-derived EOAs with no code, so every transaction
// is a "pure transfer".
func benchPureTransferBlock(b *testing.B, naccounts int) (*Genesis, *types.Block) {
	// Derive a set of funded keys.
	keys := make([]*ecdsa.PrivateKey, naccounts)
	addrs := make([]common.Address, naccounts)
	keys[0] = benchRootKey
	addrs[0] = benchRootAddr
	for i := 1; i < naccounts; i++ {
		k, _ := crypto.GenerateKey()
		keys[i] = k
		addrs[i] = crypto.PubkeyToAddress(k.PublicKey)
	}

	alloc := make(types.GenesisAlloc, naccounts)
	for _, a := range addrs {
		alloc[a] = types.Account{Balance: benchRootFunds}
	}
	gspec := &Genesis{
		Config:   params.TestChainConfig,
		GasLimit: 200_000_000,
		Alloc:    alloc,
	}

	from := 0
	_, blocks, _ := GenerateChainWithGenesis(gspec, beacon.New(ethash.NewFaker()), 1, func(i int, gen *BlockGen) {
		gas := gen.header.GasLimit
		gasPrice := big.NewInt(0)
		if gen.header.BaseFee != nil {
			gasPrice = gen.header.BaseFee
		}
		signer := gen.Signer()
		for gas >= params.TxGas {
			gas -= params.TxGas
			to := addrs[(from+1)%naccounts]
			tx, err := types.SignNewTx(keys[from], signer, &types.LegacyTx{
				Nonce:    gen.TxNonce(addrs[from]),
				To:       &to,
				Value:    big.NewInt(1),
				Gas:      params.TxGas,
				GasPrice: gasPrice,
			})
			if err != nil {
				b.Fatal(err)
			}
			gen.AddTx(tx)
			from = (from + 1) % naccounts
		}
	})
	return gspec, blocks[0]
}

// benchmarkProcess builds a block of pure transfers and times repeated
// Process calls against a fresh state derived from genesis each iteration,
// isolating the cost of execution from disk commit / trie hashing.
func benchmarkProcess(b *testing.B, naccounts int) {
	gspec, block := benchPureTransferBlock(b, naccounts)

	db := rawdb.NewMemoryDatabase()
	chain, err := NewBlockChain(db, gspec, beacon.New(ethash.NewFaker()), nil)
	if err != nil {
		b.Fatal(err)
	}
	defer chain.Stop()

	processor := chain.Processor()
	genesisHeader := chain.Genesis().Header()

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		statedb, err := chain.StateAt(genesisHeader)
		if err != nil {
			b.Fatal(err)
		}
		if _, err := processor.Process(context.Background(), block, statedb, nil, vm.Config{}); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkProcessPureTransfers(b *testing.B) {
	benchmarkProcess(b, 1000)
}
