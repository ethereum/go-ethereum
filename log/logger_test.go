package log

import (
	"os"
	"testing"
)

func BenchmarkTraceLogging(b *testing.B) {
	Root().SetHandler(LvlFilterHandler(LvlInfo, StreamHandler(os.Stderr, TerminalFormat(true))))
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		Trace("a message", "v", i)
	}
}
