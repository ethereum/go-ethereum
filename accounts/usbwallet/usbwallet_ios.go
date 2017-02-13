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

// This file contains the implementation for interacting with the Ledger hardware
// wallets. The wire protocol spec can be found in the Ledger Blue GitHub repo:
// https://raw.githubusercontent.com/LedgerHQ/blue-app-eth/master/doc/ethapp.asc

// +build ios

package usbwallet

import (
	"errors"

	"github.com/ethereum/go-ethereum/accounts"
)

// Here be dragons! There is no USB support on iOS.

// ErrIOSNotSupported is returned for all USB hardware backends on iOS.
var ErrIOSNotSupported = errors.New("no USB support on iOS")

func NewLedgerHub() (accounts.Backend, error) {
	return nil, ErrIOSNotSupported
}
