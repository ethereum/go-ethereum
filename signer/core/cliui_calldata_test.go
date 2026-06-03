package core

import (
	"bufio"
	"bytes"
	"os"
	"strings"
	"testing"

	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/signer/core/apitypes"
)

func captureOutput(t *testing.T, fn func()) string {
	t.Helper()
	old := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("pipe error: %v", err)
	}
	os.Stdout = w
	var buf bytes.Buffer
	done := make(chan struct{})
	go func() {
		_, _ = buf.ReadFrom(r)
		close(done)
	}()
	fn()
	_ = w.Close()
	<-done
	os.Stdout = old
	return buf.String()
}

func TestApproveTx_DisplaysEffectiveCalldata(t *testing.T) {
	mkUI := func(input string) *CommandlineUI {
		return &CommandlineUI{in: bufio.NewReader(strings.NewReader(input))}
	}
	hex := func(b []byte) string { return hexutil.Encode(b) }
	// helper to create *hexutil.Bytes
	hx := func(b []byte) *hexutil.Bytes { v := hexutil.Bytes(b); return &v }
	// zero 1559 fee fields to avoid GasPrice nil deref in CLI
	zeroFee := func(a *apitypes.SendTxArgs) {
		z := new(hexutil.Big)
		a.MaxFeePerGas = z
		a.MaxPriorityFeePerGas = z
	}

	cases := []struct {
		name          string
		args          apitypes.SendTxArgs
		wantContains  []string
		wantNotExists []string
	}{
		{
			name: "only input",
			args: func() apitypes.SendTxArgs {
				a := apitypes.SendTxArgs{Input: hx([]byte{0x01, 0x02})}
				zeroFee(&a)
				return a
			}(),
			wantContains: []string{
				"input:",
				hex([]byte{0x01, 0x02}),
			},
			wantNotExists: []string{"WARNING: both input and data provided and differ"},
		},
		{
			name: "only data",
			args: func() apitypes.SendTxArgs { a := apitypes.SendTxArgs{Data: hx([]byte{0x0a})}; zeroFee(&a); return a }(),
			wantContains: []string{
				"data:",
				hex([]byte{0x0a}),
			},
			wantNotExists: []string{"WARNING: both input and data provided and differ"},
		},
		{
			name: "both equal",
			args: func() apitypes.SendTxArgs {
				b := hexutil.Bytes([]byte{0xaa, 0xbb})
				a := apitypes.SendTxArgs{Input: &b, Data: &b}
				zeroFee(&a)
				return a
			}(),
			wantContains: []string{
				"input:",
				hex([]byte{0xaa, 0xbb}),
			},
			wantNotExists: []string{"WARNING: both input and data provided and differ", "data:"},
		},
		{
			name: "both different",
			args: func() apitypes.SendTxArgs {
				a := apitypes.SendTxArgs{Input: hx([]byte{0x01}), Data: hx([]byte{0x02})}
				zeroFee(&a)
				return a
			}(),
			wantContains: []string{
				"input:",
				"data:",
				"WARNING: both input and data provided and differ",
				hex([]byte{0x01}),
				hex([]byte{0x02}),
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			ui := mkUI("y\n")
			req := &SignTxRequest{Transaction: tc.args}
			out := captureOutput(t, func() {
				_, err := ui.ApproveTx(req)
				if err != nil {
					t.Fatalf("ApproveTx error: %v", err)
				}
			})
			for _, s := range tc.wantContains {
				if !strings.Contains(out, s) {
					t.Fatalf("output does not contain %q. Output:\n%s", s, out)
				}
			}
			for _, s := range tc.wantNotExists {
				if strings.Contains(out, s) {
					t.Fatalf("output should not contain %q. Output:\n%s", s, out)
				}
			}
		})
	}
}

func TestLogDiff_EffectiveCalldata(t *testing.T) {
	mk := func(inOld, dataOld, inNew, dataNew []byte) (SignTxRequest, SignTxResponse) {
		var (
			inO, dO, inN, dN *hexutil.Bytes
		)
		if inOld != nil {
			v := hexutil.Bytes(inOld)
			inO = &v
		}
		if dataOld != nil {
			v := hexutil.Bytes(dataOld)
			dO = &v
		}
		if inNew != nil {
			v := hexutil.Bytes(inNew)
			inN = &v
		}
		if dataNew != nil {
			v := hexutil.Bytes(dataNew)
			dN = &v
		}
		return SignTxRequest{Transaction: apitypes.SendTxArgs{Input: inO, Data: dO}}, SignTxResponse{Transaction: apitypes.SendTxArgs{Input: inN, Data: dN}}
	}
	cases := []struct {
		name     string
		inOld    []byte
		dataOld  []byte
		inNew    []byte
		dataNew  []byte
		modified bool
	}{
		{"only input unchanged", []byte{0x01}, nil, []byte{0x01}, nil, false},
		{"only data unchanged", nil, []byte{0x02}, nil, []byte{0x02}, false},
		{"both equal unchanged", []byte{0x0a}, []byte{0x0a}, []byte{0x0a}, []byte{0x0a}, false},
		{"effective changed (input differs)", []byte{0x01}, nil, []byte{0x02}, nil, true},
		{"effective changed (data differs, no input)", nil, []byte{0x01}, nil, []byte{0x02}, true},
		{"effective equal though underlying fields differ", []byte{0xaa}, []byte{0xbb}, []byte{0xaa}, []byte{0xbb}, false},
		{"both set but only new input changes", []byte{0x01}, []byte{0x01}, []byte{0x02}, []byte{0x01}, true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			orig, nw := mk(tc.inOld, tc.dataOld, tc.inNew, tc.dataNew)
			m := logDiff(&orig, &nw)
			if m != tc.modified {
				t.Fatalf("modified mismatch: have %v want %v", m, tc.modified)
			}
		})
	}
}
