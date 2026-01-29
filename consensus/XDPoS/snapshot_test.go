// Copyright 2023 The XDC Network Authors
// This file is part of the XDC Network library.

package XDPoS

import (
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
)

func TestSnapshotCreation(t *testing.T) {
	// Create test masternodes
	masternodes := make([]common.Address, 5)
	for i := 0; i < 5; i++ {
		masternodes[i] = common.BigToAddress(big.NewInt(int64(i + 1)))
	}
	
	// Create snapshot
	snap := NewSnapshot(100, common.Hash{}, masternodes)
	
	if snap.Number != 100 {
		t.Errorf("Expected number 100, got %d", snap.Number)
	}
	
	if len(snap.Masternodes) != 5 {
		t.Errorf("Expected 5 masternodes, got %d", len(snap.Masternodes))
	}
}

func TestSnapshotInturn(t *testing.T) {
	masternodes := make([]common.Address, 5)
	for i := 0; i < 5; i++ {
		masternodes[i] = common.BigToAddress(big.NewInt(int64(i + 1)))
	}
	
	snap := NewSnapshot(100, common.Hash{}, masternodes)
	
	tests := []struct {
		blockNumber uint64
		signer      common.Address
		expected    bool
	}{
		{100, masternodes[0], true},
		{101, masternodes[1], true},
		{102, masternodes[2], true},
		{100, masternodes[1], false},
	}
	
	for _, tt := range tests {
		result := snap.Inturn(tt.blockNumber, tt.signer)
		if result != tt.expected {
			t.Errorf("Block %d, signer %s: expected %v, got %v",
				tt.blockNumber, tt.signer.Hex(), tt.expected, result)
		}
	}
}

func TestSnapshotCopy(t *testing.T) {
	masternodes := make([]common.Address, 3)
	for i := 0; i < 3; i++ {
		masternodes[i] = common.BigToAddress(big.NewInt(int64(i + 1)))
	}
	
	snap := NewSnapshot(100, common.Hash{}, masternodes)
	copy := snap.Copy()
	
	// Verify copy is independent
	copy.Number = 200
	if snap.Number == copy.Number {
		t.Error("Copy should be independent of original")
	}
}

func TestSnapshotApply(t *testing.T) {
	masternodes := make([]common.Address, 3)
	for i := 0; i < 3; i++ {
		masternodes[i] = common.BigToAddress(big.NewInt(int64(i + 1)))
	}
	
	snap := NewSnapshot(100, common.Hash{}, masternodes)
	
	// Create a header
	header := &types.Header{
		Number:   big.NewInt(101),
		Coinbase: masternodes[1],
	}
	
	// Apply header
	newSnap, err := snap.Apply([]*types.Header{header})
	if err != nil {
		t.Fatalf("Apply failed: %v", err)
	}
	
	if newSnap.Number != 101 {
		t.Errorf("Expected number 101, got %d", newSnap.Number)
	}
}

func TestSnapshotSerialization(t *testing.T) {
	masternodes := make([]common.Address, 3)
	for i := 0; i < 3; i++ {
		masternodes[i] = common.BigToAddress(big.NewInt(int64(i + 1)))
	}
	
	snap := NewSnapshot(100, common.Hash{1, 2, 3}, masternodes)
	
	// Test JSON marshaling
	data, err := snap.MarshalJSON()
	if err != nil {
		t.Fatalf("MarshalJSON failed: %v", err)
	}
	
	// Test unmarshaling
	snap2 := &Snapshot{}
	if err := snap2.UnmarshalJSON(data); err != nil {
		t.Fatalf("UnmarshalJSON failed: %v", err)
	}
	
	if snap2.Number != snap.Number {
		t.Errorf("Expected number %d, got %d", snap.Number, snap2.Number)
	}
}

func TestSnapshotRecents(t *testing.T) {
	masternodes := make([]common.Address, 5)
	for i := 0; i < 5; i++ {
		masternodes[i] = common.BigToAddress(big.NewInt(int64(i + 1)))
	}
	
	snap := NewSnapshot(100, common.Hash{}, masternodes)
	
	// Add recents
	snap.Recents[100] = masternodes[0]
	snap.Recents[101] = masternodes[1]
	
	// Check recent
	if snap.Recents[100] != masternodes[0] {
		t.Error("Recent signer mismatch")
	}
}

func TestSnapshotIsMasternode(t *testing.T) {
	masternodes := make([]common.Address, 3)
	for i := 0; i < 3; i++ {
		masternodes[i] = common.BigToAddress(big.NewInt(int64(i + 1)))
	}
	
	snap := NewSnapshot(100, common.Hash{}, masternodes)
	
	// Test masternode check
	if !snap.IsMasternode(masternodes[0]) {
		t.Error("Should be masternode")
	}
	
	nonMasternode := common.BigToAddress(big.NewInt(100))
	if snap.IsMasternode(nonMasternode) {
		t.Error("Should not be masternode")
	}
}

func TestCalculateM1(t *testing.T) {
	masternodes := make([]common.Address, 10)
	for i := 0; i < 10; i++ {
		masternodes[i] = common.BigToAddress(big.NewInt(int64(i + 1)))
	}
	
	// Test M1 calculation for different rounds
	tests := []struct {
		round    uint64
		expected int
	}{
		{0, 0},
		{1, 1},
		{9, 9},
		{10, 0}, // Wraps around
	}
	
	for _, tt := range tests {
		idx := int(tt.round) % len(masternodes)
		if idx != tt.expected {
			t.Errorf("Round %d: expected index %d, got %d", tt.round, tt.expected, idx)
		}
	}
}

// Benchmark tests

func BenchmarkSnapshotCopy(b *testing.B) {
	masternodes := make([]common.Address, 150)
	for i := 0; i < 150; i++ {
		masternodes[i] = common.BigToAddress(big.NewInt(int64(i + 1)))
	}
	
	snap := NewSnapshot(100, common.Hash{}, masternodes)
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = snap.Copy()
	}
}

func BenchmarkSnapshotInturn(b *testing.B) {
	masternodes := make([]common.Address, 150)
	for i := 0; i < 150; i++ {
		masternodes[i] = common.BigToAddress(big.NewInt(int64(i + 1)))
	}
	
	snap := NewSnapshot(100, common.Hash{}, masternodes)
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = snap.Inturn(uint64(i), masternodes[i%150])
	}
}

func BenchmarkSnapshotIsMasternode(b *testing.B) {
	masternodes := make([]common.Address, 150)
	for i := 0; i < 150; i++ {
		masternodes[i] = common.BigToAddress(big.NewInt(int64(i + 1)))
	}
	
	snap := NewSnapshot(100, common.Hash{}, masternodes)
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = snap.IsMasternode(masternodes[i%150])
	}
}
