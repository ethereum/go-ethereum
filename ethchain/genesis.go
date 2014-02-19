package ethchain

import (
	"github.com/ethereum/eth-go/ethutil"
	"math/big"
)

/*
 * This is the special genesis block.
 */

var ZeroHash256 = make([]byte, 32)
var ZeroHash160 = make([]byte, 20)
var EmptyShaList = ethutil.Sha3Bin(ethutil.Encode([]interface{}{}))

var GenesisHeader = []interface{}{
	// Previous hash (none)
	//"",
	ZeroHash256,
	// Sha of uncles
	ethutil.Sha3Bin(ethutil.Encode([]interface{}{})),
	// Coinbase
	ZeroHash160,
	// Root state
	"",
	// Sha of transactions
	//EmptyShaList,
	ethutil.Sha3Bin(ethutil.Encode([]interface{}{})),
	// Difficulty
	ethutil.BigPow(2, 22),
	// Time
	int64(0),
	// Extra
	"",
	// Nonce
	ethutil.Sha3Bin(big.NewInt(42).Bytes()),
}

var Genesis = []interface{}{GenesisHeader, []interface{}{}, []interface{}{}}
