// Copyright 2015 The go-burnout Authors
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

// Contains the metrics collected by the downloader.

package downloader

import (
	"github.com/burnoutcoin/go-burnout/metrics"
)

var (
	headerInMeter      = metrics.NewMeter("brn/downloader/headers/in")
	headerReqTimer     = metrics.NewTimer("brn/downloader/headers/req")
	headerDropMeter    = metrics.NewMeter("brn/downloader/headers/drop")
	headerTimeoutMeter = metrics.NewMeter("brn/downloader/headers/timeout")

	bodyInMeter      = metrics.NewMeter("brn/downloader/bodies/in")
	bodyReqTimer     = metrics.NewTimer("brn/downloader/bodies/req")
	bodyDropMeter    = metrics.NewMeter("brn/downloader/bodies/drop")
	bodyTimeoutMeter = metrics.NewMeter("brn/downloader/bodies/timeout")

	receiptInMeter      = metrics.NewMeter("brn/downloader/receipts/in")
	receiptReqTimer     = metrics.NewTimer("brn/downloader/receipts/req")
	receiptDropMeter    = metrics.NewMeter("brn/downloader/receipts/drop")
	receiptTimeoutMeter = metrics.NewMeter("brn/downloader/receipts/timeout")

	stateInMeter   = metrics.NewMeter("brn/downloader/states/in")
	stateDropMeter = metrics.NewMeter("brn/downloader/states/drop")
)
