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
)

// This file exposes the interpreter's opcode metadata to the code generator in
// core/vm/gen. It is not used at runtime. It exists so the generator can derive
// the per-opcode spec (static gas, stack bounds, the fork an opcode first
// appears in, and the FuncForPC names of its handler/gas/memory functions) from
// the existing per-fork instruction sets, rather than restating that metadata.
//
// The function names let the generator confirm the directCold ops are
// fork-invariant. The fork-varying gas/execute functions themselves are still
// reached through the active per-fork JumpTable at runtime (see interpreter_gen.go),
// not emitted by name: several are closures (gasCall, the memoryCopierGas
// family, makeGasLog) that FuncForPC reports only as anonymous labels, so they
// could not be called by name in any case.

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

// genForkOrder is the canonical fork progression for code generation, oldest to
// newest, each paired with the params.Rules field that activates it.
//
// Petersburg is omitted: it shares Constantinople's opcode set and only changes
// SSTORE dynamic gas, which flows through the shared gas function. Verkle/UBT is
// omitted: over its Shanghai base it adds no new opcodes (it only swaps gas and
// execute on existing opcodes), which the generated switch picks up from the
// active table at runtime.
var genForkOrder = []struct {
	name string
	rule string
	set  *JumpTable
}{
	{"Frontier", "", &frontierInstructionSet},
	{"Homestead", "IsHomestead", &homesteadInstructionSet},
	{"TangerineWhistle", "IsEIP150", &tangerineWhistleInstructionSet},
	{"SpuriousDragon", "IsEIP158", &spuriousDragonInstructionSet},
	{"Byzantium", "IsByzantium", &byzantiumInstructionSet},
	{"Constantinople", "IsConstantinople", &constantinopleInstructionSet},
	{"Istanbul", "IsIstanbul", &istanbulInstructionSet},
	{"Berlin", "IsBerlin", &berlinInstructionSet},
	{"London", "IsLondon", &londonInstructionSet},
	{"Merge", "IsMerge", &mergeInstructionSet},
	{"Shanghai", "IsShanghai", &shanghaiInstructionSet},
	{"Cancun", "IsCancun", &cancunInstructionSet},
	{"Prague", "IsPrague", &pragueInstructionSet},
	{"Osaka", "IsOsaka", &osakaInstructionSet},
	{"Amsterdam", "IsAmsterdam", &amsterdamInstructionSet},
}

// genFnName returns the short FuncForPC name of a jump-table function value
// (e.g. "gasKeccak256"), or "" if nil. An aliased var resolves to the underlying
// function (gasMLoad reports "pureMemoryGascost"), which is still stable across
// forks and so serves the directCold fork-invariance check.
func genFnName(fn any) string {
	v := reflect.ValueOf(fn)
	if !v.IsValid() || v.IsNil() {
		return ""
	}
	full := runtime.FuncForPC(v.Pointer()).Name()
	if i := strings.LastIndex(full, "."); i >= 0 {
		return full[i+1:]
	}
	return full
}

// GenForks returns per-fork opcode metadata for the interpreter code generator
// (core/vm/gen). It is exported solely for that purpose.
func GenForks() []GenFork {
	out := make([]GenFork, len(genForkOrder))
	for i, f := range genForkOrder {
		gf := GenFork{Name: f.name, RuleField: f.rule}
		for code := 0; code < 256; code++ {
			op := f.set[code]
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
		out[i] = gf
	}
	return out
}
