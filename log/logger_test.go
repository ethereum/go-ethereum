package log

import (
	"bytes"
	"os"
	"strings"
	"testing"
	"time"
)

func BenchmarkTraceLogging(b *testing.B) {
	Root().SetHandler(LvlFilterHandler(LvlInfo, StreamHandler(os.Stderr, TerminalFormat(true))))
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		Trace("a message", "v", i)
	}
}

type notimeHandler struct {
	next Handler
}

func (n notimeHandler) Log(r *Record) error {
	r.Time = time.Unix(0, 0)
	return n.next.Log(r)
}

// TestLoggingWithTrace checks that if BackTraceAt is set, then the
// gloghandler is capable of spitting out a stacktrace
func TestLoggingWithTrace(t *testing.T) {
	defer stackEnabled.Store(stackEnabled.Load())
	out := new(bytes.Buffer)
	logger := New()
	{
		glog := NewGlogHandler(StreamHandler(out, TerminalFormat(false)))
		glog.Verbosity(LvlTrace)
		if err := glog.BacktraceAt("logger_test.go:42"); err != nil {
			t.Fatal(err)
		}
		logger.SetHandler(notimeHandler{glog})
	}
	logger.Trace("a message", "foo", "bar") // Will be bumped to INFO
	have := out.String()
	wantPrefix := "INFO [01-01|01:00:00.000] a message\n\ngoroutine"
	if len(have) < len(wantPrefix) || !strings.HasPrefix(have, wantPrefix) {
		t.Errorf("\nhave: %q\nwant: %q\n", have, wantPrefix)
	}
}

// TestLoggingWithVmodule checks that vmodule works.
func TestLoggingWithVmodule(t *testing.T) {
	defer stackEnabled.Store(stackEnabled.Load())
	out := new(bytes.Buffer)
	logger := New()
	{
		glog := NewGlogHandler(StreamHandler(out, TerminalFormat(false)))
		glog.Verbosity(LvlCrit)
		logger.SetHandler(notimeHandler{glog})
		logger.Warn("This should not be seen", "ignored", "true")
		glog.Vmodule("logger_test.go=5")
	}
	logger.Trace("a message", "foo", "bar")
	have := out.String()
	want := "TRACE[01-01|01:00:00.000] a message                                foo=bar\n"
	if have != want {
		t.Errorf("\nhave: %q\nwant: %q\n", have, want)
	}
}
