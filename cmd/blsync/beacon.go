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
	"crypto/sha256"

	"github.com/ethereum/go-ethereum/common"
)

const (
	MinSyncCommitteeParticipants = 1
	SecondsPerSlot               = 12
	SlotsPerEpoch                = 32
	EpochsPerSyncCommitteePeriod = 256
)

// https://github.com/ethereum/consensus-specs/blob/dev/specs/phase0/beacon-chain.md#compute_fork_data_root
func computeForkDataRoot(version []byte, genesisValidatorsRoot common.Hash) common.Hash {
	var padded common.Hash
	copy(padded[:], version)
	return hash(padded.Bytes(), genesisValidatorsRoot.Bytes())
}

func computeSigningRoot(root, domain common.Hash) common.Hash {
	return hash(root.Bytes(), domain.Bytes())
}

func hash(left, right []byte) common.Hash {
	var (
		hasher = sha256.New()
		sum    common.Hash
	)
	hasher.Write(left)
	hasher.Write(right)
	hasher.Sum(sum[:0])
	return sum
}
