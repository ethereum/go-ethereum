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

package forks

// Fork is a numerical identifier of specific network upgrades (forks).
type Fork int

const (
	Frontier = iota
	FrontierThawing
	Homestead
	DAO
	TangerineWhistle
	SpuriousDragon
	Byzantium
	Constantinople
	Petersburg
	Istanbul
	MuirGlacier
	Berlin
	London
	ArrowGlacier
	GrayGlacier
	Paris
	Shanghai
	Cancun
	Prague
)

func (f Fork) String() string {
	switch f {
	case Prague:
		return "Prague"
	case Cancun:
		return "Cancun"
	case Shanghai:
		return "Shanghai"
	case Paris:
		return "Paris"
	case GrayGlacier:
		return "GrayGlacier"
	case ArrowGlacier:
		return "ArrowGlacier"
	case London:
		return "London"
	case Berlin:
		return "Berlin"
	case MuirGlacier:
		return "MuirGlacier"
	case Istanbul:
		return "Istanbul"
	case Petersburg:
		return "Petersburg"
	case Constantinople:
		return "Constantinople"
	case Byzantium:
		return "Byzantium"
	case SpuriousDragon:
		return "SpuriousDragon"
	case TangerineWhistle:
		return "TangerineWhistle"
	case DAO:
		return "Dao"
	case Homestead:
		return "Homestead"
	case FrontierThawing:
		return "FrontierThawing"
	case Frontier:
		return "Frontier"
	default:
		panic("unknown fork")
	}
}
