package log

import (
	"math/rand"
	"testing"
)

var sink []byte

func BenchmarkPrettyInt64Logfmt(b *testing.B) {
	buf := make([]byte, 100)
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		sink = appendInt64(buf, rand.Int63())
	}
}

func BenchmarkPrettyUint64Logfmt(b *testing.B) {
	buf := make([]byte, 100)
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		sink = appendUint64(buf, rand.Uint64(), false)
	}
}
