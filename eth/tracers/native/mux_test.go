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

package native

import (
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/tracing"
	"github.com/ethereum/go-ethereum/eth/tracers"
)

// TestMuxForwardsV2StateHooks verifies that the mux tracer fans out the V2
// variants of state-change hooks to child tracers. A child tracer that only
// implements OnCodeChangeV2 / OnNonceChangeV2 must still receive events when
// wrapped behind the mux. The mux must also fall back to the V1 hook when a
// child only implements V1, mirroring the precedence used in
// core/state_processor.go.
func TestMuxForwardsV2StateHooks(t *testing.T) {
	var (
		codeV2Calls  int
		nonceV2Calls int
		codeV1Calls  int
		nonceV1Calls int
	)
	v2Child := &tracers.Tracer{
		Hooks: &tracing.Hooks{
			OnCodeChangeV2: func(addr common.Address, prevCodeHash common.Hash, prevCode []byte, codeHash common.Hash, code []byte, reason tracing.CodeChangeReason) {
				codeV2Calls++
			},
			OnNonceChangeV2: func(addr common.Address, prev, new uint64, reason tracing.NonceChangeReason) {
				nonceV2Calls++
			},
		},
	}
	v1Child := &tracers.Tracer{
		Hooks: &tracing.Hooks{
			OnCodeChange: func(addr common.Address, prevCodeHash common.Hash, prevCode []byte, codeHash common.Hash, code []byte) {
				codeV1Calls++
			},
			OnNonceChange: func(addr common.Address, prev, new uint64) {
				nonceV1Calls++
			},
		},
	}
	mux, err := NewMuxTracer([]string{"v2", "v1"}, []*tracers.Tracer{v2Child, v1Child})
	if err != nil {
		t.Fatalf("NewMuxTracer: %v", err)
	}

	if mux.Hooks.OnCodeChangeV2 == nil {
		t.Fatal("mux does not expose OnCodeChangeV2; V2-only child tracers will miss code changes")
	}
	if mux.Hooks.OnNonceChangeV2 == nil {
		t.Fatal("mux does not expose OnNonceChangeV2; V2-only child tracers will miss nonce changes")
	}

	mux.Hooks.OnCodeChangeV2(common.Address{}, common.Hash{}, nil, common.Hash{}, nil, tracing.CodeChangeContractCreation)
	mux.Hooks.OnNonceChangeV2(common.Address{}, 0, 1, tracing.NonceChangeEoACall)

	if codeV2Calls != 1 {
		t.Fatalf("V2 child OnCodeChangeV2 got %d calls, want 1", codeV2Calls)
	}
	if nonceV2Calls != 1 {
		t.Fatalf("V2 child OnNonceChangeV2 got %d calls, want 1", nonceV2Calls)
	}
	if codeV1Calls != 1 {
		t.Fatalf("V1 child OnCodeChange got %d calls, want 1 (mux should fall back from V2 to V1)", codeV1Calls)
	}
	if nonceV1Calls != 1 {
		t.Fatalf("V1 child OnNonceChange got %d calls, want 1 (mux should fall back from V2 to V1)", nonceV1Calls)
	}
}
