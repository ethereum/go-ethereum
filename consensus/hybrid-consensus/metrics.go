package hybridconsensus

import (
    "time"
)

type Metrics struct {
    BlockCreationTime time.Duration
    TotalBlocks       uint64
}

func (hc *HybridConsensus) RecordBlockCreationTime(start time.Time) {
    hc.metrics.BlockCreationTime = time.Since(start)
    hc.metrics.TotalBlocks++
}

