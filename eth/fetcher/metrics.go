// Copyright 2025 The go-ethereum Authors
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
// along with the go-ethereum library. If not, see <http://www.gnu.org/licenses/

// Contains the metrics collected by the txfetcher.

package fetcher

import "github.com/ethereum/go-ethereum/metrics"

var (
	txAnnounceInMeter          = metrics.NewRegisteredMeter("eth/fetcher/transaction/announces/in", nil)
	txAnnounceKnownMeter       = metrics.NewRegisteredMeter("eth/fetcher/transaction/announces/known", nil)
	txAnnounceUnderpricedMeter = metrics.NewRegisteredMeter("eth/fetcher/transaction/announces/underpriced", nil)
	txAnnounceDOSMeter         = metrics.NewRegisteredMeter("eth/fetcher/transaction/announces/dos", nil)

	txBroadcastInMeter          = metrics.NewRegisteredMeter("eth/fetcher/transaction/broadcasts/in", nil)
	txBroadcastKnownMeter       = metrics.NewRegisteredMeter("eth/fetcher/transaction/broadcasts/known", nil)
	txBroadcastUnderpricedMeter = metrics.NewRegisteredMeter("eth/fetcher/transaction/broadcasts/underpriced", nil)
	txBroadcastOtherRejectMeter = metrics.NewRegisteredMeter("eth/fetcher/transaction/broadcasts/otherreject", nil)

	txRequestOutMeter     = metrics.NewRegisteredMeter("eth/fetcher/transaction/request/out", nil)
	txRequestFailMeter    = metrics.NewRegisteredMeter("eth/fetcher/transaction/request/fail", nil)
	txRequestDoneMeter    = metrics.NewRegisteredMeter("eth/fetcher/transaction/request/done", nil)
	txRequestTimeoutMeter = metrics.NewRegisteredMeter("eth/fetcher/transaction/request/timeout", nil)

	txReplyInMeter          = metrics.NewRegisteredMeter("eth/fetcher/transaction/replies/in", nil)
	txReplyKnownMeter       = metrics.NewRegisteredMeter("eth/fetcher/transaction/replies/known", nil)
	txReplyUnderpricedMeter = metrics.NewRegisteredMeter("eth/fetcher/transaction/replies/underpriced", nil)
	txReplyOtherRejectMeter = metrics.NewRegisteredMeter("eth/fetcher/transaction/replies/otherreject", nil)

	txFetcherWaitingPeers   = metrics.NewRegisteredGauge("eth/fetcher/transaction/waiting/peers", nil)
	txFetcherWaitingHashes  = metrics.NewRegisteredGauge("eth/fetcher/transaction/waiting/hashes", nil)
	txFetcherQueueingPeers  = metrics.NewRegisteredGauge("eth/fetcher/transaction/queueing/peers", nil)
	txFetcherQueueingHashes = metrics.NewRegisteredGauge("eth/fetcher/transaction/queueing/hashes", nil)
	txFetcherFetchingPeers  = metrics.NewRegisteredGauge("eth/fetcher/transaction/fetching/peers", nil)
	txFetcherFetchingHashes = metrics.NewRegisteredGauge("eth/fetcher/transaction/fetching/hashes", nil)

	txFetcherSlowPeers = metrics.NewRegisteredGauge("eth/fetcher/transaction/slow/peers", nil)
	// Note: this metric does not mean that the fetching of a transaction
	// was blocked by a specific peer during this period, since we request
	// another peer to fetch the same transaction hash.
	// The purpose of this metric is to measure how long it takes for a slow peer
	// to become "unfrozen", either by eventually replying to the request
	// or by being dropped, measuring from the moment the request was sent.
	txFetcherSlowWait = metrics.NewRegisteredHistogram("eth/fetcher/transaction/slow/wait", nil, metrics.NewExpDecaySample(1028, 0.015))
)
