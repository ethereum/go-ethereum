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

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/crypto"
)

type bytesBacked interface {
	Bytes() []byte
}

const bloomLength = 256

// Bloom represents a 256 bit bloom filter.
type Bloom [bloomLength]byte

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
	copy(b[bloomLength-len(d):], d)
}

// Add adds d to the filter. Future calls of Test(d) will return true.
func (b *Bloom) Add(d *big.Int) {
	bin := new(big.Int).SetBytes(b[:])
	bin.Or(bin, bloom9(d.Bytes()))
	b.SetBytes(bin.Bytes())
}

// Big converts b to a big integer.
func (b Bloom) Big() *big.Int {
	return common.Bytes2Big(b[:])
}

func (b Bloom) Bytes() []byte {
	return b[:]
}

func (b Bloom) Test(test *big.Int) bool {
	return BloomLookup(b, test)
}

func (b Bloom) TestBytes(test []byte) bool {
	return b.Test(common.BytesToBig(test))
}

// MarshalJSON encodes b as a hex string with 0x prefix.
func (b Bloom) MarshalJSON() ([]byte, error) {
	return hexutil.Bytes(b[:]).MarshalJSON()
}

// UnmarshalJSON b as a hex string with 0x prefix.
func (b *Bloom) UnmarshalJSON(input []byte) error {
	return hexutil.UnmarshalJSON("Bloom", input, b[:])
}

func CreateBloom(receipts Receipts) Bloom {
	bin := new(big.Int)
	for _, receipt := range receipts {
		bin.Or(bin, LogsBloom(receipt.Logs))
	}

	return BytesToBloom(bin.Bytes())
}

func LogsBloom(logs []*Log) *big.Int {
	bin := new(big.Int)
	for _, log := range logs {
		bin.Or(bin, bloom9(log.Address.Bytes()))
		for _, b := range log.Topics {
			bin.Or(bin, bloom9(b[:]))
		}
	}

	return bin
}

func bloom9(b []byte) *big.Int {
	b = crypto.Keccak256(b[:])

	r := new(big.Int)

	for i := 0; i < 6; i += 2 {
		t := big.NewInt(1)
		b := (uint(b[i+1]) + (uint(b[i]) << 8)) & 2047
		r.Or(r, t.Lsh(t, b))
	}

	return r
}

var Bloom9 = bloom9

func BloomLookup(bin Bloom, topic bytesBacked) bool {
	bloom := bin.Big()
	cmp := bloom9(topic.Bytes()[:])

	return bloom.And(bloom, cmp).Cmp(cmp) == 0
}
