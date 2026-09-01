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
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/params"
)

// This file holds the generator's opcode model: which tier each opcode is
// dispatched by, and the per-opcode spec derived from the per-fork jump tables.

// tier is how the dispatch handles one opcode.
type tier int

const (
	tierTable  tier = iota // through the active per-fork table, in the default case
	tierDirect             // handler, gas and memory functions called by name
	tierInline             // handler body spliced into the loop
)

// opTiers gives an opcode its own case in the switch. Anything absent is
// tierTable and falls through to the default case, which is where every
// fork-varying op (CALL, CREATE, SSTORE, LOG, the copy family) belongs.
//
// tierInline is for hot, fork-stable ops with no dynamic gas. The handler whose
// body gets spliced is derived from the per-fork tables (see deriveSpecs), never
// restated here. Every one of them is a top-level opXxx, since emitInlineOp has
// no way to reach into a closure.
//
// tierDirect is for ops with dynamic gas whose handler, gas and memory functions
// are identical in every fork (checkDirectStable enforces it), so they can be
// called by name rather than through a table function pointer Go cannot inline.
var opTiers = func() map[vm.OpCode]tier {
	t := map[vm.OpCode]tier{
		vm.ADD: tierInline, vm.MUL: tierInline, vm.SUB: tierInline, vm.DIV: tierInline, vm.SDIV: tierInline,
		vm.MOD: tierInline, vm.SMOD: tierInline, vm.ADDMOD: tierInline, vm.MULMOD: tierInline, vm.SIGNEXTEND: tierInline,
		vm.LT: tierInline, vm.GT: tierInline, vm.SLT: tierInline, vm.SGT: tierInline, vm.EQ: tierInline, vm.ISZERO: tierInline,
		vm.AND: tierInline, vm.OR: tierInline, vm.XOR: tierInline, vm.NOT: tierInline, vm.BYTE: tierInline,
		vm.SHL: tierInline, vm.SHR: tierInline, vm.SAR: tierInline, vm.CLZ: tierInline,
		vm.POP: tierInline, vm.JUMP: tierInline, vm.JUMPI: tierInline,
		vm.PC: tierInline, vm.MSIZE: tierInline, vm.JUMPDEST: tierInline,
		vm.PUSH0: tierInline, vm.PUSH1: tierInline, vm.PUSH2: tierInline,
		vm.CALLDATALOAD: tierInline,

		// An aliased gas var derives as the function behind it, so MLOAD's
		// charge is emitted as pureMemoryGascost, not the gasMLoad var.
		vm.KECCAK256: tierDirect,
		vm.MLOAD:     tierDirect,
		vm.MSTORE:    tierDirect,
	}
	for op := vm.PUSH3; op <= vm.PUSH32; op++ {
		t[op] = tierInline
	}
	for op := vm.DUP1; op <= vm.DUP16; op++ {
		t[op] = tierInline
	}
	for op := vm.SWAP1; op <= vm.SWAP16; op++ {
		t[op] = tierInline
	}
	for _, op := range coldOps {
		delete(t, op)
	}
	return t
}()

// coldOps are opcodes that qualify for a fast tier but run too rarely to be worth
// the bytes. An arm costs 16 to 53 lines of the switch, and the switch is one
// large function against a 32KB L1 instruction cache, so arms that never run
// still push the hot ones further apart.
//
// The cut is 0.05% of mainnet executions. These 41 arms are together under half
// a percent of all execution.
//
// Dropping an opcode here only moves it to the table tier, which is the general
// path and correct for every opcode in every fork, so this cannot affect
// behaviour. Eligibility is still decided by checkInlineStable and
// checkDirectStable above, and this list only ever removes.
//
// Frequencies are mainnet execution counts over 592,123 blocks, from
// lab.ethpandaops.io/api/v1/mainnet/fct_opcode_gas_by_opcode_hourly. A thirty
// block sample got four of these wrong at the boundary, so re-derive from a wide
// range if the opcode mix shifts.
var coldOps = []vm.OpCode{
	// arithmetic and misc
	vm.MSIZE, vm.PC, vm.CLZ, vm.SMOD, vm.MOD, vm.SDIV, vm.BYTE,

	// MSTORE8 was tierDirect. The direct tier saves a fixed cost per execution,
	// one indirect call plus meterDynamicGas's nil checks, so its value tracks the
	// count and not how expensive the opcode's work is. At 0.022% that is a few
	// hundred executions a block, which does not pay for a 53 line arm.
	vm.MSTORE8,

	// the tail of each family. DUP decays gently so only the last two go, SWAP
	// falls off faster, and PUSH is bimodal: the widths that mean something stay
	// (1, 2, 3, 4, 8, 16, 20, 32) and the rest are noise.
	vm.DUP15, vm.DUP16,
	vm.SWAP10, vm.SWAP11, vm.SWAP12, vm.SWAP13, vm.SWAP14, vm.SWAP15, vm.SWAP16,
	vm.PUSH5, vm.PUSH6, vm.PUSH7, vm.PUSH9, vm.PUSH10, vm.PUSH11, vm.PUSH12,
	vm.PUSH13, vm.PUSH14, vm.PUSH15, vm.PUSH17, vm.PUSH18, vm.PUSH19, vm.PUSH21,
	vm.PUSH22, vm.PUSH23, vm.PUSH24, vm.PUSH25, vm.PUSH26, vm.PUSH27, vm.PUSH28,
	vm.PUSH29, vm.PUSH30, vm.PUSH31,
}

// tierOf returns how the dispatch handles an opcode.
func tierOf(code byte) tier {
	return opTiers[vm.OpCode(code)]
}

// skippedForks are forks the switch gets no lane for. Verkle/UBT is the only one,
// and the skip is a no-op today, since LookupInstructionSet has no verkle table yet
// and hands back Cancun's. It matters once there is one: enable4762 only repoints
// existing opcodes, which the switch picks up from the active table anyway, and
// PUSH1-PUSH32 among them would trip checkInlineStable.
var skippedForks = map[string]bool{"IsUBT": true}

// genForks returns the fork lanes the generator derives its specs from.
func genForks() []vm.GenFork {
	var out []vm.GenFork
	for _, fork := range vm.GenForks() {
		if !skippedForks[fork.RuleField] {
			out = append(out, fork)
		}
	}
	return out
}

// opSpec holds the per-opcode facts the generator emits from: the constants
// (gas, stack bounds, intro fork) and the FuncForPC names of the opcode's
// handler, dynamic-gas and memory-size functions, all derived from the
// per-fork tables.
type opSpec struct {
	defined  bool
	name     string
	fork     string
	constGas uint64
	minStack int
	maxStack int
	execFn   string
	dynFn    string
	memFn    string
}

// stackGuards returns the bounds emitStackChecks needs, plus which of the two
// guards are worth emitting. A minStack of 0 cannot underflow, and a maxStack
// at the stack limit cannot overflow, so those are left out. Emitting both
// unconditionally would grow execUntraced by 14%, because the compiler cannot
// track sp's range across a switch this size and emits every compare.
func (s opSpec) stackGuards() (minStack, maxStack int, under, over bool) {
	return s.minStack, s.maxStack, s.minStack > 0, s.maxStack < int(params.StackLimit)
}

// deriveSpecs records each opcode's constants and function names from the first fork
// that defines it, then checks that everything opTiers gives a case is fork-stable
// enough to emit that way.
func (g *generator) deriveSpecs(forks []vm.GenFork) {
	for code := range 256 {
		for _, fork := range forks {
			o := fork.Ops[code]
			if !o.Defined {
				continue
			}
			g.specs[code] = opSpec{
				defined:  true,
				name:     o.Name,
				fork:     fork.RuleField,
				constGas: o.ConstantGas,
				minStack: o.MinStack,
				maxStack: o.MaxStack,
				execFn:   o.ExecuteFn,
				dynFn:    o.DynamicGasFn,
				memFn:    o.MemorySizeFn,
			}
			break // first fork that defines it wins (its intro fork)
		}
	}

	// Both tiers that get their own case emit static gas and stack bounds as
	// constants, so both must be fork-stable. Bail loudly otherwise.
	for code := range 256 {
		switch b := byte(code); tierOf(b) {
		case tierInline:
			g.checkInlineStable(b, forks)
		case tierDirect:
			g.checkDirectStable(b, forks)
		}
	}
}

// checkInlineStable verifies a tierInline opcode is safe to inline.
func (g *generator) checkInlineStable(code byte, forks []vm.GenFork) {
	// The spec is what gets emitted, so there has to be one.
	spec := g.specs[code]
	if !spec.defined {
		abortf("opcode %#x selected for inlining but never defined", code)
	}
	for _, fork := range forks {
		// Nothing to compare in a fork where the opcode does not exist.
		o := fork.Ops[code]
		if !o.Defined {
			continue
		}
		// The handler body, gas and stack bounds all come from the first defining
		// fork, so a later fork changing any of them would be silently ignored.
		// Dynamic gas is barred outright: an inlined op charges only its constant.
		if o.ExecuteFn != spec.execFn || o.ConstantGas != spec.constGas || o.MinStack != spec.minStack || o.MaxStack != spec.maxStack || o.DynamicGasFn != "" {
			abortf("opcode %#x (%s) is not fork-stable (fork %s): cannot inline", code, spec.name, fork.Name)
		}
	}
}

// checkDirectStable verifies a tierDirect opcode is safe to direct-call. Unlike
// checkInlineStable it allows dynamic gas, which these ops carry by definition.
func (g *generator) checkDirectStable(code byte, forks []vm.GenFork) {
	spec := g.specs[code]
	if !spec.defined {
		abortf("opcode %#x (tierDirect) is never defined", code)
	}
	for _, fork := range forks {
		o := fork.Ops[code]
		if !o.Defined {
			continue
		}
		// Emitted as constants, so they cannot vary.
		if o.ConstantGas != spec.constGas || o.MinStack != spec.minStack || o.MaxStack != spec.maxStack {
			abortf("opcode %#x (%s) is tierDirect but not fork-stable (fork %s): static gas or stack bounds vary, cannot emit as constants", code, spec.name, fork.Name)
		}
		// Called by the first defining fork's names, so a fork that swapped one
		// would be run with the wrong function.
		if o.ExecuteFn != spec.execFn || o.DynamicGasFn != spec.dynFn || o.MemorySizeFn != spec.memFn {
			abortf("opcode %#x (%s) is tierDirect but its functions vary by fork (fork %s): got %s/%s/%s, want %s/%s/%s, cannot direct-call",
				code, spec.name, fork.Name, o.ExecuteFn, o.DynamicGasFn, o.MemorySizeFn, spec.execFn, spec.dynFn, spec.memFn)
		}
	}
}
