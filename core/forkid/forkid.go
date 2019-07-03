// Copyright 2019 The go-ethereum Authors
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

// Package forkid implements EIP-2124 (https://eips.ethereum.org/EIPS/eip-2124).
package forkid

import (
	"encoding/binary"
	"errors"
	"fmt"
	"hash/crc32"
	"math"
	"math/big"
	"reflect"
	"strings"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/params"
)

var (
	// ErrRemoteStale is returned by the validator if a remote fork checksum is a
	// subset of our already applied forks, but the announced next fork block is
	// not on our already passed chain.
	ErrRemoteStale = errors.New("remote needs update")

	// ErrLocalIncompatibleOrStale is returned by the validator if a remote fork
	// checksum does not match any local checksum variation, signalling that the
	// two chains have diverged in the past at some point (possibly at genesis).
	ErrLocalIncompatibleOrStale = errors.New("local incompatible or needs update")
)

// ID is a 2x4-byte tuple containing:
//   - forkhash: CRC32 checksum of the genesis block and passed fork block numbers
//   - forknext: CRC32 checksum of the next fork block number (or 0, if no known)
type ID [8]byte

// NewID calculates the Ethereum fork ID from the chain config and head.
func NewID(chain *core.BlockChain) ID {
	return newID(
		chain.Config(),
		chain.Genesis().Hash(),
		chain.CurrentHeader().Number.Uint64(),
	)
}

// newID is the internal version of NewID, which takes extracted values as its
// arguments instead of a chain. The reason is to allow testing the IDs without
// having to simulate an entire blockchain.
func newID(config *params.ChainConfig, genesis common.Hash, head uint64) ID {
	// Calculate the starting checksum from the genesis hash
	forkHash := crc32.ChecksumIEEE(genesis[:])

	// Calculate the current fork checksum and the next fork block
	var forkNext uint32
	for _, fork := range gatherForks(config) {
		if fork <= head {
			// Fork already passed, checksum the previous hash and the fork number
			forkHash = checksumUpdate(forkHash, fork)
			continue
		}
		forkNext = checksum(fork)
		break
	}
	// Aggregate everything into a single binary blob
	var entry ID
	binary.BigEndian.PutUint32(entry[0:], forkHash)
	binary.BigEndian.PutUint32(entry[4:], forkNext)
	return entry
}

// NewFilter creates an filter that returns if a fork ID should be rejected or not
// based on the local chain's status.
func NewFilter(chain *core.BlockChain) func(id ID) error {
	return newFilter(
		chain.Config(),
		chain.Genesis().Hash(),
		func() uint64 {
			return chain.CurrentHeader().Number.Uint64()
		},
	)
}

// newFilter is the internal version of NewFilter, taking closures as its arguments
// instead of a chain. The reason is to allow testing it without having to simulate
// an entire blockchain.
func newFilter(config *params.ChainConfig, genesis common.Hash, headfn func() uint64) func(id ID) error {
	// Calculate the all the valid fork hash and fork next combos
	var (
		forks = gatherForks(config)
		sums  = make([]uint32, len(forks)+1) // 0th is the genesis
		next  = make([]uint32, len(forks))
	)
	sums[0] = crc32.ChecksumIEEE(genesis[:])
	for i, fork := range forks {
		sums[i+1] = checksumUpdate(sums[i], fork)
		next[i] = checksum(fork)
	}
	// Add two sentries to simplify the fork checks and don't require special
	// casing the last one.
	forks = append(forks, math.MaxUint64) // Last fork will never be passed
	next = append(next, 0)                // Last fork is all 0

	// Create a validator that will filter out incompatible chains
	return func(id ID) error {
		// Run the fork checksum validation ruleset:
		//   1. If local and remote FORK_CSUM matches, connect.
		//        The two nodes are in the same fork state currently. They might know
		//        of differing future forks, but that's not relevant until the fork
		//        triggers (might be postponed, nodes might be updated to match).
		//   2. If the remote FORK_CSUM is a subset of the local past forks and the
		//      remote FORK_NEXT matches with the locally following fork block number,
		//      connect.
		//        Remote node is currently syncing. It might eventually diverge from
		//        us, but at this current point in time we don't have enough information.
		//   3. If the remote FORK_CSUM is a superset of the local past forks and can
		//      be completed with locally known future forks, connect.
		//        Local node is currently syncing. It might eventually diverge from
		//        the remote, but at this current point in time we don't have enough
		//        information.
		//   4. Reject in all other cases.
		var (
			head       = headfn()
			remoteSum  = binary.BigEndian.Uint32(id[0:4])
			remoteNext = binary.BigEndian.Uint32(id[4:8])
		)
		for i, fork := range forks {
			// If our head is beyond this fork, continue to the next (we have a dummy
			// fork of maxuint64 as the last item to always fail this check eventually).
			if head > fork {
				continue
			}
			// Found the first unpassed fork block, check if our current state matches
			// the remote checksum (rule #1).
			if sums[i] == remoteSum {
				// Yay, fork checksum matched, ignore any upcoming fork
				return nil
			}
			// The local and remote nodes are in different forks currently, check if the
			// remote checksum is a subset of our local forks (rule #2).
			for j := 0; j < i; j++ {
				if sums[j] == remoteSum {
					// Remote checksum is a subset, validate based on the announced next fork
					if next[j] != remoteNext {
						return ErrRemoteStale
					}
					return nil
				}
			}
			// Remote chain is not a subset of our local one, check if it's a superset by
			// any chance, signalling that we're simply out of sync (rule #3).
			for j := i + 1; j < len(sums); j++ {
				if sums[j] == remoteSum {
					// Yay, remote checksum is a superset, ignore upcoming forks
					return nil
				}
			}
			// No exact, subset or superset match. We are on differing chains, reject.
			return ErrLocalIncompatibleOrStale
		}
		log.Error("Impossible fork ID validation", "id", fmt.Sprintf("%x", id))
		return nil // Something's very wrong, accept rather than reject
	}
}

// checksum calculates the IEEE CRC32 checksum of a block number.
func checksum(fork uint64) uint32 {
	var blob [8]byte
	binary.BigEndian.PutUint64(blob[:], fork)
	return crc32.ChecksumIEEE(blob[:])
}

// checksumUpdate calculates the next IEEE CRC32 checksum based on the previous
// one and a fork block number (equivalent to CRC32(original-blob || fork)).
func checksumUpdate(hash uint32, fork uint64) uint32 {
	var blob [8]byte
	binary.BigEndian.PutUint64(blob[:], fork)
	return crc32.Update(hash, crc32.IEEETable, blob[:])
}

// gatherForks gathers all the known forks and creates a sorted list out of them.
func gatherForks(config *params.ChainConfig) []uint64 {
	// Gather all the fork block numbers via reflection
	kind := reflect.TypeOf(params.ChainConfig{})
	conf := reflect.ValueOf(config).Elem()

	var forks []uint64
	for i := 0; i < kind.NumField(); i++ {
		// Fetch the next field and skip non-fork rules
		field := kind.Field(i)
		if !strings.HasSuffix(field.Name, "Block") {
			continue
		}
		if field.Type != reflect.TypeOf(new(big.Int)) {
			continue
		}
		// Extract the fork rule block number and aggregate it
		rule := conf.Field(i).Interface().(*big.Int)
		if rule != nil {
			forks = append(forks, rule.Uint64())
		}
	}
	// Sort the fork block numbers to permit chronologival XOR
	for i := 0; i < len(forks); i++ {
		for j := i + 1; j < len(forks); j++ {
			if forks[i] > forks[j] {
				forks[i], forks[j] = forks[j], forks[i]
			}
		}
	}
	// Deduplicate block numbers applying multiple forks
	for i := 1; i < len(forks); i++ {
		if forks[i] == forks[i-1] {
			forks = append(forks[:i], forks[i+1:]...)
			i--
		}
	}
	// Skip any forks in block 0, that's the genesis ruleset
	if len(forks) > 0 && forks[0] == 0 {
		forks = forks[1:]
	}
	return forks
}
