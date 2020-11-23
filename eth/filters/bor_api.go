package filters

import (
	"bytes"
	"context"

	ethereum "github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/rpc"
)

// NewDeposits send a notification each time a new deposit received from bridge.
func (api *PublicFilterAPI) NewDeposits(ctx context.Context, crit ethereum.StateSyncFilter) (*rpc.Subscription, error) {
	notifier, supported := rpc.NotifierFromContext(ctx)
	if !supported {
		return &rpc.Subscription{}, rpc.ErrNotificationsUnsupported
	}

	rpcSub := notifier.CreateSubscription()
	go func() {
		stateSyncData := make(chan *types.StateSyncData)
		stateSyncSub := api.events.SubscribeNewDeposits(stateSyncData)

		for {
			select {
			case h := <-stateSyncData:
				if crit.ID == h.ID || bytes.Compare(crit.Contract.Bytes(), h.Contract.Bytes()) == 0 ||
					(crit.ID == 0 && crit.Contract == common.Address{}) {
					notifier.Notify(rpcSub.ID, h)
				}
			case <-rpcSub.Err():
				stateSyncSub.Unsubscribe()
				return
			case <-notifier.Closed():
				stateSyncSub.Unsubscribe()
				return
			}
		}
	}()

	return rpcSub, nil
}
