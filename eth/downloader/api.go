// Copyright 2015 The go-ethereum Authors
// This file is part of go-ethereum.
//
// go-ethereum is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// go-ethereum is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU General Public License for more details.
//
// You should have received a copy of the GNU General Public License
// along with go-ethereum. If not, see <http://www.gnu.org/licenses/>.

package downloader

import (
	"github.com/ethereum/go-ethereum/rpc"
)

// PublicDownloaderAPI provides an API which gives informatoin about the current synchronisation status.
// It offers only methods that operates on data that can be available to anyone without security risks.
type PublicDownloaderAPI struct {
	d *Downloader
}

// NewPublicDownloaderAPI create a new PublicDownloaderAPI.
func NewPublicDownloaderAPI(d *Downloader) *PublicDownloaderAPI {
	return &PublicDownloaderAPI{d}
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
func (s *PublicDownloaderAPI) Syncing() (rpc.Subscription, error) {
	sub := s.d.mux.Subscribe(StartEvent{}, DoneEvent{}, FailedEvent{})

	output := func(event interface{}) interface{} {
		switch event.(type) {
		case StartEvent:
			result := &SyncingResult{Syncing: true}
			result.Status.Origin, result.Status.Current, result.Status.Height, result.Status.Pulled, result.Status.Known = s.d.Progress()
			return result
		case DoneEvent, FailedEvent:
			return false
		}
		return nil
	}
	return rpc.NewSubscriptionWithOutputFormat(sub, output), nil
}
