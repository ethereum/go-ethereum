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
	"time"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/event"
	"github.com/ethereum/go-ethereum/rpc"
)

// DownloaderAPI provides an API which gives information about the current
// synchronisation status. It offers only methods that operates on data that
// can be available to anyone without security risks.
type DownloaderAPI struct {
	d     *Downloader
	chain *core.BlockChain
	feed  *event.FeedOf[SyncEvent]
}

// NewDownloaderAPI creates a new DownloaderAPI.
func NewDownloaderAPI(d *Downloader, chain *core.BlockChain, f *event.FeedOf[SyncEvent]) *DownloaderAPI {
	return &DownloaderAPI{
		d:     d,
		chain: chain,
		feed:  f,
	}
}

// Syncing provides information when this node starts synchronising with the Ethereum network and when it's finished.
func (api *DownloaderAPI) Syncing(ctx context.Context) (*rpc.Subscription, error) {
	notifier, supported := rpc.NotifierFromContext(ctx)
	if !supported {
		return &rpc.Subscription{}, rpc.ErrNotificationsUnsupported
	}

	rpcSub := notifier.CreateSubscription()

	go func() {
		statuses := make(chan interface{})
		api.SubscribeSyncStatus(statuses)

		for {
			select {
			case status := <-statuses:
				notifier.Notify(rpcSub.ID, status)
			case <-rpcSub.Err():
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

// SubscribeSyncStatus creates a subscription that will broadcast new synchronisation updates.
// The given channel must receive interface values, the result can either be a SyncingResult or false.
func (api *DownloaderAPI) SubscribeSyncStatus(status chan interface{}) {
	eventCh := make(chan SyncEvent, 16)
	sub := api.feed.Subscribe(eventCh)

	go func() {
		defer close(status)
		defer sub.Unsubscribe()

		var (
			checkInterval = time.Second * 60
			checkTimer    = time.NewTimer(checkInterval)
			started       bool
			done          bool

			getProgress = func() ethereum.SyncProgress {
				prog := api.d.Progress()
				if txProg, err := api.chain.TxIndexProgress(); err == nil {
					prog.TxIndexFinishedBlocks = txProg.Indexed
					prog.TxIndexRemainingBlocks = txProg.Remaining
				}
				remain, err := api.chain.StateIndexProgress()
				if err == nil {
					prog.StateIndexRemaining = remain
				}
				return prog
			}
		)
		defer checkTimer.Stop()

		for {
			select {
			case event := <-eventCh:
				if event.Type == SyncStarted {
					started = true
				}
			case <-checkTimer.C:
				if !started {
					checkTimer.Reset(checkInterval)
					continue
				}
				prog := getProgress()
				if !prog.Done() {
					notification := &SyncingResult{
						Syncing: true,
						Status:  prog,
					}
					select {
					case status <- notification:
					case <-sub.Err():
						return
					}
					checkTimer.Reset(checkInterval)
					continue
				}
				if !done {
					select {
					case status <- false:
					case <-sub.Err():
						return
					}
					done = true
				}
				checkTimer.Reset(checkInterval)
			case <-sub.Err():
				return
			}
		}
	}()
}
