package zkproof

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"testing"

	"github.com/scroll-tech/go-ethereum/common"
	"github.com/scroll-tech/go-ethereum/core/types"
)

func init() {
	orderScheme := os.Getenv("OP_ORDER")
	var orderSchemeI int
	if orderScheme != "" {
		if n, err := fmt.Sscanf(orderScheme, "%d", &orderSchemeI); err == nil && n == 1 {
			usedOrdererScheme = MPTWitnessType(orderSchemeI)
		}
	}
}

func loadStaff(t *testing.T, fname string) *types.BlockResult {
	f, err := os.Open(fname)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	bt, err := io.ReadAll(f)
	if err != nil {
		t.Fatal(err)
	}

	out := new(types.BlockResult)

	err = json.Unmarshal(bt, out)
	if err != nil {
		t.Fatal(err)
	}

	return out
}

func TestWriterCreation(t *testing.T) {
	trace := loadStaff(t, "deploy_trace.json")
	writer, err := NewZkTrieProofWriter(trace.StorageTrace)
	if err != nil {
		t.Fatal(err)
	}

	if len(writer.tracingAccounts) != 4 {
		t.Error("unexpected tracing account data", writer.tracingAccounts)
	}

	if v, existed := writer.tracingAccounts[common.HexToAddress("0x000000000000000000636F6e736F6c652e6c6f67")]; !existed || v != nil {
		t.Error("wrong tracing status for uninited address", v, existed)
	}

	if v, existed := writer.tracingAccounts[common.HexToAddress("0xb36feAEaF76c2A33335b73bEF9aEf7a23d9af1e3")]; !existed || v != nil {
		t.Error("wrong tracing status for uninited address", v, existed)
	}

	if v, existed := writer.tracingAccounts[common.HexToAddress("0x4cb1aB63aF5D8931Ce09673EbD8ae2ce16fD6571")]; !existed || v == nil {
		t.Error("wrong tracing status for establied address", v, existed)
	}

	if len(writer.tracingStorageTries) != 1 {
		t.Error("unexpected tracing storage data", writer.tracingStorageTries)
	}

	if v, existed := writer.tracingStorageTries[common.HexToAddress("0xb36feAEaF76c2A33335b73bEF9aEf7a23d9af1e3")]; !existed || v == nil {
		t.Error("wrong tracing storage statu", existed, v)
	}

}

func TestGreeterTx(t *testing.T) {
	trace := loadStaff(t, "greeter_trace.json")
	writer, err := NewZkTrieProofWriter(trace.StorageTrace)
	if err != nil {
		t.Fatal(err)
	}

	od := &simpleOrderer{}
	theTx := trace.ExecutionResults[0]
	handleTx(od, theTx)

	t.Log(od)

	for _, op := range od.savedOp {
		_, err = writer.HandleNewState(op)
		if err != nil {
			t.Fatal(err)
		}
	}

	traces, err := HandleBlockResult(trace)
	t.Log("traces: ", len(traces))
	outObj, _ := json.Marshal(traces)
	t.Log(string(outObj))
	if err != nil {
		t.Fatal(err)
	}
}

func TestTokenTx(t *testing.T) {
	trace := loadStaff(t, "token_trace.json")
	traces, err := HandleBlockResult(trace)
	outObj, _ := json.Marshal(traces)
	t.Log(string(outObj))
	if err != nil {
		t.Fatal(err)
	}

}

func TestCallTx(t *testing.T) {
	trace := loadStaff(t, "call_trace.json")
	traces, err := HandleBlockResult(trace)
	outObj, _ := json.Marshal(traces)
	t.Log(string(outObj))
	if err != nil {
		t.Fatal(err)
	}

	trace = loadStaff(t, "staticcall_trace.json")
	traces, err = HandleBlockResult(trace)
	outObj, _ = json.Marshal(traces)
	t.Log(string(outObj))
	if err != nil {
		t.Fatal(err)
	}
}

func TestCreateTx(t *testing.T) {
	trace := loadStaff(t, "create_trace.json")
	traces, err := HandleBlockResult(trace)
	outObj, _ := json.Marshal(traces)
	t.Log(string(outObj))
	if err != nil {
		t.Fatal(err)
	}

	trace = loadStaff(t, "deploy_trace.json")
	traces, err = HandleBlockResult(trace)
	outObj, _ = json.Marshal(traces)
	t.Log(string(outObj))
	if err != nil {
		t.Fatal(err)
	}

}

func TestFailedCallTx(t *testing.T) {
	trace := loadStaff(t, "fail_call_trace.json")
	traces, err := HandleBlockResult(trace)
	outObj, _ := json.Marshal(traces)
	t.Log(string(outObj))
	if err != nil {
		t.Fatal(err)
	}

	trace = loadStaff(t, "fail_create_trace.json")
	traces, err = HandleBlockResult(trace)
	outObj, _ = json.Marshal(traces)
	t.Log(string(outObj))
	if err != nil {
		t.Fatal(err)
	}

}

//notice: now only work with OP_ORDER=2
func TestDeleteTx(t *testing.T) {
	trace := loadStaff(t, "delete_trace.json")
	traces, err := HandleBlockResult(trace)
	outObj, _ := json.Marshal(traces)
	t.Log(string(outObj))
	if err != nil {
		t.Fatal(err)
	}
}

func TestMutipleTx(t *testing.T) {
	trace := loadStaff(t, "multi_txs.json")
	traces, err := HandleBlockResult(trace)
	outObj, _ := json.Marshal(traces)
	t.Log(string(outObj))
	if err != nil {
		t.Fatal(err)
	}
}
