package da

import (
	"context"
	"errors"
	"fmt"

	"github.com/scroll-tech/da-codec/encoding"

	"github.com/scroll-tech/go-ethereum/accounts/abi"
	"github.com/scroll-tech/go-ethereum/common"
	"github.com/scroll-tech/go-ethereum/ethdb"
	"github.com/scroll-tech/go-ethereum/log"
	"github.com/scroll-tech/go-ethereum/rollup/da_syncer/blob_client"
	"github.com/scroll-tech/go-ethereum/rollup/da_syncer/serrors"
	"github.com/scroll-tech/go-ethereum/rollup/l1"
)

const (
	callDataBlobSourceFetchBlockRange uint64 = 500
)

var (
	ErrSourceExhausted = errors.New("data source has been exhausted")
)

type CalldataBlobSource struct {
	ctx            context.Context
	l1Reader       *l1.Reader
	blobClient     blob_client.BlobClient
	l1Height       uint64
	scrollChainABI *abi.ABI
	db             ethdb.Database

	l1Finalized uint64
}

func NewCalldataBlobSource(ctx context.Context, l1height uint64, l1Reader *l1.Reader, blobClient blob_client.BlobClient, db ethdb.Database) (*CalldataBlobSource, error) {
	scrollChainABI, err := l1.ScrollChainMetaData.GetAbi()
	if err != nil {
		return nil, fmt.Errorf("failed to get scroll chain abi: %w", err)
	}
	return &CalldataBlobSource{
		ctx:            ctx,
		l1Reader:       l1Reader,
		blobClient:     blobClient,
		l1Height:       l1height,
		scrollChainABI: scrollChainABI,
		db:             db,
	}, nil
}

func (ds *CalldataBlobSource) NextData() (Entries, error) {
	var err error
	to := ds.l1Height + callDataBlobSourceFetchBlockRange

	// If there's not enough finalized blocks to request up to, we need to query finalized block number.
	// Otherwise, we know that there's more finalized blocks than we want to request up to
	// -> no need to query finalized block number
	if to > ds.l1Finalized {
		ds.l1Finalized, err = ds.l1Reader.GetLatestFinalizedBlockNumber()
		if err != nil {
			return nil, serrors.NewTemporaryError(fmt.Errorf("failed to query GetLatestFinalizedBlockNumber, error: %v", err))
		}
		// make sure we don't request more than finalized blocks
		to = min(to, ds.l1Finalized)
	}

	log.Debug("Fetching rollup events", "from", ds.l1Height, "to", to, "finalized", ds.l1Finalized)

	if ds.l1Height > to {
		return nil, ErrSourceExhausted
	}

	rollupEvents, err := ds.l1Reader.FetchRollupEventsInRange(ds.l1Height, to)
	if err != nil {
		return nil, serrors.NewTemporaryError(fmt.Errorf("cannot get rollup events, l1Height: %d, error: %v", ds.l1Height, err))
	}
	da, err := ds.processRollupEventsToDA(rollupEvents)
	if err != nil {
		return nil, serrors.NewTemporaryError(fmt.Errorf("failed to process rollup events to DA, error: %v", err))
	}

	ds.l1Height = to + 1
	return da, nil
}

func (ds *CalldataBlobSource) SetL1Height(l1Height uint64) {
	ds.l1Height = l1Height
}

func (ds *CalldataBlobSource) L1Height() uint64 {
	return ds.l1Height
}

func (ds *CalldataBlobSource) L1Finalized() uint64 {
	return ds.l1Finalized
}

func (ds *CalldataBlobSource) processRollupEventsToDA(rollupEvents l1.RollupEvents) (Entries, error) {
	var entries Entries
	// we keep track of the last commit transaction hash, so we can process all events created in the same tx together.
	// if we have a different commit transaction, we need to create a new commit batch DA.
	var lastCommitTransactionHash common.Hash
	// we keep track of the commit events created in the same tx, so we can process them together.
	var lastCommitEvents []*l1.CommitBatchEvent

	// getAndAppendCommitBatchDA is a helper function that gets the commit batch DA for the last commit events and appends it to the entries list.
	// It also resets the last commit events and last commit transaction hash.
	// This is necessary because we need to process all events created in the same tx together.
	// However, we only know all events created in the same tx when we see a different commit transaction (next iteration of the loop).
	// Therefore, we need to process the last commit events when we see a different event (finalize, revert) or commit transaction (or when we reach the end of the rollup events).
	getAndAppendCommitBatchDA := func() error {
		commitBatchDAEntries, err := ds.getCommitBatchDA(lastCommitEvents)
		if err != nil {
			return fmt.Errorf("failed to get commit batch da: %v, err: %w", lastCommitEvents[0].BatchIndex().Uint64(), err)
		}

		entries = append(entries, commitBatchDAEntries...)
		lastCommitEvents = nil
		lastCommitTransactionHash = common.Hash{}

		return nil
	}

	var entry Entry
	var err error
	for _, rollupEvent := range rollupEvents {
		switch rollupEvent.Type() {
		case l1.CommitEventType:
			commitEvent, ok := rollupEvent.(*l1.CommitBatchEvent)
			// this should never happen because we just check event type
			if !ok {
				return nil, fmt.Errorf("unexpected type of rollup event: %T", rollupEvent)
			}

			// if this is a different commit transaction, we need to create a new DA
			if lastCommitTransactionHash != commitEvent.TxHash() && len(lastCommitEvents) > 0 {
				if err = getAndAppendCommitBatchDA(); err != nil {
					return nil, fmt.Errorf("failed to get and append commit batch DA: %w", err)
				}
			}

			// add commit event to the list of previous commit events, so we can process events created in the same tx together
			lastCommitTransactionHash = commitEvent.TxHash()
			lastCommitEvents = append(lastCommitEvents, commitEvent)
		case l1.RevertEventType:
			// if we have any previous commit events, we need to create a new DA before processing the revert event
			if len(lastCommitEvents) > 0 {
				if err = getAndAppendCommitBatchDA(); err != nil {
					return nil, fmt.Errorf("failed to get and append commit batch DA: %w", err)
				}
			}

			revertEvent, ok := rollupEvent.(*l1.RevertBatchEvent)
			// this should never happen because we just check event type
			if !ok {
				return nil, fmt.Errorf("unexpected type of rollup event: %T", rollupEvent)
			}

			entry = NewRevertBatch(revertEvent)
			entries = append(entries, entry)
		case l1.FinalizeEventType:
			// if we have any previous commit events, we need to create a new DA before processing the finalized event
			if len(lastCommitEvents) > 0 {
				if err = getAndAppendCommitBatchDA(); err != nil {
					return nil, fmt.Errorf("failed to get and append commit batch DA: %w", err)
				}
			}

			finalizeEvent, ok := rollupEvent.(*l1.FinalizeBatchEvent)
			// this should never happen because we just check event type
			if !ok {
				return nil, fmt.Errorf("unexpected type of rollup event: %T", rollupEvent)
			}

			entry = NewFinalizeBatch(finalizeEvent)
			entries = append(entries, entry)
		default:
			return nil, fmt.Errorf("unknown rollup event, type: %v", rollupEvent.Type())
		}
	}

	// if we have any previous commit events, we need to process them before returning
	if len(lastCommitEvents) > 0 {
		if err = getAndAppendCommitBatchDA(); err != nil {
			return nil, fmt.Errorf("failed to get and append commit batch DA: %w", err)
		}
	}

	return entries, nil
}

func (ds *CalldataBlobSource) getCommitBatchDA(commitEvents []*l1.CommitBatchEvent) (Entries, error) {
	if len(commitEvents) == 0 {
		return nil, fmt.Errorf("commit events are empty")
	}

	if commitEvents[0].BatchIndex().Uint64() == 0 {
		return Entries{NewCommitBatchDAV0Empty(commitEvents[0])}, nil
	}

	firstCommitEvent := commitEvents[0]
	args, err := ds.l1Reader.FetchCommitTxData(firstCommitEvent)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch commit tx data of batch %d, tx hash: %v, err: %w", firstCommitEvent.BatchIndex().Uint64(), firstCommitEvent.TxHash().Hex(), err)
	}

	blockHeader, err := ds.l1Reader.FetchBlockHeaderByNumber(firstCommitEvent.BlockNumber())
	if err != nil {
		return nil, fmt.Errorf("failed to get header by number, err: %w", err)
	}

	codec, err := encoding.CodecFromVersion(encoding.CodecVersion(args.Version))
	if err != nil {
		return nil, fmt.Errorf("unsupported codec version: %v, batch index: %v, err: %w", args.Version, firstCommitEvent.BatchIndex().Uint64(), err)
	}

	var entries Entries
	var entry Entry
	var previousEvent *l1.CommitBatchEvent
	for i, commitEvent := range commitEvents {
		// sanity check commit events from batches submitted in the same L1 transaction
		if commitEvent.TxHash() != firstCommitEvent.TxHash() {
			return nil, fmt.Errorf("commit events have different tx hashes, batch index: %d, tx: %s - batch index: %d, tx: %s", firstCommitEvent.BatchIndex().Uint64(), firstCommitEvent.TxHash().Hex(), commitEvent.BatchIndex().Uint64(), commitEvent.TxHash().Hex())
		}
		if commitEvent.BlockNumber() != firstCommitEvent.BlockNumber() {
			return nil, fmt.Errorf("commit events have different block numbers, batch index: %d, block number: %d - batch index: %d, block number: %d", firstCommitEvent.BatchIndex().Uint64(), firstCommitEvent.BlockNumber(), commitEvent.BatchIndex().Uint64(), commitEvent.BlockNumber())
		}
		if commitEvent.BlockHash() != firstCommitEvent.BlockHash() {
			return nil, fmt.Errorf("commit events have different block hashes, batch index: %d, hash: %s - batch index: %d, hash: %s", firstCommitEvent.BatchIndex().Uint64(), firstCommitEvent.BlockHash().Hex(), commitEvent.BatchIndex().Uint64(), commitEvent.BlockHash().Hex())
		}
		if previousEvent != nil && commitEvent.BatchIndex().Uint64() != previousEvent.BatchIndex().Uint64()+1 {
			return nil, fmt.Errorf("commit events are not in sequence, batch index: %d, hash: %s - previous batch index: %d, hash: %s", commitEvent.BatchIndex().Uint64(), commitEvent.BatchHash().Hex(), previousEvent.BatchIndex().Uint64(), previousEvent.BatchHash().Hex())
		}

		switch codec.Version() {
		case 0:
			if entry, err = NewCommitBatchDAV0(ds.db, codec, commitEvent, args.ParentBatchHeader, args.Chunks, args.SkippedL1MessageBitmap); err != nil {
				return nil, fmt.Errorf("failed to decode DA, batch index: %d, err: %w", commitEvent.BatchIndex().Uint64(), err)
			}
		case 1, 2, 3, 4, 5, 6:
			if entry, err = NewCommitBatchDAV1(ds.ctx, ds.db, ds.blobClient, codec, commitEvent, args.ParentBatchHeader, args.Chunks, args.SkippedL1MessageBitmap, args.BlobHashes, blockHeader.Time); err != nil {
				return nil, fmt.Errorf("failed to decode DA, batch index: %d, err: %w", commitEvent.BatchIndex().Uint64(), err)
			}
		default: // CodecVersion 7 and above
			if i >= len(args.BlobHashes) {
				return nil, fmt.Errorf("not enough blob hashes for commit transaction: %s, index in tx: %d, batch index: %d, hash: %s", firstCommitEvent.TxHash(), i, commitEvent.BatchIndex().Uint64(), commitEvent.BatchHash().Hex())
			}
			blobHash := args.BlobHashes[i]

			var parentBatchHash common.Hash
			if previousEvent == nil {
				parentBatchHash = common.BytesToHash(args.ParentBatchHeader)
			} else {
				parentBatchHash = previousEvent.BatchHash()
			}

			if entry, err = NewCommitBatchDAV7(ds.ctx, ds.db, ds.blobClient, codec, commitEvent, blobHash, parentBatchHash, blockHeader.Time); err != nil {
				return nil, fmt.Errorf("failed to decode DA, batch index: %d, err: %w", commitEvent.BatchIndex().Uint64(), err)
			}
		}

		previousEvent = commitEvent
		entries = append(entries, entry)
	}

	return entries, nil
}
