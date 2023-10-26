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
	"os"
	"reflect"
	"strings"

	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/params"
)

// Algorand implements the precompile to access the Algorand blockchain.
type Algorand struct {
	algodClient Client
}

// New creates a new Algorand precompiled contract with a client to access Algorand blockchain.
func New() *Algorand {
	algodAddress := os.Getenv("ALGOD_ADDRESS")
	algodToken := os.Getenv("ALGOD_TOKEN")
	if !strings.HasPrefix(algodAddress, "http://") {
		algodAddress = "http://" + algodAddress
	}
	algorand := &Algorand{
		algodClient: NewClient(algodAddress, algodToken),
	}
	return algorand
}

// RequiredGas estimates the gas required for running the point evaluation precompile.
func (a *Algorand) RequiredGas(input []byte) uint64 {
	return params.AlgorandPrecompileGas
}

// Run executes the Algorand precompile with the given input.
func (a *Algorand) Run(input []byte) ([]byte, error) {
	err := a.algodClient.CheckStatus()
	if err != nil {
		return nil, err
	}

	params, err := UnpackInput(input)
	if err != nil {
		return nil, err
	}
	var info interface{}
	switch params.GetCmdType() {
	case AccountCmd:
		info, err = a.algodClient.GetAccount(params.(*AccountInput).Address)
		if err != nil {
			return nil, err
		}
	}
	log.Info("Algorand.Run", "info", info)
	value := reflect.ValueOf(info).Elem().FieldByName(params.GetFieldName())
	if !value.IsValid() {
		return nil, fmt.Errorf("field %s does not exist", params.GetFieldName())
	}
	return pack(value)
}
