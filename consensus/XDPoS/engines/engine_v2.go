// Copyright 2023 The XDC Network Authors
// This file is part of the XDC Network library.
//
// The XDC Network library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.

// Package engines contains XDPoS 2.0 engine implementations.
package engines

import (
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
	"github.com/ethereum/go-ethereum/rlp"
)

var (
	// ErrInvalidRound is returned when round is invalid
	ErrInvalidRound = errors.New("invalid round")
	
	// ErrFutureBlock is returned for future blocks
	ErrFutureBlock = errors.New("block in future")
	
	// ErrInvalidQC is returned for invalid quorum certificate
	ErrInvalidQC = errors.New("invalid quorum certificate")
	
	// ExtraVanity is the fixed number of extra-data prefix bytes reserved for signer vanity
	ExtraVanity = 32
	
	// ExtraSeal is the fixed number of extra-data suffix bytes reserved for signer seal
	ExtraSeal = 65
)

// V2Engine implements XDPoS 2.0 consensus
type V2Engine struct {
	config     *params.XDPoSConfig
	db         consensus.ChainHeaderReader
	lock       sync.RWMutex
	
	// State
	round      uint64
	epoch      uint64
	highQC     *types.QuorumCert
	
	// Signer
	signer     common.Address
	signFn     SignerFn
	
	// Services
	xdcxService    interface{}
	lendingService interface{}
}

// SignerFn is a callback for signing
type SignerFn func(signer common.Address, data []byte) ([]byte, error)

// NewV2Engine creates a new V2 engine
func NewV2Engine(config *params.XDPoSConfig, db consensus.ChainHeaderReader) *V2Engine {
	return &V2Engine{
		config: config,
		db:     db,
	}
}

// Author implements consensus.Engine
func (e *V2Engine) Author(header *types.Header) (common.Address, error) {
	return header.Coinbase, nil
}

// VerifyHeader implements consensus.Engine
func (e *V2Engine) VerifyHeader(chain consensus.ChainHeaderReader, header *types.Header, seal bool) error {
	return e.verifyHeader(chain, header, nil, seal)
}

// VerifyHeaders implements consensus.Engine
func (e *V2Engine) VerifyHeaders(chain consensus.ChainHeaderReader, headers []*types.Header, seals []bool) (chan<- struct{}, <-chan error) {
	abort := make(chan struct{})
	results := make(chan error, len(headers))
	
	go func() {
		for i, header := range headers {
			err := e.verifyHeader(chain, header, headers[:i], seals[i])
			select {
			case <-abort:
				return
			case results <- err:
			}
		}
	}()
	
	return abort, results
}

func (e *V2Engine) verifyHeader(chain consensus.ChainHeaderReader, header *types.Header, parents []*types.Header, seal bool) error {
	if header.Number == nil {
		return consensus.ErrUnknownAncestor
	}
	
	// Verify future block
	if header.Time > uint64(time.Now().Unix()+15) {
		return ErrFutureBlock
	}
	
	// Verify extra data length
	if len(header.Extra) < ExtraVanity {
		return errors.New("missing extra vanity")
	}
	if len(header.Extra) < ExtraVanity+ExtraSeal {
		return errors.New("missing seal")
	}
	
	// Verify difficulty is 1 for V2
	if header.Difficulty.Cmp(big.NewInt(1)) != 0 {
		return errors.New("invalid difficulty for V2")
	}
	
	// Verify seal if required
	if seal {
		return e.verifySeal(chain, header, parents)
	}
	
	return nil
}

func (e *V2Engine) verifySeal(chain consensus.ChainHeaderReader, header *types.Header, parents []*types.Header) error {
	// Extract signature
	signature := header.Extra[len(header.Extra)-ExtraSeal:]
	
	// Recover signer
	sealHash := e.SealHash(header)
	pubkey, err := crypto.SigToPub(sealHash.Bytes(), signature)
	if err != nil {
		return err
	}
	
	signer := crypto.PubkeyToAddress(*pubkey)
	
	// Verify signer is authorized (would check masternode list)
	log.Debug("Verified block seal", "signer", signer.Hex())
	
	return nil
}

// VerifyUncles implements consensus.Engine
func (e *V2Engine) VerifyUncles(chain consensus.ChainReader, block *types.Block) error {
	// XDPoS doesn't have uncles
	if len(block.Uncles()) > 0 {
		return errors.New("uncles not allowed")
	}
	return nil
}

// Prepare implements consensus.Engine
func (e *V2Engine) Prepare(chain consensus.ChainHeaderReader, header *types.Header) error {
	// Set difficulty to 1
	header.Difficulty = big.NewInt(1)
	
	// Prepare extra data
	if len(header.Extra) < ExtraVanity {
		header.Extra = append(header.Extra, make([]byte, ExtraVanity-len(header.Extra))...)
	}
	header.Extra = header.Extra[:ExtraVanity]
	header.Extra = append(header.Extra, make([]byte, ExtraSeal)...)
	
	return nil
}

// Finalize implements consensus.Engine
func (e *V2Engine) Finalize(chain consensus.ChainHeaderReader, header *types.Header, state *state.StateDB, body *types.Body) {
	// Add block reward
	reward := big.NewInt(0).Mul(big.NewInt(250), big.NewInt(1e18))
	state.AddBalance(header.Coinbase, reward, 0)
}

// FinalizeAndAssemble implements consensus.Engine
func (e *V2Engine) FinalizeAndAssemble(chain consensus.ChainHeaderReader, header *types.Header, state *state.StateDB, body *types.Body, receipts []*types.Receipt) (*types.Block, error) {
	e.Finalize(chain, header, state, body)
	return types.NewBlock(header, body, receipts, nil), nil
}

// Seal implements consensus.Engine
func (e *V2Engine) Seal(chain consensus.ChainHeaderReader, block *types.Block, results chan<- *types.Block, stop <-chan struct{}) error {
	header := block.Header()
	
	if header.Number.Uint64() == 0 {
		return errors.New("cannot seal genesis")
	}
	
	e.lock.RLock()
	signer, signFn := e.signer, e.signFn
	e.lock.RUnlock()
	
	if signFn == nil {
		return errors.New("sign function not set")
	}
	
	// Sign the seal hash
	sealHash := e.SealHash(header)
	signature, err := signFn(signer, sealHash.Bytes())
	if err != nil {
		return err
	}
	
	copy(header.Extra[len(header.Extra)-ExtraSeal:], signature)
	
	select {
	case results <- block.WithSeal(header):
	default:
		log.Warn("Sealing result not consumed")
	}
	
	return nil
}

// SealHash returns the hash that is used for signing
func (e *V2Engine) SealHash(header *types.Header) common.Hash {
	return SealHash(header)
}

// SealHash calculates the seal hash
func SealHash(header *types.Header) common.Hash {
	return rlpHash([]interface{}{
		header.ParentHash,
		header.UncleHash,
		header.Coinbase,
		header.Root,
		header.TxHash,
		header.ReceiptHash,
		header.Bloom,
		header.Difficulty,
		header.Number,
		header.GasLimit,
		header.GasUsed,
		header.Time,
		header.Extra[:len(header.Extra)-ExtraSeal], // Exclude seal
		header.MixDigest,
		header.Nonce,
	})
}

func rlpHash(x interface{}) common.Hash {
	data, _ := rlp.EncodeToBytes(x)
	return crypto.Keccak256Hash(data)
}

// CalcDifficulty implements consensus.Engine
func (e *V2Engine) CalcDifficulty(chain consensus.ChainHeaderReader, time uint64, parent *types.Header) *big.Int {
	return big.NewInt(1)
}

// APIs implements consensus.Engine
func (e *V2Engine) APIs(chain consensus.ChainHeaderReader) []consensus.API {
	return nil
}

// Close implements consensus.Engine
func (e *V2Engine) Close() error {
	return nil
}

// Authorize sets the signer
func (e *V2Engine) Authorize(signer common.Address, signFn SignerFn) {
	e.lock.Lock()
	defer e.lock.Unlock()
	
	e.signer = signer
	e.signFn = signFn
}

// SetXDCxService sets the XDCx service
func (e *V2Engine) SetXDCxService(service interface{}) {
	e.lock.Lock()
	defer e.lock.Unlock()
	e.xdcxService = service
}

// SetLendingService sets the lending service
func (e *V2Engine) SetLendingService(service interface{}) {
	e.lock.Lock()
	defer e.lock.Unlock()
	e.lendingService = service
}

// GetXDCXService returns the XDCx service
func (e *V2Engine) GetXDCXService() interface{} {
	e.lock.RLock()
	defer e.lock.RUnlock()
	return e.xdcxService
}

// GetLendingService returns the lending service
func (e *V2Engine) GetLendingService() interface{} {
	e.lock.RLock()
	defer e.lock.RUnlock()
	return e.lendingService
}

// HandleVote handles a vote message
func (e *V2Engine) HandleVote(vote *types.Vote) error {
	e.lock.Lock()
	defer e.lock.Unlock()
	
	log.Debug("Handling vote",
		"block", vote.ProposedBlockInfo.Hash,
		"round", vote.ProposedBlockInfo.Round,
	)
	
	return nil
}

// HandleTimeout handles a timeout message
func (e *V2Engine) HandleTimeout(timeout *types.Timeout) error {
	e.lock.Lock()
	defer e.lock.Unlock()
	
	log.Debug("Handling timeout", "round", timeout.Round)
	
	return nil
}

// HandleSyncInfo handles a sync info message
func (e *V2Engine) HandleSyncInfo(syncInfo *types.SyncInfo) error {
	e.lock.Lock()
	defer e.lock.Unlock()
	
	// Update high QC if newer
	if syncInfo.HighestQC != nil {
		if e.highQC == nil || syncInfo.HighestQC.Round > e.highQC.Round {
			e.highQC = syncInfo.HighestQC
		}
	}
	
	return nil
}

// HandleProposedBlock handles a proposed block
func (e *V2Engine) HandleProposedBlock(chain consensus.ChainHeaderReader, header *types.Header) error {
	return e.VerifyHeader(chain, header, true)
}

// IsEpochSwitch checks if block is an epoch switch
func (e *V2Engine) IsEpochSwitch(header *types.Header) (bool, uint64, error) {
	number := header.Number.Uint64()
	if e.config.Epoch == 0 {
		return false, 0, nil
	}
	
	isSwitch := number%e.config.Epoch == 0
	epoch := number / e.config.Epoch
	
	return isSwitch, epoch, nil
}

// GetCurrentRound returns the current round
func (e *V2Engine) GetCurrentRound() uint64 {
	e.lock.RLock()
	defer e.lock.RUnlock()
	return e.round
}

// GetCurrentEpoch returns the current epoch
func (e *V2Engine) GetCurrentEpoch() uint64 {
	e.lock.RLock()
	defer e.lock.RUnlock()
	return e.epoch
}
