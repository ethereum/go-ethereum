// Copyright 2025 The go-ethereum Authors
// This file is part of go-ethereum.
//
// go-ethereum is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// go-ethereum is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU General Public License for more details.
//
// You should have received a copy of the GNU General Public License
// along with go-ethereum. If not, see <http://www.gnu.org/licenses/>.

package main

import (
	"crypto/ecdsa"
	"encoding/json"
	"io"
	"math/big"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
)

// signedTestTx returns a signed legacy transaction and its hex encoding.
func signedTestTx(t *testing.T) (*types.Transaction, string) {
	t.Helper()

	key, err := crypto.GenerateKey()
	if err != nil {
		t.Fatal(err)
	}
	return signedTestTxWithKey(t, key)
}

func signedTestTxWithKey(t *testing.T, key *ecdsa.PrivateKey) (*types.Transaction, string) {
	t.Helper()

	tx := types.NewTx(&types.LegacyTx{
		Nonce:    6,
		GasPrice: big.NewInt(1_000_000_000), // 1 Gwei – real networks reject gas price 0
		Gas:      21055,
		To:       addrPtr(common.HexToAddress("0x78b5290269740033b05bd8d71c97331295eb5918")),
		Value:    new(big.Int).Mul(big.NewInt(10), big.NewInt(1e18)), // 10 ETH
	})
	signer := types.NewEIP155Signer(big.NewInt(1))
	signed, err := types.SignTx(tx, signer, key)
	if err != nil {
		t.Fatal(err)
	}
	data, err := signed.MarshalBinary()
	if err != nil {
		t.Fatal(err)
	}
	return signed, hexutil.Encode(data)
}

func addrPtr(a common.Address) *common.Address { return &a }

// fakeRPC starts an HTTP server that responds to eth_sendRawTransaction.
func fakeRPC(t *testing.T, wantErr bool) *httptest.Server {
	t.Helper()

	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Method string            `json:"method"`
			Params []json.RawMessage `json:"params"`
			ID     json.RawMessage   `json:"id"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		if req.Method != "eth_sendRawTransaction" {
			json.NewEncoder(w).Encode(map[string]interface{}{
				"jsonrpc": "2.0",
				"id":      req.ID,
				"error":   map[string]interface{}{"code": -32601, "message": "method not found"},
			})
			return
		}
		if wantErr {
			json.NewEncoder(w).Encode(map[string]interface{}{
				"jsonrpc": "2.0",
				"id":      req.ID,
				"error":   map[string]interface{}{"code": -32000, "message": "already known"},
			})
			return
		}
		// Decode the raw tx to return its hash as the result.
		var hexData string
		if err := json.Unmarshal(req.Params[0], &hexData); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		rawBytes, err := hexutil.Decode(hexData)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		var tx types.Transaction
		if err := tx.UnmarshalBinary(rawBytes); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		json.NewEncoder(w).Encode(map[string]interface{}{
			"jsonrpc": "2.0",
			"id":      req.ID,
			"result":  tx.Hash().Hex(),
		})
	}))
}

func TestRunSuccess(t *testing.T) {
	srv := fakeRPC(t, false)
	defer srv.Close()

	_, txHex := signedTestTx(t)
	err := run([]string{"--rpc", srv.URL, txHex}, strings.NewReader(""))
	if err != nil {
		t.Fatal("unexpected error:", err)
	}
}

func TestRunFromStdin(t *testing.T) {
	srv := fakeRPC(t, false)
	defer srv.Close()

	_, txHex := signedTestTx(t)
	err := run([]string{"--rpc", srv.URL}, strings.NewReader(txHex))
	if err != nil {
		t.Fatal("unexpected error:", err)
	}
}

func TestRunRPCError(t *testing.T) {
	srv := fakeRPC(t, true)
	defer srv.Close()

	_, txHex := signedTestTx(t)

	// Capture stdout – raw hex should still be printed on RPC failure.
	oldStdout := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	os.Stdout = w

	runErr := run([]string{"--rpc", srv.URL, txHex}, strings.NewReader(""))

	w.Close()
	os.Stdout = oldStdout

	if runErr == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(runErr.Error(), "already known") {
		t.Fatalf("unexpected error message: %v", runErr)
	}

	out, err := io.ReadAll(r)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(out), "Raw tx: 0x") {
		t.Fatal("expected raw hex in output even on RPC error")
	}
}

func TestRunNoInput(t *testing.T) {
	err := run(nil, strings.NewReader(""))
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "no transaction data") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestRunBadHex(t *testing.T) {
	err := run([]string{"not-hex-data"}, strings.NewReader(""))
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "invalid hex") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestRunBadTx(t *testing.T) {
	err := run([]string{"0xdeadbeef"}, strings.NewReader(""))
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "decoding transaction") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestRunHelp(t *testing.T) {
	err := run([]string{"--help"}, strings.NewReader(""))
	if err != nil {
		t.Fatal("unexpected error:", err)
	}
}

func TestRunUnknownFlag(t *testing.T) {
	err := run([]string{"--unknown"}, strings.NewReader(""))
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "unknown flag") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestRunExtraArgs(t *testing.T) {
	err := run([]string{"0xaa", "0xbb"}, strings.NewReader(""))
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "unexpected argument") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestFormatWei(t *testing.T) {
	tests := []struct {
		wei  *big.Int
		want string
	}{
		{nil, "0 wei (0 ETH)"},
		{big.NewInt(0), "0 wei (0 ETH)"},
		{big.NewInt(1e18), "1000000000000000000 wei (1.000000000000000000 ETH)"},
		{new(big.Int).Mul(big.NewInt(10), big.NewInt(1e18)), "10000000000000000000 wei (10.000000000000000000 ETH)"},
	}
	for _, tt := range tests {
		got := formatWei(tt.wei)
		if got != tt.want {
			t.Errorf("formatWei(%v) = %q, want %q", tt.wei, got, tt.want)
		}
	}
}

func TestFormatGwei(t *testing.T) {
	tests := []struct {
		wei  *big.Int
		want string
	}{
		{nil, "0 wei (0 Gwei)"},
		{big.NewInt(0), "0 wei (0 Gwei)"},
		{big.NewInt(1_000_000_000), "1000000000 wei (1.000000000 Gwei)"},
		{big.NewInt(20_000_000_000), "20000000000 wei (20.000000000 Gwei)"},
	}
	for _, tt := range tests {
		got := formatGwei(tt.wei)
		if got != tt.want {
			t.Errorf("formatGwei(%v) = %q, want %q", tt.wei, got, tt.want)
		}
	}
}

func TestRunEqualsSyntax(t *testing.T) {
	srv := fakeRPC(t, false)
	defer srv.Close()

	_, txHex := signedTestTx(t)
	err := run([]string{"--rpc=" + srv.URL, txHex}, strings.NewReader(""))
	if err != nil {
		t.Fatal("unexpected error:", err)
	}
}

func TestRunOutputEndsWithRawHex(t *testing.T) {
	srv := fakeRPC(t, false)
	defer srv.Close()

	_, txHex := signedTestTx(t)

	// Capture stdout to verify "Raw tx:" appears in output.
	oldStdout := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	os.Stdout = w

	runErr := run([]string{"--rpc", srv.URL, txHex}, strings.NewReader(""))

	w.Close()
	os.Stdout = oldStdout

	if runErr != nil {
		t.Fatal("unexpected error:", runErr)
	}

	out, err := io.ReadAll(r)
	if err != nil {
		t.Fatal(err)
	}
	lines := strings.Split(strings.TrimSpace(string(out)), "\n")
	lastLine := lines[len(lines)-1]

	// The last line must be the raw hex transaction.
	if !strings.HasPrefix(lastLine, "Raw tx: 0x") {
		t.Fatalf("last output line = %q, want prefix \"Raw tx: 0x\"", lastLine)
	}
	// Verify the hex payload round-trips back to the input.
	rawHex := strings.TrimPrefix(lastLine, "Raw tx: ")
	if rawHex != txHex {
		t.Fatalf("raw hex mismatch:\n got  %s\n want %s", rawHex, txHex)
	}
}
