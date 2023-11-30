package log

import (
	"bytes"
	"os"
	"strings"
	"testing"

	"golang.org/x/exp/slog"
)

// TestLoggingWithVmodule checks that vmodule works.
func TestLoggingWithVmodule(t *testing.T) {
	out := new(bytes.Buffer)
	glog := NewGlogHandler(NewTerminalHandlerWithLevel(out, LevelTrace, false))
	glog.Verbosity(LevelCrit)
	logger := NewLogger(glog)
	logger.Warn("This should not be seen", "ignored", "true")
	glog.Vmodule("logger_test.go=5")
	logger.Trace("a message", "foo", "bar")
	have := out.String()
	// The timestamp is locale-dependent, so we want to trim that off
	// "INFO [01-01|00:00:00.000] a messag ..." -> "a messag..."
	have = strings.Split(have, "]")[1]
	want := " a message                                foo=bar\n"
	if have != want {
		t.Errorf("\nhave: %q\nwant: %q\n", have, want)
	}
}

func TestTerminalHandlerWithAttrs(t *testing.T) {
	out := new(bytes.Buffer)
	glog := NewGlogHandler(NewTerminalHandlerWithLevel(out, LevelTrace, false).WithAttrs([]slog.Attr{slog.String("baz", "bat")}))
	glog.Verbosity(LevelTrace)
	logger := NewLogger(glog)
	logger.Trace("a message", "foo", "bar")
	have := out.String()
	// The timestamp is locale-dependent, so we want to trim that off
	// "INFO [01-01|00:00:00.000] a messag ..." -> "a messag..."
	have = strings.Split(have, "]")[1]
	want := " a message                                baz=bat foo=bar\n"
	if have != want {
		t.Errorf("\nhave: %q\nwant: %q\n", have, want)
	}
}

func BenchmarkTraceLogging(b *testing.B) {
	SetDefault(NewLogger(NewTerminalHandler(os.Stderr, true)))
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		Trace("a message", "v", i)
	}
}
