package log

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"math/big"
	"strings"
	"testing"
	"time"

	"github.com/holiman/uint256"
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
	// "INFO [01-01|00:00:00.000] a message ..." -> "a message..."
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
	// "INFO [01-01|00:00:00.000] a message ..." -> "a message..."
	have = strings.Split(have, "]")[1]
	want := " a message                                baz=bat foo=bar\n"
	if have != want {
		t.Errorf("\nhave: %q\nwant: %q\n", have, want)
	}
}

// Make sure the default json handler outputs debug log lines
func TestJSONHandler(t *testing.T) {
	out := new(bytes.Buffer)
	handler := JSONHandler(out)
	logger := slog.New(handler)
	logger.Debug("hi there")
	if len(out.String()) == 0 {
		t.Error("expected non-empty debug log output from default JSON Handler")
	}

	out.Reset()
	handler = JSONHandlerWithLevel(out, slog.LevelInfo)
	logger = slog.New(handler)
	logger.Debug("hi there")
	if len(out.String()) != 0 {
		t.Errorf("expected empty debug log output, but got: %v", out.String())
	}
}

func BenchmarkTraceLogging(b *testing.B) {
	SetDefault(NewLogger(NewTerminalHandler(io.Discard, true)))
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		Trace("a message", "v", i)
	}
}

func BenchmarkTerminalHandler(b *testing.B) {
	l := NewLogger(NewTerminalHandler(io.Discard, false))
	benchmarkLogger(b, l)
}
func BenchmarkLogfmtHandler(b *testing.B) {
	l := NewLogger(LogfmtHandler(io.Discard))
	benchmarkLogger(b, l)
}

func BenchmarkJSONHandler(b *testing.B) {
	l := NewLogger(JSONHandler(io.Discard))
	benchmarkLogger(b, l)
}

func benchmarkLogger(b *testing.B, l Logger) {
	var (
		bb     = make([]byte, 10)
		tt     = time.Now()
		bigint = big.NewInt(100)
		nilbig *big.Int
		err    = errors.New("oh nooes it's crap")
	)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		l.Info("This is a message",
			"foo", int16(i),
			"bytes", bb,
			"bonk", "a string with text",
			"time", tt,
			"bigint", bigint,
			"nilbig", nilbig,
			"err", err)
	}
	b.StopTimer()
}

func TestLoggerOutput(t *testing.T) {
	type custom struct {
		A string
		B int8
	}
	var (
		customA   = custom{"Foo", 12}
		customB   = custom{"Foo\nLinebreak", 122}
		bb        = make([]byte, 10)
		tt        = time.Time{}
		bigint    = big.NewInt(100)
		nilbig    *big.Int
		err       = errors.New("oh nooes it's crap")
		smallUint = uint256.NewInt(500_000)
		bigUint   = &uint256.Int{0xff, 0xff, 0xff, 0xff}
	)

	out := new(bytes.Buffer)
	glogHandler := NewGlogHandler(NewTerminalHandler(out, false))
	glogHandler.Verbosity(LevelInfo)
	NewLogger(glogHandler).Info("This is a message",
		"foo", int16(123),
		"bytes", bb,
		"bonk", "a string with text",
		"time", tt,
		"bigint", bigint,
		"nilbig", nilbig,
		"err", err,
		"struct", customA,
		"struct", customB,
		"ptrstruct", &customA,
		"smalluint", smallUint,
		"bigUint", bigUint)

	have := out.String()
	t.Logf("output %v", out.String())
	want := `INFO [11-07|19:14:33.821] This is a message                        foo=123 bytes="[0 0 0 0 0 0 0 0 0 0]" bonk="a string with text" time=0001-01-01T00:00:00+0000 bigint=100 nilbig=<nil> err="oh nooes it's crap" struct="{A:Foo B:12}" struct="{A:Foo\nLinebreak B:122}" ptrstruct="&{A:Foo B:12}" smalluint=500,000 bigUint=1,600,660,942,523,603,594,864,898,306,482,794,244,293,965,082,972,225,630,372,095
`
	if !bytes.Equal([]byte(have)[25:], []byte(want)[25:]) {
		t.Errorf("Error\nhave: %q\nwant: %q", have, want)
	}
}

const termTimeFormat = "01-02|15:04:05.000"

func BenchmarkAppendFormat(b *testing.B) {
	var now = time.Now()
	b.Run("fmt time.Format", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			fmt.Fprintf(io.Discard, "%s", now.Format(termTimeFormat))
		}
	})
	b.Run("time.AppendFormat", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			now.AppendFormat(nil, termTimeFormat)
		}
	})
	var buf = new(bytes.Buffer)
	b.Run("time.Custom", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			writeTimeTermFormat(buf, now)
			buf.Reset()
		}
	})
}

func TestTermTimeFormat(t *testing.T) {
	var now = time.Now()
	want := now.AppendFormat(nil, termTimeFormat)
	var b = new(bytes.Buffer)
	writeTimeTermFormat(b, now)
	have := b.Bytes()
	if !bytes.Equal(have, want) {
		t.Errorf("have != want\nhave: %q\nwant: %q\n", have, want)
	}
}
