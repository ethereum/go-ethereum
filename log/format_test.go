package log

import (
	"log/slog"
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

// testStringer implements both TerminalStringer and fmt.Stringer
type testStringer struct{}

func (t testStringer) String() string {
	return "full string representation"
}

func (t testStringer) TerminalString() string {
	return "terminal string representation"
}

func TestFormatSlogValue(t *testing.T) {
	tests := []struct {
		name     string
		level    slog.Level
		value    slog.Value
		expected string
	}{
		{
			name:     "DEBUG level uses String()",
			level:    slog.LevelDebug,
			value:    slog.AnyValue(testStringer{}),
			expected: `"full string representation"`,
		},
		{
			name:     "TRACE level uses String()",
			level:    LevelTrace,
			value:    slog.AnyValue(testStringer{}),
			expected: `"full string representation"`,
		},
		{
			name:     "INFO level uses TerminalString()",
			level:    slog.LevelInfo,
			value:    slog.AnyValue(testStringer{}),
			expected: `"terminal string representation"`,
		},
		{
			name:     "WARN level uses TerminalString()",
			level:    slog.LevelWarn,
			value:    slog.AnyValue(testStringer{}),
			expected: `"terminal string representation"`,
		},
		{
			name:     "ERROR level uses TerminalString()",
			level:    slog.LevelError,
			value:    slog.AnyValue(testStringer{}),
			expected: `"terminal string representation"`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := FormatSlogValue(tt.level, tt.value, nil)
			if string(result) != tt.expected {
				t.Errorf("FormatSlogValue() = '%v', want '%v'", string(result), tt.expected)
			}
		})
	}
}
