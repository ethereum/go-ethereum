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

// Package usbwallet implements support for USB hardware wallets.
package usbwallet

import "time"

// deviceID is a combined vendor/product identifier to uniquely identify a USB
// hardware device.
type deviceID struct {
	Vendor  uint16 // The Vendor identifer
	Product uint16 // The Product identifier
}

// Maximum time between wallet health checks to detect USB unplugs.
const heartbeatCycle = time.Second

// Minimum time to wait between self derivation attempts, even it the user is
// requesting accounts like crazy.
const selfDeriveThrottling = time.Second
