package types

import (
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/ethereum/go-ethereum/trie"
)

type DerivableList interface {
	Len() int
	GetRlp(i int) []byte
}

func DeriveSha(list DerivableList) common.Hash {
	db, _ := ethdb.NewMemDatabase()
	trie := trie.New(nil, db)
	for i := 0; i < list.Len(); i++ {
		key, _ := rlp.EncodeToBytes(uint(i))
		trie.Update(key, list.GetRlp(i))
	}

	return common.BytesToHash(trie.Root())
}
