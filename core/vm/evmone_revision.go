//go:build evmone

package vm

import "github.com/ethereum/go-ethereum/params"

// EVMC revision constants matching evmc/evmc.h enum evmc_revision.
const (
	evmcFrontier             int32 = 0
	evmcHomestead            int32 = 1
	evmcTangerineWhistle     int32 = 2
	evmcSpuriousDragon       int32 = 3
	evmcByzantium            int32 = 4
	evmcConstantinople       int32 = 5
	evmcPetersburg           int32 = 6
	evmcIstanbul             int32 = 7
	evmcBerlin               int32 = 8
	evmcLondon               int32 = 9
	evmcParis                int32 = 10
	evmcShanghai             int32 = 11
	evmcCancun               int32 = 12
	evmcPrague               int32 = 13
	evmcOsaka                int32 = 14
)

// evmcRevision maps go-ethereum chain rules to the corresponding EVMC revision.
func evmcRevision(rules params.Rules) int32 {
	switch {
	case rules.IsOsaka:
		return evmcOsaka
	case rules.IsPrague:
		return evmcPrague
	case rules.IsCancun:
		return evmcCancun
	case rules.IsShanghai:
		return evmcShanghai
	case rules.IsMerge:
		return evmcParis
	case rules.IsLondon:
		return evmcLondon
	case rules.IsBerlin:
		return evmcBerlin
	case rules.IsIstanbul:
		return evmcIstanbul
	case rules.IsConstantinople:
		return evmcConstantinople
	case rules.IsByzantium:
		return evmcByzantium
	case rules.IsEIP158:
		return evmcSpuriousDragon
	case rules.IsEIP150:
		return evmcTangerineWhistle
	case rules.IsHomestead:
		return evmcHomestead
	default:
		return evmcFrontier
	}
}
