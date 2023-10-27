package rollup_sync_service

import (
	"context"
	"errors"
	"fmt"
	"math/big"

	"github.com/scroll-tech/go-ethereum"
	"github.com/scroll-tech/go-ethereum/accounts/abi"
	"github.com/scroll-tech/go-ethereum/common"
	"github.com/scroll-tech/go-ethereum/core/types"
	"github.com/scroll-tech/go-ethereum/log"
	"github.com/scroll-tech/go-ethereum/rpc"

	"github.com/scroll-tech/go-ethereum/rollup/sync_service"
)

// L1Client is a wrapper around EthClient that adds
// methods for conveniently collecting rollup events of ScrollChain contract.
type L1Client struct {
	ctx                           context.Context
	client                        sync_service.EthClient
	scrollChainAddress            common.Address
	l1CommitBatchEventSignature   common.Hash
	l1RevertBatchEventSignature   common.Hash
	l1FinalizeBatchEventSignature common.Hash
}

// newL1Client initializes a new L1Client instance with the provided configuration.
// It checks for a valid scrollChainAddress and verifies the chain ID.
func newL1Client(ctx context.Context, l1Client sync_service.EthClient, l1ChainId uint64, scrollChainAddress common.Address, scrollChainABI *abi.ABI) (*L1Client, error) {
	if scrollChainAddress == (common.Address{}) {
		return nil, errors.New("must pass non-zero scrollChainAddress to L1Client")
	}

	// sanity check: compare chain IDs
	got, err := l1Client.ChainID(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to query L1 chain ID, err: %w", err)
	}
	if got.Cmp(big.NewInt(0).SetUint64(l1ChainId)) != 0 {
		return nil, fmt.Errorf("unexpected chain ID, expected: %v, got: %v", l1ChainId, got)
	}

	client := L1Client{
		ctx:                           ctx,
		client:                        l1Client,
		scrollChainAddress:            scrollChainAddress,
		l1CommitBatchEventSignature:   scrollChainABI.Events["CommitBatch"].ID,
		l1RevertBatchEventSignature:   scrollChainABI.Events["RevertBatch"].ID,
		l1FinalizeBatchEventSignature: scrollChainABI.Events["FinalizeBatch"].ID,
	}

	return &client, nil
}

// fetcRollupEventsInRange retrieves and parses commit/revert/finalize rollup events between block numbers: [from, to].
func (c *L1Client) fetchRollupEventsInRange(ctx context.Context, from, to uint64) ([]types.Log, error) {
	log.Trace("L1Client fetchRollupEventsInRange", "fromBlock", from, "toBlock", to)

	query := ethereum.FilterQuery{
		FromBlock: big.NewInt(int64(from)), // inclusive
		ToBlock:   big.NewInt(int64(to)),   // inclusive
		Addresses: []common.Address{
			c.scrollChainAddress,
		},
		Topics: make([][]common.Hash, 1),
	}
	query.Topics[0] = make([]common.Hash, 3)
	query.Topics[0][0] = c.l1CommitBatchEventSignature
	query.Topics[0][1] = c.l1RevertBatchEventSignature
	query.Topics[0][2] = c.l1FinalizeBatchEventSignature

	logs, err := c.client.FilterLogs(c.ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to filter logs, err: %w", err)
	}
	return logs, nil
}

// getLatestFinalizedBlockNumber fetches the block number of the latest finalized block from the L1 chain.
func (c *L1Client) getLatestFinalizedBlockNumber(ctx context.Context) (uint64, error) {
	header, err := c.client.HeaderByNumber(ctx, big.NewInt(int64(rpc.FinalizedBlockNumber)))
	if err != nil {
		return 0, err
	}
	if !header.Number.IsInt64() {
		return 0, fmt.Errorf("received unexpected block number in L1Client: %v", header.Number)
	}
	return header.Number.Uint64(), nil
}
