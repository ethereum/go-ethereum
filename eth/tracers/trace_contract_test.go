// Copyright 2026 The go-ethereum Authors
// This file is part of the go-ethereum library.
//
// The go-ethereum library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The go-ethereum library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the go-ethereum library. If not, see <http://www.gnu.org/licenses/>.

package tracers

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"math/big"
	"strings"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/params"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/holiman/uint256"
)

func traceContractClient(t *testing.T, api *TraceAPI) *rpc.Client {
	t.Helper()
	server := rpc.NewServer()
	t.Cleanup(server.Stop)
	if err := server.RegisterName("trace", api); err != nil {
		t.Fatal(err)
	}
	client := rpc.DialInProc(server)
	t.Cleanup(client.Close)
	return client
}

func requireTraceCode(t *testing.T, err error, code int) {
	t.Helper()
	var rpcErr rpc.Error
	if !errors.As(err, &rpcErr) || rpcErr.ErrorCode() != code {
		t.Fatalf("want RPC code %d, got %v", code, err)
	}
}

func TestTraceNamespaceBlockErrors(t *testing.T) {
	api, _ := traceTestAPI(t, nil, nil)
	client := traceContractClient(t, api)
	for _, method := range []string{"trace_call", "trace_callMany", "trace_block", "trace_replayBlockTransactions", "trace_filter"} {
		t.Run(method, func(t *testing.T) {
			var args []any
			switch method {
			case "trace_call":
				args = []any{map[string]any{"to": traceTestTarget}, TraceTypes{}, "0xffff"}
			case "trace_callMany":
				args = []any{TraceCalls{}, "0xffff"}
			case "trace_replayBlockTransactions":
				args = []any{"0xffff", TraceTypes{}}
			case "trace_filter":
				args = []any{map[string]string{"toBlock": "0xffff"}}
			default:
				args = []any{"0xffff"}
			}
			var result json.RawMessage
			requireTraceCode(t, client.Call(&result, method, args...), -32001)
		})
	}
}

func TestTraceNamespaceRawRejectionCodes(t *testing.T) {
	key, _ := crypto.HexToECDSA(strings.Repeat("0", 63) + "2")
	sender := crypto.PubkeyToAddress(key.PublicKey)
	api, backend := traceTestAPI(t, common.FromHex("60006000fd"), types.GenesisAlloc{sender: {Balance: big.NewInt(1e18), Nonce: 1}})
	client := traceContractClient(t, api)
	for _, name := range []string{"low nonce", "high nonce", "intrinsic gas", "balance", "fee", "chain", "signature"} {
		t.Run(name, func(t *testing.T) {
			data := &types.LegacyTx{Nonce: 1, Gas: 100000, GasPrice: big.NewInt(params.InitialBaseFee), To: &traceTestTarget}
			signer := types.LatestSigner(backend.ChainConfig())
			switch name {
			case "low nonce":
				data.Nonce = 0
			case "high nonce":
				data.Nonce = 2
			case "intrinsic gas":
				data.Gas = 20000
			case "balance":
				data.Value = new(big.Int).Exp(big.NewInt(10), big.NewInt(20), nil)
			case "fee":
				data.GasPrice = big.NewInt(1)
			case "chain":
				signer = types.LatestSignerForChainID(big.NewInt(999999))
			}
			tx := types.MustSignNewTx(key, signer, data)
			if name == "signature" {
				tx = types.NewTx(data)
			}
			encoded, _ := tx.MarshalBinary()
			var result json.RawMessage
			requireTraceCode(t, client.Call(&result, "trace_rawTransaction", hexutil.Bytes(encoded), TraceTypes{"trace"}), -32003)
		})
	}
	var result json.RawMessage
	requireTraceCode(t, client.Call(&result, "trace_rawTransaction", "0x01", TraceTypes{}), -32602)
	// Valid transactions which fail inside the EVM still return execution results.
	for _, gas := range []uint64{21000, 100000} {
		tx := types.MustSignNewTx(key, types.LatestSigner(backend.ChainConfig()), &types.LegacyTx{Nonce: 1, Gas: gas, GasPrice: big.NewInt(params.InitialBaseFee), To: &traceTestTarget})
		encoded, _ := tx.MarshalBinary()
		var result TraceExecution
		if err := client.Call(&result, "trace_rawTransaction", hexutil.Bytes(encoded), TraceTypes{"trace"}); err != nil {
			t.Fatal(err)
		}
		if len(result.Trace) != 1 || result.Trace[0].Error == "" {
			t.Fatalf("missing EVM failure: %+v", result)
		}
	}
}

func TestTraceNamespaceCallFields(t *testing.T) {
	api, backend := traceTestAPI(t, common.FromHex("602a60005260206000f3"), nil)
	client := traceContractClient(t, api)
	var baseline TraceExecution
	args := map[string]any{"from": traceTestSender, "to": traceTestTarget, "chainId": hexutil.EncodeBig(backend.ChainConfig().ChainID)}
	if err := client.Call(&baseline, "trace_call", args, TraceTypes{}); err != nil {
		t.Fatal(err)
	}
	for _, field := range []string{"futureField", "blobs", "proofs"} {
		for _, value := range []any{nil, "ignored"} {
			args[field] = value
			var result TraceExecution
			if err := client.Call(&result, "trace_call", args, TraceTypes{}); err != nil {
				t.Fatal(err)
			}
			if !bytes.Equal(result.Output, baseline.Output) {
				t.Fatal("unknown field changed execution")
			}
			delete(args, field)
		}
	}
	for _, field := range []string{"gas", "chainId", "blobVersionedHashes", "authorizationList"} {
		var result json.RawMessage
		requireTraceCode(t, client.Call(&result, "trace_call", map[string]any{field: nil}, TraceTypes{}), -32602)
	}
}

func TestTraceNamespaceBlobCall(t *testing.T) {
	api, _ := traceTestAPI(t, common.FromHex("60004960005260206000f3"), nil)
	client := traceContractClient(t, api)
	hash := common.Hash{0: 1, 31: 42}
	for _, kind := range []string{"", "0x03"} {
		args := map[string]any{"from": traceTestSender, "to": traceTestTarget, "blobVersionedHashes": []common.Hash{hash}, "maxFeePerBlobGas": "0x0"}
		if kind != "" {
			args["type"] = kind
		}
		var result TraceExecution
		if err := client.Call(&result, "trace_call", args, TraceTypes{"trace"}); err != nil {
			t.Fatal(err)
		}
		if !bytes.Equal(result.Output, hash[:]) {
			t.Fatalf("BLOBHASH output: %x", result.Output)
		}
	}
}

func TestTraceNamespaceAuthorizationCall(t *testing.T) {
	key, _ := crypto.HexToECDSA(strings.Repeat("0", 63) + "3")
	authority := crypto.PubkeyToAddress(key.PublicKey)
	api, backend := traceTestAPI(t, common.FromHex("602a60005260206000f3"), types.GenesisAlloc{authority: {Balance: big.NewInt(1)}})
	auth, err := types.SignSetCode(key, types.SetCodeAuthorization{ChainID: *uint256.MustFromBig(backend.ChainConfig().ChainID), Address: traceTestTarget})
	if err != nil {
		t.Fatal(err)
	}
	client := traceContractClient(t, api)
	for _, kind := range []string{"", "0x04"} {
		args := map[string]any{"from": traceTestSender, "to": authority, "authorizationList": []types.SetCodeAuthorization{auth}}
		if kind != "" {
			args["type"] = kind
		}
		var result TraceExecution
		if err := client.Call(&result, "trace_call", args, TraceTypes{"trace", "stateDiff"}); err != nil {
			t.Fatal(err)
		}
		if !bytes.Equal(result.Output, common.LeftPadBytes([]byte{42}, 32)) || result.StateDiff[authority] == nil {
			t.Fatalf("authorization did not execute: %+v", result)
		}
	}
}

func TestTraceNamespaceConflictingCallFields(t *testing.T) {
	api, _ := traceTestAPI(t, nil, nil)
	client := traceContractClient(t, api)
	for _, fields := range []map[string]any{
		{"type": "0x03"},
		{"type": "0x04"},
		{"type": "0x02", "blobVersionedHashes": []common.Hash{}},
		{"type": "0x03", "authorizationList": []types.SetCodeAuthorization{}},
		{"gasPrice": "0x0", "blobVersionedHashes": []common.Hash{}},
		{"gasPrice": "0x0", "authorizationList": []types.SetCodeAuthorization{}},
		{"blobVersionedHashes": []common.Hash{}, "authorizationList": []types.SetCodeAuthorization{}},
		{"chainId": "0xffff"},
	} {
		fields["to"] = traceTestTarget
		var result json.RawMessage
		requireTraceCode(t, client.Call(&result, "trace_call", fields, TraceTypes{}), -32602)
	}
}

type traceBlockErrorBackend struct {
	Backend
	err error
}

func (b traceBlockErrorBackend) HeaderByNumber(context.Context, rpc.BlockNumber) (*types.Header, error) {
	return nil, b.err
}

func (b traceBlockErrorBackend) BlockByNumber(context.Context, rpc.BlockNumber) (*types.Block, error) {
	return nil, b.err
}

func TestTraceNamespaceBackendErrors(t *testing.T) {
	_, backend := traceTestAPI(t, nil, nil)
	for _, number := range []rpc.BlockNumber{rpc.SafeBlockNumber, rpc.FinalizedBlockNumber} {
		for _, original := range []error{errors.New(number.String() + " block not found"), errors.New("database unavailable"), &traceRPCError{4444, "pruned history"}} {
			api := NewTraceAPI(traceBlockErrorBackend{backend, original})
			_, blockErr := api.Block(context.Background(), number)
			_, filterErr := api.Filter(context.Background(), TraceFilter{FromBlock: &number})
			for _, err := range []error{blockErr, filterErr} {
				if original.Error() == number.String()+" block not found" {
					requireTraceCode(t, err, -32001)
				} else if err != original {
					t.Fatalf("backend error was reclassified: %v -> %v", original, err)
				}
			}
		}
	}
}

func TestTraceNamespacePrunedTransactionLookup(t *testing.T) {
	api, backend := traceTestAPI(t, nil, nil)
	client := traceContractClient(t, api)
	for _, tail := range []uint64{0, 1, 2} {
		rawdb.WriteTxIndexTail(backend.ChainDb(), tail)
		for _, method := range []string{"trace_transaction", "trace_get", "trace_replayTransaction"} {
			args := []any{common.Hash{31: 1}}
			if method != "trace_transaction" {
				args = append(args, []string{})
			}
			var result json.RawMessage
			err := client.Call(&result, method, args...)
			if tail > 1 {
				requireTraceCode(t, err, 4444)
			} else if err != nil || string(result) != "null" {
				t.Fatalf("complete lookup: %s %v", result, err)
			}
		}
	}
}

// Model a backend whose earliest tag is a retention boundary, not genesis.
type traceRetentionBackend struct{ Backend }

func (b traceRetentionBackend) HeaderByNumber(ctx context.Context, n rpc.BlockNumber) (*types.Header, error) {
	if n == rpc.EarliestBlockNumber {
		return nil, errors.New("retention boundary selected")
	}
	return b.Backend.HeaderByNumber(ctx, n)
}

func (b traceRetentionBackend) BlockByNumber(ctx context.Context, n rpc.BlockNumber) (*types.Block, error) {
	if n == rpc.EarliestBlockNumber {
		return nil, errors.New("retention boundary selected")
	}
	return b.Backend.BlockByNumber(ctx, n)
}

func TestTraceNamespaceEarliestMeansGenesis(t *testing.T) {
	_, backend := traceTestAPI(t, nil, nil)
	api := NewTraceAPI(traceRetentionBackend{backend})
	n := rpc.EarliestBlockNumber
	if _, err := api.Block(context.Background(), n); err != nil {
		t.Fatal(err)
	}
	if _, err := api.Filter(context.Background(), TraceFilter{FromBlock: &n, ToBlock: &n}); err != nil {
		t.Fatal(err)
	}
}
