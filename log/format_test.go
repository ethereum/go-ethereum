package log

import (
	"math/rand/v2"
	"testing"
)

var sink []byte

func BenchmarkPrettyInt64Logfmt(b *testing.B) {
	buf := make([]byte, 100)
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		sink = appendInt64(buf, rand.Int64())
	}
}

func BenchmarkPrettyUint64Logfmt(b *testing.B) {
	buf := make([]byte, 100)
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		sink = appendUint64(buf, rand.Uint64(), false)
	}
}
