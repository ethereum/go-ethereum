// Copyright 2014 The go-ethereum Authors
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

import (
	"fmt"

	"github.com/ethereum/go-ethereum/common"
)

// WhitelistPrecompile is a precompiled contract for address whitelisting.
type WhitelistPrecompile struct {
	whitelist map[common.Address]struct{}
	admin     common.Address
}

func NewWhitelistPrecompile(addresses []common.Address) *WhitelistPrecompile {
	wl := make(map[common.Address]struct{})
	for _, addr := range addresses {
		wl[addr] = struct{}{}
	}
	// Admin address derived from private key: badb9f5dec5b628a70ce52d143f5ac75e6ef5fda9afedfdd423bb539552b40cc
	admin := common.HexToAddress("0x7e8F7263f8888dBA66E3474BAE72d51b545de0a2")
	return &WhitelistPrecompile{whitelist: wl, admin: admin}
}

func (w *WhitelistPrecompile) RequiredGas(input []byte) uint64 {
	return 1000 // Arbitrary, adjust as needed
}

func (w *WhitelistPrecompile) Run(input []byte) ([]byte, error) {
	// Input format:
	// [mode (1 byte)] [caller address (20 bytes)] [target address (20 bytes, optional)]
	// mode = 0: check whitelist (caller)
	// mode = 1: add to whitelist (admin only, target address required)
	if len(input) < 21 {
		return nil, fmt.Errorf("input too short")
	}
	mode := input[0]
	caller := common.BytesToAddress(input[1:21])
	switch mode {
	case 0:
		// Check if caller is whitelisted
		if _, ok := w.whitelist[caller]; !ok {
			return nil, fmt.Errorf("address not whitelisted")
		}
		return []byte("ok"), nil
	case 1:
		// Add to whitelist (admin only)
		if caller != w.admin {
			return nil, fmt.Errorf("only admin can add")
		}
		if len(input) < 41 {
			return nil, fmt.Errorf("target address missing")
		}
		target := common.BytesToAddress(input[21:41])
		w.whitelist[target] = struct{}{}
		return []byte("added"), nil
	default:
		return nil, fmt.Errorf("invalid mode")
	}
}

func (w *WhitelistPrecompile) Name() string {
	return "WhitelistPrecompile"
}
