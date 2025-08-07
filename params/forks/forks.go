// Copyright 2025 The go-ethereum Authors
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

// Ethereum mainnet forks.
var (
	Frontier = Define(Spec{
		Name:       "Frontier",
		BlockBased: true,
	})

	Homestead = Define(Spec{
		Name:       "Homestead",
		BlockBased: true,
		Requires:   []Fork{Frontier},
	})

	DAO = Define(Spec{
		Name:       "DAO",
		ConfigName: "daoFork",
		BlockBased: true,
		Requires:   []Fork{Homestead},
	})

	TangerineWhistle = Define(Spec{
		Name:       "TangerineWhistle",
		ConfigName: "eip150",
		BlockBased: true,
		Requires:   []Fork{Homestead},
	})

	SpuriousDragon = Define(Spec{
		Name:       "SpuriousDragon",
		ConfigName: "eip155",
		BlockBased: true,
		Requires:   []Fork{TangerineWhistle},
	})

	Byzantium = Define(Spec{
		Name:       "Byzantium",
		BlockBased: true,
		Requires:   []Fork{SpuriousDragon},
	})

	Constantinople = Define(Spec{
		Name:       "Constantinople",
		BlockBased: true,
		Requires:   []Fork{Byzantium},
	})

	Petersburg = Define(Spec{
		Name:       "Petersburg",
		BlockBased: true,
		Requires:   []Fork{Constantinople},
	})

	Istanbul = Define(Spec{
		Name:       "Istanbul",
		BlockBased: true,
		Requires:   []Fork{Petersburg},
	})

	MuirGlacier = Define(Spec{
		Name:       "MuirGlacier",
		BlockBased: true,
		Requires:   []Fork{Istanbul},
	})

	Berlin = Define(Spec{
		Name:       "Berlin",
		BlockBased: true,
		Requires:   []Fork{Istanbul},
	})

	London = Define(Spec{
		Name:       "London",
		BlockBased: true,
		Requires:   []Fork{Berlin},
	})

	ArrowGlacier = Define(Spec{
		Name:       "ArrowGlacier",
		BlockBased: true,
		Requires:   []Fork{London, MuirGlacier},
	})

	GrayGlacier = Define(Spec{
		Name:       "GrayGlacier",
		BlockBased: true,
		Requires:   []Fork{London, ArrowGlacier},
	})

	Paris = Define(Spec{
		Name:       "Paris",
		ConfigName: "mergeNetsplit",
		BlockBased: true,
		Requires:   []Fork{London},
	})

	Shanghai = Define(Spec{
		Name:     "Shanghai",
		Requires: []Fork{Paris},
	})

	Cancun = Define(Spec{
		Name:     "Cancun",
		Requires: []Fork{Shanghai},
	})

	Prague = Define(Spec{
		Name:     "Prague",
		Requires: []Fork{Cancun},
	})

	Osaka = Define(Spec{
		Name:     "Osaka",
		Requires: []Fork{Prague},
	})
)

// Verkle forks.
var (
	Verkle = Define(Spec{
		Name:     "Verkle",
		Requires: []Fork{Prague},
	})
)

// BPOs - 'blob parameter only' forks.
var (
	BPO1 = Define(Spec{
		Name:       "BPO1",
		ConfigName: "bpo1",
		Requires:   []Fork{Osaka},
	})
	BPO2 = Define(Spec{
		Name:       "BPO2",
		ConfigName: "bpo2",
		Requires:   []Fork{BPO1},
	})
	BPO3 = Define(Spec{
		Name:       "BPO3",
		ConfigName: "bpo3",
		Requires:   []Fork{BPO2},
	})
	BPO4 = Define(Spec{
		Name:       "BPO4",
		ConfigName: "bpo4",
		Requires:   []Fork{BPO3},
	})
	BPO5 = Define(Spec{
		Name:       "BPO5",
		ConfigName: "bpo5",
		Requires:   []Fork{BPO4},
	})
)
