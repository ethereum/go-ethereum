// Copyright 2014 The go-ethereum Authors
// This file is part of go-ethereum.
//
// go-ethereum is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// go-ethereum is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with go-ethereum.  If not, see <http://www.gnu.org/licenses/>.

package core

import (
	"encoding/json"
	"fmt"
	"math/big"
	"os"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/params"
)

// GenesisBlock creates a genesis block with the given nonce.
func GenesisBlock(nonce uint64, db common.Database) *types.Block {
	var accounts map[string]struct {
		Balance string
		Code    string
	}
	err := json.Unmarshal(GenesisAccounts, &accounts)
	if err != nil {
		fmt.Println("unable to decode genesis json data:", err)
		os.Exit(1)
	}
	statedb := state.New(common.Hash{}, db)
	for addr, account := range accounts {
		codedAddr := common.Hex2Bytes(addr)
		accountState := statedb.CreateAccount(common.BytesToAddress(codedAddr))
		accountState.SetBalance(common.Big(account.Balance))
		accountState.SetCode(common.FromHex(account.Code))
		statedb.UpdateStateObject(accountState)
	}
	statedb.Sync()

	block := types.NewBlock(&types.Header{
		Difficulty: params.GenesisDifficulty,
		GasLimit:   params.GenesisGasLimit,
		Nonce:      types.EncodeNonce(nonce),
		Root:       statedb.Root(),
	}, nil, nil, nil)
	block.Td = params.GenesisDifficulty
	return block
}

var GenesisAccounts = []byte(`{
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
}`)

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
