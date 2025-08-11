package trie

import (
	"sync"

	"github.com/ethereum/go-ethereum/metrics"
)

var (
	avgAccessDepthInBlock = metrics.NewRegisteredMeter("trie/access/depth/avg", nil)
	minAccessDepthInBlock = metrics.NewRegisteredMeter("trie/access/depth/min", nil)
	stateDepthAggregator  = &depthAggregator{}
)

// depthAggregator aggregates trie access depth metrics for a block.
type depthAggregator struct {
	mu       sync.Mutex
	sum, cnt int64
	min      int64
}

// start initializes the aggregator for a new block.
func (d *depthAggregator) start() {
	d.mu.Lock()
	d.sum, d.cnt, d.min = 0, 0, -1
	d.mu.Unlock()
}

// record records the access depth for a trie operation.
func (d *depthAggregator) record(depth int64) {
	d.mu.Lock()
	v := depth
	d.sum += v
	d.cnt++
	if d.min < 0 || v < d.min {
		d.min = v
	}
	d.mu.Unlock()
}

// end finalizes the metrics for the current block and updates the registered metrics.
func (d *depthAggregator) end() {
	d.mu.Lock()
	sum, cnt, min := d.sum, d.cnt, d.min
	d.mu.Unlock()
	if cnt > 0 {
		avgAccessDepthInBlock.Mark(sum / cnt)
		minAccessDepthInBlock.Mark(min)
	}
}

// StateDepthMetricsStartBlock initializes the depth aggregator for a new block.
func StateDepthMetricsStartBlock() { stateDepthAggregator.start() }

// StateDepthMetricsEndBlock finalizes the depth metrics for the current block.
func StateDepthMetricsEndBlock() { stateDepthAggregator.end() }
