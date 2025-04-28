package l1

import (
	"context"
	"errors"
	"fmt"
	"math/big"

	"github.com/scroll-tech/da-codec/encoding"

	"github.com/scroll-tech/go-ethereum"
	"github.com/scroll-tech/go-ethereum/accounts/abi"
	"github.com/scroll-tech/go-ethereum/common"
	"github.com/scroll-tech/go-ethereum/core/types"
	"github.com/scroll-tech/go-ethereum/log"
	"github.com/scroll-tech/go-ethereum/rpc"
)

const (
	commitBatchEventName      = "CommitBatch"
	revertBatchV0EventName    = "RevertBatch"
	revertBatchV7EventName    = "RevertBatch0"
	finalizeBatchEventName    = "FinalizeBatch"
	nextUnfinalizedQueueIndex = "nextUnfinalizedQueueIndex"
	lastFinalizedBatchIndex   = "lastFinalizedBatchIndex"
	finalizedStateRoots       = "finalizedStateRoots"

	defaultRollupEventsFetchBlockRange = 100
)

type Reader struct {
	ctx    context.Context
	config Config
	client Client

	scrollChainABI                *abi.ABI
	l1MessageQueueABI             *abi.ABI
	l1CommitBatchEventSignature   common.Hash
	l1RevertBatchEventV0Signature common.Hash
	l1RevertBatchEventV7Signature common.Hash
	l1FinalizeBatchEventSignature common.Hash
}

// Config is the configuration parameters of data availability syncing.
type Config struct {
	ScrollChainAddress    common.Address // address of ScrollChain contract
	L1MessageQueueAddress common.Address // address of L1MessageQueue contract
}

// NewReader initializes a new Reader instance
func NewReader(ctx context.Context, config Config, l1Client Client) (*Reader, error) {
	if config.ScrollChainAddress == (common.Address{}) {
		return nil, errors.New("must pass non-zero scrollChainAddress to L1Client")
	}

	if config.L1MessageQueueAddress == (common.Address{}) {
		return nil, errors.New("must pass non-zero l1MessageQueueAddress to L1Client")
	}

	reader := Reader{
		ctx:    ctx,
		config: config,
		client: l1Client,

		scrollChainABI:                ScrollChainABI,
		l1MessageQueueABI:             L1MessageQueueABIManual,
		l1CommitBatchEventSignature:   ScrollChainABI.Events[commitBatchEventName].ID,
		l1RevertBatchEventV0Signature: ScrollChainABI.Events[revertBatchV0EventName].ID,
		l1RevertBatchEventV7Signature: ScrollChainABI.Events[revertBatchV7EventName].ID,
		l1FinalizeBatchEventSignature: ScrollChainABI.Events[finalizeBatchEventName].ID,
	}

	return &reader, nil
}

func (r *Reader) NextUnfinalizedL1MessageQueueIndex(blockNumber uint64) (uint64, error) {
	data, err := r.l1MessageQueueABI.Pack(nextUnfinalizedQueueIndex)
	if err != nil {
		return 0, fmt.Errorf("failed to pack %s: %w", nextUnfinalizedQueueIndex, err)
	}

	result, err := r.client.CallContract(r.ctx, ethereum.CallMsg{
		To:   &r.config.L1MessageQueueAddress,
		Data: data,
	}, new(big.Int).SetUint64(blockNumber))
	if err != nil {
		return 0, fmt.Errorf("failed to call %s: %w", nextUnfinalizedQueueIndex, err)
	}

	var parsedResult *big.Int
	if err = r.l1MessageQueueABI.UnpackIntoInterface(&parsedResult, nextUnfinalizedQueueIndex, result); err != nil {
		return 0, fmt.Errorf("failed to unpack result: %w", err)
	}

	next := parsedResult.Uint64()
	return next, nil
}

func (r *Reader) LatestFinalizedBatchIndex(blockNumber uint64) (uint64, error) {
	data, err := r.scrollChainABI.Pack(lastFinalizedBatchIndex)
	if err != nil {
		return 0, fmt.Errorf("failed to pack %s: %w", lastFinalizedBatchIndex, err)
	}

	result, err := r.client.CallContract(r.ctx, ethereum.CallMsg{
		To:   &r.config.ScrollChainAddress,
		Data: data,
	}, new(big.Int).SetUint64(blockNumber))
	if err != nil {
		return 0, fmt.Errorf("failed to call %s: %w", lastFinalizedBatchIndex, err)
	}

	var parsedResult *big.Int
	if err = r.scrollChainABI.UnpackIntoInterface(&parsedResult, lastFinalizedBatchIndex, result); err != nil {
		return 0, fmt.Errorf("failed to unpack result: %w", err)
	}

	return parsedResult.Uint64(), nil
}

func (r *Reader) GetFinalizedStateRootByBatchIndex(blockNumber uint64, batchIndex uint64) (common.Hash, error) {
	data, err := r.scrollChainABI.Pack(finalizedStateRoots, big.NewInt(int64(batchIndex)))
	if err != nil {
		return common.Hash{}, fmt.Errorf("failed to pack %s: %w", finalizedStateRoots, err)
	}

	result, err := r.client.CallContract(r.ctx, ethereum.CallMsg{
		To:   &r.config.ScrollChainAddress,
		Data: data,
	}, new(big.Int).SetUint64(blockNumber))
	if err != nil {
		return common.Hash{}, fmt.Errorf("failed to call %s: %w", finalizedStateRoots, err)
	}

	var parsedResult common.Hash
	if err = r.scrollChainABI.UnpackIntoInterface(&parsedResult, finalizedStateRoots, result); err != nil {
		return common.Hash{}, fmt.Errorf("failed to unpack result: %w", err)
	}

	return parsedResult, nil
}

// GetLatestFinalizedBlockNumber fetches the block number of the latest finalized block from the L1 chain.
func (r *Reader) GetLatestFinalizedBlockNumber() (uint64, error) {
	header, err := r.client.HeaderByNumber(r.ctx, big.NewInt(int64(rpc.FinalizedBlockNumber)))
	if err != nil {
		return 0, err
	}
	if !header.Number.IsInt64() {
		return 0, fmt.Errorf("received unexpected block number in L1Client: %v", header.Number)
	}
	return header.Number.Uint64(), nil
}

// FetchBlockHeaderByNumber fetches the block header by number
func (r *Reader) FetchBlockHeaderByNumber(blockNumber uint64) (*types.Header, error) {
	return r.client.HeaderByNumber(r.ctx, big.NewInt(int64(blockNumber)))
}

// FetchTxData fetches tx data corresponding to given event log
func (r *Reader) FetchTxData(txHash, blockHash common.Hash) ([]byte, error) {
	tx, err := r.fetchTx(txHash, blockHash)
	if err != nil {
		return nil, err
	}

	return tx.Data(), nil
}

// FetchTxBlobHashes fetches tx blob hash corresponding to given event log
func (r *Reader) FetchTxBlobHashes(txHash, blockHash common.Hash) ([]common.Hash, error) {
	tx, err := r.fetchTx(txHash, blockHash)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch tx, tx hash: %v, block hash: %v, err: %w", txHash.Hex(), blockHash.Hex(), err)
	}

	blobHashes := tx.BlobHashes()
	if len(blobHashes) == 0 {
		return nil, fmt.Errorf("transaction does not contain any blobs, tx hash: %v", txHash.Hex())
	}

	return blobHashes, nil
}

// FetchRollupEventsInRange retrieves and parses commit/revert/finalize rollup events between block numbers: [from, to].
func (r *Reader) FetchRollupEventsInRange(from, to uint64) (RollupEvents, error) {
	log.Trace("L1Client fetchRollupEventsInRange", "fromBlock", from, "toBlock", to)
	var logs []types.Log

	err := queryInBatches(r.ctx, from, to, defaultRollupEventsFetchBlockRange, func(from, to uint64) (bool, error) {
		query := ethereum.FilterQuery{
			FromBlock: big.NewInt(int64(from)), // inclusive
			ToBlock:   big.NewInt(int64(to)),   // inclusive
			Addresses: []common.Address{
				r.config.ScrollChainAddress,
			},
			Topics: make([][]common.Hash, 1),
		}
		query.Topics[0] = make([]common.Hash, 4)
		query.Topics[0][0] = r.l1CommitBatchEventSignature
		query.Topics[0][1] = r.l1RevertBatchEventV0Signature
		query.Topics[0][2] = r.l1RevertBatchEventV7Signature
		query.Topics[0][3] = r.l1FinalizeBatchEventSignature

		logsBatch, err := r.client.FilterLogs(r.ctx, query)
		if err != nil {
			return false, fmt.Errorf("failed to filter logs, err: %w", err)
		}
		logs = append(logs, logsBatch...)
		return true, nil
	})
	if err != nil {
		return nil, err
	}
	return r.processLogsToRollupEvents(logs)
}

// FetchRollupEventsInRangeWithCallback retrieves and parses commit/revert/finalize rollup events between block numbers: [from, to].
func (r *Reader) FetchRollupEventsInRangeWithCallback(from, to uint64, callback func(event RollupEvent) bool) error {
	log.Trace("L1Client fetchRollupEventsInRange", "fromBlock", from, "toBlock", to)

	err := queryInBatches(r.ctx, from, to, defaultRollupEventsFetchBlockRange, func(from, to uint64) (bool, error) {
		query := ethereum.FilterQuery{
			FromBlock: big.NewInt(int64(from)), // inclusive
			ToBlock:   big.NewInt(int64(to)),   // inclusive
			Addresses: []common.Address{
				r.config.ScrollChainAddress,
			},
			Topics: make([][]common.Hash, 1),
		}
		query.Topics[0] = make([]common.Hash, 4)
		query.Topics[0][0] = r.l1CommitBatchEventSignature
		query.Topics[0][1] = r.l1RevertBatchEventV0Signature
		query.Topics[0][2] = r.l1RevertBatchEventV7Signature
		query.Topics[0][3] = r.l1FinalizeBatchEventSignature

		logsBatch, err := r.client.FilterLogs(r.ctx, query)
		if err != nil {
			return false, fmt.Errorf("failed to filter logs, err: %w", err)
		}

		rollupEvents, err := r.processLogsToRollupEvents(logsBatch)
		if err != nil {
			return false, fmt.Errorf("failed to process logs to rollup events, err: %w", err)
		}

		for _, event := range rollupEvents {
			if !callback(event) {
				return false, nil
			}
		}

		return true, nil
	})
	if err != nil {
		return err
	}

	return nil
}

func (r *Reader) processLogsToRollupEvents(logs []types.Log) (RollupEvents, error) {
	var rollupEvents RollupEvents
	var rollupEvent RollupEvent
	var err error

	for _, vLog := range logs {
		switch vLog.Topics[0] {
		case r.l1CommitBatchEventSignature:
			event := &CommitBatchEventUnpacked{}
			if err = UnpackLog(r.scrollChainABI, event, commitBatchEventName, vLog); err != nil {
				return nil, fmt.Errorf("failed to unpack commit rollup event log, err: %w", err)
			}
			log.Trace("found new CommitBatch event", "batch index", event.BatchIndex.Uint64(), "batch hash", event.BatchHash.Hex())
			rollupEvent = &CommitBatchEvent{
				batchIndex:  event.BatchIndex,
				batchHash:   event.BatchHash,
				txHash:      vLog.TxHash,
				blockHash:   vLog.BlockHash,
				blockNumber: vLog.BlockNumber,
			}

		case r.l1RevertBatchEventV0Signature:
			event := &RevertBatchEventV0Unpacked{}
			if err = UnpackLog(r.scrollChainABI, event, revertBatchV0EventName, vLog); err != nil {
				return nil, fmt.Errorf("failed to unpack revert V0 rollup event log, err: %w", err)
			}
			log.Trace("found new RevertBatchV0Type event", "batch index", event.BatchIndex.Uint64(), "batch hash", event.BatchHash.Hex())
			rollupEvent = &RevertBatchEventV0{
				batchIndex:  event.BatchIndex,
				batchHash:   event.BatchHash,
				txHash:      vLog.TxHash,
				blockHash:   vLog.BlockHash,
				blockNumber: vLog.BlockNumber,
			}
		case r.l1RevertBatchEventV7Signature:
			event := &RevertBatchEventV7Unpacked{}
			if err = UnpackLog(r.scrollChainABI, event, revertBatchV7EventName, vLog); err != nil {
				return nil, fmt.Errorf("failed to unpack revert V7 rollup event log, err: %w", err)
			}

			log.Trace("found new RevertBatchV7Type event", "start batch index", event.StartBatchIndex.Uint64(), "finish batch index", event.FinishBatchIndex.Uint64())
			rollupEvent = &RevertBatchEventV7{
				startBatchIndex:  event.StartBatchIndex,
				finishBatchIndex: event.FinishBatchIndex,
				txHash:           vLog.TxHash,
				blockHash:        vLog.BlockHash,
				blockNumber:      vLog.BlockNumber,
			}
		case r.l1FinalizeBatchEventSignature:
			event := &FinalizeBatchEventUnpacked{}
			if err = UnpackLog(r.scrollChainABI, event, finalizeBatchEventName, vLog); err != nil {
				return nil, fmt.Errorf("failed to unpack finalized rollup event log, err: %w", err)
			}
			log.Trace("found new FinalizeBatchType event", "batch index", event.BatchIndex.Uint64(), "batch hash", event.BatchHash.Hex())
			rollupEvent = &FinalizeBatchEvent{
				batchIndex:   event.BatchIndex,
				batchHash:    event.BatchHash,
				stateRoot:    event.StateRoot,
				withdrawRoot: event.WithdrawRoot,
				txHash:       vLog.TxHash,
				blockHash:    vLog.BlockHash,
				blockNumber:  vLog.BlockNumber,
			}

		default:
			return nil, fmt.Errorf("unknown event, topic: %v, tx hash: %v", vLog.Topics[0].Hex(), vLog.TxHash.Hex())
		}

		rollupEvents = append(rollupEvents, rollupEvent)
	}
	return rollupEvents, nil
}

func queryInBatches(ctx context.Context, fromBlock, toBlock uint64, batchSize uint64, queryFunc func(from, to uint64) (bool, error)) error {
	for from := fromBlock; from <= toBlock; from += batchSize {
		// check if context is done and return if it is
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		to := from + batchSize - 1
		if to > toBlock {
			to = toBlock
		}
		cont, err := queryFunc(from, to)
		if err != nil {
			return fmt.Errorf("error querying blocks %d to %d: %w", from, to, err)
		}
		if !cont {
			break
		}
	}
	return nil
}

// fetchTx fetches tx corresponding to given event log
func (r *Reader) fetchTx(txHash, blockHash common.Hash) (*types.Transaction, error) {
	tx, _, err := r.client.TransactionByHash(r.ctx, txHash)
	if err != nil {
		log.Debug("failed to get transaction by hash, probably an unindexed transaction, fetching the whole block to get the transaction",
			"tx hash", txHash.Hex(), "block hash", blockHash.Hex(), "err", err)
		block, err := r.client.BlockByHash(r.ctx, blockHash)
		if err != nil {
			return nil, fmt.Errorf("failed to get block by hash, block hash: %v, err: %w", blockHash.Hex(), err)
		}

		found := false
		for _, txInBlock := range block.Transactions() {
			if txInBlock.Hash() == txHash {
				tx = txInBlock
				found = true
				break
			}
		}
		if !found {
			return nil, fmt.Errorf("transaction not found in the block, tx hash: %v, block hash: %v", txHash.Hex(), blockHash.Hex())
		}
	}

	return tx, nil
}

func (r *Reader) FetchCommitTxData(commitEvent *CommitBatchEvent) (*CommitBatchArgs, error) {
	tx, err := r.fetchTx(commitEvent.TxHash(), commitEvent.BlockHash())
	if err != nil {
		return nil, err
	}
	txData := tx.Data()

	if len(txData) < methodIDLength {
		return nil, fmt.Errorf("transaction data is too short, length of tx data: %v, minimum length required: %v", len(txData), methodIDLength)
	}

	method, err := r.scrollChainABI.MethodById(txData[:methodIDLength])
	if err != nil {
		return nil, fmt.Errorf("failed to get method by ID, ID: %v, err: %w", txData[:methodIDLength], err)
	}
	values, err := method.Inputs.Unpack(txData[methodIDLength:])
	if err != nil {
		return nil, fmt.Errorf("failed to unpack transaction data using ABI, tx data: %v, err: %w", txData, err)
	}

	var args *CommitBatchArgs
	if method.Name == commitBatchMethodName {
		args, err = newCommitBatchArgs(method, values)
		if err != nil {
			return nil, fmt.Errorf("failed to decode calldata into commitBatch args %s, values: %+v, err: %w", commitBatchMethodName, values, err)
		}
	} else if method.Name == commitBatchWithBlobProofMethodName {
		args, err = newCommitBatchArgsFromCommitBatchWithProof(method, values)
		if err != nil {
			return nil, fmt.Errorf("failed to decode calldata into commitBatch args %s, values: %+v, err: %w", commitBatchWithBlobProofMethodName, values, err)
		}
	} else if method.Name == commitBatchesV7MethodName {
		args, err = newCommitBatchArgsFromCommitBatchesV7(method, values)
		if err != nil {
			return nil, fmt.Errorf("failed to decode calldata into commitBatch args %s, values: %+v, err: %w", commitBatchesV7MethodName, values, err)
		}
	} else if method.Name == commitAndFinalizeBatch {
		commitAndFinalizeArgs, err := newCommitAndFinalizeBatchArgs(method, values)
		if err != nil {
			return nil, fmt.Errorf("failed to decode calldata into commitAndFinalizeBatch args %s, values: %+v, err: %w", commitAndFinalizeBatch, values, err)
		}

		// in commitAndFinalizeBatch, the last batch hash is encoded in the finalize struct as this is the only batch we're
		// committing when calling this function.
		codec, err := encoding.CodecFromVersion(encoding.CodecVersion(commitAndFinalizeArgs.Version))
		if err != nil {
			return nil, fmt.Errorf("failed to get codec from version %d, err: %w", commitAndFinalizeArgs.Version, err)
		}
		daBatch, err := codec.NewDABatchFromBytes(commitAndFinalizeArgs.FinalizeStruct.BatchHeader)
		if err != nil {
			return nil, fmt.Errorf("failed to decode daBatch from bytes, err: %w", err)
		}

		args = &CommitBatchArgs{
			Version:         commitAndFinalizeArgs.Version,
			ParentBatchHash: commitAndFinalizeArgs.ParentBatchHash,
			LastBatchHash:   daBatch.Hash(),
		}
	} else {
		return nil, fmt.Errorf("unknown method name for commit transaction: %s", method.Name)
	}

	args.BlobHashes = tx.BlobHashes()

	return args, nil
}

func (r *Reader) FetchFinalizeTxDataPostEuclidV2(event *FinalizeBatchEvent) (*FinalizeBatchArgs, error) {
	tx, err := r.fetchTx(event.TxHash(), event.BlockHash())
	if err != nil {
		return nil, err
	}
	txData := tx.Data()

	if len(txData) < methodIDLength {
		return nil, fmt.Errorf("transaction data is too short, length of tx data: %v, minimum length required: %v", len(txData), methodIDLength)
	}

	method, err := r.scrollChainABI.MethodById(txData[:methodIDLength])
	if err != nil {
		return nil, fmt.Errorf("failed to get method by ID, ID: %v, err: %w", txData[:methodIDLength], err)
	}
	values, err := method.Inputs.Unpack(txData[methodIDLength:])
	if err != nil {
		return nil, fmt.Errorf("failed to unpack transaction data using ABI, tx data: %v, err: %w", txData, err)
	}

	var args *FinalizeBatchArgs
	if method.Name == finalizeBundlePostEuclidV2MethodName {
		args, err = newFinalizeBatchArgs(method, values)
		if err != nil {
			return nil, fmt.Errorf("failed to decode calldata into finalizeBatch args %s, values: %+v, err: %w", finalizeBundlePostEuclidV2MethodName, values, err)
		}
	} else {
		return nil, fmt.Errorf("unknown method name for finalize transaction: %s", method.Name)
	}

	return args, nil
}
