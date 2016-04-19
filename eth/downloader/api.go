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
	"sync"

	"golang.org/x/net/context"

	"github.com/ethereum/go-ethereum/event"
	"github.com/ethereum/go-ethereum/rpc"
)

// PublicDownloaderAPI provides an API which gives information about the current synchronisation status.
// It offers only methods that operates on data that can be available to anyone without security risks.
type PublicDownloaderAPI struct {
	d                   *Downloader
	mux                 *event.TypeMux
	muSyncSubscriptions sync.Mutex
	syncSubscriptions   map[string]rpc.Subscription
}

// NewPublicDownloaderAPI create a new PublicDownloaderAPI.
func NewPublicDownloaderAPI(d *Downloader, m *event.TypeMux) *PublicDownloaderAPI {
	api := &PublicDownloaderAPI{d: d, mux: m, syncSubscriptions: make(map[string]rpc.Subscription)}

	go api.run()

	return api
}

func (api *PublicDownloaderAPI) run() {
	sub := api.mux.Subscribe(StartEvent{}, DoneEvent{}, FailedEvent{})

	for event := range sub.Chan() {
		var notification interface{}

		switch event.Data.(type) {
		case StartEvent:
			result := &SyncingResult{Syncing: true}
			result.Status.Origin, result.Status.Current, result.Status.Height, result.Status.Pulled, result.Status.Known = api.d.Progress()
			notification = result
		case DoneEvent, FailedEvent:
			notification = false
		}

		api.muSyncSubscriptions.Lock()
		for id, sub := range api.syncSubscriptions {
			if sub.Notify(notification) == rpc.ErrNotificationNotFound {
				delete(api.syncSubscriptions, id)
			}
		}
		api.muSyncSubscriptions.Unlock()
	}
}

// Progress gives progress indications when the node is synchronising with the Ethereum network.
type Progress struct {
	Origin  uint64 `json:"startingBlock"`
	Current uint64 `json:"currentBlock"`
	Height  uint64 `json:"highestBlock"`
	Pulled  uint64 `json:"pulledStates"`
	Known   uint64 `json:"knownStates"`
}

// SyncingResult provides information about the current synchronisation status for this node.
type SyncingResult struct {
	Syncing bool     `json:"syncing"`
	Status  Progress `json:"status"`
}

// Syncing provides information when this nodes starts synchronising with the Ethereum network and when it's finished.
func (api *PublicDownloaderAPI) Syncing(ctx context.Context) (rpc.Subscription, error) {
	notifier, supported := rpc.NotifierFromContext(ctx)
	if !supported {
		return nil, rpc.ErrNotificationsUnsupported
	}

	subscription, err := notifier.NewSubscription(func(id string) {
		api.muSyncSubscriptions.Lock()
		delete(api.syncSubscriptions, id)
		api.muSyncSubscriptions.Unlock()
	})

	if err != nil {
		return nil, err
	}

	api.muSyncSubscriptions.Lock()
	api.syncSubscriptions[subscription.ID()] = subscription
	api.muSyncSubscriptions.Unlock()

	return subscription, nil
}
