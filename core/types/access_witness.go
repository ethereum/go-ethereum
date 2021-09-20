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

package types

import "github.com/ethereum/go-ethereum/common"
import "github.com/ethereum/go-ethereum/params"

// AccessWitness lists the locations of the state that are being accessed
// during the production of a block.
// TODO(@gballet) this doesn't support deletions
type AccessWitness struct {
	Witness map[common.Hash]map[byte]struct{}
}

func NewAccessWitness() *AccessWitness {
	return &AccessWitness{Witness: make(map[common.Hash]map[byte]struct{})}
}

// TouchAddress adds any missing addr to the witness and returns respectively
// true if the stem or the stub weren't arleady present.
func (aw *AccessWitness) TouchAddress(addr []byte) (bool, bool) {
	var (
		stem        common.Hash
		newStem     bool
		newSelector bool
		selector    = addr[31]
	)
	copy(stem[:], addr[:31])

	// Check for the presence of the stem
	if _, newStem := aw.Witness[stem]; !newStem {
		aw.Witness[stem] = make(map[byte]struct{})
	}

	// Check for the presence of the selector
	if _, newSelector := aw.Witness[stem][selector]; !newSelector {
		aw.Witness[stem][selector] = struct{}{}
	}

	return newStem, newSelector
}

// TouchAddressAndChargeGas checks if a location has already been touched in
// the current witness, and charge extra gas if that isn't the case. This is
// meant to only be called on a tx-context access witness (i.e. before it is
// merged), not a block-context witness: witness costs are charged per tx.
func (aw *AccessWitness) TouchAddressAndChargeGas(addr []byte) uint64 {
	var gas uint64

	nstem, nsel := aw.TouchAddress(addr)
	if nstem {
		gas += params.WitnessBranchCost
	}
	if nsel {
		gas += params.WitnessChunkCost
	}
	return gas
}

// Merge is used to merge the witness that got generated during the execution
// of a tx, with the accumulation of witnesses that were generated during the
// execution of all the txs preceding this one in a given block.
func (aw *AccessWitness) Merge(other *AccessWitness) {
	for k, mo := range other.Witness {
		// LeafNode-level merge
		if ma, ok := aw.Witness[k]; ok {
			for b, y := range mo {
				// If a particular location isn't already
				// present, then flag it. The block witness
				// require only the initial value be present.
				if _, ok := ma[b]; !ok {
					ma[b] = y
				}
			}
		} else {
			aw.Witness[k] = mo
		}
	}
}

// Key returns, predictably, the list of keys that were touched during the
// buildup of the access witness.
func (aw *AccessWitness) Keys() [][]byte {
	keys := make([][]byte, 0, len(aw.Witness))
	for stem, branches := range aw.Witness {
		for selector := range branches {
			var key [32]byte
			copy(key[:31], stem[:31])
			key[31] = selector
			keys = append(keys, key[:])
		}
	}
	return keys
}
