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
		return "prague"
	case Cancun:
		return "cancun"
	case Shanghai:
		return "shanghai"
	case Paris:
		return "paris"
	case GrayGlacier:
		return "grayGlacier"
	case ArrowGlacier:
		return "arrowGlacier"
	case London:
		return "london"
	case Berlin:
		return "berlin"
	case MuirGlacier:
		return "muirGlacier"
	case Istanbul:
		return "istanbul"
	case Petersburg:
		return "petersburg"
	case Constantinople:
		return "constantinople"
	case Byzantium:
		return "byzantium"
	case SpuriousDragon:
		return "spuriousDragon"
	case TangerineWhistle:
		return "tangerineWhistle"
	case DAO:
		return "dao"
	case Homestead:
		return "homestead"
	case FrontierThawing:
		return "frontierThawing"
	case Frontier:
		return "frontier"
	default:
		panic("unknown fork")
	}
}
