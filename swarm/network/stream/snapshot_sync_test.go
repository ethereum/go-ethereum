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
	crand "crypto/rand"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"math/rand"
	"os"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/p2p/discover"
	"github.com/ethereum/go-ethereum/p2p/simulations"
	"github.com/ethereum/go-ethereum/p2p/simulations/adapters"
	"github.com/ethereum/go-ethereum/pot"
	"github.com/ethereum/go-ethereum/swarm/network"
	"github.com/ethereum/go-ethereum/swarm/storage"
)

const testMinProxBinSize = 2

var (
	pof = pot.DefaultPof(256)

	startTime time.Time
	ids       []discover.NodeID
	datadirs  map[discover.NodeID]string
	conf      *synctestConfig
	ppmap     map[discover.NodeID]*network.PeerPot
)

type synctestConfig struct {
	addrs            [][]byte
	chunks           []storage.Key
	retrievalMap     map[string]map[string]time.Duration
	nodesToChunksMap map[string][]int
	chunksToNodesMap map[string][]int
	idToAddrMap      map[discover.NodeID][]byte
	addrToIdMap      map[string]discover.NodeID
}

func init() {
	rand.Seed(time.Now().Unix())

	initSyncTest()
}

//common_test needs to initialize the test in a init() func
//in order for adapters to register the NewStreamerService;
//this service is dependent on some global variables
//we thus need to initialize first as init() as well.
func initSyncTest() {
	//assign the toAddr func so NewStreamerService can build the addr
	toAddr = func(id discover.NodeID) *network.BzzAddr {
		addr := network.NewAddrFromNodeID(id)
		addr.OAddr[0] = byte(0)
		return addr
	}

	//local stores
	stores = make(map[discover.NodeID]storage.ChunkStore)
	//data directories for each node and store
	datadirs = make(map[discover.NodeID]string)
	//deliveries for each node
	deliveries = make(map[discover.NodeID]*Delivery)

	//channel to wait for peers connected
	waitPeerErrC = make(chan error)
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

func TestSyncing_1_16(t *testing.T)     { testSyncing(t, 1, 16) }
func TestSyncing_1_32(t *testing.T)     { testSyncing(t, 1, 32) }
func TestSyncing_1_64(t *testing.T)     { testSyncing(t, 1, 64) }
func TestSyncing_1_128(t *testing.T)    { testSyncing(t, 1, 128) }
func TestSyncing_1_256(t *testing.T)    { testSyncing(t, 1, 256) }
func TestSyncing_4_16(t *testing.T)     { testSyncing(t, 4, 16) }
func TestSyncing_4_32(t *testing.T)     { testSyncing(t, 4, 32) }
func TestSyncing_4_64(t *testing.T)     { testSyncing(t, 4, 64) }
func TestSyncing_4_128(t *testing.T)    { testSyncing(t, 4, 128) }
func TestSyncing_4_256(t *testing.T)    { testSyncing(t, 4, 256) }
func TestSyncing_8_16(t *testing.T)     { testSyncing(t, 8, 16) }
func TestSyncing_8_32(t *testing.T)     { testSyncing(t, 8, 32) }
func TestSyncing_8_64(t *testing.T)     { testSyncing(t, 8, 64) }
func TestSyncing_8_128(t *testing.T)    { testSyncing(t, 8, 128) }
func TestSyncing_8_256(t *testing.T)    { testSyncing(t, 8, 256) }
func TestSyncing_32_16(t *testing.T)    { testSyncing(t, 32, 16) }
func TestSyncing_32_32(t *testing.T)    { testSyncing(t, 32, 32) }
func TestSyncing_32_64(t *testing.T)    { testSyncing(t, 32, 64) }
func TestSyncing_32_128(t *testing.T)   { testSyncing(t, 32, 128) }
func TestSyncing_32_256(t *testing.T)   { testSyncing(t, 32, 256) }
func TestSyncing_128_16(t *testing.T)   { testSyncing(t, 128, 16) }
func TestSyncing_128_32(t *testing.T)   { testSyncing(t, 128, 32) }
func TestSyncing_128_64(t *testing.T)   { testSyncing(t, 128, 64) }
func TestSyncing_128_128(t *testing.T)  { testSyncing(t, 128, 128) }
func TestSyncing_128_256(t *testing.T)  { testSyncing(t, 128, 256) }
func TestSyncing_256_16(t *testing.T)   { testSyncing(t, 256, 16) }
func TestSyncing_256_32(t *testing.T)   { testSyncing(t, 256, 32) }
func TestSyncing_256_64(t *testing.T)   { testSyncing(t, 256, 64) }
func TestSyncing_256_128(t *testing.T)  { testSyncing(t, 256, 128) }
func TestSyncing_256_256(t *testing.T)  { testSyncing(t, 256, 256) }
func TestSyncing_1024_16(t *testing.T)  { testSyncing(t, 1024, 16) }
func TestSyncing_1024_32(t *testing.T)  { testSyncing(t, 1024, 32) }
func TestSyncing_1024_64(t *testing.T)  { testSyncing(t, 1024, 64) }
func TestSyncing_1024_128(t *testing.T) { testSyncing(t, 1024, 128) }
func TestSyncing_1024_256(t *testing.T) { testSyncing(t, 1024, 256) }

// Benchmarks to test the average time it takes for an N-node ring
// to full a healthy kademlia topology
/*
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

func testSyncing(t *testing.T, chunkCount int, nodeCount int) {
	ids = make([]discover.NodeID, nodeCount)
	err := runSyncTest(chunkCount, nodeCount)
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
func runSyncTest(chunkCount int, nodeCount int) error {

	conf = &synctestConfig{}
	//mapping of nearest node addresses for chunk hashes
	//nodesToChunksMap = make(map[discover.NodeID][]storage.Key)
	conf.retrievalMap = make(map[string]map[string]time.Duration)
	conf.idToAddrMap = make(map[discover.NodeID][]byte)
	conf.addrToIdMap = make(map[string]discover.NodeID)
	//First load the snapshot from the file
	net, err := initNetWithSnapshot(nodeCount)
	if err != nil {
		return err
	}
	defer net.Shutdown()

	//get the nodes of the network
	nodes := net.GetNodes()
	//select one index at random...
	idx := rand.Intn(len(nodes))
	//...and get the the node at that index
	//this is the node selected for upload
	node := nodes[idx]
	//iterate over all nodes...
	for c := 0; c < len(nodes); c++ {
		//create an array of discovery nodeIDS
		ids[c] = nodes[c].ID()
		//and a correspondent array of overlay addresses,
		//later used for chunk proximity calculation
		a := network.ToOverlayAddr(ids[c].Bytes())
		conf.addrs = append(conf.addrs, a)
		//the proximity calculation is on overlay addr,
		//the p2p/simulations check func triggers on discover.NodeID,
		//so we need to know which overlay addr maps to which nodeID
		conf.idToAddrMap[ids[c]] = a
		conf.addrToIdMap[string(a)] = ids[c]
	}

	ppmap = network.NewPeerPot(testMinProxBinSize, ids, conf.addrs)
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

		log.Debug(fmt.Sprintf("Checking node: %s", id))
		//select the local store for the given node
		lstore := stores[id]
		//if there are more than one chunk, test only succeeds if all expected chunks are found
		allSuccess := true
		//this selects which chunks are expected to be found with the given node
		//localChunks := nodesToChunksMap[id]
		localChunks := conf.nodesToChunksMap[string(conf.idToAddrMap[id])]

		//for each expected chunk, check if it is in the local store
		for i := 0; i < len(localChunks); i++ {
			//ignore zero chunks
			chunk := conf.chunks[localChunks[i]]
			if storage.IsZeroKey(chunk) {
				continue
			}
			log.Debug(fmt.Sprintf("node has chunk: %s:", chunk))
			if _, err := lstore.Get(chunk); err != nil {
				log.Error(fmt.Sprintf("Chunk %s NOT found for id %s", chunk, id))
				allSuccess = false
			} else {
				fmt.Println("^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^")
				log.Info(fmt.Sprintf("Chunk %s FOUND for id %s", chunk, id))
			}
		}

		return allSuccess, nil
	}

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

		time.Sleep(10 * time.Second)
		// each node Subscribes to each other's swarmChunkServerStreamName
		for j := 0; j < len(ids); j++ {
			log.Debug(fmt.Sprintf("subscribe: %d", j))
			ctx, cancel := context.WithTimeout(ctx, 1*time.Second)
			defer cancel()
			client, err := net.GetNode(ids[j]).Client()
			if err != nil {
				return err
			}
			err = client.CallContext(ctx, nil, "stream_startSyncing")
			if err != nil {
				log.Error(fmt.Sprintf("FAILED CallContext %v", err))
				return nil
			}
		}
		time.Sleep(10 * time.Second)
		//now upload the chunks to the selected random single node
		conf.chunks, err = uploadFileToSingleNodeStore(node.ID(), chunkCount)
		if err != nil {
			return err
		}
		//finally map chunks to the closest addresses
		conf = mapKeysToNodes(conf)
		log.Debug(fmt.Sprintf("%v", conf.nodesToChunksMap))

		return nil
	}

	//for each tick, run the checks on all nodes
	go func() {
		ticker := time.NewTicker(time.Second * 1)
		for range ticker.C {
			for i := 0; i < len(ids); i++ {
				log.Debug(fmt.Sprintf("triggering step %d, id %s", i, ids[i]))
				trigger <- ids[i]
			}
		}
	}()
	/*
		go func() {
			startTime = time.Now()
			ticker := time.NewTicker(time.Second / 10)
			for range ticker.C {
				checkChunkIsAtNode(conf)
			}
		}()
	*/

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

func (r *TestRegistry) StartSyncing(ctx context.Context) error {
	var err error

	add := r.addr.ID()
	pp := ppmap[add]
	h := r.delivery.overlay.Healthy(pp)
	fmt.Println("----------------------------------")
	fmt.Println(r.delivery.overlay.String())
	fmt.Println(fmt.Sprintf("IS HEALTHY: %t", h.GotNN && h.KnowNN && h.Full))

	pos := make(map[int]discover.NodeID)

	r.delivery.overlay.EachConn(nil, 256, func(addr network.OverlayConn, po int, nn bool) bool {
		lastPO := po
		if nn {
			lastPO = maxPO
		}
		peerId := conf.addrToIdMap[string(addr.Address())]
		fmt.Println(fmt.Sprintf("node %s has conn with %s at po %d and is nn: %t", r.addr.ID(), peerId, po, nn))
		pos[po] = peerId
		for i := po; i <= lastPO; i++ {
			err = r.Subscribe(peerId, NewStream("SYNC", []byte{byte(i)}, false), &Range{From: 0, To: 0}, Top)
			if err != nil {
				log.Error(fmt.Sprintf("Error subscribing! %v", err))
				return false
			}
		}
		return true
	})
	prev := 0
	kad, ok := r.delivery.overlay.(*network.Kademlia)
	if !ok {
		return fmt.Errorf("Not a Kademlia!")
	}

	kad.EachBin(r.addr.Over(), pof, 0, func(po, size int, f func(func(val pot.Val, i int) bool) bool) bool {
		skip := po - prev
		/*
			fmt.Println(prev)
			fmt.Println(po)
			fmt.Println(skip)
		*/
		remember := make(map[int]bool)
		if skip > 1 {
			f(func(val pot.Val, i int) bool {
				//for c := po + 1; c < po+skip; c++ {
				for c := po - 1; c > po-skip; c-- {
					//fmt.Println(c)
					if exists, _ := remember[c]; exists {
						continue
					}
					a := val.(network.OverlayPeer)
					log.Warn(fmt.Sprintf("Request subscription for bin: %d", c))
					log.Debug(fmt.Sprintf("Requesting subscription by: registry %s from peer %s", r.addr.ID(), conf.addrToIdMap[string(a.Address())]))
					err = r.RequestSubscription(conf.addrToIdMap[string(a.Address())], NewStream("SYNC", []byte{byte(uint8(c))}, false), Top)
					if err != nil {
						log.Error(fmt.Sprintf("Error subscribing! %v", err))
						return false
					}
					remember[c] = true
				}
				return true
			})
		}
		prev = po
		return true
	})

	return nil
}

func checkChunkIsAtNode(conf *synctestConfig) {
	allOk := true
	for chunk, nodes := range conf.chunksToNodesMap {
		for _, node := range nodes {
			if ok, _ := stores[conf.addrToIdMap[string(conf.addrs[node])]].Get([]byte(chunk)); ok != nil {
				if len(conf.retrievalMap[chunk]) == 0 {
					conf.retrievalMap[chunk] = make(map[string]time.Duration)
				}
				conf.retrievalMap[chunk][string(conf.addrs[node])] = time.Since(startTime)
			}
			if conf.retrievalMap[chunk][string(conf.addrs[node])] == 0 {
				allOk = false
			}
		}
	}
	if allOk {
		log.Info("All chunks arrived at destination")
		for ch, n := range conf.retrievalMap {
			for a, t := range n {
				log.Info(fmt.Sprintf("Chunk %s at node %s took %d ms", string(ch), string(a), t.Seconds()*1e3))
			}
		}
	}
}

//map chunk keys to addresses which are responsible
func mapKeysToNodes(conf *synctestConfig) *synctestConfig {
	kmap := make(map[string][]int)
	nodemap := make(map[string][]int)
	//build a pot for chunk hashes
	np := pot.NewPot(nil, 0)
	mm := make(map[string]int)
	for i, a := range conf.addrs {
		mm[string(a)] = i
		np, _, _ = pot.Add(np, a, pof)
	}
	//for each address, run EachNeighbour on the chunk hashes pot to identify closest nodes
	fmt.Println(conf.chunks)
	for i := 0; i < len(conf.chunks); i++ {
		pl := 256 //highest proximity
		var nns []int
		np.EachNeighbour([]byte(conf.chunks[i]), pof, func(val pot.Val, po int) bool {
			a := val.([]byte)
			if pl == 256 || pl == po {
				fmt.Println(fmt.Sprintf("appending %s", conf.addrToIdMap[string(a)]))
				nns = append(nns, mm[string(a)])
				nodemap[string(a)] = append(nodemap[string(a)], i)
			}
			if pl == 256 && len(nns) >= testMinProxBinSize {
				pl = po
			}
			return true
		})
		kmap[conf.chunks[i].String()] = nns
		//log.Debug(fmt.Sprintf("Length for id %s: %d",ids[i],len(kmap[ids[i]])))
	}
	for k, v := range nodemap {
		fmt.Print(fmt.Sprintf("Node %s: ", conf.addrToIdMap[k]))
		for _, vv := range v {
			fmt.Println(conf.chunks[vv])
		}
		fmt.Println(conf.addrToIdMap[k])
		fmt.Println("-------------------------------")
	}
	for k, v := range kmap {
		fmt.Print(fmt.Sprintf("Chunk %s: ", k))
		for _, vv := range v {
			fmt.Println(conf.addrToIdMap[string(conf.addrs[vv])])
		}
		fmt.Println("###############################")
	}
	conf.nodesToChunksMap = nodemap
	conf.chunksToNodesMap = kmap
	return conf
}

//upload a file(chunks) to a single local node store
func uploadFileToSingleNodeStore(id discover.NodeID, chunkCount int) ([]storage.Key, error) {
	log.Debug(fmt.Sprintf("Uploading to node id: %s", id))
	lstore := stores[id]
	size := chunkSize
	dpa := storage.NewDPA(lstore, storage.NewChunkerParams())
	dpa.Start()
	var rootkeys []storage.Key
	for i := 0; i < chunkCount; i++ {
		rk, wait, err := dpa.Store(io.LimitReader(crand.Reader, int64(size)), int64(size))
		wait()
		if err != nil {
			return nil, err
		}
		rootkeys = append(rootkeys, (rk))
	}

	defer dpa.Stop()

	return rootkeys, nil
}

//initialize a network from a snapshot
func initNetWithSnapshot(nodeCount int) (*simulations.Network, error) {

	adapter := "sim"

	var a adapters.NodeAdapter
	//add the streamer service to the node adapter
	//discovery["streamer"] = NewStreamerService

	if adapter == "exec" {
		dirname, err := ioutil.TempDir(".", "")
		if err != nil {
			return nil, err
		}
		a = adapters.NewExecAdapter(dirname)
	} else if adapter == "sock" {
		a = adapters.NewSocketAdapter(services)
	} else if adapter == "tcp" {
		a = adapters.NewTCPAdapter(services)
	} else if adapter == "sim" {
		a = adapters.NewSimAdapter(services)
	}

	net := simulations.NewNetwork(a, &simulations.NetworkConfig{
		ID:             "0",
		DefaultService: "streamer",
	})

	f, err := os.Open(fmt.Sprintf("testing/snapshot_%d.json", nodeCount))
	if err != nil {
		return nil, err
	}
	defer f.Close()
	jsonbyte, err := ioutil.ReadAll(f)
	if err != nil {
		return nil, err
	}
	var snap simulations.Snapshot
	err = json.Unmarshal(jsonbyte, &snap)
	if err != nil {
		return nil, err
	}
	err = net.Load(&snap)
	if err != nil {
		return nil, err
	}
	return net, nil
}
