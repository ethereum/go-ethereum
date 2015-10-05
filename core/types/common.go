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
	"math/big"

	"fmt"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/vm"
)

type BlockProcessor interface {
	Process(*Block) (vm.Logs, Receipts, error)
}

const bloomLength = 256

type Bloom [bloomLength]byte

func BytesToBloom(b []byte) Bloom {
	var bloom Bloom
	bloom.SetBytes(b)
	return bloom
}

func (b *Bloom) SetBytes(d []byte) {
	if len(b) < len(d) {
		panic(fmt.Sprintf("bloom bytes too big %d %d", len(b), len(d)))
	}

	copy(b[bloomLength-len(d):], d)
}

func (b Bloom) Big() *big.Int {
	return common.Bytes2Big(b[:])
}

func (b Bloom) Bytes() []byte {
	return b[:]
}
