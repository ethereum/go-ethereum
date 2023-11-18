// Copyright 2023 The go-ethereum Authors
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

package main

import (
	"github.com/ethereum/go-ethereum/common"
)

// SyncCommitteeDomain is the signature type for sync committee signatures.
var SyncCommitteeDomain = []byte{7, 0, 0, 0}

// SepoliaChainConfig is the fork schedule and genesis info for the Sepolia beacon chain.
var SepoliaChainConfig = &ChainConfig{
	Genesis:               &ForkConfig{0, []byte{144, 0, 0, 105}},
	Altair:                &ForkConfig{50, []byte{144, 0, 0, 112}},
	Bellatrix:             &ForkConfig{100, []byte{144, 0, 0, 113}},
	Capella:               &ForkConfig{56832, []byte{144, 0, 0, 114}},
	GenesisValidatorsRoot: common.HexToHash("0xd8ea171f3c94aea21ebc42a1ed61052acf3f9209c00e4efbaaddac09ed9b8078"),
}

// Fork config encodes the epoch at which a certain fork must activate and the
// version bytes to be used for signatures.
type ForkConfig struct {
	Epoch   uint64
	Version []byte
}

// ChainConfig represents the fork schedule for a certain chain.
type ChainConfig struct {
	GenesisValidatorsRoot common.Hash

	Genesis   *ForkConfig
	Altair    *ForkConfig
	Bellatrix *ForkConfig
	Capella   *ForkConfig
}

// Version returns the active version for a given slot.
func (c *ChainConfig) Version(slot uint64) []byte {
	epoch := slot / SlotsPerEpoch
	switch {
	case c.Capella.Epoch <= epoch:
		return c.Capella.Version
	case c.Bellatrix.Epoch <= epoch:
		return c.Bellatrix.Version
	case c.Altair.Epoch <= epoch:
		return c.Altair.Version
	default:
		return c.Genesis.Version
	}
}

// Domain returns the domain for a given slot.
func (c *ChainConfig) Domain(typ []byte, slot uint64) common.Hash {
	var (
		forkData = computeForkDataRoot(c.Version(slot), c.GenesisValidatorsRoot)
		domain   common.Hash
	)
	copy(domain[0:4], typ[:])
	copy(domain[4:], forkData[0:28])
	return domain
}
