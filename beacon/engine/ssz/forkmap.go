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

package ssz

import (
	"github.com/ethereum/go-ethereum/params/forks"
	"github.com/karalabe/ssz"
)

// karalabe/ssz's Fork enum currently stops at Pectra (Prague). The monolith
// types below gate fields with ForkFilter{Added,Removed}, and the codec only
// performs ordered comparisons (`fork < Added`, `fork >= Removed`) against
// Codec.fork. Any strictly-monotonic values placed after ForkPectra therefore
// behave correctly. We define local aliases for the forks the library doesn't
// name yet.
//
// IMPORTANT: these are only meaningful relative to one another and to the
// library's own enum; never persist them. If a future karalabe/ssz release
// adds Osaka/Amsterdam (or new forks between Pectra and these), revisit this
// table so the ordering stays gap-consistent with the upstream enum.
const (
	forkOsaka     = ssz.ForkPectra + 1
	forkAmsterdam = ssz.ForkPectra + 2
)

// ForkFor maps a geth params/forks.Fork onto the karalabe/ssz Fork value
// the codec multiplexes on. The bool is false for forks that predate the
// Engine API (and thus have no SSZ wire representation here).
func ForkFor(f forks.Fork) (ssz.Fork, bool) {
	switch f {
	case forks.Paris:
		return ssz.ForkParis, true
	case forks.Shanghai:
		return ssz.ForkShapella, true // EL Shanghai == CL Shapella
	case forks.Cancun:
		return ssz.ForkDencun, true // EL Cancun == CL Dencun
	case forks.Prague:
		return ssz.ForkPectra, true // EL Prague == CL Pectra
	case forks.Osaka:
		return forkOsaka, true
	case forks.Amsterdam:
		return forkAmsterdam, true
	default:
		return ssz.ForkUnknown, false
	}
}
