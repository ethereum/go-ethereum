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
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"math/big"
	"strings"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/logger"
	"github.com/ethereum/go-ethereum/logger/glog"
	"github.com/ethereum/go-ethereum/params"
)

// WriteGenesisBlock writes the genesis block to the database as block number 0
func WriteGenesisBlock(chainDb common.Database, reader io.Reader) (*types.Block, error) {
	contents, err := ioutil.ReadAll(reader)
	if err != nil {
		return nil, err
	}

	var genesis struct {
		Nonce      string
		Timestamp  string
		ParentHash string
		ExtraData  string
		GasLimit   string
		Difficulty string
		Mixhash    string
		Coinbase   string
		Alloc      map[string]struct {
			Code    string
			Storage map[string]string
			Balance string
		}
	}

	if err := json.Unmarshal(contents, &genesis); err != nil {
		return nil, err
	}

	statedb := state.New(common.Hash{}, chainDb)
	for addr, account := range genesis.Alloc {
		address := common.HexToAddress(addr)
		statedb.AddBalance(address, common.String2Big(account.Balance))
		statedb.SetCode(address, common.Hex2Bytes(account.Code))
		for key, value := range account.Storage {
			statedb.SetState(address, common.HexToHash(key), common.HexToHash(value))
		}
	}
	statedb.SyncObjects()

	difficulty := common.String2Big(genesis.Difficulty)
	block := types.NewBlock(&types.Header{
		Nonce:      types.EncodeNonce(common.String2Big(genesis.Nonce).Uint64()),
		Time:       common.String2Big(genesis.Timestamp),
		ParentHash: common.HexToHash(genesis.ParentHash),
		Extra:      common.FromHex(genesis.ExtraData),
		GasLimit:   common.String2Big(genesis.GasLimit),
		Difficulty: difficulty,
		MixDigest:  common.HexToHash(genesis.Mixhash),
		Coinbase:   common.HexToAddress(genesis.Coinbase),
		Root:       statedb.Root(),
	}, nil, nil, nil)
	block.Td = difficulty

	if block := GetBlockByHash(chainDb, block.Hash()); block != nil {
		glog.V(logger.Info).Infoln("Genesis block already in chain. Writing canonical number")
		err := WriteCanonNumber(chainDb, block)
		if err != nil {
			return nil, err
		}
		return block, nil
	}

	statedb.Sync()

	err = WriteBlock(chainDb, block)
	if err != nil {
		return nil, err
	}
	err = WriteHead(chainDb, block)
	if err != nil {
		return nil, err
	}

	return block, nil
}

// GenesisBlockForTesting creates a block in which addr has the given wei balance.
// The state trie of the block is written to db.
func GenesisBlockForTesting(db common.Database, addr common.Address, balance *big.Int) *types.Block {
	statedb := state.New(common.Hash{}, db)
	obj := statedb.GetOrNewStateObject(addr)
	obj.SetBalance(balance)
	statedb.SyncObjects()
	statedb.Sync()
	block := types.NewBlock(&types.Header{
		Difficulty: params.GenesisDifficulty,
		GasLimit:   params.GenesisGasLimit,
		Root:       statedb.Root(),
	}, nil, nil, nil)
	block.Td = params.GenesisDifficulty
	return block
}

func WriteGenesisBlockForTesting(db common.Database, addr common.Address, balance *big.Int) *types.Block {
	testGenesis := fmt.Sprintf(`{
	"nonce":"0x%x",
	"gasLimit":"0x%x",
	"difficulty":"0x%x",
	"alloc": {
		"0x%x":{"balance":"0x%x"}
	}
}`, types.EncodeNonce(0), params.GenesisGasLimit.Bytes(), params.GenesisDifficulty.Bytes(), addr, balance.Bytes())
	block, _ := WriteGenesisBlock(db, strings.NewReader(testGenesis))
	return block
}

func WriteTestNetGenesisBlock(chainDb common.Database, nonce uint64) (*types.Block, error) {
	testGenesis := fmt.Sprintf(`{
	"nonce":"0x%x",
	"gasLimit":"0x%x",
	"difficulty":"0x%x",
	"alloc": {
		"0000000000000000000000000000000000000001": {"balance": "1"},
		"0000000000000000000000000000000000000002": {"balance": "1"},
		"0000000000000000000000000000000000000003": {"balance": "1"},
		"0000000000000000000000000000000000000004": {"balance": "1"},
		"dbdbdb2cbd23b783741e8d7fcf51e459b497e4a6": {"balance": "1606938044258990275541962092341162602522202993782792835301376"},
		"e4157b34ea9615cfbde6b4fda419828124b70c78": {"balance": "1606938044258990275541962092341162602522202993782792835301376"},
		"b9c015918bdaba24b4ff057a92a3873d6eb201be": {"balance": "1606938044258990275541962092341162602522202993782792835301376"},
		"6c386a4b26f73c802f34673f7248bb118f97424a": {"balance": "1606938044258990275541962092341162602522202993782792835301376"},
		"cd2a3d9f938e13cd947ec05abc7fe734df8dd826": {"balance": "1606938044258990275541962092341162602522202993782792835301376"},
		"2ef47100e0787b915105fd5e3f4ff6752079d5cb": {"balance": "1606938044258990275541962092341162602522202993782792835301376"},
		"e6716f9544a56c530d868e4bfbacb172315bdead": {"balance": "1606938044258990275541962092341162602522202993782792835301376"},
		"1a26338f0d905e295fccb71fa9ea849ffa12aaf4": {"balance": "1606938044258990275541962092341162602522202993782792835301376"}
	}
}`, types.EncodeNonce(nonce), params.GenesisGasLimit.Bytes(), params.GenesisDifficulty.Bytes())
	return WriteGenesisBlock(chainDb, strings.NewReader(testGenesis))
}
