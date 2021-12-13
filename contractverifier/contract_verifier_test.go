package contractverifier

import (
	"encoding/json"
	"fmt"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/common/math"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/stretchr/testify/require"

	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/state/snapshot"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/rlp"
	"io/ioutil"
	"math/big"
	"path/filepath"
	"strings"
	"testing"
	"unicode"
)

const (
	UNKNOWN = iota
	SELFDESTRUCT
	NORMAL_CALL
)

type contractVerifierTest struct {
	verifiers []verifierTest
}

func NewContractVerifier() *contractVerifierTest {
	return &contractVerifierTest{verifiers: make([]verifierTest, 0)}
}

type verifierTest struct {
	VerifierType int
	Op           vm.OpCode
	From         common.Address
	To           common.Address
	Input        string
	Value        string
}

func NewVerifierTest(verifierType int, op vm.OpCode, from, to common.Address, input []byte, value *big.Int) verifierTest {
	return verifierTest{
		VerifierType: verifierType,
		Op:           op,
		From:         from,
		To:           to,
		Input:        hexutil.Encode(input),
		Value:        fmt.Sprintf("0x%s", value.Text(16)),
	}
}

func (cv *contractVerifierTest) Verify(stateDB vm.StateDB, op vm.OpCode, from, to common.Address, input []byte, value *big.Int) error {
	verifierTest := NewVerifierTest(UNKNOWN, op, from, to, input, value)
	if op == vm.SELFDESTRUCT {
		verifierTest.VerifierType = SELFDESTRUCT
	} else if op == vm.CALL || op == vm.DELEGATECALL || op == vm.STATICCALL || op == vm.CALLCODE {
		verifierTest.VerifierType = NORMAL_CALL
	}
	cv.verifiers = append(cv.verifiers, verifierTest)
	return nil
}

func convertVerifierType(op vm.OpCode) int {
	result := UNKNOWN
	if op == vm.SELFDESTRUCT {
		result = SELFDESTRUCT
	} else if op == vm.CALL || op == vm.DELEGATECALL || op == vm.STATICCALL || op == vm.CALLCODE {
		result = NORMAL_CALL
	}
	return result
}

// callTrace is the result of a callTracer run.
type callTrace struct {
	Type    string          `json:"type"`
	From    common.Address  `json:"from"`
	To      common.Address  `json:"to"`
	Input   hexutil.Bytes   `json:"input"`
	Output  hexutil.Bytes   `json:"output"`
	Gas     *hexutil.Uint64 `json:"gas,omitempty"`
	GasUsed *hexutil.Uint64 `json:"gasUsed,omitempty"`
	Value   *hexutil.Big    `json:"value,omitempty"`
	Error   string          `json:"error,omitempty"`
	Calls   []callTrace     `json:"calls,omitempty"`
}

func (ct *callTrace) GetArrayCall() []verifierTest {
	results := make([]verifierTest, 0)
	self := NewVerifierTest(convertVerifierType(vm.StringToOp(ct.Type)), vm.StringToOp(ct.Type), ct.From, ct.To, nil, nil)
	self.Input = ct.Input.String()
	self.Value = ct.Value.String()
	results = append(results, self)

	callsNumber := len(ct.Calls)
	if callsNumber == 0 {
		return results
	}

	for i, _ := range ct.Calls {
		nextResults := ct.Calls[i].GetArrayCall()
		results = append(results, nextResults...)
	}
	return results
}

type callContext struct {
	Number     math.HexOrDecimal64   `json:"number"`
	Difficulty *math.HexOrDecimal256 `json:"difficulty"`
	Time       math.HexOrDecimal64   `json:"timestamp"`
	GasLimit   math.HexOrDecimal64   `json:"gasLimit"`
	Miner      common.Address        `json:"miner"`
}

// callTracerTest defines a single test to check the call tracer against.
type callTracerTest struct {
	Genesis *core.Genesis `json:"genesis"`
	Context *callContext  `json:"context"`
	Input   string        `json:"input"`
	Result  *callTrace    `json:"result"`
}

func (ct callTracerTest) GetArrayResult() []verifierTest {
	return ct.Result.GetArrayCall()
}

// camel converts a snake cased input string into a camel cased output.
func camel(str string) string {
	pieces := strings.Split(str, "_")
	for i := 1; i < len(pieces); i++ {
		pieces[i] = string(unicode.ToUpper(rune(pieces[i][0]))) + pieces[i][1:]
	}
	return strings.Join(pieces, "")
}

func makePreState(db ethdb.Database, accounts core.GenesisAlloc, snapshotter bool) (*snapshot.Tree, *state.StateDB) {
	sdb := state.NewDatabase(db)
	statedb, _ := state.New(common.Hash{}, sdb, nil)
	for addr, a := range accounts {
		statedb.SetCode(addr, a.Code)
		statedb.SetNonce(addr, a.Nonce)
		statedb.SetBalance(addr, a.Balance)
		for k, v := range a.Storage {
			statedb.SetState(addr, k, v)
		}
	}
	// Commit and re-open to start with a clean state.
	root, _ := statedb.Commit(false)

	var snaps *snapshot.Tree
	if snapshotter {
		snaps, _ = snapshot.New(db, sdb.TrieDB(), 1, root, false, true, false)
	}
	statedb, _ = state.New(root, sdb, snaps)
	return snaps, statedb
}

// Iterates over all the input-output datasets in the tracer test harness and
// runs the JavaScript tracers against them.
func TestOKVerify(t *testing.T) {
	files, err := ioutil.ReadDir("testdata")
	if err != nil {
		t.Fatalf("failed to retrieve tracer test suite: %v", err)
	}
	//index := 0
	for _, file := range files {
		//if i < index {
		//	continue
		//} else if i > index {
		//	return
		//} else {
		//	fmt.Println("@@@@@@@@@@@@@@@@@@@@", file.Name())
		//}
		file := file // capture range variable
		t.Run(camel(strings.TrimSuffix(file.Name(), ".json")), func(t *testing.T) {
			t.Parallel()

			// Call tracer test found, read if from disk
			blob, err := ioutil.ReadFile(filepath.Join("testdata", file.Name()))
			if err != nil {
				t.Fatalf("failed to read testcase: %v", err)
			}
			test := new(callTracerTest)
			if err := json.Unmarshal(blob, test); err != nil {
				t.Fatalf("failed to parse testcase: %v", err)
			}
			// Configure a blockchain with the given prestate
			tx := new(types.Transaction)
			if err := rlp.DecodeBytes(common.FromHex(test.Input), tx); err != nil {
				t.Fatalf("failed to parse testcase input: %v", err)
			}
			signer := types.MakeSigner(test.Genesis.Config, new(big.Int).SetUint64(uint64(test.Context.Number)))
			origin, _ := signer.Sender(tx)
			txContext := vm.TxContext{
				Origin:   origin,
				GasPrice: tx.GasPrice(),
			}
			context := vm.BlockContext{
				CanTransfer: core.CanTransfer,
				Transfer:    core.Transfer,
				Coinbase:    test.Context.Miner,
				BlockNumber: new(big.Int).SetUint64(uint64(test.Context.Number)),
				Time:        new(big.Int).SetUint64(uint64(test.Context.Time)),
				Difficulty:  (*big.Int)(test.Context.Difficulty),
				GasLimit:    uint64(test.Context.GasLimit),
			}
			_, statedb := makePreState(rawdb.NewMemoryDatabase(), test.Genesis.Alloc, false)

			verifierTest := NewContractVerifier()
			evm := vm.NewEVM(context, txContext, statedb, test.Genesis.Config, vm.Config{ContractVerifier: verifierTest})

			msg, err := tx.AsMessage(signer, nil)
			if err != nil {
				t.Fatalf("failed to prepare transaction for tracing: %v", err)
			}
			st := core.NewStateTransition(evm, msg, new(core.GasPool).AddGas(tx.Gas()))
			if _, err = st.TransitionDb(); err != nil {
				t.Fatalf("failed to execute transaction: %v", err)
			}
			actual := verifierTest.verifiers
			excepted := test.GetArrayResult()
			require.Equal(t, excepted, actual)
		})
	}
}
