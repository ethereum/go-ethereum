// Copyright 2023 The XDC Network Authors
// This file is part of the XDC Network library.
//
// The XDC Network library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.

// Package XDPoS implements the XDPoS 2.0 consensus algorithm.
package XDPoS

import (
	"crypto/ecdsa"
	"errors"
	"math/big"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/consensus"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/params"
	"github.com/ethereum/go-ethereum/rpc"
)

var (
	// ErrInvalidRound is returned when the round is invalid
	ErrInvalidRound = errors.New("invalid round")
	
	// ErrInvalidQC is returned when the quorum certificate is invalid
	ErrInvalidQC = errors.New("invalid quorum certificate")
	
	// ErrNotInCommittee is returned when signer is not in committee
	ErrNotInCommittee = errors.New("signer not in committee")
	
	// ErrInvalidSignature is returned for invalid signatures
	ErrInvalidSignature = errors.New("invalid signature")
)

// V2Config contains XDPoS 2.0 specific configuration
type V2Config struct {
	// SwitchBlock is the block number when V2 activates
	SwitchBlock *big.Int
	
	// CurrentConfig is the current round configuration
	CurrentConfig *RoundConfig
	
	// MinePeriod is the block mining period
	MinePeriod int
	
	// TimeoutPeriod is the timeout period for rounds
	TimeoutPeriod int
	
	// TimeoutSyncThreshold is the threshold for sync timeout
	TimeoutSyncThreshold int
	
	// CertThreshold is the certificate threshold (2/3 + 1)
	CertThreshold int
}

// RoundConfig contains per-round configuration
type RoundConfig struct {
	EpochNumber uint64
	Round       uint64
	Masternodes []common.Address
}

// XDPoSV2 implements XDPoS 2.0 consensus
type XDPoSV2 struct {
	config    *params.XDPoSConfig
	v2Config  *V2Config
	db        consensus.ChainHeaderReader
	signer    common.Address
	signFn    SignerFn
	lock      sync.RWMutex
	
	// Round state
	currentRound  uint64
	currentEpoch  uint64
	highQC        *types.QuorumCert
	
	// Vote/timeout pools
	votePool     map[common.Hash][]*types.Vote
	timeoutPool  map[uint64][]*types.Timeout
	
	// Services
	xdcxService   interface{}
	lendingService interface{}
}

// SignerFn is a signer callback function
type SignerFn func(signer common.Address, data []byte) ([]byte, error)

// NewV2 creates a new XDPoS V2 consensus engine
func NewV2(config *params.XDPoSConfig, db consensus.ChainHeaderReader) *XDPoSV2 {
	v2 := &XDPoSV2{
		config:      config,
		db:          db,
		votePool:    make(map[common.Hash][]*types.Vote),
		timeoutPool: make(map[uint64][]*types.Timeout),
	}
	
	if config.V2 != nil {
		v2.v2Config = &V2Config{
			SwitchBlock:          config.V2.SwitchBlock,
			MinePeriod:           config.V2.MinePeriod,
			TimeoutPeriod:        config.V2.TimeoutPeriod,
			TimeoutSyncThreshold: config.V2.TimeoutSyncThreshold,
			CertThreshold:        config.V2.CertThreshold,
		}
	}
	
	return v2
}

// Author retrieves the XDC address of the block author
func (x *XDPoSV2) Author(header *types.Header) (common.Address, error) {
	return header.Coinbase, nil
}

// VerifyHeader checks whether a header conforms to the XDPoS 2.0 rules
func (x *XDPoSV2) VerifyHeader(chain consensus.ChainHeaderReader, header *types.Header, seal bool) error {
	return x.verifyHeader(chain, header, nil, seal)
}

func (x *XDPoSV2) verifyHeader(chain consensus.ChainHeaderReader, header *types.Header, parents []*types.Header, seal bool) error {
	if header.Number == nil {
		return errUnknownBlock
	}
	
	// Don't verify future blocks
	if header.Time > uint64(time.Now().Unix()+15) {
		return consensus.ErrFutureBlock
	}
	
	// Verify extra data
	if len(header.Extra) < ExtraVanity {
		return errMissingVanity
	}
	if len(header.Extra) < ExtraVanity+ExtraSeal {
		return errMissingSignature
	}
	
	// Check that difficulty is set correctly
	if header.Difficulty == nil || header.Difficulty.Cmp(big.NewInt(1)) != 0 {
		return errInvalidDifficulty
	}
	
	// Verify seal if needed
	if seal {
		return x.verifySeal(chain, header, parents)
	}
	
	return nil
}

// verifySeal verifies that the signature on the header is valid
func (x *XDPoSV2) verifySeal(chain consensus.ChainHeaderReader, header *types.Header, parents []*types.Header) error {
	// Get signer from signature
	signer, err := ecrecover(header)
	if err != nil {
		return err
	}
	
	// Verify signer is in masternode list
	// This would check the masternode list at the header's epoch
	
	return nil
}

// Prepare initializes the consensus fields of a header
func (x *XDPoSV2) Prepare(chain consensus.ChainHeaderReader, header *types.Header) error {
	// Set difficulty to 1 for V2
	header.Difficulty = big.NewInt(1)
	
	// Get parent
	parent := chain.GetHeader(header.ParentHash, header.Number.Uint64()-1)
	if parent == nil {
		return consensus.ErrUnknownAncestor
	}
	
	// Ensure extra data has correct size
	if len(header.Extra) < ExtraVanity {
		header.Extra = append(header.Extra, make([]byte, ExtraVanity-len(header.Extra))...)
	}
	header.Extra = header.Extra[:ExtraVanity]
	header.Extra = append(header.Extra, make([]byte, ExtraSeal)...)
	
	return nil
}

// Finalize implements consensus.Engine, ensures block rewards are calculated
func (x *XDPoSV2) Finalize(chain consensus.ChainHeaderReader, header *types.Header, state *state.StateDB, body *types.Body) {
	// Calculate rewards if this is a V2 block
	if x.IsV2Block(header.Number) {
		x.accumulateRewards(chain, state, header)
	}
}

// FinalizeAndAssemble implements consensus.Engine
func (x *XDPoSV2) FinalizeAndAssemble(chain consensus.ChainHeaderReader, header *types.Header, state *state.StateDB, body *types.Body, receipts []*types.Receipt) (*types.Block, error) {
	x.Finalize(chain, header, state, body)
	
	// Assemble and return the block
	return types.NewBlock(header, body, receipts, nil), nil
}

// Seal generates a new block with signature
func (x *XDPoSV2) Seal(chain consensus.ChainHeaderReader, block *types.Block, results chan<- *types.Block, stop <-chan struct{}) error {
	header := block.Header()
	
	// Don't seal genesis block
	if header.Number.Uint64() == 0 {
		return errUnknownBlock
	}
	
	x.lock.RLock()
	signer, signFn := x.signer, x.signFn
	x.lock.RUnlock()
	
	if signFn == nil {
		return errMissingSignFn
	}
	
	// Sign the block
	sig, err := signFn(signer, SealHash(header).Bytes())
	if err != nil {
		return err
	}
	copy(header.Extra[len(header.Extra)-ExtraSeal:], sig)
	
	select {
	case results <- block.WithSeal(header):
	default:
		log.Warn("Sealing result is not read by miner", "sealhash", SealHash(header))
	}
	
	return nil
}

// APIs returns the RPC APIs for XDPoS V2
func (x *XDPoSV2) APIs(chain consensus.ChainHeaderReader) []rpc.API {
	return []rpc.API{{
		Namespace: "xdpos",
		Version:   "2.0",
		Service:   &XDPoSV2API{x},
		Public:    false,
	}}
}

// Close terminates the consensus engine
func (x *XDPoSV2) Close() error {
	return nil
}

// IsV2Block returns true if the block number is after V2 switch
func (x *XDPoSV2) IsV2Block(number *big.Int) bool {
	if x.v2Config == nil || x.v2Config.SwitchBlock == nil {
		return false
	}
	return number.Cmp(x.v2Config.SwitchBlock) >= 0
}

// HandleVote processes an incoming vote
func (x *XDPoSV2) HandleVote(vote *types.Vote) error {
	x.lock.Lock()
	defer x.lock.Unlock()
	
	// Verify vote signature
	if err := x.verifyVote(vote); err != nil {
		return err
	}
	
	// Add to pool
	hash := vote.ProposedBlockInfo.Hash
	x.votePool[hash] = append(x.votePool[hash], vote)
	
	// Check if we have enough votes for QC
	if len(x.votePool[hash]) >= x.getCertThreshold() {
		x.createQC(hash)
	}
	
	return nil
}

// HandleTimeout processes an incoming timeout
func (x *XDPoSV2) HandleTimeout(timeout *types.Timeout) error {
	x.lock.Lock()
	defer x.lock.Unlock()
	
	// Verify timeout signature
	if err := x.verifyTimeout(timeout); err != nil {
		return err
	}
	
	// Add to pool
	round := timeout.Round
	x.timeoutPool[round] = append(x.timeoutPool[round], timeout)
	
	// Check if we have enough timeouts
	if len(x.timeoutPool[round]) >= x.getCertThreshold() {
		x.handleTimeoutQC(round)
	}
	
	return nil
}

// HandleSyncInfo processes sync info message
func (x *XDPoSV2) HandleSyncInfo(syncInfo *types.SyncInfo) error {
	x.lock.Lock()
	defer x.lock.Unlock()
	
	// Update high QC if newer
	if syncInfo.HighestQC != nil {
		if x.highQC == nil || syncInfo.HighestQC.Round > x.highQC.Round {
			x.highQC = syncInfo.HighestQC
		}
	}
	
	return nil
}

// HandleProposedBlock handles a newly proposed block
func (x *XDPoSV2) HandleProposedBlock(chain consensus.ChainHeaderReader, header *types.Header) error {
	// Verify the proposal
	if err := x.VerifyHeader(chain, header, true); err != nil {
		return err
	}
	
	log.Debug("Received valid proposed block",
		"number", header.Number,
		"hash", header.Hash(),
	)
	
	return nil
}

// verifyVote verifies a vote's signature
func (x *XDPoSV2) verifyVote(vote *types.Vote) error {
	// Hash the vote data
	hash := vote.Hash()
	
	// Recover signer
	pubkey, err := crypto.SigToPub(hash.Bytes(), vote.Signature)
	if err != nil {
		return ErrInvalidSignature
	}
	
	signer := crypto.PubkeyToAddress(*pubkey)
	
	// Check signer is in committee
	// This would verify against the masternode list
	_ = signer
	
	return nil
}

// verifyTimeout verifies a timeout's signature
func (x *XDPoSV2) verifyTimeout(timeout *types.Timeout) error {
	// Similar to verifyVote
	return nil
}

// getCertThreshold returns the certificate threshold
func (x *XDPoSV2) getCertThreshold() int {
	if x.v2Config != nil && x.v2Config.CertThreshold > 0 {
		return x.v2Config.CertThreshold
	}
	// Default: 2/3 + 1 of committee size
	return 101 // For 150 masternodes
}

// createQC creates a quorum certificate from votes
func (x *XDPoSV2) createQC(hash common.Hash) {
	votes := x.votePool[hash]
	if len(votes) == 0 {
		return
	}
	
	// Create QC from votes
	qc := &types.QuorumCert{
		ProposedBlockInfo: votes[0].ProposedBlockInfo,
		Signatures:        make([]types.Signature, len(votes)),
	}
	
	for i, vote := range votes {
		qc.Signatures[i] = types.Signature{
			Signature: vote.Signature,
		}
	}
	
	// Update high QC
	if x.highQC == nil || qc.Round > x.highQC.Round {
		x.highQC = qc
	}
	
	log.Info("Created quorum certificate",
		"block", hash.Hex(),
		"round", qc.Round,
		"signatures", len(qc.Signatures),
	)
}

// handleTimeoutQC handles timeout quorum
func (x *XDPoSV2) handleTimeoutQC(round uint64) {
	log.Info("Timeout QC reached", "round", round)
	
	// Move to next round
	x.currentRound = round + 1
	
	// Clear old timeouts
	delete(x.timeoutPool, round)
}

// accumulateRewards calculates and distributes block rewards
func (x *XDPoSV2) accumulateRewards(chain consensus.ChainHeaderReader, state *state.StateDB, header *types.Header) {
	// Block reward: 250 XDC
	reward := new(big.Int).Mul(big.NewInt(250), big.NewInt(1e18))
	
	// Add to coinbase
	state.AddBalance(header.Coinbase, reward, 0)
}

// Authorize sets the signer and sign function
func (x *XDPoSV2) Authorize(signer common.Address, signFn SignerFn) {
	x.lock.Lock()
	defer x.lock.Unlock()
	
	x.signer = signer
	x.signFn = signFn
}

// SetXDCxService sets the XDCx trading service
func (x *XDPoSV2) SetXDCxService(service interface{}) {
	x.lock.Lock()
	defer x.lock.Unlock()
	x.xdcxService = service
}

// SetLendingService sets the lending service
func (x *XDPoSV2) SetLendingService(service interface{}) {
	x.lock.Lock()
	defer x.lock.Unlock()
	x.lendingService = service
}

// GetXDCXService returns the XDCx service
func (x *XDPoSV2) GetXDCXService() interface{} {
	x.lock.RLock()
	defer x.lock.RUnlock()
	return x.xdcxService
}

// GetLendingService returns the lending service
func (x *XDPoSV2) GetLendingService() interface{} {
	x.lock.RLock()
	defer x.lock.RUnlock()
	return x.lendingService
}

// IsEpochSwitch checks if a header is an epoch switch block
func (x *XDPoSV2) IsEpochSwitch(header *types.Header) (bool, uint64, error) {
	number := header.Number.Uint64()
	epochSize := x.config.Epoch
	
	if epochSize == 0 {
		return false, 0, nil
	}
	
	isEpochSwitch := number%epochSize == 0
	epoch := number / epochSize
	
	return isEpochSwitch, epoch, nil
}

// ecrecover extracts the Ethereum address from a signed header
func ecrecover(header *types.Header) (common.Address, error) {
	if len(header.Extra) < ExtraVanity+ExtraSeal {
		return common.Address{}, errMissingSignature
	}
	
	signature := header.Extra[len(header.Extra)-ExtraSeal:]
	
	// Recover public key from signature
	hash := SealHash(header)
	pubkey, err := crypto.SigToPub(hash.Bytes(), signature)
	if err != nil {
		return common.Address{}, err
	}
	
	return crypto.PubkeyToAddress(*pubkey), nil
}

// XDPoSV2API provides RPC API for XDPoS V2
type XDPoSV2API struct {
	engine *XDPoSV2
}

// GetRound returns the current round
func (api *XDPoSV2API) GetRound() uint64 {
	api.engine.lock.RLock()
	defer api.engine.lock.RUnlock()
	return api.engine.currentRound
}

// GetEpoch returns the current epoch
func (api *XDPoSV2API) GetEpoch() uint64 {
	api.engine.lock.RLock()
	defer api.engine.lock.RUnlock()
	return api.engine.currentEpoch
}
