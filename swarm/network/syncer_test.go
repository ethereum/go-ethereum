// Copyright 2016 The go-ethereum Authors
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

package network

import (
	"context"
	crand "crypto/rand"
	"fmt"
	"io"
	"math"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/node"
	"github.com/ethereum/go-ethereum/p2p"
	"github.com/ethereum/go-ethereum/p2p/discover"
	"github.com/ethereum/go-ethereum/p2p/protocols"
	"github.com/ethereum/go-ethereum/p2p/simulations"
	"github.com/ethereum/go-ethereum/p2p/simulations/adapters"
	"github.com/ethereum/go-ethereum/swarm/storage"
)

func TestSyncerSimulation(t *testing.T) {
	testSimulation(t, testSyncBetweenNodes(2, 1, 81000, true, 1))
	testSimulation(t, testSyncBetweenNodes(3, 1, 81000, true, 1))
}

func testSyncBetweenNodes(nodes, conns, size int, skipCheck bool, po uint8) func(adapter adapters.NodeAdapter) (*simulations.StepResult, error) {
	return func(adapter adapters.NodeAdapter) (*simulations.StepResult, error) {
		trigger := func(net *simulations.Network) chan discover.NodeID {
			triggerC := make(chan discover.NodeID)
			ticker := time.NewTicker(500 * time.Millisecond)
			go func() {
				defer ticker.Stop()
				// we are only testing the pivot node (net.Nodes[0])
				for range ticker.C {
					triggerC <- net.Nodes[0].ID()
				}
			}()
			return triggerC
		}

		action := func(net *simulations.Network) func(context.Context) error {
			// here we distribute chunks of a random file into localstores of nodes 1 to nodes
			rrdpa := storage.NewDPA(newRoundRobinStore(localStores[1:]...), storage.NewChunkerParams())
			rrdpa.Start()
			// create a retriever dpa for the pivot node
			return func(context.Context) error {
				defer rrdpa.Stop()
				// upload an actual random file of size size
				_, wait, err := rrdpa.Store(io.LimitReader(crand.Reader, int64(size)), int64(size))
				if err != nil {
					return err
				}
				// wait until all chunks stored
				wait()
				return nil
			}
		}

		check := func(net *simulations.Network, dpa *storage.DPA) func(ctx context.Context, id discover.NodeID) (bool, error) {
			dbAccesses := make([]*DbAccess, nodes)

			for i := 0; i < nodes; i++ {
				dbAccesses[i] = NewDbAccess(localStores[i].(*storage.LocalStore))
			}
			return func(ctx context.Context, id discover.NodeID) (bool, error) {
				if id != net.Nodes[0].ID() {
					return true, nil
				}
				select {
				case <-ctx.Done():
					return false, ctx.Err()
				default:
				}

				var found, total int
				for i := 1; i < nodes; i++ {
					dbAccesses[i].iterator(0, math.MaxUint64, po, func(key storage.Key, index uint64) bool {
						_, err := dbAccesses[0].get(key)
						if err == nil {
							found++
						}
						total++
						return true
					})
				}
				log.Debug("sync check", "bin", po, "found", found, "total", total)
				return found == total, nil
			}
		}
		toAddr := func(id discover.NodeID) *BzzAddr {
			addr := NewAddrFromNodeID(id)
			addr.OAddr[0] = byte(0)
			return addr
		}

		result, err := runSimulation(nodes, conns, "syncer", toAddr, action, trigger, check, adapter)
		if err != nil {
			return nil, fmt.Errorf("Setting up simulation failed: %v", err)
		}
		if result.Error != nil {
			return nil, fmt.Errorf("Simulation failed: %s", result.Error)
		}
		return result, err
	}
}

func newSyncerService(ctx *adapters.ServiceContext) (node.Service, error) {
	id := ctx.Config.ID
	addr := NewAddrFromNodeID(id)
	// for the test we make all peers share 8 bits so that syncing full bins make sense
	addr.OAddr[0] = byte(0)
	kad := NewKademlia(addr.Over(), NewKadParams())
	localStore := localStores[nodeCount]
	dbAccess := NewDbAccess(localStore.(*storage.LocalStore))
	streamer := NewStreamer(NewDelivery(kad, dbAccess))
	RegisterIncomingSyncer(streamer, dbAccess)
	RegisterOutgoingSyncer(streamer, dbAccess)

	self := &testStreamerService{
		index:    nodeCount,
		addr:     addr,
		streamer: streamer,
	}
	self.run = self.runSyncer
	nodeCount++
	return self, nil
}

func (b *testStreamerService) runSyncer(p *p2p.Peer, rw p2p.MsgReadWriter) error {
	addr := NewAddrFromNodeID(p.ID())
	addr.OAddr[0] = byte(0)
	bzzPeer := &bzzPeer{
		Peer:      protocols.NewPeer(p, rw, StreamerSpec),
		localAddr: b.addr,
		BzzAddr:   addr,
	}
	b.streamer.delivery.overlay.On(bzzPeer)
	defer b.streamer.delivery.overlay.Off(bzzPeer)
	// if len(addr) > b.index+1 && bytes.Equal(addrs[b.index+1], addr) {
	go func() {
		// each node Subscribes to each other's retrieveRequestStream
		// need to wait till an aynchronous process registers the peers in streamer.peers
		// that is used by Subscribe
		time.Sleep(1 * time.Second)
		if err := b.streamer.Subscribe(p.ID(), "SYNC", []byte{uint8(1)}, 0, 0, Top, false); err != nil {
			log.Warn("error in subscribe", "err", err)
		}
	}()
	// }
	return b.streamer.Run(bzzPeer)
}
