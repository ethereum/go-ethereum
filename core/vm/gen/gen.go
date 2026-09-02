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
//
// The generator emits calls, not bodies. An opcode's case charges its gas and
// checks its stack from constants derived from the per-fork tables, then calls
// the handler by name. Getting the body into the loop is the compiler's job.
//
// That is the whole design, and it is what keeps this file printf. Splicing the
// bodies instead would mean parsing core/vm, rewriting every return into the
// loop's control flow, and rewriting every stack call into loop locals, which is
// roughly 740 lines of AST machinery. The trade is that inlining stops being
// guaranteed: Go's budget is 80 and these bodies cost far more, so they only make
// it into the loop when a PGO profile marks them hot and raises the budget.

// Names the generator needs in more than one place: the file it writes, the
// dispatch inside it, and the profile that gets that dispatch's calls inlined.
const (
	generatedFile = "interpreter_gen.go"
	handlerFile   = "instructions.go"
	dispatchFunc  = "execUntraced"
	dispatchName  = "(*EVM)." + dispatchFunc
	vmPkgPath     = "github.com/ethereum/go-ethereum/core/vm"
	pgoFile       = "cmd/geth/default.pgo"
)

type generator struct {
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
// stack before gas. minExpr and maxExpr are constants in an opcode's own case,
// operation.minStack and operation.maxStack in the table path. under and over let a
// path omit a guard whose bound is trivial.
//
// The bounds are compared against sp, the loop's own depth counter, rather than
// stack.len(). stack.len() reads size back through a pointer, and the compiler has
// to reload it after every handler call because a handler can change it, so it was
// a memory access per opcode.
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

// emitStaticGas charges amount through ChargeExecutionOnly. amount is a constant in
// an opcode's own case, operation.constantGas in the table path. The method is small
// enough that the compiler inlines it without help.
func (g *generator) emitStaticGas(amount any) {
	g.p(`
		if gerr := contract.Gas.ChargeExecutionOnly(%v); gerr != nil {
			res, err = nil, gerr
			break mainLoop
		}
	`, amount)
}

// emitDynamicGas emits the memory sizing and dynamic gas charge through the shared
// meterDynamicGas, then grows memory to what it charged for. It needs the opcode's
// operation, so the case pays one table load even though it calls its handler by
// name.
func (g *generator) emitDynamicGas() {
	g.p(`
		operation := table[op]
		var memorySize uint64
		if memorySize, _, err = contract.meterDynamicGas(operation, evm, stack, mem); err != nil {
			return nil, err
		}
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
// handler can fail, so bail on error, then step sp and the pc.
//
// step is what the handler did to the stack depth. An opcode's own case knows it as
// a constant, so it adjusts sp. The table path does not, so it passes an empty step
// and re-reads the depth instead.
func (g *generator) emitCallHandler(call, step string) {
	g.p(`
		res, err = %s
		if err != nil {
			break mainLoop
		}
	`, call)
	if step != "" {
		g.p("%s\n", step)
	}
	g.p(`
		pc++
		continue mainLoop
	`)
}

// stackStep returns the statement that moves sp by an opcode's stack delta, or ""
// when the opcode leaves the depth alone.
func stackStep(delta int) string {
	switch {
	case delta > 0:
		return fmt.Sprintf("sp += %d", delta)
	case delta < 0:
		return fmt.Sprintf("sp -= %d", -delta)
	}
	return ""
}

// emitStaticOp emits a case for an opcode whose whole cost is its constant gas: the
// stack and gas guards from constants, then a call to its handler by name. A
// fork-introduced opcode wraps that in a fork gate so it runs only when the opcode is
// active, otherwise the case mirrors the legacy loop's undefined-opcode handling.
func (g *generator) emitStaticOp(code byte) {
	spec := g.specs[code]
	g.p("case %s:\n", spec.name)
	if spec.fork != "" {
		g.p("if rules.%s {\n", spec.fork)
	}

	g.emitStackChecks(spec.stackGuards())
	if spec.constGas != 0 {
		g.emitStaticGas(spec.constGas)
	}
	g.emitCallHandler(g.handlerCall(code), stackStep(spec.stackDelta()))

	// Close the fork gate opened above, then the branch taken while the fork is
	// still inactive.
	if spec.fork != "" {
		g.p("}\n")
		g.emitUndefinedFallback()
	}
}

// emitDynamicOp emits a case for a fork-invariant opcode that carries dynamic gas.
// It differs from the table path only in calling the handler by name rather than
// through a table pointer, and in emitting its static gas and stack bounds as
// constants.
func (g *generator) emitDynamicOp(code byte) {
	spec := g.specs[code]
	g.p("case %s:\n", spec.name)

	g.emitStackChecks(spec.stackGuards())
	if spec.constGas != 0 {
		g.emitStaticGas(spec.constGas)
	}
	g.emitDynamicGas()
	g.emitCallHandler(g.handlerCall(code), stackStep(spec.stackDelta()))
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
	g.emitStackChecks("operation.minStack", "operation.maxStack", true, true)
	g.emitStaticGas("operation.constantGas")
	g.p(`
			var memorySize uint64
			if memorySize, _, err = contract.meterDynamicGas(operation, evm, stack, mem); err != nil {
				return nil, err
			}
			if memorySize > 0 {
				mem.Resize(memorySize)
			}
	`)
	g.emitCallHandler("operation.execute(&pc, evm, scope)", "sp = stack.len()")
}

// handlerCall returns the call expression for an opcode's handler. The name comes
// from the per-fork tables via FuncForPC, which leaves a closure's enclosing chain in
// the name, so a dotted one is a factory-built handler with no name to call.
func (g *generator) handlerCall(code byte) string {
	spec := g.specs[code]
	if strings.Contains(spec.execFn, ".") {
		abortf("opcode %#x (%s) is built by the closure %q, which has no name to call. Give it a named handler or leave it on the table tier",
			code, spec.name, spec.execFn)
	}
	return spec.execFn + "(&pc, evm, scope)"
}

// createFile writes the whole generated file into g.buf, in order: header, imports,
// execUntraced's loop locals, the dispatch switch, then the loop's exit. generate
// formats the buffer and main writes it out. The switch gets a case per opcode
// opTiers assigns a tier, plus emitTableOp's default.
func (g *generator) createFile() {
	// file header, package clause, and imports
	g.p(`
		// Code generated by core/vm/gen; DO NOT EDIT.

		package vm
	`)

	// execUntraced: doc comment, loop-local declarations, and the dispatch loop
	g.p(`
		// execUntraced is the generated, tracing-free interpreter fast path. It is a
		// switch over the opcode byte, which Go lowers to a jump table, replacing the
		// legacy loop's indirect call through the per-fork JumpTable.
		//
		// Hot, fork-stable opcodes get their own case, with static gas and stack bounds
		// emitted as constants and the handler called by name. Everything fork-varying
		// is dispatched through the active per-fork table in the default case, so its
		// volatile gas logic stays shared. EVM.Run selects this path when no tracer is
		// configured.
		//
		// The handlers are called, not inlined here. Go's normal inline budget is far
		// below the cost of these bodies, so getting them into the loop depends on a
		// PGO profile marking them hot, which raises the budget to 2000. Build without
		// a profile and this is a switch of calls, which is still cheaper than the
		// legacy indirect call but gives up the inlining.
		func (evm *EVM) execUntraced(scope *ScopeContext) (ret []byte, err error) {
			var (
				contract = scope.Contract
				mem      = scope.Memory
				stack    = scope.Stack
				table    = evm.table
				rules    = evm.chainRules
				pc       = uint64(0)
				res      []byte
			)
			// Which of these the switch uses depends on the tier assignments, so
			// keep them all live rather than tracking usage while emitting.
			_, _, _ = mem, rules, table
			// sp shadows the stack depth for the bounds checks. Each case steps it
			// by its opcode's known delta, and the table case re-reads it because
			// its delta is not known until run time.
			sp := stack.len()
		mainLoop:
			for {
	`)

	// fetch the opcode and open the dispatch switch
	g.p(`
				op := contract.GetOp(pc)
				switch op {
	`)

	// one case per opcode with its own tier, in opcode order
	for code := range 256 {
		switch b := byte(code); tierOf(b) {
		case tierStatic:
			g.emitStaticOp(b)
		case tierDynamic:
			g.emitDynamicOp(b)
		}
	}

	// the default case: fork-varying ops via the per-fork table
	g.emitTableOp()

	// close the switch and loop, clear the stop token, and return
	g.p(`
				}
			}
			if err == errStopToken {
				err = nil
			}
			return res, err
		}
	`)
}

// genError is what abortf panics with. The generator's guards unwind to generate
// rather than each call site threading an error back. Panicking rather than exiting
// is what makes them testable.
type genError string

func (e genError) Error() string { return string(e) }

// abortf stops generation. Every check the generator makes about the code it emits
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

// generate derives the per-opcode spec from the per-fork tables and returns the
// formatted contents of interpreter_gen.go. It is the shared core of the generator:
// main writes the result to disk, and the up-to-date test in gen_test.go compares it
// against the committed file.
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

	g := &generator{buf: new(bytes.Buffer)}
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
