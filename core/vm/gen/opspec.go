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

// hotOps are the opcodes that get their own case in the switch. Everything else
// is tierTable and goes through the default case, which walks the active per-fork
// table the way the legacy loop did.
var hotOps = []vm.OpCode{
	// Arithmetic, comparison and bitwise.
	vm.ADD, vm.MUL, vm.SUB, vm.DIV, vm.MULMOD, vm.SIGNEXTEND,
	vm.LT, vm.GT, vm.SLT, vm.SGT, vm.EQ, vm.ISZERO,
	vm.AND, vm.OR, vm.NOT, vm.SHL, vm.SHR, vm.SAR,

	// Environment reads, stack and control flow.
	vm.CALLER, vm.CALLVALUE, vm.CALLDATALOAD, vm.CALLDATASIZE, vm.RETURNDATASIZE,
	vm.POP, vm.JUMP, vm.JUMPI, vm.GAS, vm.JUMPDEST,

	// Dynamic gas, so these pay a table load either way and their case saves less
	// than the rest.
	vm.KECCAK256, vm.CODECOPY, vm.MLOAD, vm.MSTORE, vm.RETURN,

	// The PUSH widths that carry something: 4 a selector, 20 an address, 32 a
	// hash or a full-width constant. The others are rare.
	vm.PUSH0, vm.PUSH1, vm.PUSH2, vm.PUSH3, vm.PUSH4, vm.PUSH8, vm.PUSH20, vm.PUSH32,

	// DUP and SWAP
	vm.DUP1, vm.DUP2, vm.DUP3, vm.DUP4, vm.DUP5, vm.DUP6, vm.DUP7, vm.DUP8,
	vm.DUP9, vm.DUP10, vm.DUP11, vm.DUP12, vm.DUP13, vm.DUP14, vm.DUP15, vm.DUP16,
	vm.SWAP1, vm.SWAP2, vm.SWAP3, vm.SWAP4, vm.SWAP5, vm.SWAP6, vm.SWAP7, vm.SWAP8,
	vm.SWAP9, vm.SWAP10, vm.SWAP11, vm.SWAP12, vm.SWAP13, vm.SWAP14, vm.SWAP15, vm.SWAP16,
}

// opFamily is a run of opcodes like DUP1 to DUP16 that the dispatch emits as one
// shared case. The members differ only in n, which the case works out from the
// opcode, so giving each its own case would just make the dispatch bigger. DUP and
// SWAP are the only families. Both charge a fixed gas, need n plus a fixed number
// of stack items, and have a body that's one stack call taking n. hotOps still
// decides which members get the fast path, and the case covers exactly those. They
// have to form a run starting at base, since the generator emits the case as one
// range from there.
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

// lastHotFamilyMember returns the highest member of a family that's in hotOps, or
// ok false if none are. The family's case is emitted as one range from base, so a
// hot member after one that isn't would quietly lose its case. The generator stops
// instead.
func (g *generator) lastHotFamilyMember(f opFamily) (last byte, ok bool) {
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

// familyConstants returns the values a family's case emits as constants, after
// checking every member up to last agrees on them. The underflow bound is the one
// thing that varies, so it returns minOffset, how far the bound sits above n. That's
// 0 for DUP, since DUPn needs n items, and 1 for SWAP, since SWAPn needs n+1. If any
// other value differs between members, the shared case would be wrong for some of
// them, so the generator stops instead.
func (g *generator) familyConstants(f opFamily, last byte) (minOffset, maxStack int, gas uint64, delta int) {
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
