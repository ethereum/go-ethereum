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
// survives as a balance-only account (no code, zero nonce, balance preserved)
// whose storage is cleared at transaction finalization.
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
		slot    = common.BigToHash(big.NewInt(0x05))
		// Init code: SSTORE(5, 0x2a); ADDRESS (0x30); SELFDESTRUCT (0xff). The
		// created contract stores a value and self-destructs to itself during
		// its own creation transaction.
		initcode = []byte{0x60, 0x2a, 0x60, 0x05, 0x55, 0x30, 0xff}
	)
	// TODO: drop this hacky Amsterdam config initialization once the final
	// Amsterdam config is available (mirrors TestEthTransferLogs).
	config.AmsterdamTime = new(uint64)

	gspec := &Genesis{
		Config: &config,
		Alloc: types.GenesisAlloc{
			addr1: {Balance: newGwei(1_000_000_000)},
		},
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
	if got := state.GetState(created, slot); got != (common.Hash{}) {
		t.Errorf("created storage slot %x = %x, want 0 (storage must be cleared)", slot, got)
	}
}

// TestEIP8246SelfdestructRefunded verifies that ETH sent back to a
// same-transaction selfdestructed account is retained at finalization instead
// of being burned. The factory funds the account twice after SELFDESTRUCT.
func TestEIP8246SelfdestructRefunded(t *testing.T) {
	var (
		key, _      = crypto.HexToECDSA("b71c71a67e1177ad4e901695e1b4b9ee17ae16c6668d313eac2f96dbcda3f291")
		sender      = crypto.PubkeyToAddress(key.PublicKey)
		factory     = common.HexToAddress("0xfac8246")
		beneficiary = common.HexToAddress("0xbeef")
		config      = *params.MergedTestChainConfig
		signer      = types.LatestSigner(&config)
		engine      = beacon.New(ethash.NewFaker())
	)
	config.AmsterdamTime = new(uint64)
	// The child initcode selfdestructs to another account. The factory then
	// sends it 7 and 8 wei after it is marked for selfdestruction.
	childInit := append([]byte{0x73}, beneficiary.Bytes()...)
	childInit = append(childInit, 0xff)
	var word [32]byte
	copy(word[32-len(childInit):], childInit)
	factoryCode := append([]byte{0x7f}, word[:]...)
	factoryCode = append(factoryCode, 0x5f, 0x52) // PUSH0; MSTORE
	// CREATE(value=0, offset=10, length=22), then store the returned address.
	factoryCode = append(factoryCode, 0x60, byte(len(childInit)), 0x60, byte(32-len(childInit)), 0x5f, 0xf0, 0x5f, 0x52)
	for _, amount := range []byte{7, 8} {
		// CALL(child, value=amount) with empty input/output.
		factoryCode = append(factoryCode, 0x5f, 0x5f, 0x5f, 0x5f, 0x60, amount, 0x5f, 0x51, 0x5a, 0xf1, 0x50)
	}
	factoryCode = append(factoryCode, 0x00)
	gspec := &Genesis{
		Config: &config,
		Alloc: types.GenesisAlloc{
			sender:  {Balance: newGwei(1_000_000_000)},
			factory: {Nonce: 1, Code: factoryCode, Balance: big.NewInt(15)},
		},
	}
	child := crypto.CreateAddress(factory, 1)
	db, blocks, _ := GenerateChainWithGenesis(gspec, engine, 1, func(_ int, b *BlockGen) {
		b.AddTx(types.MustSignNewTx(key, signer, &types.DynamicFeeTx{
			ChainID:   gspec.Config.ChainID,
			Nonce:     0,
			To:        &factory,
			Gas:       1_000_000,
			GasFeeCap: newGwei(5),
			GasTipCap: newGwei(5),
		}))
	})
	chain, err := NewBlockChain(db, gspec, engine, nil)
	if err != nil {
		t.Fatalf("failed to create chain: %v", err)
	}
	defer chain.Stop()
	state, err := chain.StateAt(blocks[0].Header())
	if err != nil {
		t.Fatalf("failed to obtain block state: %v", err)
	}
	if got := state.GetBalance(child).Uint64(); got != 15 {
		t.Errorf("refunded child balance = %d, want 15", got)
	}
	if got := state.GetNonce(child); got != 0 {
		t.Errorf("refunded child nonce = %d, want 0", got)
	}
	if got := state.GetCodeSize(child); got != 0 {
		t.Errorf("refunded child code size = %d, want 0", got)
	}
}

// TestEIP8246Create2RecreatesBalanceOnly verifies that an EIP-8246
// balance-only account does not block recreating the same CREATE2 address in a
// later transaction. The second creation contributes another wei to the
// preserved balance, proving that it executed rather than collided.
func TestEIP8246Create2RecreatesBalanceOnly(t *testing.T) {
	var (
		key, _  = crypto.HexToECDSA("b71c71a67e1177ad4e901695e1b4b9ee17ae16c6668d313eac2f96dbcda3f291")
		sender  = crypto.PubkeyToAddress(key.PublicKey)
		factory = common.HexToAddress("0xfac8246")
		config  = *params.MergedTestChainConfig
		signer  = types.LatestSigner(&config)
		engine  = beacon.New(ethash.NewFaker())
		init    = []byte{0x30, 0xff} // ADDRESS; SELFDESTRUCT
	)
	config.AmsterdamTime = new(uint64)
	var word [32]byte
	copy(word[32-len(init):], init)
	factoryCode := append([]byte{0x7f}, word[:]...)
	factoryCode = append(factoryCode,
		0x5f, 0x52, // PUSH0; MSTORE
		0x60, 0x01, // salt
		0x60, byte(len(init)),
		0x60, byte(32-len(init)),
		0x34, // CALLVALUE
		0xf5, // CREATE2
		0x50, // POP
		0x00,
	)
	gspec := &Genesis{
		Config: &config,
		Alloc: types.GenesisAlloc{
			sender:  {Balance: newGwei(1_000_000_000)},
			factory: {Nonce: 1, Code: factoryCode, Balance: common.Big0},
		},
	}
	var salt [32]byte
	salt[31] = 1
	child := crypto.CreateAddress2(factory, salt, crypto.Keccak256(init))
	db, blocks, _ := GenerateChainWithGenesis(gspec, engine, 2, func(i int, b *BlockGen) {
		value := big.NewInt(5)
		if i == 1 {
			value.SetInt64(1)
		}
		b.AddTx(types.MustSignNewTx(key, signer, &types.DynamicFeeTx{
			ChainID:   gspec.Config.ChainID,
			Nonce:     uint64(i),
			To:        &factory,
			Gas:       1_000_000,
			GasFeeCap: newGwei(5),
			GasTipCap: newGwei(5),
			Value:     value,
		}))
	})
	chain, err := NewBlockChain(db, gspec, engine, nil)
	if err != nil {
		t.Fatalf("failed to create chain: %v", err)
	}
	defer chain.Stop()
	state, err := chain.StateAt(blocks[1].Header())
	if err != nil {
		t.Fatalf("failed to obtain block state: %v", err)
	}
	if got := state.GetBalance(child).Uint64(); got != 6 {
		t.Errorf("CREATE2 child balance = %d, want 6", got)
	}
	if got := state.GetNonce(child); got != 0 {
		t.Errorf("CREATE2 child nonce = %d, want 0", got)
	}
	if got := state.GetCodeSize(child); got != 0 {
		t.Errorf("CREATE2 child code size = %d, want 0", got)
	}
}
