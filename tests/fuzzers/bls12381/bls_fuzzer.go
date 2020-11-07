// Copyright 2020 The go-ethereum Authors
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

package bls

import (
	"bytes"
	"fmt"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/vm"
)

// The function must return
// 1 if the fuzzer should increase priority of the
//    given input during subsequent fuzzing (for example, the input is lexically
//    correct and was parsed successfully);
// -1 if the input must not be added to corpus even if gives new coverage; and
// 0  otherwise
// other values are reserved for future use.
func Fuzz(data []byte) int {

	// The bls ones are at 10 - 18
	var precompiles = []vm.PrecompiledContract{
		vm.PrecompiledContractsYoloV2[common.BytesToAddress([]byte{10})],
		vm.PrecompiledContractsYoloV2[common.BytesToAddress([]byte{11})],
		vm.PrecompiledContractsYoloV2[common.BytesToAddress([]byte{12})],
		vm.PrecompiledContractsYoloV2[common.BytesToAddress([]byte{13})],
		vm.PrecompiledContractsYoloV2[common.BytesToAddress([]byte{14})],
		vm.PrecompiledContractsYoloV2[common.BytesToAddress([]byte{15})],
		vm.PrecompiledContractsYoloV2[common.BytesToAddress([]byte{16})],
		vm.PrecompiledContractsYoloV2[common.BytesToAddress([]byte{17})],
		vm.PrecompiledContractsYoloV2[common.BytesToAddress([]byte{18})],
	}

	cpy := make([]byte, len(data))
	copy(cpy, data)
	var useful = false
	for i, precompile := range precompiles {
		precompile.RequiredGas(cpy)
		if _, err := precompile.Run(cpy); err == nil {
			useful = true
		}
		if !bytes.Equal(cpy, data) {
			panic(fmt.Sprintf("input data modified, precompile %d: %x %x", i, data, cpy))
		}
	}
	if !useful {
		// Input not great
		return 0
	}
	return 1
}
