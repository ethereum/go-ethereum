package core

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/params"
)

/*
 * This is the special genesis block.
 */

var ZeroHash256 = make([]byte, 32)
var ZeroHash160 = make([]byte, 20)
var ZeroHash512 = make([]byte, 64)

func GenesisBlock(nonce uint64, db common.Database) *types.Block {
	genesis := types.NewBlock(common.Hash{}, common.Address{}, common.Hash{}, params.GenesisDifficulty, nonce, nil)
	genesis.Header().Number = common.Big0
	genesis.Header().GasLimit = params.GenesisGasLimit
	genesis.Header().GasUsed = common.Big0
	genesis.Header().Time = 0

	genesis.Td = common.Big0

	genesis.SetUncles([]*types.Header{})
	genesis.SetTransactions(types.Transactions{})
	genesis.SetReceipts(types.Receipts{})

	var accounts map[string]struct {
		Balance string
		Code    string
	}
	err := json.Unmarshal(GenesisAccounts, &accounts)
	if err != nil {
		fmt.Println("enable to decode genesis json data:", err)
		os.Exit(1)
	}

	statedb := state.New(genesis.Root(), db)
	for addr, account := range accounts {
		codedAddr := common.Hex2Bytes(addr)
		accountState := statedb.CreateAccount(common.BytesToAddress(codedAddr))
		accountState.SetBalance(common.Big(account.Balance))
		accountState.SetCode(common.FromHex(account.Code))
		statedb.UpdateStateObject(accountState)
	}
	statedb.Sync()
	genesis.Header().Root = statedb.Root()
	genesis.Td = params.GenesisDifficulty

	return genesis
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
