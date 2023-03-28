package log

import (
	"fmt"
	"math"
	"math/big"
	"math/rand"
	"strings"
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

func TestSanitation(t *testing.T) {
	msg := "\u001b[1G\u001b[K\u001b[1A"
	msg2 := "\u001b  \u0000"
	msg3 := "NiceMessage"
	msg4 := "Space Message"
	msg5 := "Enter\nMessage"

	for i, tt := range []struct {
		msg  string
		want string
	}{
		{
			msg:  msg,
			want: fmt.Sprintf("] %q                   %q=%q\n", msg, msg, msg),
		},
		{
			msg:  msg2,
			want: fmt.Sprintf("] %q                             %q=%q\n", msg2, msg2, msg2),
		},
		{
			msg:  msg3,
			want: fmt.Sprintf("] %s                              %s=%s\n", msg3, msg3, msg3),
		},
		{
			msg:  msg4,
			want: fmt.Sprintf("] %s                            %q=%q\n", msg4, msg4, msg4),
		},
		{
			msg:  msg5,
			want: fmt.Sprintf("] %s                            %q=%q\n", msg5, msg5, msg5),
		},
	} {
		var (
			logger = New()
			out    = new(strings.Builder)
		)
		logger.SetHandler(LvlFilterHandler(LvlInfo, StreamHandler(out, TerminalFormat(false))))
		logger.Info(tt.msg, tt.msg, tt.msg)
		if have := out.String()[24:]; tt.want != have {
			t.Fatalf("test %d: want / have: \n%v\n%v", i, tt.want, have)
		}
	}
}
