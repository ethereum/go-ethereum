// Copyright 2022 The go-ethereum Authors
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

package eth

import (
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/eth/ethconfig"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/node"
)

// SyncOverride is an auxiliary service that allows Geth to perform full sync
// alone without consensus-layer attached. Users must specify a valid block hash
// as the sync target.
//
// This tester can be applied to different networks, no matter it's pre-merge or
// post-merge, but only for full-sync.
type SyncOverride struct {
	stack   *node.Node
	backend *Ethereum
	closed  chan struct{}
}

// SyncTarget sets the target of the client sync to the given block hash
func (f *SyncOverride) SyncTarget(target common.Hash, exitWhenSynced bool) {
	go func() {
		// Trigger beacon sync with the provided block hash as trusted
		// chain head.
		err := f.backend.Downloader().BeaconDevSync(ethconfig.FullSync, target, f.closed)
		if err != nil {
			log.Info("Failed to trigger beacon sync", "err", err)
		}

		ticker := time.NewTicker(time.Second * 5)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				// Stop in case the target block is already stored locally.
				if block := f.backend.BlockChain().GetBlockByHash(target); block != nil {
					log.Info("Full-sync target reached", "number", block.NumberU64(), "hash", block.Hash())
					if exitWhenSynced {
						log.Info("Terminating node")
						f.stack.Close()
					}
					return
				}

			case <-f.closed:
				return
			}
		}
	}()
}

// Stop stops the full-sync tester to stop all background activities.
// This function can only be called for one time.
func (f *SyncOverride) Stop() error {
	close(f.closed)
	return nil
}
