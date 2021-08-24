package backends


import (
	"context"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/event"
	"github.com/ethereum/go-ethereum/consensus"
)


func (fb *filterBackend) SubscribeDropTxsEvent(ch chan<- core.DropTxsEvent) event.Subscription {
	return nullSubscription()
}

func (fb *filterBackend) SubscribeRejectedTxEvent(ch chan<- core.RejectedTxEvent) event.Subscription {
	return nullSubscription()
}

func (fb *filterBackend) GetPoolTransaction(hash common.Hash) *types.Transaction {
	return nil
}
func (fb *filterBackend) BlockByHash(ctx context.Context, hash common.Hash) (*types.Block, error) {
	return nil, nil
}

func (fb *filterBackend) Engine() consensus.Engine {
	return nil
}
