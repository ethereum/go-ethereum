package log

import (
	"math"
	"math/big"
	"math/rand"
	"testing"
)

func TestPrettyInt64(t *testing.T) {
	tests := []struct {
		n int64
		s string
	}{
		{0, "0"},
		{10, "10"},
		{-10, "-10"},
		{100, "100"},
		{-100, "-100"},
		{1000, "1000"},
		{-1000, "-1000"},
		{10000, "10000"},
		{-10000, "-10000"},
		{99999, "99999"},
		{-99999, "-99999"},
		{100000, "100,000"},
		{-100000, "-100,000"},
		{1000000, "1,000,000"},
		{-1000000, "-1,000,000"},
		{math.MaxInt64, "9,223,372,036,854,775,807"},
		{math.MinInt64, "-9,223,372,036,854,775,808"},
	}
	for i, tt := range tests {
		if have := FormatLogfmtInt64(tt.n); have != tt.s {
			t.Errorf("test %d: format mismatch: have %s, want %s", i, have, tt.s)
		}
	}
}

func TestPrettyUint64(t *testing.T) {
	tests := []struct {
		n uint64
		s string
	}{
		{0, "0"},
		{10, "10"},
		{100, "100"},
		{1000, "1000"},
		{10000, "10000"},
		{99999, "99999"},
		{100000, "100,000"},
		{1000000, "1,000,000"},
		{math.MaxUint64, "18,446,744,073,709,551,615"},
	}
	for i, tt := range tests {
		if have := FormatLogfmtUint64(tt.n); have != tt.s {
			t.Errorf("test %d: format mismatch: have %s, want %s", i, have, tt.s)
		}
	}
}

func TestPrettyBigInt(t *testing.T) {
	tests := []struct {
		int string
		s   string
	}{
		{"111222333444555678999", "111,222,333,444,555,678,999"},
		{"-111222333444555678999", "-111,222,333,444,555,678,999"},
		{"11122233344455567899900", "11,122,233,344,455,567,899,900"},
		{"-11122233344455567899900", "-11,122,233,344,455,567,899,900"},
	}

	for _, tt := range tests {
		v, _ := new(big.Int).SetString(tt.int, 10)
		if have := formatLogfmtBigInt(v); have != tt.s {
			t.Errorf("invalid output %s, want %s", have, tt.s)
		}
	}
}

var sink string

func BenchmarkPrettyInt64Logfmt(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		sink = FormatLogfmtInt64(rand.Int63())
	}
}

func BenchmarkPrettyUint64Logfmt(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		sink = FormatLogfmtUint64(rand.Uint64())
	}
}
