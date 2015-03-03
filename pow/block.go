package pow

import (
	"github.com/ethereum/go-ethereum/core/types"
	"math/big"
)

type Block interface {
	Difficulty() *big.Int
	HashNoNonce() []byte
	Nonce() []byte
	MixDigest() []byte
	SeedHash() []byte
	NumberU64() uint64
}

type ChainManager interface {
	GetBlockByNumber(uint64) *types.Block
	CurrentBlock() *types.Block
}
