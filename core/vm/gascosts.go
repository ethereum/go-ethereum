// Copyright 2026 The go-ethereum Authors
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

import "fmt"

// GasCosts denotes a vector of gas costs in the
// multidimensional metering paradigm.
type GasCosts struct {
	RegularGas uint64
	StateGas   uint64
}

// Sum returns the total gas (regular + state).
func (g GasCosts) Sum() uint64 {
	return g.RegularGas + g.StateGas
}

// String returns a visual representation of the gas vector.
func (g GasCosts) String() string {
	return fmt.Sprintf("<%v,%v>", g.RegularGas, g.StateGas)
}
