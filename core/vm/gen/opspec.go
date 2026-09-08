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
	"fmt"
	"strings"

	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/params"
)

// This file holds the generator's opcode model: which tier each opcode is
// dispatched by, and the per-opcode spec derived from the per-fork jump tables.
//
// Two things decide a tier and they are kept apart. tierFor answers what the
// generator can safely emit: an opcode whose gas or stack bounds vary by fork
// cannot have them written out as constants, and a handler built by a closure has
// no name to write a call to. hotOps defines what is worth emitting.
//
// A case saves a fixed amount per execution, the indirect call through the table
// and the metering around it, so what it buys is that saving times the count, as
// a share of the time the opcode spends. A cheap opcode pays the overhead on top
// of very little work, so the share is large, while an expensive one absorbs it.
// Frequent low-gas opcodes are the ones worth a case. Against that, the switch is
// one large function competing for a 32KB L1 instruction cache, so a case that
// rarely runs still pushes the hot ones further apart.

// tier is how the dispatch handles one opcode.
type tier int

const (
	tierTable   tier = iota // through the active per-fork table, in the default case
	tierDynamic             // own case, handler called by name, dynamic gas via meterDynamicGas
	tierStatic              // own case, handler called by name, constant gas only
)

// hotOps are the opcodes that get their own case in the switch: the 60 most
// frequently executed of the ones tierFor says can have one, in rank order.
// Everything else is tierTable and goes through the default case, which walks the
// active per-fork table the way the legacy loop did.
//
// Counts are mainnet executions over 592,123 blocks, from
// lab.ethpandaops.io/api/v1/mainnet/fct_opcode_gas_by_opcode_hourly, kept at
// .evm-research-tmp/blockreplay/data/opcode_counts_592k_blocks.json.
var hotOps = []vm.OpCode{
	// Ranks 1-10, PUSH1 at 10.441% down to DUP1 at 3.455%.
	vm.PUSH1, vm.PUSH2, vm.JUMPDEST, vm.POP, vm.SWAP1,
	vm.JUMPI, vm.DUP2, vm.JUMP, vm.ADD, vm.DUP1,

	// 11-20, DUP3 3.341% to SWAP3 1.351%.
	vm.DUP3, vm.SWAP2, vm.ISZERO, vm.MSTORE, vm.MLOAD,
	vm.AND, vm.DUP4, vm.SUB, vm.EQ, vm.SWAP3,

	// 21-30, PUSH4 1.245% to PUSH20 0.660%.
	vm.PUSH4, vm.DUP5, vm.PUSH0, vm.LT, vm.SHL,
	vm.GT, vm.SWAP4, vm.DUP6, vm.MUL, vm.PUSH20,

	// 31-40, CALLDATALOAD 0.644% to NOT 0.297%.
	vm.CALLDATALOAD, vm.SHR, vm.DUP7, vm.PUSH32, vm.DIV,
	vm.DUP8, vm.CALLDATASIZE, vm.SWAP5, vm.KECCAK256, vm.NOT,

	// 41-50, OR 0.293% to SIGNEXTEND 0.140%.
	vm.OR, vm.SLT, vm.DUP9, vm.SWAP6, vm.PUSH8,
	vm.MULMOD, vm.GAS, vm.RETURNDATASIZE, vm.DUP10, vm.SIGNEXTEND,

	// 51-60, CALLER 0.139% down to the cut at CODECOPY 0.097%.
	vm.CALLER, vm.SGT, vm.DUP11, vm.PUSH3, vm.SAR,
	vm.SWAP7, vm.DUP12, vm.CALLVALUE, vm.RETURN, vm.CODECOPY,

	// Rank 61 is ADDMOD at 0.095%, then PUSH16, DUP13, SWAP8, XOR, CALLDATACOPY,
	// DUP14 and SWAP9. Nothing below the cut reaches a tenth of a percent, and the
	// 74 eligible opcodes left out are 1.26% of executions between them.
}

// tierFor returns the tier an opcode can be dispatched by. tierTable comes back
// with the reason the opcode cannot take its own case, which is what the abort in
// deriveSpecs reports.
//
// An opcode qualifies when it is defined, every fork from its introduction on
// defines it and agrees on its metadata, and its handler is a named top-level
// function.
func (g *generator) tierFor(code byte, forks []vm.GenFork) (tier, string) {
	spec := g.specs[code]
	if !spec.Defined {
		return tierTable, "no fork defines it"
	}
	if strings.Contains(spec.ExecuteFn, ".") {
		return tierTable, fmt.Sprintf("its handler is the closure %q, which has no name to call", spec.ExecuteFn)
	}
	var introduced bool
	for _, fork := range forks {
		o := fork.Ops[code]
		if !o.Defined {
			if introduced {
				return tierTable, fmt.Sprintf("fork %s removes it, which a case gated on its introducing fork cannot express", fork.Name)
			}
			continue
		}
		introduced = true
		if o != spec.GenOp {
			return tierTable, fmt.Sprintf("fork %s changes its gas, stack bounds or functions, so they cannot be emitted as constants", fork.Name)
		}
	}
	if spec.DynamicGasFn == "" {
		return tierStatic, ""
	}
	return tierDynamic, ""
}

// tierOf returns how the dispatch handles an opcode. deriveSpecs fills this in.
func (g *generator) tierOf(code byte) tier {
	return g.tiers[code]
}

// skippedForks are forks the switch gets no lane for. Verkle/UBT is the only one,
// and the skip is a no-op today, since LookupInstructionSet has no verkle table yet
// and hands back Cancun's. It matters once there is one: enable4762 only repoints
// existing opcodes, which the switch picks up from the active table anyway, and
// PUSH1-PUSH32 among them would stop generation as fork-varying.
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

// opSpec holds the per-opcode facts the generator emits from: the metadata the
// first defining fork records for the opcode, plus which fork that was.
type opSpec struct {
	vm.GenOp
	fork string
}

// stackGuards returns the bounds emitStackChecks needs, plus which of the two
// guards are worth emitting. A minStack of 0 cannot underflow, and a maxStack
// at the stack limit cannot overflow, so those are left out.
func (s opSpec) stackGuards() (minStack, maxStack int, under, over bool) {
	return s.MinStack, s.MaxStack, s.MinStack > 0, s.MaxStack < int(params.StackLimit)
}

// stackDelta returns how much an opcode changes the stack depth, which is push
// minus pop. maxStack is built as StackLimit+pop-push (see stack_table.go), so
// the difference from the limit is the net effect. ADD's maxStack is 1025, so it
// is -1. PUSH1's is 1023, so +1. JUMPDEST leaves the stack alone at 0. The
// dispatch uses this to keep its own depth counter rather than reading
// stack.size back through a pointer on every opcode.
func (s opSpec) stackDelta() int {
	return int(params.StackLimit) - s.MaxStack
}

// deriveSpecs records each opcode's constants and function names from the first
// fork that defines it, then gives every opcode hotOps names its tier.
func (g *generator) deriveSpecs(forks []vm.GenFork) {
	for code := range 256 {
		for _, fork := range forks {
			o := fork.Ops[code]
			if !o.Defined {
				continue
			}
			g.specs[code] = opSpec{GenOp: o, fork: fork.RuleField}
			break // first fork that defines it wins (its intro fork)
		}
	}

	g.assignTiers(hotOps, forks)
}

// assignTiers gives each listed opcode the tier it qualifies for. Everything else
// keeps tierTable. An entry that does not qualify would silently get the general
// path instead of the case it was listed for, so stop rather than emit a switch
// nobody asked for.
func (g *generator) assignTiers(ops []vm.OpCode, forks []vm.GenFork) {
	for _, op := range ops {
		t, why := g.tierFor(byte(op), forks)
		if t == tierTable {
			abortf("opcode %#x (%s) is in hotOps but cannot take its own case: %s", byte(op), op, why)
		}
		g.tiers[op] = t
	}
}
