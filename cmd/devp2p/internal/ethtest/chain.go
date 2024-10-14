// Copyright 2020 The go-ethereum Authors
// This file is part of go-ethereum.
//
// go-ethereum is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// go-ethereum is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU General Public License for more details.
//
// You should have received a copy of the GNU General Public License
// along with go-ethereum. If not, see <http://www.gnu.org/licenses/>.

package ethtest

import (
	"bytes"
	"compress/gzip"
	"crypto/ecdsa"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math/big"
	"os"
	"path/filepath"
	"slices"
	"sort"
	"strings"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/forkid"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/eth/protocols/eth"
	"github.com/ethereum/go-ethereum/params"
	"github.com/ethereum/go-ethereum/rlp"
)

// Chain is a lightweight blockchain-like store which can read a hivechain
// created chain.
type Chain struct {
	genesis core.Genesis
	blocks  []*types.Block
	state   map[common.Address]state.DumpAccount // state of head block
	senders map[common.Address]*senderInfo
	config  *params.ChainConfig
}

// NewChain takes the given chain.rlp file, and decodes and returns
// the blocks from the file.
func NewChain(dir string) (*Chain, error) {
	gen, err := loadGenesis(filepath.Join(dir, "genesis.json"))
	if err != nil {
		return nil, err
	}
	gblock := gen.ToBlock()

	blocks, err := blocksFromFile(filepath.Join(dir, "chain.rlp"), gblock)
	if err != nil {
		return nil, err
	}
	state, err := readState(filepath.Join(dir, "headstate.json"))
	if err != nil {
		return nil, err
	}
	accounts, err := readAccounts(filepath.Join(dir, "accounts.json"))
	if err != nil {
		return nil, err
	}
	return &Chain{
		genesis: gen,
		blocks:  blocks,
		state:   state,
		senders: accounts,
		config:  gen.Config,
	}, nil
}

// senderInfo is an account record as output in the "accounts.json" file from
// hivechain.
type senderInfo struct {
	Key   *ecdsa.PrivateKey `json:"key"`
	Nonce uint64            `json:"nonce"`
}

// Head returns the chain head.
func (c *Chain) Head() *types.Block {
	return c.blocks[c.Len()-1]
}

// AccountsInHashOrder returns all accounts of the head state, ordered by hash of address.
func (c *Chain) AccountsInHashOrder() []state.DumpAccount {
	list := make([]state.DumpAccount, len(c.state))
	i := 0
	for addr, acc := range c.state {
		list[i] = acc
		list[i].Address = &addr
		if len(acc.AddressHash) != 32 {
			panic(fmt.Errorf("missing/invalid SecureKey in dump account %v", addr))
		}
		i++
	}
	slices.SortFunc(list, func(x, y state.DumpAccount) int {
		return bytes.Compare(x.AddressHash, y.AddressHash)
	})
	return list
}

// CodeHashes returns all bytecode hashes contained in the head state.
func (c *Chain) CodeHashes() []common.Hash {
	var hashes []common.Hash
	seen := make(map[common.Hash]struct{})
	seen[types.EmptyCodeHash] = struct{}{}
	for _, acc := range c.state {
		h := common.BytesToHash(acc.CodeHash)
		if _, ok := seen[h]; ok {
			continue
		}
		hashes = append(hashes, h)
		seen[h] = struct{}{}
	}
	slices.SortFunc(hashes, (common.Hash).Cmp)
	return hashes
}

// Len returns the length of the chain.
func (c *Chain) Len() int {
	return len(c.blocks)
}

// ForkID gets the fork id of the chain.
func (c *Chain) ForkID() forkid.ID {
	return forkid.NewID(c.config, c.blocks[0], uint64(c.Len()), c.blocks[c.Len()-1].Time())
}

// TD calculates the total difficulty of the chain at the
// chain head.
func (c *Chain) TD() *big.Int {
	sum := new(big.Int)
	for _, block := range c.blocks[:c.Len()] {
		sum.Add(sum, block.Difficulty())
	}
	return sum
}

// GetBlock returns the block at the specified number.
func (c *Chain) GetBlock(number int) *types.Block {
	return c.blocks[number]
}

// RootAt returns the state root for the block at the given height.
func (c *Chain) RootAt(height int) common.Hash {
	if height < c.Len() {
		return c.blocks[height].Root()
	}
	return common.Hash{}
}

// GetSender returns the address associated with account at the index in the
// pre-funded accounts list.
func (c *Chain) GetSender(idx int) (common.Address, uint64) {
	var accounts Addresses
	for addr := range c.senders {
		accounts = append(accounts, addr)
	}
	sort.Sort(accounts)
	addr := accounts[idx]
	return addr, c.senders[addr].Nonce
}

// IncNonce increases the specified signing account's pending nonce.
func (c *Chain) IncNonce(addr common.Address, amt uint64) {
	if _, ok := c.senders[addr]; !ok {
		panic("nonce increment for non-signer")
	}
	c.senders[addr].Nonce += amt
}

// Balance returns the balance of an account at the head of the chain.
func (c *Chain) Balance(addr common.Address) *big.Int {
	bal := new(big.Int)
	if acc, ok := c.state[addr]; ok {
		bal, _ = bal.SetString(acc.Balance, 10)
	}
	return bal
}

// SignTx signs a transaction for the specified from account, so long as that
// account was in the hivechain accounts dump.
func (c *Chain) SignTx(from common.Address, tx *types.Transaction) (*types.Transaction, error) {
	signer := types.LatestSigner(c.config)
	acc, ok := c.senders[from]
	if !ok {
		return nil, fmt.Errorf("account not available for signing: %s", from)
	}
	return types.SignTx(tx, signer, acc.Key)
}

// GetHeaders returns the headers base on an ethGetPacketHeadersPacket.
func (c *Chain) GetHeaders(req *eth.GetBlockHeadersPacket) ([]*types.Header, error) {
	if req.Amount < 1 {
		return nil, errors.New("no block headers requested")
	}
	var (
		headers     = make([]*types.Header, req.Amount)
		blockNumber uint64
	)
	// Range over blocks to check if our chain has the requested header.
	for _, block := range c.blocks {
		if block.Hash() == req.Origin.Hash || block.Number().Uint64() == req.Origin.Number {
			headers[0] = block.Header()
			blockNumber = block.Number().Uint64()
		}
	}
	if headers[0] == nil {
		return nil, fmt.Errorf("no headers found for given origin number %v, hash %v", req.Origin.Number, req.Origin.Hash)
	}
	if req.Reverse {
		for i := 1; i < int(req.Amount); i++ {
			blockNumber -= (1 - req.Skip)
			headers[i] = c.blocks[blockNumber].Header()
		}
		return headers, nil
	}
	for i := 1; i < int(req.Amount); i++ {
		blockNumber += (1 + req.Skip)
		headers[i] = c.blocks[blockNumber].Header()
	}
	return headers, nil
}

// Shorten returns a copy chain of a desired height from the imported
func (c *Chain) Shorten(height int) *Chain {
	blocks := make([]*types.Block, height)
	copy(blocks, c.blocks[:height])

	config := *c.config
	return &Chain{
		blocks: blocks,
		config: &config,
	}
}

func loadGenesis(genesisFile string) (core.Genesis, error) {
	chainConfig, err := os.ReadFile(genesisFile)
	if err != nil {
		return core.Genesis{}, err
	}
	var gen core.Genesis
	if err := json.Unmarshal(chainConfig, &gen); err != nil {
		return core.Genesis{}, err
	}
	return gen, nil
}

type Addresses []common.Address

func (a Addresses) Len() int {
	return len(a)
}

func (a Addresses) Less(i, j int) bool {
	return bytes.Compare(a[i][:], a[j][:]) < 0
}

func (a Addresses) Swap(i, j int) {
	tmp := a[i]
	a[i] = a[j]
	a[j] = tmp
}

func blocksFromFile(chainfile string, gblock *types.Block) ([]*types.Block, error) {
	// Load chain.rlp.
	fh, err := os.Open(chainfile)
	if err != nil {
		return nil, err
	}
	defer fh.Close()
	var reader io.Reader = fh
	if strings.HasSuffix(chainfile, ".gz") {
		if reader, err = gzip.NewReader(reader); err != nil {
			return nil, err
		}
	}
	stream := rlp.NewStream(reader, 0)
	var blocks = make([]*types.Block, 1)
	blocks[0] = gblock
	for i := 0; ; i++ {
		var b types.Block
		if err := stream.Decode(&b); err == io.EOF {
			break
		} else if err != nil {
			return nil, fmt.Errorf("at block index %d: %v", i, err)
		}
		if b.NumberU64() != uint64(i+1) {
			return nil, fmt.Errorf("block at index %d has wrong number %d", i, b.NumberU64())
		}
		blocks = append(blocks, &b)
	}
	return blocks, nil
}

func readState(file string) (map[common.Address]state.DumpAccount, error) {
	f, err := os.ReadFile(file)
	if err != nil {
		return nil, fmt.Errorf("unable to read state: %v", err)
	}
	var dump state.Dump
	if err := json.Unmarshal(f, &dump); err != nil {
		return nil, fmt.Errorf("unable to unmarshal state: %v", err)
	}

	state := make(map[common.Address]state.DumpAccount)
	for key, acct := range dump.Accounts {
		var addr common.Address
		if err := addr.UnmarshalText([]byte(key)); err != nil {
			return nil, fmt.Errorf("invalid address %q", key)
		}
		state[addr] = acct
	}
	return state, nil
}

func readAccounts(file string) (map[common.Address]*senderInfo, error) {
	f, err := os.ReadFile(file)
	if err != nil {
		return nil, fmt.Errorf("unable to read state: %v", err)
	}
	type account struct {
		Key hexutil.Bytes `json:"key"`
	}
	keys := make(map[common.Address]account)
	if err := json.Unmarshal(f, &keys); err != nil {
		return nil, fmt.Errorf("unable to unmarshal accounts: %v", err)
	}
	accounts := make(map[common.Address]*senderInfo)
	for addr, acc := range keys {
		pk, err := crypto.HexToECDSA(common.Bytes2Hex(acc.Key))
		if err != nil {
			return nil, fmt.Errorf("unable to read private key for %s: %v", err, addr)
		}
		accounts[addr] = &senderInfo{Key: pk, Nonce: 0}
	}
	return accounts, nil
}
