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
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/params"
	"github.com/holiman/uint256"
)

// TestAccountMetrics verifies that AccountLoaded and AccountUpdated are correctly
// tracked when processing a simple ETH transfer transaction.
func TestAccountMetrics(t *testing.T) {
	var (
		config   = *params.MergedTestChainConfig
		signer   = types.LatestSigner(&config)
		engine   = beacon.New(ethash.NewFaker())
		key, _   = crypto.HexToECDSA("b71c71a67e1177ad4e901695e1b4b9ee17ae16c6668d313eac2f96dbcda3f291")
		sender   = crypto.PubkeyToAddress(key.PublicKey)
		receiver = common.HexToAddress("0x1111111111111111111111111111111111111111")
		funds    = new(big.Int).Mul(common.Big1, big.NewInt(params.Ether))
	)

	gspec := &Genesis{
		Config: &config,
		Alloc: types.GenesisAlloc{
			sender:   {Balance: funds},
			receiver: {Balance: big.NewInt(1)}, // Pre-existing account
		},
	}

	// Generate block with simple ETH transfer
	_, blocks, _ := GenerateChainWithGenesis(gspec, engine, 1, func(i int, b *BlockGen) {
		tx, _ := types.SignTx(types.NewTransaction(
			0,                                  // nonce
			receiver,                           // to
			big.NewInt(1000),                   // value
			21000,                              // gas
			uint256.MustFromBig(newGwei(5)).ToBig(), // gasPrice
			nil,                                // data
		), signer, key)
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

	stats := result.Stats()

	// Both sender and receiver should be loaded from the database
	if stats.AccountLoaded < 2 {
		t.Errorf("Expected AccountLoaded >= 2, got %d", stats.AccountLoaded)
	}

	// Both sender (nonce+balance) and receiver (balance) should be updated
	if stats.AccountUpdated < 2 {
		t.Errorf("Expected AccountUpdated >= 2, got %d", stats.AccountUpdated)
	}
}

// TestStorageLoadedMetric verifies that StorageLoaded is correctly incremented
// when a contract reads from storage that was set in genesis.
func TestStorageLoadedMetric(t *testing.T) {
	var (
		config   = *params.MergedTestChainConfig
		signer   = types.LatestSigner(&config)
		engine   = beacon.New(ethash.NewFaker())
		key, _   = crypto.HexToECDSA("b71c71a67e1177ad4e901695e1b4b9ee17ae16c6668d313eac2f96dbcda3f291")
		sender   = crypto.PubkeyToAddress(key.PublicKey)
		contract = common.HexToAddress("0x2222222222222222222222222222222222222222")
		funds    = new(big.Int).Mul(common.Big1, big.NewInt(params.Ether))
	)

	// Contract that reads slot 0 via SLOAD and returns it
	// PUSH1 0x00 SLOAD POP STOP
	contractCode := []byte{
		byte(vm.PUSH1), 0x00,
		byte(vm.SLOAD),
		byte(vm.POP),
		byte(vm.STOP),
	}

	gspec := &Genesis{
		Config: &config,
		Alloc: types.GenesisAlloc{
			sender: {Balance: funds},
			contract: {
				Code:    contractCode,
				Balance: big.NewInt(0),
				Storage: map[common.Hash]common.Hash{
					common.HexToHash("0x00"): common.HexToHash("0x42"), // Pre-populated storage
				},
			},
		},
	}

	// Generate block that calls the contract
	_, blocks, _ := GenerateChainWithGenesis(gspec, engine, 1, func(i int, b *BlockGen) {
		tx, _ := types.SignTx(types.NewTransaction(
			0,
			contract,
			big.NewInt(0),
			50000,
			uint256.MustFromBig(newGwei(5)).ToBig(),
			nil,
		), signer, key)
		b.AddTx(tx)
	})

	chain, err := NewBlockChain(rawdb.NewMemoryDatabase(), gspec, engine, nil)
	if err != nil {
		t.Fatalf("failed to create chain: %v", err)
	}
	defer chain.Stop()

	result, err := chain.ProcessBlock(chain.CurrentBlock().Root, blocks[0], true, false)
	if err != nil {
		t.Fatalf("failed to process block: %v", err)
	}

	stats := result.Stats()

	// Storage slot 0 should be loaded from DB
	if stats.StorageLoaded < 1 {
		t.Errorf("Expected StorageLoaded >= 1, got %d", stats.StorageLoaded)
	}
}

// TestStorageUpdatedMetric verifies that StorageUpdated is correctly incremented
// when a contract writes to storage.
func TestStorageUpdatedMetric(t *testing.T) {
	var (
		config   = *params.MergedTestChainConfig
		signer   = types.LatestSigner(&config)
		engine   = beacon.New(ethash.NewFaker())
		key, _   = crypto.HexToECDSA("b71c71a67e1177ad4e901695e1b4b9ee17ae16c6668d313eac2f96dbcda3f291")
		sender   = crypto.PubkeyToAddress(key.PublicKey)
		contract = common.HexToAddress("0x3333333333333333333333333333333333333333")
		funds    = new(big.Int).Mul(common.Big1, big.NewInt(params.Ether))
	)

	// Contract that writes 0x42 to slot 0x01
	// PUSH1 0x42 PUSH1 0x01 SSTORE STOP
	contractCode := []byte{
		byte(vm.PUSH1), 0x42, // value
		byte(vm.PUSH1), 0x01, // key
		byte(vm.SSTORE),
		byte(vm.STOP),
	}

	gspec := &Genesis{
		Config: &config,
		Alloc: types.GenesisAlloc{
			sender:   {Balance: funds},
			contract: {Code: contractCode, Balance: big.NewInt(0)},
		},
	}

	_, blocks, _ := GenerateChainWithGenesis(gspec, engine, 1, func(i int, b *BlockGen) {
		tx, _ := types.SignTx(types.NewTransaction(
			0,
			contract,
			big.NewInt(0),
			50000,
			uint256.MustFromBig(newGwei(5)).ToBig(),
			nil,
		), signer, key)
		b.AddTx(tx)
	})

	chain, err := NewBlockChain(rawdb.NewMemoryDatabase(), gspec, engine, nil)
	if err != nil {
		t.Fatalf("failed to create chain: %v", err)
	}
	defer chain.Stop()

	result, err := chain.ProcessBlock(chain.CurrentBlock().Root, blocks[0], true, false)
	if err != nil {
		t.Fatalf("failed to process block: %v", err)
	}

	stats := result.Stats()

	// Storage slot 1 should be updated
	if stats.StorageUpdated < 1 {
		t.Errorf("Expected StorageUpdated >= 1, got %d", stats.StorageUpdated)
	}
}

// TestStorageDeletedMetric verifies that StorageDeleted is correctly incremented
// when a contract clears a storage slot (sets to zero).
func TestStorageDeletedMetric(t *testing.T) {
	var (
		config   = *params.MergedTestChainConfig
		signer   = types.LatestSigner(&config)
		engine   = beacon.New(ethash.NewFaker())
		key, _   = crypto.HexToECDSA("b71c71a67e1177ad4e901695e1b4b9ee17ae16c6668d313eac2f96dbcda3f291")
		sender   = crypto.PubkeyToAddress(key.PublicKey)
		contract = common.HexToAddress("0x4444444444444444444444444444444444444444")
		funds    = new(big.Int).Mul(common.Big1, big.NewInt(params.Ether))
	)

	// Contract that clears slot 0x01 (sets to zero)
	// PUSH1 0x00 PUSH1 0x01 SSTORE STOP
	contractCode := []byte{
		byte(vm.PUSH1), 0x00, // value (zero = delete)
		byte(vm.PUSH1), 0x01, // key
		byte(vm.SSTORE),
		byte(vm.STOP),
	}

	gspec := &Genesis{
		Config: &config,
		Alloc: types.GenesisAlloc{
			sender: {Balance: funds},
			contract: {
				Code:    contractCode,
				Balance: big.NewInt(0),
				Storage: map[common.Hash]common.Hash{
					common.HexToHash("0x01"): common.HexToHash("0x42"), // Pre-existing non-zero value
				},
			},
		},
	}

	_, blocks, _ := GenerateChainWithGenesis(gspec, engine, 1, func(i int, b *BlockGen) {
		tx, _ := types.SignTx(types.NewTransaction(
			0,
			contract,
			big.NewInt(0),
			50000,
			uint256.MustFromBig(newGwei(5)).ToBig(),
			nil,
		), signer, key)
		b.AddTx(tx)
	})

	chain, err := NewBlockChain(rawdb.NewMemoryDatabase(), gspec, engine, nil)
	if err != nil {
		t.Fatalf("failed to create chain: %v", err)
	}
	defer chain.Stop()

	result, err := chain.ProcessBlock(chain.CurrentBlock().Root, blocks[0], true, false)
	if err != nil {
		t.Fatalf("failed to process block: %v", err)
	}

	stats := result.Stats()

	// Storage slot 1 should be deleted (cleared)
	if stats.StorageDeleted < 1 {
		t.Errorf("Expected StorageDeleted >= 1, got %d", stats.StorageDeleted)
	}
}

// TestCodeLoadedMetric verifies that CodeLoaded is correctly incremented
// when a contract's code is fetched to execute it.
func TestCodeLoadedMetric(t *testing.T) {
	var (
		config   = *params.MergedTestChainConfig
		signer   = types.LatestSigner(&config)
		engine   = beacon.New(ethash.NewFaker())
		key, _   = crypto.HexToECDSA("b71c71a67e1177ad4e901695e1b4b9ee17ae16c6668d313eac2f96dbcda3f291")
		sender   = crypto.PubkeyToAddress(key.PublicKey)
		contract = common.HexToAddress("0x5555555555555555555555555555555555555555")
		funds    = new(big.Int).Mul(common.Big1, big.NewInt(params.Ether))
	)

	// Simple contract: PUSH1 0x42 POP STOP
	contractCode := []byte{
		byte(vm.PUSH1), 0x42,
		byte(vm.POP),
		byte(vm.STOP),
	}

	gspec := &Genesis{
		Config: &config,
		Alloc: types.GenesisAlloc{
			sender:   {Balance: funds},
			contract: {Code: contractCode, Balance: big.NewInt(0)},
		},
	}

	_, blocks, _ := GenerateChainWithGenesis(gspec, engine, 1, func(i int, b *BlockGen) {
		tx, _ := types.SignTx(types.NewTransaction(
			0,
			contract,
			big.NewInt(0),
			50000,
			uint256.MustFromBig(newGwei(5)).ToBig(),
			nil,
		), signer, key)
		b.AddTx(tx)
	})

	chain, err := NewBlockChain(rawdb.NewMemoryDatabase(), gspec, engine, nil)
	if err != nil {
		t.Fatalf("failed to create chain: %v", err)
	}
	defer chain.Stop()

	result, err := chain.ProcessBlock(chain.CurrentBlock().Root, blocks[0], true, false)
	if err != nil {
		t.Fatalf("failed to process block: %v", err)
	}

	stats := result.Stats()

	// Contract code should be loaded to execute
	if stats.CodeLoaded < 1 {
		t.Errorf("Expected CodeLoaded >= 1, got %d", stats.CodeLoaded)
	}
}

// TestCodeUpdatedMetricCREATE verifies that CodeUpdated and CodeBytesWrite are correctly
// incremented when a contract is deployed via a creation transaction.
func TestCodeUpdatedMetricCREATE(t *testing.T) {
	var (
		config = *params.MergedTestChainConfig
		signer = types.LatestSigner(&config)
		engine = beacon.New(ethash.NewFaker())
		key, _ = crypto.HexToECDSA("b71c71a67e1177ad4e901695e1b4b9ee17ae16c6668d313eac2f96dbcda3f291")
		sender = crypto.PubkeyToAddress(key.PublicKey)
		funds  = new(big.Int).Mul(common.Big1, big.NewInt(params.Ether))
	)

	// Runtime code: PUSH1 0x42 POP STOP (4 bytes)
	runtimeCode := []byte{
		byte(vm.PUSH1), 0x42,
		byte(vm.POP),
		byte(vm.STOP),
	}
	runtimeLen := len(runtimeCode)

	// Initcode that returns the runtime code
	// PUSH4 <runtime> PUSH1 0x00 MSTORE PUSH1 <len> PUSH1 0x1c RETURN
	initCode := []byte{
		byte(vm.PUSH4),
		runtimeCode[0], runtimeCode[1], runtimeCode[2], runtimeCode[3],
		byte(vm.PUSH1), 0x00,
		byte(vm.MSTORE),
		byte(vm.PUSH1), byte(runtimeLen), // size
		byte(vm.PUSH1), 0x1c,             // offset (32 - 4 = 28 = 0x1c)
		byte(vm.RETURN),
	}

	gspec := &Genesis{
		Config: &config,
		Alloc: types.GenesisAlloc{
			sender: {Balance: funds},
		},
	}

	// Contract creation transaction (to = nil)
	_, blocks, _ := GenerateChainWithGenesis(gspec, engine, 1, func(i int, b *BlockGen) {
		tx, _ := types.SignTx(types.NewContractCreation(
			0,
			big.NewInt(0),
			100000,
			uint256.MustFromBig(newGwei(5)).ToBig(),
			initCode,
		), signer, key)
		b.AddTx(tx)
	})

	chain, err := NewBlockChain(rawdb.NewMemoryDatabase(), gspec, engine, nil)
	if err != nil {
		t.Fatalf("failed to create chain: %v", err)
	}
	defer chain.Stop()

	result, err := chain.ProcessBlock(chain.CurrentBlock().Root, blocks[0], true, false)
	if err != nil {
		t.Fatalf("failed to process block: %v", err)
	}

	stats := result.Stats()

	// One contract should be deployed
	if stats.CodeUpdated != 1 {
		t.Errorf("Expected CodeUpdated = 1, got %d", stats.CodeUpdated)
	}

	// Runtime code is 4 bytes
	if stats.CodeBytesWrite != runtimeLen {
		t.Errorf("Expected CodeBytesWrite = %d, got %d", runtimeLen, stats.CodeBytesWrite)
	}
}

// TestAccountDeletedMetric verifies that AccountDeleted is correctly incremented
// when a contract self-destructs. Post-Cancun (EIP-6780), this only works if the
// contract is created and destroyed in the same transaction.
func TestAccountDeletedMetric(t *testing.T) {
	var (
		config      = *params.MergedTestChainConfig
		signer      = types.LatestSigner(&config)
		engine      = beacon.New(ethash.NewFaker())
		key, _      = crypto.HexToECDSA("b71c71a67e1177ad4e901695e1b4b9ee17ae16c6668d313eac2f96dbcda3f291")
		sender      = crypto.PubkeyToAddress(key.PublicKey)
		beneficiary = common.HexToAddress("0x6666666666666666666666666666666666666666")
		funds       = new(big.Int).Mul(common.Big1, big.NewInt(params.Ether))
	)

	// Initcode that:
	// 1. Deploys runtime code that will never execute (we self-destruct in initcode)
	// 2. Immediately calls SELFDESTRUCT to beneficiary during contract creation
	// This is EIP-6780 compatible because we self-destruct in the same tx as creation.
	//
	// PUSH20 <beneficiary> SELFDESTRUCT
	initCode := make([]byte, 0, 22)
	initCode = append(initCode, byte(vm.PUSH20))
	initCode = append(initCode, beneficiary.Bytes()...)
	initCode = append(initCode, byte(vm.SELFDESTRUCT))

	gspec := &Genesis{
		Config: &config,
		Alloc: types.GenesisAlloc{
			sender:      {Balance: funds},
			beneficiary: {Balance: big.NewInt(1)}, // Pre-existing beneficiary
		},
	}

	// Contract creation that immediately self-destructs
	_, blocks, _ := GenerateChainWithGenesis(gspec, engine, 1, func(i int, b *BlockGen) {
		tx, _ := types.SignTx(types.NewContractCreation(
			0,
			big.NewInt(1000), // Send some ETH to the contract
			100000,
			uint256.MustFromBig(newGwei(5)).ToBig(),
			initCode,
		), signer, key)
		b.AddTx(tx)
	})

	chain, err := NewBlockChain(rawdb.NewMemoryDatabase(), gspec, engine, nil)
	if err != nil {
		t.Fatalf("failed to create chain: %v", err)
	}
	defer chain.Stop()

	result, err := chain.ProcessBlock(chain.CurrentBlock().Root, blocks[0], true, false)
	if err != nil {
		t.Fatalf("failed to process block: %v", err)
	}

	stats := result.Stats()

	// The newly created contract should be deleted (self-destructed)
	if stats.AccountDeleted < 1 {
		t.Errorf("Expected AccountDeleted >= 1, got %d", stats.AccountDeleted)
	}
}

// TestMultipleStorageOperationsMetrics verifies that storage metrics correctly
// accumulate when multiple operations occur.
func TestMultipleStorageOperationsMetrics(t *testing.T) {
	var (
		config   = *params.MergedTestChainConfig
		signer   = types.LatestSigner(&config)
		engine   = beacon.New(ethash.NewFaker())
		key, _   = crypto.HexToECDSA("b71c71a67e1177ad4e901695e1b4b9ee17ae16c6668d313eac2f96dbcda3f291")
		sender   = crypto.PubkeyToAddress(key.PublicKey)
		contract = common.HexToAddress("0x7777777777777777777777777777777777777777")
		funds    = new(big.Int).Mul(common.Big1, big.NewInt(params.Ether))
	)

	// Contract that:
	// 1. Reads slots 0, 1, 2 (3 SLOADs)
	// 2. Writes to slots 3, 4 (2 SSTOREs with non-zero values)
	// 3. Clears slot 5 (1 SSTORE with zero, existing value was non-zero)
	contractCode := []byte{
		// SLOAD slot 0
		byte(vm.PUSH1), 0x00, byte(vm.SLOAD), byte(vm.POP),
		// SLOAD slot 1
		byte(vm.PUSH1), 0x01, byte(vm.SLOAD), byte(vm.POP),
		// SLOAD slot 2
		byte(vm.PUSH1), 0x02, byte(vm.SLOAD), byte(vm.POP),
		// SSTORE slot 3 = 0xFF
		byte(vm.PUSH1), 0xFF, byte(vm.PUSH1), 0x03, byte(vm.SSTORE),
		// SSTORE slot 4 = 0xEE
		byte(vm.PUSH1), 0xEE, byte(vm.PUSH1), 0x04, byte(vm.SSTORE),
		// SSTORE slot 5 = 0x00 (delete)
		byte(vm.PUSH1), 0x00, byte(vm.PUSH1), 0x05, byte(vm.SSTORE),
		byte(vm.STOP),
	}

	gspec := &Genesis{
		Config: &config,
		Alloc: types.GenesisAlloc{
			sender: {Balance: funds},
			contract: {
				Code:    contractCode,
				Balance: big.NewInt(0),
				Storage: map[common.Hash]common.Hash{
					common.HexToHash("0x00"): common.HexToHash("0x01"),
					common.HexToHash("0x01"): common.HexToHash("0x02"),
					common.HexToHash("0x02"): common.HexToHash("0x03"),
					common.HexToHash("0x05"): common.HexToHash("0xAA"), // Will be cleared
				},
			},
		},
	}

	_, blocks, _ := GenerateChainWithGenesis(gspec, engine, 1, func(i int, b *BlockGen) {
		tx, _ := types.SignTx(types.NewTransaction(
			0,
			contract,
			big.NewInt(0),
			100000,
			uint256.MustFromBig(newGwei(5)).ToBig(),
			nil,
		), signer, key)
		b.AddTx(tx)
	})

	chain, err := NewBlockChain(rawdb.NewMemoryDatabase(), gspec, engine, nil)
	if err != nil {
		t.Fatalf("failed to create chain: %v", err)
	}
	defer chain.Stop()

	result, err := chain.ProcessBlock(chain.CurrentBlock().Root, blocks[0], true, false)
	if err != nil {
		t.Fatalf("failed to process block: %v", err)
	}

	stats := result.Stats()

	// 3 slots read: 0, 1, 2
	if stats.StorageLoaded < 3 {
		t.Errorf("Expected StorageLoaded >= 3, got %d", stats.StorageLoaded)
	}

	// 2 slots written with non-zero values: 3, 4
	if stats.StorageUpdated < 2 {
		t.Errorf("Expected StorageUpdated >= 2, got %d", stats.StorageUpdated)
	}

	// 1 slot cleared: 5
	if stats.StorageDeleted < 1 {
		t.Errorf("Expected StorageDeleted >= 1, got %d", stats.StorageDeleted)
	}
}
