package log

import (
	"bytes"
	"fmt"
	"math/big"
	"reflect"
	"strconv"
	"time"
	"unicode/utf8"

	"github.com/holiman/uint256"
	"golang.org/x/exp/slog"
)

const (
	timeFormat        = "2006-01-02T15:04:05-0700"
	termTimeFormat    = "01-02|15:04:05.000"
	floatFormat       = 'f'
	termMsgJust       = 40
	termCtxMaxPadding = 40
)

type Format interface {
	Format(r slog.Record) []byte
}

// FormatFunc returns a new Format object which uses
// the given function to perform record formatting.
func FormatFunc(f func(slog.Record) []byte) Format {
	return formatFunc(f)
}

type formatFunc func(slog.Record) []byte

func (f formatFunc) Format(r slog.Record) []byte {
	return f(r)
}

// TerminalStringer is an analogous interface to the stdlib stringer, allowing
// own types to have custom shortened serialization formats when printed to the
// screen.
type TerminalStringer interface {
	TerminalString() string
}

func (h *TerminalHandler) TerminalFormat(r slog.Record, usecolor bool) []byte {
	msg := escapeMessage(r.Message)
	var color = 0
	if usecolor {
		switch r.Level {
		case LevelCrit:
			color = 35
		case slog.LevelError:
			color = 31
		case slog.LevelWarn:
			color = 33
		case slog.LevelInfo:
			color = 32
		case slog.LevelDebug:
			color = 36
		case LevelTrace:
			color = 34
		}
	}

	b := &bytes.Buffer{}
	lvl := LevelAlignedString(r.Level)
	if color > 0 {
		fmt.Fprintf(b, "\x1b[%dm%s\x1b[0m[%s] %s ", color, lvl, r.Time.Format(termTimeFormat), msg)
	} else {
		fmt.Fprintf(b, "%s[%s] %s ", lvl, r.Time.Format(termTimeFormat), msg)
	}
	// try to justify the log output for short messages
	length := utf8.RuneCountInString(msg)
	if r.NumAttrs() > 0 && length < termMsgJust {
		b.Write(bytes.Repeat([]byte{' '}, termMsgJust-length))
	}
	// print the keys logfmt style
	h.logfmt(b, r, color)

	return b.Bytes()
}

func (h *TerminalHandler) logfmt(buf *bytes.Buffer, r slog.Record, color int) {
	attrs := []slog.Attr{}
	r.Attrs(func(attr slog.Attr) bool {
		attrs = append(attrs, attr)
		return true
	})

	attrs = append(h.attrs, attrs...)

	for i, attr := range attrs {
		if i != 0 {
			buf.WriteByte(' ')
		}

		key := escapeString(attr.Key)
		rawVal := attr.Value.Any()
		val := FormatLogfmtValue(rawVal, true)

		// XXX: we should probably check that all of your key bytes aren't invalid
		// TODO (jwasinger) above comment was from log15 code.  what does it mean?  check that key bytes are ascii characters?
		padding := h.fieldPadding[key]

		length := utf8.RuneCountInString(val)
		if padding < length && length <= termCtxMaxPadding {
			padding = length
			h.fieldPadding[key] = padding
		}
		if color > 0 {
			fmt.Fprintf(buf, "\x1b[%dm%s\x1b[0m=", color, key)
		} else {
			buf.WriteString(key)
			buf.WriteByte('=')
		}
		buf.WriteString(val)
		if i < r.NumAttrs()-1 && padding > length {
			buf.Write(bytes.Repeat([]byte{' '}, padding-length))
		}
	}
	buf.WriteByte('\n')
}

// formatValue formats a value for serialization
func FormatLogfmtValue(value interface{}, term bool) (result string) {
	if value == nil {
		return "<nil>"
	}
	defer func() {
		if err := recover(); err != nil {
			if v := reflect.ValueOf(value); v.Kind() == reflect.Ptr && v.IsNil() {
				result = "<nil>"
			} else {
				panic(err)
			}
		}
	}()

	switch v := value.(type) {
	case time.Time:
		// Performance optimization: No need for escaping since the provided
		// timeFormat doesn't have any escape characters, and escaping is
		// expensive.
		return v.Format(timeFormat)

	case *big.Int:
		// Big ints get consumed by the Stringer clause, so we need to handle
		// them earlier on.
		if v == nil {
			return "<nil>"
		}
		return formatLogfmtBigInt(v)

	case *uint256.Int:
		// Uint256s get consumed by the Stringer clause, so we need to handle
		// them earlier on.
		if v == nil {
			return "<nil>"
		}
		return formatLogfmtUint256(v)
	}
	if term {
		if s, ok := value.(TerminalStringer); ok {
			// Custom terminal stringer provided, use that
			return escapeString(s.TerminalString())
		}
	}
	switch v := value.(type) {
	case error:
		return escapeString(v.Error())
	case fmt.Stringer:
		return escapeString(v.String())
	case bool:
		return strconv.FormatBool(v)
	case float32:
		return strconv.FormatFloat(float64(v), floatFormat, 3, 64)
	case float64:
		return strconv.FormatFloat(v, floatFormat, 3, 64)
	case int8:
		return strconv.FormatInt(int64(v), 10)
	case uint8:
		return strconv.FormatInt(int64(v), 10)
	case int16:
		return strconv.FormatInt(int64(v), 10)
	case uint16:
		return strconv.FormatInt(int64(v), 10)
	// Larger integers get thousands separators.
	case int:
		return FormatLogfmtInt64(int64(v))
	case int32:
		return FormatLogfmtInt64(int64(v))
	case int64:
		return FormatLogfmtInt64(v)
	case uint:
		return FormatLogfmtUint64(uint64(v))
	case uint32:
		return FormatLogfmtUint64(uint64(v))
	case uint64:
		return FormatLogfmtUint64(v)
	case string:
		return escapeString(v)
	default:
		return escapeString(fmt.Sprintf("%+v", value))
	}
}

// FormatLogfmtInt64 formats n with thousand separators.
func FormatLogfmtInt64(n int64) string {
	if n < 0 {
		return formatLogfmtUint64(uint64(-n), true)
	}
	return formatLogfmtUint64(uint64(n), false)
}

// FormatLogfmtUint64 formats n with thousand separators.
func FormatLogfmtUint64(n uint64) string {
	return formatLogfmtUint64(n, false)
}

func formatLogfmtUint64(n uint64, neg bool) string {
	// Small numbers are fine as is
	if n < 100000 {
		if neg {
			return strconv.Itoa(-int(n))
		} else {
			return strconv.Itoa(int(n))
		}
	}
	// Large numbers should be split
	const maxLength = 26

	var (
		out   = make([]byte, maxLength)
		i     = maxLength - 1
		comma = 0
	)
	for ; n > 0; i-- {
		if comma == 3 {
			comma = 0
			out[i] = ','
		} else {
			comma++
			out[i] = '0' + byte(n%10)
			n /= 10
		}
	}
	if neg {
		out[i] = '-'
		i--
	}
	return string(out[i+1:])
}

// formatLogfmtBigInt formats n with thousand separators.
func formatLogfmtBigInt(n *big.Int) string {
	if n.IsUint64() {
		return FormatLogfmtUint64(n.Uint64())
	}
	if n.IsInt64() {
		return FormatLogfmtInt64(n.Int64())
	}

	var (
		text  = n.String()
		buf   = make([]byte, len(text)+len(text)/3)
		comma = 0
		i     = len(buf) - 1
	)
	for j := len(text) - 1; j >= 0; j, i = j-1, i-1 {
		c := text[j]

		switch {
		case c == '-':
			buf[i] = c
		case comma == 3:
			buf[i] = ','
			i--
			comma = 0
			fallthrough
		default:
			buf[i] = c
			comma++
		}
	}
	return string(buf[i+1:])
}

// formatLogfmtUint256 formats n with thousand separators.
func formatLogfmtUint256(n *uint256.Int) string {
	if n.IsUint64() {
		return FormatLogfmtUint64(n.Uint64())
	}
	var (
		text  = n.Dec()
		buf   = make([]byte, len(text)+len(text)/3)
		comma = 0
		i     = len(buf) - 1
	)
	for j := len(text) - 1; j >= 0; j, i = j-1, i-1 {
		c := text[j]

		switch {
		case c == '-':
			buf[i] = c
		case comma == 3:
			buf[i] = ','
			i--
			comma = 0
			fallthrough
		default:
			buf[i] = c
			comma++
		}
	}
	return string(buf[i+1:])
}

// escapeString checks if the provided string needs escaping/quoting, and
// calls strconv.Quote if needed
func escapeString(s string) string {
	needsQuoting := false
	for _, r := range s {
		// We quote everything below " (0x22) and above~ (0x7E), plus equal-sign
		if r <= '"' || r > '~' || r == '=' {
			needsQuoting = true
			break
		}
	}
	if !needsQuoting {
		return s
	}
	return strconv.Quote(s)
}

// escapeMessage checks if the provided string needs escaping/quoting, similarly
// to escapeString. The difference is that this method is more lenient: it allows
// for spaces and linebreaks to occur without needing quoting.
func escapeMessage(s string) string {
	needsQuoting := false
	for _, r := range s {
		// Allow CR/LF/TAB. This is to make multi-line messages work.
		if r == '\r' || r == '\n' || r == '\t' {
			continue
		}
		// We quote everything below <space> (0x20) and above~ (0x7E),
		// plus equal-sign
		if r < ' ' || r > '~' || r == '=' {
			needsQuoting = true
			break
		}
	}
	if !needsQuoting {
		return s
	}
	return strconv.Quote(s)
}
