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

// Contains the metrics collected by the fetcher.

package fetcher

import (
	"github.com/ethereum/go-ethereum/metrics"
)

var (
	blockAnnounceInMeter   = metrics.NewRegisteredMeter("eth/fetcher/prop/block/announces/in", nil)
	blockAnnounceOutTimer  = metrics.NewRegisteredTimer("eth/fetcher/prop/block/announces/out", nil)
	blockAnnounceDropMeter = metrics.NewRegisteredMeter("eth/fetcher/prop/block/announces/drop", nil)
	blockAnnounceDOSMeter  = metrics.NewRegisteredMeter("eth/fetcher/prop/block/announces/dos", nil)

	blockBroadcastInMeter   = metrics.NewRegisteredMeter("eth/fetcher/prop/block/broadcasts/in", nil)
	blockBroadcastOutTimer  = metrics.NewRegisteredTimer("eth/fetcher/prop/block/broadcasts/out", nil)
	blockBroadcastDropMeter = metrics.NewRegisteredMeter("eth/fetcher/prop/block/broadcasts/drop", nil)
	blockBroadcastDOSMeter  = metrics.NewRegisteredMeter("eth/fetcher/prop/block/broadcasts/dos", nil)

	headerFetchMeter = metrics.NewRegisteredMeter("eth/fetcher/fetch/headers", nil)
	bodyFetchMeter   = metrics.NewRegisteredMeter("eth/fetcher/fetch/bodies", nil)

	headerFilterInMeter  = metrics.NewRegisteredMeter("eth/fetcher/filter/headers/in", nil)
	headerFilterOutMeter = metrics.NewRegisteredMeter("eth/fetcher/filter/headers/out", nil)
	bodyFilterInMeter    = metrics.NewRegisteredMeter("eth/fetcher/filter/bodies/in", nil)
	bodyFilterOutMeter   = metrics.NewRegisteredMeter("eth/fetcher/filter/bodies/out", nil)

	txAnnounceInMeter         = metrics.NewRegisteredMeter("eth/fetcher/prop/transaction/announces/in", nil)
	txAnnounceDOSMeter        = metrics.NewRegisteredMeter("eth/fetcher/prop/transaction/announces/dos", nil)
	txAnnounceSkipMeter       = metrics.NewRegisteredMeter("eth/fetcher/prop/transaction/announces/skip", nil)
	txAnnounceUnderpriceMeter = metrics.NewRegisteredMeter("eth/fetcher/prop/transaction/announces/underprice", nil)
	txBroadcastInMeter        = metrics.NewRegisteredMeter("eth/fetcher/prop/transaction/broadcasts/in", nil)
	txFetchOutMeter           = metrics.NewRegisteredMeter("eth/fetcher/fetch/transaction/out", nil)
	txFetchSuccessMeter       = metrics.NewRegisteredMeter("eth/fetcher/fetch/transaction/success", nil)
	txFetchTimeoutMeter       = metrics.NewRegisteredMeter("eth/fetcher/fetch/transaction/timeout", nil)
	txFetchInvalidMeter       = metrics.NewRegisteredMeter("eth/fetcher/fetch/transaction/invalid", nil)
	txFetchDurationTimer      = metrics.NewRegisteredTimer("eth/fetcher/fetch/transaction/duration", nil)
)
