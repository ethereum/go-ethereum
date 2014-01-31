package main

import (
	"encoding/hex"
	_ "fmt"
	"github.com/ethereum/ethdb-go"
	"github.com/ethereum/ethutil-go"
	"testing"
)

var testsource = `
{
	"inputs":{
		"doe": "reindeer",
		"dog": "puppy",
		"dogglesworth": "cat"
	},
	"expectation":"e378927bfc1bd4f01a2e8d9f59bd18db8a208bb493ac0b00f93ce51d4d2af76c"
}`

func TestTestRunner(t *testing.T) {
	db, _ := ethdb.NewMemDatabase()
	trie := ethutil.NewTrie(db, "")

	runner := NewTestRunner(t)
	runner.RunFromString(testsource, func(source *TestSource) {
		for key, value := range source.Inputs {
			trie.Update(key, value)
		}
		if hex.EncodeToString(trie.Root.([]byte)) != source.Expectation {
			t.Error("trie root did not match")
		}
	})
}
