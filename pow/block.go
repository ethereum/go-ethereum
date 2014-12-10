package pow

import "math/big"

type Block interface {
	Diff() *big.Int
	HashNoNonce() []byte
	N() []byte
}
