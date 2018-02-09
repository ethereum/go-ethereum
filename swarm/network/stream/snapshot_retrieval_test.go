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
package stream

/*

import (
	"context"
	crand "crypto/rand"
	"flag"
	"fmt"
	"io"
	"math/rand"
//	"os"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/log"
//	"github.com/ethereum/go-ethereum/node"
//	"github.com/ethereum/go-ethereum/pot"
	"github.com/ethereum/go-ethereum/p2p/discover"
	"github.com/ethereum/go-ethereum/p2p/simulations"
//	"github.com/ethereum/go-ethereum/p2p/simulations/adapters"
	"github.com/ethereum/go-ethereum/swarm/network"
	"github.com/ethereum/go-ethereum/swarm/storage"
	//streamTesting "github.com/ethereum/go-ethereum/swarm/network/stream/testing"
)

var rootHash storage.Key


func init() {
	flag.Parse()
	rand.Seed(time.Now().Unix())

	initRetrievalTest()
}


func initRetrievalTest() {
  //assign the toAddr func so NewStreamerService can build the addr
  toAddr = func(id discover.NodeID) *network.BzzAddr {
    addr := network.NewAddrFromNodeID(id)
    addr.OAddr[0] = byte(0)
    return addr
  }

  //nodeCount is needed to load a specific json snapshot file,
  //e.g. "snapshot_16.json"
  nodeCount = 16
  //is used to continuosly provide the current discoverID to NewStreamerService
  //while loading the snapshot
  currentId = 0
  //local stores
	stores        = make(map[discover.NodeID]storage.ChunkStore)
  //data directories for each node and store
	datadirs      = make(map[discover.NodeID]string)
  //deliveries for each node
	deliveries    = make(map[discover.NodeID]*Delivery)
  //the list of the ids loaded from the snapshot
  ids           = make([]discover.NodeID, nodeCount)
  //mapping of nearest node addresses for chunk hashes
  //chunksForAddressesMap = make(map[discover.NodeID][]storage.Key)


	waitPeerErrC  = make(chan error)
	// peerCount function gives the number of peer connections for a nodeID
	// this is needed for the service run function to wait until
	// each protocol  instance runs and the streamer peers are available
	peerCount = func(id discover.NodeID) int {
		if ids[0] == id || ids[len(ids)-1] == id {
			return 1
		}
		return 2
	}
}


func TestRetrieval_4(t *testing.T)   { retrievalTest(t, 4) }
/*
func TestRetrieval_1(t *testing.T)   { retrievalTest(t, 1) }
func TestSyncing_4(t *testing.T) { testSyncing(t, 4) }
func TestSyncing_8(t *testing.T) { testSyncing(t, 8) }
func TestSyncing_32(t *testing.T) { testSyncing(t, 32) }
func TestSyncing_128(t *testing.T) { testSyncing(t, 128) }
func TestSyncing_256(t *testing.T) { testSyncing(t, 256) }
func TestSyncing_1024(t *testing.T) { testSyncing(t,1024) }

// Benchmarks to test the average time it takes for an N-node ring
// to full a healthy kademlia topology
func BenchmarkSyncing_1(b *testing.B)   { benchmarkSyncing(b, 1) }
func BenchmarkSyncing_4(b *testing.B)  { benchmarkSyncing(b, 4) }
func BenchmarkSyncing_8(b *testing.B)  { benchmarkSyncing(b, 8) }
func BenchmarkSyncing_32(b *testing.B)  { benchmarkSyncing(b, 32) }
func BenchmarkSyncing_128(b *testing.B) { benchmarkSyncing(b, 128) }
func BenchmarkSyncing_256(b *testing.B) { benchmarkSyncing(b, 256) }
func BenchmarkSyncing_1024(b *testing.B) { benchmarkSyncing(b, 1024) }

func benchmarkSyncing(b *testing.B, chunkCount int) {
	for i := 0; i < b.N; i++ {
		result, err := testSyncing(b.T, chunkCount)
		if err != nil {
			b.Fatalf("setting up simulation failed", result)
		}
		if result.Error != nil {
			b.Logf("simulation failed: %s", result.Error)
		}
	}
}

*/
/*
func retrievalTest(t *testing.T, chunkCount int) {
  err := runRetrievalTest(chunkCount)
  if err != nil {
    t.Fatal(err)
  }
}

/*
The test generates the given number of chunks,
then uploads these to a random node.
Afterwards for every chunk generated, the nearest node addresses
are identified, syncing is started, and finally we verify
that the nodes closer to the chunk addresses actually do have 
the chunks in their local stores.

The test loads a snapshot file to construct the swarm network, 
assuming that the snapshot file identifies a healthy
kademlia network. The snapshot should have 'streamer' in its service list.
*/
/*
func runRetrievalTest(chunkCount int) error {

  //reset global vars
  resetVars()
  //First load the snapshot from the file
  net,err := initNetWithSnapshot()
  if err != nil {
    return err
  }
	defer net.Shutdown()

  //get the nodes of the network
	nodes := net.GetNodes()
  //select one index at random...
	idx   := rand.Intn(len(nodes))
  //...and get the the node at that index
  //this is the node selected for upload
  node  := nodes[idx]
  //iterate over all nodes...
  for c:=0; c<len(nodes); c++ {
    //create an array of discovery nodeIDS
    ids[c] = nodes[c].ID()
    //and a correspondent array of overlay addresses, 
    //later used for chunk proximity calculation
    addrs = append(addrs, network.ToOverlayAddr(ids[c].Bytes()))
  }

	// channel to signal simulation initialisation with action call complete
	// or node disconnections
	//disconnectC := make(chan error)
	//quitC := make(chan struct{})

  //after the test, clean up local stores initialized with createLocalStoreForId
  defer localStoreCleanup()

	trigger := make(chan discover.NodeID)
  //triggerCheck defines what will be checked during the test
	triggerCheck := func(ctx context.Context, id discover.NodeID) (bool, error) {
		select {
		case <-ctx.Done():
			return false, ctx.Err()
	  //case <-disconnectC:
    //  log.Error("Disconnect event detected")
    //  return false, ctx.Err()
		default:
		}

    log.Warn(fmt.Sprintf("Checking node: %s", id))
    //select the !!!!NETstore!!! for the given node
    /*
    lstore := stores[id]
    if _,err := lstore.Get(rootHash); err !=nil {
      log.Warn("File Not Found")
      return false, nil
    }
    log.Warn("File Found")
    */
/*
    return true, nil
	}

  //for each tick, select a new node to be checked
  ticker := time.NewTicker(time.Second * 1)
  go func() {
        for i:=0;i<len(ids);i++ {
          <-ticker.C
          trigger <- ids[i]
          log.Debug(fmt.Sprintf("triggering step %d, id %s", i, ids[i]))
        }
  }()

	timeout := 300 * time.Second
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
  //define the action to be performed before the test checks: start syncing
	action := func(ctx context.Context) error {
		// need to wait till an aynchronous process registers the peers in streamer.peers
		// that is used by Subscribe
		// the global peerCount function tells how many connections each node has
		// TODO: this is to be reimplemented with peerEvent watcher without global var
		i := 0
		for err := range waitPeerErrC {
			if err != nil {
				return fmt.Errorf("error waiting for peers: %s", err)
			}
			i++
			if i == len(ids)-1 {
				break
			}
		}

		// each node Subscribes to each other's swarmChunkServerStreamName
		for j := 0; j < len(ids); j++ {
      log.Debug(fmt.Sprintf("subscribe: %d",j))
      ctx, cancel := context.WithTimeout(ctx, 1*time.Second)
      defer cancel()
      client,err := net.GetNode(ids[j]).Client()
      if err != nil {
        return err
      }
      //RPC call to subscribe, select bin 0
      //client.CallContext(ctx, nil, "stream_subscribeStream", sid, "SYNC", []byte{0}, 0, 0, Top, false)
      // report disconnect events to the error channel cos peers should not disconnect
      //err = streamTesting.WatchDisconnections(ids[j], client, disconnectC, quitC)
      //if err != nil {
      //  return err
      //}
      // start syncing, i.e., subscribe to upstream peers po 1 bin
      //each node subscribes to the next index, last subscribes to 0
      idx:= j+1
      if j==len(ids)-1 {
        idx = 0
      }
      sid := ids[idx]
      client.CallContext(ctx, nil, "stream_subscribeStream", sid, "SYNC", []byte{0}, 0, 0, Top, false)
    }
    //now upload the chunks to the selected random single node
    rootHash,err = uploadFileToRandomNodeStore(node.ID(), chunkCount)
    if err != nil {
      return err
    }
    //finally map chunks to the closest addresses
    //chunksForAddressesMap = mapIdsToKeys(chunks, ids)
    log.Debug(fmt.Sprintf("%v",chunksForAddressesMap))

		return nil
  }
  //run the simulation
	result := simulations.NewSimulation(net).Run(ctx, &simulations.Step{
		Action:  action,
		Trigger: trigger,
		Expect: &simulations.Expectation{
			Nodes: ids,
			Check: triggerCheck,
		},
	})
  //close(quitC)
	if result.Error != nil {
    return result.Error
	}
  return nil
}

//upload a file(chunks) to a single local node store
func uploadFileToRandomNodeStore(id discover.NodeID, chunkCount int) (storage.Key, error) {
  log.Debug(fmt.Sprintf("Uploading to node id: %s", id))
  lstore := stores[id]
	size := chunkCount * chunkSize
  dpa := storage.NewDPA(lstore, storage.NewChunkerParams())
  dpa.Start()
  rootHash, wait, err := dpa.Store(io.LimitReader(crand.Reader, int64(size)), int64(size))
  wait()
  if err != nil {
    return nil, err
  }

  defer dpa.Stop()

  return rootHash, nil
}
*/
