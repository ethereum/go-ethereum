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

package algorand

import (
	"fmt"
	"reflect"
	"strings"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/log"
)

// pack encodes the value into bytes.
func pack(value reflect.Value) ([]byte, error) {
	// Convert int to int64 or uint to uint64.
	// This is because the ABI does not support int or uint.
	// The cast is safe because int/uint is either 32-bit or 64-bit.
	if value.Kind() == reflect.Int {
		value = reflect.ValueOf(int64(value.Int()))
	} else if value.Kind() == reflect.Uint {
		value = reflect.ValueOf(uint64(value.Uint()))
	}

	defn := fmt.Sprintf(`[{"type": "constructor", "inputs": [%s]}]`, abiType("", value.Type()))
	abi, err := abi.JSON(strings.NewReader(defn))
	if err != nil {
		return nil, err
	}

	data, err := abi.Pack("", value.Interface())
	if err != nil {
		return nil, err
	}
	log.Info("Pack", "data", common.Bytes2Hex(data))
	return data, nil
}
