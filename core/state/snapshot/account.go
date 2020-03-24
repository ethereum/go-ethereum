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

package snapshot

import (
	"bytes"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/rlp"
	"golang.org/x/crypto/sha3"
)

// Account is a slim version of a state.Account, where the root and code hash
// are replaced with a nil byte slice for empty accounts.
type Account struct {
	Nonce    uint64
	Balance  *big.Int
	Root     []byte
	CodeHash []byte
}

// AccountRLP converts a state.Account content into a slim snapshot version RLP
// encoded.
func AccountRLP(nonce uint64, balance *big.Int, root common.Hash, codehash []byte) []byte {
	slim := Account{
		Nonce:   nonce,
		Balance: balance,
	}
	if root != emptyRoot {
		slim.Root = root[:]
	}
	if !bytes.Equal(codehash, emptyCode[:]) {
		slim.CodeHash = codehash
	}
	data, err := rlp.EncodeToBytes(slim)
	if err != nil {
		panic(err)
	}
	return data
}

func SlimToFull(data []byte) []byte {
	acc := &Account{}
	rlp.DecodeBytes(data, acc)
	if len(acc.Root) == 0 {
		acc.Root = emptyRoot[:]
	}
	if len(acc.CodeHash) == 0 {
		acc.CodeHash = emptyCode[:]
	}
	fullData, err := rlp.EncodeToBytes(acc)
	if err != nil {
		panic(err)
	}
	return fullData
}

// conversionAccount is used for converting between full and slim format. When
// doing this, we can consider 'balance' as a byte array, as it has already
// been converted from big.Int into an rlp-byteslice.
type conversionAccount struct {
	Nonce    uint64
	Balance  []byte
	Root     []byte
	CodeHash []byte
}

type converter struct {
	tmpAcc *conversionAccount
	sha3   crypto.KeccakState
	stream rlp.Stream
}

func newConverter() *converter {
	return &converter{
		tmpAcc: &conversionAccount{},
		sha3:   sha3.NewLegacyKeccak256().(crypto.KeccakState),
	}
}

func (c *converter) SlimToHash(data []byte) common.Hash {
	var (
		result common.Hash
		tmp    = c.tmpAcc
		sha3   = c.sha3
	)
	c.stream.Reset(bytes.NewReader(data), 0)
	c.stream.Decode(c.tmpAcc)
	if len(tmp.Root) == 0 {
		tmp.Root = emptyRoot[:]
	}
	if len(tmp.CodeHash) == 0 {
		tmp.CodeHash = emptyCode[:]
	}
	sha3.Reset()
	_ = rlp.Encode(sha3, tmp)
	sha3.Read(result[:])
	return result
}

// SlimToHash produces a hash of a main account trie, where the input is the
// 'slim' version
func SlimToHash(data []byte, sha3 crypto.KeccakState) common.Hash {
	tmp := &conversionAccount{}
	var result common.Hash
	rlp.DecodeBytes(data, tmp)
	if len(tmp.Root) == 0 {
		tmp.Root = emptyRoot[:]
	}
	if len(tmp.CodeHash) == 0 {
		tmp.CodeHash = emptyCode[:]
	}
	sha3.Reset()
	_ = rlp.Encode(sha3, tmp)
	sha3.Read(result[:])
	return result
}
