// Copyright 2014 The go-ethereum Authors
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

// Package core implements the Ethereum consensus protocol.
package core

import (
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/mclock"
	"github.com/ethereum/go-ethereum/consensus"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/stateless"
	"github.com/ethereum/go-ethereum/core/tracing"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/log"
)

var (
	PrefetchBALTime   = time.Duration(0)
	PostMergeTime     = time.Duration(0)
	PrefetchTrieTimer = time.Duration(0)
	PrefetchChTime    = time.Duration(0)
)

func (bc *BlockChain) insertChainN(chain types.Blocks, setHead bool, makeWitness bool) (*stateless.Witness, int, error) {
	// If the chain is terminating, don't even bother starting up.
	if bc.insertStopped() {
		return nil, 0, nil
	}

	if atomic.AddInt32(&bc.blockProcCounter, 1) == 1 {
		bc.blockProcFeed.Send(true)
	}
	defer func() {
		if atomic.AddInt32(&bc.blockProcCounter, -1) == 0 {
			bc.blockProcFeed.Send(false)
		}
	}()

	// Start a parallel signature recovery (signer will fluke on fork transition, minimal perf loss)
	SenderCacher().RecoverFromBlocks(types.MakeSigner(bc.chainConfig, chain[0].Number(), chain[0].Time()), chain)

	var (
		stats     = insertStats{startTime: mclock.Now()}
		lastCanon *types.Block
	)
	// Fire a single chain head event if we've progressed the chain
	defer func() {
		if lastCanon != nil && bc.CurrentBlock().Hash() == lastCanon.Hash() {
			bc.chainHeadFeed.Send(ChainHeadEvent{Header: lastCanon.Header()})
		}
	}()
	// Start the parallel header verifier
	headers := make([]*types.Header, len(chain))
	for i, block := range chain {
		headers[i] = block.Header()
	}
	abort, results := bc.engine.VerifyHeaders(bc, headers)
	defer close(abort)

	// Peek the error for the first block to decide the directing import logic
	it := newInsertIterator(chain, results, bc.validator)
	block, err := it.next()

	// Left-trim all the known blocks that don't need to build snapshot
	if bc.skipBlock(err, it) {
		// First block (and state) is known
		//   1. We did a roll-back, and should now do a re-import
		//   2. The block is stored as a sidechain, and is lying about it's stateroot, and passes a stateroot
		//      from the canonical chain, which has not been verified.
		// Skip all known blocks that are behind us.
		current := bc.CurrentBlock()
		for block != nil && bc.skipBlock(err, it) {
			if block.NumberU64() > current.Number.Uint64() || bc.GetCanonicalHash(block.NumberU64()) != block.Hash() {
				break
			}
			log.Debug("Ignoring already known block", "number", block.Number(), "hash", block.Hash())
			stats.ignored++

			block, err = it.next()
		}
		// The remaining blocks are still known blocks, the only scenario here is:
		// During the snap sync, the pivot point is already submitted but rollback
		// happens. Then node resets the head full block to a lower height via `rollback`
		// and leaves a few known blocks in the database.
		//
		// When node runs a snap sync again, it can re-import a batch of known blocks via
		// `insertChain` while a part of them have higher total difficulty than current
		// head full block(new pivot point).
		for block != nil && bc.skipBlock(err, it) {
			log.Debug("Writing previously known block", "number", block.Number(), "hash", block.Hash())
			if err := bc.writeKnownBlock(block); err != nil {
				return nil, it.index, err
			}
			lastCanon = block

			block, err = it.next()
		}
		// Falls through to the block import
	}
	switch {
	// First block is pruned
	case errors.Is(err, consensus.ErrPrunedAncestor):
		if setHead {
			// First block is pruned, insert as sidechain and reorg only if TD grows enough
			log.Debug("Pruned ancestor, inserting as sidechain", "number", block.Number(), "hash", block.Hash())
			return bc.insertSideChain(block, it, makeWitness)
		} else {
			// We're post-merge and the parent is pruned, try to recover the parent state
			log.Debug("Pruned ancestor", "number", block.Number(), "hash", block.Hash())
			_, err := bc.recoverAncestors(block, makeWitness)
			return nil, it.index, err
		}
	// Some other error(except ErrKnownBlock) occurred, abort.
	// ErrKnownBlock is allowed here since some known blocks
	// still need re-execution to generate snapshots that are missing
	case err != nil && !errors.Is(err, ErrKnownBlock):
		stats.ignored += len(it.chain)
		bc.reportBlock(block, nil, err)
		return nil, it.index, err
	}
	// Track the singleton witness from this chain insertion (if any)
	var (
		witness      *stateless.Witness
		wg           sync.WaitGroup
		allReceipts  types.Receipts
		allLogs      []*types.Log
		totalGasUsed uint64
		headerTime   time.Duration
	)

	// All blocks share the same stateDB to simulate commiting after processing multiple blocks
	startBlock := block
	endBlock := chain[len(chain)-1]
	parent := it.previous()
	if parent == nil {
		parent = bc.GetHeader(startBlock.ParentHash(), startBlock.NumberU64()-1)
	}
	statedb, err := state.New(parent.Root, bc.statedb)
	if err != nil {
		log.Crit("failed to initailzied state", "error", err, "root:", parent.Root.Hex())
	}
	// Prefetch pre-N-blocks state with merged pre-block BALs for N-blocks
	// Here pre-block BALs are not merged, but we skipped allready fetch state to simulate merged operations.
	prefetchStart := time.Now()
	for _, blk := range chain {
		statedb.PrefetchStateBAL(blk.NumberU64())
	}
	PrefetchBALTime += time.Since(prefetchStart)

	// pre-block state for block N
	prestateCh := make(chan *state.StateDB, len(chain))

	wg.Add(2)
	// process post-N-blocks state with merged post-BALs for N-blocks
	go func() {
		defer wg.Done()

		mstart := time.Now()
		// Stop prefetcher cause we'll directly fetch tries in parallel
		statedb.StopPrefetcher()

		for _, blk := range chain {
			// We don't need to worry about blockNumber is not changed, because it'll be set during process block
			prestateCh <- statedb.CopyState()
			statedb.MergePostBalStates(blk.NumberU64())
		}
		PostMergeTime += time.Since(mstart)

		// Prewarm the trie for future committing
		pstart := time.Now()
		statedb.PrefetchTrie()
		PrefetchTrieTimer += time.Since(pstart)
	}()

	go func() {
		defer wg.Done()

		hstart := time.Now()
		// Write all headers then parallel executing to validate post-tx BALs
		bc.writeNBlockHeaders(chain)
		headerTime = time.Since(hstart)

		for ; block != nil && err == nil || errors.Is(err, ErrKnownBlock); block, err = it.next() {
			// If the chain is terminating, stop processing blocks
			if bc.insertStopped() {
				log.Debug("Abort during block processing")
				break
			}
			// If the block is known (in the middle of the chain), it's a special case for
			// Clique blocks where they can share state among each other, so importing an
			// older block might complete the state of the subsequent one. In this case,
			// just skip the block (we already validated it once fully (and crashed), since
			// its header and body was already in the database). But if the corresponding
			// snapshot layer is missing, forcibly rerun the execution to build it.
			if bc.skipBlock(err, it) {
				logger := log.Debug
				if bc.chainConfig.Clique == nil {
					logger = log.Warn
				}
				logger("Inserted known block", "number", block.Number(), "hash", block.Hash(),
					"uncles", len(block.Uncles()), "txs", len(block.Transactions()), "gas", block.GasUsed(),
					"root", block.Root())

				// Special case. Commit the empty receipt slice if we meet the known
				// block in the middle. It can only happen in the clique chain. Whenever
				// we insert blocks via `insertSideChain`, we only commit `td`, `header`
				// and `body` if it's non-existent. Since we don't have receipts without
				// reexecution, so nothing to commit. But if the sidechain will be adopted
				// as the canonical chain eventually, it needs to be reexecuted for missing
				// state, but if it's this special case here(skip reexecution) we will lose
				// the empty receipt entry.
				if len(block.Transactions()) == 0 {
					rawdb.WriteReceipts(bc.db, block.Hash(), block.NumberU64(), nil)
				} else {
					log.Error("Please file an issue, skip known block execution without receipt",
						"hash", block.Hash(), "number", block.NumberU64())
				}
				if err := bc.writeKnownBlock(block); err != nil {
					witness = nil
					return
				}
				stats.processed++
				if bc.logger != nil && bc.logger.OnSkippedBlock != nil {
					bc.logger.OnSkippedBlock(tracing.BlockEvent{
						Block:     block,
						Finalized: bc.CurrentFinalBlock(),
						Safe:      bc.CurrentSafeBlock(),
					})
				}
				// We can assume that logs are empty here, since the only way for consecutive
				// Clique blocks to have the same state is if there are no transactions.
				lastCanon = block
				continue
			}
			// Retrieve the parent block and it's state to execute on top
			parent := it.previous()
			if parent == nil {
				parent = bc.GetHeader(block.ParentHash(), block.NumberU64()-1)
			}
			// The traced section of block import.
			pstart := time.Now()
			statedbForBlock := <-prestateCh
			PrefetchChTime += time.Since(pstart)

			res, err := bc.processBlockWithState(parent.Root, block, setHead, makeWitness && len(chain) == 1, statedbForBlock)
			if err != nil {
				witness = nil
				log.Crit("Failed to processBlock", "error", err)
				return
			}

			// collect logs and receipts for N-blocks
			allReceipts = append(allReceipts, res.receipts...)
			allLogs = append(allLogs, res.logs...)
			totalGasUsed += res.usedGas
			// Report the import stats before returning the various results
			stats.processed++
			stats.usedGas += res.usedGas
			witness = res.witness

			var snapDiffItems, snapBufItems common.StorageSize
			if bc.snaps != nil {
				snapDiffItems, snapBufItems = bc.snaps.Size()
			}
			trieDiffNodes, trieBufNodes, _ := bc.triedb.Size()
			stats.report(chain, it.index, snapDiffItems, snapBufItems, trieDiffNodes, trieBufNodes, setHead)

			// Print confirmation that a future fork is scheduled, but not yet active.
			bc.logForkReadiness(block)

			if !setHead {
				// After merge we expect few side chains. Simply count
				// all blocks the CL gives us for GC processing time
				bc.gcproc += res.procTime
				witness = nil
				return // Direct block insertion of a single block
			}
		}

		stats.ignored += it.remaining()
	}()
	wg.Wait()

	// Validate stateRoot for N-blocks at once.
	xvtime := time.Now()
	header := endBlock.Header()
	if root := statedb.IntermediateRoot(true); header.Root != root {
		log.Crit("invalid merkle root (remote: %x local: %x) dberr: %w", header.Root, root, statedb.Error())
	}
	blockCrossValidationTimer.Update(time.Since(xvtime))

	// Commit N-blocks together to the chain and get the status.
	var (
		wstart = time.Now()
		status WriteStatus
	)
	if !setHead {
		// Don't set the head, only insert the block
		err = bc.writeNBlocksWithState(startBlock, endBlock, allReceipts, statedb)
	} else {
		status, err = bc.writeNBlocksAndSetHead(startBlock, endBlock, allReceipts, allLogs, statedb, false)
	}
	if err != nil {
		return nil, it.index, err
	}

	// Update the metrics touched during N-blocks commit
	accountCommitTimer.Update(statedb.AccountCommits)   // Account commits are complete, we can mark them
	storageCommitTimer.Update(statedb.StorageCommits)   // Storage commits are complete, we can mark them
	snapshotCommitTimer.Update(statedb.SnapshotCommits) // Snapshot commits are complete, we can mark them
	triedbCommitTimer.Update(statedb.TrieDBCommits)     // Trie database commits are complete, we can mark them

	blockWriteTimer.Update(time.Since(wstart) - max(statedb.AccountCommits, statedb.StorageCommits) /* concurrent */ - statedb.SnapshotCommits - statedb.TrieDBCommits + headerTime)

	switch status {
	case CanonStatTy:
		log.Debug("Inserted new blocks", "numberStart", startBlock.Number(), "numberEnd", endBlock.Number(),
			"elapsed", common.PrettyDuration(time.Since(prefetchStart)),
			"root", endBlock.Root())

		lastCanon = endBlock

	default:
		// This in theory is impossible, but lets be nice to our future selves and leave
		// a log, instead of trying to track down blocks imports that don't emit logs.
		log.Warn("Inserted block with unknown status", "number", endBlock.Number(), "hash", endBlock.Hash(),
			"diff", endBlock.Difficulty(), "elapsed", common.PrettyDuration(time.Since(prefetchStart)),
			"txs", len(endBlock.Transactions()), "gas", endBlock.GasUsed(), "uncles", len(endBlock.Uncles()),
			"root", endBlock.Root())
	}

	elapsed := time.Since(prefetchStart) + 1 // prevent zero division
	blockInsertTimer.Update(elapsed)

	// TODO(rjl493456442) generalize the ResettingTimer
	mgasps := float64(totalGasUsed) * 1000 / float64(elapsed)
	chainMgaspsMeter.Update(time.Duration(mgasps))

	return witness, it.index, err
}

func (bc *BlockChain) writeNBlocksWithState(startBlock, endBlock *types.Block, receipts []*types.Receipt, statedb *state.StateDB) error {
	if !bc.HasHeader(startBlock.ParentHash(), startBlock.NumberU64()-1) {
		return consensus.ErrUnknownAncestor
	}
	// Irrelevant of the canonical status, write the block itself to the database.
	//
	// Note all the components of block(hash->number map, header, body, receipts)
	// should be written atomically. BlockBatch is used for containing all components.
	blockBatch := bc.db.NewBatch()
	rawdb.WriteBlock(blockBatch, endBlock)
	rawdb.WriteReceipts(blockBatch, endBlock.Hash(), endBlock.NumberU64(), receipts)
	rawdb.WritePreimages(blockBatch, statedb.Preimages())
	if err := blockBatch.Write(); err != nil {
		log.Crit("Failed to write block into disk", "err", err)
	}
	// Commit all cached state changes into underlying memory database.
	root, err := statedb.Commit(endBlock.NumberU64(), bc.chainConfig.IsEIP158(endBlock.Number()), bc.chainConfig.IsCancun(endBlock.Number(), endBlock.Time()))
	if err != nil {
		return err
	}
	// If node is running in path mode, skip explicit gc operation
	// which is unnecessary in this mode.
	if bc.triedb.Scheme() == rawdb.PathScheme {
		return nil
	}
	// If we're running an archive node, always flush
	if bc.cfg.ArchiveMode {
		return bc.triedb.Commit(root, false)
	}
	// Full but not archive node, do proper garbage collection
	bc.triedb.Reference(root, common.Hash{}) // metadata reference to keep trie alive
	bc.triegc.Push(root, -int64(endBlock.NumberU64()))

	// Flush limits are not considered for the first TriesInMemory blocks.
	current := endBlock.NumberU64()
	if current <= state.TriesInMemory {
		return nil
	}
	// If we exceeded our memory allowance, flush matured singleton nodes to disk
	var (
		_, nodes, imgs = bc.triedb.Size() // all memory is contained within the nodes return for hashdb
		limit          = common.StorageSize(bc.cfg.TrieDirtyLimit) * 1024 * 1024
	)
	if nodes > limit || imgs > 4*1024*1024 {
		bc.triedb.Cap(limit - ethdb.IdealBatchSize)
	}
	// Find the next state trie we need to commit
	chosen := current - state.TriesInMemory
	flushInterval := time.Duration(bc.flushInterval.Load())
	// If we exceeded time allowance, flush an entire trie to disk
	if bc.gcproc > flushInterval {
		// If the header is missing (canonical chain behind), we're reorging a low
		// diff sidechain. Suspend committing until this operation is completed.
		header := bc.GetHeaderByNumber(chosen)
		if header == nil {
			log.Warn("Reorg in progress, trie commit postponed", "number", chosen)
		} else {
			// If we're exceeding limits but haven't reached a large enough memory gap,
			// warn the user that the system is becoming unstable.
			if chosen < bc.lastWrite+state.TriesInMemory && bc.gcproc >= 2*flushInterval {
				log.Info("State in memory for too long, committing", "time", bc.gcproc, "allowance", flushInterval, "optimum", float64(chosen-bc.lastWrite)/state.TriesInMemory)
			}
			// Flush an entire trie and restart the counters
			bc.triedb.Commit(header.Root, true)
			bc.lastWrite = chosen
			bc.gcproc = 0
		}
	}
	// Garbage collect anything below our required write retention
	for !bc.triegc.Empty() {
		root, number := bc.triegc.Pop()
		if uint64(-number) > chosen {
			bc.triegc.Push(root, number)
			break
		}
		bc.triedb.Dereference(root)
	}
	return nil
}

// Due to blockHash opCode, blockHeaders must be written to DB before commit.
func (bc *BlockChain) writeNBlockHeaders(chain types.Blocks) {
	blockBatch := bc.db.NewBatch()
	for _, block := range chain {
		rawdb.WriteHeader(blockBatch, block.Header())
	}
	if err := blockBatch.Write(); err != nil {
		log.Crit("Failed to write block into disk", "err", err)
	}
}

func (bc *BlockChain) writeNBlocksAndSetHead(startBlock, endBlock *types.Block, receipts []*types.Receipt, logs []*types.Log, state *state.StateDB, emitHeadEvent bool) (status WriteStatus, err error) {
	if err := bc.writeBlockWithState(endBlock, receipts, state); err != nil {
		return NonStatTy, err
	}
	currentBlock := bc.CurrentBlock()

	// Reorganise the chain if the parent is not the head block
	if startBlock.ParentHash() != currentBlock.Hash() {
		if err := bc.reorg(currentBlock, endBlock.Header()); err != nil {
			return NonStatTy, err
		}
	}

	// Set new head.
	bc.writeHeadBlock(endBlock)

	bc.chainFeed.Send(ChainEvent{Header: endBlock.Header()})
	if len(logs) > 0 {
		bc.logsFeed.Send(logs)
	}
	// In theory, we should fire a ChainHeadEvent when we inject
	// a canonical block, but sometimes we can insert a batch of
	// canonical blocks. Avoid firing too many ChainHeadEvents,
	// we will fire an accumulated ChainHeadEvent and disable fire
	// event here.
	if emitHeadEvent {
		bc.chainHeadFeed.Send(ChainHeadEvent{Header: endBlock.Header()})
	}
	return CanonStatTy, nil
}

func (bc *BlockChain) processBlockWithState(parentRoot common.Hash, block *types.Block, setHead bool, makeWitness bool, statedb *state.StateDB) (_ *blockProcessingResult, blockEndErr error) {
	var (
		err       error
		startTime = time.Now()
		interrupt atomic.Bool
	)
	defer interrupt.Store(true) // terminate the prefetch at the end
	if !bc.cfg.NoPrefetch {
		panic("Must enable --cache.noprefetch to perf N-blocks")
	}

	if bc.cfg.NoPrefetch {
		// Use stateDB snapshotted from statedb.MergePostBalStates
	} else {
		fmt.Println("prefetch enabled===========================")
		// If prefetching is enabled, run that against the current state to pre-cache
		// transactions and probabilistically some of the account/storage trie nodes.
		//
		// Note: the main processor and prefetcher share the same reader with a local
		// cache for mitigating the overhead of state access.
		prefetch, process, err := bc.statedb.ReadersWithCacheStats(parentRoot)
		if err != nil {
			return nil, err
		}
		throwaway, err := state.NewWithReader(parentRoot, bc.statedb, prefetch)
		if err != nil {
			return nil, err
		}
		statedb, err = state.NewWithReader(parentRoot, bc.statedb, process)
		if err != nil {
			return nil, err
		}
		// Upload the statistics of reader at the end
		defer func() {
			stats := prefetch.GetStats()
			accountCacheHitPrefetchMeter.Mark(stats.AccountHit)
			accountCacheMissPrefetchMeter.Mark(stats.AccountMiss)
			storageCacheHitPrefetchMeter.Mark(stats.StorageHit)
			storageCacheMissPrefetchMeter.Mark(stats.StorageMiss)
			stats = process.GetStats()
			accountCacheHitMeter.Mark(stats.AccountHit)
			accountCacheMissMeter.Mark(stats.AccountMiss)
			storageCacheHitMeter.Mark(stats.StorageHit)
			storageCacheMissMeter.Mark(stats.StorageMiss)
		}()

		go func(start time.Time, throwaway *state.StateDB, block *types.Block) {
			// Disable tracing for prefetcher executions.
			vmCfg := bc.cfg.VmConfig
			vmCfg.Tracer = nil
			bc.prefetcher.Prefetch(block, throwaway, vmCfg, &interrupt)

			blockPrefetchExecuteTimer.Update(time.Since(start))
			if interrupt.Load() {
				blockPrefetchInterruptMeter.Mark(1)
			}
		}(time.Now(), throwaway, block)
	}

	// If we are past Byzantium, enable prefetching to pull in trie node paths
	// while processing transactions. Before Byzantium the prefetcher is mostly
	// useless due to the intermediate root hashing after each transaction.
	var witness *stateless.Witness
	if bc.chainConfig.IsByzantium(block.Number()) {
		// Generate witnesses either if we're self-testing, or if it's the
		// only block being inserted. A bit crude, but witnesses are huge,
		// so we refuse to make an entire chain of them.
		if bc.cfg.VmConfig.StatelessSelfValidation || makeWitness {
			witness, err = stateless.NewWitness(block.Header(), bc)
			if err != nil {
				return nil, err
			}
		}
		// Don't start the prefetcher, as it can significantly degrade performance. We've already prefetched the trie in insertChainN.
		// statedb.StartPrefetcher("chain", witness)
		// defer statedb.StopPrefetcher()
	}

	if bc.logger != nil && bc.logger.OnBlockStart != nil {
		bc.logger.OnBlockStart(tracing.BlockEvent{
			Block:     block,
			Finalized: bc.CurrentFinalBlock(),
			Safe:      bc.CurrentSafeBlock(),
		})
	}
	if bc.logger != nil && bc.logger.OnBlockEnd != nil {
		defer func() {
			bc.logger.OnBlockEnd(blockEndErr)
		}()
	}

	// Process block using the parent state as reference point
	pstart := time.Now()
	res, err := bc.processor.Process(block, statedb, bc.cfg.VmConfig)
	if err != nil {
		bc.reportBlock(block, res, err)
		return nil, err
	}
	ptime := time.Since(pstart)

	vstart := time.Now()
	if err := bc.validator.ValidateState(block, statedb, res, false); err != nil {
		bc.reportBlock(block, res, err)
		return nil, err
	}
	vtime := time.Since(vstart)

	// If witnesses was generated and stateless self-validation requested, do
	// that now. Self validation should *never* run in production, it's more of
	// a tight integration to enable running *all* consensus tests through the
	// witness builder/runner, which would otherwise be impossible due to the
	// various invalid chain states/behaviors being contained in those tests.
	xvstart := time.Now()
	if witness := statedb.Witness(); witness != nil && bc.cfg.VmConfig.StatelessSelfValidation {
		log.Warn("Running stateless self-validation", "block", block.Number(), "hash", block.Hash())

		// Remove critical computed fields from the block to force true recalculation
		context := block.Header()
		context.Root = common.Hash{}
		context.ReceiptHash = common.Hash{}

		task := types.NewBlockWithHeader(context).WithBody(*block.Body())

		// Run the stateless self-cross-validation
		crossStateRoot, crossReceiptRoot, err := ExecuteStateless(bc.chainConfig, bc.cfg.VmConfig, task, witness)
		if err != nil {
			return nil, fmt.Errorf("stateless self-validation failed: %v", err)
		}
		if crossStateRoot != block.Root() {
			return nil, fmt.Errorf("stateless self-validation root mismatch (cross: %x local: %x)", crossStateRoot, block.Root())
		}
		if crossReceiptRoot != block.ReceiptHash() {
			return nil, fmt.Errorf("stateless self-validation receipt root mismatch (cross: %x local: %x)", crossReceiptRoot, block.ReceiptHash())
		}
	}
	xvtime := time.Since(xvstart)
	proctime := time.Since(startTime) // processing + validation + cross validation

	// Update the metrics touched during block processing and validation
	accountReadTimer.Update(statedb.AccountReads) // Account reads are complete(in processing)
	storageReadTimer.Update(statedb.StorageReads) // Storage reads are complete(in processing)
	if statedb.AccountLoaded != 0 {
		accountReadSingleTimer.Update(statedb.AccountReads / time.Duration(statedb.AccountLoaded))
	}
	if statedb.StorageLoaded != 0 {
		storageReadSingleTimer.Update(statedb.StorageReads / time.Duration(statedb.StorageLoaded))
	}
	accountUpdateTimer.Update(statedb.AccountUpdates)                                 // Account updates are complete(in validation)
	storageUpdateTimer.Update(statedb.StorageUpdates)                                 // Storage updates are complete(in validation)
	accountHashTimer.Update(statedb.AccountHashes)                                    // Account hashes are complete(in validation)
	triehash := statedb.AccountHashes                                                 // The time spent on tries hashing
	trieUpdate := statedb.AccountUpdates + statedb.StorageUpdates                     // The time spent on tries update
	blockExecutionTimer.Update(ptime - (statedb.AccountReads + statedb.StorageReads)) // The time spent on EVM processing
	blockValidationTimer.Update(vtime - (triehash + trieUpdate))                      // The time spent on block validation
	blockCrossValidationTimer.Update(xvtime)                                          // The time spent on stateless cross validation

	return &blockProcessingResult{
		usedGas:  res.GasUsed,
		procTime: proctime,
		status:   CanonStatTy, // Assue write status is always correct
		witness:  witness,
	}, nil
}
