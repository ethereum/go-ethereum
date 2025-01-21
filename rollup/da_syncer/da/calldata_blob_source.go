package da

import (
	"context"
	"errors"
	"fmt"

	"github.com/scroll-tech/da-codec/encoding"

	"github.com/scroll-tech/go-ethereum/accounts/abi"
	"github.com/scroll-tech/go-ethereum/ethdb"
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

func (ds *CalldataBlobSource) L1Height() uint64 {
	return ds.l1Height
}

func (ds *CalldataBlobSource) processRollupEventsToDA(rollupEvents l1.RollupEvents) (Entries, error) {
	var entries Entries
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
			if entry, err = ds.getCommitBatchDA(commitEvent); err != nil {
				return nil, fmt.Errorf("failed to get commit batch da: %v, err: %w", rollupEvent.BatchIndex().Uint64(), err)
			}

		case l1.RevertEventType:
			entry = NewRevertBatch(rollupEvent.BatchIndex().Uint64())

		case l1.FinalizeEventType:
			entry = NewFinalizeBatch(rollupEvent.BatchIndex().Uint64())

		default:
			return nil, fmt.Errorf("unknown rollup event, type: %v", rollupEvent.Type())
		}

		entries = append(entries, entry)
	}
	return entries, nil
}

func (ds *CalldataBlobSource) getCommitBatchDA(commitEvent *l1.CommitBatchEvent) (Entry, error) {
	if commitEvent.BatchIndex().Uint64() == 0 {
		return NewCommitBatchDAV0Empty(), nil
	}

	args, err := ds.l1Reader.FetchCommitTxData(commitEvent)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch commit tx data of batch %d, tx hash: %v, err: %w", commitEvent.BatchIndex().Uint64(), commitEvent.TxHash().Hex(), err)
	}

	codec, err := encoding.CodecFromVersion(encoding.CodecVersion(args.Version))
	if err != nil {
		return nil, fmt.Errorf("unsupported codec version: %v, batch index: %v, err: %w", args.Version, commitEvent.BatchIndex().Uint64(), err)
	}

	switch codec.Version() {
	case 0:
		return NewCommitBatchDAV0(ds.db, codec, commitEvent, args.ParentBatchHeader, args.Chunks, args.SkippedL1MessageBitmap)
	case 1, 2, 3, 4:
		return NewCommitBatchDAWithBlob(ds.ctx, ds.db, ds.l1Reader, ds.blobClient, codec, commitEvent, args.ParentBatchHeader, args.Chunks, args.SkippedL1MessageBitmap)
	default:
		return nil, fmt.Errorf("failed to decode DA, codec version is unknown: codec version: %d", args.Version)
	}
}
