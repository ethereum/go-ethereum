// Copyright 2015 The go-ethereum Authors
// This file is part of the go-ethereum library.
//
// The go-ethereum library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The go-ethereum library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the go-ethereum library. If not, see <http://www.gnu.org/licenses/>.

package downloader

import (
	"context"
	"sync"

	ethereum "github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/event"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/rpc"
)

const (
	// startChanSize is the size of channel listening to StartEvent.
	startChanSize = 10
	// finishChanSize is the size of channel listening to FinishEvent.
	finishChanSize = 10
)

// PublicDownloaderAPI provides an API which gives information about the current synchronisation status.
// It offers only methods that operates on data that can be available to anyone without security risks.
type PublicDownloaderAPI struct {
	d *Downloader

	// Channels
	startCh                   chan StartEvent
	finishCh                  chan FinishEvent
	installSyncSubscription   chan chan interface{}
	uninstallSyncSubscription chan *uninstallSyncSubscriptionRequest

	// Subscriptions
	startSub  event.Subscription
	finishSub event.Subscription
}

// NewPublicDownloaderAPI create a new PublicDownloaderAPI. The API has an internal event loop that
// listens for events from the downloader through the global event mux. In case it receives one of
// these events it broadcasts it to all syncing subscriptions that are installed through the
// installSyncSubscription channel.
func NewPublicDownloaderAPI(d *Downloader) *PublicDownloaderAPI {
	api := &PublicDownloaderAPI{
		d:                         d,
		startCh:                   make(chan StartEvent, startChanSize),
		finishCh:                  make(chan FinishEvent, finishChanSize),
		installSyncSubscription:   make(chan chan interface{}),
		uninstallSyncSubscription: make(chan *uninstallSyncSubscriptionRequest),
	}

	// Subscribe downloader events
	api.startSub = d.SubscribeStartEvent(api.startCh)
	api.finishSub = d.SubscribeFinishEvent(api.finishCh)
	if api.startSub == nil || api.finishSub == nil {
		log.Crit("Subscribe downloader events failed")
	}

	go api.eventLoop()

	return api
}

// eventLoop runs a loop until the event mux closes. It will install and uninstall new
// sync subscriptions and broadcasts sync status updates to the installed sync subscriptions.
func (api *PublicDownloaderAPI) eventLoop() {
	var syncSubscriptions = make(map[chan interface{}]struct{})

	defer func() {
		api.startSub.Unsubscribe()
		api.finishSub.Unsubscribe()
	}()

	broadcast := func(notification interface{}) {
		for c := range syncSubscriptions {
			c <- notification
		}
	}

	for {
		select {
		case i := <-api.installSyncSubscription:
			syncSubscriptions[i] = struct{}{}

		case u := <-api.uninstallSyncSubscription:
			delete(syncSubscriptions, u.c)
			close(u.uninstalled)

		case <-api.startCh:
			broadcast(&SyncingResult{
				Syncing: true,
				Status:  api.d.Progress(),
			})

		case <-api.finishCh:
			broadcast(false)

		case <-api.startSub.Err():
			return

		case <-api.finishSub.Err():
			return
		}
	}
}

// Syncing provides information when this nodes starts synchronising with the Ethereum network and when it's finished.
func (api *PublicDownloaderAPI) Syncing(ctx context.Context) (*rpc.Subscription, error) {
	notifier, supported := rpc.NotifierFromContext(ctx)
	if !supported {
		return &rpc.Subscription{}, rpc.ErrNotificationsUnsupported
	}

	rpcSub := notifier.CreateSubscription()

	go func() {
		statuses := make(chan interface{})
		sub := api.SubscribeSyncStatus(statuses)

		for {
			select {
			case status := <-statuses:
				notifier.Notify(rpcSub.ID, status)
			case <-rpcSub.Err():
				sub.Unsubscribe()
				return
			case <-notifier.Closed():
				sub.Unsubscribe()
				return
			}
		}
	}()

	return rpcSub, nil
}

// SyncingResult provides information about the current synchronisation status for this node.
type SyncingResult struct {
	Syncing bool                  `json:"syncing"`
	Status  ethereum.SyncProgress `json:"status"`
}

// uninstallSyncSubscriptionRequest uninstalles a syncing subscription in the API event loop.
type uninstallSyncSubscriptionRequest struct {
	c           chan interface{}
	uninstalled chan interface{}
}

// SyncStatusSubscription represents a syncing subscription.
type SyncStatusSubscription struct {
	api       *PublicDownloaderAPI // register subscription in event loop of this api instance
	c         chan interface{}     // channel where events are broadcasted to
	unsubOnce sync.Once            // make sure unsubscribe logic is executed once
}

// Unsubscribe uninstalls the subscription from the DownloadAPI event loop.
// The status channel that was passed to subscribeSyncStatus isn't used anymore
// after this method returns.
func (s *SyncStatusSubscription) Unsubscribe() {
	s.unsubOnce.Do(func() {
		req := uninstallSyncSubscriptionRequest{s.c, make(chan interface{})}
		s.api.uninstallSyncSubscription <- &req

		for {
			select {
			case <-s.c:
				// drop new status events until uninstall confirmation
				continue
			case <-req.uninstalled:
				return
			}
		}
	})
}

// SubscribeSyncStatus creates a subscription that will broadcast new synchronisation updates.
// The given channel must receive interface values, the result can either
func (api *PublicDownloaderAPI) SubscribeSyncStatus(status chan interface{}) *SyncStatusSubscription {
	api.installSyncSubscription <- status
	return &SyncStatusSubscription{api: api, c: status}
}
