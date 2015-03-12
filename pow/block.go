package pow

import (
	"math/big"

	"github.com/ethereum/go-ethereum/core/types"
)

type Block interface {
	Difficulty() *big.Int
	HashNoNonce() []byte
	Nonce() uint64
	MixDigest() []byte
	SeedHash() []byte
	NumberU64() uint64
}

type ChainManager interface {
	GetBlockByNumber(uint64) *types.Block
	CurrentBlock() *types.Block
}
