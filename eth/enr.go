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

package eth

import (
	"encoding/binary"
	"errors"
	"fmt"
	"math"
	"math/big"
	"reflect"
	"strings"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/p2p/enr"
	"github.com/ethereum/go-ethereum/params"
)

var (
	// errENRGenesisMismatch is returned by the ENR validator if the local genesis
	// checksum doesn't match with one contained in a remote ENR record.
	errENRGenesisMismatch = errors.New("genesis mismatch")

	// errENRRemoteStale is returned by the ENR validator if a remote fork checksum
	// is a subset of our already applied forks, but the announced next fork block is
	// not on our already passed chain.
	errENRRemoteStale = errors.New("remote needs update")

	// errENRLocalStale is returned by the ENR validator if a remote fork checksum
	// does not match any local checksum variation, signalling that the two chains have
	// diverged in the past at some point.
	errENRLocalStale = errors.New("local needs update")
)

// ENR is the "eth" Ethereum Node Record, holding the genesis hash checksum,
// the enabled fork checksum and the next scheduled fork number.
type ENR [12]byte

// NewENR calculates the Ethereum network ENR from the chain config and head.
// https://eips.ethereum.org/EIPS/eip-2124
func NewENR(chain *core.BlockChain) ENR {
	return newENR(
		chain.Config(),
		chain.Genesis().Hash(),
		chain.CurrentHeader().Number.Uint64(),
	)
}

// newENR is the internal version of NewENR, which takes extracted values as its
// arguments instead of a chain. The reason is to allow testing the ENRs without
// having to simulate an entire blockchain.
func newENR(config *params.ChainConfig, genesis common.Hash, head uint64) ENR {
	// Calculate the fork checksum and the next fork block
	var (
		forkSum  uint32
		forkNext uint32
	)
	for _, fork := range gatherForks(config) {
		if fork <= head {
			forkSum ^= uint32(fork)
			continue
		}
		forkNext = uint32(fork)
		break
	}
	// Aggregate everything into a single binary blob
	var entry ENR

	binary.BigEndian.PutUint32(entry[0:], makeGenesisChecksum(genesis))
	binary.BigEndian.PutUint32(entry[4:], forkSum)
	binary.BigEndian.PutUint32(entry[8:], forkNext)

	return entry
}

// ENRKey implements enr.Entry, returning the key for the chain config.
func (e ENR) ENRKey() string { return "eth" }

// NewENRFilter creates an ENR filter that returns if a record should be rejected
// or not (may be rejected by another filter).
func NewENRFilter(chain *core.BlockChain) func(r *enr.Record) error {
	return newENRFilter(
		chain.Config(),
		chain.Genesis().Hash(),
		func() uint64 {
			return chain.CurrentHeader().Number.Uint64()
		},
	)
}

// newENRFilter is the internal version of NewENRFilter which takes closures as
// its arguments instead of a chain. The reason is to allow testing it without
// having to simulate an entire blockchain.
func newENRFilter(config *params.ChainConfig, genesis common.Hash, headfn func() uint64) func(r *enr.Record) error {
	// Calculate the genesis checksum and all the valid fork checksum and block combos
	var (
		gensum = makeGenesisChecksum(genesis)
		forks  = gatherForks(config)
		sums   = make([]uint32, len(forks)+1) // 0th is no-forks
	)
	for i, fork := range forks {
		sums[i+1] = sums[i] ^ uint32(fork)
	}
	// Add two sentries to simplify the fork checks and don't require special
	// casing the last one.
	forks = append(forks, math.MaxUint32)  // Last fork will never be passed
	sums = append(sums, sums[len(sums)-1]) // Last checksum is a noop

	// Create a validator that will filter out incompatible chains
	return func(r *enr.Record) error {
		// Retrieve the remote chain ENR entry, accept record if not found
		var entry ENR
		if err := r.Load(&entry); err != nil {
			return nil
		}
		// Cross reference the genesis checksum and reject on mismatch
		if binary.BigEndian.Uint32(entry[:4]) != gensum {
			return errENRGenesisMismatch
		}
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
			remoteSum  = binary.BigEndian.Uint32(entry[4:8])
			remoteNext = binary.BigEndian.Uint32(entry[8:12])
		)
		for i, fork := range forks {
			// If our head is beyond this fork, continue to the next (we have a dummy
			// fork of maxuint32 as the last item to always fail this check eventually).
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
					if uint32(forks[j]) != remoteNext {
						return errENRRemoteStale
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
			return errENRLocalStale
		}
		log.Error("Impossible eth ENR validation", "record", fmt.Sprintf("%x", entry))
		return nil // Something's very wrong, accept rather than reject
	}
}

// makeGenesisChecksum calculates the ENR checksum for the genesis block.
func makeGenesisChecksum(hash common.Hash) uint32 {
	var checksum uint32
	for i := 0; i < 8; i++ {
		for j := 0; j < 4; j++ {
			checksum ^= uint32(hash[i*4+j]) << uint32(24-8*j)
		}
	}
	return checksum
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
	// Deduplicate block number applying multiple forks and return
	for i := 1; i < len(forks); i++ {
		if forks[i] == forks[i-1] {
			forks = append(forks[:i], forks[i+1:]...)
			i--
		}
	}
	return forks
}
