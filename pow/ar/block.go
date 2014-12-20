package ar

import (
	"math/big"

	"github.com/ethereum/go-ethereum/trie"
)

type Block interface {
	Trie() *trie.Trie
	Diff() *big.Int
}
