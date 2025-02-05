// Copyright 2025 The go-ethereum Authors
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

package tracing

import (
	"errors"
	"math/big"
	"reflect"
	"testing"

	"github.com/ethereum/go-ethereum/common"
)

type testTracer struct {
	t       *testing.T
	bal     *big.Int
	nonce   uint64
	code    []byte
	storage map[common.Hash]common.Hash
}

func (t *testTracer) OnBalanceChange(addr common.Address, prev *big.Int, new *big.Int, reason BalanceChangeReason) {
	t.t.Logf("OnBalanceChange(%v, %v -> %v, %v)", addr, prev, new, reason)
	if t.bal != nil && t.bal.Cmp(prev) != 0 {
		t.t.Errorf("  !! wrong prev balance (expected %v)", t.bal)
	}
	t.bal = new
}

func (t *testTracer) OnNonceChange(addr common.Address, prev uint64, new uint64) {
	t.t.Logf("OnNonceChange(%v, %v -> %v)", addr, prev, new)
	t.nonce = new
}

func (t *testTracer) OnNonceChangeV2(addr common.Address, prev uint64, new uint64, reason NonceChangeReason) {
	t.t.Logf("OnNonceChangeV2(%v, %v -> %v, %v)", addr, prev, new, reason)
	t.nonce = new
}

func (t *testTracer) OnCodeChange(addr common.Address, prevCodeHash common.Hash, prevCode []byte, codeHash common.Hash, code []byte) {
	t.t.Logf("OnCodeChange(%v, %v -> %v)", addr, prevCodeHash, codeHash)
	t.code = code
}

func (t *testTracer) OnStorageChange(addr common.Address, slot common.Hash, prev common.Hash, new common.Hash) {
	t.t.Logf("OnStorageCodeChange(%v, %v, %v -> %v)", addr, slot, prev, new)
	if t.storage == nil {
		t.storage = make(map[common.Hash]common.Hash)
	}
	if new == (common.Hash{}) {
		delete(t.storage, slot)
	} else {
		t.storage[slot] = new
	}
}

func TestJournalIntegration(t *testing.T) {
	tr := &testTracer{t: t}
	wr, err := WrapWithJournal(&Hooks{OnBalanceChange: tr.OnBalanceChange, OnNonceChange: tr.OnNonceChange, OnCodeChange: tr.OnCodeChange, OnStorageChange: tr.OnStorageChange})
	if err != nil {
		t.Fatalf("failed to wrap test tracer: %v", err)
	}

	addr := common.HexToAddress("0x1234")
	{
		wr.OnEnter(0, 0, addr, addr, nil, 1000, big.NewInt(0))
		wr.OnBalanceChange(addr, nil, big.NewInt(100), BalanceChangeUnspecified)
		wr.OnCodeChange(addr, common.Hash{}, nil, common.Hash{}, []byte{1, 2, 3})
		wr.OnStorageChange(addr, common.Hash{1}, common.Hash{}, common.Hash{2})
		{
			wr.OnEnter(1, 0, addr, addr, nil, 1000, big.NewInt(0))
			wr.OnNonceChangeV2(addr, 0, 1, NonceChangeUnspecified)
			wr.OnBalanceChange(addr, big.NewInt(100), big.NewInt(200), BalanceChangeUnspecified)
			wr.OnBalanceChange(addr, big.NewInt(200), big.NewInt(250), BalanceChangeUnspecified)
			wr.OnStorageChange(addr, common.Hash{1}, common.Hash{2}, common.Hash{3})
			wr.OnStorageChange(addr, common.Hash{2}, common.Hash{}, common.Hash{4})
			wr.OnExit(1, nil, 100, errors.New("revert"), true)
		}
		wr.OnExit(0, nil, 150, nil, false)
	}

	if tr.bal.Cmp(big.NewInt(100)) != 0 {
		t.Fatalf("unexpected balance: %v", tr.bal)
	}
	if tr.nonce != 0 {
		t.Fatalf("unexpected nonce: %v", tr.nonce)
	}
	if len(tr.code) != 3 {
		t.Fatalf("unexpected code: %v", tr.code)
	}
	if len(tr.storage) != 1 {
		t.Fatalf("unexpected storage len. want %d, have %d", 1, len(tr.storage))
	}
	if tr.storage[common.Hash{1}] != (common.Hash{2}) {
		t.Fatalf("unexpected storage. want %v, have %v", common.Hash{2}, tr.storage[common.Hash{1}])
	}
}

func TestJournalTopRevert(t *testing.T) {
	tr := &testTracer{t: t}
	wr, err := WrapWithJournal(&Hooks{OnBalanceChange: tr.OnBalanceChange, OnNonceChange: tr.OnNonceChange})
	if err != nil {
		t.Fatalf("failed to wrap test tracer: %v", err)
	}

	addr := common.HexToAddress("0x1234")
	{
		wr.OnEnter(0, 0, addr, addr, nil, 1000, big.NewInt(0))
		wr.OnBalanceChange(addr, big.NewInt(0), big.NewInt(100), BalanceChangeUnspecified)
		{
			wr.OnEnter(1, 0, addr, addr, nil, 1000, big.NewInt(0))
			wr.OnNonceChangeV2(addr, 0, 1, NonceChangeUnspecified)
			wr.OnBalanceChange(addr, big.NewInt(100), big.NewInt(200), BalanceChangeUnspecified)
			wr.OnBalanceChange(addr, big.NewInt(200), big.NewInt(250), BalanceChangeUnspecified)
			wr.OnExit(1, nil, 100, errors.New("revert"), true)
		}
		wr.OnExit(0, nil, 150, errors.New("revert"), true)
	}

	if tr.bal.Cmp(big.NewInt(0)) != 0 {
		t.Fatalf("unexpected balance: %v", tr.bal)
	}
	if tr.nonce != 0 {
		t.Fatalf("unexpected nonce: %v", tr.nonce)
	}
}

// This test checks that changes in nested calls are reverted properly.
func TestJournalNestedCalls(t *testing.T) {
	tr := &testTracer{t: t}
	wr, err := WrapWithJournal(&Hooks{OnBalanceChange: tr.OnBalanceChange, OnNonceChange: tr.OnNonceChange})
	if err != nil {
		t.Fatalf("failed to wrap test tracer: %v", err)
	}

	addr := common.HexToAddress("0x1234")
	{
		wr.OnEnter(0, 0, addr, addr, nil, 1000, big.NewInt(0))
		wr.OnBalanceChange(addr, big.NewInt(0), big.NewInt(100), BalanceChangeUnspecified)
		{
			wr.OnEnter(1, 0, addr, addr, nil, 1000, big.NewInt(0))
			wr.OnBalanceChange(addr, big.NewInt(100), big.NewInt(200), BalanceChangeUnspecified)
			{
				wr.OnEnter(2, 0, addr, addr, nil, 1000, big.NewInt(0))
				wr.OnExit(2, nil, 100, nil, false)
			}
			{
				wr.OnEnter(2, 0, addr, addr, nil, 1000, big.NewInt(0))
				wr.OnBalanceChange(addr, big.NewInt(200), big.NewInt(300), BalanceChangeUnspecified)
				wr.OnExit(2, nil, 100, nil, false)
			}
			{
				wr.OnEnter(2, 0, addr, addr, nil, 1000, big.NewInt(0))
				wr.OnExit(2, nil, 100, nil, false)
			}
			wr.OnBalanceChange(addr, big.NewInt(300), big.NewInt(400), BalanceChangeUnspecified)
			{
				wr.OnEnter(2, 0, addr, addr, nil, 1000, big.NewInt(0))
				wr.OnBalanceChange(addr, big.NewInt(400), big.NewInt(500), BalanceChangeUnspecified)
				wr.OnExit(2, nil, 100, errors.New("revert"), true)
			}
			{
				wr.OnEnter(2, 0, addr, addr, nil, 1000, big.NewInt(0))
				wr.OnExit(2, nil, 100, errors.New("revert"), true)
			}
			{
				wr.OnEnter(2, 0, addr, addr, nil, 1000, big.NewInt(0))
				wr.OnBalanceChange(addr, big.NewInt(400), big.NewInt(600), BalanceChangeUnspecified)
				wr.OnExit(2, nil, 100, nil, false)
			}
			wr.OnExit(1, nil, 100, errors.New("revert"), true)
		}
		wr.OnExit(0, nil, 150, nil, false)
	}

	if tr.bal.Uint64() != 100 {
		t.Fatalf("unexpected balance: %v", tr.bal)
	}
}

func TestNonceIncOnCreate(t *testing.T) {
	const opCREATE = 0xf0

	tr := &testTracer{t: t}
	wr, err := WrapWithJournal(&Hooks{OnNonceChange: tr.OnNonceChange})
	if err != nil {
		t.Fatalf("failed to wrap test tracer: %v", err)
	}

	addr := common.HexToAddress("0x1234")
	{
		wr.OnEnter(0, opCREATE, addr, addr, nil, 1000, big.NewInt(0))
		wr.OnNonceChangeV2(addr, 0, 1, NonceChangeContractCreator)
		wr.OnExit(0, nil, 100, errors.New("revert"), true)
	}

	if tr.nonce != 1 {
		t.Fatalf("unexpected nonce: %v", tr.nonce)
	}
}

func TestOnNonceChangeV2(t *testing.T) {
	tr := &testTracer{t: t}
	wr, err := WrapWithJournal(&Hooks{OnNonceChangeV2: tr.OnNonceChangeV2})
	if err != nil {
		t.Fatalf("failed to wrap test tracer: %v", err)
	}

	addr := common.HexToAddress("0x1234")
	{
		wr.OnEnter(2, 0, addr, addr, nil, 1000, big.NewInt(0))
		wr.OnNonceChangeV2(addr, 0, 1, NonceChangeEoACall)
		wr.OnExit(2, nil, 100, nil, true)
	}

	if tr.nonce != 0 {
		t.Fatalf("unexpected nonce: %v", tr.nonce)
	}
}

func TestAllHooksCalled(t *testing.T) {
	tracer := newTracerAllHooks()
	hooks := tracer.hooks()

	wrapped, err := WrapWithJournal(hooks)
	if err != nil {
		t.Fatalf("failed to wrap hooks with journal: %v", err)
	}

	// Get the underlying value of the wrapped hooks
	wrappedValue := reflect.ValueOf(wrapped).Elem()
	wrappedType := wrappedValue.Type()

	// Iterate over all fields of the wrapped hooks
	for i := 0; i < wrappedType.NumField(); i++ {
		field := wrappedType.Field(i)

		// Skip fields that are not function types
		if field.Type.Kind() != reflect.Func {
			continue
		}
		// Skip non-hooks, i.e. Copy
		if field.Name == "copy" {
			continue
		}
		// Skip if field is not set
		if wrappedValue.Field(i).IsNil() {
			continue
		}

		// Get the method
		method := wrappedValue.Field(i)

		// Call the method with zero values
		params := make([]reflect.Value, method.Type().NumIn())
		for j := 0; j < method.Type().NumIn(); j++ {
			params[j] = reflect.Zero(method.Type().In(j))
		}
		method.Call(params)
	}

	// Check if all hooks were called
	if tracer.numCalled() != tracer.hooksCount() {
		t.Errorf("Not all hooks were called. Expected %d, got %d", tracer.hooksCount(), tracer.numCalled())
	}

	for hookName, called := range tracer.hooksCalled {
		if !called {
			t.Errorf("Hook %s was not called", hookName)
		}
	}
}

type tracerAllHooks struct {
	hooksCalled map[string]bool
}

func newTracerAllHooks() *tracerAllHooks {
	t := &tracerAllHooks{hooksCalled: make(map[string]bool)}
	// Initialize all hooks to false. We will use this to
	// get total count of hooks.
	hooksType := reflect.TypeOf((*Hooks)(nil)).Elem()
	for i := 0; i < hooksType.NumField(); i++ {
		t.hooksCalled[hooksType.Field(i).Name] = false
	}
	delete(t.hooksCalled, "OnNonceChange")
	return t
}

func (t *tracerAllHooks) hooksCount() int {
	return len(t.hooksCalled)
}

func (t *tracerAllHooks) numCalled() int {
	count := 0
	for _, called := range t.hooksCalled {
		if called {
			count++
		}
	}
	return count
}

func (t *tracerAllHooks) hooks() *Hooks {
	h := &Hooks{}
	// Create a function for each hook that sets the
	// corresponding hooksCalled field to true.
	hooksValue := reflect.ValueOf(h).Elem()
	for i := 0; i < hooksValue.NumField(); i++ {
		field := hooksValue.Type().Field(i)
		if field.Name == "OnNonceChange" {
			continue
		}
		hookMethod := reflect.MakeFunc(field.Type, func(args []reflect.Value) []reflect.Value {
			t.hooksCalled[field.Name] = true
			return nil
		})
		hooksValue.Field(i).Set(hookMethod)
	}
	return h
}
