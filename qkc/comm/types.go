// Copyright 2026-2027 The quarkchain Authors
// This file is part of the quarkchain library.
//
// The quarkchain library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The quarkchain library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the quarkchain library. If not, see <http://www.gnu.org/licenses/>.

// Package comm contains business types shared by business interfaces and
// communication layers across the system.
//
// Interface methods should return a single result value directly, followed by
// an error. When a method needs to return multiple result values, define a
// result struct here and return it with an error. When a method's parameters
// are complex, define a parameter struct here as well. These structs keep
// business data grouped consistently without exposing wire message
// representations.
package comm

import (
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/holiman/uint256"
)

// SlaveInfo identifies a configured slave endpoint without wire encoding.
type SlaveInfo struct {
	Host    string
	Port    uint16
	ChainID *big.Int
}

// RootChainStakesResult groups the staking amount and signer returned for a
// chain and block query. It is the shared business result passed between the
// communication layer and business interfaces; it does not contain wire or
// SSZ-specific representations.
type RootChainStakesResult struct {
	Stakes *uint256.Int
	Signer common.Address
}
