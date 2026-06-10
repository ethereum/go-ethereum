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

// Command gen generates core/vm/interp_gen.go, the EVM interpreter's untraced
// fast-path dispatch. The output is a switch over the opcode byte that:
//
//   - inlines the hot, fork-stable opcodes (arithmetic / comparison / bitwise /
//     PUSH / DUP / SWAP / POP / JUMP / JUMPI / PC / MSIZE / JUMPDEST) by
//     splicing the existing opXxx handler bodies from instructions.go and
//     eips.go, with their static gas and stack bounds baked in as constants
//     derived from the per-fork instruction tables via vm.GenForks.
//
//   - calls the fork-invariant cold ops (KECCAK256 / MLOAD / MSTORE / MSTORE8,
//     see directCold) directly by name, skipping the table's function
//     pointers, which Go cannot inline through.
//
//   - dispatches everything fork-varying (CALL / CREATE / SSTORE / SLOAD / LOG /
//     the COPY family and so on) through the active per-fork JumpTable in the
//     default case, exactly as the legacy loop did, so volatile gas and opcode
//     logic stays shared rather than restated.
//
// The generated file is committed and a CI test asserts it matches `go generate`
// output. Do not hand-edit interp_gen.go.
//
// Usage: go generate ./core/vm/...
package main

import (
	"bytes"
	"fmt"
	"go/ast"
	"go/format"
	"go/parser"
	"go/printer"
	"go/token"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"

	"github.com/ethereum/go-ethereum/core/vm"
)

const stackLimit = 1024 // params.StackLimit

// inlineHandler maps an opcode byte to the opXxx handler whose body is spliced
// inline for that opcode. These are the hot, fork-stable opcodes with no dynamic
// gas. Opcodes not listed here (or in directCold, or PUSH3-32 / DUP1-16, which
// are handled specially) fall through to the default case, which dispatches via
// the per-fork table.
var inlineHandler = map[byte]string{
	0x01: "opAdd", 0x02: "opMul", 0x03: "opSub", 0x04: "opDiv", 0x05: "opSdiv",
	0x06: "opMod", 0x07: "opSmod", 0x08: "opAddmod", 0x09: "opMulmod", 0x0b: "opSignExtend",
	0x10: "opLt", 0x11: "opGt", 0x12: "opSlt", 0x13: "opSgt", 0x14: "opEq", 0x15: "opIszero",
	0x16: "opAnd", 0x17: "opOr", 0x18: "opXor", 0x19: "opNot", 0x1a: "opByte",
	0x1b: "opSHL", 0x1c: "opSHR", 0x1d: "opSAR", 0x1e: "opCLZ",
	0x50: "opPop", 0x56: "opJump", 0x57: "opJumpi", 0x58: "opPc", 0x59: "opMsize", 0x5b: "opJumpdest",
	0x5f: "opPush0", 0x60: "opPush1", 0x61: "opPush2",
	0x90: "opSwap1", 0x91: "opSwap2", 0x92: "opSwap3", 0x93: "opSwap4",
	0x94: "opSwap5", 0x95: "opSwap6", 0x96: "opSwap7", 0x97: "opSwap8",
	0x98: "opSwap9", 0x99: "opSwap10", 0x9a: "opSwap11", 0x9b: "opSwap12",
	0x9c: "opSwap13", 0x9d: "opSwap14", 0x9e: "opSwap15", 0x9f: "opSwap16",
}

// directCold lists cold opcodes (dynamic gas, not inlined) whose handler,
// dynamic-gas, and memory-size functions are the same across every fork
// (verified: untouched by any enableXxx). They are emitted as direct calls to
// those functions by name instead of the indirect operation.* pointer calls
// in the default case. Go inlines the plain functions, and the var-aliased gas
// funcs (gasMLoad and friends alias pureMemoryGascost) at least skip the
// table load. Measured at ~3.4% on snailtracer (p=0.000). Fork-varying cold ops
// (CALL/SSTORE/SLOAD and friends) do not qualify and stay on the per-fork
// table in the default case.
//
//	opcode → {handler, dynamicGas, memorySize}
//
// Limited to the memory/hash ops that appear in hot loops. Adding more (e.g.
// CALLDATACOPY/RETURN, typically once per call) grows the generated function
// and regresses tiny benchmarks through code layout, for negligible gain.
var directCold = map[byte][3]string{
	0x20: {"opKeccak256", "gasKeccak256", "memoryKeccak256"}, // KECCAK256
	0x51: {"opMload", "gasMLoad", "memoryMLoad"},             // MLOAD
	0x52: {"opMstore", "gasMStore", "memoryMStore"},          // MSTORE
	0x53: {"opMstore8", "gasMStore8", "memoryMStore8"},       // MSTORE8
}

// opMeta is the per-opcode metadata derived from the per-fork tables.
type opMeta struct {
	defined  bool
	name     string // opcode mnemonic, e.g. "ADD"
	introF   string // params.Rules field activating it, empty for Frontier (always on)
	constGas uint64
	minStack int
	maxStack int
}

type generator struct {
	fset     *token.FileSet
	handlers map[string]*ast.FuncDecl
	meta     [256]opMeta
	buf      *bytes.Buffer
}

func (g *generator) p(format string, args ...any) { fmt.Fprintf(g.buf, format, args...) }

// ---------------------------------------------------------------------------
// Handler parsing + body splicing
// ---------------------------------------------------------------------------

// parseHandlers parses instructions.go and eips.go and returns every top-level
// opXxx function declaration by name.
func parseHandlers(vmDir string) (*token.FileSet, map[string]*ast.FuncDecl) {
	fset := token.NewFileSet()
	handlers := map[string]*ast.FuncDecl{}
	for _, name := range []string{"instructions.go", "eips.go"} {
		path := filepath.Join(vmDir, name)
		f, err := parser.ParseFile(fset, path, nil, parser.ParseComments)
		if err != nil {
			fatalf("parse %s: %v", path, err)
		}
		for _, decl := range f.Decls {
			fn, ok := decl.(*ast.FuncDecl)
			if !ok || fn.Recv != nil || fn.Body == nil {
				continue
			}
			handlers[fn.Name.Name] = fn
		}
	}
	return fset, handlers
}

var returnRe = regexp.MustCompile(`^(\s*)return\s+([^,]+),\s*(.+)$`)

// inlineBody returns the source of handler's body, rewritten so it can be
// spliced into the dispatch loop: the `*pc` dereference becomes the loop's `pc`
// local, and each `return r0, r1` becomes loop control flow. Success
// (r1 == nil) advances pc and continues, an error sets err and breaks.
func (g *generator) inlineBody(handler string) string {
	fn := g.handlers[handler]
	if fn == nil {
		fatalf("no handler %q to inline", handler)
	}
	var raw bytes.Buffer
	cfg := printer.Config{Mode: printer.UseSpaces | printer.TabIndent, Tabwidth: 8}
	for _, stmt := range fn.Body.List {
		if err := cfg.Fprint(&raw, g.fset, stmt); err != nil {
			fatalf("print %s body: %v", handler, err)
		}
		raw.WriteByte('\n')
	}
	src := strings.ReplaceAll(raw.String(), "*pc", "pc")

	var out strings.Builder
	for _, line := range strings.Split(src, "\n") {
		if m := returnRe.FindStringSubmatch(line); m != nil {
			indent, r0, r1 := m[1], strings.TrimSpace(m[2]), strings.TrimSpace(m[3])
			// The error and halt path must overwrite res and err. Otherwise a
			// halting inlined op (JUMPI on an invalid jump, say) returns stale
			// res from an earlier res-setting op such as a DELEGATECALL. The
			// fuzzer caught exactly that bug. The success path advances pc and
			// continues without writing res or err: stale res on a continuing
			// op is always overwritten by whichever op ends the run, and err
			// stays nil through normal iteration. Keeping these stores off the
			// success path avoids growing every inlined case, which regresses
			// tiny, layout-sensitive benchmarks.
			if r1 == "nil" {
				out.WriteString(indent + "pc++\n")
				out.WriteString(indent + "continue mainLoop\n")
			} else {
				out.WriteString(indent + "res, err = " + r0 + ", " + r1 + "\n")
				out.WriteString(indent + "break mainLoop\n")
			}
			continue
		}
		out.WriteString(line + "\n")
	}
	return out.String()
}

// ---------------------------------------------------------------------------
// Metadata derivation (from the per-fork tables, via vm.GenForks)
// ---------------------------------------------------------------------------

func (g *generator) deriveMeta(forks []vm.GenFork) {
	for code := 0; code < 256; code++ {
		for _, fork := range forks {
			o := fork.Ops[code]
			if !o.Defined {
				continue
			}
			g.meta[code] = opMeta{
				defined:  true,
				name:     o.Name,
				introF:   fork.RuleField,
				constGas: o.ConstantGas,
				minStack: o.MinStack,
				maxStack: o.MaxStack,
			}
			break // first fork that defines it wins (its intro fork)
		}
	}
	// Sanity: every inlined opcode must be defined and have fork-stable static
	// gas / stack bounds across all forks where it appears (that is what makes
	// it safe to bake as a constant). Bail loudly otherwise.
	for code, handler := range inlineHandler {
		g.checkStable(byte(code), handler, forks)
	}
	for code := 0x62; code <= 0x7f; code++ { // PUSH3-32
		g.checkStable(byte(code), "makePush", forks)
	}
	for code := 0x80; code <= 0x8f; code++ { // DUP1-16
		g.checkStable(byte(code), "makeDup", forks)
	}
}

func (g *generator) checkStable(code byte, what string, forks []vm.GenFork) {
	m := g.meta[code]
	if !m.defined {
		fatalf("opcode %#x (%s) selected for inlining but never defined", code, what)
	}
	for _, fork := range forks {
		o := fork.Ops[code]
		if !o.Defined {
			continue
		}
		if o.ConstantGas != m.constGas || o.MinStack != m.minStack || o.MaxStack != m.maxStack || o.HasDynamicGas {
			fatalf("opcode %#x (%s) is not fork-stable (fork %s): cannot inline", code, what, fork.Name)
		}
	}
}

// ---------------------------------------------------------------------------
// Case emission
// ---------------------------------------------------------------------------

// emitStackChecks emits the underflow/overflow guards for a baked opcode,
// mirroring the legacy loop's order (stack validated before gas).
func (g *generator) emitStackChecks(m opMeta) {
	under := m.minStack > 0
	over := m.maxStack < stackLimit
	switch {
	case under && over:
		g.p("if sLen := stack.len(); sLen < %d {\nreturn nil, &ErrStackUnderflow{stackLen: sLen, required: %d}\n} else if sLen > %d {\nreturn nil, &ErrStackOverflow{stackLen: sLen, limit: %d}\n}\n", m.minStack, m.minStack, m.maxStack, m.maxStack)
	case under:
		g.p("if sLen := stack.len(); sLen < %d {\nreturn nil, &ErrStackUnderflow{stackLen: sLen, required: %d}\n}\n", m.minStack, m.minStack)
	case over:
		g.p("if sLen := stack.len(); sLen > %d {\nreturn nil, &ErrStackOverflow{stackLen: sLen, limit: %d}\n}\n", m.maxStack, m.maxStack)
	}
}

func (g *generator) emitGasCheck(m opMeta) {
	if m.constGas == 0 {
		return
	}
	g.p("if contract.Gas.RegularGas < %d {\nreturn nil, ErrOutOfGas\n}\ncontract.Gas.RegularGas -= %d\n", m.constGas, m.constGas)
}

// emitWork emits the stack/gas guards and the opcode body (the portion that runs
// when the opcode is active for the current fork).
func (g *generator) emitWork(code byte) {
	m := g.meta[code]
	g.emitStackChecks(m)
	g.emitGasCheck(m)

	// PUSH1-PUSH32 swap their execute function under EIP-4762 (verkle) to charge
	// code-chunk gas on the immediate bytes. Defer to the table handler there.
	// The baked static gas and stack guard above already match.
	if code >= 0x60 && code <= 0x7f {
		g.p("if isEIP4762 {\nres, err = table[op].execute(&pc, evm, scope)\nif err != nil {\nbreak mainLoop\n}\npc++\ncontinue mainLoop\n}\n")
	}

	switch {
	case code >= 0x62 && code <= 0x7f: // PUSH3-PUSH32: inline makePush(n,n)
		g.emitPushFixed(int(code) - 0x5f)
	case code >= 0x80 && code <= 0x8f: // DUP1-DUP16: inline makeDup(n)
		g.p("scope.Stack.dup(%d)\npc++\ncontinue mainLoop\n", int(code)-0x7f)
	default:
		g.p("%s", g.inlineBody(inlineHandler[code]))
	}
}

// emitPushFixed inlines makePush(n, n) for PUSH<n> (n = 3..32).
func (g *generator) emitPushFixed(n int) {
	g.p("codeLen := len(scope.Contract.Code)\n")
	g.p("start := min(codeLen, int(pc+1))\n")
	g.p("end := min(codeLen, start+%d)\n", n)
	g.p("a := scope.Stack.get()\n")
	g.p("a.SetBytes(scope.Contract.Code[start:end])\n")
	g.p("if missing := %d - (end - start); missing > 0 {\na.Lsh(a, uint(8*missing))\n}\n", n)
	g.p("pc += %d\npc++\ncontinue mainLoop\n", n)
}

func (g *generator) emitInlineCase(code byte) {
	m := g.meta[code]
	g.p("case %s:\n", m.name)
	if m.introF == "" {
		g.emitWork(code)
		return
	}
	// Fork-gated: run the inlined body only when the opcode is active for the
	// current fork. Otherwise mirror the legacy loop's undefined-opcode handling.
	g.p("if rules.%s {\n", m.introF)
	g.emitWork(code)
	g.p("}\n")
	g.p("res, err = opUndefined(&pc, evm, scope)\nbreak mainLoop\n")
}

// emitDirectCold emits a cold opcode case identical to the default case, except
// the handler, dynamic-gas, and memory-size functions are called by name
// rather than through the indirect operation.* table pointers. Valid only for
// fork-invariant ops (see directCold).
func (g *generator) emitDirectCold(code byte) {
	m := g.meta[code]
	fns := directCold[code]
	g.p("case %s:\n", m.name)
	g.emitStackChecks(m)
	g.emitGasCheck(m)
	g.p("var memorySize uint64\n{\n")
	g.p("memSize, overflow := %s(stack)\n", fns[2])
	g.p("if overflow {\nreturn nil, ErrGasUintOverflow\n}\n")
	g.p("if memorySize, overflow = math.SafeMul(toWordSize(memSize), 32); overflow {\nreturn nil, ErrGasUintOverflow\n}\n}\n")
	g.p("var dynamicCost GasCosts\n")
	g.p("dynamicCost, err = %s(evm, contract, stack, mem, memorySize)\n", fns[1])
	// WriteString: keep %w/%v literal (not generator format verbs).
	g.buf.WriteString("if err != nil {\nreturn nil, fmt.Errorf(\"%w: %v\", ErrOutOfGas, err)\n}\n")
	g.p("if contract.Gas.RegularGas < dynamicCost.RegularGas {\nreturn nil, ErrOutOfGas\n}\n")
	g.p("contract.Gas.RegularGas -= dynamicCost.RegularGas\n")
	g.p("if memorySize > 0 {\nmem.Resize(memorySize)\n}\n")
	g.p("res, err = %s(&pc, evm, scope)\n", fns[0])
	g.p("if err != nil {\nbreak mainLoop\n}\npc++\ncontinue mainLoop\n")
}

func (g *generator) emitDefault() {
	// WriteString, not p(): this template contains %w/%v that must reach the
	// output verbatim (they are not generator format verbs).
	g.buf.WriteString(`default:
operation := table[op]
if sLen := stack.len(); sLen < operation.minStack {
return nil, &ErrStackUnderflow{stackLen: sLen, required: operation.minStack}
} else if sLen > operation.maxStack {
return nil, &ErrStackOverflow{stackLen: sLen, limit: operation.maxStack}
}
cost := operation.constantGas
if contract.Gas.RegularGas < cost {
return nil, ErrOutOfGas
}
contract.Gas.RegularGas -= cost
var memorySize uint64
if operation.dynamicGas != nil {
if operation.memorySize != nil {
memSize, overflow := operation.memorySize(stack)
if overflow {
return nil, ErrGasUintOverflow
}
if memorySize, overflow = math.SafeMul(toWordSize(memSize), 32); overflow {
return nil, ErrGasUintOverflow
}
}
var dynamicCost GasCosts
dynamicCost, err = operation.dynamicGas(evm, contract, stack, mem, memorySize)
if err != nil {
return nil, fmt.Errorf("%w: %v", ErrOutOfGas, err)
}
if contract.Gas.RegularGas < dynamicCost.RegularGas {
return nil, ErrOutOfGas
}
contract.Gas.RegularGas -= dynamicCost.RegularGas
}
if memorySize > 0 {
mem.Resize(memorySize)
}
res, err = operation.execute(&pc, evm, scope)
if err != nil {
break mainLoop
}
pc++
continue mainLoop
`)
}

// ---------------------------------------------------------------------------
// File emission
// ---------------------------------------------------------------------------

func (g *generator) emitFile() {
	g.p("// Code generated by core/vm/gen; DO NOT EDIT.\n\n")
	g.p("package vm\n\n")
	g.p("import (\n")
	g.p("\t\"fmt\"\n\n")
	g.p("\t\"github.com/ethereum/go-ethereum/common/math\"\n")
	g.p("\t\"github.com/ethereum/go-ethereum/core/tracing\"\n")
	g.p(")\n\n")

	g.buf.WriteString(`// execUntraced is the generated, tracing-free interpreter fast path. Hot,
// fork-stable opcodes are inlined with their static gas and stack bounds baked
// in. Fork-invariant cold ops (KECCAK256/MLOAD/MSTORE/MSTORE8) call their
// handler and gas functions directly by name. Everything fork-varying is
// dispatched through the active per-fork table in the default case. EVM.Run
// selects this path when no tracer is configured.
func (evm *EVM) execUntraced(scope *ScopeContext) (ret []byte, err error) {
var (
contract  = scope.Contract
mem       = scope.Memory
stack     = scope.Stack
table     = evm.table
rules     = evm.chainRules
isEIP4762 = rules.IsEIP4762
pc        = uint64(0)
res       []byte
)
_ = mem
_ = rules
_ = isEIP4762
_ = table
mainLoop:
for {
if isEIP4762 && !contract.IsDeployment && !contract.IsSystemCall {
contractAddr := contract.Address()
consumed, wanted := evm.TxContext.AccessEvents.CodeChunksRangeGas(contractAddr, pc, 1, uint64(len(contract.Code)), false, contract.Gas.RegularGas)
contract.UseGas(GasCosts{RegularGas: consumed}, evm.Config.Tracer, tracing.GasChangeWitnessCodeChunk)
if consumed < wanted {
return nil, ErrOutOfGas
}
}
op := contract.GetOp(pc)
switch op {
`)
	// Inlined hot cases, in opcode order for readability.
	for code := 0; code < 256; code++ {
		b := byte(code)
		_, named := inlineHandler[b]
		isPushFixed := code >= 0x62 && code <= 0x7f
		isDup := code >= 0x80 && code <= 0x8f
		if named || isPushFixed || isDup {
			g.emitInlineCase(b)
		} else if _, dc := directCold[b]; dc {
			g.emitDirectCold(b)
		}
	}
	g.emitDefault()
	g.p("}\n") // switch
	g.p("}\n") // for
	g.p("if err == errStopToken {\nerr = nil\n}\n")
	g.p("return res, err\n")
	g.p("}\n") // func
}

// ---------------------------------------------------------------------------

func fatalf(format string, args ...any) {
	fmt.Fprintf(os.Stderr, "gen: "+format+"\n", args...)
	os.Exit(1)
}

func main() {
	_, self, _, ok := runtime.Caller(0)
	if !ok {
		fatalf("cannot resolve generator source path")
	}
	vmDir := filepath.Dir(filepath.Dir(self)) // .../core/vm/gen -> .../core/vm

	fset, handlers := parseHandlers(vmDir)
	g := &generator{fset: fset, handlers: handlers, buf: new(bytes.Buffer)}
	g.deriveMeta(vm.GenForks())
	g.emitFile()

	formatted, err := format.Source(g.buf.Bytes())
	if err != nil {
		dbg := filepath.Join(vmDir, "interp_gen.go.broken")
		os.WriteFile(dbg, g.buf.Bytes(), 0644)
		fatalf("gofmt failed (%v); wrote unformatted output to %s", err, dbg)
	}
	// INTERP_GEN_OUT lets the CI-match test (interp_gen_test.go) regenerate to a
	// temporary file and diff it against the committed one, without clobbering it.
	out := filepath.Join(vmDir, "interp_gen.go")
	if env := os.Getenv("INTERP_GEN_OUT"); env != "" {
		out = env
	}
	if err := os.WriteFile(out, formatted, 0644); err != nil {
		fatalf("write %s: %v", out, err)
	}
	fmt.Fprintf(os.Stderr, "gen: wrote %s (%d bytes)\n", out, len(formatted))
}
