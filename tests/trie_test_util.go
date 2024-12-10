package tests

import (
	"fmt"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/params"
	"github.com/ethereum/go-ethereum/trie"
)

//go:generate go run github.com/fjl/gencodec -type TrieTest -field-override trieTestMarshaling -out gen_trietest.go

type TrieTest struct {
	In   [][]string  `json:"in"`
	Root common.Hash `json:"root"`
}

type trieTestMarshaling struct {
	In   [][]string  `json:"in"`
	Root common.Hash `json:"root"`
}

func (tt *TrieTest) Run(config *params.ChainConfig) error {
	// dbConf := new(triedb.Config)
	// tdb := triedb.NewDatabase(rawdb.NewMemoryDatabase(), dbConf)
	// trie := trie.NewEmpty(tdb)
	id := &trie.ID{
		Root: tt.Root,
	}

	for _, slices := range tt.In {
		slices := slices
		if len(slices) == 0 {
			return fmt.Errorf("empty input")
		}

		for _, v := range slices {
			id.Owner = common.HexToHash(v)
			tr, _ := trie.New(id, nil)
			actual := tr.Hash()

			if id.Root != actual {
				return fmt.Errorf("root hash mismatch: %s != %s", id.Root.Hex(), actual.Hex())
			}
		}
	}
	return nil
}
