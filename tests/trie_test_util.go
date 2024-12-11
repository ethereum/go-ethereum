package tests

import (
	"fmt"
	"strings"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/trie"
)

type TrieTest struct {
	In   [][]string  `json:"in"`
	Root common.Hash `json:"root"`
}

func (tt *TrieTest) Run(secure bool) error {
	tr := trie.NewEmpty(nil)
	for _, slices := range tt.In {
		key := []byte(slices[0])
		val := []byte(slices[1])

		if strings.HasPrefix(slices[0], "0x") {
			key, _ = FromHex(slices[0])
		}
		if secure {
			key = crypto.Keccak256(key)
		}
		if strings.HasPrefix(slices[1], "0x") {
			val, _ = FromHex(slices[1])
		}
		tr.Update(key, val)
	}
	if have, want := tr.Hash(), tt.Root; have != want {
		return fmt.Errorf("root mismatch: have %#x want %#x", have, want)
	}
	return nil
}
