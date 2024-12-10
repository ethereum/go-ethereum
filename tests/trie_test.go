package tests

import (
	"testing"

	"github.com/ethereum/go-ethereum/params"
)

func TestTrie(t *testing.T) {
	t.Parallel()

	tm := new(testMatcher)

	tm.skipLoad("hex_encoded_securetrie_test.json")
	tm.skipLoad("trieanyorder_secureTrie.json")
	tm.skipLoad("trieanyorder.json")
	tm.skipLoad("trietest_secureTrie.json")
	tm.skipLoad("trietestnextprev.json")

	tm.walk(t, trieTestDir, func(t *testing.T, name string, test *TrieTest) {
		cfg := params.MainnetChainConfig
		if err := tm.checkFailure(t, test.Run(cfg)); err != nil {
			t.Error(err)
		}
	})
}
