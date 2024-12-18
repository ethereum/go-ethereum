package da

import (
	"context"
	"errors"
	"fmt"

	"github.com/scroll-tech/da-codec/encoding"

	"github.com/scroll-tech/go-ethereum/accounts/abi"
	"github.com/scroll-tech/go-ethereum/common"
	"github.com/scroll-tech/go-ethereum/core/types"
	"github.com/scroll-tech/go-ethereum/ethdb"
	"github.com/scroll-tech/go-ethereum/log"
	"github.com/scroll-tech/go-ethereum/rollup/da_syncer/blob_client"
	"github.com/scroll-tech/go-ethereum/rollup/da_syncer/serrors"
	"github.com/scroll-tech/go-ethereum/rollup/rollup_sync_service"
)

const (
	callDataBlobSourceFetchBlockRange  uint64 = 500
	commitBatchEventName                      = "CommitBatch"
	revertBatchEventName                      = "RevertBatch"
	finalizeBatchEventName                    = "FinalizeBatch"
	commitBatchMethodName                     = "commitBatch"
	commitBatchWithBlobProofMethodName        = "commitBatchWithBlobProof"

	// the length of method ID at the beginning of transaction data
	methodIDLength = 4
)

var (
	ErrSourceExhausted = errors.New("data source has been exhausted")
)

type CalldataBlobSource struct {
	ctx                           context.Context
	l1Client                      *rollup_sync_service.L1Client
	blobClient                    blob_client.BlobClient
	l1height                      uint64
	scrollChainABI                *abi.ABI
	l1CommitBatchEventSignature   common.Hash
	l1RevertBatchEventSignature   common.Hash
	l1FinalizeBatchEventSignature common.Hash
	db                            ethdb.Database

	l1Finalized uint64
}

func NewCalldataBlobSource(ctx context.Context, l1height uint64, l1Client *rollup_sync_service.L1Client, blobClient blob_client.BlobClient, db ethdb.Database) (*CalldataBlobSource, error) {
	scrollChainABI, err := rollup_sync_service.ScrollChainMetaData.GetAbi()
	if err != nil {
		return nil, fmt.Errorf("failed to get scroll chain abi: %w", err)
	}
	return &CalldataBlobSource{
		ctx:                           ctx,
		l1Client:                      l1Client,
		blobClient:                    blobClient,
		l1height:                      l1height,
		scrollChainABI:                scrollChainABI,
		l1CommitBatchEventSignature:   scrollChainABI.Events[commitBatchEventName].ID,
		l1RevertBatchEventSignature:   scrollChainABI.Events[revertBatchEventName].ID,
		l1FinalizeBatchEventSignature: scrollChainABI.Events[finalizeBatchEventName].ID,
		db:                            db,
	}, nil
}

func (ds *CalldataBlobSource) NextData() (Entries, error) {
	var err error
	to := ds.l1height + callDataBlobSourceFetchBlockRange

	// If there's not enough finalized blocks to request up to, we need to query finalized block number.
	// Otherwise, we know that there's more finalized blocks than we want to request up to
	// -> no need to query finalized block number
	if to > ds.l1Finalized {
		ds.l1Finalized, err = ds.l1Client.GetLatestFinalizedBlockNumber()
		if err != nil {
			return nil, serrors.NewTemporaryError(fmt.Errorf("failed to query GetLatestFinalizedBlockNumber, error: %v", err))
		}
		// make sure we don't request more than finalized blocks
		to = min(to, ds.l1Finalized)
	}

	if ds.l1height > to {
		return nil, ErrSourceExhausted
	}

	logs, err := ds.l1Client.FetchRollupEventsInRange(ds.l1height, to)
	if err != nil {
		return nil, serrors.NewTemporaryError(fmt.Errorf("cannot get events, l1height: %d, error: %v", ds.l1height, err))
	}
	da, err := ds.processLogsToDA(logs)
	if err != nil {
		return nil, serrors.NewTemporaryError(fmt.Errorf("failed to process logs to DA, error: %v", err))
	}

	ds.l1height = to + 1
	return da, nil
}

func (ds *CalldataBlobSource) L1Height() uint64 {
	return ds.l1height
}

func (ds *CalldataBlobSource) processLogsToDA(logs []types.Log) (Entries, error) {
	var entries Entries
	var entry Entry
	var err error

	for _, vLog := range logs {
		switch vLog.Topics[0] {
		case ds.l1CommitBatchEventSignature:
			event := &rollup_sync_service.L1CommitBatchEvent{}
			if err = rollup_sync_service.UnpackLog(ds.scrollChainABI, event, commitBatchEventName, vLog); err != nil {
				return nil, fmt.Errorf("failed to unpack commit rollup event log, err: %w", err)
			}

			batchIndex := event.BatchIndex.Uint64()
			log.Trace("found new CommitBatch event", "batch index", batchIndex)

			if entry, err = ds.getCommitBatchDA(batchIndex, &vLog); err != nil {
				return nil, fmt.Errorf("failed to get commit batch da: %v, err: %w", batchIndex, err)
			}

		case ds.l1RevertBatchEventSignature:
			event := &rollup_sync_service.L1RevertBatchEvent{}
			if err = rollup_sync_service.UnpackLog(ds.scrollChainABI, event, revertBatchEventName, vLog); err != nil {
				return nil, fmt.Errorf("failed to unpack revert rollup event log, err: %w", err)
			}

			batchIndex := event.BatchIndex.Uint64()
			log.Trace("found new RevertBatchType event", "batch index", batchIndex)
			entry = NewRevertBatch(batchIndex)

		case ds.l1FinalizeBatchEventSignature:
			event := &rollup_sync_service.L1FinalizeBatchEvent{}
			if err = rollup_sync_service.UnpackLog(ds.scrollChainABI, event, finalizeBatchEventName, vLog); err != nil {
				return nil, fmt.Errorf("failed to unpack finalized rollup event log, err: %w", err)
			}

			batchIndex := event.BatchIndex.Uint64()
			log.Trace("found new FinalizeBatchType event", "batch index", event.BatchIndex.Uint64())
			entry = NewFinalizeBatch(batchIndex)

		default:
			return nil, fmt.Errorf("unknown event, topic: %v, tx hash: %v", vLog.Topics[0].Hex(), vLog.TxHash.Hex())
		}

		entries = append(entries, entry)
	}
	return entries, nil
}

type commitBatchArgs struct {
	Version                uint8
	ParentBatchHeader      []byte
	Chunks                 [][]byte
	SkippedL1MessageBitmap []byte
}

func newCommitBatchArgs(method *abi.Method, values []interface{}) (*commitBatchArgs, error) {
	var args commitBatchArgs
	err := method.Inputs.Copy(&args, values)
	return &args, err
}

func newCommitBatchArgsFromCommitBatchWithProof(method *abi.Method, values []interface{}) (*commitBatchArgs, error) {
	var args commitBatchWithBlobProofArgs
	err := method.Inputs.Copy(&args, values)
	if err != nil {
		return nil, err
	}
	return &commitBatchArgs{
		Version:                args.Version,
		ParentBatchHeader:      args.ParentBatchHeader,
		Chunks:                 args.Chunks,
		SkippedL1MessageBitmap: args.SkippedL1MessageBitmap,
	}, nil
}

type commitBatchWithBlobProofArgs struct {
	Version                uint8
	ParentBatchHeader      []byte
	Chunks                 [][]byte
	SkippedL1MessageBitmap []byte
	BlobDataProof          []byte
}

func (ds *CalldataBlobSource) getCommitBatchDA(batchIndex uint64, vLog *types.Log) (Entry, error) {
	if batchIndex == 0 {
		return NewCommitBatchDAV0Empty(), nil
	}

	txData, err := ds.l1Client.FetchTxData(vLog)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch tx data, tx hash: %v, err: %w", vLog.TxHash.Hex(), err)
	}
	if len(txData) < methodIDLength {
		return nil, fmt.Errorf("transaction data is too short, length of tx data: %v, minimum length required: %v", len(txData), methodIDLength)
	}

	method, err := ds.scrollChainABI.MethodById(txData[:methodIDLength])
	if err != nil {
		return nil, fmt.Errorf("failed to get method by ID, ID: %v, err: %w", txData[:methodIDLength], err)
	}
	values, err := method.Inputs.Unpack(txData[methodIDLength:])
	if err != nil {
		return nil, fmt.Errorf("failed to unpack transaction data using ABI, tx data: %v, err: %w", txData, err)
	}
	if method.Name == commitBatchMethodName {
		args, err := newCommitBatchArgs(method, values)
		if err != nil {
			return nil, fmt.Errorf("failed to decode calldata into commitBatch args, values: %+v, err: %w", values, err)
		}
		codecVersion := encoding.CodecVersion(args.Version)
		codec, err := encoding.CodecFromVersion(codecVersion)
		if err != nil {
			return nil, fmt.Errorf("unsupported codec version: %v, batch index: %v, err: %w", codecVersion, batchIndex, err)
		}
		switch args.Version {
		case 0:
			return NewCommitBatchDAV0(ds.db, codec, args.Version, batchIndex, args.ParentBatchHeader, args.Chunks, args.SkippedL1MessageBitmap, vLog.BlockNumber)
		case 1, 2:
			return NewCommitBatchDAWithBlob(ds.ctx, ds.db, codec, ds.l1Client, ds.blobClient, vLog, args.Version, batchIndex, args.ParentBatchHeader, args.Chunks, args.SkippedL1MessageBitmap)
		default:
			return nil, fmt.Errorf("failed to decode DA, codec version is unknown: codec version: %d", args.Version)
		}
	} else if method.Name == commitBatchWithBlobProofMethodName {
		args, err := newCommitBatchArgsFromCommitBatchWithProof(method, values)
		if err != nil {
			return nil, fmt.Errorf("failed to decode calldata into commitBatch args, values: %+v, err: %w", values, err)
		}
		codecVersion := encoding.CodecVersion(args.Version)
		codec, err := encoding.CodecFromVersion(codecVersion)
		if err != nil {
			return nil, fmt.Errorf("unsupported codec version: %v, batch index: %v, err: %w", codecVersion, batchIndex, err)
		}
		switch args.Version {
		case 3, 4:
			return NewCommitBatchDAWithBlob(ds.ctx, ds.db, codec, ds.l1Client, ds.blobClient, vLog, args.Version, batchIndex, args.ParentBatchHeader, args.Chunks, args.SkippedL1MessageBitmap)
		default:
			return nil, fmt.Errorf("failed to decode DA, codec version is unknown: codec version: %d", args.Version)
		}
	}

	return nil, fmt.Errorf("unknown method name: %s", method.Name)
}
