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
	"reflect"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/params"
)

// EIP-8297 swaps the state commitment. It does not touch the opcode set or its
// pricing: the reference implementation's binary_tree fork is the underlying
// fork's VM verbatim on a different state backend. So the binary tree must be
// invisible to instruction-set selection, and these tests pin that.
//
// This replaces an earlier pair of tests asserting the opposite - that the tree
// layered EIP-4762's gas schedule onto each fork. That composition was geth's
// own invention, inherited from the verkle era, and is what made PBT work only
// on the forks somebody remembered to enumerate.

// pbtForkLadder is every fork the binary tree can be activated on. A new fork
// belongs here; if one is added and PBT starts diverging on it, these tests are
// what notice.
var pbtForkLadder = []struct {
	name  string
	rules params.Rules
}{
	{"shanghai", params.Rules{IsShanghai: true}},
	{"cancun", params.Rules{IsCancun: true}},
	{"prague", params.Rules{IsPrague: true}},
	{"osaka", params.Rules{IsOsaka: true}},
	{"amsterdam", params.Rules{IsAmsterdam: true}},
	{"bogota", params.Rules{IsBogota: true}},
}

// sameOperation compares two jump-table entries semantically.
//
// Pointer identity is useless here: LookupInstructionSet builds a fresh table
// on every call, so two *identical* lookups already differ at every opcode.
// Function values are not comparable in Go either, hence the reflect-based
// comparison of their code pointers - which is what actually distinguishes,
// say, gasSStore from gasSStore4762.
func sameOperation(a, b *operation) bool {
	if a == nil || b == nil {
		return a == b
	}
	if a.constantGas != b.constantGas || a.minStack != b.minStack ||
		a.maxStack != b.maxStack || a.undefined != b.undefined {
		return false
	}
	fn := func(x any) uintptr { return reflect.ValueOf(x).Pointer() }
	return fn(a.execute) == fn(b.execute) &&
		fn(a.dynamicGas) == fn(b.dynamicGas) &&
		fn(a.memorySize) == fn(b.memorySize)
}

// TestPBTLeavesInstructionSetAlone pins that turning the binary tree on changes
// no opcode at any fork level.
func TestPBTLeavesInstructionSetAlone(t *testing.T) {
	for _, tc := range pbtForkLadder {
		t.Run(tc.name, func(t *testing.T) {
			plain, err := LookupInstructionSet(tc.rules)
			if err != nil {
				t.Fatalf("lookup without the binary tree: %v", err)
			}
			pbtRules := tc.rules
			pbtRules.IsPBT = true
			withPBT, err := LookupInstructionSet(pbtRules)
			if err != nil {
				t.Fatalf("lookup with the binary tree: %v", err)
			}
			for op := 0; op < 256; op++ {
				if !sameOperation(plain[op], withPBT[op]) {
					t.Fatalf("opcode %s differs under the binary tree", OpCode(op))
				}
			}
		})
	}
}

// TestSameOperationDetectsAGasSwap guards the comparison above. Without it a
// sameOperation that returned true unconditionally would make the test it
// serves pass on any implementation at all.
func TestSameOperationDetectsAGasSwap(t *testing.T) {
	base := newAmsterdamInstructionSet()
	swapped := base[SSTORE]
	altered := *swapped
	altered.constantGas = swapped.constantGas + 1

	if sameOperation(swapped, swapped) != true {
		t.Fatal("sameOperation says an operation differs from itself")
	}
	if sameOperation(swapped, &altered) {
		t.Fatal("sameOperation cannot see a changed gas cost")
	}
}

// TestPBTTableSelectedByEVM pins the same property through the constructor the
// EVM actually uses, not only the exported lookup: the two select tables in
// separate switches and could drift apart.
func TestPBTTableSelectedByEVM(t *testing.T) {
	// Amsterdam-with-binary-tree is the combination the BinaryTree fork means.
	cfg := &params.ChainConfig{
		ChainID:                 big.NewInt(1),
		HomesteadBlock:          big.NewInt(0),
		EIP150Block:             big.NewInt(0),
		EIP155Block:             big.NewInt(0),
		EIP158Block:             big.NewInt(0),
		ByzantiumBlock:          big.NewInt(0),
		ConstantinopleBlock:     big.NewInt(0),
		PetersburgBlock:         big.NewInt(0),
		IstanbulBlock:           big.NewInt(0),
		BerlinBlock:             big.NewInt(0),
		LondonBlock:             big.NewInt(0),
		MergeNetsplitBlock:      big.NewInt(0),
		TerminalTotalDifficulty: common.Big0,
		ShanghaiTime:            new(uint64),
		CancunTime:              new(uint64),
		PragueTime:              new(uint64),
		OsakaTime:               new(uint64),
		AmsterdamTime:           new(uint64),
	}
	pbtTime := uint64(0)
	withTree := *cfg
	withTree.PBTTime = &pbtTime

	blockCtx := BlockContext{BlockNumber: big.NewInt(1), Random: &common.Hash{}, Time: 1}
	plain := NewEVM(blockCtx, nil, cfg, Config{})
	tree := NewEVM(blockCtx, nil, &withTree, Config{})

	if !tree.chainRules.IsPBT {
		t.Fatal("the binary tree is not active; this proves nothing")
	}
	if plain.table != tree.table {
		t.Fatal("the EVM selected a different instruction set for the binary tree")
	}
}

// TestPBTKeepsForkPrecompiles pins that the binary tree does not change which
// precompiles are active.
//
// It used to pin them to Berlin's set, which silently dropped KZG point
// evaluation, the BLS precompiles and p256Verify from Cancun onwards - and
// disagreed with the exported ActivePrecompiles, so the EVM and the access-list
// warming in Prepare had different ideas about which addresses were precompiled.
func TestPBTKeepsForkPrecompiles(t *testing.T) {
	for _, tc := range pbtForkLadder {
		t.Run(tc.name, func(t *testing.T) {
			pbtRules := tc.rules
			pbtRules.IsPBT = true

			plain := activePrecompiledContracts(tc.rules)
			withPBT := activePrecompiledContracts(pbtRules)
			if plain != withPBT {
				t.Fatal("the binary tree selected a different precompile set")
			}
			// The two selectors must also agree with each other: the EVM reads
			// one and statedb.Prepare warms addresses from the other.
			active := ActivePrecompiles(pbtRules)
			if len(active) != len(*withPBT) {
				t.Fatalf("ActivePrecompiles lists %d addresses, the active set has %d", len(active), len(*withPBT))
			}
			for _, addr := range active {
				if _, ok := (*withPBT)[addr]; !ok {
					t.Fatalf("%x is listed as active but is not in the set the EVM uses", addr)
				}
			}
		})
	}
}
