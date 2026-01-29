// Copyright 2023 The XDC Network Authors
// This file is part of the XDC Network library.
//
// The XDC Network library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.

package types

import (
	"math/big"
	"sync/atomic"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/rlp"
)

// BlockInfo contains information about a proposed block
type BlockInfo struct {
	Hash   common.Hash `json:"hash"`
	Number *big.Int    `json:"number"`
	Round  uint64      `json:"round"`
}

// Vote represents a vote for a proposed block in XDPoS 2.0
type Vote struct {
	ProposedBlockInfo *BlockInfo `json:"proposedBlockInfo"`
	Signature         []byte     `json:"signature"`
	GapNumber         uint64     `json:"gapNumber"`
	
	// Cache
	hash atomic.Value
}

// Hash returns the hash of the vote
func (v *Vote) Hash() common.Hash {
	if hash := v.hash.Load(); hash != nil {
		return hash.(common.Hash)
	}
	
	// Calculate hash from RLP encoding (excluding signature)
	data, _ := rlp.EncodeToBytes([]interface{}{
		v.ProposedBlockInfo,
		v.GapNumber,
	})
	hash := crypto.Keccak256Hash(data)
	v.hash.Store(hash)
	return hash
}

// Copy creates a deep copy of the vote
func (v *Vote) Copy() *Vote {
	cpy := &Vote{
		GapNumber: v.GapNumber,
	}
	
	if v.ProposedBlockInfo != nil {
		cpy.ProposedBlockInfo = &BlockInfo{
			Hash:   v.ProposedBlockInfo.Hash,
			Number: new(big.Int).Set(v.ProposedBlockInfo.Number),
			Round:  v.ProposedBlockInfo.Round,
		}
	}
	
	if v.Signature != nil {
		cpy.Signature = make([]byte, len(v.Signature))
		copy(cpy.Signature, v.Signature)
	}
	
	return cpy
}

// Timeout represents a timeout message in XDPoS 2.0
type Timeout struct {
	Round     uint64      `json:"round"`
	Signature []byte      `json:"signature"`
	HighQC    *QuorumCert `json:"highQC"`
	GapNumber uint64      `json:"gapNumber"`
	
	// Cache
	hash atomic.Value
}

// Hash returns the hash of the timeout
func (t *Timeout) Hash() common.Hash {
	if hash := t.hash.Load(); hash != nil {
		return hash.(common.Hash)
	}
	
	// Calculate hash from RLP encoding (excluding signature)
	data, _ := rlp.EncodeToBytes([]interface{}{
		t.Round,
		t.HighQC,
		t.GapNumber,
	})
	hash := crypto.Keccak256Hash(data)
	t.hash.Store(hash)
	return hash
}

// Copy creates a deep copy of the timeout
func (t *Timeout) Copy() *Timeout {
	cpy := &Timeout{
		Round:     t.Round,
		GapNumber: t.GapNumber,
	}
	
	if t.Signature != nil {
		cpy.Signature = make([]byte, len(t.Signature))
		copy(cpy.Signature, t.Signature)
	}
	
	if t.HighQC != nil {
		cpy.HighQC = t.HighQC.Copy()
	}
	
	return cpy
}

// SyncInfo carries synchronization information
type SyncInfo struct {
	HighestQC *QuorumCert  `json:"highestQC"`
	HighestTC *TimeoutCert `json:"highestTC"`
	
	// Cache
	hash atomic.Value
}

// Hash returns the hash of the sync info
func (s *SyncInfo) Hash() common.Hash {
	if hash := s.hash.Load(); hash != nil {
		return hash.(common.Hash)
	}
	
	data, _ := rlp.EncodeToBytes([]interface{}{
		s.HighestQC,
		s.HighestTC,
	})
	hash := crypto.Keccak256Hash(data)
	s.hash.Store(hash)
	return hash
}

// Copy creates a deep copy of sync info
func (s *SyncInfo) Copy() *SyncInfo {
	cpy := &SyncInfo{}
	
	if s.HighestQC != nil {
		cpy.HighestQC = s.HighestQC.Copy()
	}
	
	if s.HighestTC != nil {
		cpy.HighestTC = s.HighestTC.Copy()
	}
	
	return cpy
}

// QuorumCert represents a quorum certificate
type QuorumCert struct {
	ProposedBlockInfo *BlockInfo  `json:"proposedBlockInfo"`
	Signatures        []Signature `json:"signatures"`
	GapNumber         uint64      `json:"gapNumber"`
	Round             uint64      `json:"round"`
}

// Signature represents a signature from a validator
type Signature struct {
	Signer    common.Address `json:"signer,omitempty"`
	Signature []byte         `json:"signature"`
}

// Copy creates a deep copy of the quorum cert
func (qc *QuorumCert) Copy() *QuorumCert {
	cpy := &QuorumCert{
		GapNumber: qc.GapNumber,
		Round:     qc.Round,
	}
	
	if qc.ProposedBlockInfo != nil {
		cpy.ProposedBlockInfo = &BlockInfo{
			Hash:   qc.ProposedBlockInfo.Hash,
			Number: new(big.Int).Set(qc.ProposedBlockInfo.Number),
			Round:  qc.ProposedBlockInfo.Round,
		}
	}
	
	if qc.Signatures != nil {
		cpy.Signatures = make([]Signature, len(qc.Signatures))
		for i, sig := range qc.Signatures {
			cpy.Signatures[i] = Signature{
				Signer: sig.Signer,
			}
			if sig.Signature != nil {
				cpy.Signatures[i].Signature = make([]byte, len(sig.Signature))
				copy(cpy.Signatures[i].Signature, sig.Signature)
			}
		}
	}
	
	return cpy
}

// TimeoutCert represents a timeout certificate
type TimeoutCert struct {
	Round      uint64      `json:"round"`
	Signatures []Signature `json:"signatures"`
	GapNumber  uint64      `json:"gapNumber"`
}

// Copy creates a deep copy of the timeout cert
func (tc *TimeoutCert) Copy() *TimeoutCert {
	cpy := &TimeoutCert{
		Round:     tc.Round,
		GapNumber: tc.GapNumber,
	}
	
	if tc.Signatures != nil {
		cpy.Signatures = make([]Signature, len(tc.Signatures))
		for i, sig := range tc.Signatures {
			cpy.Signatures[i] = Signature{
				Signer: sig.Signer,
			}
			if sig.Signature != nil {
				cpy.Signatures[i].Signature = make([]byte, len(sig.Signature))
				copy(cpy.Signatures[i].Signature, sig.Signature)
			}
		}
	}
	
	return cpy
}

// ExtraFields_v2 represents v2 extra data in block header
type ExtraFields_v2 struct {
	Round      uint64      `json:"round"`
	QuorumCert *QuorumCert `json:"quorumCert"`
}

// VotePool contains votes grouped by block hash
type VotePool struct {
	votes map[common.Hash][]*Vote
}

// NewVotePool creates a new vote pool
func NewVotePool() *VotePool {
	return &VotePool{
		votes: make(map[common.Hash][]*Vote),
	}
}

// Add adds a vote to the pool
func (p *VotePool) Add(vote *Vote) {
	hash := vote.ProposedBlockInfo.Hash
	p.votes[hash] = append(p.votes[hash], vote)
}

// Get returns all votes for a block
func (p *VotePool) Get(hash common.Hash) []*Vote {
	return p.votes[hash]
}

// Count returns the number of votes for a block
func (p *VotePool) Count(hash common.Hash) int {
	return len(p.votes[hash])
}

// Clear removes all votes for a block
func (p *VotePool) Clear(hash common.Hash) {
	delete(p.votes, hash)
}
