package state

import (
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethdb/memorydb"
)

func TestBlockToBaseStateRootMapping(t *testing.T) {
	// Create a new mapping
	mapping := NewBlockToBaseStateRootMapping()

	// Test data
	blockHash1 := common.HexToHash("0x1111111111111111111111111111111111111111111111111111111111111111")
	stateRoot1 := common.HexToHash("0xaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa")
	blockHash2 := common.HexToHash("0x2222222222222222222222222222222222222222222222222222222222222222")
	stateRoot2 := common.HexToHash("0xbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb")

	// Test Add and Get
	mapping.Add(blockHash1, stateRoot1)
	mapping.Add(blockHash2, stateRoot2)

	root, exists := mapping.Get(blockHash1)
	if !exists {
		t.Fatalf("Expected mapping to exist for blockHash1")
	}
	if root != stateRoot1 {
		t.Fatalf("Expected stateRoot1, got %x", root)
	}

	root, exists = mapping.Get(blockHash2)
	if !exists {
		t.Fatalf("Expected mapping to exist for blockHash2")
	}
	if root != stateRoot2 {
		t.Fatalf("Expected stateRoot2, got %x", root)
	}

	// Test Has
	if !mapping.Has(blockHash1) {
		t.Fatalf("Expected Has to return true for blockHash1")
	}
	if mapping.Has(common.HexToHash("0x3333")) {
		t.Fatalf("Expected Has to return false for unknown hash")
	}

	// Test Store and Load
	db := memorydb.New()
	err := mapping.Store(db)
	if err != nil {
		t.Fatalf("Failed to store mapping: %v", err)
	}

	loadedMapping, err := LoadBlockToBaseStateRoot(db)
	if err != nil {
		t.Fatalf("Failed to load mapping: %v", err)
	}

	root, exists = loadedMapping.Get(blockHash1)
	if !exists {
		t.Fatalf("Expected loaded mapping to contain blockHash1")
	}
	if root != stateRoot1 {
		t.Fatalf("Expected stateRoot1 after loading, got %x", root)
	}
}

func TestStateDBVerkleTransition(t *testing.T) {
	// Create a test state
	db := NewDatabase(memorydb.New())
	baseState, _ := New(common.Hash{}, db, nil)

	// Create base state with account1
	addr1 := common.HexToAddress("0x1111111111111111111111111111111111111111")
	key1 := common.HexToHash("0xaaaa")
	value1 := common.HexToHash("0xbbbb")
	baseState.SetState(addr1, key1, value1)
	baseRoot, _ := baseState.Commit(false)

	// Create new verkle state with account2
	statedb, _ := New(common.Hash{}, db, nil)
	statedb.SetVerkleTransitionData(common.HexToHash("0xblock"), baseRoot)

	addr2 := common.HexToAddress("0x2222222222222222222222222222222222222222")
	key2 := common.HexToHash("0xcccc")
	value2 := common.HexToHash("0xdddd")
	statedb.SetState(addr2, key2, value2)

	// Test fallback to base state
	retrievedValue1 := statedb.GetState(addr1, key1)
	if retrievedValue1 != value1 {
		t.Fatalf("Expected fallback to base state to return correct value, got %x", retrievedValue1)
	}

	// Test current state access
	retrievedValue2 := statedb.GetState(addr2, key2)
	if retrievedValue2 != value2 {
		t.Fatalf("Expected current state access to return correct value, got %x", retrievedValue2)
	}

	// Check state properties
	if !statedb.IsVerkleTransitionActive() {
		t.Fatalf("Expected IsVerkleTransitionActive to return true")
	}

	if statedb.BaseStateRoot() != baseRoot {
		t.Fatalf("Expected BaseStateRoot to return correct root")
	}
}