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
)

type CmdType = uint8

const (
	// AccountCmd is the type of the command to get account information.
	AccountCmd CmdType = iota
)

// Input is the interface for the input of the Algorand precompiled contract.
type Input interface {
	GetCmdType() CmdType  // GetCmdType returns the command type.
	GetFieldName() string // GetFieldName returns the name of the field to get.
}

// AccountInput is the input for the command to get account information.
type AccountInput struct {
	Cmd       CmdType // Command type.
	FieldName string  // Name of the field to get.
	Address   string  // Address of the account.
}

// GetCmdType returns the command type.
func (input *AccountInput) GetCmdType() CmdType {
	return input.Cmd
}

// GetFieldName returns the name of the field to get.
func (input *AccountInput) GetFieldName() string {
	return input.FieldName
}

// getCmdTypeFromRawInput gets the command type from the raw input.
func getCmdTypeFromRawInput(inputBytes []byte) (CmdType, error) {
	cmd := new(CmdType)
	err := unpack(inputBytes, cmd)
	if err != nil {
		return 0, err
	}
	return *cmd, nil
}

// UnpackInput decodes the raw input into the input of the Algorand precompiled contract.
func UnpackInput(inputBytes []byte) (Input, error) {
	cmd, err := getCmdTypeFromRawInput(inputBytes)
	if err != nil {
		return nil, err
	}
	switch cmd {
	case AccountCmd:
		input := new(AccountInput)
		err = unpack(inputBytes, input)
		if err != nil {
			return nil, err
		}
		return input, nil
	default:
		return nil, fmt.Errorf("unknown command type: %d", cmd)
	}
}

// abiType returns the ABI of the given name and type.
func abiType(name string, typ reflect.Type) string {
	if typ.Kind() == reflect.Ptr {
		return abiType(name, typ.Elem())
	} else if typ.Kind() == reflect.Struct {
		var fields []string
		for i := 0; i < typ.NumField(); i++ {
			fields = append(fields, abiType(typ.Field(i).Name, typ.Field(i).Type))
		}
		return strings.Join(fields, ", ")
	} else if typ.Kind() == reflect.Int {
		return fmt.Sprintf(`{"name": "%s", "type": "int256"}`, name)
	} else if typ.Kind() == reflect.Uint {
		return fmt.Sprintf(`{"name": "%s", "type": "uint256"}`, name)
	} else {
		return fmt.Sprintf(`{"name": "%s", "type": "%s"}`, name, typ.String())
	}
}

// unpack decodes the raw data into the given output interface.
func unpack(rawData []byte, output interface{}) error {
	typ := reflect.TypeOf(output)
	defn := fmt.Sprintf(`[{"type": "function", "outputs": [%s]}]`, abiType("", typ))
	abi, err := abi.JSON(strings.NewReader(defn))
	if err != nil {
		return err
	}
	err = abi.UnpackIntoInterface(output, "", rawData)
	if err != nil {
		return err
	}
	return nil
}
