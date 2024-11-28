// Copyright 2024 The go-ethereum Authors
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

package internal

import (
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/state/snapshot"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/plugins"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/holiman/uint256"
)

var _ = plugins.Chain(Chain{})

type Chain struct {
	chain *core.BlockChain
}

func (c Chain) Head() (uint64, uint64) {
	return c.chain.CurrentHeader().Number.Uint64(), c.chain.CurrentFinalBlock().Number.Uint64()
}

func (c Chain) Header(number uint64) *types.Header {
	hash := c.chain.GetCanonicalHash(number)
	return c.chain.GetHeaderByHash(hash)
}

func (c Chain) Block(number uint64) *types.Block {
	hash := c.chain.GetCanonicalHash(number)
	return c.chain.GetBlockByHash(hash)
}

func (c Chain) Receipts(number uint64) types.Receipts {
	hash := c.chain.GetCanonicalHash(number)
	return c.chain.GetReceiptsByHash(hash)
}

func (c Chain) State(root common.Hash) plugins.State {
	reader, err := c.chain.StateCache().Reader(root)
	if err != nil {
		return nil
	}

	return State{
		root:   root,
		cache:  c.chain.StateCache(),
		reader: reader,
	}
}

type State struct {
	root   common.Hash
	cache  state.Database
	reader state.Reader
}

func (s State) Account(addr common.Address) plugins.Account {
	hash := crypto.Keccak256Hash(addr.Bytes())
	reader, err := s.cache.Reader(s.root)
	if err != nil {
		return nil
	}
	account, err := reader.Account(addr)
	if err != nil {
		return nil
	}
	return Account{
		root:    s.root,
		hash:    hash,
		account: account,
		cache:   s.cache,
	}
}

func (s State) AccountIterator(seek common.Hash) snapshot.AccountIterator {
	if it, err := s.cache.Snapshot().AccountIterator(s.root, seek); err == nil {
		return it
	}
	return nil
}

func (s State) NewAccount(addr common.Address, accRLP []byte) plugins.Account {
	hash := crypto.Keccak256Hash(addr.Bytes())
	var slim *types.SlimAccount
	if err := rlp.DecodeBytes(accRLP, &slim); err != nil {
		return nil
	}
	account := &types.StateAccount{
		Nonce:    slim.Nonce,
		Balance:  slim.Balance,
		CodeHash: slim.CodeHash,
		Root:     common.BytesToHash(slim.Root),
	}
	if len(account.CodeHash) == 0 {
		account.CodeHash = types.EmptyCodeHash.Bytes()
	}
	if account.Root == (common.Hash{}) {
		account.Root = types.EmptyRootHash
	}
	return Account{
		root:    s.root,
		hash:    hash,
		account: account,
		cache:   s.cache,
	}
}

type Account struct {
	addr    common.Address
	root    common.Hash
	hash    common.Hash
	account *types.StateAccount
	cache   state.Database
}

func (a Account) Balance() *uint256.Int {
	return a.account.Balance
}

func (a Account) Nonce() uint64 {
	return a.account.Nonce
}

func (a Account) Code() []byte {
	if code, err := a.cache.ContractCode(a.addr, a.account.Root); err == nil {
		return code
	}
	return nil
}

func (a Account) Storage(slot common.Hash) common.Hash {
	reader, err := a.cache.Reader(a.root)
	if err != nil {
		return common.Hash{}
	}
	if storage, err := reader.Storage(a.addr, slot); err == nil {
		return storage
	}
	return common.Hash{}
}

func (a Account) StorageIterator(seek common.Hash) snapshot.StorageIterator {
	if it, err := a.cache.Snapshot().StorageIterator(a.root, a.hash, seek); err == nil {
		return it
	}
	return nil
}
