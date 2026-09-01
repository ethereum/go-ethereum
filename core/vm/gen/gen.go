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

package main

import (
	"bytes"
	"fmt"
	"go/format"
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

// This file holds the shape of the generated dispatch: the emitters for one
// opcode case of each tier, and the assembly of the whole file.

type generator struct {
	*source
	specs [256]opSpec
	buf   *bytes.Buffer
}

// p is the writer of the generated file. Every line of output is appended
// to g.buf through it.
func (g *generator) p(format string, args ...any) {
	format = strings.TrimRight(strings.TrimPrefix(format, "\n"), " \t")
	fmt.Fprintf(g.buf, format, args...)
}

// emitStackChecks emits the underflow/overflow guards, in the legacy loop's order of
// stack before gas. minExpr and maxExpr are constants on the inlined and direct
// paths, operation.minStack and operation.maxStack in the table path. under and over
// let a path omit a guard whose bound is trivial.
func (g *generator) emitStackChecks(minExpr, maxExpr any, under, over bool) {
	switch {
	case under && over: // the table path, which knows neither bound statically
		g.p(`
			if sp < %v {
				res, err = nil, &ErrStackUnderflow{stackLen: sp, required: %v}
				break mainLoop
			} else if sp > %v {
				res, err = nil, &ErrStackOverflow{stackLen: sp, limit: %v}
				break mainLoop
			}
		`, minExpr, minExpr, maxExpr, maxExpr)
	case under:
		g.p(`
			if sp < %v {
				res, err = nil, &ErrStackUnderflow{stackLen: sp, required: %v}
				break mainLoop
			}
		`, minExpr, minExpr)
	case over:
		g.p(`
			if sp > %v {
				res, err = nil, &ErrStackOverflow{stackLen: sp, limit: %v}
				break mainLoop
			}
		`, maxExpr, maxExpr)
	}
}

// emitStaticGas charges amount by splicing ChargeExecutionOnly, so the loop makes no
// call. amount is a constant on the inlined and direct paths, operation.constantGas in
// the table path.
//
// ChargeExecutionOnly reads:
//
//	if g.ExecutionGas < r {
//		return ErrOutOfGas
//	}
//	g.ExecutionGas -= r
//	g.UsedExecutionGas += r
//	return nil
//
// and for a 3 gas opcode this emits:
//
//	if contract.Gas.ExecutionGas < 3 {
//		res, err = nil, ErrOutOfGas
//		break mainLoop
//	}
//	contract.Gas.ExecutionGas -= 3
//	contract.Gas.UsedExecutionGas += 3
func (g *generator) emitStaticGas(amount any) {
	// ChargeExecutionOnly charges its receiver, the budget, the amount in its one
	// parameter. In the dispatch that budget is the loop's contract and the amount
	// is this opcode's constant.
	src := g.spliceHelper("ChargeExecutionOnly", helperBinds{
		recv:   "contract.Gas",
		params: []string{fmt.Sprint(amount)},
	})

	// return ErrOutOfGas -> res, err = nil, ErrOutOfGas plus break, and return nil -> nothing.
	g.p("%s", g.rewriteGasReturns(src))
}

// emitSyncStackView publishes sp into the stack view, ahead of anything that reads
// the stack through it: a memory-size function, a dynamic-gas function, or an opcode
// handler. emitReloadStackView is its other half.
func (g *generator) emitSyncStackView() {
	g.p(`
		stack.size = sp
		stack.inner.top = stack.bottom + sp
	`)
}

// emitReloadStackView reloads sp and sd after a call that may have pushed,
// popped, or grown the arena underneath them.
func (g *generator) emitReloadStackView() {
	g.p(`
		sp = stack.size
		sd = stack.inner.data[stack.bottom:]
	`)
}

// emitResizeMemory grows memory to the size the gas step charged for.
func (g *generator) emitResizeMemory() {
	g.p(`
		if memorySize > 0 {
			mem.Resize(memorySize)
		}
	`)
}

// emitUndefinedFallback emits what a fork-gated opcode does before its fork activates.
// The opcode does not exist yet, so the case matches the legacy loop's handling of an
// undefined one.
func (g *generator) emitUndefinedFallback() {
	g.p(`
		res, err = opUndefined(&pc, evm, scope)
		break mainLoop
	`)
}

// emitCallHandler calls an opcode handler and does the bookkeeping around it. The
// handler can move the stack and can fail, so reload, bail on error, then step the pc.
func (g *generator) emitCallHandler(call string) {
	g.p("res, err = %s\n", call)
	g.emitReloadStackView()
	g.p(`
		if err != nil {
			break mainLoop
		}
		pc++
		continue mainLoop
	`)
}

// emitInlineOp emits an inlined opcode case: the stack and gas guards followed by
// the spliced opcode body. A fork-introduced opcode wraps that body in a fork gate
// so it runs only when the opcode is active for the current fork, otherwise the
// case mirrors the legacy loop's undefined-opcode handling.
func (g *generator) emitInlineOp(code byte) {
	spec := g.specs[code]
	g.p("case %s:\n", spec.name)
	if spec.fork != "" {
		g.p("if rules.%s {\n", spec.fork)
	}

	// stack bounds check
	g.emitStackChecks(spec.stackGuards())

	// static gas
	if spec.constGas != 0 {
		g.emitStaticGas(spec.constGas)
	}

	// PUSH1-PUSH32 swap their execute function under EIP-4762 (verkle) to charge
	// code-chunk gas on the immediate bytes. Defer to the table handler there.
	// The constant static gas and stack guard above already match.
	if code >= 0x60 && code <= 0x7f {
		g.p("if isEIP4762 {\n")
		g.emitSyncStackView()
		g.emitCallHandler("table[op].execute(&pc, evm, scope)")
		g.p("}\n")
	}

	// opcode body. genFnName leaves a closure's enclosing chain in the name, so a
	// dotted one is a factory-built handler with no body to look up.
	if strings.Contains(spec.execFn, ".") {
		abortf("opcode %#x (%s) is built by the closure %q, which the generator cannot inline", code, spec.name, spec.execFn)
	}
	g.p("%s", g.spliceOpcodeBody(spec.execFn))

	// Close the fork gate opened above, then the branch taken while the fork is
	// still inactive.
	if spec.fork != "" {
		g.p("}\n")
		g.emitUndefinedFallback()
	}
}

// emitDirectOp emits the default case's steps with two shortcuts: the handler, gas
// and memory functions are called by name instead of through table pointers Go cannot
// inline, and the gas step splices computeMemorySize and chargeDynamicGas directly,
// skipping meterDynamicGas's nil checks. Both need a fork-invariant op.
func (g *generator) emitDirectOp(code byte) {
	spec := g.specs[code]
	g.p("case %s:\n", spec.name)

	// stack bounds check
	g.emitStackChecks(spec.stackGuards())

	// static gas
	if spec.constGas != 0 {
		g.emitStaticGas(spec.constGas)
	}

	// the memory-size and dynamic-gas functions read the stack view
	g.emitSyncStackView()

	// The two spliced bodies below both assign it, so declare it once ahead of them.
	g.p("\nvar memorySize uint64\n")

	// computeMemorySize, bound to this opcode's memory-size function, with its result
	// landing in memorySize. For KECCAK256 that turns
	//
	//	memSize, overflow := memFn(stack)
	//	...
	//	return size, nil
	//
	// into
	//
	//	memSize, overflow := memoryKeccak256(stack)
	//	...
	//	memorySize = size
	memSize := g.spliceHelper("computeMemorySize", helperBinds{params: []string{spec.memFn}})
	g.p("%s", g.rewriteStepReturns(memSize, "memorySize"))

	// chargeDynamicGas the same way, except its value is the cost the traced loop
	// reports, which this path has no use for, so the empty target drops it:
	//
	//	dynamicCost, gerr := dynFn(evm, contract, stack, mem, memorySize)
	//
	// becomes
	//
	//	dynamicCost, gerr := gasKeccak256(evm, contract, stack, mem, memorySize)
	dynGas := g.spliceHelper("chargeDynamicGas", helperBinds{params: []string{spec.dynFn}})
	g.p("%s", g.rewriteStepReturns(dynGas, ""))

	// resize memory
	g.emitResizeMemory()

	// call the opcode handler by name, no table pointer
	g.emitCallHandler(spec.execFn + "(&pc, evm, scope)")
}

// emitTableOp emits the switch's default case, which walks the table exactly as the
// legacy loop did. Every fork-varying op lands here, along with the undefined ones,
// so their volatile logic stays shared rather than restated.
func (g *generator) emitTableOp() {
	// One table load, reused by every step below.
	g.p(`
		default:
			operation := table[op]
	`)
	// stack bounds check
	g.emitStackChecks("operation.minStack", "operation.maxStack", true, true)

	// static gas
	g.emitStaticGas("operation.constantGas")

	// dynamic gas, computed on the stack view
	g.emitSyncStackView()
	g.p(`
			var memorySize uint64
			if memorySize, _, err = contract.meterDynamicGas(operation, evm, stack, mem); err != nil {
				return nil, err
			}
	`)

	// resize memory
	g.emitResizeMemory()

	// call the opcode handler through the table, as the legacy loop did
	g.emitCallHandler("operation.execute(&pc, evm, scope)")
}

// createFile writes the whole generated file into g.buf, in order: header, imports,
// execUntraced's loop locals, the verkle code-chunk charge, the dispatch switch, then
// the loop's exit. generate formats the buffer and main writes it out. The switch gets
// a case per opcode opTiers assigns a tier, plus emitTableOp's default.
func (g *generator) createFile() {
	// file header, package clause, and imports
	g.p(`
		// Code generated by core/vm/gen; DO NOT EDIT.

		package vm

		import (
			"fmt"

			"github.com/ethereum/go-ethereum/common/math"
			"github.com/ethereum/go-ethereum/core/tracing"
		)

	`)

	// execUntraced: doc comment, loop-local declarations, and the dispatch loop
	g.p(`
		// execUntraced is the generated, tracing-free interpreter fast path. Hot,
		// fork-stable opcodes are inlined with their static gas and stack bounds emitted
		// as constants. Fork-invariant ops (KECCAK256/MLOAD/MSTORE/MSTORE8) call their
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
			// Which of these the switch uses depends on the tier assignments, so
			// keep them all live rather than tracking usage while emitting.
			_, _, _, _ = mem, rules, isEIP4762, table
			// sp and sd shadow stack.size and the stack's window of the arena
			// as loop locals, so hot opcodes work on registers instead of the
			// view. They are written back before any call that can see the
			// stack and reloaded after any call that can move it.
			sp := stack.size
			sd := stack.inner.data[stack.bottom:]
		mainLoop:
			for {
	`)

	// verkle code-chunk gas, spliced from chargeVerkleCodeChunkGas
	g.p("%s", g.rewriteGasReturns(g.renderAst(g.gasHelper("chargeVerkleCodeChunkGas").Body.List)))

	// fetch the opcode and open the dispatch switch
	g.p(`
				op := contract.GetOp(pc)
				switch op {
	`)

	// one case per inlined or direct-call opcode, in opcode order
	for code := range 256 {
		switch b := byte(code); tierOf(b) {
		case tierInline:
			g.emitInlineOp(b)
		case tierDirect:
			g.emitDirectOp(b)
		}
	}

	// the default case: fork-varying ops via the per-fork table
	g.emitTableOp()

	// close the switch and loop, clear the stop token, and return
	g.p(`
				}
			}
			stack.size = sp
			stack.inner.top = stack.bottom + sp
			if err == errStopToken {
				err = nil
			}
			return res, err
		}
	`)
}

// genError is what abortf panics with. The generator's guards sit deep inside the
// splicer, so they unwind to generate rather than each call site threading an error
// back. Panicking rather than exiting is what makes them testable.
type genError string

func (e genError) Error() string { return string(e) }

// abortf stops generation. Every check the generator makes about the code it splices
// ends here, so a change it cannot express fails the build instead of producing wrong
// dispatch. It panics rather than exits so generate can report it and tests can
// assert on it.
func abortf(format string, args ...any) {
	panic(genError(fmt.Sprintf(format, args...)))
}

// fatalf reports a failure and exits. It is for the callers outside generate's
// recover, namely main and vmDir, where a panic would surface as a stack trace
// instead of a message.
func fatalf(format string, args ...any) {
	fmt.Fprintf(os.Stderr, "gen: "+format+"\n", args...)
	os.Exit(1)
}

// vmDir returns the core/vm directory, the parent of this generator package. It
// is resolved from this source file's own path so it does not depend on the
// directory the generator or the test happens to run from.
func vmDir() string {
	_, self, _, ok := runtime.Caller(0)
	if !ok {
		fatalf("cannot resolve generator source path")
	}
	return filepath.Dir(filepath.Dir(self)) // .../core/vm/gen -> .../core/vm
}

// generate parses the opcode, gas and fork definitions under core/vm and returns
// the formatted contents of interpreter_gen.go. It is the shared core of the
// generator: main writes the result to disk, and the up-to-date test in
// gen_test.go compares it against the committed file.
func generate() (out []byte, err error) {
	// Turn a tripped guard into an error. Anything else is a bug in the generator
	// itself, so let it crash with its stack.
	defer func() {
		switch r := recover().(type) {
		case nil:
		case genError:
			err = r
		default:
			panic(r)
		}
	}()

	g := &generator{source: parseSource(vmDir()), buf: new(bytes.Buffer)}
	g.deriveSpecs(genForks())
	g.createFile()

	formatted, err := format.Source(g.buf.Bytes())
	if err != nil {
		dbg := filepath.Join(vmDir(), "interpreter_gen.go.broken")
		os.WriteFile(dbg, g.buf.Bytes(), 0644)
		return nil, fmt.Errorf("gofmt failed (%v); wrote unformatted output to %s", err, dbg)
	}
	return formatted, nil
}
