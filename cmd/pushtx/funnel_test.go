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
	"encoding/json"
	"math/big"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
)

func TestBuildERC20Transfer(t *testing.T) {
	to := common.HexToAddress("0xfe89cc7abb2c4183683ab71653c4cdc9b02d44b7")
	amount := new(big.Int).SetUint64(900_000_000_000)

	data := buildERC20Transfer(to, amount)

	// First 4 bytes must be the transfer(address,uint256) selector.
	selector := hexutil.Encode(data[:4])
	if selector != "0xa9059cbb" {
		t.Fatalf("wrong selector: got %s, want 0xa9059cbb", selector)
	}

	// Expected calldata from the Tenderly simulation.
	want := "0xa9059cbb000000000000000000000000fe89cc7abb2c4183683ab71653c4cdc9b02d44b7000000000000000000000000000000000000000000000000000000d18c2e2800"
	got := hexutil.Encode(data)
	if got != want {
		t.Fatalf("calldata mismatch:\n got  %s\n want %s", got, want)
	}
}

func TestBuildExecTransaction(t *testing.T) {
	cfg := defaultFunnelConfig()
	data, err := buildExecTransaction(cfg)
	if err != nil {
		t.Fatal(err)
	}

	// execTransaction selector = 0x6a761202.
	selector := hexutil.Encode(data[:4])
	if selector != "0x6a761202" {
		t.Fatalf("wrong selector: got %s, want 0x6a761202", selector)
	}

	// Encoded data must contain the USDC address.
	dataHex := strings.ToLower(hexutil.Encode(data))
	if !strings.Contains(dataHex, "a0b86991c6218b36c1d19d4a2e9eb0ce3606eb48") {
		t.Fatal("encoded data does not contain USDC address")
	}
}

func TestDefaultFunnelConfig(t *testing.T) {
	cfg := defaultFunnelConfig()

	if cfg.Safe != safeAddr {
		t.Errorf("Safe = %s, want %s", cfg.Safe.Hex(), safeAddr.Hex())
	}
	if cfg.To != usdcAddr {
		t.Errorf("To = %s, want %s", cfg.To.Hex(), usdcAddr.Hex())
	}
	if cfg.Value.Sign() != 0 {
		t.Errorf("Value = %s, want 0", cfg.Value)
	}
	if cfg.Operation != 0 {
		t.Errorf("Operation = %d, want 0", cfg.Operation)
	}
	if len(cfg.Signatures) != 65 {
		t.Errorf("Signatures length = %d, want 65", len(cfg.Signatures))
	}
}

func TestIsCalldata(t *testing.T) {
	tests := []struct {
		name string
		data []byte
		want bool
	}{
		{"ERC20 transfer selector", common.FromHex("a9059cbb0000"), true},
		{"short data", []byte{0xa9}, false},
		{"legacy tx RLP", common.FromHex("f86c0184"), false},
		{"typed tx EIP-1559", common.FromHex("02f86c01"), false},
	}
	for _, tt := range tests {
		if got := isCalldata(tt.data); got != tt.want {
			t.Errorf("isCalldata(%s) = %v, want %v", tt.name, got, tt.want)
		}
	}
}

func TestRunCalldataError(t *testing.T) {
	// Sending raw calldata (not a signed tx) should produce a helpful error.
	calldata := "0xa9059cbb00000000000000000000000099d580d3a7fe7bd183b2464517b2cd7ce5a8f15a0000000000000000000000000000000000000000000000000de0b6b3a7640000"
	err := run([]string{calldata}, strings.NewReader(""))
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "contract calldata") {
		t.Fatalf("expected calldata detection message, got: %v", err)
	}
	if !strings.Contains(err.Error(), "a9059cbb") {
		t.Fatalf("expected selector in error, got: %v", err)
	}
}

// fakeRPCFunnel starts an HTTP server that rejects eth_sendRawTransaction
// and accepts the funnel flow (eth_call + eth_sendTransaction + receipt).
func fakeRPCFunnel(t *testing.T) *httptest.Server {
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
		switch req.Method {
		case "eth_sendRawTransaction":
			// Simulate insufficient funds error.
			json.NewEncoder(w).Encode(map[string]interface{}{
				"jsonrpc": "2.0",
				"id":      req.ID,
				"error":   map[string]interface{}{"code": -32000, "message": "insufficient funds for gas * price + value"},
			})
		case "eth_call":
			// Validation succeeds – return ABI-encoded true.
			json.NewEncoder(w).Encode(map[string]interface{}{
				"jsonrpc": "2.0",
				"id":      req.ID,
				"result":  "0x0000000000000000000000000000000000000000000000000000000000000001",
			})
		case "eth_sendTransaction":
			// Return a fake tx hash.
			json.NewEncoder(w).Encode(map[string]interface{}{
				"jsonrpc": "2.0",
				"id":      req.ID,
				"result":  "0xabcdef1234567890abcdef1234567890abcdef1234567890abcdef1234567890",
			})
		case "eth_getTransactionReceipt":
			json.NewEncoder(w).Encode(map[string]interface{}{
				"jsonrpc": "2.0",
				"id":      req.ID,
				"result": map[string]interface{}{
					"status":      "0x1",
					"blockNumber": "0x178C3C9",
				},
			})
		default:
			json.NewEncoder(w).Encode(map[string]interface{}{
				"jsonrpc": "2.0",
				"id":      req.ID,
				"error":   map[string]interface{}{"code": -32601, "message": "method not found"},
			})
		}
	}))
}

func TestRunFunnelFallback(t *testing.T) {
	srv := fakeRPCFunnel(t)
	defer srv.Close()

	_, txHex := signedTestTx(t)
	err := run([]string{"--rpc", srv.URL, "--funnel", txHex}, strings.NewReader(""))
	if err != nil {
		t.Fatal("unexpected error:", err)
	}
}

func TestRunFunnelNotEnabledOnError(t *testing.T) {
	srv := fakeRPCFunnel(t)
	defer srv.Close()

	_, txHex := signedTestTx(t)
	// Without --funnel, the insufficient funds error should propagate.
	err := run([]string{"--rpc", srv.URL, txHex}, strings.NewReader(""))
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "insufficient funds") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestValidateTransactionSuccess(t *testing.T) {
	srv := fakeRPCFunnel(t)
	defer srv.Close()

	hash := common.HexToHash("0xabcdef1234567890abcdef1234567890abcdef1234567890abcdef1234567890")
	err := validateTransaction(srv.URL, hash)
	if err != nil {
		t.Fatal("unexpected error:", err)
	}
}
