// Copyright 2018 The go-ethereum Authors
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

// +build withserver

package stream

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/node"
	"github.com/ethereum/go-ethereum/p2p"
	"github.com/ethereum/go-ethereum/p2p/enode"
	"github.com/ethereum/go-ethereum/p2p/protocols"
	"github.com/ethereum/go-ethereum/p2p/simulations"
	"github.com/ethereum/go-ethereum/p2p/simulations/adapters"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/ethereum/go-ethereum/swarm/log"
	"github.com/ethereum/go-ethereum/swarm/network"
	"github.com/ethereum/go-ethereum/swarm/network/simulation"
	"github.com/ethereum/go-ethereum/swarm/state"
	"github.com/ethereum/go-ethereum/swarm/storage"
)

/*
The tests in this file need to be executed with

			-tags=withserver

Also, they will stall if executed stand-alone, because they wait
for the visualization frontend to send a POST /runsim message.
*/

//setup the sim, evaluate nodeCount and chunkCount and create the sim
func setupSim(serviceMap map[string]simulation.ServiceFunc) (int, int, *simulation.Simulation) {
	nodeCount := *nodes
	chunkCount := *chunks

	if nodeCount == 0 || chunkCount == 0 {
		nodeCount = 32
		chunkCount = 1
	}

	//setup the simulation with server, which means the sim won't run
	//until it receives a POST /runsim from the frontend
	sim := simulation.New(serviceMap).WithServer(":8888")
	return nodeCount, chunkCount, sim
}

//watch for disconnections and wait for healthy
func watchSim(sim *simulation.Simulation) (context.Context, context.CancelFunc) {
	ctx, cancelSimRun := context.WithTimeout(context.Background(), 1*time.Minute)

	if _, err := sim.WaitTillHealthy(ctx); err != nil {
		panic(err)
	}

	disconnections := sim.PeerEvents(
		context.Background(),
		sim.NodeIDs(),
		simulation.NewPeerEventsFilter().Drop(),
	)

	go func() {
		for d := range disconnections {
			log.Error("peer drop", "node", d.NodeID, "peer", d.PeerID)
			panic("unexpected disconnect")
			cancelSimRun()
		}
	}()

	return ctx, cancelSimRun
}

//This test requests bogus hashes into the network
func TestNonExistingHashesWithServer(t *testing.T) {

	nodeCount, _, sim := setupSim(retrievalSimServiceMap)
	defer sim.Close()

	err := sim.UploadSnapshot(fmt.Sprintf("testing/snapshot_%d.json", nodeCount))
	if err != nil {
		panic(err)
	}

	ctx, cancelSimRun := watchSim(sim)
	defer cancelSimRun()

	//in order to get some meaningful visualization, it is beneficial
	//to define a minimum duration of this test
	testDuration := 20 * time.Second

	result := sim.Run(ctx, func(ctx context.Context, sim *simulation.Simulation) error {
		//check on the node's FileStore (netstore)
		id := sim.Net.GetRandomUpNode().ID()
		item, ok := sim.NodeItem(id, bucketKeyFileStore)
		if !ok {
			t.Fatalf("No filestore")
		}
		fileStore := item.(*storage.FileStore)
		//create a bogus hash
		fakeHash := storage.GenerateRandomChunk(1000).Address()
		//try to retrieve it - will propagate RetrieveRequestMsg into the network
		reader, _ := fileStore.Retrieve(context.TODO(), fakeHash)
		if _, err := reader.Size(ctx, nil); err != nil {
			log.Debug("expected error for non-existing chunk")
		}
		//sleep so that the frontend can have something to display
		time.Sleep(testDuration)

		return nil
	})
	if result.Error != nil {
		sendSimTerminatedEvent(sim)
		t.Fatal(result.Error)
	}

	sendSimTerminatedEvent(sim)

}

//send a termination event to the frontend
func sendSimTerminatedEvent(sim *simulation.Simulation) {
	evt := &simulations.Event{
		Type:    EventTypeSimTerminated,
		Control: false,
	}
	sim.Net.Events().Send(evt)
}

//This test is the same as the snapshot sync test,
//but with a HTTP server
//It also sends some custom events so that the frontend
//can visualize messages like SendOfferedMsg, WantedHashesMsg, DeliveryMsg
func TestSnapshotSyncWithServer(t *testing.T) {
	//t.Skip("temporarily disabled as simulations.WaitTillHealthy cannot be trusted")

	//define a wrapper object to be able to pass around data
	wrapper := &netWrapper{}

	nodeCount := *nodes
	chunkCount := *chunks

	if nodeCount == 0 || chunkCount == 0 {
		nodeCount = 32
		chunkCount = 1
	}

	log.Info(fmt.Sprintf("Running the simulation with %d nodes and %d chunks", nodeCount, chunkCount))

	sim := simulation.New(map[string]simulation.ServiceFunc{
		"streamer": func(ctx *adapters.ServiceContext, bucket *sync.Map) (s node.Service, cleanup func(), err error) {
			n := ctx.Config.Node()
			addr := network.NewAddr(n)
			store, datadir, err := createTestLocalStorageForID(n.ID(), addr)
			if err != nil {
				return nil, nil, err
			}
			bucket.Store(bucketKeyStore, store)
			localStore := store.(*storage.LocalStore)
			netStore, err := storage.NewNetStore(localStore, nil)
			if err != nil {
				return nil, nil, err
			}
			kad := network.NewKademlia(addr.Over(), network.NewKadParams())
			delivery := NewDelivery(kad, netStore)
			netStore.NewNetFetcherFunc = network.NewFetcherFactory(dummyRequestFromPeers, true).New

			r := NewRegistry(addr.ID(), delivery, netStore, state.NewInmemoryStore(), &RegistryOptions{
				Retrieval:       RetrievalDisabled,
				Syncing:         SyncingAutoSubscribe,
				SyncUpdateDelay: 3 * time.Second,
			}, nil)

			tr := &testRegistry{
				Registry: r,
				w:        wrapper,
			}

			bucket.Store(bucketKeyRegistry, tr)

			cleanup = func() {
				netStore.Close()
				tr.Close()
				os.RemoveAll(datadir)
			}

			return tr, cleanup, nil
		},
	}).WithServer(":8888") //start with the HTTP server

	nodeCount, chunkCount, sim := setupSim(simServiceMap)
	defer sim.Close()

	log.Info("Initializing test config")

	conf := &synctestConfig{}
	//map of discover ID to indexes of chunks expected at that ID
	conf.idToChunksMap = make(map[enode.ID][]int)
	//map of overlay address to discover ID
	conf.addrToIDMap = make(map[string]enode.ID)
	//array where the generated chunk hashes will be stored
	conf.hashes = make([]storage.Address, 0)
	//pass the network to the wrapper object
	wrapper.setNetwork(sim.Net)
	err := sim.UploadSnapshot(fmt.Sprintf("testing/snapshot_%d.json", nodeCount))
	if err != nil {
		panic(err)
	}

	ctx, cancelSimRun := watchSim(sim)
	defer cancelSimRun()

	//run the sim
	result := runSim(conf, ctx, sim, chunkCount)

	//send terminated event
	evt := &simulations.Event{
		Type:    EventTypeSimTerminated,
		Control: false,
	}
	go sim.Net.Events().Send(evt)

	if result.Error != nil {
		panic(result.Error)
	}
	log.Info("Simulation ended")
}

//testRegistry embeds registry
//it allows to replace the protocol run function
type testRegistry struct {
	*Registry
	w *netWrapper
}

//Protocols replaces the protocol's run function
func (tr *testRegistry) Protocols() []p2p.Protocol {
	regProto := tr.Registry.Protocols()
	//set the `stream` protocol's run function with the testRegistry's one
	regProto[0].Run = tr.runProto
	return regProto
}

//runProto is the new overwritten protocol's run function for this test
func (tr *testRegistry) runProto(p *p2p.Peer, rw p2p.MsgReadWriter) error {
	//create a custom rw message ReadWriter
	testRw := &testMsgReadWriter{
		MsgReadWriter: rw,
		Peer:          p,
		w:             tr.w,
		Registry:      tr.Registry,
	}
	//now run the actual upper layer `Registry`'s protocol function
	return tr.runProtocol(p, testRw)
}

//testMsgReadWriter is a custom rw
//it will allow us to re-use the message twice
type testMsgReadWriter struct {
	*Registry
	p2p.MsgReadWriter
	*p2p.Peer
	w *netWrapper
}

//netWrapper wrapper object so we can pass data around
type netWrapper struct {
	net *simulations.Network
}

//set the network to the wrapper for later use (used inside the custom rw)
func (w *netWrapper) setNetwork(n *simulations.Network) {
	w.net = n
}

//get he network from the wrapper (used inside the custom rw)
func (w *netWrapper) getNetwork() *simulations.Network {
	return w.net
}

// ReadMsg reads a message from the underlying MsgReadWriter and emits a
// "message received" event
//we do this because we are interested in the Payload of the message for custom use
//in this test, but messages can only be consumed once (stream io.Reader)
func (ev *testMsgReadWriter) ReadMsg() (p2p.Msg, error) {
	//read the message from the underlying rw
	msg, err := ev.MsgReadWriter.ReadMsg()
	if err != nil {
		return msg, err
	}

	//don't do anything with message codes we actually are not needing/reading
	subCodes := []uint64{1, 2, 10}
	found := false
	for _, c := range subCodes {
		if c == msg.Code {
			found = true
		}
	}
	//just return if not a msg code we are interested in
	if !found {
		return msg, nil
	}

	//we use a io.TeeReader so that we can read the message twice
	//the Payload is a io.Reader, so if we read from it, the actual protocol handler
	//cannot access it anymore.
	//But we need that handler to be able to consume the message as normal,
	//as if we would not do anything here with that message
	var buf bytes.Buffer
	tee := io.TeeReader(msg.Payload, &buf)

	mcp := &p2p.Msg{
		Code:       msg.Code,
		Size:       msg.Size,
		ReceivedAt: msg.ReceivedAt,
		Payload:    tee,
	}
	//assign the copy for later use
	msg.Payload = &buf

	//now let's look into the message
	var wmsg protocols.WrappedMsg
	err = mcp.Decode(&wmsg)
	if err != nil {
		log.Error(err.Error())
		return msg, err
	}
	//create a new message from the code
	val, ok := ev.Registry.GetSpec().NewMsg(mcp.Code)
	if !ok {
		return msg, errors.New(fmt.Sprintf("Invalid message code: %v", msg.Code))
	}
	//decode it
	if err := rlp.DecodeBytes(wmsg.Payload, val); err != nil {
		return msg, errors.New(fmt.Sprintf("Decoding error <= %v: %v", msg, err))
	}
	//now for every message type we are interested in, create a custom event and send it
	var evt *simulations.Event
	switch val := val.(type) {
	case *OfferedHashesMsg:
		evt = &simulations.Event{
			Type:    EventTypeChunkOffered,
			Node:    ev.w.getNetwork().GetNode(ev.ID()),
			Control: false,
			Data:    val.Hashes,
		}
	case *WantedHashesMsg:
		evt = &simulations.Event{
			Type:    EventTypeChunkWanted,
			Node:    ev.w.getNetwork().GetNode(ev.ID()),
			Control: false,
		}
	case *ChunkDeliveryMsgSyncing:
		evt = &simulations.Event{
			Type:    EventTypeChunkDelivered,
			Node:    ev.w.getNetwork().GetNode(ev.ID()),
			Control: false,
			Data:    val.Addr.String(),
		}
	}
	if evt != nil {
		//send custom event to feed; frontend will listen to it and display
		ev.w.getNetwork().Events().Send(evt)
	}
	return msg, nil
}
