package ar

import (
	"math/big"
	"github.com/ethereum/eth-go/ethtrie"
)

type Block interface {
	Trie() *ethtrie.Trie
	Diff() *big.Int
}
