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

	// The rest of DUP and SWAP. Each family is one parametric case, see families,
	// so widening its range to these costs no code and no clause, and the
	// frequency cut above does not apply to them. Together they are 0.39% of
	// executions, DUP13 at 0.079% down to SWAP16 at 0.001%.
	vm.DUP13, vm.DUP14, vm.DUP15, vm.DUP16,
	vm.SWAP8, vm.SWAP9, vm.SWAP10, vm.SWAP11, vm.SWAP12, vm.SWAP13, vm.SWAP14, vm.SWAP15, vm.SWAP16,

	// Rank 61 is ADDMOD at 0.095%, then PUSH16, XOR and CALLDATACOPY. Nothing
	// below the cut reaches a tenth of a percent, and the 61 eligible opcodes
	// left out are 0.87% of executions between them.
}

// opFamily is a run of opcodes the dispatch collapses into one parametric case.
// The members differ only in one constant, n, which the case recovers from the
// opcode byte, so a case each would cost bytes in the dispatch and buy nothing.
// DUP and SWAP are the whole of it: both charge one constant gas, both have an
// underflow bound that is n plus a fixed offset, and both have a body that is one
// stack method taking n.
//
// A family changes how its members are emitted, not which of them get the fast
// path. That is still hotOps, and the case covers exactly the members listed
// there. They have to form a run starting at base, because the case recovers n
// from the distance to it.
type opFamily struct {
	base vm.OpCode // the member with n == 1
	body string    // the parametric body, with n in scope
}

// families are the runs the dispatch collapses.
var families = []opFamily{
	{base: vm.DUP1, body: "stack.dup(n)"},
	{base: vm.SWAP1, body: "stack.swap(n)"},
}

// familyOf returns the family an opcode belongs to, if any. Membership runs
// from the base to the last member the opcode table defines with the same
// mnemonic stem, which for DUP and SWAP is sixteen values.
func familyOf(code byte) (opFamily, bool) {
	for _, f := range families {
		if code >= byte(f.base) && code < byte(f.base)+familyWidth {
			return f, true
		}
	}
	return opFamily{}, false
}

// familyWidth is how many members DUP and SWAP each have.
const familyWidth = 16

// familyRun returns the members of a family that hotOps gives the fast path, as
// the highest one, or ok false when none has it. They have to be a run from the
// base: a member with a case cannot sit past one without, because the case is
// one range and n is recovered from the byte. Stop rather than leave a listed
// member without the case it was listed for.
func (g *generator) familyRun(f opFamily) (last byte, ok bool) {
	base := byte(f.base)
	for code := base; code < base+familyWidth; code++ {
		if g.tierOf(code) == tierTable {
			for rest := code + 1; rest < base+familyWidth; rest++ {
				if g.tierOf(rest) != tierTable {
					abortf("opcode %#x (%s) is in hotOps but %s is not, and the %s family is emitted as one run from %s",
						rest, g.specs[rest].Name, g.specs[code].Name, g.specs[base].Name, g.specs[base].Name)
				}
			}
			return code - 1, code > base
		}
	}
	return base + familyWidth - 1, true
}

// familyFacts returns what a family's case emits as constants, after checking
// every member up to last agrees on them. The underflow bound is the one thing
// allowed to vary, and minOffset is how far it sits above n: 0 for DUP, whose
// DUPn needs n items, 1 for SWAP, whose SWAPn needs n+1. Anything else
// differing across members would make the shared case wrong for some of them,
// so stop rather than emit it.
func (g *generator) familyFacts(f opFamily, last byte) (minOffset, maxStack int, gas uint64, delta int) {
	base := g.specs[byte(f.base)]
	minOffset, maxStack, gas, delta = base.MinStack-1, base.MaxStack, base.ConstantGas, base.stackDelta()
	for code := byte(f.base); code <= last; code++ {
		spec, n := g.specs[code], int(code-byte(f.base))+1
		switch {
		case g.tierOf(code) != tierStatic:
			abortf("opcode %#x (%s) is in the %s family but is not on the static tier, so the family case cannot emit its gas",
				code, spec.Name, base.Name)
		case spec.MinStack != n+minOffset:
			abortf("opcode %#x (%s) needs %d stack items, but the %s family case would check for %d",
				code, spec.Name, spec.MinStack, base.Name, n+minOffset)
		case spec.MaxStack != maxStack || spec.ConstantGas != gas || spec.stackDelta() != delta:
			abortf("opcode %#x (%s) disagrees with %s on gas, overflow bound or stack delta, so they cannot share a case",
				code, spec.Name, base.Name)
		}
	}
	return minOffset, maxStack, gas, delta
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
