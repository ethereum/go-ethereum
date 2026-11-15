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
	"bytes"
	"testing"

	"github.com/ethereum/go-ethereum/core/types"
	"github.com/holiman/uint256"
)

func TestSigParamLeanSphincsSigner(t *testing.T) {
	pub := bytes.Repeat([]byte{0xab}, 32)
	pub[0] = 0x7f
	fc := &FrameContext{Signatures: types.SignatureList{{Scheme: types.FrameTxSchemeLeanSphincs, Signer: pub}}}
	evm := &EVM{TxContext: TxContext{FrameContext: fc}}
	scope := &ScopeContext{Stack: newStackForTesting()}
	defer scope.Stack.release()
	for selector, want := range map[uint64]*uint256.Int{
		0: new(uint256.Int).SetBytes(pub),
		1: uint256.NewInt(types.FrameTxSchemeLeanSphincs),
	} {
		scope.Stack.push(uint256.NewInt(selector))
		scope.Stack.push(uint256.NewInt(0))
		if _, err := opSigParam(nil, evm, scope); err != nil {
			t.Fatalf("selector %d: %v", selector, err)
		}
		got := scope.Stack.pop()
		if !got.Eq(want) {
			t.Fatalf("selector %d: got %x, want %x", selector, got.Bytes32(), want.Bytes32())
		}
	}
	for _, selector := range []uint64{3} {
		scope.Stack.push(uint256.NewInt(selector))
		scope.Stack.push(uint256.NewInt(0))
		if _, err := opSigParam(nil, evm, scope); err == nil {
			t.Fatalf("selector %d unexpectedly exposes signature data", selector)
		}
	}
}
