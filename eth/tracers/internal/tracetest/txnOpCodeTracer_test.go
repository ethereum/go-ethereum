package tracetest

import (
	"encoding/json"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/eth/tracers"
	"github.com/ethereum/go-ethereum/eth/tracers/blocknative"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/ethereum/go-ethereum/tests"
	"math/big"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

type txnOpCodeTracerTest struct {
	Genesis      *core.Genesis      `json:"genesis"`
	Context      *callContext       `json:"context"`
	Input        string             `json:"input"`
	TracerConfig json.RawMessage    `json:"tracerConfig"`
	Result       *blocknative.Trace `json:"result"`
}

func TestTxnOpCodeTracer(t *testing.T) {
	testTxnOpCodeTracer("txnOpCodeTracer", "txnOpCode_tracer", t)
}

func testTxnOpCodeTracer(tracerName string, dirPath string, t *testing.T) {
	files, err := os.ReadDir(filepath.Join("testdata", dirPath))
	if err != nil {
		t.Fatalf("failed to retrieve tracer test suite: %v", err)
	}
	for _, file := range files {
		if !strings.HasSuffix(file.Name(), ".json") {
			continue
		}
		file := file
		t.Run(camel(strings.TrimSuffix(file.Name(), ".json")), func(t *testing.T) {
			t.Parallel()

			var (
				test = new(txnOpCodeTracerTest)
				tx   = new(types.Transaction)
			)
			// Call tracer test found, read if from disk
			if blob, err := os.ReadFile(filepath.Join("testdata", dirPath, file.Name())); err != nil {
				t.Fatalf("failed to read testcase: %v", err)
			} else if err := json.Unmarshal(blob, test); err != nil {
				t.Fatalf("failed to parse testcase: %v", err)
			}
			if err := rlp.DecodeBytes(common.FromHex(test.Input), tx); err != nil {
				t.Fatalf("failed to parse testcase input: %v", err)
			}

			baseFee := big.NewInt(0xFF0000)
			if test.Context.BaseFee != 0 {
				baseFee = new(big.Int).SetUint64(uint64(test.Context.BaseFee))
			}

			// Configure a blockchain with the given prestate
			var (
				signer    = types.MakeSigner(test.Genesis.Config, new(big.Int).SetUint64(uint64(test.Context.Number)))
				origin, _ = signer.Sender(tx)
				txContext = vm.TxContext{
					Origin:   origin,
					GasPrice: tx.GasPrice(),
				}
				context = vm.BlockContext{
					CanTransfer: core.CanTransfer,
					Transfer:    core.Transfer,
					Coinbase:    test.Context.Miner,
					BlockNumber: new(big.Int).SetUint64(uint64(test.Context.Number)),
					Time:        new(big.Int).SetUint64(uint64(test.Context.Time)),
					Difficulty:  (*big.Int)(test.Context.Difficulty),
					GasLimit:    uint64(test.Context.GasLimit),
					BaseFee:     baseFee,
					Random:      test.Context.Random,
				}
				_, statedb = tests.MakePreState(rawdb.NewMemoryDatabase(), test.Genesis.Alloc, false)
			)
			tracer, err := tracers.New(tracerName, new(tracers.Context), test.TracerConfig)
			if err != nil {
				t.Fatalf("failed to create call tracer: %v", err)
			}
			evm := vm.NewEVM(context, txContext, statedb, test.Genesis.Config, vm.Config{Debug: true, Tracer: tracer})
			msg, err := tx.AsMessage(signer, nil)
			if err != nil {
				t.Fatalf("failed to prepare transaction for tracing: %v", err)
			}
			st := core.NewStateTransition(evm, msg, new(core.GasPool).AddGas(tx.Gas()))
			if _, err = st.TransitionDb(); err != nil {
				t.Fatalf("failed to execute transaction: %v", err)
			}
			res, err := tracer.GetResult()
			if err != nil {
				t.Fatalf("failed to retrieve trace result: %v", err)
			}
			ret := new(blocknative.Trace)
			if err := json.Unmarshal(res, ret); err != nil {
				t.Fatalf("failed to unmarshal trace result: %v", err)
			}

			if !tracesEqual(ret, test.Result) {
				t.Fatalf("trace mismatch: \nhave %+v\nwant %+v", ret, test.Result)
			}
		})
	}
}

func tracesEqual(x, y *blocknative.Trace) bool {
	// Clear out non-deterministic time
	x.Time = ""
	y.Time = ""

	xTrace := new(blocknative.Trace)
	yTrace := new(blocknative.Trace)
	if xj, err := json.Marshal(x); err == nil {
		if json.Unmarshal(xj, xTrace) != nil {
			return false
		}
	} else {
		return false
	}
	if yj, err := json.Marshal(y); err == nil {
		if json.Unmarshal(yj, yTrace) != nil {
			return false
		}
	} else {
		return false
	}

	return reflect.DeepEqual(xTrace, yTrace)
}
