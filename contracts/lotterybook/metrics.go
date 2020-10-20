// Copyright 2020 The go-ethereum Authors
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

package lotterybook

import "github.com/ethereum/go-ethereum/metrics"

var (
	chequeDBCacheHitMeter  = metrics.NewRegisteredMeter("payment/lotterybook/db/hit", nil)
	chequeDBCacheMissMeter = metrics.NewRegisteredMeter("payment/lotterybook/db/miss", nil)
	chequeDBReadMeter      = metrics.NewRegisteredMeter("payment/lotterybook/db/read", nil)
	chequeDBWriteMeter     = metrics.NewRegisteredMeter("payment/lotterybook/db/write", nil)

	staleChequeMeter   = metrics.NewRegisteredMeter("payment/lotterybook/receiver/stale", nil)
	invalidChequeMeter = metrics.NewRegisteredMeter("payment/lotterybook/receiver/invalid", nil)

	claimDurationTimer   = metrics.NewRegisteredTimer("payment/lotterybook/duration/claim", nil)
	depositDurationTimer = metrics.NewRegisteredTimer("payment/lotterybook/duration/deposit", nil)
	destroyDurationTimer = metrics.NewRegisteredTimer("payment/lotterybook/duration/destroy", nil)

	createLotteryGauge = metrics.NewRegisteredGauge("payment/lotterybook/lottery/create", nil)
	winLotteryGauge    = metrics.NewRegisteredGauge("payment/lotterybook/lottery/win", nil)
	loseLotteryGauge   = metrics.NewRegisteredGauge("payment/lotterybook/lottery/lose", nil)
	reownLotteryGauge  = metrics.NewRegisteredGauge("payment/lotterybook/lottery/reown", nil)
)
