// Copyright 2016 The go-ethereum Authors
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

package backends

import (
	"math/big"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
)

// nilBackend implements bind.ContractBackend, but panics on any method call.
// Its sole purpose is to support the binding tests to construct the generated
// wrappers without calling any methods on them.
type nilBackend struct{}

func (*nilBackend) ContractCall(common.Address, []byte, bool) ([]byte, error) {
	panic("not implemented")
}
func (*nilBackend) GasLimit(common.Address, *common.Address, *big.Int, []byte) (*big.Int, error) {
	panic("not implemented")
}
func (*nilBackend) GasPrice() (*big.Int, error)                 { panic("not implemented") }
func (*nilBackend) AccountNonce(common.Address) (uint64, error) { panic("not implemented") }
func (*nilBackend) SendTransaction(*types.Transaction) error    { panic("not implemented") }

// NewNilBackend creates a new binding backend that can be used for instantiation
// but will panic on any invocation. Its sole purpose is to help testing.
func NewNilBackend() bind.ContractBackend {
	return new(nilBackend)
}
