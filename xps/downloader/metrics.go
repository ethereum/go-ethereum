// Copyright 2015 The go-ethereum Authors
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

// Copyright 2021-2022 The go-xpayments Authors
// This file is part of go-xpayments.

// Contains the metrics collected by the downloader.

package downloader

import (
	"github.com/xpaymentsorg/go-xpayments/metrics"
	// "github.com/ethereum/go-ethereum/metrics"
)

var (
	headerInMeter      = metrics.NewRegisteredMeter("xps/downloader/headers/in", nil)
	headerReqTimer     = metrics.NewRegisteredTimer("xps/downloader/headers/req", nil)
	headerDropMeter    = metrics.NewRegisteredMeter("xps/downloader/headers/drop", nil)
	headerTimeoutMeter = metrics.NewRegisteredMeter("xps/downloader/headers/timeout", nil)

	bodyInMeter      = metrics.NewRegisteredMeter("xps/downloader/bodies/in", nil)
	bodyReqTimer     = metrics.NewRegisteredTimer("xps/downloader/bodies/req", nil)
	bodyDropMeter    = metrics.NewRegisteredMeter("xps/downloader/bodies/drop", nil)
	bodyTimeoutMeter = metrics.NewRegisteredMeter("xps/downloader/bodies/timeout", nil)

	receiptInMeter      = metrics.NewRegisteredMeter("xps/downloader/receipts/in", nil)
	receiptReqTimer     = metrics.NewRegisteredTimer("xps/downloader/receipts/req", nil)
	receiptDropMeter    = metrics.NewRegisteredMeter("xps/downloader/receipts/drop", nil)
	receiptTimeoutMeter = metrics.NewRegisteredMeter("xps/downloader/receipts/timeout", nil)

	throttleCounter = metrics.NewRegisteredCounter("xps/downloader/throttle", nil)
)
