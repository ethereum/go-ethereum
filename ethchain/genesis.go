package ethchain

import (
	"math/big"

	"github.com/ethereum/eth-go/ethcrypto"
	"github.com/ethereum/eth-go/ethutil"
)

/*
 * This is the special genesis block.
 */

var ZeroHash256 = make([]byte, 32)
var ZeroHash160 = make([]byte, 20)
var EmptyShaList = ethcrypto.Sha3(ethutil.Encode([]interface{}{}))

var GenesisHeader = []interface{}{
	// Previous hash (none)
	ZeroHash256,
	// Empty uncles
	"",
	// Coinbase
	ZeroHash160,
	// Root state
	"",
	// tx sha
	"",
	// Difficulty
	//ethutil.BigPow(2, 22),
	big.NewInt(131072),
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
	ethcrypto.Sha3(big.NewInt(42).Bytes()),
}

var Genesis = []interface{}{GenesisHeader, []interface{}{}, []interface{}{}}
