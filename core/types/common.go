package types

import (
	"math/big"

	"github.com/ethereum/go-ethereum/state"
	"github.com/ethereum/go-ethereum/common"

	"fmt"
)

type BlockProcessor interface {
	Process(*Block) (*big.Int, state.Logs, error)
}

const bloomLength = 256

type Bloom [bloomLength]byte

func BytesToBloom(b []byte) Bloom {
	var bloom Bloom
	bloom.SetBytes(b)
	return bloom
}

func (b *Bloom) SetBytes(d []byte) {
	if len(b) < len(d) {
		panic(fmt.Sprintf("bloom bytes too big %d %d", len(b), len(d)))
	}

	// reverse loop
	for i := len(d) - 1; i >= 0; i-- {
		b[bloomLength-len(d)+i] = b[i]
	}
}

func (b Bloom) Big() *big.Int {
	return common.Bytes2Big(b[:])
}

func (b Bloom) Bytes() []byte {
	return b[:]
}
