package pow

import "math/big"

type Block interface {
	Difficulty() *big.Int
	HashNoNonce() []byte
	N() []byte
}
