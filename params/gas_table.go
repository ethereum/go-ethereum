// Copyright 2016 The go-ethereum Authors
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

package params

// GasTable organizes gas prices for different ethereum phases.
type GasTable struct {
	Calls uint64
}

// Variables containing gas prices for different ethereum phases.
var (
	// GasTableHomestead contain the gas prices for
	// the homestead phase.
	GasTableHomestead = GasTable{
		Calls: 40,
	}

	// GasTableEIP150 contain the gas re-prices for
	// the EIP150 phase (a.k.a TangerineWhistle).
	GasTableEIP150 = GasTable{
		Calls: 700,
	}
	// GasTableEIP158 contain the gas re-prices for
	// the EIP155/EIP158 phase (a.k.a Spurious Dragon).
	GasTableEIP158 = GasTable{
		Calls: 700,
	}
)
