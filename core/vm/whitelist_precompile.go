// Copyright 2014 The go-ethereum Authors
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
	"github.com/ethereum/go-ethereum/common"
)

// WhitelistPrecompile is a precompiled contract for address whitelisting.
type WhitelistPrecompile struct {
	whitelist map[common.Address]struct{}
}

func NewWhitelistPrecompile(addresses []common.Address) *WhitelistPrecompile {
	wl := make(map[common.Address]struct{})
	for _, addr := range addresses {
		wl[addr] = struct{}{}
	}
	return &WhitelistPrecompile{whitelist: wl}
}

func (w *WhitelistPrecompile) RequiredGas(input []byte) uint64 {
	return 1000
}

func (w *WhitelistPrecompile) Run(input []byte) ([]byte, error) {
	// Expect input: first 20 bytes = caller address
	return []byte{}, nil

}

func (w *WhitelistPrecompile) Name() string {
	return "WhitelistPrecompile"
}
