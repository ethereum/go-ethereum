// Copyright 2026 The go-ethereum Authors
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

package core

import (
	"time"

	"github.com/ethereum/go-ethereum/core/history"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/params"
)

// pruneChainHistory prunes block bodies, receipts, and transaction index entries
// below the given target block. It is the single shared implementation used by
// both startup pruning and the rolling history pruner.
func (bc *BlockChain) pruneChainHistory(target uint64) error {
	tail, err := bc.db.Tail(rawdb.ChainFreezerBlockDataGroup)
	if err != nil {
		return err
	}
	if tail >= target {
		return nil
	}
	rawdb.PruneTransactionIndex(bc.db, target)
	if _, err := bc.db.TruncateTail(rawdb.ChainFreezerBlockDataGroup, target); err != nil {
		return err
	}
	bc.updateHistoryPrunePoint(target)
	log.Debug("Pruned chain history", "from", tail, "to", target)
	return nil
}

// updateHistoryPrunePoint updates the atomic prune point on the blockchain.
func (bc *BlockChain) updateHistoryPrunePoint(blockNumber uint64) {
	hash := bc.GetCanonicalHash(blockNumber)
	bc.historyPrunePoint.Store(&history.PrunePoint{
		BlockNumber: blockNumber,
		BlockHash:   hash,
	})
}

// historyPruner continuously prunes old block bodies and receipts, maintaining
// a rolling window of recent blocks.
type historyPruner struct {
	historyBlocks uint64
	chain         *BlockChain
	term          chan chan struct{}
	closed        chan struct{}
}

// newHistoryPruner creates a new history pruner and starts its background loop.
func newHistoryPruner(historyBlocks uint64, chain *BlockChain) *historyPruner {
	pruner := &historyPruner{
		historyBlocks: historyBlocks,
		chain:         chain,
		term:          make(chan chan struct{}),
		closed:        make(chan struct{}),
	}
	go pruner.loop()
	log.Info("Initialized rolling history pruner", "window", historyBlocks)
	return pruner
}

// loop is the main background goroutine that periodically checks if pruning is needed.
func (p *historyPruner) loop() {
	defer close(p.closed)

	// Fire immediately on first run
	timer := time.NewTimer(0)
	defer timer.Stop()

	for {
		select {
		case <-timer.C:
			p.prune()
			timer.Reset(3 * time.Hour)

		case ch := <-p.term:
			close(ch)
			return
		}
	}
}

// prune performs a single round of pruning if needed.
func (p *historyPruner) prune() {
	head := p.chain.CurrentBlock()
	if head == nil {
		return
	}
	headNum := head.Number.Uint64()
	if headNum <= p.historyBlocks {
		return
	}
	target := headNum - p.historyBlocks

	// Sanity check that target has been frozen.
	frozen := headNum - params.FullImmutabilityThreshold
	if target > frozen {
		log.Error("Rolling pruner target exceeds frozen range", "target", target, "frozen", frozen, "head", headNum, "window", p.historyBlocks)
		return
	}
	if err := p.chain.pruneChainHistory(target); err != nil {
		log.Error("Failed to prune chain history", "err", err, "target", target)
	}
}

// close signals the pruner to stop and waits for it to exit.
func (p *historyPruner) close() {
	ch := make(chan struct{})
	select {
	case p.term <- ch:
		<-ch
	case <-p.closed:
	}
}
