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

import "fmt"

// Fork is a numerical identifier of specific network upgrades (forks).
type Fork uint32

const (
	Frontier         Fork = iota | blockBased
	FrontierThawing  Fork = iota | blockBased | optional
	Homestead        Fork = iota | blockBased
	DAO              Fork = iota | blockBased | optional
	TangerineWhistle Fork = iota | blockBased // a.k.a. the EIP150 fork
	SpuriousDragon   Fork = iota | blockBased // a.k.a. the EIP155/EIP158 fork
	Byzantium        Fork = iota | blockBased
	Constantinople   Fork = iota | blockBased
	Petersburg       Fork = iota | blockBased
	Istanbul         Fork = iota | blockBased
	MuirGlacier      Fork = iota | blockBased | optional
	Berlin           Fork = iota | blockBased
	London           Fork = iota | blockBased
	ArrowGlacier     Fork = iota | blockBased | optional
	GrayGlacier      Fork = iota | blockBased | optional
	Paris            Fork = iota | blockBased | ismerge
	Shanghai         Fork = iota
	Cancun           Fork = iota | hasBlobs
	Prague           Fork = iota | hasBlobs
	Osaka            Fork = iota | hasBlobs
	Verkle           Fork = iota | optional
)

var CanonOrder = []Fork{
	Frontier,
	FrontierThawing,
	Homestead,
	DAO,
	TangerineWhistle,
	SpuriousDragon,
	Byzantium,
	Constantinople,
	Petersburg,
	Istanbul,
	MuirGlacier,
	Berlin,
	London,
	ArrowGlacier,
	GrayGlacier,
	Paris,
	Shanghai,
	Cancun,
	Prague,
	Osaka,
	Verkle,
}

const (
	// Config bits: these bits are set on specific fork enum values and encode metadata
	// about the fork.
	blockBased = 1 << 31
	ismerge    = 1 << 30
	optional   = 1 << 29
	hasBlobs   = 1 << 28

	// The config bits can be stripped using this bit mask.
	unconfigMask = ^Fork(0) >> 8
)

// IsMerge returns true for the merge fork.
func (f Fork) IsMerge() bool {
	return f&ismerge != 0
}

// Optional reports whether the fork can be left out of the config.
func (f Fork) Optional() bool {
	return f&optional != 0
}

// BlockBased reports whether the fork is scheduled by block number instead of timestamp.
// This is true for pre-merge forks.
func (f Fork) BlockBased() bool {
	return f&blockBased != 0
}

// HasBlobs reports whether the fork must have a corresponding blob count configuration.
func (f Fork) HasBlobs() bool {
	return f&hasBlobs != 0
}

func (f Fork) After(other Fork) bool {
	return f&unconfigMask >= other&unconfigMask
}

// String implements fmt.Stringer.
func (f Fork) String() string {
	s, ok := forkToString[f]
	if !ok {
		return fmt.Sprintf("unknownFork(%#x)", f)
	}
	return s
}

var forkToString = map[Fork]string{
	Frontier:         "Frontier",
	Homestead:        "Homestead",
	DAO:              "DAOFork",
	TangerineWhistle: "TangerineWhistle",
	SpuriousDragon:   "SpuriousDragon",
	Byzantium:        "Byzantium",
	Constantinople:   "Constantinople",
	Petersburg:       "Petersburg",
	Istanbul:         "Istanbul",
	MuirGlacier:      "MuirGlacier",
	Berlin:           "Berlin",
	London:           "London",
	ArrowGlacier:     "ArrowGlacier",
	GrayGlacier:      "GrayGlacier",
	Paris:            "Paris",
	Shanghai:         "Shanghai",
	Cancun:           "Cancun",
	Prague:           "Prague",
	Osaka:            "Osaka",
}

var forkFromString = make(map[string]Fork, len(forkToString))

func init() {
	for f, name := range forkToString {
		forkFromString[name] = f
	}
}

func ByName(name string) (Fork, bool) {
	f, ok := forkFromString[name]
	return f, ok
}
