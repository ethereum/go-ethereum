// Contains the metrics collected by the fetcher.

package fetcher

import (
	"github.com/ethereum/go-ethereum/metrics"
)

var (
	announceMeter  = metrics.NewMeter("eth/sync/RemoteAnnounces")
	announceTimer  = metrics.NewTimer("eth/sync/LocalAnnounces")
	broadcastMeter = metrics.NewMeter("eth/sync/RemoteBroadcasts")
	broadcastTimer = metrics.NewTimer("eth/sync/LocalBroadcasts")
	discardMeter   = metrics.NewMeter("eth/sync/DiscardedBlocks")
	futureMeter    = metrics.NewMeter("eth/sync/FutureBlocks")
)
