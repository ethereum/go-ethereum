package filters

import (
	"context"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/rpc"

)

type dropNotification struct {
	TxHash common.Hash `json:"txhash"`
	Reason string `json:"reason"`
}

type rejectNotification struct {
	Tx *types.Transaction `json:"tx"`
	Reason string `json:"reason"`
}

// DroppedTransactions send a notification each time a transaction is dropped from the mempool
func (api *PublicFilterAPI) DroppedTransactions(ctx context.Context) (*rpc.Subscription, error) {
	notifier, supported := rpc.NotifierFromContext(ctx)
	if !supported {
		return &rpc.Subscription{}, rpc.ErrNotificationsUnsupported
	}

	rpcSub := notifier.CreateSubscription()

	go func() {
		dropped := make(chan core.DropTxsEvent)
		droppedSub := api.backend.SubscribeDropTxsEvent(dropped)

		for {
			select {
			case d := <-dropped:
				for _, tx := range d.Txs {
					notifier.Notify(rpcSub.ID, &dropNotification{TxHash: tx.Hash(), Reason: d.Reason})
				}
			case <-rpcSub.Err():
				droppedSub.Unsubscribe()
				return
			case <-notifier.Closed():
				droppedSub.Unsubscribe()
				return
			}
		}
	}()

	return rpcSub, nil
}

// RejectedTransactions send a notification each time a transaction is rejected from entering the mempool
func (api *PublicFilterAPI) RejectedTransactions(ctx context.Context) (*rpc.Subscription, error) {
	notifier, supported := rpc.NotifierFromContext(ctx)
	if !supported {
		return &rpc.Subscription{}, rpc.ErrNotificationsUnsupported
	}

	rpcSub := notifier.CreateSubscription()

	go func() {
		rejected := make(chan core.RejectedTxEvent)
		rejectedSub := api.backend.SubscribeRejectedTxEvent(rejected)

		for {
			select {
			case d := <-rejected:
				reason := ""
				if d.Reason != nil {
					reason = d.Reason.Error()
				}
				notifier.Notify(rpcSub.ID, &rejectNotification{Tx: d.Tx, Reason: reason})
			case <-rpcSub.Err():
				rejectedSub.Unsubscribe()
				return
			case <-notifier.Closed():
				rejectedSub.Unsubscribe()
				return
			}
		}
	}()

	return rpcSub, nil
}
