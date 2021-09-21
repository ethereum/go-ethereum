package tradingstate

import (
	"fmt"
	"github.com/XinFinOrg/XDPoSChain/common"
	"github.com/XinFinOrg/XDPoSChain/core/rawdb"
	"math/big"
	"math/rand"
	"testing"
	"time"
)

func TestXDCxTrieTest(t *testing.T) {
	t.SkipNow()
	db := rawdb.NewMemoryDatabase()
	stateCache := NewDatabase(db)
	trie, _ := stateCache.OpenStorageTrie(EmptyHash, EmptyHash)
	max := 1000000
	for i := 1; i < max; i++ {
		k := common.BigToHash(big.NewInt(int64(i))).Bytes()
		trie.TryUpdate(k, k)
	}
	left, _, _ := trie.TryGetBestLeftKeyAndValue()
	right, _, _ := trie.TryGetBestRightKeyAndValue()
	fmt.Println(left, right)
	for i := 0; i < 100; i++ {
		limit := big.NewInt(rand.Int63n(int64(max / 10)))
		fmt.Println(time.Now().Format("2006-01-02 15:04:05"), "test", i, limit)
		allKeyLefts, allValues, err := trie.TryGetAllLeftKeyAndValue(common.BigToHash(limit).Bytes())
		if err != nil {
			t.Fatal("err", err)
		}
		if len(allKeyLefts) != int(limit.Int64())-1 {
			t.Fatal("err when length", len(allKeyLefts), "limit", limit)
		}
		for j := 0; j < len(allKeyLefts); j++ {
			key := new(big.Int).SetBytes(allKeyLefts[j])
			value := new(big.Int).SetBytes(allValues[j])
			if key.Cmp(value) != 0 {
				t.Fatal("err when compare key", key, "value", value)
			}
			if key.Cmp(limit) >= 0 {
				t.Fatal("err when compare key", key, "limit", limit)
			}
		}
	}
}
