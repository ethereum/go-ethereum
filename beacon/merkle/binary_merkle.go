// Copyright 2022 The go-ethereum Authors
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

package merkle

import (
	"reflect"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/minio/sha256-simd"
)

// Value represents either a 32 byte value or hash node in a binary merkle tree/partial proof
type (
	Value  [32]byte
	Values []Value
)

var ValueT = reflect.TypeOf(Value{})

// UnmarshalJSON parses a merkle value in hex syntax.
func (m *Value) UnmarshalJSON(input []byte) error {
	return hexutil.UnmarshalFixedJSON(ValueT, input, m[:])
}

// VerifySingleProof verifies a Merkle proof branch for a single value in a
// binary Merkle tree (index is a generalized tree index).
func VerifySingleProof(proof Values, index uint64, value Value) (common.Hash, bool) {
	hasher := sha256.New()
	for _, proofHash := range proof {
		hasher.Reset()
		if index&1 == 0 {
			hasher.Write(value[:])
			hasher.Write(proofHash[:])
		} else {
			hasher.Write(proofHash[:])
			hasher.Write(value[:])
		}
		hasher.Sum(value[:0])
		index /= 2
		if index == 0 {
			return common.Hash{}, false
		}
	}
	if index != 1 {
		return common.Hash{}, false
	}
	return common.Hash(value), true
}
