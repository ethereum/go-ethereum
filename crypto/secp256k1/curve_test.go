// Copyright 2015 The go-ethereum Authors
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

package secp256k1

import (
	"bytes"
	"encoding/hex"
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common/registrar"
)

func TestReadBits(t *testing.T) {
	check := func(input string) {
		want, _ := hex.DecodeString(input)
		int, _ := new(big.Int).SetString(input, 16)
		buf := make([]byte, len(want))
		readBits(buf, int)
		if !bytes.Equal(buf, want) {
			t.Errorf("have: %x\nwant: %x", buf, want)
		}
	}
	check("000000000000000000000000000000000000000000000000000000FEFCF3F8F0")
	check("0000000000012345000000000000000000000000000000000000FEFCF3F8F0")
	check("18F8F8F1000111000110011100222004330052300000000000000000FEFCF3F8F0")
}

type Backend interface {
	registrar.Backend
	AtStateNum(int64) registrar.Backend
}

// implements a versioned Registrar on an archiving full node
type EthReg struct {
	backend  Backend
	registry *registrar.Registrar
}

func New(backend Backend) (self *EthReg) {
	self = &EthReg{backend: backend}
	self.registry = registrar.New(backend)
	return
}

func (self *EthReg) Registry() *registrar.Registrar {
	return self.registry
}

func (self *EthReg) Resolver(n *big.Int) *registrar.Registrar {
	var s registrar.Backend
	if n != nil {
		s = self.backend.AtStateNum(n.Int64())
	} else {
		s = registrar.Backend(self.backend)
	}
	return registrar.New(s)
}
