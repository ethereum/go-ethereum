package filters

import (
	context "context"
	"crypto/rand"
	"math/big"

	common "github.com/ethereum/go-ethereum/common"
	core "github.com/ethereum/go-ethereum/core"
	bloombits "github.com/ethereum/go-ethereum/core/bloombits"
	"github.com/ethereum/go-ethereum/core/rawdb"
	types "github.com/ethereum/go-ethereum/core/types"
	ethdb "github.com/ethereum/go-ethereum/ethdb"
	event "github.com/ethereum/go-ethereum/event"
	"github.com/ethereum/go-ethereum/params"
	rpc "github.com/ethereum/go-ethereum/rpc"
)

type TestBackend struct {
	DB              ethdb.Database
	sections        uint64
	txFeed          event.Feed
	logsFeed        event.Feed
	rmLogsFeed      event.Feed
	pendingLogsFeed event.Feed
	chainFeed       event.Feed

	stateSyncFeed event.Feed
}

func (b *TestBackend) BloomStatus() (uint64, uint64) {
	return params.BloomBitsBlocks, b.sections
}

func (b *TestBackend) GetBorBlockReceipt(ctx context.Context, hash common.Hash) (*types.Receipt, error) {
	number := rawdb.ReadHeaderNumber(b.DB, hash)
	if number == nil {
		return &types.Receipt{}, nil
	}

	receipt := rawdb.ReadBorReceipt(b.DB, hash, *number, nil)
	if receipt == nil {
		return &types.Receipt{}, nil
	}

	return receipt, nil
}

func (b *TestBackend) GetBorBlockLogs(ctx context.Context, hash common.Hash) ([]*types.Log, error) {
	receipt, err := b.GetBorBlockReceipt(ctx, hash)
	if err != nil {
		return []*types.Log{}, err
	}

	if receipt == nil {
		return []*types.Log{}, nil
	}

	return receipt.Logs, nil
}

func (b *TestBackend) ChainDb() ethdb.Database {
	return b.DB
}

func (b *TestBackend) HeaderByNumber(ctx context.Context, blockNr rpc.BlockNumber) (*types.Header, error) {
	var (
		hash common.Hash
		num  uint64
	)

	if blockNr == rpc.LatestBlockNumber {
		hash = rawdb.ReadHeadBlockHash(b.DB)
		number := rawdb.ReadHeaderNumber(b.DB, hash)

		if number == nil {
			return &types.Header{}, nil
		}

		num = *number
	} else {
		num = uint64(blockNr)
		hash = rawdb.ReadCanonicalHash(b.DB, num)
	}

	return rawdb.ReadHeader(b.DB, hash, num), nil
}

func (b *TestBackend) HeaderByHash(ctx context.Context, hash common.Hash) (*types.Header, error) {
	number := rawdb.ReadHeaderNumber(b.DB, hash)
	if number == nil {
		return &types.Header{}, nil
	}

	return rawdb.ReadHeader(b.DB, hash, *number), nil
}

func (b *TestBackend) GetReceipts(ctx context.Context, hash common.Hash) (types.Receipts, error) {
	if number := rawdb.ReadHeaderNumber(b.DB, hash); number != nil {
		block := rawdb.ReadBlock(b.DB, hash, *number)
		return rawdb.ReadReceipts(b.DB, hash, *number, block.Time(), params.TestChainConfig), nil
	}

	return nil, nil
}

func (b *TestBackend) GetVoteOnHash(ctx context.Context, starBlockNr uint64, endBlockNr uint64, hash string, milestoneId string) (bool, error) {
	return false, nil
}

func (b *TestBackend) GetLogs(ctx context.Context, hash common.Hash, number uint64) ([][]*types.Log, error) {
	block := rawdb.ReadBlock(b.DB, hash, number)
	receipts := rawdb.ReadReceipts(b.DB, hash, number, block.Time(), params.TestChainConfig)

	logs := make([][]*types.Log, len(receipts))
	for i, receipt := range receipts {
		logs[i] = receipt.Logs
	}

	return logs, nil
}

func (b *TestBackend) SubscribeNewTxsEvent(ch chan<- core.NewTxsEvent) event.Subscription {
	return b.txFeed.Subscribe(ch)
}

func (b *TestBackend) SubscribeRemovedLogsEvent(ch chan<- core.RemovedLogsEvent) event.Subscription {
	return b.rmLogsFeed.Subscribe(ch)
}

func (b *TestBackend) SubscribeLogsEvent(ch chan<- []*types.Log) event.Subscription {
	return b.logsFeed.Subscribe(ch)
}

func (b *TestBackend) SubscribePendingLogsEvent(ch chan<- []*types.Log) event.Subscription {
	return b.pendingLogsFeed.Subscribe(ch)
}

func (b *TestBackend) SubscribeChainEvent(ch chan<- core.ChainEvent) event.Subscription {
	return b.chainFeed.Subscribe(ch)
}

func (b *TestBackend) ServiceFilter(ctx context.Context, session *bloombits.MatcherSession) {
	requests := make(chan chan *bloombits.Retrieval)

	go session.Multiplex(16, 0, requests)
	go func() {
		for {
			// Wait for a service request or a shutdown
			select {
			case <-ctx.Done():
				return

			case request := <-requests:
				task := <-request

				task.Bitsets = make([][]byte, len(task.Sections))
				for i, section := range task.Sections {
					nBig, err := rand.Int(rand.Reader, big.NewInt(100))

					if err != nil {
						panic(err)
					}

					if nBig.Int64()%4 != 0 { // Handle occasional missing deliveries
						head := rawdb.ReadCanonicalHash(b.DB, (section+1)*params.BloomBitsBlocks-1)
						task.Bitsets[i], _ = rawdb.ReadBloomBits(b.DB, task.Bit, section, head)
					}
				}
				request <- task
			}
		}
	}()
}

func (b *TestBackend) SubscribeStateSyncEvent(ch chan<- core.StateSyncEvent) event.Subscription {
	return b.stateSyncFeed.Subscribe(ch)
}

func (b *TestBackend) ChainConfig() *params.ChainConfig { panic("not implemented") }

func (b *TestBackend) CurrentHeader() *types.Header { panic("not implemented") }

func (b *TestBackend) GetBody(context.Context, common.Hash, rpc.BlockNumber) (*types.Body, error) {
	panic("not implemented")
}

func (b *TestBackend) PendingBlockAndReceipts() (*types.Block, types.Receipts) {
	panic("not implemented")
}
