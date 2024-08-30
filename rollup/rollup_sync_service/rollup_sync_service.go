package rollup_sync_service

import (
	"context"
	"encoding/json"
	"fmt"
	"math/big"
	"os"
	"reflect"
	"time"

	"github.com/scroll-tech/da-codec/encoding"
	"github.com/scroll-tech/da-codec/encoding/codecv0"
	"github.com/scroll-tech/da-codec/encoding/codecv1"
	"github.com/scroll-tech/da-codec/encoding/codecv2"
	"github.com/scroll-tech/da-codec/encoding/codecv3"
	"github.com/scroll-tech/da-codec/encoding/codecv4"

	"github.com/scroll-tech/go-ethereum/accounts/abi"
	"github.com/scroll-tech/go-ethereum/common"
	"github.com/scroll-tech/go-ethereum/core"
	"github.com/scroll-tech/go-ethereum/core/rawdb"
	"github.com/scroll-tech/go-ethereum/core/types"
	"github.com/scroll-tech/go-ethereum/ethdb"
	"github.com/scroll-tech/go-ethereum/log"
	"github.com/scroll-tech/go-ethereum/node"
	"github.com/scroll-tech/go-ethereum/params"
	"github.com/scroll-tech/go-ethereum/rollup/rcfg"
	"github.com/scroll-tech/go-ethereum/rollup/sync_service"
	"github.com/scroll-tech/go-ethereum/rollup/withdrawtrie"
)

const (
	// defaultFetchBlockRange is the number of blocks that we collect in a single eth_getLogs query.
	defaultFetchBlockRange = uint64(100)

	// defaultSyncInterval is the frequency at which we query for new rollup event.
	defaultSyncInterval = 60 * time.Second

	// defaultMaxRetries is the maximum number of retries allowed when the local node is not synced up to the required block height.
	defaultMaxRetries = 20

	// defaultGetBlockInRangeRetryDelay is the time delay between retries when attempting to get blocks in range.
	// The service will wait for this duration if it detects that the local node has not synced up to the block height
	// of a specific L1 batch finalize event.
	defaultGetBlockInRangeRetryDelay = 60 * time.Second

	// defaultLogInterval is the frequency at which we print the latestProcessedBlock.
	defaultLogInterval = 5 * time.Minute
)

// RollupSyncService collects ScrollChain batch commit/revert/finalize events and stores metadata into db.
type RollupSyncService struct {
	ctx                           context.Context
	cancel                        context.CancelFunc
	client                        *L1Client
	db                            ethdb.Database
	latestProcessedBlock          uint64
	scrollChainABI                *abi.ABI
	l1CommitBatchEventSignature   common.Hash
	l1RevertBatchEventSignature   common.Hash
	l1FinalizeBatchEventSignature common.Hash
	bc                            *core.BlockChain
	stack                         *node.Node
}

func NewRollupSyncService(ctx context.Context, genesisConfig *params.ChainConfig, db ethdb.Database, l1Client sync_service.EthClient, bc *core.BlockChain, stack *node.Node) (*RollupSyncService, error) {
	// terminate if the caller does not provide an L1 client (e.g. in tests)
	if l1Client == nil || (reflect.ValueOf(l1Client).Kind() == reflect.Ptr && reflect.ValueOf(l1Client).IsNil()) {
		log.Warn("No L1 client provided, L1 rollup sync service will not run")
		return nil, nil
	}

	if genesisConfig.Scroll.L1Config == nil {
		return nil, fmt.Errorf("missing L1 config in genesis")
	}

	scrollChainABI, err := scrollChainMetaData.GetAbi()
	if err != nil {
		return nil, fmt.Errorf("failed to get scroll chain abi: %w", err)
	}

	client, err := newL1Client(ctx, l1Client, genesisConfig.Scroll.L1Config.L1ChainId, genesisConfig.Scroll.L1Config.ScrollChainAddress, scrollChainABI)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize l1 client: %w", err)
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

	ctx, cancel := context.WithCancel(ctx)

	service := RollupSyncService{
		ctx:                           ctx,
		cancel:                        cancel,
		client:                        client,
		db:                            db,
		latestProcessedBlock:          latestProcessedBlock,
		scrollChainABI:                scrollChainABI,
		l1CommitBatchEventSignature:   scrollChainABI.Events["CommitBatch"].ID,
		l1RevertBatchEventSignature:   scrollChainABI.Events["RevertBatch"].ID,
		l1FinalizeBatchEventSignature: scrollChainABI.Events["FinalizeBatch"].ID,
		bc:                            bc,
		stack:                         stack,
	}

	return &service, nil
}

func (s *RollupSyncService) Start() {
	if s == nil {
		return
	}

	log.Info("Starting rollup event sync background service", "latest processed block", s.latestProcessedBlock)

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
				s.fetchRollupEvents()
			case <-logTicker.C:
				log.Info("Sync rollup events progress update", "latestProcessedBlock", s.latestProcessedBlock)
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

func (s *RollupSyncService) fetchRollupEvents() {
	latestConfirmed, err := s.client.getLatestFinalizedBlockNumber()
	if err != nil {
		log.Warn("failed to get latest confirmed block number", "err", err)
		return
	}

	log.Trace("Sync service fetch rollup events", "latest processed block", s.latestProcessedBlock, "latest confirmed", latestConfirmed)

	// query in batches
	for from := s.latestProcessedBlock + 1; from <= latestConfirmed; from += defaultFetchBlockRange {
		if s.ctx.Err() != nil {
			log.Info("Context canceled", "reason", s.ctx.Err())
			return
		}

		to := from + defaultFetchBlockRange - 1
		if to > latestConfirmed {
			to = latestConfirmed
		}

		logs, err := s.client.fetchRollupEventsInRange(from, to)
		if err != nil {
			log.Error("failed to fetch rollup events in range", "from block", from, "to block", to, "err", err)
			return
		}

		if err := s.parseAndUpdateRollupEventLogs(logs, to); err != nil {
			log.Error("failed to parse and update rollup event logs", "err", err)
			return
		}

		s.latestProcessedBlock = to
	}
}

func (s *RollupSyncService) parseAndUpdateRollupEventLogs(logs []types.Log, endBlockNumber uint64) error {
	for _, vLog := range logs {
		switch vLog.Topics[0] {
		case s.l1CommitBatchEventSignature:
			event := &L1CommitBatchEvent{}
			if err := UnpackLog(s.scrollChainABI, event, "CommitBatch", vLog); err != nil {
				return fmt.Errorf("failed to unpack commit rollup event log, err: %w", err)
			}
			batchIndex := event.BatchIndex.Uint64()
			log.Trace("found new CommitBatch event", "batch index", batchIndex)

			committedBatchMeta, chunkBlockRanges, err := s.getCommittedBatchMeta(batchIndex, &vLog)
			if err != nil {
				return fmt.Errorf("failed to get chunk ranges, batch index: %v, err: %w", batchIndex, err)
			}
			rawdb.WriteCommittedBatchMeta(s.db, batchIndex, committedBatchMeta)
			rawdb.WriteBatchChunkRanges(s.db, batchIndex, chunkBlockRanges)

		case s.l1RevertBatchEventSignature:
			event := &L1RevertBatchEvent{}
			if err := UnpackLog(s.scrollChainABI, event, "RevertBatch", vLog); err != nil {
				return fmt.Errorf("failed to unpack revert rollup event log, err: %w", err)
			}
			batchIndex := event.BatchIndex.Uint64()
			log.Trace("found new RevertBatch event", "batch index", batchIndex)

			rawdb.DeleteCommittedBatchMeta(s.db, batchIndex)
			rawdb.DeleteBatchChunkRanges(s.db, batchIndex)

		case s.l1FinalizeBatchEventSignature:
			event := &L1FinalizeBatchEvent{}
			if err := UnpackLog(s.scrollChainABI, event, "FinalizeBatch", vLog); err != nil {
				return fmt.Errorf("failed to unpack finalized rollup event log, err: %w", err)
			}
			batchIndex := event.BatchIndex.Uint64()
			log.Trace("found new FinalizeBatch event", "batch index", batchIndex)

			lastFinalizedBatchIndex := rawdb.ReadLastFinalizedBatchIndex(s.db)

			// After darwin, FinalizeBatch event emitted every bundle, which contains multiple batches.
			// Therefore there are a range of finalized batches need to be saved into db.
			//
			// The range logic also applies to the batches before darwin when FinalizeBatch event emitted
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
				committedBatchMeta := rawdb.ReadCommittedBatchMeta(s.db, index)

				chunks, err := s.getLocalChunksForBatch(index)
				if err != nil {
					return fmt.Errorf("failed to get local node info, batch index: %v, err: %w", index, err)
				}

				endBlock, finalizedBatchMeta, err := validateBatch(index, event, parentFinalizedBatchMeta, committedBatchMeta, chunks, s.bc.Config(), s.stack)
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
			rawdb.WriteLastFinalizedBatchIndex(s.db, batchIndex)
			log.Debug("write finalized l2 block number", "batch index", batchIndex, "finalized l2 block height", highestFinalizedBlockNumber)

		default:
			return fmt.Errorf("unknown event, topic: %v, tx hash: %v", vLog.Topics[0].Hex(), vLog.TxHash.Hex())
		}
	}

	// note: the batch updates above are idempotent, if we crash
	// before this line and reexecute the previous steps, we will
	// get the same result.
	rawdb.WriteRollupEventSyncedL1BlockNumber(s.db, endBlockNumber)
	return nil
}

func (s *RollupSyncService) getLocalChunksForBatch(batchIndex uint64) ([]*encoding.Chunk, error) {
	chunkBlockRanges := rawdb.ReadBatchChunkRanges(s.db, batchIndex)
	if len(chunkBlockRanges) == 0 {
		return nil, fmt.Errorf("failed to get batch chunk ranges, empty chunk block ranges")
	}

	endBlockNumber := chunkBlockRanges[len(chunkBlockRanges)-1].EndBlockNumber
	for i := 0; i < defaultMaxRetries; i++ {
		if s.ctx.Err() != nil {
			log.Info("Context canceled", "reason", s.ctx.Err())
			return nil, s.ctx.Err()
		}

		localSyncedBlockHeight := s.bc.CurrentBlock().Number.Uint64()
		if localSyncedBlockHeight >= endBlockNumber {
			break // ready to proceed, exit retry loop
		}

		log.Debug("local node is not synced up to the required block height, waiting for next retry",
			"retries", i+1, "local synced block height", localSyncedBlockHeight, "required end block number", endBlockNumber)
		time.Sleep(defaultGetBlockInRangeRetryDelay)
	}

	localSyncedBlockHeight := s.bc.CurrentBlock().Number.Uint64()
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
			state, err := s.bc.StateAt(block.Root())
			if err != nil {
				return nil, fmt.Errorf("failed to get block state, block: %v, err: %w", block.Hash().Hex(), err)
			}
			withdrawRoot := withdrawtrie.ReadWTRSlot(rcfg.L2MessageQueueAddress, state)
			chunks[i].Blocks[j-cr.StartBlockNumber] = &encoding.Block{
				Header:       block.Header(),
				Transactions: txData,
				WithdrawRoot: withdrawRoot,
			}
		}
	}

	return chunks, nil
}

func (s *RollupSyncService) getCommittedBatchMeta(batchIndex uint64, vLog *types.Log) (*rawdb.CommittedBatchMeta, []*rawdb.ChunkBlockRange, error) {
	if batchIndex == 0 {
		return &rawdb.CommittedBatchMeta{
			Version:             0,
			BlobVersionedHashes: nil,
			ChunkBlockRanges:    []*rawdb.ChunkBlockRange{{StartBlockNumber: 0, EndBlockNumber: 0}},
		}, []*rawdb.ChunkBlockRange{{StartBlockNumber: 0, EndBlockNumber: 0}}, nil
	}

	tx, _, err := s.client.client.TransactionByHash(s.ctx, vLog.TxHash)
	if err != nil {
		log.Debug("failed to get transaction by hash, probably an unindexed transaction, fetching the whole block to get the transaction",
			"tx hash", vLog.TxHash.Hex(), "block number", vLog.BlockNumber, "block hash", vLog.BlockHash.Hex(), "err", err)
		block, err := s.client.client.BlockByHash(s.ctx, vLog.BlockHash)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to get block by hash, block number: %v, block hash: %v, err: %w", vLog.BlockNumber, vLog.BlockHash.Hex(), err)
		}

		if block == nil {
			return nil, nil, fmt.Errorf("failed to get block by hash, block not found, block number: %v, block hash: %v", vLog.BlockNumber, vLog.BlockHash.Hex())
		}

		found := false
		for _, txInBlock := range block.Transactions() {
			if txInBlock.Hash() == vLog.TxHash {
				tx = txInBlock
				found = true
				break
			}
		}
		if !found {
			return nil, nil, fmt.Errorf("transaction not found in the block, tx hash: %v, block number: %v, block hash: %v", vLog.TxHash.Hex(), vLog.BlockNumber, vLog.BlockHash.Hex())
		}
	}

	var commitBatchMeta rawdb.CommittedBatchMeta

	if tx.Type() == types.BlobTxType {
		blobVersionedHashes := tx.BlobHashes()
		if blobVersionedHashes == nil {
			return nil, nil, fmt.Errorf("invalid blob transaction, blob hashes is nil, tx hash: %v", tx.Hash().Hex())
		}
		commitBatchMeta.BlobVersionedHashes = blobVersionedHashes
	}

	version, ranges, err := s.decodeBatchVersionAndChunkBlockRanges(tx.Data())
	if err != nil {
		return nil, nil, fmt.Errorf("failed to decode chunk block ranges, batch index: %v, err: %w", batchIndex, err)
	}

	commitBatchMeta.Version = version
	commitBatchMeta.ChunkBlockRanges = ranges
	return &commitBatchMeta, ranges, nil
}

// decodeBatchVersionAndChunkBlockRanges decodes version and chunks' block ranges in a batch based on the commit batch transaction's calldata.
func (s *RollupSyncService) decodeBatchVersionAndChunkBlockRanges(txData []byte) (uint8, []*rawdb.ChunkBlockRange, error) {
	const methodIDLength = 4
	if len(txData) < methodIDLength {
		return 0, nil, fmt.Errorf("transaction data is too short, length of tx data: %v, minimum length required: %v", len(txData), methodIDLength)
	}

	method, err := s.scrollChainABI.MethodById(txData[:methodIDLength])
	if err != nil {
		return 0, nil, fmt.Errorf("failed to get method by ID, ID: %v, err: %w", txData[:methodIDLength], err)
	}

	values, err := method.Inputs.Unpack(txData[methodIDLength:])
	if err != nil {
		return 0, nil, fmt.Errorf("failed to unpack transaction data using ABI, tx data: %v, err: %w", txData, err)
	}

	if method.Name == "commitBatch" {
		type commitBatchArgs struct {
			Version                uint8
			ParentBatchHeader      []byte
			Chunks                 [][]byte
			SkippedL1MessageBitmap []byte
		}

		var args commitBatchArgs
		if err = method.Inputs.Copy(&args, values); err != nil {
			return 0, nil, fmt.Errorf("failed to decode calldata into commitBatch args, values: %+v, err: %w", values, err)
		}

		chunkRanges, err := decodeBlockRangesFromEncodedChunks(encoding.CodecVersion(args.Version), args.Chunks)
		if err != nil {
			return 0, nil, fmt.Errorf("failed to decode block ranges from encoded chunks, version: %v, chunks: %+v, err: %w", args.Version, args.Chunks, err)
		}

		return args.Version, chunkRanges, nil
	} else if method.Name == "commitBatchWithBlobProof" {
		type commitBatchWithBlobProofArgs struct {
			Version                uint8
			ParentBatchHeader      []byte
			Chunks                 [][]byte
			SkippedL1MessageBitmap []byte
			BlobDataProof          []byte
		}

		var args commitBatchWithBlobProofArgs
		if err = method.Inputs.Copy(&args, values); err != nil {
			return 0, nil, fmt.Errorf("failed to decode calldata into commitBatchWithBlobProofArgs args, values: %+v, err: %w", values, err)
		}

		chunkRanges, err := decodeBlockRangesFromEncodedChunks(encoding.CodecVersion(args.Version), args.Chunks)
		if err != nil {
			return 0, nil, fmt.Errorf("failed to decode block ranges from encoded chunks, version: %v, chunks: %+v, err: %w", args.Version, args.Chunks, err)
		}

		return args.Version, chunkRanges, nil
	}

	return 0, nil, fmt.Errorf("unexpected method name: %v", method.Name)
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
//   - committedBatchMeta: committed batch metadata stored in the database.
//     Can be nil for older client versions that don't store this information.
//   - chunks: slice of chunk data for the current batch
//   - chainCfg: chain configuration to identify the codec version when committedBatchMeta is nil
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
func validateBatch(batchIndex uint64, event *L1FinalizeBatchEvent, parentFinalizedBatchMeta *rawdb.FinalizedBatchMeta, committedBatchMeta *rawdb.CommittedBatchMeta, chunks []*encoding.Chunk, chainCfg *params.ChainConfig, stack *node.Node) (uint64, *rawdb.FinalizedBatchMeta, error) {
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
	batch := &encoding.Batch{
		Index:                      batchIndex,
		TotalL1MessagePoppedBefore: parentFinalizedBatchMeta.TotalL1MessagePopped,
		ParentBatchHash:            parentFinalizedBatchMeta.BatchHash,
		Chunks:                     chunks,
	}

	var codecVersion encoding.CodecVersion
	if committedBatchMeta != nil {
		codecVersion = encoding.CodecVersion(committedBatchMeta.Version)
	} else {
		codecVersion = determineCodecVersion(startBlock.Header.Number, startBlock.Header.Time, chainCfg)
	}

	var localBatchHash common.Hash
	if codecVersion == encoding.CodecV0 {
		daBatch, err := codecv0.NewDABatch(batch)
		if err != nil {
			return 0, nil, fmt.Errorf("failed to create codecv0 DA batch, batch index: %v, err: %w", batchIndex, err)
		}
		localBatchHash = daBatch.Hash()
	} else if codecVersion == encoding.CodecV1 {
		daBatch, err := codecv1.NewDABatch(batch)
		if err != nil {
			return 0, nil, fmt.Errorf("failed to create codecv1 DA batch, batch index: %v, err: %w", batchIndex, err)
		}
		localBatchHash = daBatch.Hash()
	} else if codecVersion == encoding.CodecV2 {
		daBatch, err := codecv2.NewDABatch(batch)
		if err != nil {
			return 0, nil, fmt.Errorf("failed to create codecv2 DA batch, batch index: %v, err: %w", batchIndex, err)
		}
		localBatchHash = daBatch.Hash()
	} else if codecVersion == encoding.CodecV3 {
		daBatch, err := codecv3.NewDABatch(batch)
		if err != nil {
			return 0, nil, fmt.Errorf("failed to create codecv3 DA batch, batch index: %v, err: %w", batchIndex, err)
		}
		localBatchHash = daBatch.Hash()
	} else if codecVersion == encoding.CodecV4 {
		// Check if committedBatchMeta exists, for backward compatibility with older client versions
		if committedBatchMeta == nil {
			return 0, nil, fmt.Errorf("missing committed batch metadata for codecV4, please use the latest client version, batch index: %v", batchIndex)
		}

		// Validate BlobVersionedHashes
		if committedBatchMeta.BlobVersionedHashes == nil || len(committedBatchMeta.BlobVersionedHashes) != 1 {
			return 0, nil, fmt.Errorf("invalid blob hashes, batch index: %v, blob hashes: %v", batchIndex, committedBatchMeta.BlobVersionedHashes)
		}

		// Attempt to create DA batch with compression
		daBatch, err := codecv4.NewDABatch(batch, true)
		if err != nil {
			// If compression fails, try without compression
			log.Warn("failed to create codecv4 DA batch with compress enabling", "batch index", batchIndex, "err", err)
			daBatch, err = codecv4.NewDABatch(batch, false)
			if err != nil {
				return 0, nil, fmt.Errorf("failed to create codecv4 DA batch, batch index: %v, err: %w", batchIndex, err)
			}
		} else if daBatch.BlobVersionedHash != committedBatchMeta.BlobVersionedHashes[0] {
			// Inconsistent blob versioned hash, fallback to uncompressed DA batch
			log.Warn("impossible case: inconsistent blob versioned hash", "batch index", batchIndex, "expected", committedBatchMeta.BlobVersionedHashes[0], "actual", daBatch.BlobVersionedHash)
			daBatch, err = codecv4.NewDABatch(batch, false)
			if err != nil {
				return 0, nil, fmt.Errorf("failed to create codecv4 DA batch, batch index: %v, err: %w", batchIndex, err)
			}
		}

		localBatchHash = daBatch.Hash()
	} else {
		return 0, nil, fmt.Errorf("unsupported codec version: %v", codecVersion)
	}

	localStateRoot := endBlock.Header.Root
	localWithdrawRoot := endBlock.WithdrawRoot

	// Note: If the state root, withdraw root, and batch headers match, this ensures the consistency of blocks and transactions
	// (including skipped transactions) between L1 and L2.
	//
	// Only check when batch index matches the index of the event. This is compatible with both "finalize by batch" and "finalize by bundle":
	// - finalize by batch: check all batches
	// - finalize by bundle: check the last batch, because only one event (containing the info of the last batch) is emitted per bundle
	if batchIndex == event.BatchIndex.Uint64() {
		if localStateRoot != event.StateRoot {
			log.Error("State root mismatch", "batch index", event.BatchIndex.Uint64(), "start block", startBlock.Header.Number.Uint64(), "end block", endBlock.Header.Number.Uint64(), "parent batch hash", parentFinalizedBatchMeta.BatchHash.Hex(), "l1 finalized state root", event.StateRoot.Hex(), "l2 state root", localStateRoot.Hex())
			stack.Close()
			os.Exit(1)
		}

		if localWithdrawRoot != event.WithdrawRoot {
			log.Error("Withdraw root mismatch", "batch index", event.BatchIndex.Uint64(), "start block", startBlock.Header.Number.Uint64(), "end block", endBlock.Header.Number.Uint64(), "parent batch hash", parentFinalizedBatchMeta.BatchHash.Hex(), "l1 finalized withdraw root", event.WithdrawRoot.Hex(), "l2 withdraw root", localWithdrawRoot.Hex())
			stack.Close()
			os.Exit(1)
		}

		// Verify batch hash
		// This check ensures the correctness of all batch hashes in the bundle
		// due to the parent-child relationship between batch hashes
		if localBatchHash != event.BatchHash {
			log.Error("Batch hash mismatch", "batch index", event.BatchIndex.Uint64(), "start block", startBlock.Header.Number.Uint64(), "end block", endBlock.Header.Number.Uint64(), "parent batch hash", parentFinalizedBatchMeta.BatchHash.Hex(), "parent TotalL1MessagePopped", parentFinalizedBatchMeta.TotalL1MessagePopped, "l1 finalized batch hash", event.BatchHash.Hex(), "l2 batch hash", localBatchHash.Hex())
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
		WithdrawRoot:         localWithdrawRoot,
	}
	return endBlock.Header.Number.Uint64(), finalizedBatchMeta, nil
}

// determineCodecVersion determines the codec version based on the block number and chain configuration.
func determineCodecVersion(startBlockNumber *big.Int, startBlockTimestamp uint64, chainCfg *params.ChainConfig) encoding.CodecVersion {
	switch {
	case startBlockNumber.Uint64() == 0 || !chainCfg.IsBernoulli(startBlockNumber):
		return encoding.CodecV0 // codecv0: genesis batch or batches before Bernoulli
	case !chainCfg.IsCurie(startBlockNumber):
		return encoding.CodecV1 // codecv1: batches after Bernoulli and before Curie
	case !chainCfg.IsDarwin(startBlockNumber, startBlockTimestamp):
		return encoding.CodecV2 // codecv2: batches after Curie and before Darwin
	case !chainCfg.IsDarwinV2(startBlockNumber, startBlockTimestamp):
		return encoding.CodecV3 // codecv3: batches after Darwin
	default:
		return encoding.CodecV4 // codecv4: batches after DarwinV2
	}
}

// decodeBlockRangesFromEncodedChunks decodes the provided chunks into a list of block ranges.
func decodeBlockRangesFromEncodedChunks(codecVersion encoding.CodecVersion, chunks [][]byte) ([]*rawdb.ChunkBlockRange, error) {
	var chunkBlockRanges []*rawdb.ChunkBlockRange
	for _, chunk := range chunks {
		if len(chunk) < 1 {
			return nil, fmt.Errorf("invalid chunk, length is less than 1")
		}

		numBlocks := int(chunk[0])

		switch codecVersion {
		case encoding.CodecV0:
			if len(chunk) < 1+numBlocks*60 {
				return nil, fmt.Errorf("invalid chunk byte length, expected: %v, got: %v", 1+numBlocks*60, len(chunk))
			}
			daBlocks := make([]*codecv0.DABlock, numBlocks)
			for i := 0; i < numBlocks; i++ {
				startIdx := 1 + i*60 // add 1 to skip numBlocks byte
				endIdx := startIdx + 60
				daBlocks[i] = &codecv0.DABlock{}
				if err := daBlocks[i].Decode(chunk[startIdx:endIdx]); err != nil {
					return nil, err
				}
			}

			chunkBlockRanges = append(chunkBlockRanges, &rawdb.ChunkBlockRange{
				StartBlockNumber: daBlocks[0].BlockNumber,
				EndBlockNumber:   daBlocks[len(daBlocks)-1].BlockNumber,
			})
		case encoding.CodecV1:
			if len(chunk) != 1+numBlocks*60 {
				return nil, fmt.Errorf("invalid chunk byte length, expected: %v, got: %v", 1+numBlocks*60, len(chunk))
			}
			daBlocks := make([]*codecv1.DABlock, numBlocks)
			for i := 0; i < numBlocks; i++ {
				startIdx := 1 + i*60 // add 1 to skip numBlocks byte
				endIdx := startIdx + 60
				daBlocks[i] = &codecv1.DABlock{}
				if err := daBlocks[i].Decode(chunk[startIdx:endIdx]); err != nil {
					return nil, err
				}
			}

			chunkBlockRanges = append(chunkBlockRanges, &rawdb.ChunkBlockRange{
				StartBlockNumber: daBlocks[0].BlockNumber,
				EndBlockNumber:   daBlocks[len(daBlocks)-1].BlockNumber,
			})
		case encoding.CodecV2:
			if len(chunk) != 1+numBlocks*60 {
				return nil, fmt.Errorf("invalid chunk byte length, expected: %v, got: %v", 1+numBlocks*60, len(chunk))
			}
			daBlocks := make([]*codecv2.DABlock, numBlocks)
			for i := 0; i < numBlocks; i++ {
				startIdx := 1 + i*60 // add 1 to skip numBlocks byte
				endIdx := startIdx + 60
				daBlocks[i] = &codecv2.DABlock{}
				if err := daBlocks[i].Decode(chunk[startIdx:endIdx]); err != nil {
					return nil, err
				}
			}

			chunkBlockRanges = append(chunkBlockRanges, &rawdb.ChunkBlockRange{
				StartBlockNumber: daBlocks[0].BlockNumber,
				EndBlockNumber:   daBlocks[len(daBlocks)-1].BlockNumber,
			})
		case encoding.CodecV3:
			if len(chunk) != 1+numBlocks*60 {
				return nil, fmt.Errorf("invalid chunk byte length, expected: %v, got: %v", 1+numBlocks*60, len(chunk))
			}
			daBlocks := make([]*codecv3.DABlock, numBlocks)
			for i := 0; i < numBlocks; i++ {
				startIdx := 1 + i*60 // add 1 to skip numBlocks byte
				endIdx := startIdx + 60
				daBlocks[i] = &codecv3.DABlock{}
				if err := daBlocks[i].Decode(chunk[startIdx:endIdx]); err != nil {
					return nil, err
				}
			}

			chunkBlockRanges = append(chunkBlockRanges, &rawdb.ChunkBlockRange{
				StartBlockNumber: daBlocks[0].BlockNumber,
				EndBlockNumber:   daBlocks[len(daBlocks)-1].BlockNumber,
			})
		case encoding.CodecV4:
			if len(chunk) != 1+numBlocks*60 {
				return nil, fmt.Errorf("invalid chunk byte length, expected: %v, got: %v", 1+numBlocks*60, len(chunk))
			}
			daBlocks := make([]*codecv4.DABlock, numBlocks)
			for i := 0; i < numBlocks; i++ {
				startIdx := 1 + i*60 // add 1 to skip numBlocks byte
				endIdx := startIdx + 60
				daBlocks[i] = &codecv4.DABlock{}
				if err := daBlocks[i].Decode(chunk[startIdx:endIdx]); err != nil {
					return nil, err
				}
			}

			chunkBlockRanges = append(chunkBlockRanges, &rawdb.ChunkBlockRange{
				StartBlockNumber: daBlocks[0].BlockNumber,
				EndBlockNumber:   daBlocks[len(daBlocks)-1].BlockNumber,
			})
		default:
			return nil, fmt.Errorf("unexpected batch version %v", codecVersion)
		}
	}
	return chunkBlockRanges, nil
}
