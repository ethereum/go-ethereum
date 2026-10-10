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
// fork-invariance checks. An opcode whose functions are the same in every fork can
// then be emitted as a call by name. The fork-varying ones cannot, and are still
// reached through the active per-fork JumpTable at runtime (see interpreter_gen.go):
// several are closures (gasCall, the memoryCopierGas family, makeGasLog) with no
// callable name.

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

// genFnName returns the FuncForPC name of a jump-table function value with the
// package path stripped (e.g. "gasKeccak256"), or "" if nil. An aliased var
// resolves to the underlying function (gasMLoad reports "pureMemoryGascost").
// A closure keeps its enclosing chain (LOG1's handler reports
// "newFrontierInstructionSet.makeLog.func19"), so the generator can tell a
// factory-built handler from a plain one and unrelated closures cannot collide
// on a bare "funcN".
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

// GenForks returns per-fork opcode metadata for the interpreter code generator,
// one entry per fork that changes any opcode metadata, including the handler,
// dynamic-gas and memory-size function names, oldest to newest.
func GenForks() []GenFork {
	// Frontier is always active and carries no rule gate.
	frontier, _ := LookupInstructionSet(params.Rules{})
	out := []GenFork{genFork("Frontier", "", &frontier)}

	var (
		rules params.Rules
		rv    = reflect.ValueOf(&rules).Elem()
	)
	for i := range rv.NumField() {
		field := rv.Type().Field(i)
		if field.Type.Kind() != reflect.Bool {
			continue
		}
		// Activate this field on top of all earlier ones, as a chain does.
		rv.Field(i).SetBool(true)
		set, _ := LookupInstructionSet(rules)
		fork := genFork(strings.TrimPrefix(field.Name, "Is"), field.Name, &set)

		// Skip forks that change nothing over the previous one.
		if fork.Ops == out[len(out)-1].Ops {
			continue
		}
		out = append(out, fork)
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
