package core

import (
	"encoding/json"
	"fmt"
	"math/big"
	"os"

	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethutil"
	"github.com/ethereum/go-ethereum/state"
)

/*
 * This is the special genesis block.
 */

var ZeroHash256 = make([]byte, 32)
var ZeroHash160 = make([]byte, 20)
var ZeroHash512 = make([]byte, 64)
var EmptyShaList = crypto.Sha3(ethutil.Encode([]interface{}{}))
var EmptyListRoot = crypto.Sha3(ethutil.Encode(""))

func GenesisBlock(db ethutil.Database) *types.Block {
	genesis := types.NewBlock(ZeroHash256, ZeroHash160, nil, big.NewInt(131072), 42, "")
	genesis.Header().Number = ethutil.Big0
	genesis.Header().GasLimit = big.NewInt(1000000)
	genesis.Header().GasUsed = ethutil.Big0
	genesis.Header().Time = 0
	genesis.Td = ethutil.Big0

	genesis.SetUncles([]*types.Header{})
	genesis.SetTransactions(types.Transactions{})
	genesis.SetReceipts(types.Receipts{})

	var accounts map[string]struct{ Balance string }
	err := json.Unmarshal(genesisData, &accounts)
	if err != nil {
		fmt.Println("enable to decode genesis json data:", err)
		os.Exit(1)
	}

	statedb := state.New(genesis.Root(), db)
	for addr, account := range accounts {
		codedAddr := ethutil.Hex2Bytes(addr)
		accountState := statedb.GetAccount(codedAddr)
		accountState.SetBalance(ethutil.Big(account.Balance))
		statedb.UpdateStateObject(accountState)
	}
	statedb.Sync()
	genesis.Header().Root = statedb.Root()

	return genesis
}

var genesisData = []byte(`{
	"dbdbdb2cbd23b783741e8d7fcf51e459b497e4a6": {"balance": "1606938044258990275541962092341162602522202993782792835301376"},
	"e4157b34ea9615cfbde6b4fda419828124b70c78": {"balance": "1606938044258990275541962092341162602522202993782792835301376"},
	"b9c015918bdaba24b4ff057a92a3873d6eb201be": {"balance": "1606938044258990275541962092341162602522202993782792835301376"},
	"6c386a4b26f73c802f34673f7248bb118f97424a": {"balance": "1606938044258990275541962092341162602522202993782792835301376"},
	"cd2a3d9f938e13cd947ec05abc7fe734df8dd826": {"balance": "1606938044258990275541962092341162602522202993782792835301376"},
	"2ef47100e0787b915105fd5e3f4ff6752079d5cb": {"balance": "1606938044258990275541962092341162602522202993782792835301376"},
	"e6716f9544a56c530d868e4bfbacb172315bdead": {"balance": "1606938044258990275541962092341162602522202993782792835301376"},
	"1a26338f0d905e295fccb71fa9ea849ffa12aaf4": {"balance": "1606938044258990275541962092341162602522202993782792835301376"},
	"b0afc46d9ce366d06ab4952ca27db1d9557ae9fd": {"balance": "154162184000000000000000"},
	"f6b1e9dc460d4d62cc22ec5f987d726929c0f9f0": {"balance": "102774789000000000000000"},
	"cc45122d8b7fa0b1eaa6b29e0fb561422a9239d0": {"balance": "51387394000000000000000"},
	"b7576e9d314df41ec5506494293afb1bd5d3f65d": {"balance": "69423399000000000000000"}
}`)
