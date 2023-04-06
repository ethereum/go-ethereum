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

package light

import (
	"math/rand"
	"testing"

	"github.com/ethereum/go-ethereum/beacon/light/types"
	"github.com/ethereum/go-ethereum/beacon/merkle"
	"github.com/ethereum/go-ethereum/common"
)

func makeChain(tail types.Header, headSlot uint64, format merkle.ProofFormat) (headers []types.Header, stateProofs []merkle.MultiProof) {
	valueCount := merkle.ValueCount(format)
	for tail.Slot < headSlot {
		var (
			slot       uint64
			parentRoot common.Hash
		)
		if tail != (types.Header{}) {
			slot = tail.Slot + 1
			parentRoot = tail.Hash()
		}
		for slot < headSlot && rand.Intn(5) == 0 {
			slot++
		}
		stateProof := merkle.MultiProof{
			Format: format,
			Values: make(merkle.Values, valueCount),
		}
		for i, _ := range stateProof.Values {
			stateProof.Values[i] = merkle.Value(randomHash())
		}
		header := types.Header{
			Slot:          slot,
			ProposerIndex: uint64(rand.Intn(10000)),
			BodyRoot:      randomHash(),
			StateRoot:     stateProof.RootHash(),
			ParentRoot:    parentRoot,
		}

		headers = append(headers, header)
		stateProofs = append(stateProofs, stateProof)
		tail = header
	}
}

func randomHash() (hash common.Hash) {
	rand.Read(hash[:])
	return
}
