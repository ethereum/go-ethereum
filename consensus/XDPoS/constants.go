// Copyright (c) 2018 XDCchain
// Copyright 2024 The go-ethereum Authors
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with this program. If not, see <http://www.gnu.org/licenses/>.

package XDPoS

import "math/big"

// XDC Network constants
const (
	// Reward distribution percentages
	RewardMasterPercent     = 90
	RewardVoterPercent      = 0
	RewardFoundationPercent = 10

	// Method signatures for contract calls
	HexSignMethod  = "e341eaa4"
	HexSetSecret   = "34d38600"
	HexSetOpening  = "e11f5ba2"

	// Epoch block offsets
	EpocBlockSecret    = 800
	EpocBlockOpening   = 850
	EpocBlockRandomize = 900

	// Masternode limits
	MaxMasternodes   = 18
	MaxMasternodesV2 = 108

	// Penalty configuration
	LimitPenaltyEpoch = 4

	// Blocks per year (assuming 2 second blocks)
	BlocksPerYear = uint64(15768000)

	// Queue limits
	LimitThresholdNonceInQueue = 10

	// Gas price
	DefaultMinGasPrice = 2500

	// Block signing
	MergeSignRange    = 15
	RangeReturnSigner = 150

	// Minimum miner blocks per epoch
	MinimunMinerBlockPerEpoch = 1
)

// Fork block numbers
var (
	TIP2019Block           = big.NewInt(1)
	TIPSigning             = big.NewInt(3000000)
	TIPRandomize           = big.NewInt(3464000)
	TIPIncreaseMasternodes = big.NewInt(5000000) // Upgrade MN Count at Block
)

// Global configuration variables
var (
	IsTestnet         bool   = false
	StoreRewardFolder string
	MinGasPrice       int64
)
