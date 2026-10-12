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

package logger

import (
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/holiman/uint256"
)

// opContext is a tracing.OpContext exposing only a stack and an address.
type opContext struct {
	stack []uint256.Int
	addr  common.Address
}

func (c *opContext) MemoryData() []byte       { return nil }
func (c *opContext) StackData() []uint256.Int { return c.stack }
func (c *opContext) Caller() common.Address   { return common.Address{} }
func (c *opContext) Address() common.Address  { return c.addr }
func (c *opContext) CallValue() *uint256.Int  { return new(uint256.Int) }
func (c *opContext) CallInput() []byte        { return nil }
func (c *opContext) ContractCode() []byte     { return nil }

// TestAccessListTracerSetCodeFrom checks that the source account read by
// SETCODEFROM (EIP-8298) enters the generated access list.
func TestAccessListTracerSetCodeFrom(t *testing.T) {
	source := common.HexToAddress("0x1111111111111111111111111111111111111111")
	tracer := NewAccessListTracer(nil, nil)
	scope := &opContext{stack: []uint256.Int{*new(uint256.Int).SetBytes(source.Bytes())}}
	tracer.OnOpcode(0, byte(vm.SETCODEFROM), 0, 0, scope, nil, 1, nil)
	if got := tracer.AccessList(); len(got) != 1 || got[0].Address != source {
		t.Fatalf("access list %+v, want only %x", got, source)
	}
}

func TestNewAccessListTracerExcludedAddress(t *testing.T) {
	excluded := common.HexToAddress("0x2222222222222222222222222222222222222222")
	slot := common.HexToHash("0x01")
	prelude := types.AccessList{{
		Address:     excluded,
		StorageKeys: []common.Hash{slot},
	}}
	excl := map[common.Address]struct{}{excluded: {}}
	tracer := NewAccessListTracer(prelude, excl)
	got := tracer.AccessList()
	if len(got) != 0 {
		t.Fatalf("excluded prelude address must not contribute tuples, got %+v", got)
	}
}
