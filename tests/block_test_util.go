// Copyright 2015 The go-ethereum Authors
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

// Package tests implements execution of Ethereum JSON tests.
package tests

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/common/math"
	"github.com/ethereum/go-ethereum/consensus/ethash"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/params"
	"github.com/ethereum/go-ethereum/rlp"
)

// A BlockTest checks handling of entire blocks.
type BlockTest struct {
	json btJSON
}

// UnmarshalJSON implements json.Unmarshaler interface.
func (t *BlockTest) UnmarshalJSON(in []byte) error {
	return json.Unmarshal(in, &t.json)
}

type btJSON struct {
	Blocks    []btBlock             `json:"blocks"`
	Genesis   btHeader              `json:"genesisBlockHeader"`
	Pre       core.GenesisAlloc     `json:"pre"`
	Post      core.GenesisAlloc     `json:"postState"`
	BestBlock common.UnprefixedHash `json:"lastblockhash"`
	Network   string                `json:"network"`
}

type btBlock struct {
	BlockHeader  *btHeader
	Rlp          string
	UncleHeaders []*btHeader
}

//go:generate gencodec -type btHeader -field-override btHeaderMarshaling -out gen_btheader.go

type btHeader struct {
	Bloom            types.Bloom
	Coinbase         common.Address
	MixHash          common.Hash
	Nonce            types.BlockNonce
	Number           *big.Int
	Hash             common.Hash
	ParentHash       common.Hash
	ReceiptTrie      common.Hash
	StateRoot        common.Hash
	TransactionsTrie common.Hash
	UncleHash        common.Hash
	ExtraData        []byte
	Difficulty       *big.Int
	GasLimit         uint64
	GasUsed          uint64
	Timestamp        *big.Int
}

type btHeaderMarshaling struct {
	ExtraData  hexutil.Bytes
	Number     *math.HexOrDecimal256
	Difficulty *math.HexOrDecimal256
	GasLimit   math.HexOrDecimal64
	GasUsed    math.HexOrDecimal64
	Timestamp  *math.HexOrDecimal256
}

func (t *BlockTest) Run() error {
	config, ok := Forks[t.json.Network]
	if !ok {
		return UnsupportedForkError{t.json.Network}
	}

	// import pre accounts & construct test genesis block & state root
	db := ethdb.NewMemDatabase()
	gblock, err := t.genesis(config).Commit(db)
	if err != nil {
		return err
	}
	if gblock.Hash() != t.json.Genesis.Hash {
		return fmt.Errorf("genesis block hash doesn't match test: computed=%x, test=%x", gblock.Hash().Bytes()[:6], t.json.Genesis.Hash[:6])
	}
	if gblock.Root() != t.json.Genesis.StateRoot {
		return fmt.Errorf("genesis block state root does not match test: computed=%x, test=%x", gblock.Root().Bytes()[:6], t.json.Genesis.StateRoot[:6])
	}

	chain, err := core.NewBlockChain(db, nil, config, ethash.NewShared(), vm.Config{}, nil)
	if err != nil {
		return err
	}
	defer chain.Stop()

	validBlocks, err := t.insertBlocks(chain)
	if err != nil {
		return err
	}
	cmlast := chain.CurrentBlock().Hash()
	if common.Hash(t.json.BestBlock) != cmlast {
		return fmt.Errorf("last block hash validation mismatch: want: %x, have: %x", t.json.BestBlock, cmlast)
	}
	newDB, err := chain.State()
	if err != nil {
		return err
	}
	if err = t.validatePostState(newDB); err != nil {
		return fmt.Errorf("post state validation failed: %v", err)
	}
	return t.validateImportedHeaders(chain, validBlocks)
}

func (t *BlockTest) genesis(config *params.ChainConfig) *core.Genesis {
	return &core.Genesis{
		Config:     config,
		Nonce:      t.json.Genesis.Nonce.Uint64(),
		Timestamp:  t.json.Genesis.Timestamp.Uint64(),
		ParentHash: t.json.Genesis.ParentHash,
		ExtraData:  t.json.Genesis.ExtraData,
		GasLimit:   t.json.Genesis.GasLimit,
		GasUsed:    t.json.Genesis.GasUsed,
		Difficulty: t.json.Genesis.Difficulty,
		Mixhash:    t.json.Genesis.MixHash,
		Coinbase:   t.json.Genesis.Coinbase,
		Alloc:      t.json.Pre,
	}
}

/* See https://github.com/ethereum/tests/wiki/Blockchain-Tests-II

   Whether a block is valid or not is a bit subtle, it's defined by presence of
   blockHeader, transactions and uncleHeaders fields. If they are missing, the block is
   invalid and we must verify that we do not accept it.

   Since some tests mix valid and invalid blocks we need to check this for every block.

   If a block is invalid it does not necessarily fail the test, if it's invalidness is
   expected we are expected to ignore it and continue processing and then validate the
   post state.
*/
func (t *BlockTest) insertBlocks(blockchain *core.BlockChain) ([]btBlock, error) {
	validBlocks := make([]btBlock, 0)
	// insert the test blocks, which will execute all transactions
	for _, b := range t.json.Blocks {
		cb, err := b.decode()
		if err != nil {
			if b.BlockHeader == nil {
				continue // OK - block is supposed to be invalid, continue with next block
			} else {
				return nil, fmt.Errorf("Block RLP decoding failed when expected to succeed: %v", err)
			}
		}
		// RLP decoding worked, try to insert into chain:
		blocks := types.Blocks{cb}
		i, err := blockchain.InsertChain(blocks)
		if err != nil {
			if b.BlockHeader == nil {
				continue // OK - block is supposed to be invalid, continue with next block
			} else {
				return nil, fmt.Errorf("Block #%v insertion into chain failed: %v", blocks[i].Number(), err)
			}
		}
		if b.BlockHeader == nil {
			return nil, fmt.Errorf("Block insertion should have failed")
		}

		// validate RLP decoding by checking all values against test file JSON
		if err = validateHeader(b.BlockHeader, cb.Header()); err != nil {
			return nil, fmt.Errorf("Deserialised block header validation failed: %v", err)
		}
		validBlocks = append(validBlocks, b)
	}
	return validBlocks, nil
}

func validateHeader(h *btHeader, h2 *types.Header) error {
	if h.Bloom != h2.Bloom {
		return fmt.Errorf("Bloom: want: %x have: %x", h.Bloom, h2.Bloom)
	}
	if h.Coinbase != h2.Coinbase {
		return fmt.Errorf("Coinbase: want: %x have: %x", h.Coinbase, h2.Coinbase)
	}
	if h.MixHash != h2.MixDigest {
		return fmt.Errorf("MixHash: want: %x have: %x", h.MixHash, h2.MixDigest)
	}
	if h.Nonce != h2.Nonce {
		return fmt.Errorf("Nonce: want: %x have: %x", h.Nonce, h2.Nonce)
	}
	if h.Number.Cmp(h2.Number) != 0 {
		return fmt.Errorf("Number: want: %v have: %v", h.Number, h2.Number)
	}
	if h.ParentHash != h2.ParentHash {
		return fmt.Errorf("Parent hash: want: %x have: %x", h.ParentHash, h2.ParentHash)
	}
	if h.ReceiptTrie != h2.ReceiptHash {
		return fmt.Errorf("Receipt hash: want: %x have: %x", h.ReceiptTrie, h2.ReceiptHash)
	}
	if h.TransactionsTrie != h2.TxHash {
		return fmt.Errorf("Tx hash: want: %x have: %x", h.TransactionsTrie, h2.TxHash)
	}
	if h.StateRoot != h2.Root {
		return fmt.Errorf("State hash: want: %x have: %x", h.StateRoot, h2.Root)
	}
	if h.UncleHash != h2.UncleHash {
		return fmt.Errorf("Uncle hash: want: %x have: %x", h.UncleHash, h2.UncleHash)
	}
	if !bytes.Equal(h.ExtraData, h2.Extra) {
		return fmt.Errorf("Extra data: want: %x have: %x", h.ExtraData, h2.Extra)
	}
	if h.Difficulty.Cmp(h2.Difficulty) != 0 {
		return fmt.Errorf("Difficulty: want: %v have: %v", h.Difficulty, h2.Difficulty)
	}
	if h.GasLimit != h2.GasLimit {
		return fmt.Errorf("GasLimit: want: %d have: %d", h.GasLimit, h2.GasLimit)
	}
	if h.GasUsed != h2.GasUsed {
		return fmt.Errorf("GasUsed: want: %d have: %d", h.GasUsed, h2.GasUsed)
	}
	if h.Timestamp.Cmp(h2.Time) != 0 {
		return fmt.Errorf("Timestamp: want: %v have: %v", h.Timestamp, h2.Time)
	}
	return nil
}

func (t *BlockTest) validatePostState(statedb *state.StateDB) error {
	// validate post state accounts in test file against what we have in state db
	for addr, acct := range t.json.Post {
		// address is indirectly verified by the other fields, as it's the db key
		code2 := statedb.GetCode(addr)
		balance2 := statedb.GetBalance(addr)
		nonce2 := statedb.GetNonce(addr)
		if !bytes.Equal(code2, acct.Code) {
			return fmt.Errorf("account code mismatch for addr: %s want: %v have: %s", addr, acct.Code, hex.EncodeToString(code2))
		}
		if balance2.Cmp(acct.Balance) != 0 {
			return fmt.Errorf("account balance mismatch for addr: %s, want: %d, have: %d", addr, acct.Balance, balance2)
		}
		if nonce2 != acct.Nonce {
			return fmt.Errorf("account nonce mismatch for addr: %s want: %d have: %d", addr, acct.Nonce, nonce2)
		}
	}
	return nil
}

func (t *BlockTest) validateImportedHeaders(cm *core.BlockChain, validBlocks []btBlock) error {
	// to get constant lookup when verifying block headers by hash (some tests have many blocks)
	bmap := make(map[common.Hash]btBlock, len(t.json.Blocks))
	for _, b := range validBlocks {
		bmap[b.BlockHeader.Hash] = b
	}
	// iterate over blocks backwards from HEAD and validate imported
	// headers vs test file. some tests have reorgs, and we import
	// block-by-block, so we can only validate imported headers after
	// all blocks have been processed by BlockChain, as they may not
	// be part of the longest chain until last block is imported.
	for b := cm.CurrentBlock(); b != nil && b.NumberU64() != 0; b = cm.GetBlockByHash(b.Header().ParentHash) {
		if err := validateHeader(bmap[b.Hash()].BlockHeader, b.Header()); err != nil {
			return fmt.Errorf("Imported block header validation failed: %v", err)
		}
	}
	return nil
}

func (bb *btBlock) decode() (*types.Block, error) {
	data, err := hexutil.Decode(bb.Rlp)
	if err != nil {
		return nil, err
	}
	var b types.Block
	err = rlp.DecodeBytes(data, &b)
	return &b, err
}
