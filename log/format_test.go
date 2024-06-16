package log

import (
	"math/rand"
	"testing"
)

var sink []byte

// Benchmark for formatting int64 using logfmt
func BenchmarkPrettyInt64Logfmt(b *testing.B) {
	buf := make([]byte, 100)
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		sink = appendInt64(buf, rand.Int63())
	}
}

// Benchmark for formatting uint64 using logfmt
func BenchmarkPrettyUint64Logfmt(b *testing.B) {
	buf := make([]byte, 100)
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		sink = appendUint64(buf, rand.Uint64(), false)
	}
}
