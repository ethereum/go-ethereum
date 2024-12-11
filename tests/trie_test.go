package tests

import (
	"strings"
	"testing"
)

func TestTrie(t *testing.T) {
	t.Parallel()

	tm := new(testMatcher)

	tm.skipLoad("hex_encoded_securetrie_test.json")
	tm.skipLoad("trieanyorder_secureTrie.json")
	tm.skipLoad("trieanyorder.json")
	tm.skipLoad("trietestnextprev.json")

	tm.walk(t, trieTestDir, func(t *testing.T, name string, test *TrieTest) {
		secure := strings.Contains(name, "secure")
		if err := tm.checkFailure(t, test.Run(secure)); err != nil {
			t.Error(err)
		}
	})
}
