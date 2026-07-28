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

import (
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"

	"github.com/ethereum/go-ethereum/params"
)

// TestPBTInstructionSetComposes pins that activating the binary tree keeps
// EIP-4762's state-access pricing at every fork level, and keeps the fork's
// own opcodes. Selecting the set by fork level first would silently drop
// EIP-4762 for any tree activated on Osaka or later, leaving the access-event
// bookkeeping running while nothing charged for it.
func TestPBTInstructionSetComposes(t *testing.T) {
	// EIP-4762 replaces the pricing of these state-access opcodes; the
	// dynamic gas function is what changes, so compare against the
	// corresponding non-PBT set.
	stateAccessOps := []OpCode{SSTORE, SLOAD, BALANCE, EXTCODESIZE, EXTCODEHASH, EXTCODECOPY, SELFDESTRUCT}

	// Every slot of a validated table is populated (undefined opcodes get a
	// placeholder), so presence proves nothing: compare operation identity
	// against the same fork's non-PBT set instead.
	var modifiedAt map[string][]OpCode = map[string][]OpCode{}

	for _, tc := range []struct {
		name  string
		rules params.Rules
		plain *JumpTable
	}{
		{"shanghai", params.Rules{IsShanghai: true, IsPBT: true}, &shanghaiInstructionSet},
		{"cancun", params.Rules{IsCancun: true, IsPBT: true}, &cancunInstructionSet},
		{"prague", params.Rules{IsPrague: true, IsPBT: true}, &pragueInstructionSet},
		{"osaka", params.Rules{IsOsaka: true, IsPBT: true}, &osakaInstructionSet},
		{"amsterdam", params.Rules{IsAmsterdam: true, IsPBT: true}, &amsterdamInstructionSet},
	} {
		t.Run(tc.name, func(t *testing.T) {
			pbt := pbtInstructionSet(tc.rules)
			for _, op := range stateAccessOps {
				if (*pbt)[op] == nil {
					t.Fatalf("%s: opcode %s missing from the binary tree set", tc.name, op)
				}
				if (*pbt)[op] == (*tc.plain)[op] {
					t.Fatalf("%s: opcode %s keeps its non-4762 pricing", tc.name, op)
				}
			}
			// Everything else is the fork's own set, untouched: the tree
			// layers onto it rather than replacing it.
			for op := range *tc.plain {
				if (*pbt)[op] != (*tc.plain)[op] {
					modifiedAt[tc.name] = append(modifiedAt[tc.name], OpCode(op))
				}
			}
		})
	}
	// EIP-4762 repriced the same opcodes at every fork level; if a base fork
	// were silently swapping the whole table, this set would differ.
	var reference []OpCode
	for name, ops := range modifiedAt {
		if reference == nil {
			reference = ops
			continue
		}
		if len(ops) != len(reference) {
			t.Fatalf("%s reprices %d opcodes, another fork level reprices %d", name, len(ops), len(reference))
		}
		for i := range ops {
			if ops[i] != reference[i] {
				t.Fatalf("%s reprices %s where another fork level reprices %s", name, ops[i], reference[i])
			}
		}
	}

	// Amsterdam's own additions must survive the composition, and must not
	// leak below their fork. Undefined slots carry an explicit marker, which
	// is what distinguishes "the fork defines this opcode" from "the table
	// filled the hole".
	amsterdamPBT := pbtInstructionSet(params.Rules{IsAmsterdam: true, IsPBT: true})
	osakaPBT := pbtInstructionSet(params.Rules{IsOsaka: true, IsPBT: true})
	if amsterdamInstructionSet[SLOTNUM].undefined || !osakaInstructionSet[SLOTNUM].undefined {
		t.Fatal("SLOTNUM is not an Amsterdam addition in this tree; pick another marker opcode")
	}
	if (*amsterdamPBT)[SLOTNUM].undefined {
		t.Fatal("amsterdam: SLOTNUM lost when the binary tree composed onto it")
	}
	if !(*osakaPBT)[SLOTNUM].undefined {
		t.Fatal("osaka: the binary tree introduced SLOTNUM before its fork")
	}
}

// TestPBTTableSelectedByEVM pins the selection itself: an EVM built from a
// binary-tree chain config must run on the composed table. This is the
// regression guard for the switch order in NewEVM - with the fork cases
// checked first, an Osaka-or-later binary tree silently ran without
// EIP-4762 pricing.
func TestPBTTableSelectedByEVM(t *testing.T) {
	zero := new(uint64)
	cfg := &params.ChainConfig{
		ChainID: big.NewInt(1), HomesteadBlock: big.NewInt(0), EIP150Block: big.NewInt(0),
		EIP155Block: big.NewInt(0), EIP158Block: big.NewInt(0), ByzantiumBlock: big.NewInt(0),
		ConstantinopleBlock: big.NewInt(0), PetersburgBlock: big.NewInt(0), IstanbulBlock: big.NewInt(0),
		BerlinBlock: big.NewInt(0), LondonBlock: big.NewInt(0), MergeNetsplitBlock: big.NewInt(0),
		TerminalTotalDifficulty: big.NewInt(0),
		ShanghaiTime:            zero, CancunTime: zero, PragueTime: zero,
		OsakaTime: zero, AmsterdamTime: zero, PBTTime: zero,
	}
	var randomHash common.Hash
	evm := NewEVM(BlockContext{BlockNumber: big.NewInt(0), Random: &randomHash, Time: 0}, nil, cfg, Config{})
	if !evm.chainRules.IsPBT || !evm.chainRules.IsAmsterdam {
		t.Fatalf("test config does not activate both: IsPBT=%v IsAmsterdam=%v", evm.chainRules.IsPBT, evm.chainRules.IsAmsterdam)
	}
	if evm.table != pbtInstructionSet(evm.chainRules) {
		t.Fatal("EVM selected the plain fork table under the binary tree: EIP-4762 pricing is not applied")
	}
}

// TestLookupInstructionSetPBT pins that the exported lookup agrees with the
// EVM's own selection instead of erroring out.
func TestLookupInstructionSetPBT(t *testing.T) {
	rules := params.Rules{IsAmsterdam: true, IsPBT: true}
	table, err := LookupInstructionSet(rules)
	if err != nil {
		t.Fatalf("lookup failed for the binary tree: %v", err)
	}
	want := pbtInstructionSet(rules)
	for op := range table {
		got, expect := table[op], (*want)[op]
		if got.undefined != expect.undefined || got.constantGas != expect.constantGas ||
			got.minStack != expect.minStack || got.maxStack != expect.maxStack ||
			(got.dynamicGas == nil) != (expect.dynamicGas == nil) {
			t.Fatalf("opcode %s disagrees with the EVM's own selection", OpCode(op))
		}
	}
}
