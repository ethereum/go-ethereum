package codehash

import (
	"github.com/scroll-tech/go-ethereum/common"
	"github.com/scroll-tech/go-ethereum/crypto/poseidon"
)

var EmptyCodeHash common.Hash

func CodeHash(code []byte) (h common.Hash) {
	return poseidon.CodeHash(code)
}

func init() {
	EmptyCodeHash = poseidon.CodeHash(nil)
}
