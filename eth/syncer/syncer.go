// Copyright 2025 The go-ethereum Authors
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

package syncer

import (
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/beacon/params"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/eth"
	"github.com/ethereum/go-ethereum/eth/ethconfig"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/node"
	"github.com/ethereum/go-ethereum/rpc"
)

type syncReq struct {
	hash common.Hash
	errc chan error
}

// Syncer is an auxiliary service that allows Geth to perform full sync
// alone without consensus-layer attached. Users must specify a valid block hash
// as the sync target.
//
// This tool can be applied to different networks, no matter it's pre-merge or
// post-merge, but only for full-sync.
type Syncer struct {
	stack          *node.Node
	backend        *eth.Ethereum
	target         common.Hash
	request        chan *syncReq
	closed         chan struct{}
	wg             sync.WaitGroup
	exitWhenSynced bool
}

// Register registers the synchronization override service into the node
// stack for launching and stopping the service controlled by node.
func Register(stack *node.Node, backend *eth.Ethereum, target common.Hash, exitWhenSynced bool) (*Syncer, error) {
	s := &Syncer{
		stack:          stack,
		backend:        backend,
		target:         target,
		request:        make(chan *syncReq),
		closed:         make(chan struct{}),
		exitWhenSynced: exitWhenSynced,
	}
	stack.RegisterAPIs(s.APIs())
	stack.RegisterLifecycle(s)
	return s, nil
}

// APIs return the collection of RPC services the ethereum package offers.
// NOTE, some of these services probably need to be moved to somewhere else.
func (s *Syncer) APIs() []rpc.API {
	return []rpc.API{
		{
			Namespace: "debug",
			Service:   NewAPI(s),
		},
	}
}

// run is the main loop that monitors sync requests from users and initiates
// sync operations when necessary. It also checks whether the specified target
// has been reached and shuts down Geth if requested by the user.
func (s *Syncer) run() {
	defer s.wg.Done()

	var (
		target *types.Header
		ticker = time.NewTicker(time.Second * 5)
	)
	defer ticker.Stop()
	for {
		select {
		case req := <-s.request:
			var (
				resync  bool
				retries int
				logged  bool
			)
			for {
				if retries >= 10 {
					req.errc <- fmt.Errorf("sync target is not available, %x", req.hash)
					break
				}
				select {
				case <-s.closed:
					req.errc <- errors.New("syncer closed")
					return
				default:
				}

				header, err := s.backend.Downloader().GetHeader(req.hash)
				if err != nil {
					if !logged {
						logged = true
						log.Info("Waiting for peers to retrieve sync target", "hash", req.hash)
					}
					time.Sleep(time.Second * time.Duration(retries+1))
					retries++
					continue
				}
				if target != nil && header.Number.Cmp(target.Number) <= 0 {
					req.errc <- fmt.Errorf("stale sync target, current: %d, received: %d", target.Number, header.Number)
					break
				}
				target = header
				resync = true
				break
			}
			if resync {
				if mode := s.backend.Downloader().ConfigSyncMode(); mode != ethconfig.FullSync {
					req.errc <- fmt.Errorf("unsupported syncmode %v, please relaunch geth with --syncmode full", mode)
				} else {
					req.errc <- s.backend.Downloader().BeaconDevSync(target)
				}
			}

		case <-ticker.C:
			if target == nil {
				continue
			}

			// Terminate the node if the target has been reached
			if s.exitWhenSynced {
				if block := s.backend.BlockChain().GetBlockByHash(target.Hash()); block != nil {
					log.Info("Sync target reached", "number", block.NumberU64(), "hash", block.Hash())
					go s.stack.Close() // async since we need to close ourselves
					return
				}
			}

			// Set the finalized and safe markers relative to the current head.
			// The finalized marker is set two epochs behind the target,
			// and the safe marker is set one epoch behind the target.
			head := s.backend.BlockChain().CurrentHeader()
			if head == nil {
				continue
			}
			if header := s.backend.BlockChain().GetHeaderByNumber(head.Number.Uint64() - params.EpochLength*2); header != nil {
				if final := s.backend.BlockChain().CurrentFinalBlock(); final == nil || final.Number.Cmp(header.Number) < 0 {
					s.backend.BlockChain().SetFinalized(header)
				}
			}
			if header := s.backend.BlockChain().GetHeaderByNumber(head.Number.Uint64() - params.EpochLength); header != nil {
				if safe := s.backend.BlockChain().CurrentSafeBlock(); safe == nil || safe.Number.Cmp(header.Number) < 0 {
					s.backend.BlockChain().SetSafe(header)
				}
			}

		case <-s.closed:
			return
		}
	}
}

// Start launches the synchronization service.
func (s *Syncer) Start() error {
	s.wg.Add(1)
	go s.run()
	if s.target == (common.Hash{}) {
		return nil
	}
	return s.Sync(s.target)
}

// Stop terminates the synchronization service and stop all background activities.
// This function can only be called for one time.
func (s *Syncer) Stop() error {
	close(s.closed)
	s.wg.Wait()
	return nil
}

// Sync sets the synchronization target. Notably, setting a target lower than the
// previous one is not allowed, as backward synchronization is not supported.
func (s *Syncer) Sync(hash common.Hash) error {
	req := &syncReq{
		hash: hash,
		errc: make(chan error, 1),
	}
	select {
	case s.request <- req:
		return <-req.errc
	case <-s.closed:
		return errors.New("syncer is closed")
	}
}

// API is the collection of synchronization service APIs for debugging the
// protocol.
type API struct {
	s *Syncer
}

// NewAPI creates a new debug API instance.
func NewAPI(s *Syncer) *API {
	return &API{s: s}
}

// Sync initiates a full sync to the target block hash.
func (api *API) Sync(target common.Hash) error {
	return api.s.Sync(target)
}
