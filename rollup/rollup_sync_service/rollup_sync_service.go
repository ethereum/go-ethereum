package rollup_sync_service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/scroll-tech/da-codec/encoding"

	"github.com/scroll-tech/go-ethereum/common"
	"github.com/scroll-tech/go-ethereum/core"
	"github.com/scroll-tech/go-ethereum/core/rawdb"
	"github.com/scroll-tech/go-ethereum/ethdb"
	"github.com/scroll-tech/go-ethereum/log"
	"github.com/scroll-tech/go-ethereum/metrics"
	"github.com/scroll-tech/go-ethereum/node"
	"github.com/scroll-tech/go-ethereum/params"
	"github.com/scroll-tech/go-ethereum/rollup/da_syncer"
	"github.com/scroll-tech/go-ethereum/rollup/da_syncer/blob_client"
	"github.com/scroll-tech/go-ethereum/rollup/da_syncer/da"
	"github.com/scroll-tech/go-ethereum/rollup/l1"
	"github.com/scroll-tech/go-ethereum/rollup/rcfg"
	"github.com/scroll-tech/go-ethereum/rollup/withdrawtrie"
)

const (
	// defaultSyncInterval is the frequency at which we query for new rollup event.
	defaultSyncInterval = 30 * time.Second

	// defaultMaxRetries is the maximum number of retries allowed when the local node is not synced up to the required block height.
	defaultMaxRetries = 20

	// defaultGetBlockInRangeRetryDelay is the time delay between retries when attempting to get blocks in range.
	// The service will wait for this duration if it detects that the local node has not synced up to the block height
	// of a specific L1 batch finalize event.
	defaultGetBlockInRangeRetryDelay = 60 * time.Second

	// defaultLogInterval is the frequency at which we print the latest processed block.
	defaultLogInterval = 5 * time.Minute

	// rewindL1Height is the number of blocks to rewind the L1 sync height when a missing batch event is detected.
	rewindL1Height = 100
)

var (
	finalizedBlockGauge  = metrics.NewRegisteredGauge("chain/head/finalized", nil)
	ErrMissingBatchEvent = errors.New("ErrMissingBatchEvent")
)

type errShouldResetSyncHeight struct {
	height uint64
}

func (e errShouldResetSyncHeight) Error() string {
	return fmt.Sprintf("ErrShouldResetSyncHeight: height=%d", e.height)
}

// RollupSyncService collects ScrollChain batch commit/revert/finalize events and stores metadata into db.
type RollupSyncService struct {
	ctx     context.Context
	cancel  context.CancelFunc
	db      ethdb.Database
	bc      *core.BlockChain
	stack   *node.Node
	stateMu sync.Mutex

	callDataBlobSource *da.CalldataBlobSource
}

func NewRollupSyncService(ctx context.Context, genesisConfig *params.ChainConfig, db ethdb.Database, l1Client l1.Client, bc *core.BlockChain, stack *node.Node, config da_syncer.Config) (*RollupSyncService, error) {
	if genesisConfig.Scroll.L1Config == nil {
		return nil, fmt.Errorf("missing L1 config in genesis")
	}

	// Initialize the latestProcessedBlock with the block just before the L1 deployment block.
	// This serves as a default value when there's no L1 rollup events synced in the database.
	var latestProcessedBlock uint64
	if stack.Config().L1DeploymentBlock > 0 {
		latestProcessedBlock = stack.Config().L1DeploymentBlock - 1
	}

	block := rawdb.ReadRollupEventSyncedL1BlockNumber(db)
	if block != nil {
		// restart from latest synced block number
		latestProcessedBlock = *block
	}

	var success bool
	ctx, cancel := context.WithCancel(ctx)
	defer func() {
		if !success {
			cancel()
		}
	}()

	l1Reader, err := l1.NewReader(ctx, l1.Config{
		ScrollChainAddress:    genesisConfig.Scroll.L1Config.ScrollChainAddress,
		L1MessageQueueAddress: genesisConfig.Scroll.L1Config.L1MessageQueueAddress,
	}, l1Client)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize l1.Reader, err = %w", err)
	}

	blobClientList := blob_client.NewBlobClients()
	if config.BeaconNodeAPIEndpoint != "" {
		beaconNodeClient, err := blob_client.NewBeaconNodeClient(config.BeaconNodeAPIEndpoint)
		if err != nil {
			log.Warn("failed to create BeaconNodeClient", "err", err)
		} else {
			blobClientList.AddBlobClient(beaconNodeClient)
		}
	}
	if config.BlobScanAPIEndpoint != "" {
		blobClientList.AddBlobClient(blob_client.NewBlobScanClient(config.BlobScanAPIEndpoint))
	}
	if config.BlockNativeAPIEndpoint != "" {
		blobClientList.AddBlobClient(blob_client.NewBlockNativeClient(config.BlockNativeAPIEndpoint))
	}
	if config.AwsS3BlobAPIEndpoint != "" {
		blobClientList.AddBlobClient(blob_client.NewAwsS3Client(config.AwsS3BlobAPIEndpoint))
	}
	if blobClientList.Size() == 0 {
		return nil, errors.New("no blob client is configured for rollup verifier. Please provide at least one blob client via command line flag")
	}

	calldataBlobSource, err := da.NewCalldataBlobSource(ctx, latestProcessedBlock, l1Reader, blobClientList, db)
	if err != nil {
		return nil, fmt.Errorf("failed to create calldata blob source: %w", err)
	}

	success = true

	return &RollupSyncService{
		ctx:    ctx,
		cancel: cancel,
		db:     db,
		bc:     bc,
		stack:  stack,

		callDataBlobSource: calldataBlobSource,
	}, nil
}

func (s *RollupSyncService) Start() {
	if s == nil {
		return
	}

	log.Info("Starting rollup event sync background service", "latest processed block", s.callDataBlobSource.L1Height())

	finalizedBlockHeightPtr := rawdb.ReadFinalizedL2BlockNumber(s.db)
	if finalizedBlockHeightPtr != nil {
		finalizedBlockGauge.Update(int64(*finalizedBlockHeightPtr))
	}

	go func() {
		syncTicker := time.NewTicker(defaultSyncInterval)
		defer syncTicker.Stop()

		logTicker := time.NewTicker(defaultLogInterval)
		defer logTicker.Stop()

		for {
			select {
			case <-s.ctx.Done():
				return
			case <-syncTicker.C:
				err := s.fetchRollupEvents()
				if err != nil {
					// Do not log the error if the context is canceled.
					select {
					case <-s.ctx.Done():
						return
					default:
					}

					log.Error("failed to fetch rollup events", "err", err)
				}
			case <-logTicker.C:
				log.Info("Sync rollup events progress update", "latest processed block", s.callDataBlobSource.L1Height())
			}
		}
	}()
}

func (s *RollupSyncService) Stop() {
	if s == nil {
		return
	}

	log.Info("Stopping rollup event sync background service")

	if s.cancel != nil {
		s.cancel()
	}
}

// ResetStartSyncHeight resets the RollupSyncService to a specific L1 block height
func (s *RollupSyncService) ResetStartSyncHeight(height uint64) {
	if s == nil {
		return
	}

	s.stateMu.Lock()
	defer s.stateMu.Unlock()

	s.callDataBlobSource.SetL1Height(height)
	log.Info("Reset sync service", "height", height)
}

func (s *RollupSyncService) fetchRollupEvents() error {
	s.stateMu.Lock()
	defer s.stateMu.Unlock()

	for {
		prevL1Height := s.callDataBlobSource.L1Height()

		daEntries, err := s.callDataBlobSource.NextData()
		if err != nil {
			if errors.Is(err, da.ErrSourceExhausted) {
				log.Trace("Sync service exhausted data source, waiting for next data")
				return nil
			}

			return fmt.Errorf("failed to get next data: %w", err)
		}

		if err = s.updateRollupEvents(daEntries); err != nil {
			var resetSyncErr errShouldResetSyncHeight
			if errors.As(err, &resetSyncErr) {
				log.Warn("Resetting rollup sync height", "height", resetSyncErr.height)
				s.callDataBlobSource.SetL1Height(resetSyncErr.height)
				return nil
			}
			if errors.Is(err, ErrMissingBatchEvent) {
				// make sure no underflow
				var rewindTo uint64
				if prevL1Height > rewindL1Height {
					rewindTo = prevL1Height - rewindL1Height
				}

				// If there's a missing batch event, rewind the L1 sync height by some blocks to re-fetch from L1 RPC and
				// replay creating corresponding CommittedBatchMeta in local DB.
				// This happens recursively until the missing event has been recovered as we will call fetchRollupEvents again
				// with the `L1Height = prevL1Height - rewindL1Height`.
				s.callDataBlobSource.SetL1Height(rewindTo)

				return fmt.Errorf("missing batch event, rewinding L1 sync height by %d blocks to %d: %w", rewindL1Height, rewindTo, err)
			}

			// Reset the L1 height to the previous value to retry fetching the same data.
			s.callDataBlobSource.SetL1Height(prevL1Height)
			return fmt.Errorf("failed to parse and update rollup event logs: %w", err)
		}

		log.Trace("Sync service fetched rollup events", "latest processed L1 block", s.callDataBlobSource.L1Height(), "latest finalized L1 block", s.callDataBlobSource.L1Finalized())

		// note: the batch updates in updateRollupEvents are idempotent, if we crash
		// before this line and re-execute the previous steps, we will get the same result.
		rawdb.WriteRollupEventSyncedL1BlockNumber(s.db, s.callDataBlobSource.L1Height())
	}
}

func (s *RollupSyncService) updateRollupEvents(daEntries da.Entries) error {
	for _, entry := range daEntries {
		switch entry.Type() {
		case da.CommitBatchV0Type, da.CommitBatchWithBlobType:
			log.Trace("found new CommitBatch event", "batch index", entry.BatchIndex())

			entryWithBlocks, ok := entry.(da.EntryWithBlocks)
			if !ok {
				return fmt.Errorf("failed to cast to EntryWithBlocks, batch index: %v", entry.BatchIndex())
			}

			committedBatchMeta, err := s.getCommittedBatchMeta(entryWithBlocks)
			if err != nil {
				return fmt.Errorf("failed to get committed batch meta, batch index: %v, err: %w", entry.BatchIndex(), err)
			}

			rawdb.WriteCommittedBatchMeta(s.db, entry.BatchIndex(), committedBatchMeta)

		case da.RevertBatchType:
			log.Trace("found new RevertBatch event", "batch index", entry.BatchIndex())
			if err := s.handleRevertEvent(entry.Event()); err != nil {
				return fmt.Errorf("failed to handle revert event, batch index: %v, err: %w", entry.BatchIndex(), err)
			}

		case da.FinalizeBatchType:
			event, ok := entry.Event().(*l1.FinalizeBatchEvent)
			// This should never happen because we just checked the batch type
			if !ok {
				return fmt.Errorf("failed to cast to FinalizeBatchEvent, batch index: %v", entry.BatchIndex())
			}

			batchIndex := entry.BatchIndex()
			log.Trace("found new FinalizeBatch event", "batch index", batchIndex)

			lastFinalizedBatchIndex := rawdb.ReadLastFinalizedBatchIndex(s.db)

			// After Darwin, FinalizeBatch event emitted every bundle, which contains multiple batches.
			// Therefore, there are a range of finalized batches need to be saved into db.
			//
			// The range logic also applies to the batches before Darwin when FinalizeBatch event emitted
			// per single batch. In this situation, `batchIndex` just equals to `*lastFinalizedBatchIndex + 1`
			// and only one batch is processed through the for loop.
			startBatchIndex := batchIndex
			if lastFinalizedBatchIndex != nil {
				startBatchIndex = *lastFinalizedBatchIndex + 1
			} else {
				log.Warn("got nil when reading last finalized batch index. This should happen only once.")
			}

			parentFinalizedBatchMeta := &rawdb.FinalizedBatchMeta{}
			if startBatchIndex > 0 {
				parentFinalizedBatchMeta = rawdb.ReadFinalizedBatchMeta(s.db, startBatchIndex-1)
			}

			var highestFinalizedBlockNumber uint64
			batchWriter := s.db.NewBatch()
			for index := startBatchIndex; index <= batchIndex; index++ {
				var parentCommittedBatchMeta *rawdb.CommittedBatchMeta
				var err error
				if index > 0 {
					if parentCommittedBatchMeta, err = rawdb.ReadCommittedBatchMeta(s.db, index-1); err != nil {
						return fmt.Errorf("failed to read parent committed batch meta, batch index: %v, err: %w", index-1, errors.Join(ErrMissingBatchEvent, err))
					}
					if parentCommittedBatchMeta == nil {
						return fmt.Errorf("parent committed batch meta = nil, batch index: %v, err: %w", index-1, ErrMissingBatchEvent)
					}
				}
				committedBatchMeta, err := rawdb.ReadCommittedBatchMeta(s.db, index)
				if err != nil {
					return fmt.Errorf("failed to read committed batch meta, batch index: %v, err: %w", index, errors.Join(ErrMissingBatchEvent, err))
				}
				if committedBatchMeta == nil {
					return fmt.Errorf("committed batch meta = nil, batch index: %v, err: %w", index, ErrMissingBatchEvent)
				}

				chunks, err := s.getLocalChunksForBatch(committedBatchMeta.ChunkBlockRanges)
				if err != nil {
					return fmt.Errorf("failed to get local node info, batch index: %v, err: %w", index, err)
				}

				endBlock, finalizedBatchMeta, err := validateBatch(index, event, parentFinalizedBatchMeta, parentCommittedBatchMeta, committedBatchMeta, chunks, s.stack)
				if err != nil {
					return fmt.Errorf("fatal: validateBatch failed: finalize event: %v, err: %w", event, err)
				}

				rawdb.WriteFinalizedBatchMeta(batchWriter, index, finalizedBatchMeta)
				highestFinalizedBlockNumber = endBlock
				parentFinalizedBatchMeta = finalizedBatchMeta

				if index%100 == 0 {
					log.Info("finalized batch progress", "batch index", index, "finalized l2 block height", endBlock)
				}
			}

			if err := batchWriter.Write(); err != nil {
				log.Error("fatal: failed to batch write finalized batch meta to database", "startBatchIndex", startBatchIndex, "endBatchIndex", batchIndex,
					"batchCount", batchIndex-startBatchIndex+1, "highestFinalizedBlockNumber", highestFinalizedBlockNumber, "err", err)
				return fmt.Errorf("failed to batch write finalized batch meta to database: %w", err)
			}
			rawdb.WriteFinalizedL2BlockNumber(s.db, highestFinalizedBlockNumber)
			finalizedBlockGauge.Update(int64(highestFinalizedBlockNumber))
			rawdb.WriteLastFinalizedBatchIndex(s.db, batchIndex)
			log.Debug("write finalized l2 block number", "batch index", batchIndex, "finalized l2 block height", highestFinalizedBlockNumber)

		default:
			return fmt.Errorf("unknown daEntry, type: %d, batch index: %d", entry.Type(), entry.BatchIndex())
		}
	}

	return nil
}

func (s *RollupSyncService) handleRevertEvent(event l1.RollupEvent) error {
	switch event.Type() {
	case l1.RevertEventV0Type:
		revertBatch, ok := event.(*l1.RevertBatchEventV0)
		if !ok {
			return fmt.Errorf("unexpected type of revert event: %T, expected RevertEventV0Type", event)
		}

		rawdb.DeleteCommittedBatchMeta(s.db, revertBatch.BatchIndex().Uint64())

	case l1.RevertEventV7Type:
		revertBatch, ok := event.(*l1.RevertBatchEventV7)
		if !ok {
			return fmt.Errorf("unexpected type of revert event: %T, expected RevertEventV7Type", event)
		}

		// delete all batches from revertBatch.StartBatchIndex (inclusive) to revertBatch.FinishBatchIndex (inclusive)
		for i := revertBatch.StartBatchIndex().Uint64(); i <= revertBatch.FinishBatchIndex().Uint64(); i++ {
			rawdb.DeleteCommittedBatchMeta(s.db, i)
		}
	default:
		return fmt.Errorf("unexpected type of revert event: %T", event)
	}

	return nil
}

func (s *RollupSyncService) getLocalChunksForBatch(chunkBlockRanges []*rawdb.ChunkBlockRange) ([]*encoding.Chunk, error) {
	if len(chunkBlockRanges) == 0 {
		return nil, fmt.Errorf("chunkBlockRanges is empty")
	}
	endBlockNumber := chunkBlockRanges[len(chunkBlockRanges)-1].EndBlockNumber
	for i := 0; i < defaultMaxRetries; i++ {
		if s.ctx.Err() != nil {
			log.Info("Context canceled", "reason", s.ctx.Err())
			return nil, s.ctx.Err()
		}

		localSyncedBlockHeight := s.bc.CurrentBlock().Number().Uint64()
		if localSyncedBlockHeight >= endBlockNumber {
			break // ready to proceed, exit retry loop
		}

		log.Debug("local node is not synced up to the required block height, waiting for next retry",
			"retries", i+1, "local synced block height", localSyncedBlockHeight, "required end block number", endBlockNumber)
		time.Sleep(defaultGetBlockInRangeRetryDelay)
	}

	localSyncedBlockHeight := s.bc.CurrentBlock().Number().Uint64()
	if localSyncedBlockHeight < endBlockNumber {
		return nil, fmt.Errorf("local node is not synced up to the required block height: %v, local synced block height: %v", endBlockNumber, localSyncedBlockHeight)
	}

	chunks := make([]*encoding.Chunk, len(chunkBlockRanges))
	for i, cr := range chunkBlockRanges {
		chunks[i] = &encoding.Chunk{Blocks: make([]*encoding.Block, cr.EndBlockNumber-cr.StartBlockNumber+1)}
		for j := cr.StartBlockNumber; j <= cr.EndBlockNumber; j++ {
			block := s.bc.GetBlockByNumber(j)
			if block == nil {
				return nil, fmt.Errorf("failed to get block by number: %v", i)
			}
			txData := encoding.TxsToTxsData(block.Transactions())
			chunks[i].Blocks[j-cr.StartBlockNumber] = &encoding.Block{
				Header:       block.Header(),
				Transactions: txData,
			}

			// read withdraw root, if available
			// note: historical state is not available on full nodes
			state, err := s.bc.StateAt(block.Root())
			if err != nil {
				log.Trace("State is not available, skipping withdraw trie validation", "blockNumber", block.NumberU64(), "blockHash", block.Hash().Hex(), "err", err)
				continue
			}
			withdrawRoot := withdrawtrie.ReadWTRSlot(rcfg.L2MessageQueueAddress, state)
			chunks[i].Blocks[j-cr.StartBlockNumber].WithdrawRoot = withdrawRoot
		}
	}

	return chunks, nil
}

func (s *RollupSyncService) getCommittedBatchMeta(commitedBatch da.EntryWithBlocks) (*rawdb.CommittedBatchMeta, error) {
	if commitedBatch.BatchIndex() == 0 {
		return &rawdb.CommittedBatchMeta{
			Version:                0,
			ChunkBlockRanges:       []*rawdb.ChunkBlockRange{{StartBlockNumber: 0, EndBlockNumber: 0}},
			PostL1MessageQueueHash: common.Hash{},
		}, nil
	}

	chunkRanges, err := blockRangesFromChunks(commitedBatch.Chunks())
	if err != nil {
		return nil, fmt.Errorf("failed to decode block ranges from chunks, batch index: %v, err: %w", commitedBatch.BatchIndex(), err)
	}

	// With >= CodecV7 the batch creation changed. We need to compute and store PostL1MessageQueueHash.
	// PrevL1MessageQueueHash of a batch == PostL1MessageQueueHash of the previous batch.
	// We need to do this for every committed batch (instead of finalized batch) because the L1MessageQueueHash
	// is a continuous hash of all L1 messages over all batches. With bundles we only receive the finalize event
	// for the last batch of the bundle.
	var lastL1MessageQueueHash common.Hash
	if commitedBatch.Version() >= encoding.CodecV7 {
		parentCommittedBatchMeta, err := rawdb.ReadCommittedBatchMeta(s.db, commitedBatch.BatchIndex()-1)
		if err != nil {
			return nil, fmt.Errorf("failed to read parent committed batch meta, batch index: %v, err: %w", commitedBatch.BatchIndex()-1, errors.Join(ErrMissingBatchEvent, err))
		}
		if parentCommittedBatchMeta == nil {
			return nil, fmt.Errorf("parent committed batch meta = nil, batch index: %v, err: %w", commitedBatch.BatchIndex()-1, ErrMissingBatchEvent)
		}

		// For the first batch of CodecV7, this will be the empty hash.
		prevL1MessageQueueHash := parentCommittedBatchMeta.PostL1MessageQueueHash

		chunks, err := s.getLocalChunksForBatch(chunkRanges)
		if err != nil {
			return nil, fmt.Errorf("failed to get local node info, batch index: %v, err: %w", commitedBatch.BatchIndex(), err)
		}

		// There is no chunks encoded in a batch anymore with >= CodecV7.
		// For compatibility reason here we still use a single chunk to store the block ranges of the batch.
		// We make sure that there is really only one chunk which contains all blocks of the batch.
		if len(chunks) != 1 {
			return nil, fmt.Errorf("invalid argument: chunk count is not 1 for CodecV%v, batch index: %v", commitedBatch.Version(), commitedBatch.BatchIndex())
		}

		lastL1MessageQueueHash, err = encoding.MessageQueueV2ApplyL1MessagesFromBlocks(prevL1MessageQueueHash, chunks[0].Blocks)
		if err != nil {
			return nil, fmt.Errorf("failed to apply L1 messages from blocks, batch index: %v, err: %w", chunks[0], err)
		}
	}

	return &rawdb.CommittedBatchMeta{
		Version:                uint8(commitedBatch.Version()),
		ChunkBlockRanges:       chunkRanges,
		PostL1MessageQueueHash: lastL1MessageQueueHash,
	}, nil
}

// validateBatch verifies the consistency between the L1 contract and L2 node data.
// It performs the following checks:
// 1. Recalculates the batch hash locally
// 2. Compares local state root, local withdraw root, and locally calculated batch hash with L1 data (for the last batch only when "finalize by bundle")
//
// The function will terminate the node and exit if any consistency check fails.
//
// Parameters:
//   - batchIndex: batch index of the validated batch
//   - event: L1 finalize batch event data
//   - parentFinalizedBatchMeta: metadata of the finalized parent batch
//   - committedBatchMeta: committed batch metadata stored in the database
//   - chunks: slice of chunk data for the current batch
//   - stack: node stack to terminate the node in case of inconsistency
//
// Returns:
// - uint64: the end block height of the batch
// - *rawdb.FinalizedBatchMeta: finalized batch metadata
// - error: any error encountered during validation
//
// Note: This function is compatible with both "finalize by batch" and "finalize by bundle" methods.
// In "finalize by bundle", only the last batch of each bundle is fully verified.
// This check still ensures the correctness of all batch hashes in the bundle due to the parent-child relationship between batch hashes.
func validateBatch(batchIndex uint64, event *l1.FinalizeBatchEvent, parentFinalizedBatchMeta *rawdb.FinalizedBatchMeta, parentCommittedBatchMeta *rawdb.CommittedBatchMeta, committedBatchMeta *rawdb.CommittedBatchMeta, chunks []*encoding.Chunk, stack *node.Node) (uint64, *rawdb.FinalizedBatchMeta, error) {
	if len(chunks) == 0 {
		return 0, nil, fmt.Errorf("invalid argument: length of chunks is 0, batch index: %v", batchIndex)
	}

	startChunk := chunks[0]
	if len(startChunk.Blocks) == 0 {
		return 0, nil, fmt.Errorf("invalid argument: block count of start chunk is 0, batch index: %v", batchIndex)
	}
	startBlock := startChunk.Blocks[0]

	endChunk := chunks[len(chunks)-1]
	if len(endChunk.Blocks) == 0 {
		return 0, nil, fmt.Errorf("invalid argument: block count of end chunk is 0, batch index: %v", batchIndex)
	}
	endBlock := endChunk.Blocks[len(endChunk.Blocks)-1]

	// Note: All params of batch are calculated locally based on the block data.
	var batch *encoding.Batch
	if encoding.CodecVersion(committedBatchMeta.Version) < encoding.CodecV7 {
		batch = &encoding.Batch{
			Index:                      batchIndex,
			TotalL1MessagePoppedBefore: parentFinalizedBatchMeta.TotalL1MessagePopped,
			ParentBatchHash:            parentFinalizedBatchMeta.BatchHash,
			Chunks:                     chunks,
		}
	} else {
		// With >= CodecV7 the batch creation changed. There is no chunks encoded in a batch anymore.
		// For compatibility reason here we still use a single chunk to store the block ranges of the batch.
		// We make sure that there is really only one chunk which contains all blocks of the batch.
		if len(chunks) != 1 {
			return 0, nil, fmt.Errorf("invalid argument: chunk count is not 1 for CodecV%v, batch index: %v", committedBatchMeta.Version, batchIndex)
		}

		batch = &encoding.Batch{
			Index:                  batchIndex,
			ParentBatchHash:        parentFinalizedBatchMeta.BatchHash,
			Blocks:                 startChunk.Blocks,
			PrevL1MessageQueueHash: parentCommittedBatchMeta.PostL1MessageQueueHash,
			PostL1MessageQueueHash: committedBatchMeta.PostL1MessageQueueHash,
		}
	}

	codecVersion := encoding.CodecVersion(committedBatchMeta.Version)
	codec, err := encoding.CodecFromVersion(codecVersion)
	if err != nil {
		return 0, nil, fmt.Errorf("unsupported codec version: %v, batch index: %v, err: %w", codecVersion, batchIndex, err)
	}

	daBatch, err := codec.NewDABatch(batch)
	if err != nil {
		// This is hotfix for the L1 message hash mismatch issue which lead to wrong committedBatchMeta.PostL1MessageQueueHash hashes.
		// These in turn lead to a wrongly computed batch hash locally. This happened after upgrading to EuclidV2
		// where da-codec was not updated to the latest version in l2geth.
		// If the error message due to mismatching PostL1MessageQueueHash contains the same hash as the hardcoded one,
		// this means the node ran into this issue.
		// We need to reset the sync height to 1 block before the L1 block in which the last batch in CodecV6 was committed.
		// The node will overwrite the wrongly computed message queue hashes.
		if strings.Contains(err.Error(), "0xaa16faf2a1685fe1d7e0f2810b1a0e98c2841aef96596d10456a6d0f00000000") {
			log.Warn("Resetting sync height to L1 block 7892668 to fix L1 message queue hash calculation issue after EuclidV2 on Scroll Sepolia")
			return 0, nil, errShouldResetSyncHeight{height: 7892668}
		}
		// This is hotfix for the L1 message hash mismatch issue which lead to wrong committedBatchMeta.PostL1MessageQueueHash hashes.
		// This happened after upgrading to Feyman where rollup-verifier erroneously reset the prevMessageQueueHash to the empty hash.
		// If the error message due to mismatching PostL1MessageQueueHash contains the same hash as the hardcoded one,
		// this means the node ran into this issue.
		// We need to reset the sync height to before committing the first Feynman batch.
		if strings.Contains(err.Error(), "expected 0x19c790f49efb448b523d94e5672d9ed108656886be12c038cf39062700000000, got 0x0000000000000000000000000000000000000000000000000000000000000000") {
			log.Warn("Resetting sync height to L1 block 8816625 to fix L1 message queue hash calculation issue after Feynman on Scroll Sepolia")
			return 0, nil, errShouldResetSyncHeight{height: 8816625}
		}
		return 0, nil, fmt.Errorf("failed to create DA batch, batch index: %v, codec version: %v, err: %w", batchIndex, codecVersion, err)
	}
	localBatchHash := daBatch.Hash()

	localStateRoot := endBlock.Header.Root
	localWithdrawRoot := endBlock.WithdrawRoot

	// Note: If the state root, withdraw root, and batch headers match, this ensures the consistency of blocks and transactions
	// (including skipped transactions) between L1 and L2.
	//
	// Only check when batch index matches the index of the event. This is compatible with both "finalize by batch" and "finalize by bundle":
	// - finalize by batch: check all batches
	// - finalize by bundle: check the last batch, because only one event (containing the info of the last batch) is emitted per bundle
	if batchIndex == event.BatchIndex().Uint64() {
		if localStateRoot != event.StateRoot() {
			log.Error("State root mismatch", "batch index", event.BatchIndex().Uint64(), "start block", startBlock.Header.Number.Uint64(), "end block", endBlock.Header.Number.Uint64(), "parent batch hash", parentFinalizedBatchMeta.BatchHash.Hex(), "l1 finalized state root", event.StateRoot().Hex(), "l2 state root", localStateRoot.Hex())
			stack.Close()
			os.Exit(1)
		}

		// note: this check is optional,
		// withdraw root correctness is already implied by state root correctness.
		if localWithdrawRoot != (common.Hash{}) && localWithdrawRoot != event.WithdrawRoot() {
			log.Error("Withdraw root mismatch", "batch index", event.BatchIndex().Uint64(), "start block", startBlock.Header.Number.Uint64(), "end block", endBlock.Header.Number.Uint64(), "parent batch hash", parentFinalizedBatchMeta.BatchHash.Hex(), "l1 finalized withdraw root", event.WithdrawRoot().Hex(), "l2 withdraw root", localWithdrawRoot.Hex())
			stack.Close()
			os.Exit(1)
		}

		// Verify batch hash
		// This check ensures the correctness of all batch hashes in the bundle
		// due to the parent-child relationship between batch hashes
		if localBatchHash != event.BatchHash() {
			log.Error("Batch hash mismatch", "batch index", event.BatchIndex().Uint64(), "start block", startBlock.Header.Number.Uint64(), "end block", endBlock.Header.Number.Uint64(), "parent batch hash", parentFinalizedBatchMeta.BatchHash.Hex(), "parent TotalL1MessagePopped", parentFinalizedBatchMeta.TotalL1MessagePopped, "l1 finalized batch hash", event.BatchHash().Hex(), "l2 batch hash", localBatchHash.Hex())
			chunksJson, err := json.Marshal(chunks)
			if err != nil {
				log.Error("marshal chunks failed", "err", err)
			}
			log.Error("Chunks", "chunks", string(chunksJson))
			stack.Close()
			os.Exit(1)
		}
	}

	totalL1MessagePopped := parentFinalizedBatchMeta.TotalL1MessagePopped
	for _, chunk := range chunks {
		totalL1MessagePopped += chunk.NumL1Messages(totalL1MessagePopped)
	}
	finalizedBatchMeta := &rawdb.FinalizedBatchMeta{
		BatchHash:            localBatchHash,
		TotalL1MessagePopped: totalL1MessagePopped,
		StateRoot:            localStateRoot,
		WithdrawRoot:         event.WithdrawRoot(),
	}
	return endBlock.Header.Number.Uint64(), finalizedBatchMeta, nil
}

// blockRangesFromChunks decodes the provided chunks into a list of block ranges.
func blockRangesFromChunks(chunks []*encoding.DAChunkRawTx) ([]*rawdb.ChunkBlockRange, error) {
	var chunkBlockRanges []*rawdb.ChunkBlockRange
	for _, daChunkRawTx := range chunks {
		if len(daChunkRawTx.Blocks) == 0 {
			return nil, fmt.Errorf("no blocks found in DA chunk, chunk: %+v", daChunkRawTx)
		}

		chunkBlockRanges = append(chunkBlockRanges, &rawdb.ChunkBlockRange{
			StartBlockNumber: daChunkRawTx.Blocks[0].Number(),
			EndBlockNumber:   daChunkRawTx.Blocks[len(daChunkRawTx.Blocks)-1].Number(),
		})
	}

	return chunkBlockRanges, nil
}
