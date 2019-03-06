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

package swarm

import (
	"context"
	"flag"
	"fmt"
	"io/ioutil"
	"math/rand"
	"os"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/swarm/testutil"

	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/node"
	"github.com/ethereum/go-ethereum/p2p/enode"
	"github.com/ethereum/go-ethereum/p2p/simulations/adapters"
	"github.com/ethereum/go-ethereum/swarm/api"
	"github.com/ethereum/go-ethereum/swarm/network/simulation"
	"github.com/ethereum/go-ethereum/swarm/storage"
	"github.com/mattn/go-colorable"
)

var (
	loglevel     = flag.Int("loglevel", 2, "verbosity of logs")
	longrunning  = flag.Bool("longrunning", false, "do run long-running tests")
	waitKademlia = flag.Bool("waitkademlia", false, "wait for healthy kademlia before checking files availability")
)

func init() {
	rand.Seed(time.Now().UnixNano())

	flag.Parse()

	log.Root().SetHandler(log.LvlFilterHandler(log.Lvl(*loglevel), log.StreamHandler(colorable.NewColorableStderr(), log.TerminalFormat(true))))
}

// TestSwarmNetwork runs a series of test simulations with
// static and dynamic Swarm nodes in network simulation, by
// uploading files to every node and retrieving them.
func TestSwarmNetwork(t *testing.T) {
	var tests = []testSwarmNetworkCase{
		{
			name: "10_nodes",
			steps: []testSwarmNetworkStep{
				{
					nodeCount: 10,
				},
			},
			options: &testSwarmNetworkOptions{
				Timeout: 45 * time.Second,
			},
		},
		{
			name: "10_nodes_skip_check",
			steps: []testSwarmNetworkStep{
				{
					nodeCount: 10,
				},
			},
			options: &testSwarmNetworkOptions{
				Timeout:   45 * time.Second,
				SkipCheck: true,
			},
		},
		{
			name: "dec_inc_node_count",
			steps: []testSwarmNetworkStep{
				{
					nodeCount: 3,
				},
				{
					nodeCount: 1,
				},
				{
					nodeCount: 5,
				},
			},
			options: &testSwarmNetworkOptions{
				Timeout: 90 * time.Second,
			},
		},
	}

	if *longrunning {
		tests = append(tests, longRunningCases()...)
	} else if testutil.RaceEnabled {
		tests = shortCaseForRace()

	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			testSwarmNetwork(t, tc.options, tc.steps...)
		})
	}
}

type testSwarmNetworkCase struct {
	name    string
	steps   []testSwarmNetworkStep
	options *testSwarmNetworkOptions
}

// testSwarmNetworkStep is the configuration
// for the state of the simulation network.
type testSwarmNetworkStep struct {
	// number of swarm nodes that must be in the Up state
	nodeCount int
}

// testSwarmNetworkOptions contains optional parameters for running
// testSwarmNetwork.
type testSwarmNetworkOptions struct {
	Timeout   time.Duration
	SkipCheck bool
}

func longRunningCases() []testSwarmNetworkCase {
	return []testSwarmNetworkCase{
		{
			name: "50_nodes",
			steps: []testSwarmNetworkStep{
				{
					nodeCount: 50,
				},
			},
			options: &testSwarmNetworkOptions{
				Timeout: 3 * time.Minute,
			},
		},
		{
			name: "50_nodes_skip_check",
			steps: []testSwarmNetworkStep{
				{
					nodeCount: 50,
				},
			},
			options: &testSwarmNetworkOptions{
				Timeout:   3 * time.Minute,
				SkipCheck: true,
			},
		},
		{
			name: "inc_node_count",
			steps: []testSwarmNetworkStep{
				{
					nodeCount: 2,
				},
				{
					nodeCount: 5,
				},
				{
					nodeCount: 10,
				},
			},
			options: &testSwarmNetworkOptions{
				Timeout: 90 * time.Second,
			},
		},
		{
			name: "dec_node_count",
			steps: []testSwarmNetworkStep{
				{
					nodeCount: 10,
				},
				{
					nodeCount: 6,
				},
				{
					nodeCount: 3,
				},
			},
			options: &testSwarmNetworkOptions{
				Timeout: 90 * time.Second,
			},
		},
		{
			name: "inc_dec_node_count",
			steps: []testSwarmNetworkStep{
				{
					nodeCount: 3,
				},
				{
					nodeCount: 5,
				},
				{
					nodeCount: 25,
				},
				{
					nodeCount: 10,
				},
				{
					nodeCount: 4,
				},
			},
			options: &testSwarmNetworkOptions{
				Timeout: 5 * time.Minute,
			},
		},
		{
			name: "inc_dec_node_count_skip_check",
			steps: []testSwarmNetworkStep{
				{
					nodeCount: 3,
				},
				{
					nodeCount: 5,
				},
				{
					nodeCount: 25,
				},
				{
					nodeCount: 10,
				},
				{
					nodeCount: 4,
				},
			},
			options: &testSwarmNetworkOptions{
				Timeout:   5 * time.Minute,
				SkipCheck: true,
			},
		},
	}
}

func shortCaseForRace() []testSwarmNetworkCase {
	// As for now, Travis with -race can only run 8 nodes
	return []testSwarmNetworkCase{
		{
			name: "8_nodes",
			steps: []testSwarmNetworkStep{
				{
					nodeCount: 8,
				},
			},
			options: &testSwarmNetworkOptions{
				Timeout: 1 * time.Minute,
			},
		},
	}
}

// file represents the file uploaded on a particular node.
type file struct {
	addr   storage.Address
	data   string
	nodeID enode.ID
}

// check represents a reference to a file that is retrieved
// from a particular node.
type check struct {
	key    string
	nodeID enode.ID
}

// testSwarmNetwork is a helper function used for testing different
// static and dynamic Swarm network simulations.
// It is responsible for:
//  - Setting up a Swarm network simulation, and updates the number of nodes within the network on every step according to steps.
//  - Uploading a unique file to every node on every step.
//  - May wait for Kademlia on every node to be healthy.
//  - Checking if a file is retrievable from all nodes.
func testSwarmNetwork(t *testing.T, o *testSwarmNetworkOptions, steps ...testSwarmNetworkStep) {
	t.Helper()

	if o == nil {
		o = new(testSwarmNetworkOptions)
	}

	sim := simulation.New(map[string]simulation.ServiceFunc{
		"swarm": func(ctx *adapters.ServiceContext, bucket *sync.Map) (s node.Service, cleanup func(), err error) {
			config := api.NewConfig()

			dir, err := ioutil.TempDir("", "swarm-network-test-node")
			if err != nil {
				return nil, nil, err
			}
			cleanup = func() {
				err := os.RemoveAll(dir)
				if err != nil {
					log.Error("cleaning up swarm temp dir", "err", err)
				}
			}

			config.Path = dir

			privkey, err := crypto.GenerateKey()
			if err != nil {
				return nil, cleanup, err
			}

			config.Init(privkey)
			config.DeliverySkipCheck = o.SkipCheck
			config.Port = ""

			swarm, err := NewSwarm(config, nil)
			if err != nil {
				return nil, cleanup, err
			}
			bucket.Store(simulation.BucketKeyKademlia, swarm.bzz.Hive.Kademlia)
			log.Info("new swarm", "bzzKey", config.BzzKey, "baseAddr", fmt.Sprintf("%x", swarm.bzz.BaseAddr()))
			return swarm, cleanup, nil
		},
	})
	defer sim.Close()

	ctx := context.Background()
	if o.Timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, o.Timeout)
		defer cancel()
	}

	files := make([]file, 0)

	for i, step := range steps {
		log.Debug("test sync step", "n", i+1, "nodes", step.nodeCount)

		change := step.nodeCount - len(sim.UpNodeIDs())

		if change > 0 {
			_, err := sim.AddNodesAndConnectChain(change)
			if err != nil {
				t.Fatal(err)
			}
		} else if change < 0 {
			_, err := sim.StopRandomNodes(-change)
			if err != nil {
				t.Fatal(err)
			}
		} else {
			t.Logf("step %v: no change in nodes", i)
			continue
		}

		var checkStatusM sync.Map
		var nodeStatusM sync.Map
		var totalFoundCount uint64

		result := sim.Run(ctx, func(ctx context.Context, sim *simulation.Simulation) error {
			nodeIDs := sim.UpNodeIDs()
			rand.Shuffle(len(nodeIDs), func(i, j int) {
				nodeIDs[i], nodeIDs[j] = nodeIDs[j], nodeIDs[i]
			})
			for _, id := range nodeIDs {
				key, data, err := uploadFile(sim.Service("swarm", id).(*Swarm))
				if err != nil {
					return err
				}
				log.Trace("file uploaded", "node", id, "key", key.String())
				files = append(files, file{
					addr:   key,
					data:   data,
					nodeID: id,
				})
			}

			if *waitKademlia {
				if _, err := sim.WaitTillHealthy(ctx); err != nil {
					return err
				}
			}

			// File retrieval check is repeated until all uploaded files are retrieved from all nodes
			// or until the timeout is reached.
			for {
				if retrieve(sim, files, &checkStatusM, &nodeStatusM, &totalFoundCount) == 0 {
					return nil
				}
			}
		})

		if result.Error != nil {
			t.Fatal(result.Error)
		}
		log.Debug("done: test sync step", "n", i+1, "nodes", step.nodeCount)
	}
}

// uploadFile, uploads a short file to the swarm instance
// using the api.Put method.
func uploadFile(swarm *Swarm) (storage.Address, string, error) {
	b := make([]byte, 8)
	_, err := rand.Read(b)
	if err != nil {
		return nil, "", err
	}
	// File data is very short, but it is ensured that its
	// uniqueness is very certain.
	data := fmt.Sprintf("test content %s %x", time.Now().Round(0), b)
	ctx := context.TODO()
	k, wait, err := swarm.api.Put(ctx, data, "text/plain", false)
	if err != nil {
		return nil, "", err
	}
	if wait != nil {
		err = wait(ctx)
	}
	return k, data, err
}

// retrieve is the function that is used for checking the availability of
// uploaded files in testSwarmNetwork test helper function.
func retrieve(
	sim *simulation.Simulation,
	files []file,
	checkStatusM *sync.Map,
	nodeStatusM *sync.Map,
	totalFoundCount *uint64,
) (missing uint64) {
	rand.Shuffle(len(files), func(i, j int) {
		files[i], files[j] = files[j], files[i]
	})

	var totalWg sync.WaitGroup
	errc := make(chan error)

	nodeIDs := sim.UpNodeIDs()

	totalCheckCount := len(nodeIDs) * len(files)

	for _, id := range nodeIDs {
		if _, ok := nodeStatusM.Load(id); ok {
			continue
		}
		start := time.Now()
		var checkCount uint64
		var foundCount uint64

		totalWg.Add(1)

		var wg sync.WaitGroup

		swarm := sim.Service("swarm", id).(*Swarm)
		for _, f := range files {

			checkKey := check{
				key:    f.addr.String(),
				nodeID: id,
			}
			if n, ok := checkStatusM.Load(checkKey); ok && n.(int) == 0 {
				continue
			}

			checkCount++
			wg.Add(1)
			go func(f file, id enode.ID) {
				defer wg.Done()

				log.Debug("api get: check file", "node", id.String(), "key", f.addr.String(), "total files found", atomic.LoadUint64(totalFoundCount))

				r, _, _, _, err := swarm.api.Get(context.TODO(), api.NOOPDecrypt, f.addr, "/")
				if err != nil {
					errc <- fmt.Errorf("api get: node %s, key %s, kademlia %s: %v", id, f.addr, swarm.bzz.Hive, err)
					return
				}
				d, err := ioutil.ReadAll(r)
				if err != nil {
					errc <- fmt.Errorf("api get: read response: node %s, key %s: kademlia %s: %v", id, f.addr, swarm.bzz.Hive, err)
					return
				}
				data := string(d)
				if data != f.data {
					errc <- fmt.Errorf("file contend missmatch: node %s, key %s, expected %q, got %q", id, f.addr, f.data, data)
					return
				}
				checkStatusM.Store(checkKey, 0)
				atomic.AddUint64(&foundCount, 1)
				log.Info("api get: file found", "node", id.String(), "key", f.addr.String(), "content", data, "files found", atomic.LoadUint64(&foundCount))
			}(f, id)
		}

		go func(id enode.ID) {
			defer totalWg.Done()
			wg.Wait()

			atomic.AddUint64(totalFoundCount, foundCount)

			if foundCount == checkCount {
				log.Info("all files are found for node", "id", id.String(), "duration", time.Since(start))
				nodeStatusM.Store(id, 0)
				return
			}
			log.Debug("files missing for node", "id", id.String(), "check", checkCount, "found", foundCount)
		}(id)

	}

	go func() {
		totalWg.Wait()
		close(errc)
	}()

	var errCount int
	for err := range errc {
		if err != nil {
			errCount++
		}
		log.Warn(err.Error())
	}

	log.Info("check stats", "total check count", totalCheckCount, "total files found", atomic.LoadUint64(totalFoundCount), "total errors", errCount)

	return uint64(totalCheckCount) - atomic.LoadUint64(totalFoundCount)
}
