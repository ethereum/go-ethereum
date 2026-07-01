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

package vm

// Tests for the generated interpreter dispatch (interpreter_gen.go): that the
// committed file is up to date, that it behaves identically to the table loop,
// and that the fast path keeps its cheap stack helpers inlined.

import (
	"bytes"
	"go/ast"
	"go/parser"
	"go/token"
	"math/big"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/tracing"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/params"
	"github.com/holiman/uint256"
)

// TestGeneratedDispatchUpToDate asserts that the committed interpreter_gen.go matches
// what `go generate` (core/vm/gen) produces from the current opcode/gas/fork
// definitions. It is the CI guard against hand-edits to the generated file and
// against the generator drifting from the committed output.
func TestGeneratedDispatchUpToDate(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping generator round-trip in -short mode")
	}
	tmp := filepath.Join(t.TempDir(), "interpreter_gen.go")
	cmd := exec.Command("go", "run", "./gen")
	cmd.Env = append(os.Environ(), "INTERPRETER_GEN_OUT="+tmp)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("running generator: %v\n%s", err, out)
	}
	got, err := os.ReadFile(tmp)
	if err != nil {
		t.Fatalf("reading regenerated output: %v", err)
	}
	want, err := os.ReadFile("interpreter_gen.go")
	if err != nil {
		t.Fatalf("reading committed interpreter_gen.go: %v", err)
	}
	if string(got) != string(want) {
		t.Fatalf("interpreter_gen.go is out of date; run `go generate ./core/vm/...` and commit the result")
	}
}

// Differential test comparing the table loop against the generated dispatch.
//
// These tests prove that the generated dispatch (execUntraced) is bit-identical
// to the table-walking loop (execTraced, run here without a tracer via
// EVM.forceTableLoop) for the observable surface of an EVM execution: return
// data, gas left, error/halt, refund counter, emitted logs, and the resulting
// state root. It runs the same program through both interpreters over freshly
// built, identical state across several forks, plus a fuzz target over
// arbitrary bytecode.
//
// execTraced is also the production tracing path, and
// if it drifted from the generated dispatch then traced re-execution would
// disagree with what consensus executed. Hook emission itself is covered by
// the tracer test suites instead.

// diffForks is the set of fork configurations the diff test runs every program
// under. Spanning forks exercises the baked runtime fork gates (e.g. SHL from
// Constantinople, PUSH0 from Shanghai, CLZ from Osaka) in both the active and
// the not-yet-activated states.
var diffForks = func() []struct {
	name   string
	cfg    *params.ChainConfig
	merged bool
} {
	// preConstantinople: Byzantium active, Constantinople and later not.
	preCon := *params.TestChainConfig
	preCon.ConstantinopleBlock = nil
	preCon.PetersburgBlock = nil
	preCon.IstanbulBlock = nil
	preCon.MuirGlacierBlock = nil
	preCon.BerlinBlock = nil
	preCon.LondonBlock = nil
	preCon.ArrowGlacierBlock = nil
	preCon.GrayGlacierBlock = nil

	// amsterdam: Merged plus the Amsterdam (EIP-8037) timestamp, so the diff test
	// exercises the multidimensional gas accounting (regular + state gas). Without
	// this lane a state-gas charging divergence between the two interpreters would
	// go unnoticed.
	ams := *params.MergedTestChainConfig
	amsTime := uint64(0)
	ams.AmsterdamTime = &amsTime

	return []struct {
		name   string
		cfg    *params.ChainConfig
		merged bool
	}{
		{"Frontier", params.NonActivatedConfig, false},
		{"Byzantium", &preCon, false},
		{"London", params.TestChainConfig, false},
		{"Merged", params.MergedTestChainConfig, true},
		{"Amsterdam", &ams, true},
	}
}()

var (
	diffContractAddr = common.HexToAddress("0x000000000000000000000000000000000000c0de")
	diffCalleeAddr   = common.HexToAddress("0x000000000000000000000000000000000000ca11")
	diffCaller       = common.HexToAddress("0x000000000000000000000000000000000000face")
)

// diffCalleeCode is deployed at diffCalleeAddr as a CALL/CREATE target: it
// writes a storage slot, logs, and returns 32 bytes of memory.
//
//	PUSH1 0x2a PUSH1 0x07 SSTORE          // sstore(7, 42)
//	PUSH1 0xbb PUSH1 0x00 MSTORE          // mem[0..32] = 0xbb
//	PUSH1 0x20 PUSH1 0x00 LOG0            // log0(mem[0:32])
//	PUSH1 0x20 PUSH1 0x00 RETURN          // return mem[0:32]
var diffCalleeCode = []byte{
	byte(PUSH1), 0x2a, byte(PUSH1), 0x07, byte(SSTORE),
	byte(PUSH1), 0xbb, byte(PUSH1), 0x00, byte(MSTORE),
	byte(PUSH1), 0x20, byte(PUSH1), 0x00, byte(LOG0),
	byte(PUSH1), 0x20, byte(PUSH1), 0x00, byte(RETURN),
}

// asm is a tiny helper to build bytecode from opcodes/immediates.
func asm(parts ...any) []byte {
	var b []byte
	for _, p := range parts {
		switch v := p.(type) {
		case OpCode:
			b = append(b, byte(v))
		case byte:
			b = append(b, v)
		case int:
			b = append(b, byte(v))
		case []byte:
			b = append(b, v...)
		default:
			panic("asm: bad part")
		}
	}
	return b
}

// diffPrograms is a curated set of bytecode snippets covering the inlined hot
// opcodes, the volatile call-through opcodes, fork-gated opcodes, control flow,
// and the principal error paths.
var diffPrograms = []struct {
	name string
	code []byte
	gas  uint64
}{
	{"arith", asm(PUSH1, 0x07, PUSH1, 0x03, ADD, PUSH1, 0x02, MUL, PUSH1, 0x04, SUB, PUSH1, 0x03, DIV, PUSH1, 0x05, MOD, PUSH1, 0x00, MSTORE, PUSH1, 0x20, PUSH1, 0x00, RETURN), 100000},
	{"signed-arith", asm(PUSH1, 0x07, PUSH1, 0xfd, SDIV, PUSH1, 0x03, SMOD, PUSH1, 0x02, SIGNEXTEND, PUSH1, 0x00, MSTORE, PUSH1, 0x20, PUSH1, 0x00, RETURN), 100000},
	{"addmod-mulmod-exp", asm(PUSH1, 0x07, PUSH1, 0x05, PUSH1, 0x03, ADDMOD, PUSH1, 0x09, PUSH1, 0x04, PUSH1, 0x02, MULMOD, PUSH1, 0x03, PUSH1, 0x02, EXP, ADD, ADD, PUSH1, 0x00, MSTORE, PUSH1, 0x20, PUSH1, 0x00, RETURN), 100000},
	{"cmp", asm(PUSH1, 0x07, PUSH1, 0x03, LT, PUSH1, 0x01, GT, PUSH1, 0x01, SLT, PUSH1, 0x01, SGT, PUSH1, 0x01, EQ, ISZERO, PUSH1, 0x00, MSTORE, PUSH1, 0x20, PUSH1, 0x00, RETURN), 100000},
	{"bitwise", asm(PUSH1, 0xf0, PUSH1, 0x0f, AND, PUSH1, 0xaa, OR, PUSH1, 0x55, XOR, NOT, PUSH1, 0x01, BYTE, PUSH1, 0x00, MSTORE, PUSH1, 0x20, PUSH1, 0x00, RETURN), 100000},
	{"shifts-clz", asm(PUSH1, 0xff, PUSH1, 0x04, SHL, PUSH1, 0x02, SHR, PUSH1, 0x01, SAR, CLZ, PUSH1, 0x00, MSTORE, PUSH1, 0x20, PUSH1, 0x00, RETURN), 100000},
	{"dup-swap", asm(PUSH1, 0x01, PUSH1, 0x02, PUSH1, 0x03, DUP3, SWAP2, DUP1, SWAP1, POP, ADD, ADD, ADD, PUSH1, 0x00, MSTORE, PUSH1, 0x20, PUSH1, 0x00, RETURN), 100000},
	{"push0-push32", asm(PUSH0, PUSH3, 0x01, 0x02, 0x03, ADD, PUSH5, 0x01, 0x02, 0x03, 0x04, 0x05, ADD, PUSH1, 0x00, MSTORE, PUSH1, 0x20, PUSH1, 0x00, RETURN), 100000},
	{"keccak", asm(PUSH1, 0x20, PUSH1, 0x00, KECCAK256, PUSH1, 0x00, MSTORE, PUSH1, 0x20, PUSH1, 0x00, RETURN), 100000},
	{"memory", asm(PUSH1, 0xab, PUSH1, 0x00, MSTORE8, PUSH1, 0xcd, PUSH2, 0x00, 0x40, MSTORE, MSIZE, PUSH1, 0x60, MSTORE, PUSH1, 0x80, PUSH1, 0x00, RETURN), 100000},
	{"mcopy", asm(PUSH1, 0xff, PUSH1, 0x00, MSTORE, PUSH1, 0x20, PUSH1, 0x00, PUSH1, 0x20, MCOPY, PUSH1, 0x40, PUSH1, 0x00, RETURN), 100000},
	{"storage", asm(PUSH1, 0x63, PUSH1, 0x07, SSTORE, PUSH1, 0x07, SLOAD, PUSH1, 0x00, MSTORE, PUSH1, 0x20, PUSH1, 0x00, RETURN), 100000},
	{"transient", asm(PUSH1, 0x63, PUSH1, 0x07, TSTORE, PUSH1, 0x07, TLOAD, PUSH1, 0x00, MSTORE, PUSH1, 0x20, PUSH1, 0x00, RETURN), 100000},
	{"loop", asm(PUSH1, 0x00, JUMPDEST, PUSH1, 0x01, ADD, DUP1, PUSH1, 0x05, LT, PUSH1, 0x02, JUMPI, PUSH1, 0x00, MSTORE, PUSH1, 0x20, PUSH1, 0x00, RETURN), 100000},
	{"jump", asm(PUSH1, 0x06, JUMP, INVALID, INVALID, JUMPDEST, PUSH1, 0x2a, PUSH1, 0x00, MSTORE, PUSH1, 0x20, PUSH1, 0x00, RETURN), 100000},
	{"env", asm(ADDRESS, CALLER, CALLVALUE, ORIGIN, GASPRICE, CODESIZE, GAS, PC, ADD, ADD, ADD, ADD, ADD, ADD, PUSH1, 0x00, MSTORE, PUSH1, 0x20, PUSH1, 0x00, RETURN), 100000},
	{"block", asm(NUMBER, TIMESTAMP, COINBASE, GASLIMIT, CHAINID, SELFBALANCE, BASEFEE, DIFFICULTY, ADD, ADD, ADD, ADD, ADD, ADD, ADD, PUSH1, 0x00, MSTORE, PUSH1, 0x20, PUSH1, 0x00, RETURN), 100000},
	{"calldata", asm(PUSH1, 0x00, CALLDATALOAD, CALLDATASIZE, PUSH1, 0x00, PUSH1, 0x00, CALLDATACOPY, ADD, PUSH1, 0x00, MSTORE, PUSH1, 0x20, PUSH1, 0x00, RETURN), 100000},
	{"codecopy", asm(PUSH1, 0x10, PUSH1, 0x00, PUSH1, 0x00, CODECOPY, PUSH1, 0x10, PUSH1, 0x00, RETURN), 100000},
	{"log", asm(PUSH1, 0x11, PUSH1, 0x00, MSTORE, PUSH1, 0x22, PUSH1, 0x33, PUSH1, 0x20, PUSH1, 0x00, LOG2, STOP), 100000},
	// Fuzz-found regression (the stale-res bug): a res-setting DELEGATECALL
	// followed by a halting inlined op (JUMPI to an invalid destination). The
	// buggy build returned the DELEGATECALL output instead of nil.
	{"delegatecall-then-invalid-jumpi", asm(
		PUSH1, 0x30, PUSH1, 0x30, PUSH1, 0x30, PUSH1, 0x30,
		PUSH20, diffCalleeAddr.Bytes(),
		PUSH2, 0x30, 0x30, DELEGATECALL,
		PC, PC, JUMPI), 100000},
	{"extaccess", asm(PUSH20, diffCalleeAddr.Bytes(), EXTCODESIZE, PUSH20, diffCalleeAddr.Bytes(), BALANCE, ADD, PUSH1, 0x00, MSTORE, PUSH1, 0x20, PUSH1, 0x00, RETURN), 100000},
	{"call", asm(
		PUSH1, 0x20, PUSH1, 0x00, PUSH1, 0x00, PUSH1, 0x00, PUSH1, 0x00,
		PUSH20, diffCalleeAddr.Bytes(), PUSH2, 0xff, 0xff, CALL,
		PUSH1, 0x20, PUSH1, 0x00, RETURN), 200000},
	{"staticcall", asm(
		PUSH1, 0x20, PUSH1, 0x00, PUSH1, 0x00, PUSH1, 0x00,
		PUSH20, diffCalleeAddr.Bytes(), PUSH2, 0xff, 0xff, STATICCALL,
		PUSH1, 0x20, PUSH1, 0x00, RETURN), 200000},
	{"delegatecall", asm(
		PUSH1, 0x20, PUSH1, 0x00, PUSH1, 0x00, PUSH1, 0x00,
		PUSH20, diffCalleeAddr.Bytes(), PUSH2, 0xff, 0xff, DELEGATECALL,
		PUSH1, 0x20, PUSH1, 0x00, RETURN), 200000},
	{"create", asm(
		// store init code that returns empty, then CREATE
		PUSH1, 0x00, PUSH1, 0x00, MSTORE,
		PUSH1, 0x00, PUSH1, 0x00, PUSH1, 0x00, CREATE,
		PUSH1, 0x00, MSTORE, PUSH1, 0x20, PUSH1, 0x00, RETURN), 200000},
	{"revert", asm(PUSH1, 0xaa, PUSH1, 0x00, MSTORE, PUSH1, 0x20, PUSH1, 0x00, REVERT), 100000},
	{"selfdestruct", asm(PUSH20, diffCaller.Bytes(), SELFDESTRUCT), 100000},
	{"stop", asm(PUSH1, 0x01, STOP), 100000},
	{"invalid-opcode", asm(PUSH1, 0x01, INVALID), 100000},
	{"undefined-opcode", asm(PUSH1, 0x01, 0x0c), 100000},
	{"stack-underflow", asm(ADD), 100000},
	{"oog", asm(PUSH1, 0x07, PUSH1, 0x03, ADD, PUSH1, 0x00, MSTORE, PUSH1, 0x20, PUSH1, 0x00, RETURN), 7},
	{"invalid-jump", asm(PUSH1, 0x03, JUMP, STOP), 100000},
}

// diffResult captures the observable outcome of running a program.
type diffResult struct {
	ret     []byte
	gasLeft uint64
	errStr  string // "" if no error
	refund  uint64
	root    common.Hash
	logs    []*types.Log
}

func (r diffResult) equal(o diffResult) (string, bool) {
	if !bytes.Equal(r.ret, o.ret) {
		return "return data", false
	}
	if r.gasLeft != o.gasLeft {
		return "gas left", false
	}
	if r.errStr != o.errStr {
		return "error", false
	}
	if r.refund != o.refund {
		return "refund", false
	}
	if r.root != o.root {
		return "state root", false
	}
	if len(r.logs) != len(o.logs) {
		return "log count", false
	}
	for i := range r.logs {
		a, b := r.logs[i], o.logs[i]
		if a.Address != b.Address || !bytes.Equal(a.Data, b.Data) || len(a.Topics) != len(b.Topics) {
			return "log content", false
		}
		for j := range a.Topics {
			if a.Topics[j] != b.Topics[j] {
				return "log topic", false
			}
		}
	}
	return "", true
}

func newDiffState(t testing.TB) *state.StateDB {
	t.Helper()
	statedb, err := state.New(types.EmptyRootHash, state.NewDatabaseForTesting())
	if err != nil {
		t.Fatalf("state.New: %v", err)
	}
	// Main contract: balance + a pre-set storage slot.
	statedb.CreateAccount(diffContractAddr)
	statedb.SetBalance(diffContractAddr, uint256.NewInt(1000), tracing.BalanceChangeUnspecified)
	statedb.SetState(diffContractAddr, common.Hash{31: 0x07}, common.Hash{31: 0x07})
	// Callee target for CALL/STATICCALL/DELEGATECALL.
	statedb.CreateAccount(diffCalleeAddr)
	statedb.SetBalance(diffCalleeAddr, uint256.NewInt(500), tracing.BalanceChangeUnspecified)
	statedb.SetCode(diffCalleeAddr, diffCalleeCode, tracing.CodeChangeUnspecified)
	// Caller EOA with a balance.
	statedb.CreateAccount(diffCaller)
	statedb.SetBalance(diffCaller, uint256.NewInt(1<<62), tracing.BalanceChangeUnspecified)
	statedb.Finalise(true)
	return statedb
}

func diffBlockCtx(merged bool) BlockContext {
	ctx := BlockContext{
		CanTransfer: func(StateDB, common.Address, *uint256.Int) bool { return true },
		Transfer:    func(StateDB, common.Address, common.Address, *uint256.Int, *params.Rules) {},
		GetHash:     func(uint64) common.Hash { return common.Hash{0xde, 0xad} },
		Coinbase:    common.HexToAddress("0xc01ba5e"),
		BlockNumber: big.NewInt(8),
		Time:        1234,
		Difficulty:  big.NewInt(0x20000),
		GasLimit:    30_000_000,
		BaseFee:     big.NewInt(7),
		BlobBaseFee: big.NewInt(3),
		// Price state gas as mainnet does (see core.NewEVMBlockContext), so the
		// Amsterdam diff lane exercises the EIP-8037 state-gas charging path.
		CostPerStateByte: params.CostPerStateByte,
	}
	if merged {
		h := common.HexToHash("0x0102030405060708090a0b0c0d0e0f101112131415161718191a1b1c1d1e1f20")
		ctx.Random = &h
	}
	return ctx
}

// TestExtraEIPs checks that EIPs enabled via Config.ExtraEips take effect even
// when they touch opcodes the generated dispatch inlines. PUSH0 (EIP-3855) on
// a pre-Shanghai config is the canary: the runtime table has it enabled but
// the baked fork gate does not, so execution must route through the table loop.
func TestExtraEIPs(t *testing.T) {
	code := asm(PUSH0, STOP)
	statedb := newDiffState(t)
	statedb.SetCode(diffContractAddr, code, tracing.CodeChangeUnspecified)
	statedb.Finalise(true)

	evm := NewEVM(diffBlockCtx(false), statedb, params.TestChainConfig, Config{ExtraEips: []int{3855}})
	evm.SetTxContext(TxContext{
		Origin:   diffCaller,
		GasPrice: uint256.NewInt(1),
	})
	_, _, err := evm.Call(diffCaller, diffContractAddr, nil, NewGasBudget(100000, 0), new(uint256.Int))
	if err != nil {
		t.Fatalf("PUSH0 enabled via ExtraEips failed: %v", err)
	}
}

// runOne executes code at diffContractAddr with the given interpreter selection
// and returns the observable result.
func runOne(t testing.TB, cfg *params.ChainConfig, merged, useTableLoop bool, code, input []byte, gas uint64) diffResult {
	t.Helper()
	statedb := newDiffState(t)
	statedb.SetCode(diffContractAddr, code, tracing.CodeChangeUnspecified)
	statedb.Finalise(true)

	evm := NewEVM(diffBlockCtx(merged), statedb, cfg, Config{})
	evm.SetTxContext(TxContext{
		Origin:     diffCaller,
		GasPrice:   uint256.NewInt(1),
		BlobHashes: []common.Hash{{0xb1, 0x0b}},
	})
	evm.forceTableLoop = useTableLoop

	ret, leftOver, err := evm.Call(diffCaller, diffContractAddr, input, NewGasBudget(gas, 0), new(uint256.Int))
	errStr := ""
	if err != nil {
		errStr = err.Error()
	}
	return diffResult{
		ret:     ret,
		gasLeft: leftOver.RegularGas,
		errStr:  errStr,
		refund:  statedb.GetRefund(),
		root:    statedb.IntermediateRoot(true),
		logs:    statedb.Logs(),
	}
}

func TestInterpreterDiff(t *testing.T) {
	for _, fk := range diffForks {
		for _, prog := range diffPrograms {
			t.Run(fk.name+"/"+prog.name, func(t *testing.T) {
				input := common.FromHex("0xdeadbeef00000000000000000000000000000000000000000000000000000042")
				table := runOne(t, fk.cfg, fk.merged, true, prog.code, input, prog.gas)
				gen := runOne(t, fk.cfg, fk.merged, false, prog.code, input, prog.gas)
				if where, ok := gen.equal(table); !ok {
					t.Fatalf("divergence in %s:\n  table: ret=%x gas=%d err=%q refund=%d root=%x logs=%d\n  gen:   ret=%x gas=%d err=%q refund=%d root=%x logs=%d",
						where,
						table.ret, table.gasLeft, table.errStr, table.refund, table.root, len(table.logs),
						gen.ret, gen.gasLeft, gen.errStr, gen.refund, gen.root, len(gen.logs))
				}
			})
		}
	}
}

// FuzzInterpreterDiff fuzzes arbitrary bytecode + calldata + gas and asserts the
// generated dispatch matches the table-walking loop on every observable axis.
func FuzzInterpreterDiff(f *testing.F) {
	for _, prog := range diffPrograms {
		f.Add(prog.code, []byte{0x01, 0x02, 0x03, 0x04}, uint64(100000))
	}
	// A couple of structurally-interesting seeds.
	f.Add(asm(PUSH1, 0x00, JUMPDEST, PUSH1, 0x01, ADD, DUP1, PUSH1, 0xff, GT, PUSH1, 0x02, JUMPI, STOP), []byte{}, uint64(50000))
	f.Add(bytes.Repeat([]byte{byte(PUSH1), 0x01}, 64), []byte{}, uint64(100000))

	f.Fuzz(func(t *testing.T, code, input []byte, gas uint64) {
		if len(code) > 24576 { // max contract code size, keep cases realistic
			return
		}
		if gas > 5_000_000 {
			gas = 5_000_000 // bound execution time
		}
		for _, fk := range diffForks {
			table := runOne(t, fk.cfg, fk.merged, true, code, input, gas)
			gen := runOne(t, fk.cfg, fk.merged, false, code, input, gas)
			if where, ok := gen.equal(table); !ok {
				t.Fatalf("divergence in %s (fork %s): code=%x input=%x gas=%d\n  table: ret=%x gas=%d err=%q refund=%d root=%x logs=%d\n  gen:   ret=%x gas=%d err=%q refund=%d root=%x logs=%d",
					where, fk.name, code, input, gas,
					table.ret, table.gasLeft, table.errStr, table.refund, table.root, len(table.logs),
					gen.ret, gen.gasLeft, gen.errStr, gen.refund, gen.root, len(gen.logs))
			}
		}
	})
}

// markedHelpers parses stack.go and returns the *Stack helpers tagged
// //gen:inline. That tag is the single source of truth, shared with the
// generator (core/vm/gen), for which helpers are spliced into the dispatch.
func markedHelpers(t *testing.T) map[string]bool {
	t.Helper()
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, "stack.go", nil, parser.ParseComments)
	if err != nil {
		t.Fatalf("parsing stack.go: %v", err)
	}
	marked := map[string]bool{}
	for _, decl := range f.Decls {
		fn, ok := decl.(*ast.FuncDecl)
		if !ok || fn.Doc == nil {
			continue
		}
		for _, c := range fn.Doc.List {
			if c.Text == "//gen:inline" {
				marked[fn.Name.Name] = true
			}
		}
	}
	if len(marked) == 0 {
		t.Fatal("found no //gen:inline helpers in stack.go")
	}
	return marked
}

// TestGeneratedFastPathHelpersExpanded asserts the generator spliced every
// //gen:inline helper inline, so none survives as a real call in interpreter_gen.go.
// Those helpers exceed the compiler's inline budget for a function as large as
// execUntraced, so a missed splice would silently drop the inlining the fast
// path exists for. It is the expand-side counterpart to
// TestGeneratedFastPathHelpersInlined: together they hold the one invariant that
// the fast path makes no real stack-helper call, the costly ones by splicing,
// the cheap ones by compiler inlining.
func TestGeneratedFastPathHelpersExpanded(t *testing.T) {
	calls := countStackCalls(t, "interpreter_gen.go")
	for h := range markedHelpers(t) {
		if n := calls[h]; n != 0 {
			t.Errorf("(*Stack).%s is //gen:inline but has %d residual call(s) in interpreter_gen.go, expected 0.\n"+
				"The generator did not splice it. Check it is still in inlinable shape (core/vm/gen).", h, n)
		}
	}
}

// TestGeneratedFastPathHelpersInlined recompiles this package with the
// compiler's inlining diagnostics on and fails if any *Stack helper call that
// survives into interpreter_gen.go was not inlined. Every survivor must be a cheap
// helper (len, pop1, peek, drop) the compiler inlines into execUntraced; the
// //gen:inline helpers are spliced away and owned by the Expanded test. The
// cheap ones inline today with margin except pop1, at cost 18 against Go's
// big-function budget of 20. A toolchain that re-scores inline cost, or an extra
// branch in one of these bodies, could silently stop the inlining and slow the
// interpreter, so this turns that into a red build.
func TestGeneratedFastPathHelpersInlined(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping inlining check (recompiles the package) in -short mode")
	}

	// go build -gcflags=-m prints every inlining decision. The build cache
	// replays the diagnostics on a hit, so repeated runs are deterministic. The
	// flag applies only to this package, cached dependencies stay quiet.
	out, err := exec.Command("go", "build", "-gcflags=-m", ".").CombinedOutput()
	if err != nil {
		t.Fatalf("compiling with inlining diagnostics: %v\n%s", err, out)
	}
	diag := string(out)
	if !strings.Contains(diag, "interpreter_gen.go") {
		t.Fatalf("captured no interpreter_gen.go diagnostics, the -m build produced nothing to check:\n%s", diag)
	}

	// Every surviving stack-helper call (i.e. not a //gen:inline target) must be
	// inlined by the compiler.
	marked := markedHelpers(t)
	for h, n := range countStackCalls(t, "interpreter_gen.go") {
		if marked[h] {
			continue // spliced away, owned by TestGeneratedFastPathHelpersExpanded
		}
		inlinedRe := regexp.MustCompile(`interpreter_gen\.go.*inlining call to \(\*Stack\)\.` + regexp.QuoteMeta(h) + `\b`)
		inlined := len(inlinedRe.FindAllString(diag, -1))
		if inlined != n {
			t.Errorf("(*Stack).%s: %d call site(s) in interpreter_gen.go, %d inlined into execUntraced.\n"+
				"The compiler stopped inlining it, so the fast path now pays a real call. Shrink the\n"+
				"body to fit the inline budget, or tag it //gen:inline in stack.go to splice it instead.", h, n, inlined)
			continue
		}
		t.Logf("(*Stack).%s: %d/%d call sites inlined", h, inlined, n)
	}
}

// countStackCalls parses a generated source file and counts calls to each
// *Stack helper method, keyed by method name. It matches the fast path's stack
// local and scope.Stack receivers. Parsing rather than grepping keeps comments
// and strings from inflating the count.
func countStackCalls(t *testing.T, file string) map[string]int {
	t.Helper()
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, file, nil, 0)
	if err != nil {
		t.Fatalf("parsing %s: %v", file, err)
	}
	counts := map[string]int{}
	ast.Inspect(f, func(n ast.Node) bool {
		call, ok := n.(*ast.CallExpr)
		if !ok {
			return true
		}
		if sel, ok := call.Fun.(*ast.SelectorExpr); ok && isStackReceiver(sel.X) {
			counts[sel.Sel.Name]++
		}
		return true
	})
	return counts
}

// isStackReceiver reports whether x is the fast path's stack local or scope.Stack.
func isStackReceiver(x ast.Expr) bool {
	switch r := x.(type) {
	case *ast.Ident:
		return r.Name == "stack"
	case *ast.SelectorExpr:
		return r.Sel.Name == "Stack"
	}
	return false
}
