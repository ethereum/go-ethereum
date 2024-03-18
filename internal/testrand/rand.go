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

package testrand

import (
	crand "crypto/rand"
	"encoding/binary"
	mrand "math/rand"

	"github.com/ethereum/go-ethereum/common"
)

// prng is a pseudo random number generator seeded by strong randomness.
// The randomness is printed on startup in order to make failures reproducible.
var prng = initRand()

func initRand() *mrand.Rand {
	var seed [8]byte
	crand.Read(seed[:])
	rnd := mrand.New(mrand.NewSource(int64(binary.LittleEndian.Uint64(seed[:]))))
	return rnd
}

// Bytes generates a random byte slice with specified length.
func Bytes(n int) []byte {
	r := make([]byte, n)
	prng.Read(r)
	return r
}

// Hash generates a random hash.
func Hash() common.Hash {
	return common.BytesToHash(Bytes(common.HashLength))
}

// Address generates a random address.
func Address() common.Address {
	return common.BytesToAddress(Bytes(common.AddressLength))
}
