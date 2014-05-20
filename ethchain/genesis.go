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
	ZeroHash256,
	// Sha of uncles
	ethutil.Sha3Bin(ethutil.Encode([]interface{}{})),
	// Coinbase
	ZeroHash160,
	// Root state
	"",
	//EmptyShaList,
	ZeroHash256,
	//ethutil.Sha3Bin(ethutil.Encode(ZeroHash256)),
	// Difficulty
	ethutil.BigPow(2, 22),
	// Number
	ethutil.Big0,
	// Block minimum gas price
	ethutil.Big0,
	// Block upper gas bound
	big.NewInt(1000000),
	// Block gas used
	ethutil.Big0,
	// Time
	ethutil.Big0,
	// Extra
	nil,
	// Nonce
	ethutil.Sha3Bin(big.NewInt(42).Bytes()),
}

var Genesis = []interface{}{GenesisHeader, []interface{}{}, []interface{}{}}
