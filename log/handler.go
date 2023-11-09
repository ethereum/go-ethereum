package log

import (
	"context"
	"fmt"
	"io"
	"math/big"
	"reflect"
	"sync"
	"time"

	"github.com/holiman/uint256"
	"golang.org/x/exp/slog"
)

// Lazy allows you to defer calculation of a logged value that is expensive
// to compute until it is certain that it must be evaluated with the given filters.
//
// You may wrap any function which takes no arguments to Lazy. It may return any
// number of values of any type.
type Lazy struct {
	Fn interface{}
}

func evaluateLazy(lz Lazy) (interface{}, error) {
	t := reflect.TypeOf(lz.Fn)

	if t.Kind() != reflect.Func {
		return nil, fmt.Errorf("INVALID_LAZY, not func: %+v", lz.Fn)
	}

	if t.NumIn() > 0 {
		return nil, fmt.Errorf("INVALID_LAZY, func takes args: %+v", lz.Fn)
	}

	if t.NumOut() == 0 {
		return nil, fmt.Errorf("INVALID_LAZY, no func return val: %+v", lz.Fn)
	}

	value := reflect.ValueOf(lz.Fn)
	results := value.Call([]reflect.Value{})
	if len(results) == 1 {
		return results[0].Interface(), nil
	}
	values := make([]interface{}, len(results))
	for i, v := range results {
		values[i] = v.Interface()
	}
	return values, nil
}

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

type funcHandler struct {
	handler handlerFunc
	attrs   []slog.Attr
}

type handlerFunc func(_ context.Context, r slog.Record) error

// FuncHandler returns a handler which forwards all records to the provided handler function
func FuncHandler(fn handlerFunc) slog.Handler {
	return &funcHandler{fn, []slog.Attr{}}
}

func (h funcHandler) Handle(_ context.Context, r slog.Record) error {
	r.AddAttrs(h.attrs...)
	return h.handler(context.Background(), r)
}

func (h *funcHandler) Enabled(_ context.Context, level slog.Level) bool {
	return true
}

func (h *funcHandler) WithGroup(name string) slog.Handler {
	panic("not implemented")
}

func (h *funcHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return &funcHandler{
		h.handler,
		append(h.attrs, attrs...),
	}
}

type terminalHandler struct {
	mu       sync.Mutex
	wr       io.Writer
	lvl      slog.Level
	useColor bool
	attrs    []slog.Attr
}

// TerminalHandler returns a handler which formats log records at all levels optimized for human readability on
// a terminal with color-coded level output and terser human friendly timestamp.
// This format should only be used for interactive programs or while developing.
//
//	[LEVEL] [TIME] MESSAGE key=value key=value ...
//
// Example:
//
//	[DBUG] [May 16 20:58:45] remove route ns=haproxy addr=127.0.0.1:50002
func TerminalHandler(wr io.Writer, useColor bool) slog.Handler {
	return TerminalHandlerWithLevel(wr, levelMaxVerbosity, useColor)
}

// TerminalHandlerWithLevel returns the same handler as TerminalHandler but only outputs
// records which are less than or equal to the specified verbosity level.
func TerminalHandlerWithLevel(wr io.Writer, lvl slog.Level, useColor bool) slog.Handler {
	return &terminalHandler{
		sync.Mutex{},
		wr,
		lvl,
		useColor,
		[]slog.Attr{},
	}
}

func (h *terminalHandler) Handle(_ context.Context, r slog.Record) error {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.wr.Write(TerminalFormat(r, h.attrs, h.useColor))
	return nil
}

func (h *terminalHandler) Enabled(_ context.Context, level slog.Level) bool {
	return level >= h.lvl
}

func (h *terminalHandler) WithGroup(name string) slog.Handler {
	panic("not implemented")
}

func (h *terminalHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return &terminalHandler{
		sync.Mutex{},
		h.wr,
		h.lvl,
		h.useColor,
		append(h.attrs, attrs...),
	}
}

type leveler struct{ minLevel slog.Level }

func (l *leveler) Level() slog.Level {
	return l.minLevel
}

func JSONHandler(wr io.Writer) slog.Handler {
	return slog.NewJSONHandler(wr, &slog.HandlerOptions{
		ReplaceAttr: builtinReplace,
	})
}

// LogfmtHandler returns a handler which prints records in logfmt format, an easy machine-parseable but human-readable
// format for key/value pairs.
//
// For more details see: http://godoc.org/github.com/kr/logfmt
func LogfmtHandler(wr io.Writer) slog.Handler {
	return slog.NewTextHandler(wr, &slog.HandlerOptions{
		ReplaceAttr: builtinReplace,
	})
}

// LogfmtHandlerWithLevel returns the same handler as LogfmtHandler but it only outputs
// records which are less than or equal to the specified verbosity level.
func LogfmtHandlerWithLevel(wr io.Writer, level slog.Level) slog.Handler {
	return slog.NewTextHandler(wr, &slog.HandlerOptions{
		ReplaceAttr: builtinReplace,
		Level:       &leveler{level},
	})
}

func builtinReplace(_ []string, attr slog.Attr) slog.Attr {
	switch attr.Key {
	case slog.TimeKey:
		if attr.Value.Kind() == slog.KindTime {
			return slog.Time("t", attr.Value.Time())
		}
	case slog.LevelKey:
		if l, ok := attr.Value.Any().(slog.Level); ok {
			attr = slog.Any("lvl", LevelString(l))
			return attr
		}
	}

	switch v := attr.Value.Any().(type) {
	case time.Time:
		attr = slog.Any(attr.Key, v.Format(timeFormat))
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
	}
	return attr
}
