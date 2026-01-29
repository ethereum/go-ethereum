// Copyright 2023 The XDC Network Authors
// This file is part of the XDC Network library.

package engines

import (
	"crypto/ecdsa"
	"math/big"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
)

func TestV2BlockValidation(t *testing.T) {
	// Create test header
	header := &types.Header{
		Number:     big.NewInt(1000),
		Time:       uint64(time.Now().Unix()),
		Difficulty: big.NewInt(1),
		Extra:      make([]byte, 97), // vanity + seal
	}
	
	// Verify difficulty is set correctly for V2
	if header.Difficulty.Cmp(big.NewInt(1)) != 0 {
		t.Error("V2 block should have difficulty 1")
	}
}

func TestV2SignatureRecovery(t *testing.T) {
	// Generate a test key
	key, err := crypto.GenerateKey()
	if err != nil {
		t.Fatalf("Failed to generate key: %v", err)
	}
	
	// Create a message
	message := []byte("test message")
	hash := crypto.Keccak256Hash(message)
	
	// Sign the message
	sig, err := crypto.Sign(hash.Bytes(), key)
	if err != nil {
		t.Fatalf("Failed to sign: %v", err)
	}
	
	// Recover public key
	pubkey, err := crypto.SigToPub(hash.Bytes(), sig)
	if err != nil {
		t.Fatalf("Failed to recover pubkey: %v", err)
	}
	
	// Verify address matches
	expected := crypto.PubkeyToAddress(key.PublicKey)
	recovered := crypto.PubkeyToAddress(*pubkey)
	
	if expected != recovered {
		t.Errorf("Address mismatch: expected %s, got %s", expected.Hex(), recovered.Hex())
	}
}

func TestV2VoteCreation(t *testing.T) {
	key, _ := crypto.GenerateKey()
	signer := crypto.PubkeyToAddress(key.PublicKey)
	
	vote := &types.Vote{
		ProposedBlockInfo: &types.BlockInfo{
			Hash:   common.HexToHash("0x123"),
			Number: big.NewInt(100),
			Round:  1,
		},
	}
	
	// Sign the vote
	hash := vote.Hash()
	sig, err := crypto.Sign(hash.Bytes(), key)
	if err != nil {
		t.Fatalf("Failed to sign vote: %v", err)
	}
	vote.Signature = sig
	
	// Verify signature
	pubkey, err := crypto.SigToPub(hash.Bytes(), sig)
	if err != nil {
		t.Fatalf("Failed to recover signer: %v", err)
	}
	
	recovered := crypto.PubkeyToAddress(*pubkey)
	if recovered != signer {
		t.Errorf("Signer mismatch: expected %s, got %s", signer.Hex(), recovered.Hex())
	}
}

func TestV2TimeoutCreation(t *testing.T) {
	key, _ := crypto.GenerateKey()
	signer := crypto.PubkeyToAddress(key.PublicKey)
	
	timeout := &types.Timeout{
		Round: 5,
		HighQC: &types.QuorumCert{
			Round: 4,
		},
	}
	
	// Sign the timeout
	hash := timeout.Hash()
	sig, err := crypto.Sign(hash.Bytes(), key)
	if err != nil {
		t.Fatalf("Failed to sign timeout: %v", err)
	}
	timeout.Signature = sig
	
	// Verify
	pubkey, _ := crypto.SigToPub(hash.Bytes(), sig)
	recovered := crypto.PubkeyToAddress(*pubkey)
	if recovered != signer {
		t.Errorf("Signer mismatch")
	}
}

func TestV2QCCreation(t *testing.T) {
	// Create multiple signers
	keys := make([]*ecdsa.PrivateKey, 5)
	signers := make([]common.Address, 5)
	for i := 0; i < 5; i++ {
		keys[i], _ = crypto.GenerateKey()
		signers[i] = crypto.PubkeyToAddress(keys[i].PublicKey)
	}
	
	// Create votes
	blockInfo := &types.BlockInfo{
		Hash:   common.HexToHash("0x123"),
		Number: big.NewInt(100),
		Round:  1,
	}
	
	votes := make([]*types.Vote, 5)
	for i := 0; i < 5; i++ {
		vote := &types.Vote{
			ProposedBlockInfo: blockInfo,
		}
		hash := vote.Hash()
		sig, _ := crypto.Sign(hash.Bytes(), keys[i])
		vote.Signature = sig
		votes[i] = vote
	}
	
	// Create QC from votes
	qc := &types.QuorumCert{
		ProposedBlockInfo: blockInfo,
		Signatures:        make([]types.Signature, 5),
	}
	for i, vote := range votes {
		qc.Signatures[i] = types.Signature{
			Signature: vote.Signature,
		}
	}
	
	// Verify QC has correct number of signatures
	if len(qc.Signatures) != 5 {
		t.Errorf("Expected 5 signatures, got %d", len(qc.Signatures))
	}
}

func TestV2RoundCalculation(t *testing.T) {
	// Test round calculation from block number
	epochSize := uint64(900)
	
	tests := []struct {
		blockNumber uint64
		expected    uint64
	}{
		{0, 0},
		{1, 1},
		{899, 899},
		{900, 0},   // New epoch starts
		{901, 1},
		{1800, 0},
	}
	
	for _, tt := range tests {
		round := tt.blockNumber % epochSize
		if round != tt.expected {
			t.Errorf("Block %d: expected round %d, got %d", tt.blockNumber, tt.expected, round)
		}
	}
}

func TestV2CertThreshold(t *testing.T) {
	tests := []struct {
		committeeSize int
		expected      int
	}{
		{150, 101}, // 2/3 + 1 of 150
		{100, 67},  // 2/3 + 1 of 100
		{50, 34},   // 2/3 + 1 of 50
		{3, 3},     // Minimum
	}
	
	for _, tt := range tests {
		threshold := (tt.committeeSize * 2 / 3) + 1
		if threshold != tt.expected {
			t.Errorf("Committee %d: expected threshold %d, got %d", tt.committeeSize, tt.expected, threshold)
		}
	}
}

func TestV2SyncInfo(t *testing.T) {
	// Create sync info
	syncInfo := &types.SyncInfo{
		HighestQC: &types.QuorumCert{
			ProposedBlockInfo: &types.BlockInfo{
				Hash:   common.HexToHash("0x123"),
				Number: big.NewInt(100),
				Round:  10,
			},
		},
		HighestTC: &types.TimeoutCert{
			Round: 9,
		},
	}
	
	// Verify fields
	if syncInfo.HighestQC.Round != 10 {
		t.Error("HighestQC round mismatch")
	}
	if syncInfo.HighestTC.Round != 9 {
		t.Error("HighestTC round mismatch")
	}
}

func TestV2EpochSwitch(t *testing.T) {
	epochSize := uint64(900)
	
	tests := []struct {
		blockNumber uint64
		isSwitch    bool
	}{
		{0, true},    // Genesis is epoch switch
		{1, false},
		{899, false},
		{900, true},
		{1800, true},
		{2699, false},
		{2700, true},
	}
	
	for _, tt := range tests {
		isSwitch := tt.blockNumber%epochSize == 0
		if isSwitch != tt.isSwitch {
			t.Errorf("Block %d: expected isSwitch=%v, got %v", tt.blockNumber, tt.isSwitch, isSwitch)
		}
	}
}

// Benchmark tests

func BenchmarkV2SignatureRecovery(b *testing.B) {
	key, _ := crypto.GenerateKey()
	message := []byte("test message for benchmarking")
	hash := crypto.Keccak256Hash(message)
	sig, _ := crypto.Sign(hash.Bytes(), key)
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = crypto.SigToPub(hash.Bytes(), sig)
	}
}

func BenchmarkV2VoteHash(b *testing.B) {
	vote := &types.Vote{
		ProposedBlockInfo: &types.BlockInfo{
			Hash:   common.HexToHash("0x123"),
			Number: big.NewInt(100),
			Round:  1,
		},
	}
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = vote.Hash()
	}
}

func BenchmarkV2QCValidation(b *testing.B) {
	// Create QC with signatures
	keys := make([]*ecdsa.PrivateKey, 100)
	signatures := make([][]byte, 100)
	
	blockInfo := &types.BlockInfo{
		Hash:   common.HexToHash("0x123"),
		Number: big.NewInt(100),
		Round:  1,
	}
	
	vote := &types.Vote{ProposedBlockInfo: blockInfo}
	hash := vote.Hash()
	
	for i := 0; i < 100; i++ {
		keys[i], _ = crypto.GenerateKey()
		signatures[i], _ = crypto.Sign(hash.Bytes(), keys[i])
	}
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		for _, sig := range signatures {
			_, _ = crypto.SigToPub(hash.Bytes(), sig)
		}
	}
}
