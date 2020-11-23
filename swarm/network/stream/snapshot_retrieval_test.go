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
	"context"
	"fmt"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/node"
	"github.com/ethereum/go-ethereum/p2p/discover"
	"github.com/ethereum/go-ethereum/p2p/simulations/adapters"
	"github.com/ethereum/go-ethereum/swarm/log"
	"github.com/ethereum/go-ethereum/swarm/network"
	"github.com/ethereum/go-ethereum/swarm/network/simulation"
	"github.com/ethereum/go-ethereum/swarm/state"
	"github.com/ethereum/go-ethereum/swarm/storage"
)

//constants for random file generation
const (
	minFileSize = 2
	maxFileSize = 40
)

//This test is a retrieval test for nodes.
//A configurable number of nodes can be
//provided to the test.
//Files are uploaded to nodes, other nodes try to retrieve the file
//Number of nodes can be provided via commandline too.
func TestFileRetrieval(t *testing.T) {
	if *nodes != 0 {
		err := runFileRetrievalTest(*nodes)
		if err != nil {
			t.Fatal(err)
		}
	} else {
		nodeCnt := []int{16}
		//if the `longrunning` flag has been provided
		//run more test combinations
		if *longrunning {
			nodeCnt = append(nodeCnt, 32, 64, 128)
		}
		for _, n := range nodeCnt {
			err := runFileRetrievalTest(n)
			if err != nil {
				t.Fatal(err)
			}
		}
	}
}

//This test is a retrieval test for nodes.
//One node is randomly selected to be the pivot node.
//A configurable number of chunks and nodes can be
//provided to the test, the number of chunks is uploaded
//to the pivot node and other nodes try to retrieve the chunk(s).
//Number of chunks and nodes can be provided via commandline too.
func TestRetrieval(t *testing.T) {
	//if nodes/chunks have been provided via commandline,
	//run the tests with these values
	if *nodes != 0 && *chunks != 0 {
		err := runRetrievalTest(*chunks, *nodes)
		if err != nil {
			t.Fatal(err)
		}
	} else {
		var nodeCnt []int
		var chnkCnt []int
		//if the `longrunning` flag has been provided
		//run more test combinations
		if *longrunning {
			nodeCnt = []int{16, 32, 128}
			chnkCnt = []int{4, 32, 256}
		} else {
			//default test
			nodeCnt = []int{16}
			chnkCnt = []int{32}
		}
		for _, n := range nodeCnt {
			for _, c := range chnkCnt {
				err := runRetrievalTest(c, n)
				if err != nil {
					t.Fatal(err)
				}
			}
		}
	}
}

/*

The test loads a snapshot file to construct the swarm network,
assuming that the snapshot file identifies a healthy
kademlia network. Nevertheless a health check runs in the
simulation's `action` function.

The snapshot should have 'streamer' in its service list.
*/
func runFileRetrievalTest(nodeCount int) error {
	sim := simulation.New(map[string]simulation.ServiceFunc{
		"streamer": func(ctx *adapters.ServiceContext, bucket *sync.Map) (s node.Service, cleanup func(), err error) {

			id := ctx.Config.ID
			addr := network.NewAddrFromNodeID(id)
			store, datadir, err := createTestLocalStorageForID(id, addr)
			if err != nil {
				return nil, nil, err
			}
			bucket.Store(bucketKeyStore, store)
			cleanup = func() {
				os.RemoveAll(datadir)
				store.Close()
			}
			localStore := store.(*storage.LocalStore)
			db := storage.NewDBAPI(localStore)
			kad := network.NewKademlia(addr.Over(), network.NewKadParams())
			delivery := NewDelivery(kad, db)

			r := NewRegistry(addr, delivery, db, state.NewInmemoryStore(), &RegistryOptions{
				DoSync:          true,
				SyncUpdateDelay: 3 * time.Second,
			})

			fileStore := storage.NewFileStore(storage.NewNetStore(localStore, nil), storage.NewFileStoreParams())
			bucket.Store(bucketKeyFileStore, fileStore)

			return r, cleanup, nil

		},
	})
	defer sim.Close()

	log.Info("Initializing test config")

	conf := &synctestConfig{}
	//map of discover ID to indexes of chunks expected at that ID
	conf.idToChunksMap = make(map[discover.NodeID][]int)
	//map of overlay address to discover ID
	conf.addrToIDMap = make(map[string]discover.NodeID)
	//array where the generated chunk hashes will be stored
	conf.hashes = make([]storage.Address, 0)

	err := sim.UploadSnapshot(fmt.Sprintf("testing/snapshot_%d.json", nodeCount))
	if err != nil {
		return err
	}

	ctx, cancelSimRun := context.WithTimeout(context.Background(), 1*time.Minute)
	defer cancelSimRun()

	result := sim.Run(ctx, func(ctx context.Context, sim *simulation.Simulation) error {
		nodeIDs := sim.UpNodeIDs()
		for _, n := range nodeIDs {
			//get the kademlia overlay address from this ID
			a := network.ToOverlayAddr(n.Bytes())
			//append it to the array of all overlay addresses
			conf.addrs = append(conf.addrs, a)
			//the proximity calculation is on overlay addr,
			//the p2p/simulations check func triggers on discover.NodeID,
			//so we need to know which overlay addr maps to which nodeID
			conf.addrToIDMap[string(a)] = n
		}

		//an array for the random files
		var randomFiles []string
		//channel to signal when the upload has finished
		//uploadFinished := make(chan struct{})
		//channel to trigger new node checks

		conf.hashes, randomFiles, err = uploadFilesToNodes(sim)
		if err != nil {
			return err
		}
		if _, err := sim.WaitTillHealthy(ctx, 2); err != nil {
			return err
		}

		// File retrieval check is repeated until all uploaded files are retrieved from all nodes
		// or until the timeout is reached.
		allSuccess := false
		for !allSuccess {
			for _, id := range nodeIDs {
				//for each expected chunk, check if it is in the local store
				localChunks := conf.idToChunksMap[id]
				localSuccess := true
				for _, ch := range localChunks {
					//get the real chunk by the index in the index array
					chunk := conf.hashes[ch]
					log.Trace(fmt.Sprintf("node has chunk: %s:", chunk))
					//check if the expected chunk is indeed in the localstore
					var err error
					//check on the node's FileStore (netstore)
					item, ok := sim.NodeItem(id, bucketKeyFileStore)
					if !ok {
						return fmt.Errorf("No registry")
					}
					fileStore := item.(*storage.FileStore)
					//check all chunks
					for i, hash := range conf.hashes {
						reader, _ := fileStore.Retrieve(context.TODO(), hash)
						//check that we can read the file size and that it corresponds to the generated file size
						if s, err := reader.Size(ctx, nil); err != nil || s != int64(len(randomFiles[i])) {
							allSuccess = false
							log.Warn("Retrieve error", "err", err, "hash", hash, "nodeId", id)
						} else {
							log.Debug(fmt.Sprintf("File with root hash %x successfully retrieved", hash))
						}
					}
					if err != nil {
						log.Warn(fmt.Sprintf("Chunk %s NOT found for id %s", chunk, id))
						localSuccess = false
					} else {
						log.Debug(fmt.Sprintf("Chunk %s IS FOUND for id %s", chunk, id))
					}
				}
				allSuccess = localSuccess
			}
		}
		if !allSuccess {
			return fmt.Errorf("Not all chunks succeeded!")
		}
		return nil
	})

	if result.Error != nil {
		return result.Error
	}

	return nil
}

/*
The test generates the given number of chunks.

The test loads a snapshot file to construct the swarm network,
assuming that the snapshot file identifies a healthy
kademlia network. Nevertheless a health check runs in the
simulation's `action` function.

The snapshot should have 'streamer' in its service list.
*/
func runRetrievalTest(chunkCount int, nodeCount int) error {
	sim := simulation.New(map[string]simulation.ServiceFunc{
		"streamer": func(ctx *adapters.ServiceContext, bucket *sync.Map) (s node.Service, cleanup func(), err error) {

			id := ctx.Config.ID
			addr := network.NewAddrFromNodeID(id)
			store, datadir, err := createTestLocalStorageForID(id, addr)
			if err != nil {
				return nil, nil, err
			}
			bucket.Store(bucketKeyStore, store)
			cleanup = func() {
				os.RemoveAll(datadir)
				store.Close()
			}
			localStore := store.(*storage.LocalStore)
			db := storage.NewDBAPI(localStore)
			kad := network.NewKademlia(addr.Over(), network.NewKadParams())
			delivery := NewDelivery(kad, db)

			r := NewRegistry(addr, delivery, db, state.NewInmemoryStore(), &RegistryOptions{
				DoSync:          true,
				SyncUpdateDelay: 0,
			})

			fileStore := storage.NewFileStore(storage.NewNetStore(localStore, nil), storage.NewFileStoreParams())
			bucketKeyFileStore = simulation.BucketKey("filestore")
			bucket.Store(bucketKeyFileStore, fileStore)

			return r, cleanup, nil

		},
	})
	defer sim.Close()

	conf := &synctestConfig{}
	//map of discover ID to indexes of chunks expected at that ID
	conf.idToChunksMap = make(map[discover.NodeID][]int)
	//map of overlay address to discover ID
	conf.addrToIDMap = make(map[string]discover.NodeID)
	//array where the generated chunk hashes will be stored
	conf.hashes = make([]storage.Address, 0)

	err := sim.UploadSnapshot(fmt.Sprintf("testing/snapshot_%d.json", nodeCount))
	if err != nil {
		return err
	}

	ctx := context.Background()
	result := sim.Run(ctx, func(ctx context.Context, sim *simulation.Simulation) error {
		nodeIDs := sim.UpNodeIDs()
		for _, n := range nodeIDs {
			//get the kademlia overlay address from this ID
			a := network.ToOverlayAddr(n.Bytes())
			//append it to the array of all overlay addresses
			conf.addrs = append(conf.addrs, a)
			//the proximity calculation is on overlay addr,
			//the p2p/simulations check func triggers on discover.NodeID,
			//so we need to know which overlay addr maps to which nodeID
			conf.addrToIDMap[string(a)] = n
		}

		//an array for the random files
		var randomFiles []string
		//this is the node selected for upload
		node := sim.RandomUpNode()
		item, ok := sim.NodeItem(node.ID, bucketKeyStore)
		if !ok {
			return fmt.Errorf("No localstore")
		}
		lstore := item.(*storage.LocalStore)
		conf.hashes, err = uploadFileToSingleNodeStore(node.ID, chunkCount, lstore)
		if err != nil {
			return err
		}
		if _, err := sim.WaitTillHealthy(ctx, 2); err != nil {
			return err
		}

		// File retrieval check is repeated until all uploaded files are retrieved from all nodes
		// or until the timeout is reached.
		allSuccess := false
		for !allSuccess {
			for _, id := range nodeIDs {
				//for each expected chunk, check if it is in the local store
				localChunks := conf.idToChunksMap[id]
				localSuccess := true
				for _, ch := range localChunks {
					//get the real chunk by the index in the index array
					chunk := conf.hashes[ch]
					log.Trace(fmt.Sprintf("node has chunk: %s:", chunk))
					//check if the expected chunk is indeed in the localstore
					var err error
					//check on the node's FileStore (netstore)
					item, ok := sim.NodeItem(id, bucketKeyFileStore)
					if !ok {
						return fmt.Errorf("No registry")
					}
					fileStore := item.(*storage.FileStore)
					//check all chunks
					for i, hash := range conf.hashes {
						reader, _ := fileStore.Retrieve(context.TODO(), hash)
						//check that we can read the file size and that it corresponds to the generated file size
						if s, err := reader.Size(ctx, nil); err != nil || s != int64(len(randomFiles[i])) {
							allSuccess = false
							log.Warn("Retrieve error", "err", err, "hash", hash, "nodeId", id)
						} else {
							log.Debug(fmt.Sprintf("File with root hash %x successfully retrieved", hash))
						}
					}
					if err != nil {
						log.Warn(fmt.Sprintf("Chunk %s NOT found for id %s", chunk, id))
						localSuccess = false
					} else {
						log.Debug(fmt.Sprintf("Chunk %s IS FOUND for id %s", chunk, id))
					}
				}
				allSuccess = localSuccess
			}
		}
		if !allSuccess {
			return fmt.Errorf("Not all chunks succeeded!")
		}
		return nil
	})

	if result.Error != nil {
		return result.Error
	}

	return nil
}
