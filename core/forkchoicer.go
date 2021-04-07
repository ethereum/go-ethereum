// Copyright 2021 The go-ethereum Authors
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
	crand "crypto/rand"
	"errors"
	"math/big"
	mrand "math/rand"
	"sync/atomic"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/math"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/log"
)

// ChainReader defines a small collection of methods needed to access the local
// blockchain during header verification. It's implemented by both blockchain
// and lightchain.
type ChainReader interface {
	// CurrentHeader retrieves the current header from the local chain.
	CurrentHeader() *types.Header

	// GetTd returns the total difficulty of a local block.
	GetTd(common.Hash, uint64) *big.Int
}

// ForkChoicer is the fork choicer based on the highest total difficulty of the
// chain(the fork choice used in the eth1) and the external fork choice (the fork
// choice used in the eth2). This main goal of this ForkChoicer is not only for
// offering fork choice during the eth1/2 merge phase, but also keep the compatibility
// for all other proof-of-work networks.
type ForkChoicer struct {
	chain ChainReader
	seed  *mrand.Rand

	// transitioned is the flag whether the chain has finished the
	// ethash -> transition. It's triggered by receiving the first
	// "NewBlock" message from the external consensus engine.
	transitioned uint32

	// preserve is a helper function used in td fork choice.
	// Miners will prefer to choose the local mined block if the
	// local td is equal to the extern one. It can nil for light
	// client
	preserve func(header *types.Header) bool
}

func NewForkChoicer(chainReader ChainReader, transitioned bool, preserve func(header *types.Header) bool) *ForkChoicer {
	// Seed a fast but crypto originating random generator
	seed, err := crand.Int(crand.Reader, big.NewInt(math.MaxInt64))
	if err != nil {
		log.Crit("Failed to initialize random seed", "err", err)
	}
	forker := &ForkChoicer{
		chain:    chainReader,
		seed:     mrand.New(mrand.NewSource(seed.Int64())),
		preserve: preserve,
	}
	if transitioned {
		forker.SetTransitioned()
	}
	return forker
}

// Reorg returns the result whether the reorg should be applied
// based on the given external header and local canonical chain.
// In the td mode, the new head is chosen if the corresponding
// total difficulty is higher.
func (f *ForkChoicer) Reorg(header *types.Header) (bool, error) {
	// If the chain is already transitioned into the casper phase,
	// always return true because the head is already decided by
	// the external fork choicer.
	if f.IsTransitioned() {
		return true, nil
	}
	var (
		headHeader = f.chain.CurrentHeader()
		localTD    = f.chain.GetTd(headHeader.Hash(), headHeader.Number.Uint64())
		externTd   = f.chain.GetTd(header.Hash(), header.Number.Uint64())
	)
	if localTD == nil || externTd == nil {
		return false, errors.New("missing td")
	}
	// If the total difficulty is higher than our known, add it to the canonical chain
	// Second clause in the if statement reduces the vulnerability to selfish mining.
	// Please refer to http://www.cs.cornell.edu/~ie53/publications/btcProcFC.pdf
	reorg := externTd.Cmp(localTD) > 0
	if !reorg && externTd.Cmp(localTD) == 0 {
		number, headNumber := header.Number.Uint64(), headHeader.Number.Uint64()
		if number < headNumber {
			reorg = true
		} else if number == headNumber {
			var currentPreserve, externPreserve bool
			if f.preserve != nil {
				currentPreserve, externPreserve = f.preserve(headHeader), f.preserve(header)
			}
			reorg = !currentPreserve && (externPreserve || mrand.Float64() < 0.5)
		}
	}
	return true, nil
}

// SetTransitioned marks the transition has been done.
func (f *ForkChoicer) SetTransitioned() {
	atomic.StoreUint32(&f.transitioned, 1)
}

// IsTransitioned reports whether the transition has finished.
func (f *ForkChoicer) IsTransitioned() bool {
	return atomic.LoadUint32(&f.transitioned) == 1
}
