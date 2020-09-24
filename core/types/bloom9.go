// Copyright 2014 The go-ethereum Authors
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

import (
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/crypto"
)

type bytesBacked interface {
	Bytes() []byte
}

const (
	// BloomByteLength represents the number of bytes used in a header log bloom.
	BloomByteLength = 256

	// BloomBitLength represents the number of bits used in a header log bloom.
	BloomBitLength = 8 * BloomByteLength
)

// Bloom represents a 2048 bit bloom filter.
type Bloom [BloomByteLength]byte

// BytesToBloom converts a byte slice to a bloom filter.
// It panics if b is not of suitable size.
func BytesToBloom(b []byte) Bloom {
	var bloom Bloom
	bloom.SetBytes(b)
	return bloom
}

// SetBytes sets the content of b to the given bytes.
// It panics if d is not of suitable size.
func (b *Bloom) SetBytes(d []byte) {
	if len(b) < len(d) {
		panic(fmt.Sprintf("bloom bytes too big %d %d", len(b), len(d)))
	}
	copy(b[BloomByteLength-len(d):], d)
}

// Add adds d to the filter. Future calls of Test(d) will return true.
func (b *Bloom) Add(d *big.Int) {
	b.SetBytes(or(b[:], bloom9(d.Bytes())))
}

// Big converts b to a big integer.
func (b Bloom) Big() *big.Int {
	return new(big.Int).SetBytes(b[:])
}

func (b Bloom) Bytes() []byte {
	return b[:]
}

func (b Bloom) Test(test *big.Int) bool {
	return BloomLookup(b, test)
}

func (b Bloom) TestBytes(test []byte) bool {
	return b.Test(new(big.Int).SetBytes(test))
}

// MarshalText encodes b as a hex string with 0x prefix.
func (b Bloom) MarshalText() ([]byte, error) {
	return hexutil.Bytes(b[:]).MarshalText()
}

// UnmarshalText b as a hex string with 0x prefix.
func (b *Bloom) UnmarshalText(input []byte) error {
	return hexutil.UnmarshalFixedText("Bloom", input, b[:])
}

func CreateBloom(receipts Receipts) Bloom {
	bin := make([]byte, BloomByteLength)
	for _, receipt := range receipts {
		bin = or(bin, LogsBloom(receipt.Logs))
	}

	return BytesToBloom(bin)
}

func LogsBloom(logs []*Log) []byte {
	bin := make([]byte, BloomByteLength)
	for _, log := range logs {
		bin = or(bin, bloom9(log.Address.Bytes()))
		for _, b := range log.Topics {
			bin = or(bin, bloom9(b[:]))
		}
	}

	return bin
}

func bloom9(data []byte) []byte {
	hash := make([]byte, 32)
	sha := hasherPool.Get().(crypto.KeccakState)
	defer hasherPool.Put(sha)
	sha.Reset()
	sha.Write(data)
	sha.Read(hash)
	r := make([]byte, BloomByteLength)

	for i := 0; i < 6; i += 2 {
		b := (uint(hash[i+1]) + (uint(hash[i]) << 8)) & 2047
		byteIdx := b >> 3             // divide by 8 bit per byte
		bitMask := byte(1 << (b % 8)) // set the b%8th's bit
		r[BloomByteLength-byteIdx-1] |= bitMask
	}
	return r
}

func or(a, b []byte) []byte {
	for i := range a {
		a[i] |= b[i]
	}
	return a
}

var Bloom9 = bloom9

func BloomLookup(bin Bloom, topic bytesBacked) bool {
	cmp := bloom9(topic.Bytes())
	for i := range cmp {
		if cmp[i] != cmp[i]&bin[i] {
			// Every topic in cmp has to be in bin
			return false
		}
	}
	return true
}
