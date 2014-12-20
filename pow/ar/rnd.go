package ar

import (
	"math/big"

	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethutil"
)

var b = big.NewInt

type Node interface {
	Big() *big.Int
}

type ByteNode []byte

func (self ByteNode) Big() *big.Int {
	return ethutil.BigD(ethutil.Encode([]byte(self)))
}

func Sha3(v interface{}) *big.Int {
	if b, ok := v.(*big.Int); ok {
		return ethutil.BigD(crypto.Sha3(b.Bytes()))
	} else if b, ok := v.([]interface{}); ok {
		return ethutil.BigD(crypto.Sha3(ethutil.Encode(b)))
	} else if s, ok := v.([]*big.Int); ok {
		v := make([]interface{}, len(s))
		for i, b := range s {
			v[i] = b
		}

		return ethutil.BigD(crypto.Sha3(ethutil.Encode(v)))
	}

	return nil
}

type NumberGenerator interface {
	rand(r *big.Int) *big.Int
	rand64(r int64) *big.Int
}

type rnd struct {
	seed *big.Int
}

func Rnd(s *big.Int) rnd {
	return rnd{s}
}

func (self rnd) rand(r *big.Int) *big.Int {
	o := b(0).Mod(self.seed, r)

	self.seed.Div(self.seed, r)

	if self.seed.Cmp(ethutil.BigPow(2, 64)) < 0 {
		self.seed = Sha3(self.seed)
	}

	return o
}

func (self rnd) rand64(r int64) *big.Int {
	return self.rand(b(r))
}
