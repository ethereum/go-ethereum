package vm

import "math/big"

type Log struct {
	Address []byte
	Topics  []*big.Int
	Data    []byte
}
