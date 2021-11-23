// This file extends the EthAPIBackend with functions for BlockNative's dropped
// transaction feeds.
//
// As this is part of a fork, and not included in core geth, keeping it in a
// separate file helps protect against potential merge conflicts. If this were
// ever to be merged into core geth, it should be relocated to ./api_backend.go

package eth

import (
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/event"
)

func (b *EthAPIBackend)  SubscribeQueuedTxsEvent(ch chan<- *types.Transaction) event.Subscription {
	return b.eth.TxPool().SubscribeQueuedTxsEvent(ch)
}

func (b *EthAPIBackend) SubscribeDropTxsEvent(ch chan<- core.DropTxsEvent) event.Subscription {
	return b.eth.TxPool().SubscribeDropTxsEvent(ch)
}

func (b *EthAPIBackend) SubscribeRejectedTxEvent(ch chan<- core.RejectedTxEvent) event.Subscription {
	return b.eth.TxPool().SubscribeRejectedTxEvent(ch)
}
