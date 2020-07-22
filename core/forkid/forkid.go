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
	"hash/crc32"
	"math"
	"math/big"
	"reflect"
	"strings"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
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

// Blockchain defines all necessary method to build a forkID.
type Blockchain interface {
	// Config retrieves the chain's fork configuration.
	Config() *params.ChainConfig

	// Genesis retrieves the chain's genesis block.
	Genesis() *types.Block

	// CurrentHeader retrieves the current head header of the canonical chain.
	CurrentHeader() *types.Header
}

// ID is a fork identifier as defined by EIP-2124.
type ID struct {
	Hash [4]byte // CRC32 checksum of the genesis block and passed fork block numbers
	Next uint64  // Block number of the next upcoming fork, or 0 if no forks are known
}

// Filter is a fork id filter to validate a remotely advertised ID.
type Filter func(id ID) error

// NewID calculates the Ethereum fork ID from the chain config and head.
func NewID(chain Blockchain) ID {
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
	hash := crc32.ChecksumIEEE(genesis[:])

	// Calculate the current fork checksum and the next fork block
	var next uint64
	for _, fork := range gatherForks(config) {
		if fork <= head {
			// Fork already passed, checksum the previous hash and the fork number
			hash = checksumUpdate(hash, fork)
			continue
		}
		next = fork
		break
	}
	return ID{Hash: checksumToBytes(hash), Next: next}
}

// NewFilter creates a filter that returns if a fork ID should be rejected or not
// based on the local chain's status.
func NewFilter(chain Blockchain) Filter {
	return newFilter(
		chain.Config(),
		chain.Genesis().Hash(),
		func() uint64 {
			return chain.CurrentHeader().Number.Uint64()
		},
	)
}

// NewStaticFilter creates a filter at block zero.
func NewStaticFilter(config *params.ChainConfig, genesis common.Hash) Filter {
	head := func() uint64 { return 0 }
	return newFilter(config, genesis, head)
}

// newFilter is the internal version of NewFilter, taking closures as its arguments
// instead of a chain. The reason is to allow testing it without having to simulate
// an entire blockchain.
func newFilter(config *params.ChainConfig, genesis common.Hash, headfn func() uint64) Filter {
	// Calculate the all the valid fork hash and fork next combos
	var (
		forks = gatherForks(config)
		sums  = make([][4]byte, len(forks)+1) // 0th is the genesis
	)
	hash := crc32.ChecksumIEEE(genesis[:])
	sums[0] = checksumToBytes(hash)
	for i, fork := range forks {
		hash = checksumUpdate(hash, fork)
		sums[i+1] = checksumToBytes(hash)
	}
	// Add two sentries to simplify the fork checks and don't require special
	// casing the last one.
	forks = append(forks, math.MaxUint64) // Last fork will never be passed

	// Create a validator that will filter out incompatible chains
	return func(id ID) error {
		// Run the fork checksum validation ruleset:
		//   1. If local and remote FORK_CSUM matches, compare local head to FORK_NEXT.
		//        The two nodes are in the same fork state currently. They might know
		//        of differing future forks, but that's not relevant until the fork
		//        triggers (might be postponed, nodes might be updated to match).
		//      1a. A remotely announced but remotely not passed block is already passed
		//          locally, disconnect, since the chains are incompatible.
		//      1b. No remotely announced fork; or not yet passed locally, connect.
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
		head := headfn()
		for i, fork := range forks {
			// If our head is beyond this fork, continue to the next (we have a dummy
			// fork of maxuint64 as the last item to always fail this check eventually).
			if head > fork {
				continue
			}
			// Found the first unpassed fork block, check if our current state matches
			// the remote checksum (rule #1).
			if sums[i] == id.Hash {
				// Fork checksum matched, check if a remote future fork block already passed
				// locally without the local node being aware of it (rule #1a).
				if id.Next > 0 && head >= id.Next {
					return ErrLocalIncompatibleOrStale
				}
				// Haven't passed locally a remote-only fork, accept the connection (rule #1b).
				return nil
			}
			// The local and remote nodes are in different forks currently, check if the
			// remote checksum is a subset of our local forks (rule #2).
			for j := 0; j < i; j++ {
				if sums[j] == id.Hash {
					// Remote checksum is a subset, validate based on the announced next fork
					if forks[j] != id.Next {
						return ErrRemoteStale
					}
					return nil
				}
			}
			// Remote chain is not a subset of our local one, check if it's a superset by
			// any chance, signalling that we're simply out of sync (rule #3).
			for j := i + 1; j < len(sums); j++ {
				if sums[j] == id.Hash {
					// Yay, remote checksum is a superset, ignore upcoming forks
					return nil
				}
			}
			// No exact, subset or superset match. We are on differing chains, reject.
			return ErrLocalIncompatibleOrStale
		}
		log.Error("Impossible fork ID validation", "id", id)
		return nil // Something's very wrong, accept rather than reject
	}
}

// checksumUpdate calculates the next IEEE CRC32 checksum based on the previous
// one and a fork block number (equivalent to CRC32(original-blob || fork)).
func checksumUpdate(hash uint32, fork uint64) uint32 {
	var blob [8]byte
	binary.BigEndian.PutUint64(blob[:], fork)
	return crc32.Update(hash, crc32.IEEETable, blob[:])
}

// checksumToBytes converts a uint32 checksum into a [4]byte array.
func checksumToBytes(hash uint32) [4]byte {
	var blob [4]byte
	binary.BigEndian.PutUint32(blob[:], hash)
	return blob
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
	// Sort the fork block numbers to permit chronological XOR
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
