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
	"net"
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

var nodeAddrById map[discover.NodeID]*BzzAddr

func TestSyncerSimulation(t *testing.T) {
	testSimulation(t, testSyncBetweenNodes(2, 1, 81000, true, 1))
}

func testSyncBetweenNodes(nodes, conns, size int, skipCheck bool, po uint8) func(adapter adapters.NodeAdapter) (*simulations.StepResult, error) {
	return func(adapter adapters.NodeAdapter) (*simulations.StepResult, error) {
		nodeAddrById = make(map[discover.NodeID]*BzzAddr)
		trigger := func(net *simulations.Network) chan discover.NodeID {
			triggerC := make(chan discover.NodeID)
			ticker := time.NewTicker(500 * time.Millisecond)
			go func() {
				defer ticker.Stop()
				// we are only testing the pivot node (net.Nodes[0]) but simulation needs
				// all nodes to pass the check so we trigger each and the check function
				// will trivially return true
				for i := 1; i < nodes; i++ {
					triggerC <- net.Nodes[i].ID()
				}
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
			dpacs := storage.NewDpaChunkStore(localStores[0].(*storage.LocalStore), func(chunk *storage.Chunk) error { return delivery.RequestFromPeers(chunk.Key[:], skipCheck) })
			dpa := storage.NewDPA(dpacs, storage.NewChunkerParams())
			dpa.Start()
			return func(context.Context) error {
				defer rrdpa.Stop()
				// upload an actual random file of size size
				_, _, err := rrdpa.Store(io.LimitReader(crand.Reader, int64(size)), int64(size))
				if err != nil {
					return err
				}
				// // wait until all chunks stored
				// wait()
				// // assign the fileHash to a global so that it is available for the check function
				// fileHash = hash
				// go func() {
				// 	defer dpa.Stop()
				// 	log.Debug(fmt.Sprintf("retrieve %v", fileHash))
				// 	// start the retrieval on the pivot node - this will spawn retrieve requests for missing chunks
				// 	// we must wait for the peer connections to have started before requesting
				// 	time.Sleep(2 * time.Second)
				// 	n, err := mustReadAll(dpa, fileHash)
				// 	log.Debug(fmt.Sprintf("retrieved %v", fileHash), "read", n, "err", err)
				// }()
				return nil
			}
		}

		check := func(net *simulations.Network, dpa *storage.DPA) func(ctx context.Context, id discover.NodeID) (bool, error) {
			dbAccesses := make([]*DbAccess, nodes)

			for i := 0; i < nodes; i++ {
				dbAccesses[i] = NewDbAccess(localStores[i].(*storage.LocalStore))
			}
			return func(ctx context.Context, id discover.NodeID) (bool, error) {
				var found, total int
				dbAccesses[1].iterator(0, math.MaxUint64, po, func(key storage.Key, index uint64) bool {
					_, err := dbAccesses[0].get(key)
					if err == nil {
						found++
					}
					total++
					return true
				})

				//
				// if id != net.Nodes[0].ID() {
				// 	return true, nil
				// }
				select {
				case <-ctx.Done():
					return false, ctx.Err()
				default:
				}
				return found == total, nil
				// // try to locally retrieve the file to check if retrieve requests have been successful
				// log.Warn(fmt.Sprintf("try to locally retrieve %v", fileHash))
				// total, err := mustReadAll(dpa, fileHash)
				// if err != nil || total != size {
				// 	log.Warn(fmt.Sprintf("number of bytes read %v/%v (error: %v)", total, size, err))
				// 	return false, nil
				// }
				// return true, nil
				// node := net.GetNode(id)
				// if node == nil {
				// 	return false, fmt.Errorf("unknown node: %s", id)
				// }
				// client, err := node.Client()
				// if err != nil {
				// 	return false, fmt.Errorf("error getting node client: %s", err)
				// }
				// var response int
				// if err := client.Call(&response, "test_haslocal", hash); err != nil {
				// 	return false, fmt.Errorf("error getting bzz_has response: %s", err)
				// }
				// log.Debug(fmt.Sprintf("node has: %v\n%v", id, response))
				// return response == 0, nil
			}
		}

		result, err := runSimulation(nodes, conns, "syncer", action, trigger, check, adapter)
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
	kad := NewKademlia(addr.Over(), NewKadParams())
	localStore := localStores[nodeCount]
	dbAccess := NewDbAccess(localStore.(*storage.LocalStore))
	streamer := NewStreamer(NewDelivery(kad, dbAccess))
	log.Warn("!!!!!!!! Registering syncers")
	RegisterIncomingSyncer(streamer, dbAccess)
	RegisterOutgoingSyncer(streamer, dbAccess)
	addrBytes := addr.Address()
	if nodeCount == 0 {
		// the delivery service for the pivot node is assigned globally
		// so that the simulation action call can use it for the
		// swarm enabled dpa
		delivery = streamer.delivery
		addrBytes[0] = 0x0
	} else {
		addrBytes[0] = 0xF0
	}
	addr = &BzzAddr{
		OAddr: addrBytes,
		UAddr: []byte(discover.NewNode(id, net.IP{127, 0, 0, 1}, 30303, 30303).String()),
	}
	nodeAddrById[id] = addr

	//else {
	// 	RegisterOutgoingSyncer(streamer, dbAccess)
	// }
	nodeCount++

	log.Warn("new service created")
	self := &testStreamerService{
		addr:     addr,
		streamer: streamer,
	}
	self.run = self.runSyncer
	return self, nil
}

func (b *testStreamerService) runSyncer(p *p2p.Peer, rw p2p.MsgReadWriter) error {
	bzzPeer := &bzzPeer{
		Peer:      protocols.NewPeer(p, rw, StreamerSpec),
		localAddr: b.addr,
		BzzAddr:   nodeAddrById[p.ID()],
	}
	b.streamer.delivery.overlay.On(bzzPeer)
	defer b.streamer.delivery.overlay.Off(bzzPeer)
	go func() {
		// each node Subscribes to each other's retrieveRequestStream
		// need to wait till an aynchronous process registers the peers in streamer.peers
		// that is used by Subscribe
		time.Sleep(1 * time.Second)
		err := b.streamer.Subscribe(p.ID(), "SYNC", []byte{uint8(1)}, 0, 0, Top, true)
		if err != nil {
			log.Warn("error in subscribe", "err", err)
		}
	}()
	return b.streamer.Run(bzzPeer)
}
