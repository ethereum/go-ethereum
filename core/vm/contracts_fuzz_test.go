// Copyright 2023 The go-ethereum Authors
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
	"testing"

	"github.com/ethereum/go-ethereum/common"
)

func FuzzPrecompiledContracts(f *testing.F) {
	// Create list of addresses
	var addrs []common.Address
	for k := range allPrecompiles {
		addrs = append(addrs, k)
	}
	f.Fuzz(func(t *testing.T, addr uint8, input []byte) {
		a := addrs[int(addr)%len(addrs)]
		p := allPrecompiles[a]
		gas := p.RequiredGas(input)
		if gas > 10_000_000 {
			return
		}
		inWant := string(input)
		RunPrecompiledContract(p, input, gas)
		if inHave := string(input); inWant != inHave {
			t.Errorf("Precompiled %v modified input data", a)
		}
	})
}
