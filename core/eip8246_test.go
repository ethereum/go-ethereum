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
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/params"
)

// TestEIP8246SelfdestructNoBurn verifies that, once EIP-8246 is active
// (Amsterdam), a contract that is created and self-destructs to itself within
// the same transaction keeps its balance instead of burning it: the account
// survives as a balance-only account (no code, zero nonce, balance preserved).
//
// https://eips.ethereum.org/EIPS/eip-8246
func TestEIP8246SelfdestructNoBurn(t *testing.T) {
	var (
		key1, _ = crypto.HexToECDSA("b71c71a67e1177ad4e901695e1b4b9ee17ae16c6668d313eac2f96dbcda3f291")
		addr1   = crypto.PubkeyToAddress(key1.PublicKey)
		config  = *params.MergedTestChainConfig
		signer  = types.LatestSigner(&config)
		engine  = beacon.New(ethash.NewFaker())
		value   = big.NewInt(1_000_000)
		// Init code: ADDRESS (0x30) ; SELFDESTRUCT (0xff). The created contract
		// self-destructs to itself during its own creation transaction.
		initcode = common.FromHex("30ff")
	)
	// TODO: drop this hacky Amsterdam config initialization once the final
	// Amsterdam config is available (mirrors TestEthTransferLogs).
	config.AmsterdamTime = new(uint64)

	gspec := &Genesis{
		Config: &config,
		Alloc: addAmsterdamRequestPredeploys(types.GenesisAlloc{
			addr1: {Balance: newGwei(1_000_000_000)},
		}),
	}
	// The contract created by addr1's first (nonce 0) transaction.
	created := crypto.CreateAddress(addr1, 0)

	db, blocks, _ := GenerateChainWithGenesis(gspec, engine, 1, func(i int, b *BlockGen) {
		tx := types.MustSignNewTx(key1, signer, &types.DynamicFeeTx{
			ChainID:   gspec.Config.ChainID,
			Nonce:     0,
			To:        nil, // contract creation
			Gas:       1_000_000,
			GasFeeCap: newGwei(5),
			GasTipCap: newGwei(5),
			Value:     value,
			Data:      initcode,
		})
		b.AddTx(tx)
	})

	chain, err := NewBlockChain(db, gspec, engine, nil)
	if err != nil {
		t.Fatalf("failed to create chain: %v", err)
	}
	defer chain.Stop()
	// Read the post-state of the generated block directly. InsertChain is avoided
	// on purpose: it would additionally verify the EIP-7928 block access list,
	// which the chain-generation harness on this branch does not yet populate
	// consistently — an orthogonal concern to the EIP-8246 state semantics under
	// test here.
	state, err := chain.StateAt(blocks[0].Header())
	if err != nil {
		t.Fatalf("failed to obtain block state: %v", err)
	}
	// EIP-8246: the self-destructed, freshly-created contract keeps its balance
	// rather than burning it, so the account survives.
	if got := state.GetBalance(created); got.ToBig().Cmp(value) != 0 {
		t.Errorf("created account balance = %v, want %v (EIP-8246: balance must be preserved, not burned)", got, value)
	}
	// It survives as a balance-only account: nonce reset to 0 and no code.
	if got := state.GetNonce(created); got != 0 {
		t.Errorf("created account nonce = %d, want 0", got)
	}
	if got := state.GetCodeSize(created); got != 0 {
		t.Errorf("created account code size = %d, want 0 (code must be cleared)", got)
	}
}
