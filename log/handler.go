package log

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"math/big"
	"reflect"
	"sync"
	"time"

	"github.com/holiman/uint256"
)

type discardHandler struct{}

// DiscardHandler returns a no-op handler
func DiscardHandler() slog.Handler {
	return &discardHandler{}
}

func (h *discardHandler) Handle(_ context.Context, r slog.Record) error {
	return nil
}

func (h *discardHandler) Enabled(_ context.Context, level slog.Level) bool {
	return false
}

func (h *discardHandler) WithGroup(name string) slog.Handler {
	panic("not implemented")
}

func (h *discardHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return &discardHandler{}
}

type TerminalHandler struct {
	mu       sync.Mutex
	wr       io.Writer
	lvl      slog.Level
	useColor bool
	attrs    []slog.Attr
	// fieldPadding is a map with maximum field value lengths seen until now
	// to allow padding log contexts in a bit smarter way.
	fieldPadding map[string]int

	buf []byte
}

// NewTerminalHandler returns a handler which formats log records at all levels optimized for human readability on
// a terminal with color-coded level output and terser human friendly timestamp.
// This format should only be used for interactive programs or while developing.
//
//	[LEVEL] [TIME] MESSAGE key=value key=value ...
//
// Example:
//
//	[DBUG] [May 16 20:58:45] remove route ns=haproxy addr=127.0.0.1:50002
func NewTerminalHandler(wr io.Writer, useColor bool) *TerminalHandler {
	return NewTerminalHandlerWithLevel(wr, levelMaxVerbosity, useColor)
}

// NewTerminalHandlerWithLevel returns the same handler as NewTerminalHandler but only outputs
// records which are less than or equal to the specified verbosity level.
func NewTerminalHandlerWithLevel(wr io.Writer, lvl slog.Level, useColor bool) *TerminalHandler {
	return &TerminalHandler{
		wr:           wr,
		lvl:          lvl,
		useColor:     useColor,
		fieldPadding: make(map[string]int),
	}
}

func (h *TerminalHandler) Handle(_ context.Context, r slog.Record) error {
	h.mu.Lock()
	defer h.mu.Unlock()
	buf := h.format(h.buf, r, h.useColor)
	h.wr.Write(buf)
	h.buf = buf[:0]
	return nil
}

func (h *TerminalHandler) Enabled(_ context.Context, level slog.Level) bool {
	return level >= h.lvl
}

func (h *TerminalHandler) WithGroup(name string) slog.Handler {
	panic("not implemented")
}

func (h *TerminalHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return &TerminalHandler{
		wr:           h.wr,
		lvl:          h.lvl,
		useColor:     h.useColor,
		attrs:        append(h.attrs, attrs...),
		fieldPadding: make(map[string]int),
	}
}

// ResetFieldPadding zeroes the field-padding for all attribute pairs.
func (h *TerminalHandler) ResetFieldPadding() {
	h.mu.Lock()
	h.fieldPadding = make(map[string]int)
	h.mu.Unlock()
}

type leveler struct{ minLevel slog.Level }

func (l *leveler) Level() slog.Level {
	return l.minLevel
}

// JSONHandler returns a handler which prints records in JSON format.
func JSONHandler(wr io.Writer) slog.Handler {
	return JSONHandlerWithLevel(wr, levelMaxVerbosity)
}

// JSONHandlerWithLevel returns a handler which prints records in JSON format that are less than or equal to
// the specified verbosity level.
func JSONHandlerWithLevel(wr io.Writer, level slog.Level) slog.Handler {
	return slog.NewJSONHandler(wr, &slog.HandlerOptions{
		ReplaceAttr: builtinReplaceJSON,
		Level:       &leveler{level},
	})
}

// LogfmtHandler returns a handler which prints records in logfmt format, an easy machine-parseable but human-readable
// format for key/value pairs.
//
// For more details see: http://godoc.org/github.com/kr/logfmt
func LogfmtHandler(wr io.Writer) slog.Handler {
	return slog.NewTextHandler(wr, &slog.HandlerOptions{
		ReplaceAttr: builtinReplaceLogfmt,
	})
}

// LogfmtHandlerWithLevel returns the same handler as LogfmtHandler but it only outputs
// records which are less than or equal to the specified verbosity level.
func LogfmtHandlerWithLevel(wr io.Writer, level slog.Level) slog.Handler {
	return slog.NewTextHandler(wr, &slog.HandlerOptions{
		ReplaceAttr: builtinReplaceLogfmt,
		Level:       &leveler{level},
	})
}

func builtinReplaceLogfmt(_ []string, attr slog.Attr) slog.Attr {
	return builtinReplace(nil, attr, true)
}

func builtinReplaceJSON(_ []string, attr slog.Attr) slog.Attr {
	return builtinReplace(nil, attr, false)
}

func builtinReplace(_ []string, attr slog.Attr, logfmt bool) slog.Attr {
	switch attr.Key {
	case slog.TimeKey:
		if attr.Value.Kind() == slog.KindTime {
			if logfmt {
				return slog.String("t", attr.Value.Time().Format(timeFormat))
			} else {
				return slog.Attr{Key: "t", Value: attr.Value}
			}
		}
	case slog.LevelKey:
		if l, ok := attr.Value.Any().(slog.Level); ok {
			attr = slog.Any("lvl", LevelString(l))
			return attr
		}
	}

	switch v := attr.Value.Any().(type) {
	case time.Time:
		if logfmt {
			attr = slog.String(attr.Key, v.Format(timeFormat))
		}
	case *big.Int:
		if v == nil {
			attr.Value = slog.StringValue("<nil>")
		} else {
			attr.Value = slog.StringValue(v.String())
		}
	case *uint256.Int:
		if v == nil {
			attr.Value = slog.StringValue("<nil>")
		} else {
			attr.Value = slog.StringValue(v.Dec())
		}
	case fmt.Stringer:
		if v == nil || (reflect.ValueOf(v).Kind() == reflect.Pointer && reflect.ValueOf(v).IsNil()) {
			attr.Value = slog.StringValue("<nil>")
		} else {
			attr.Value = slog.StringValue(v.String())
		}
	}
	return attr
}
