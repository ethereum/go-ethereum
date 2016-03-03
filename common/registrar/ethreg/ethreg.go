// Copyright 2015 The go-expanse Authors
// This file is part of the go-expanse library.
//
// The go-expanse library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The go-expanse library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the go-expanse library. If not, see <http://www.gnu.org/licenses/>.

package ethreg

import (
	"math/big"

	"github.com/expanse-project/go-expanse/common/registrar"
	"github.com/expanse-project/go-expanse/xeth"
)

// implements a versioned Registrar on an archiving full node
type EthReg struct {
	backend  *xeth.XEth
	registry *registrar.Registrar
}

func New(xe *xeth.XEth) (self *EthReg) {
	self = &EthReg{backend: xe}
	self.registry = registrar.New(xe)
	return
}

func (self *EthReg) Registry() *registrar.Registrar {
	return self.registry
}

func (self *EthReg) Resolver(n *big.Int) *registrar.Registrar {
	xe := self.backend
	if n != nil {
		xe = self.backend.AtStateNum(n.Int64())
	}
	return registrar.New(xe)
}
