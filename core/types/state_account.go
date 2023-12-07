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

import (
	"bytes"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/rlp"
)

//go:generate go run ../../rlp/rlpgen -type StateAccount -out gen_account_rlp.go

// StateAccount is the Ethereum consensus representation of accounts.
// These objects are stored in the main account trie.
type StateAccount struct {
	Nonce            uint64
	Balance          *big.Int
	Root             common.Hash // merkle root of the storage trie
	KeccakCodeHash   []byte
	PoseidonCodeHash []byte
	CodeSize         uint64
}

// NewEmptyStateAccount constructs an empty state account.
func NewEmptyStateAccount() *StateAccount {
	return &StateAccount{
		Balance:          new(big.Int),
		Root:             EmptyRootHash,
		KeccakCodeHash:   EmptyKeccakCodeHash.Bytes(),
		PoseidonCodeHash: EmptyPoseidonCodeHash.Bytes(),
		CodeSize:         0,
	}
}

// Copy returns a deep-copied state account object.
func (acct *StateAccount) Copy() *StateAccount {
	var balance *big.Int
	if acct.Balance != nil {
		balance = new(big.Int).Set(acct.Balance)
	}
	return &StateAccount{
		Nonce:            acct.Nonce,
		Balance:          balance,
		Root:             acct.Root,
		KeccakCodeHash:   common.CopyBytes(acct.KeccakCodeHash),
		PoseidonCodeHash: common.CopyBytes(acct.PoseidonCodeHash),
		CodeSize:         acct.CodeSize,
	}
}

// SlimAccount is a modified version of an Account, where the root is replaced
// with a byte slice. This format can be used to represent full-consensus format
// or slim format which replaces the empty root and code hash as nil byte slice.
type SlimAccount struct {
	Nonce            uint64
	Balance          *big.Int
	Root             []byte // Nil if root equals to types.EmptyRootHash
	KeccakCodeHash   []byte // Nil if hash equals to types.EmptyKeccakCodeHash
	PoseidonCodeHash []byte // Nil if hash equals to types.EmptyPoseidonCodeHash
	CodeSize         uint64
}

// SlimAccountRLP encodes the state account in 'slim RLP' format.
func SlimAccountRLP(account StateAccount) []byte {
	slim := SlimAccount{
		Nonce:   account.Nonce,
		Balance: account.Balance,
	}
	if account.Root != EmptyRootHash {
		slim.Root = account.Root[:]
	}
	if !bytes.Equal(account.KeccakCodeHash, EmptyKeccakCodeHash[:]) {
		slim.KeccakCodeHash = account.KeccakCodeHash
		slim.PoseidonCodeHash = account.PoseidonCodeHash
		slim.CodeSize = account.CodeSize
	}
	data, err := rlp.EncodeToBytes(slim)
	if err != nil {
		panic(err)
	}
	return data
}

// FullAccount decodes the data on the 'slim RLP' format and returns
// the consensus format account.
func FullAccount(data []byte) (*StateAccount, error) {
	var slim SlimAccount
	if err := rlp.DecodeBytes(data, &slim); err != nil {
		return nil, err
	}
	var account StateAccount
	account.Nonce, account.Balance = slim.Nonce, slim.Balance

	// Interpret the storage root and code hash in slim format.
	if len(slim.Root) == 0 {
		account.Root = EmptyRootHash
	} else {
		account.Root = common.BytesToHash(slim.Root)
	}
	if len(slim.KeccakCodeHash) == 0 {
		account.KeccakCodeHash = EmptyKeccakCodeHash[:]
		account.PoseidonCodeHash = EmptyPoseidonCodeHash[:]
		account.CodeSize = 0
	} else {
		account.KeccakCodeHash = slim.KeccakCodeHash
		account.PoseidonCodeHash = slim.PoseidonCodeHash
		account.CodeSize = slim.CodeSize
	}
	return &account, nil
}

// FullAccountRLP converts data on the 'slim RLP' format into the full RLP-format.
func FullAccountRLP(data []byte) ([]byte, error) {
	account, err := FullAccount(data)
	if err != nil {
		return nil, err
	}
	return rlp.EncodeToBytes(account)
}
