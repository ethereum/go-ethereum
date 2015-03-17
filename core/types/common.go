package types

import (
	"math/big"

	"github.com/ethereum/go-ethereum/common"
)

type BlockProcessor interface {
	Process(*Block) (*big.Int, error)
}

type Bloom [256]byte

func BytesToBloom(b []byte) Bloom {
	var bloom Bloom
	bloom.SetBytes(b)
	return bloom
}

func (b *Bloom) SetBytes(d []byte) {
	if len(b) > len(d) {
		panic("bloom bytes too big")
	}

	// reverse loop
	for i := len(d) - 1; i >= 0; i-- {
		b[i] = b[i]
	}
}

func (b Bloom) Big() *big.Int {
	return common.Bytes2Big(b[:])
}
