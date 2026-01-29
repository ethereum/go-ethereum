// Copyright 2023 The XDC Network Authors
// VotePool for XDPoS 2.0 consensus

package types

import (
	"sync"

	"github.com/ethereum/go-ethereum/common"
)

// VotePool contains votes grouped by block hash
type VotePool struct {
	mu    sync.RWMutex
	votes map[common.Hash][]*VoteXDPoS
}

// NewVotePool creates a new vote pool
func NewVotePool() *VotePool {
	return &VotePool{
		votes: make(map[common.Hash][]*VoteXDPoS),
	}
}

// Add adds a vote to the pool
func (p *VotePool) Add(vote *VoteXDPoS) {
	p.mu.Lock()
	defer p.mu.Unlock()
	
	if vote.ProposedBlockInfo == nil {
		return
	}
	hash := vote.ProposedBlockInfo.Hash
	p.votes[hash] = append(p.votes[hash], vote)
}

// Get returns all votes for a block
func (p *VotePool) Get(hash common.Hash) []*VoteXDPoS {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.votes[hash]
}

// Count returns the number of votes for a block
func (p *VotePool) Count(hash common.Hash) int {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return len(p.votes[hash])
}

// Clear removes all votes for a block
func (p *VotePool) Clear(hash common.Hash) {
	p.mu.Lock()
	defer p.mu.Unlock()
	delete(p.votes, hash)
}

// TimeoutPool contains timeouts grouped by round
type TimeoutPool struct {
	mu       sync.RWMutex
	timeouts map[Round][]*Timeout
}

// NewTimeoutPool creates a new timeout pool
func NewTimeoutPool() *TimeoutPool {
	return &TimeoutPool{
		timeouts: make(map[Round][]*Timeout),
	}
}

// Add adds a timeout to the pool
func (p *TimeoutPool) Add(timeout *Timeout) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.timeouts[timeout.Round] = append(p.timeouts[timeout.Round], timeout)
}

// Get returns all timeouts for a round
func (p *TimeoutPool) Get(round Round) []*Timeout {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.timeouts[round]
}

// Count returns the number of timeouts for a round
func (p *TimeoutPool) Count(round Round) int {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return len(p.timeouts[round])
}

// Clear removes all timeouts for a round
func (p *TimeoutPool) Clear(round Round) {
	p.mu.Lock()
	defer p.mu.Unlock()
	delete(p.timeouts, round)
}
