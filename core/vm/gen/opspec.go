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
	tierTable   tier = iota // through the active per-fork table, in the default case
	tierDynamic             // own case, handler called by name, dynamic gas via meterDynamicGas
	tierStatic              // own case, handler called by name, constant gas only
)

// opTiers gives an opcode its own case in the switch. Anything absent is
// tierTable and falls through to the default case, which is where every
// fork-varying op (CALL, CREATE, SSTORE, LOG, the copy family) belongs.
//
// tierStatic is for hot, fork-stable ops whose whole cost is their constant gas.
// The handler name is derived from the per-fork tables (see deriveSpecs), never
// restated here. Every one of them must be a top-level opXxx, since a closure has
// no name to call.
//
// tierDynamic is for ops with dynamic gas whose handler is identical in every fork
// (checkDynamicStable enforces it). They still reach meterDynamicGas through a
// table load, so the only thing their own case buys over the default is calling
// the handler by name and emitting the static gas and stack bounds as constants.
var opTiers = func() map[vm.OpCode]tier {
	t := map[vm.OpCode]tier{
		vm.ADD: tierStatic, vm.MUL: tierStatic, vm.SUB: tierStatic, vm.DIV: tierStatic, vm.SDIV: tierStatic,
		vm.MOD: tierStatic, vm.SMOD: tierStatic, vm.ADDMOD: tierStatic, vm.MULMOD: tierStatic, vm.SIGNEXTEND: tierStatic,
		vm.LT: tierStatic, vm.GT: tierStatic, vm.SLT: tierStatic, vm.SGT: tierStatic, vm.EQ: tierStatic, vm.ISZERO: tierStatic,
		vm.AND: tierStatic, vm.OR: tierStatic, vm.XOR: tierStatic, vm.NOT: tierStatic, vm.BYTE: tierStatic,
		vm.SHL: tierStatic, vm.SHR: tierStatic, vm.SAR: tierStatic, vm.CLZ: tierStatic,
		vm.POP: tierStatic, vm.JUMP: tierStatic, vm.JUMPI: tierStatic,
		vm.PC: tierStatic, vm.MSIZE: tierStatic, vm.JUMPDEST: tierStatic,
		vm.PUSH0: tierStatic, vm.PUSH1: tierStatic, vm.PUSH2: tierStatic,
		vm.CALLDATALOAD: tierStatic,

		// An aliased gas var derives as the function behind it, so MLOAD's
		// charge is emitted as pureMemoryGascost, not the gasMLoad var.
		vm.KECCAK256: tierDynamic,
		vm.MLOAD:     tierDynamic,
		vm.MSTORE:    tierDynamic,
	}
	for op := vm.PUSH3; op <= vm.PUSH32; op++ {
		t[op] = tierStatic
	}
	for op := vm.DUP1; op <= vm.DUP16; op++ {
		t[op] = tierStatic
	}
	for op := vm.SWAP1; op <= vm.SWAP16; op++ {
		t[op] = tierStatic
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
// behaviour. Eligibility is still decided by checkStaticStable and
// checkDynamicStable above, and this list only ever removes.
//
// Frequencies are mainnet execution counts over 592,123 blocks, from
// lab.ethpandaops.io/api/v1/mainnet/fct_opcode_gas_by_opcode_hourly. A thirty
// block sample got four of these wrong at the boundary, so re-derive from a wide
// range if the opcode mix shifts.
var coldOps = []vm.OpCode{
	// arithmetic and misc
	vm.MSIZE, vm.PC, vm.CLZ, vm.SMOD, vm.MOD, vm.SDIV, vm.BYTE,

	// MSTORE8 was tierDynamic. The direct tier saves a fixed cost per execution,
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
// PUSH1-PUSH32 among them would trip checkStaticStable.
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
// unconditionally grew the spliced dispatch by 14%, because the compiler cannot
// track the stack length across a switch this size and emits every compare.
func (s opSpec) stackGuards() (minStack, maxStack int, under, over bool) {
	return s.minStack, s.maxStack, s.minStack > 0, s.maxStack < int(params.StackLimit)
}

// stackDelta returns how much an opcode changes the stack depth, which is push
// minus pop. maxStack is built as StackLimit+pop-push (see stack_table.go), so
// the difference from the limit is the net effect. ADD's maxStack is 1025, so it
// is -1. PUSH1's is 1023, so +1. JUMPDEST leaves the stack alone at 0.
//
// The dispatch uses this to keep its own depth counter rather than reading
// stack.size back through a pointer on every opcode.
func (s opSpec) stackDelta() int {
	return int(params.StackLimit) - s.maxStack
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
		case tierStatic:
			g.checkStaticStable(b, forks)
		case tierDynamic:
			g.checkDynamicStable(b, forks)
		}
	}
}

// checkStaticStable verifies a tierStatic opcode is safe to give its own case.
func (g *generator) checkStaticStable(code byte, forks []vm.GenFork) {
	// The spec is what gets emitted, so there has to be one.
	spec := g.specs[code]
	if !spec.defined {
		abortf("opcode %#x selected for its own case but never defined", code)
	}
	for _, fork := range forks {
		// Nothing to compare in a fork where the opcode does not exist.
		o := fork.Ops[code]
		if !o.Defined {
			continue
		}
		// The handler name, gas and stack bounds all come from the first defining
		// fork, so a later fork changing any of them would be silently ignored.
		// Dynamic gas is barred outright: a tierStatic op charges only its constant.
		if o.ExecuteFn != spec.execFn || o.ConstantGas != spec.constGas || o.MinStack != spec.minStack || o.MaxStack != spec.maxStack || o.DynamicGasFn != "" {
			abortf("opcode %#x (%s) is not fork-stable (fork %s): cannot give it its own case", code, spec.name, fork.Name)
		}
	}
}

// checkDynamicStable verifies a tierDynamic opcode is safe to direct-call. Unlike
// checkStaticStable it allows dynamic gas, which these ops carry by definition.
func (g *generator) checkDynamicStable(code byte, forks []vm.GenFork) {
	spec := g.specs[code]
	if !spec.defined {
		abortf("opcode %#x (tierDynamic) is never defined", code)
	}
	for _, fork := range forks {
		o := fork.Ops[code]
		if !o.Defined {
			continue
		}
		// Emitted as constants, so they cannot vary.
		if o.ConstantGas != spec.constGas || o.MinStack != spec.minStack || o.MaxStack != spec.maxStack {
			abortf("opcode %#x (%s) is tierDynamic but not fork-stable (fork %s): static gas or stack bounds vary, cannot emit as constants", code, spec.name, fork.Name)
		}
		// Called by the first defining fork's names, so a fork that swapped one
		// would be run with the wrong function.
		if o.ExecuteFn != spec.execFn || o.DynamicGasFn != spec.dynFn || o.MemorySizeFn != spec.memFn {
			abortf("opcode %#x (%s) is tierDynamic but its functions vary by fork (fork %s): got %s/%s/%s, want %s/%s/%s, cannot direct-call",
				code, spec.name, fork.Name, o.ExecuteFn, o.DynamicGasFn, o.MemorySizeFn, spec.execFn, spec.dynFn, spec.memFn)
		}
	}
}
