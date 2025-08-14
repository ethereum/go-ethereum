package rawdb

import (
	"math/rand"
	"testing"

	"github.com/ethereum/go-ethereum/common"
)

var sink []byte

func BenchmarkStorageHistoryIndexBlockKey(b *testing.B) {
	var h1, h2 common.Hash
	for i := range h1 {
		h1[i] = byte(rand.Intn(256))
		h2[i] = byte(rand.Intn(256))
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		sink = storageHistoryIndexBlockKey(h1, h2, uint32(i))
	}
}
