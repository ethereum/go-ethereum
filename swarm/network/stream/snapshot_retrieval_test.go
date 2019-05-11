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

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"sync"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/node"
	"github.com/ethereum/go-ethereum/p2p/enode"
	"github.com/ethereum/go-ethereum/p2p/simulations/adapters"
	"github.com/ethereum/go-ethereum/swarm/chunk"
	"github.com/ethereum/go-ethereum/swarm/log"
	"github.com/ethereum/go-ethereum/swarm/network/simulation"
	"github.com/ethereum/go-ethereum/swarm/state"
	"github.com/ethereum/go-ethereum/swarm/storage"
	"github.com/ethereum/go-ethereum/swarm/testutil"
)

// constants for random file generation
const (
	minFileSize = 2
	maxFileSize = 40
)

// TestFileRetrieval is a retrieval test for nodes.
// A configurable number of nodes can be
// provided to the test.
// Files are uploaded to nodes, other nodes try to retrieve the file
// Number of nodes can be provided via commandline too.
func TestFileRetrieval(t *testing.T) {
	var nodeCount []int

	if *nodes != 0 {
		nodeCount = []int{*nodes}
	} else {
		nodeCount = []int{16}

		if *longrunning {
			nodeCount = append(nodeCount, 32, 64)
		} else if testutil.RaceEnabled {
			nodeCount = []int{4}
		}

	}

	for _, nc := range nodeCount {
		runFileRetrievalTest(t, nc)
	}
}

// TestPureRetrieval tests pure retrieval without syncing
// A configurable number of nodes and chunks
// can be provided to the test.
// A number of random chunks is generated, then stored directly in
// each node's localstore according to their address.
// Each chunk is supposed to end up at certain nodes
// With retrieval we then make sure that every node can actually retrieve
// the chunks.
func TestPureRetrieval(t *testing.T) {
	var nodeCount []int
	var chunkCount []int

	if *nodes != 0 && *chunks != 0 {
		nodeCount = []int{*nodes}
		chunkCount = []int{*chunks}
	} else {
		nodeCount = []int{16}
		chunkCount = []int{150}

		if *longrunning {
			nodeCount = append(nodeCount, 32, 64)
			chunkCount = append(chunkCount, 32, 256)
		} else if testutil.RaceEnabled {
			nodeCount = []int{4}
			chunkCount = []int{4}
		}

	}

	for _, nc := range nodeCount {
		for _, c := range chunkCount {
			runPureRetrievalTest(t, nc, c)
		}
	}
}

// TestRetrieval tests retrieval of chunks by random nodes.
// One node is randomly selected to be the pivot node.
// A configurable number of chunks and nodes can be
// provided to the test, the number of chunks is uploaded
// to the pivot node and other nodes try to retrieve the chunk(s).
// Number of chunks and nodes can be provided via commandline too.
func TestRetrieval(t *testing.T) {
	// if nodes/chunks have been provided via commandline,
	// run the tests with these values
	if *nodes != 0 && *chunks != 0 {
		runRetrievalTest(t, *chunks, *nodes)
	} else {
		nodeCnt := []int{16}
		chnkCnt := []int{32}

		if *longrunning {
			nodeCnt = []int{16, 32, 64}
			chnkCnt = []int{4, 32, 256}
		} else if testutil.RaceEnabled {
			nodeCnt = []int{4}
			chnkCnt = []int{4}
		}

		for _, n := range nodeCnt {
			for _, c := range chnkCnt {
				t.Run(fmt.Sprintf("TestRetrieval_%d_%d", n, c), func(t *testing.T) {
					runRetrievalTest(t, c, n)
				})
			}
		}
	}
}

var retrievalSimServiceMap = map[string]simulation.ServiceFunc{
	"streamer": func(ctx *adapters.ServiceContext, bucket *sync.Map) (s node.Service, cleanup func(), err error) {
		addr, netStore, delivery, clean, err := newNetStoreAndDelivery(ctx, bucket)
		if err != nil {
			return nil, nil, err
		}

		syncUpdateDelay := 1 * time.Second
		if *longrunning {
			syncUpdateDelay = 3 * time.Second
		}

		r := NewRegistry(addr.ID(), delivery, netStore, state.NewInmemoryStore(), &RegistryOptions{
			Syncing:         SyncingAutoSubscribe,
			SyncUpdateDelay: syncUpdateDelay,
		}, nil)

		cleanup = func() {
			r.Close()
			clean()
		}

		return r, cleanup, nil
	},
}

// runPureRetrievalTest by uploading a snapshot,
// then starting a simulation, distribute chunks to nodes
// and start retrieval.
// The snapshot should have 'streamer' in its service list.
func runPureRetrievalTest(t *testing.T, nodeCount int, chunkCount int) {

	t.Helper()
	// the pure retrieval test needs a different service map, as we want
	// syncing disabled and we don't need to set the syncUpdateDelay
	sim := simulation.New(map[string]simulation.ServiceFunc{
		"streamer": func(ctx *adapters.ServiceContext, bucket *sync.Map) (s node.Service, cleanup func(), err error) {
			addr, netStore, delivery, clean, err := newNetStoreAndDelivery(ctx, bucket)
			if err != nil {
				return nil, nil, err
			}

			r := NewRegistry(addr.ID(), delivery, netStore, state.NewInmemoryStore(), &RegistryOptions{
				Syncing: SyncingDisabled,
			}, nil)

			cleanup = func() {
				r.Close()
				clean()
			}

			return r, cleanup, nil
		},
	},
	)
	defer sim.Close()

	log.Info("Initializing test config", "node count", nodeCount)

	conf := &synctestConfig{}
	//map of discover ID to indexes of chunks expected at that ID
	conf.idToChunksMap = make(map[enode.ID][]int)
	//map of overlay address to discover ID
	conf.addrToIDMap = make(map[string]enode.ID)
	//array where the generated chunk hashes will be stored
	conf.hashes = make([]storage.Address, 0)

	ctx, cancelSimRun := context.WithTimeout(context.Background(), 3*time.Minute)
	defer cancelSimRun()

	filename := fmt.Sprintf("testing/snapshot_%d.json", nodeCount)
	err := sim.UploadSnapshot(ctx, filename)
	if err != nil {
		t.Fatal(err)
	}

	log.Info("Starting simulation")

	result := sim.Run(ctx, func(ctx context.Context, sim *simulation.Simulation) error {
		nodeIDs := sim.UpNodeIDs()
		// first iteration: create addresses
		for _, n := range nodeIDs {
			//get the kademlia overlay address from this ID
			a := n.Bytes()
			//append it to the array of all overlay addresses
			conf.addrs = append(conf.addrs, a)
			//the proximity calculation is on overlay addr,
			//the p2p/simulations check func triggers on enode.ID,
			//so we need to know which overlay addr maps to which nodeID
			conf.addrToIDMap[string(a)] = n
		}

		// now create random chunks
		chunks := storage.GenerateRandomChunks(int64(chunkSize), chunkCount)
		for _, chunk := range chunks {
			conf.hashes = append(conf.hashes, chunk.Address())
		}

		log.Debug("random chunks generated, mapping keys to nodes")

		// map addresses to nodes
		mapKeysToNodes(conf)

		// second iteration: storing chunks at the peer whose
		// overlay address is closest to a particular chunk's hash
		log.Debug("storing every chunk at correspondent node store")
		for _, id := range nodeIDs {
			// for every chunk for this node (which are only indexes)...
			for _, ch := range conf.idToChunksMap[id] {
				item, ok := sim.NodeItem(id, bucketKeyStore)
				if !ok {
					return fmt.Errorf("Error accessing localstore")
				}
				lstore := item.(chunk.Store)
				// ...get the actual chunk
				for _, chnk := range chunks {
					if bytes.Equal(chnk.Address(), conf.hashes[ch]) {
						// ...and store it in the localstore
						if _, err = lstore.Put(ctx, chunk.ModePutUpload, chnk); err != nil {
							return err
						}
					}
				}
			}
		}

		// now try to retrieve every chunk from every node
		log.Debug("starting retrieval")
		cnt := 0

		for _, id := range nodeIDs {
			item, ok := sim.NodeItem(id, bucketKeyFileStore)
			if !ok {
				return fmt.Errorf("No filestore")
			}
			fileStore := item.(*storage.FileStore)
			for _, chunk := range chunks {
				reader, _ := fileStore.Retrieve(context.TODO(), chunk.Address())
				content := make([]byte, chunkSize)
				size, err := reader.Read(content)
				//check chunk size and content
				ok := true
				if err != io.EOF {
					log.Debug("Retrieve error", "err", err, "hash", chunk.Address(), "nodeId", id)
					ok = false
				}
				if size != chunkSize {
					log.Debug("size not equal chunkSize", "size", size, "hash", chunk.Address(), "nodeId", id)
					ok = false
				}
				// skip chunk "metadata" for chunk.Data()
				if !bytes.Equal(content, chunk.Data()[8:]) {
					log.Debug("content not equal chunk data", "hash", chunk.Address(), "nodeId", id)
					ok = false
				}
				if !ok {
					return fmt.Errorf("Expected test to succeed at first run, but failed with chunk not found")
				}
				log.Debug(fmt.Sprintf("chunk with root hash %x successfully retrieved", chunk.Address()))
				cnt++
			}
		}
		log.Info("retrieval terminated, chunks retrieved: ", "count", cnt)
		return nil

	})

	log.Info("Simulation terminated")

	if result.Error != nil {
		t.Fatal(result.Error)
	}
}

// runFileRetrievalTest loads a snapshot file to construct the swarm network.
// The snapshot should have 'streamer' in its service list.
func runFileRetrievalTest(t *testing.T, nodeCount int) {

	t.Helper()

	sim := simulation.New(retrievalSimServiceMap)
	defer sim.Close()

	log.Info("Initializing test config", "node count", nodeCount)

	conf := &synctestConfig{}
	//map of discover ID to indexes of chunks expected at that ID
	conf.idToChunksMap = make(map[enode.ID][]int)
	//map of overlay address to discover ID
	conf.addrToIDMap = make(map[string]enode.ID)
	//array where the generated chunk hashes will be stored
	conf.hashes = make([]storage.Address, 0)

	ctx, cancelSimRun := context.WithTimeout(context.Background(), 3*time.Minute)
	defer cancelSimRun()

	filename := fmt.Sprintf("testing/snapshot_%d.json", nodeCount)
	err := sim.UploadSnapshot(ctx, filename)
	if err != nil {
		t.Fatal(err)
	}

	log.Info("Starting simulation")

	result := sim.Run(ctx, func(ctx context.Context, sim *simulation.Simulation) error {
		nodeIDs := sim.UpNodeIDs()
		for _, n := range nodeIDs {
			//get the kademlia overlay address from this ID
			a := n.Bytes()
			//append it to the array of all overlay addresses
			conf.addrs = append(conf.addrs, a)
			//the proximity calculation is on overlay addr,
			//the p2p/simulations check func triggers on enode.ID,
			//so we need to know which overlay addr maps to which nodeID
			conf.addrToIDMap[string(a)] = n
		}

		//an array for the random files
		var randomFiles []string

		conf.hashes, randomFiles, err = uploadFilesToNodes(sim)
		if err != nil {
			return err
		}

		log.Info("network healthy, start file checks")

		// File retrieval check is repeated until all uploaded files are retrieved from all nodes
		// or until the timeout is reached.
	REPEAT:
		for {
			for _, id := range nodeIDs {
				//for each expected file, check if it is in the local store
				item, ok := sim.NodeItem(id, bucketKeyFileStore)
				if !ok {
					return fmt.Errorf("No filestore")
				}
				fileStore := item.(*storage.FileStore)
				//check all chunks
				for i, hash := range conf.hashes {
					reader, _ := fileStore.Retrieve(context.TODO(), hash)
					//check that we can read the file size and that it corresponds to the generated file size
					if s, err := reader.Size(ctx, nil); err != nil || s != int64(len(randomFiles[i])) {
						log.Debug("Retrieve error", "err", err, "hash", hash, "nodeId", id)
						time.Sleep(500 * time.Millisecond)
						continue REPEAT
					}
					log.Debug(fmt.Sprintf("File with root hash %x successfully retrieved", hash))
				}
			}
			return nil
		}
	})

	log.Info("Simulation terminated")

	if result.Error != nil {
		t.Fatal(result.Error)
	}
}

// runRetrievalTest generates the given number of chunks.
// The test loads a snapshot file to construct the swarm network.
// The snapshot should have 'streamer' in its service list.
func runRetrievalTest(t *testing.T, chunkCount int, nodeCount int) {

	t.Helper()

	sim := simulation.New(retrievalSimServiceMap)
	defer sim.Close()

	conf := &synctestConfig{}
	//map of discover ID to indexes of chunks expected at that ID
	conf.idToChunksMap = make(map[enode.ID][]int)
	//map of overlay address to discover ID
	conf.addrToIDMap = make(map[string]enode.ID)
	//array where the generated chunk hashes will be stored
	conf.hashes = make([]storage.Address, 0)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	filename := fmt.Sprintf("testing/snapshot_%d.json", nodeCount)
	err := sim.UploadSnapshot(ctx, filename)
	if err != nil {
		t.Fatal(err)
	}

	result := sim.Run(ctx, func(ctx context.Context, sim *simulation.Simulation) error {
		nodeIDs := sim.UpNodeIDs()
		for _, n := range nodeIDs {
			//get the kademlia overlay address from this ID
			a := n.Bytes()
			//append it to the array of all overlay addresses
			conf.addrs = append(conf.addrs, a)
			//the proximity calculation is on overlay addr,
			//the p2p/simulations check func triggers on enode.ID,
			//so we need to know which overlay addr maps to which nodeID
			conf.addrToIDMap[string(a)] = n
		}

		//this is the node selected for upload
		node := sim.Net.GetRandomUpNode()
		item, ok := sim.NodeItem(node.ID(), bucketKeyStore)
		if !ok {
			return fmt.Errorf("No localstore")
		}
		lstore := item.(chunk.Store)
		conf.hashes, err = uploadFileToSingleNodeStore(node.ID(), chunkCount, lstore)
		if err != nil {
			return err
		}

		// File retrieval check is repeated until all uploaded files are retrieved from all nodes
		// or until the timeout is reached.
	REPEAT:
		for {
			for _, id := range nodeIDs {
				//for each expected chunk, check if it is in the local store
				//check on the node's FileStore (netstore)
				item, ok := sim.NodeItem(id, bucketKeyFileStore)
				if !ok {
					return fmt.Errorf("No filestore")
				}
				fileStore := item.(*storage.FileStore)
				//check all chunks
				for _, hash := range conf.hashes {
					reader, _ := fileStore.Retrieve(context.TODO(), hash)
					//check that we can read the chunk size and that it corresponds to the generated chunk size
					if s, err := reader.Size(ctx, nil); err != nil || s != int64(chunkSize) {
						log.Debug("Retrieve error", "err", err, "hash", hash, "nodeId", id, "size", s)
						time.Sleep(500 * time.Millisecond)
						continue REPEAT
					}
					log.Debug(fmt.Sprintf("Chunk with root hash %x successfully retrieved", hash))
				}
			}
			// all nodes and files found, exit loop and return without error
			return nil
		}
	})

	if result.Error != nil {
		t.Fatal(result.Error)
	}
}
