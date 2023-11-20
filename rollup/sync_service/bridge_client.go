package sync_service

import (
	"context"
	"errors"
	"fmt"
	"math/big"

	"github.com/scroll-tech/go-ethereum/accounts/abi/bind"
	"github.com/scroll-tech/go-ethereum/common"
	"github.com/scroll-tech/go-ethereum/core/types"
	"github.com/scroll-tech/go-ethereum/log"
	"github.com/scroll-tech/go-ethereum/rpc"
)

// BridgeClient is a wrapper around EthClient that adds
// methods for conveniently collecting L1 messages.
type BridgeClient struct {
	client                EthClient
	confirmations         rpc.BlockNumber
	l1MessageQueueAddress common.Address
	filterer              *L1MessageQueueFilterer
}

func newBridgeClient(ctx context.Context, l1Client EthClient, l1ChainId uint64, confirmations rpc.BlockNumber, l1MessageQueueAddress common.Address) (*BridgeClient, error) {
	if l1MessageQueueAddress == (common.Address{}) {
		return nil, errors.New("must pass non-zero l1MessageQueueAddress to BridgeClient")
	}

	// sanity check: compare chain IDs
	got, err := l1Client.ChainID(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to query L1 chain ID, err = %w", err)
	}
	if got.Cmp(big.NewInt(0).SetUint64(l1ChainId)) != 0 {
		return nil, fmt.Errorf("unexpected chain ID, expected = %v, got = %v", l1ChainId, got)
	}

	filterer, err := NewL1MessageQueueFilterer(l1MessageQueueAddress, l1Client)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize L1MessageQueueFilterer, err = %w", err)
	}

	client := BridgeClient{
		client:                l1Client,
		confirmations:         confirmations,
		l1MessageQueueAddress: l1MessageQueueAddress,
		filterer:              filterer,
	}

	return &client, nil
}

// fetchMessagesInRange retrieves and parses all L1 messages between the
// provided from and to L1 block numbers (inclusive).
func (c *BridgeClient) fetchMessagesInRange(ctx context.Context, from, to uint64) ([]types.L1MessageTx, error) {
	log.Trace("BridgeClient fetchMessagesInRange", "fromBlock", from, "toBlock", to)

	opts := bind.FilterOpts{
		Start:   from,
		End:     &to,
		Context: ctx,
	}
	it, err := c.filterer.FilterQueueTransaction(&opts, nil, nil)
	if err != nil {
		return nil, err
	}

	var msgs []types.L1MessageTx

	for it.Next() {
		event := it.Event
		log.Trace("Received new L1 QueueTransaction event", "event", event)

		if !event.GasLimit.IsUint64() {
			return nil, fmt.Errorf("invalid QueueTransaction event: QueueIndex = %v, GasLimit = %v", event.QueueIndex, event.GasLimit)
		}

		msgs = append(msgs, types.L1MessageTx{
			QueueIndex: event.QueueIndex,
			Gas:        event.GasLimit.Uint64(),
			To:         &event.Target,
			Value:      event.Value,
			Data:       event.Data,
			Sender:     event.Sender,
		})
	}

	return msgs, nil
}

func (c *BridgeClient) getLatestConfirmedBlockNumber(ctx context.Context) (uint64, error) {
	// confirmation based on "safe" or "finalized" block tag
	if c.confirmations == rpc.SafeBlockNumber || c.confirmations == rpc.FinalizedBlockNumber {
		tag := big.NewInt(int64(c.confirmations))
		header, err := c.client.HeaderByNumber(ctx, tag)
		if err != nil {
			return 0, err
		}
		if !header.Number.IsInt64() {
			return 0, fmt.Errorf("received unexpected block number in BridgeClient: %v", header.Number)
		}
		return header.Number.Uint64(), nil
	}

	// confirmation based on latest block number
	if c.confirmations == rpc.LatestBlockNumber {
		number, err := c.client.BlockNumber(ctx)
		if err != nil {
			return 0, err
		}
		return number, nil
	}

	// confirmation based on a certain number of blocks
	if c.confirmations.Int64() >= 0 {
		number, err := c.client.BlockNumber(ctx)
		if err != nil {
			return 0, err
		}
		confirmations := uint64(c.confirmations.Int64())
		if number >= confirmations {
			return number - confirmations, nil
		}
		return 0, nil
	}

	return 0, fmt.Errorf("unknown confirmation type: %v", c.confirmations)
}
