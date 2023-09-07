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

func TestLoggingNoTrace(t *testing.T) {
	out := new(bytes.Buffer)
	logger := New()
	{
		glog := NewGlogHandler(StreamHandler(out, TerminalFormat(false)))
		glog.Verbosity(LvlTrace)
		if err := glog.BacktraceAt("logger_test.go:38"); err != nil {
			t.Fatal(err)
		}
		logger.SetHandler(notimeHandler{glog})
	}
	logger.Trace("a message", "foo", "bar")
	have := out.String()
	want := `TRACE[01-01|01:00:00.000] a message                                foo=bar
`
	if have != want {
		t.Errorf("\nhave: '%v'\nwant: '%v'\n", have, want)
	}
}

func TestLoggingWithTrace(t *testing.T) {
	PrintOrigins(true)
	defer PrintOrigins(false)
	out := new(bytes.Buffer)
	logger := New()
	{
		glog := NewGlogHandler(StreamHandler(out, TerminalFormat(false)))
		glog.Verbosity(LvlTrace)
		if err := glog.BacktraceAt("logger_test.go:59"); err != nil {
			t.Fatal(err)
		}
		logger.SetHandler(notimeHandler{glog})
	}
	logger.Trace("a message", "foo", "bar")
	have := out.String()
	wantPrefix := `INFO [01-01|01:00:00.000|log/logger_test.go:59] a message
        
        goroutine`
	if len(have) < len(wantPrefix) || strings.HasPrefix(have, wantPrefix) {
		t.Errorf("\nhave: '%v'\nwant: '%v'\n", have, wantPrefix)
	}
}
