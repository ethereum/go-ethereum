// Copyright 2015 The go-ethereum Authors
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

func (aw *AccessWitness) Merge(other *AccessWitness) {
	for k, mo := range other.Witness {
		if ma, ok := aw.Witness[k]; ok {
			// merge the two lists
			for b, y := range mo {
				ma[b] = y
			}
		} else {
			aw.Witness[k] = mo
		}
	}
}

func (aw *AccessWitness) Keys() [][]byte {
	var keys [][]byte
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
