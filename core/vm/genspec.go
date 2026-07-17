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

//go:generate go run ./gen

import (
	"reflect"
	"runtime"
	"strings"

	"github.com/ethereum/go-ethereum/params"
)

// This file exposes the interpreter's opcode metadata to the code generator in
// core/vm/gen. It is not used at runtime. It exists so the generator can derive
// the per-opcode spec (static gas, stack bounds, the fork an opcode first
// appears in, and the FuncForPC names of its handler/gas/memory functions) from
// the existing per-fork instruction sets, rather than restating that metadata.
//
// The function names supply the generator's opcode-to-handler mapping and its
// fork-invariance checks. The fork-varying gas/execute functions themselves are
// still reached through the active per-fork JumpTable at runtime (see
// interpreter_gen.go), not emitted by name: several are closures (gasCall, the
// memoryCopierGas family, makeGasLog) that have no callable name.

// GenOp is the generator-facing scalar metadata for one opcode slot in one fork.
type GenOp struct {
	Name         string // opcode mnemonic, e.g. "ADD" (valid only if Defined)
	Defined      bool   // false if the slot is undefined/invalid in this fork
	ConstantGas  uint64
	MinStack     int
	MaxStack     int
	ExecuteFn    string // FuncForPC name of op.execute
	DynamicGasFn string // FuncForPC name of op.dynamicGas, "" if nil
	MemorySizeFn string // FuncForPC name of op.memorySize, "" if nil
}

// GenFork bundles a fork's name, the params.Rules bool field that activates it
// (empty for Frontier, which is always active), and its per-opcode metadata.
type GenFork struct {
	Name      string
	RuleField string
	Ops       [256]GenOp
}

// codegenSkippedForks are forks in geth's fork schedule that the generator does
// not give a lane. Verkle/UBT is the only one: over its Shanghai base it adds no
// opcodes, it only swaps gas and execute functions on existing ones (enable4762),
// which the generated switch picks up from the active table at runtime. Emitting
// a lane for it would trip the generator's fork-stability check (an inlined op's
// execute function is not allowed to vary by fork).
var codegenSkippedForks = map[string]bool{"IsUBT": true}

// genFnName returns the FuncForPC name of a jump-table function value with the
// package path stripped (e.g. "gasKeccak256"), or "" if nil. An aliased var
// resolves to the underlying function (gasMLoad reports "pureMemoryGascost").
// A closure keeps its enclosing chain (DUP7's handler reports
// "newFrontierInstructionSet.makeDup.func37"), so the generator can tell which
// factory built it and unrelated closures cannot collide on a bare "funcN".
func genFnName(fn any) string {
	v := reflect.ValueOf(fn)
	if !v.IsValid() || v.IsNil() {
		return ""
	}
	full := runtime.FuncForPC(v.Pointer()).Name()
	if i := strings.LastIndex(full, "/"); i >= 0 {
		full = full[i+1:] // strip the package path, leaving "vm.<name>"
	}
	if i := strings.Index(full, "."); i >= 0 {
		full = full[i+1:] // strip the package name
	}
	return full
}

// GenForks returns per-fork opcode metadata for the interpreter code generator
// (core/vm/gen), one entry per fork that changes the opcode table, oldest to
// newest. It derives the progression from params.Rules and LookupInstructionSet
// (in params.Rules declaration order, which is chronological) so new forks are
// picked up without restating a list here.
func GenForks() []GenFork {
	// Frontier is always active and carries no rule gate.
	frontier, _ := LookupInstructionSet(params.Rules{})
	out := []GenFork{genFork("Frontier", "", &frontier)}

	rt := reflect.TypeOf(params.Rules{})
	for i := range rt.NumField() {
		field := rt.Field(i)
		if field.Type.Kind() != reflect.Bool || codegenSkippedForks[field.Name] {
			continue
		}
		// Activate only this field so the fork resolves to the table it gates.
		var rules params.Rules
		reflect.ValueOf(&rules).Elem().Field(i).SetBool(true)
		set, _ := LookupInstructionSet(rules)
		if sameOps(&set, &frontier) {
			continue // this rule does not change the opcode table
		}
		out = append(out, genFork(strings.TrimPrefix(field.Name, "Is"), field.Name, &set))
	}
	return out
}

// genFork extracts the generator-facing per-opcode metadata from one fork's
// instruction set.
func genFork(name, rule string, set *JumpTable) GenFork {
	gf := GenFork{Name: name, RuleField: rule}
	for code := range 256 {
		op := set[code]
		if op == nil || op.undefined {
			continue
		}
		gf.Ops[code] = GenOp{
			Name:         OpCode(code).String(),
			Defined:      true,
			ConstantGas:  op.constantGas,
			MinStack:     op.minStack,
			MaxStack:     op.maxStack,
			ExecuteFn:    genFnName(op.execute),
			DynamicGasFn: genFnName(op.dynamicGas),
			MemorySizeFn: genFnName(op.memorySize),
		}
	}
	return gf
}

// sameOps reports whether two instruction sets carry identical per-opcode static
// metadata: which slots are defined, and their static gas and stack bounds. It
// is used to tell whether a fork actually changes the opcode table. Handler,
// dynamic-gas and memory-size functions are ignored (they cannot be compared for
// equality and are reached through the table at runtime).
func sameOps(a, b *JumpTable) bool {
	for code := range 256 {
		oa, ob := a[code], b[code]
		undefA := oa == nil || oa.undefined
		undefB := ob == nil || ob.undefined
		if undefA != undefB {
			return false
		}
		if undefA {
			continue
		}
		if oa.constantGas != ob.constantGas || oa.minStack != ob.minStack || oa.maxStack != ob.maxStack {
			return false
		}
	}
	return true
}
