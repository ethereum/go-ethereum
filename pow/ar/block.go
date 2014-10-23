package ar

import (
	"math/big"

	"github.com/ethereum/go-ethereum/ethtrie"
)

type Block interface {
	Trie() *ethtrie.Trie
	Diff() *big.Int
}
