package blocktraces

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"testing"

	"github.com/scroll-tech/go-ethereum/common"
	"github.com/scroll-tech/go-ethereum/core/types"
	"github.com/scroll-tech/go-ethereum/trie/zkproof"
)

func init() {
	orderScheme := os.Getenv("OP_ORDER")
	var orderSchemeI int
	if orderScheme != "" {
		if n, err := fmt.Sscanf(orderScheme, "%d", &orderSchemeI); err == nil && n == 1 {
			zkproof.SetOrderScheme(zkproof.MPTWitnessType(orderSchemeI))
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
	trace := loadStaff(t, "deploy.json")
	writer, err := zkproof.NewZkTrieProofWriter(trace.StorageTrace)
	if err != nil {
		t.Fatal(err)
	}

	if len(writer.TracingAccounts()) != 3 {
		t.Error("unexpected tracing account data", writer.TracingAccounts())
	}

	if v, existed := writer.TracingAccounts()[common.HexToAddress("0x08c683b684d1e24cab8ce6de5c8c628d993ac140")]; !existed || v != nil {
		t.Error("wrong tracing status for uninited address", v, existed)
	}

	if v, existed := writer.TracingAccounts()[common.HexToAddress("0x4cb1aB63aF5D8931Ce09673EbD8ae2ce16fD6571")]; !existed || v == nil {
		t.Error("wrong tracing status for establied address", v, existed)
	}

}

func TestGreeterTx(t *testing.T) {
	trace := loadStaff(t, "greeter.json")
	writer, err := zkproof.NewZkTrieProofWriter(trace.StorageTrace)
	if err != nil {
		t.Fatal(err)
	}

	od := zkproof.NewSimpleOrderer()
	theTx := trace.ExecutionResults[0]
	zkproof.HandleTx(od, theTx)

	t.Log(od)

	for _, op := range od.SavedOp() {
		_, err = writer.HandleNewState(op)
		if err != nil {
			t.Fatal(err)
		}
	}

	traces, err := zkproof.HandleBlockResult(trace)
	t.Log("traces: ", len(traces))
	outObj, _ := json.Marshal(traces)
	t.Log(string(outObj))
	if err != nil {
		t.Fatal(err)
	}
}

func TestTokenTx(t *testing.T) {
	trace := loadStaff(t, "token.json")
	traces, err := zkproof.HandleBlockResult(trace)
	outObj, _ := json.Marshal(traces)
	t.Log(string(outObj))
	if err != nil {
		t.Fatal(err)
	}

}

func TestCallTx(t *testing.T) {
	trace := loadStaff(t, "call.json")
	traces, err := zkproof.HandleBlockResult(trace)
	outObj, _ := json.Marshal(traces)
	t.Log(string(outObj))
	if err != nil {
		t.Fatal(err)
	}

	trace = loadStaff(t, "call_edge.json")
	traces, err = zkproof.HandleBlockResult(trace)
	outObj, _ = json.Marshal(traces)
	t.Log(string(outObj))
	if err != nil {
		t.Fatal(err)
	}
}

func TestCreateTx(t *testing.T) {
	trace := loadStaff(t, "create.json")
	traces, err := zkproof.HandleBlockResult(trace)
	outObj, _ := json.Marshal(traces)
	t.Log(string(outObj))
	if err != nil {
		t.Fatal(err)
	}

	trace = loadStaff(t, "deploy.json")
	traces, err = zkproof.HandleBlockResult(trace)
	outObj, _ = json.Marshal(traces)
	t.Log(string(outObj))
	if err != nil {
		t.Fatal(err)
	}

}

func TestFailedCallTx(t *testing.T) {
	trace := loadStaff(t, "fail_call.json")
	traces, err := zkproof.HandleBlockResult(trace)
	outObj, _ := json.Marshal(traces)
	t.Log(string(outObj))
	if err != nil {
		t.Fatal(err)
	}

	trace = loadStaff(t, "fail_create.json")
	traces, err = zkproof.HandleBlockResult(trace)
	outObj, _ = json.Marshal(traces)
	t.Log(string(outObj))
	if err != nil {
		t.Fatal(err)
	}

}

//notice: now only work with OP_ORDER=2
func TestDeleteTx(t *testing.T) {
	trace := loadStaff(t, "delete.json")
	traces, err := zkproof.HandleBlockResult(trace)
	outObj, _ := json.Marshal(traces)
	t.Log(string(outObj))
	if err != nil {
		t.Fatal(err)
	}
}

//notice: now only work with OP_ORDER=2
func TestDestructTx(t *testing.T) {
	trace := loadStaff(t, "destruct.json")
	traces, err := zkproof.HandleBlockResult(trace)
	outObj, _ := json.Marshal(traces)
	t.Log(string(outObj))
	if err != nil {
		t.Fatal(err)
	}
}

func TestMutipleTx(t *testing.T) {
	trace := loadStaff(t, "multi_txs.json")
	traces, err := zkproof.HandleBlockResult(trace)
	outObj, _ := json.Marshal(traces)
	t.Log(string(outObj))
	if err != nil {
		t.Fatal(err)
	}
}
