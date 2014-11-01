package ar

import (
	"fmt"
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/trie"
)

type TestBlock struct {
	trie *trie.Trie
}

func NewTestBlock() *TestBlock {
	db, _ := ethdb.NewMemDatabase()
	return &TestBlock{
		trie: trie.New(db, ""),
	}
}

func (self *TestBlock) Diff() *big.Int {
	return b(10)
}

func (self *TestBlock) Trie() *trie.Trie {
	return self.trie
}

func (self *TestBlock) Hash() []byte {
	a := make([]byte, 32)
	a[0] = 10
	a[1] = 2
	return a
}

func TestPow(t *testing.T) {
	entry := make([]byte, 32)
	entry[0] = 255

	block := NewTestBlock()

	pow := NewTape(block)
	nonce := pow.Run(block.Hash())
	fmt.Println("Found nonce", nonce)
}
