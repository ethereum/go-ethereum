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

package core

import (
	"compress/bzip2"
	"compress/gzip"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"math/big"
	"strings"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/math"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/params"
)

// WriteGenesisBlock writes the genesis block to the database as block number 0
func WriteGenesisBlock(chainDb ethdb.Database, reader io.Reader) (*types.Block, error) {
	contents, err := ioutil.ReadAll(reader)
	if err != nil {
		return nil, err
	}

	var genesis struct {
		ChainConfig *params.ChainConfig `json:"config"`
		Nonce       string
		Timestamp   string
		ParentHash  string
		ExtraData   string
		GasLimit    string
		Difficulty  string
		Mixhash     string
		Coinbase    string
		Alloc       map[string]struct {
			Code    string
			Storage map[string]string
			Balance string
			Nonce   string
		}
	}

	if err := json.Unmarshal(contents, &genesis); err != nil {
		return nil, err
	}
	if genesis.ChainConfig == nil {
		genesis.ChainConfig = params.AllProtocolChanges
	}

	// creating with empty hash always works
	statedb, _ := state.New(common.Hash{}, chainDb)
	for addr, account := range genesis.Alloc {
		balance, ok := math.ParseBig256(account.Balance)
		if !ok {
			return nil, fmt.Errorf("invalid balance for account %s: %q", addr, account.Balance)
		}
		nonce, ok := math.ParseUint64(account.Nonce)
		if !ok {
			return nil, fmt.Errorf("invalid nonce for account %s: %q", addr, account.Nonce)
		}

		address := common.HexToAddress(addr)
		statedb.AddBalance(address, balance)
		statedb.SetCode(address, common.FromHex(account.Code))
		statedb.SetNonce(address, nonce)
		for key, value := range account.Storage {
			statedb.SetState(address, common.HexToHash(key), common.HexToHash(value))
		}
	}
	root, stateBatch := statedb.CommitBatch(false)

	difficulty, ok := math.ParseBig256(genesis.Difficulty)
	if !ok {
		return nil, fmt.Errorf("invalid difficulty: %q", genesis.Difficulty)
	}
	gaslimit, ok := math.ParseUint64(genesis.GasLimit)
	if !ok {
		return nil, fmt.Errorf("invalid gas limit: %q", genesis.GasLimit)
	}
	nonce, ok := math.ParseUint64(genesis.Nonce)
	if !ok {
		return nil, fmt.Errorf("invalid nonce: %q", genesis.Nonce)
	}
	timestamp, ok := math.ParseBig256(genesis.Timestamp)
	if !ok {
		return nil, fmt.Errorf("invalid timestamp: %q", genesis.Timestamp)
	}

	block := types.NewBlock(&types.Header{
		Nonce:      types.EncodeNonce(nonce),
		Time:       timestamp,
		ParentHash: common.HexToHash(genesis.ParentHash),
		Extra:      common.FromHex(genesis.ExtraData),
		GasLimit:   new(big.Int).SetUint64(gaslimit),
		Difficulty: difficulty,
		MixDigest:  common.HexToHash(genesis.Mixhash),
		Coinbase:   common.HexToAddress(genesis.Coinbase),
		Root:       root,
	}, nil, nil, nil)

	if block := GetBlock(chainDb, block.Hash(), block.NumberU64()); block != nil {
		log.Info("Genesis block known, writing canonical number")
		err := WriteCanonicalHash(chainDb, block.Hash(), block.NumberU64())
		if err != nil {
			return nil, err
		}
		return block, nil
	}

	if err := stateBatch.Write(); err != nil {
		return nil, fmt.Errorf("cannot write state: %v", err)
	}
	if err := WriteTd(chainDb, block.Hash(), block.NumberU64(), difficulty); err != nil {
		return nil, err
	}
	if err := WriteBlock(chainDb, block); err != nil {
		return nil, err
	}
	if err := WriteBlockReceipts(chainDb, block.Hash(), block.NumberU64(), nil); err != nil {
		return nil, err
	}
	if err := WriteCanonicalHash(chainDb, block.Hash(), block.NumberU64()); err != nil {
		return nil, err
	}
	if err := WriteHeadBlockHash(chainDb, block.Hash()); err != nil {
		return nil, err
	}
	if err := WriteChainConfig(chainDb, block.Hash(), genesis.ChainConfig); err != nil {
		return nil, err
	}
	return block, nil
}

// GenesisBlockForTesting creates a block in which addr has the given wei balance.
// The state trie of the block is written to db. the passed db needs to contain a state root
func GenesisBlockForTesting(db ethdb.Database, addr common.Address, balance *big.Int) *types.Block {
	statedb, _ := state.New(common.Hash{}, db)
	obj := statedb.GetOrNewStateObject(addr)
	obj.SetBalance(balance)
	root, err := statedb.Commit(false)
	if err != nil {
		panic(fmt.Sprintf("cannot write state: %v", err))
	}
	block := types.NewBlock(&types.Header{
		Difficulty: params.GenesisDifficulty,
		GasLimit:   params.GenesisGasLimit,
		Root:       root,
	}, nil, nil, nil)
	return block
}

type GenesisAccount struct {
	Address common.Address
	Balance *big.Int
}

func WriteGenesisBlockForTesting(db ethdb.Database, accounts ...GenesisAccount) *types.Block {
	accountJson := "{"
	for i, account := range accounts {
		if i != 0 {
			accountJson += ","
		}
		accountJson += fmt.Sprintf(`"0x%x":{"balance":"%d"}`, account.Address, account.Balance)
	}
	accountJson += "}"

	testGenesis := fmt.Sprintf(`{
	"nonce":"0x%x",
	"gasLimit":"0x%x",
	"difficulty":"0x%x",
	"alloc": %s
}`, types.EncodeNonce(0), params.GenesisGasLimit.Bytes(), params.GenesisDifficulty.Bytes(), accountJson)
	block, err := WriteGenesisBlock(db, strings.NewReader(testGenesis))
	if err != nil {
		panic(err)
	}
	return block
}

// WriteDefaultGenesisBlock assembles the official Ethereum genesis block and
// writes it - along with all associated state - into a chain database.
func WriteDefaultGenesisBlock(chainDb ethdb.Database) (*types.Block, error) {
	return WriteGenesisBlock(chainDb, strings.NewReader(DefaultGenesisBlock()))
}

// WriteTestNetGenesisBlock assembles the test network genesis block and
// writes it - along with all associated state - into a chain database.
func WriteTestNetGenesisBlock(chainDb ethdb.Database) (*types.Block, error) {
	return WriteGenesisBlock(chainDb, strings.NewReader(DefaultTestnetGenesisBlock()))
}

// DefaultGenesisBlock assembles a JSON string representing the default Ethereum
// genesis block.
func DefaultGenesisBlock() string {
	reader, err := gzip.NewReader(base64.NewDecoder(base64.StdEncoding, strings.NewReader(defaultGenesisBlock)))
	if err != nil {
		panic(fmt.Sprintf("failed to access default genesis: %v", err))
	}
	blob, err := ioutil.ReadAll(reader)
	if err != nil {
		panic(fmt.Sprintf("failed to load default genesis: %v", err))
	}
	return string(blob)
}

// DefaultTestnetGenesisBlock assembles a JSON string representing the default Ethereum
// test network genesis block.
func DefaultTestnetGenesisBlock() string {
	reader := bzip2.NewReader(base64.NewDecoder(base64.StdEncoding, strings.NewReader(defaultTestnetGenesisBlock)))
	blob, err := ioutil.ReadAll(reader)
	if err != nil {
		panic(fmt.Sprintf("failed to load default genesis: %v", err))
	}
	return string(blob)
}

// DevGenesisBlock assembles a JSON string representing a local dev genesis block.
func DevGenesisBlock() string {
	reader := bzip2.NewReader(base64.NewDecoder(base64.StdEncoding, strings.NewReader(defaultDevnetGenesisBlock)))
	blob, err := ioutil.ReadAll(reader)
	if err != nil {
		panic(fmt.Sprintf("failed to load dev genesis: %v", err))
	}
	return string(blob)
}
