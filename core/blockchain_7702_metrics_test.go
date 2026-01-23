// Copyright 2025 The go-ethereum Authors
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
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/params"
	"github.com/holiman/uint256"
)

// TestEIP7702DelegationMetricsSet verifies that Eip7702DelegationsSet is correctly
// incremented when a SetCodeTx sets a delegation (non-zero address).
func TestEIP7702DelegationMetricsSet(t *testing.T) {
	var (
		config  = *params.MergedTestChainConfig
		signer  = types.LatestSigner(&config)
		engine  = beacon.New(ethash.NewFaker())
		key1, _ = crypto.HexToECDSA("b71c71a67e1177ad4e901695e1b4b9ee17ae16c6668d313eac2f96dbcda3f291")
		addr1   = crypto.PubkeyToAddress(key1.PublicKey)
		target  = common.HexToAddress("0x000000000000000000000000000000000000aaaa")
		funds   = new(big.Int).Mul(common.Big1, big.NewInt(params.Ether))
	)

	gspec := &Genesis{
		Config: &config,
		Alloc: types.GenesisAlloc{
			addr1:  {Balance: funds},
			target: {Code: []byte{0x60, 0x00}, Balance: big.NewInt(0)}, // Simple PUSH1 0
		},
	}

	// Sign authorization to set delegation to target address
	// Note: Nonce is 1 because key1 is also the tx sender, and sender's nonce
	// is incremented (0->1) before authorization validation runs.
	auth, err := types.SignSetCode(key1, types.SetCodeAuthorization{
		ChainID: *uint256.MustFromBig(gspec.Config.ChainID),
		Address: target, // Non-zero address = SET delegation
		Nonce:   1,
	})
	if err != nil {
		t.Fatalf("failed to sign authorization: %v", err)
	}

	// Generate block with SetCodeTx
	_, blocks, _ := GenerateChainWithGenesis(gspec, engine, 1, func(i int, b *BlockGen) {
		txdata := &types.SetCodeTx{
			ChainID:   uint256.MustFromBig(gspec.Config.ChainID),
			Nonce:     0,
			To:        addr1,
			Gas:       100000,
			GasFeeCap: uint256.MustFromBig(newGwei(5)),
			GasTipCap: uint256.NewInt(2),
			AuthList:  []types.SetCodeAuthorization{auth},
		}
		tx := types.MustSignNewTx(key1, signer, txdata)
		b.AddTx(tx)
	})

	chain, err := NewBlockChain(rawdb.NewMemoryDatabase(), gspec, engine, nil)
	if err != nil {
		t.Fatalf("failed to create chain: %v", err)
	}
	defer chain.Stop()

	// Process block and get stats
	result, err := chain.ProcessBlock(chain.CurrentBlock().Root, blocks[0], true, false)
	if err != nil {
		t.Fatalf("failed to process block: %v", err)
	}

	// Verify SET delegation metric
	stats := result.Stats()
	if stats.Eip7702DelegationsSet != 1 {
		t.Errorf("Expected Eip7702DelegationsSet=1, got %d", stats.Eip7702DelegationsSet)
	}
	if stats.Eip7702DelegationsCleared != 0 {
		t.Errorf("Expected Eip7702DelegationsCleared=0, got %d", stats.Eip7702DelegationsCleared)
	}

	// Also verify CodeUpdated and CodeBytesWrite are incremented
	// Delegation code is 23 bytes (0xef0100 + 20-byte address)
	if stats.CodeUpdated != 1 {
		t.Errorf("Expected CodeUpdated=1, got %d", stats.CodeUpdated)
	}
	if stats.CodeBytesWrite != 23 {
		t.Errorf("Expected CodeBytesWrite=23 (delegation code size), got %d", stats.CodeBytesWrite)
	}
}

// TestEIP7702DelegationMetricsClear verifies that Eip7702DelegationsCleared is correctly
// incremented when a SetCodeTx clears a delegation (zero address).
func TestEIP7702DelegationMetricsClear(t *testing.T) {
	var (
		config  = *params.MergedTestChainConfig
		signer  = types.LatestSigner(&config)
		engine  = beacon.New(ethash.NewFaker())
		key1, _ = crypto.HexToECDSA("b71c71a67e1177ad4e901695e1b4b9ee17ae16c6668d313eac2f96dbcda3f291")
		addr1   = crypto.PubkeyToAddress(key1.PublicKey)
		target  = common.HexToAddress("0x000000000000000000000000000000000000aaaa")
		funds   = new(big.Int).Mul(common.Big1, big.NewInt(params.Ether))
	)

	// Start with addr1 already having a delegation set
	gspec := &Genesis{
		Config: &config,
		Alloc: types.GenesisAlloc{
			addr1: {
				Balance: funds,
				Code:    types.AddressToDelegation(target), // Pre-existing delegation
				Nonce:   1,                                 // Nonce 1 since delegation was "set"
			},
			target: {Code: []byte{0x60, 0x00}, Balance: big.NewInt(0)},
		},
	}

	// Sign authorization to CLEAR delegation (zero address)
	// Note: Auth nonce is 2 because addr1 starts with nonce 1, and tx sender's
	// nonce is incremented (1->2) before authorization validation runs.
	auth, err := types.SignSetCode(key1, types.SetCodeAuthorization{
		ChainID: *uint256.MustFromBig(gspec.Config.ChainID),
		Address: common.Address{}, // Zero address = CLEAR delegation
		Nonce:   2,                // Post-increment nonce (1->2)
	})
	if err != nil {
		t.Fatalf("failed to sign authorization: %v", err)
	}

	// Generate block with SetCodeTx that clears delegation
	_, blocks, _ := GenerateChainWithGenesis(gspec, engine, 1, func(i int, b *BlockGen) {
		txdata := &types.SetCodeTx{
			ChainID:   uint256.MustFromBig(gspec.Config.ChainID),
			Nonce:     1, // Account starts with nonce 1
			To:        addr1,
			Gas:       100000,
			GasFeeCap: uint256.MustFromBig(newGwei(5)),
			GasTipCap: uint256.NewInt(2),
			AuthList:  []types.SetCodeAuthorization{auth},
		}
		tx := types.MustSignNewTx(key1, signer, txdata)
		b.AddTx(tx)
	})

	chain, err := NewBlockChain(rawdb.NewMemoryDatabase(), gspec, engine, nil)
	if err != nil {
		t.Fatalf("failed to create chain: %v", err)
	}
	defer chain.Stop()

	// Process block and get stats
	result, err := chain.ProcessBlock(chain.CurrentBlock().Root, blocks[0], true, false)
	if err != nil {
		t.Fatalf("failed to process block: %v", err)
	}

	// Verify CLEAR delegation metric
	stats := result.Stats()
	if stats.Eip7702DelegationsCleared != 1 {
		t.Errorf("Expected Eip7702DelegationsCleared=1, got %d", stats.Eip7702DelegationsCleared)
	}
	if stats.Eip7702DelegationsSet != 0 {
		t.Errorf("Expected Eip7702DelegationsSet=0, got %d", stats.Eip7702DelegationsSet)
	}
}

// TestEIP7702DelegationMetricsMultiple verifies metrics when multiple authorizations
// are included in a single SetCodeTx.
func TestEIP7702DelegationMetricsMultiple(t *testing.T) {
	var (
		config  = *params.MergedTestChainConfig
		signer  = types.LatestSigner(&config)
		engine  = beacon.New(ethash.NewFaker())
		key1, _ = crypto.HexToECDSA("b71c71a67e1177ad4e901695e1b4b9ee17ae16c6668d313eac2f96dbcda3f291")
		key2, _ = crypto.HexToECDSA("8a1f9a8f95be41cd7ccb6168179afb4504aefe388d1e14474d32c45c72ce7b7a")
		key3, _ = crypto.HexToECDSA("49a7b37aa6f6645917e7b807e9d1c00d4fa71f18343b0d4122a4d2df64dd6fee")
		addr1   = crypto.PubkeyToAddress(key1.PublicKey)
		addr2   = crypto.PubkeyToAddress(key2.PublicKey)
		addr3   = crypto.PubkeyToAddress(key3.PublicKey)
		targetA = common.HexToAddress("0x000000000000000000000000000000000000aaaa")
		targetB = common.HexToAddress("0x000000000000000000000000000000000000bbbb")
		funds   = new(big.Int).Mul(common.Big1, big.NewInt(params.Ether))
	)

	// addr3 starts with a delegation that will be cleared
	gspec := &Genesis{
		Config: &config,
		Alloc: types.GenesisAlloc{
			addr1:   {Balance: funds},
			addr2:   {Balance: funds},
			addr3:   {Balance: funds, Code: types.AddressToDelegation(targetA), Nonce: 1},
			targetA: {Code: []byte{0x60, 0x00}, Balance: big.NewInt(0)},
			targetB: {Code: []byte{0x60, 0x01}, Balance: big.NewInt(0)},
		},
	}

	// Auth 1: addr1 sets delegation to targetA
	// Note: Nonce is 1 because key1 is also the tx sender (nonce 0->1)
	auth1, _ := types.SignSetCode(key1, types.SetCodeAuthorization{
		ChainID: *uint256.MustFromBig(gspec.Config.ChainID),
		Address: targetA,
		Nonce:   1,
	})

	// Auth 2: addr2 sets delegation to targetB
	auth2, _ := types.SignSetCode(key2, types.SetCodeAuthorization{
		ChainID: *uint256.MustFromBig(gspec.Config.ChainID),
		Address: targetB,
		Nonce:   0,
	})

	// Auth 3: addr3 clears delegation (zero address)
	auth3, _ := types.SignSetCode(key3, types.SetCodeAuthorization{
		ChainID: *uint256.MustFromBig(gspec.Config.ChainID),
		Address: common.Address{}, // Clear
		Nonce:   1,
	})

	// Generate block with all three authorizations
	_, blocks, _ := GenerateChainWithGenesis(gspec, engine, 1, func(i int, b *BlockGen) {
		txdata := &types.SetCodeTx{
			ChainID:   uint256.MustFromBig(gspec.Config.ChainID),
			Nonce:     0,
			To:        addr1,
			Gas:       500000,
			GasFeeCap: uint256.MustFromBig(newGwei(5)),
			GasTipCap: uint256.NewInt(2),
			AuthList:  []types.SetCodeAuthorization{auth1, auth2, auth3},
		}
		tx := types.MustSignNewTx(key1, signer, txdata)
		b.AddTx(tx)
	})

	chain, err := NewBlockChain(rawdb.NewMemoryDatabase(), gspec, engine, nil)
	if err != nil {
		t.Fatalf("failed to create chain: %v", err)
	}
	defer chain.Stop()

	// Process block and get stats
	result, err := chain.ProcessBlock(chain.CurrentBlock().Root, blocks[0], true, false)
	if err != nil {
		t.Fatalf("failed to process block: %v", err)
	}

	// Verify metrics: 2 delegations set, 1 cleared
	stats := result.Stats()
	if stats.Eip7702DelegationsSet != 2 {
		t.Errorf("Expected Eip7702DelegationsSet=2, got %d", stats.Eip7702DelegationsSet)
	}
	if stats.Eip7702DelegationsCleared != 1 {
		t.Errorf("Expected Eip7702DelegationsCleared=1, got %d", stats.Eip7702DelegationsCleared)
	}

	// CodeUpdated should be 2 (two delegations set, clear doesn't add code)
	if stats.CodeUpdated != 2 {
		t.Errorf("Expected CodeUpdated=2, got %d", stats.CodeUpdated)
	}

	// CodeBytesWrite should be 46 (23 bytes per delegation * 2)
	if stats.CodeBytesWrite != 46 {
		t.Errorf("Expected CodeBytesWrite=46, got %d", stats.CodeBytesWrite)
	}
}
