package core

import (
	"math/big"

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
	genesis := types.NewBlock(ZeroHash256, ZeroHash160, nil, big.NewInt(131072), crypto.Sha3(big.NewInt(42).Bytes()), "")
	genesis.Header().Number = ethutil.Big0
	genesis.Header().GasLimit = big.NewInt(1000000)
	genesis.Header().GasUsed = ethutil.Big0
	genesis.Header().Time = 0

	genesis.SetUncles([]*types.Header{})
	genesis.SetTransactions(types.Transactions{})
	genesis.SetReceipts(types.Receipts{})

	statedb := state.New(genesis.Root(), db)
	//statedb := state.New(genesis.Trie())
	for _, addr := range []string{
		"51ba59315b3a95761d0863b05ccc7a7f54703d99",
		"e4157b34ea9615cfbde6b4fda419828124b70c78",
		"b9c015918bdaba24b4ff057a92a3873d6eb201be",
		"6c386a4b26f73c802f34673f7248bb118f97424a",
		"cd2a3d9f938e13cd947ec05abc7fe734df8dd826",
		"2ef47100e0787b915105fd5e3f4ff6752079d5cb",
		"e6716f9544a56c530d868e4bfbacb172315bdead",
		"1a26338f0d905e295fccb71fa9ea849ffa12aaf4",
	} {
		codedAddr := ethutil.Hex2Bytes(addr)
		account := statedb.GetAccount(codedAddr)
		account.SetBalance(ethutil.Big("1606938044258990275541962092341162602522202993782792835301376")) //ethutil.BigPow(2, 200)
		statedb.UpdateStateObject(account)
	}
	statedb.Sync()
	genesis.Header().Root = statedb.Root()

	return genesis
}
