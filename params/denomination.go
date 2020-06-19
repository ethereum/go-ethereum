// Copyright 2017 The go-ethereum Authors
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

import "math/big"

// These are the multipliers for ether denominations.
// Example: To get the wei value of an amount in 'gwei', use
//
//    new(big.Int).Mul(value, big.NewInt(params.GWei))
//
const (
	Wei   = 1
	GWei  = 1e9
	Ether = 1e18
)

// EtherToWei converts an Ether value to Wei.
func EtherToWei(val *big.Int) *big.Int {
	return new(big.Int).Mul(val, big.NewInt(Ether))
}

// WeiToEther converts an Wei value to Ether.
// It might round according to the rules for big.Int.
func WeiToEther(val *big.Int) *big.Int {
	return new(big.Int).Div(val, big.NewInt(Ether))
}

// GweiToWei converts a GWei value to Wei.
func GweiToWei(val *big.Int) *big.Int {
	return new(big.Int).Mul(val, big.NewInt(GWei))
}

// WeiToGwei converts an Wei value to GWei.
// It might round according to the rules for big.Int.
func WeiToGwei(val *big.Int) *big.Int {
	return new(big.Int).Div(val, big.NewInt(GWei))
}
