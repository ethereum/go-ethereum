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

// Contains the metrics collected by the downloader.

package downloader

import (
	"github.com/ethereum/go-ethereum/metrics"
)

var (
	headerInMeter      = metrics.NewRegisteredMeter("eth/downloader/headers/in", nil)
	headerReqTimer     = metrics.NewRegisteredTimer("eth/downloader/headers/req", nil)
	headerTimeoutMeter = metrics.NewRegisteredMeter("eth/downloader/headers/timeout", nil)

	bodyInMeter      = metrics.NewRegisteredMeter("eth/downloader/bodies/in", nil)
	bodyReqTimer     = metrics.NewRegisteredTimer("eth/downloader/bodies/req", nil)
	bodyDropMeter    = metrics.NewRegisteredMeter("eth/downloader/bodies/drop", nil)
	bodyTimeoutMeter = metrics.NewRegisteredMeter("eth/downloader/bodies/timeout", nil)

	receiptInMeter      = metrics.NewRegisteredMeter("eth/downloader/receipts/in", nil)
	receiptReqTimer     = metrics.NewRegisteredTimer("eth/downloader/receipts/req", nil)
	receiptDropMeter    = metrics.NewRegisteredMeter("eth/downloader/receipts/drop", nil)
	receiptTimeoutMeter = metrics.NewRegisteredMeter("eth/downloader/receipts/timeout", nil)

	balInMeter      = metrics.NewRegisteredMeter("eth/downloader/bals/in", nil)
	balReqTimer     = metrics.NewRegisteredTimer("eth/downloader/bals/req", nil)
	balDropMeter    = metrics.NewRegisteredMeter("eth/downloader/bals/drop", nil)
	balTimeoutMeter = metrics.NewRegisteredMeter("eth/downloader/bals/timeout", nil)

	bodyFetchMetrics    = newFetchMetrics("bodies")
	receiptFetchMetrics = newFetchMetrics("receipts")
	balFetchMetrics     = newFetchMetrics("bals")

	// rttTargetGauge is the round trip time (in milliseconds) requests are
	// currently sized for, derived from the median of the peer estimates.
	rttTargetGauge = metrics.NewRegisteredGauge("eth/downloader/rtt/target", nil)

	importWaitTimer           = metrics.NewRegisteredTimer("eth/downloader/import/wait", nil)
	importInsertBlocksTimer   = metrics.NewRegisteredTimer("eth/downloader/import/blocks", nil)
	importInsertReceiptsTimer = metrics.NewRegisteredTimer("eth/downloader/import/receipts", nil)
	importBatchHistogram      = metrics.NewRegisteredHistogram("eth/downloader/import/batch", nil, metrics.NewExpDecaySample(1028, 0.015))

	// Result cache metrics, reported every time a batch is drained.
	queueThrottleGauge = metrics.NewRegisteredGauge("eth/downloader/queue/throttle/threshold", nil)
	queueItemSizeGauge = metrics.NewRegisteredGauge("eth/downloader/queue/itemsize", nil)

	// snapPeerSkipMeter tracks snap peers skipped by the state syncer because
	// they negotiated a version below the one the syncer requires.
	snapPeerSkipMeter = metrics.NewRegisteredMeter("eth/downloader/snap/peerskip", nil)
)

// fetchMetrics groups the collectors the concurrent fetcher reports into for a
// single data type (bodies, receipts, access lists).
type fetchMetrics struct {
	idlePeers  *metrics.Gauge    // Peers left without a request after an assignment round
	busyPeers  *metrics.Gauge    // Peers with a request in flight
	stalePeers *metrics.Gauge    // Peers with a timed out but not yet answered request
	capacity   *metrics.Gauge    // Estimated aggregate items per second across all peers
	starved    *metrics.Meter    // Assignment rounds cut short because nothing was pending
	throttled  *metrics.Meter    // Assignment rounds cut short by result cache throttling
	items      metrics.Histogram // Items contained in each response
	bytes      *metrics.Meter    // Payload bytes contained in each response
}

// newFetchMetrics registers the scheduling collectors for a data type under
// eth/downloader/<kind>/...
func newFetchMetrics(kind string) *fetchMetrics {
	prefix := "eth/downloader/" + kind
	return &fetchMetrics{
		idlePeers:  metrics.NewRegisteredGauge(prefix+"/peers/idle", nil),
		busyPeers:  metrics.NewRegisteredGauge(prefix+"/peers/busy", nil),
		stalePeers: metrics.NewRegisteredGauge(prefix+"/peers/stale", nil),
		capacity:   metrics.NewRegisteredGauge(prefix+"/capacity", nil),
		starved:    metrics.NewRegisteredMeter(prefix+"/starved", nil),
		throttled:  metrics.NewRegisteredMeter(prefix+"/throttled", nil),
		items:      metrics.NewRegisteredHistogram(prefix+"/items", nil, metrics.NewExpDecaySample(1028, 0.015)),
		bytes:      metrics.NewRegisteredMeter(prefix+"/bytes", nil),
	}
}
