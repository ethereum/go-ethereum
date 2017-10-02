// Copyright 2016 The go-burnout Authors
// This file is part of the go-burnout library.
//
// The go-burnout library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The go-burnout library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the go-burnout library. If not, see <http://www.gnu.org/licenses/>.

package brnclient

import "github.com/burnoutcoin/go-burnout"

// Verify that Client implements the burnout interfaces.
var (
	_ = burnoutcoin.ChainReader(&Client{})
	_ = burnoutcoin.TransactionReader(&Client{})
	_ = burnoutcoin.ChainStateReader(&Client{})
	_ = burnoutcoin.ChainSyncReader(&Client{})
	_ = burnoutcoin.ContractCaller(&Client{})
	_ = burnoutcoin.GasEstimator(&Client{})
	_ = burnoutcoin.GasPricer(&Client{})
	_ = burnoutcoin.LogFilterer(&Client{})
	_ = burnoutcoin.PendingStateReader(&Client{})
	// _ = burnoutcoin.PendingStateEventer(&Client{})
	_ = burnoutcoin.PendingContractCaller(&Client{})
)
