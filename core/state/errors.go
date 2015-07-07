// Copyright 2014 The go-ethereum Authors
// This file is part of go-ethereum.
//
// go-ethereum is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// go-ethereum is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with go-ethereum.  If not, see <http://www.gnu.org/licenses/>.

package state

import (
	"fmt"
	"math/big"
)

type GasLimitErr struct {
	Message string
	Is, Max *big.Int
}

func IsGasLimitErr(err error) bool {
	_, ok := err.(*GasLimitErr)

	return ok
}
func (err *GasLimitErr) Error() string {
	return err.Message
}
func GasLimitError(is, max *big.Int) *GasLimitErr {
	return &GasLimitErr{Message: fmt.Sprintf("GasLimit error. Max %s, transaction would take it to %s", max, is), Is: is, Max: max}
}
