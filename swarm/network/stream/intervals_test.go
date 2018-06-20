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
	"encoding/binary"
	"fmt"
	"io"
	"sync"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/node"
	"github.com/ethereum/go-ethereum/p2p/discover"
	"github.com/ethereum/go-ethereum/p2p/simulations"
	"github.com/ethereum/go-ethereum/p2p/simulations/adapters"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/ethereum/go-ethereum/swarm/network"
	streamTesting "github.com/ethereum/go-ethereum/swarm/network/stream/testing"
	"github.com/ethereum/go-ethereum/swarm/state"
	"github.com/ethereum/go-ethereum/swarm/storage"
)

var (
	externalStreamName             = "externalStream"
	externalStreamSessionAt uint64 = 50
	externalStreamMaxKeys   uint64 = 100
)

func newIntervalsStreamerService(ctx *adapters.ServiceContext) (node.Service, error) {
	id := ctx.Config.ID
	addr := toAddr(id)
	kad := network.NewKademlia(addr.Over(), network.NewKadParams())
	store := stores[id].(*storage.LocalStore)
	db := storage.NewDBAPI(store)
	delivery := NewDelivery(kad, db)
	deliveries[id] = delivery
	r := NewRegistry(addr, delivery, db, state.NewInmemoryStore(), &RegistryOptions{
		SkipCheck: defaultSkipCheck,
	})

	r.RegisterClientFunc(externalStreamName, func(p *Peer, t string, live bool) (Client, error) {
		return newTestExternalClient(db), nil
	})
	r.RegisterServerFunc(externalStreamName, func(p *Peer, t string, live bool) (Server, error) {
		return newTestExternalServer(t, externalStreamSessionAt, externalStreamMaxKeys, nil), nil
	})

	go func() {
		waitPeerErrC <- waitForPeers(r, 1*time.Second, peerCount(id))
	}()
	return &TestExternalRegistry{r}, nil
}

func TestIntervals(t *testing.T) {
	testIntervals(t, true, nil, false)
	testIntervals(t, false, NewRange(9, 26), false)
	testIntervals(t, true, NewRange(9, 26), false)

	testIntervals(t, true, nil, true)
	testIntervals(t, false, NewRange(9, 26), true)
	testIntervals(t, true, NewRange(9, 26), true)
}

func testIntervals(t *testing.T, live bool, history *Range, skipCheck bool) {
	nodes := 2
	chunkCount := dataChunkCount

	defer setDefaultSkipCheck(defaultSkipCheck)
	defaultSkipCheck = skipCheck

	toAddr = network.NewAddrFromNodeID
	conf := &streamTesting.RunConfig{
		Adapter:        *adapter,
		NodeCount:      nodes,
		ConnLevel:      1,
		ToAddr:         toAddr,
		Services:       services,
		DefaultService: "intervalsStreamer",
	}

	sim, teardown, err := streamTesting.NewSimulation(conf)
	var rpcSubscriptionsWg sync.WaitGroup
	defer func() {
		rpcSubscriptionsWg.Wait()
		teardown()
	}()
	if err != nil {
		t.Fatal(err)
	}

	stores = make(map[discover.NodeID]storage.ChunkStore)
	deliveries = make(map[discover.NodeID]*Delivery)
	for i, id := range sim.IDs {
		stores[id] = sim.Stores[i]
	}

	peerCount = func(id discover.NodeID) int {
		return 1
	}

	fileStore := storage.NewFileStore(sim.Stores[0], storage.NewFileStoreParams())
	size := chunkCount * chunkSize
	_, wait, err := fileStore.Store(io.LimitReader(crand.Reader, int64(size)), int64(size), false)
	wait()
	if err != nil {
		t.Fatal(err)
	}

	errc := make(chan error, 1)
	waitPeerErrC = make(chan error)
	quitC := make(chan struct{})
	defer close(quitC)

	action := func(ctx context.Context) error {
		i := 0
		for err := range waitPeerErrC {
			if err != nil {
				return fmt.Errorf("error waiting for peers: %s", err)
			}
			i++
			if i == nodes {
				break
			}
		}

		id := sim.IDs[1]

		err := sim.CallClient(id, func(client *rpc.Client) error {

			sid := sim.IDs[0]

			doneC, err := streamTesting.WatchDisconnections(id, client, errc, quitC)
			if err != nil {
				return err
			}
			rpcSubscriptionsWg.Add(1)
			go func() {
				<-doneC
				rpcSubscriptionsWg.Done()
			}()
			ctx, cancel := context.WithTimeout(ctx, 100*time.Second)
			defer cancel()

			err = client.CallContext(ctx, nil, "stream_subscribeStream", sid, NewStream(externalStreamName, "", live), history, Top)
			if err != nil {
				return err
			}

			liveErrC := make(chan error)
			historyErrC := make(chan error)

			go func() {
				if !live {
					close(liveErrC)
					return
				}

				var err error
				defer func() {
					liveErrC <- err
				}()

				// live stream
				liveHashesChan := make(chan []byte)
				liveSubscription, err := client.Subscribe(ctx, "stream", liveHashesChan, "getHashes", sid, NewStream(externalStreamName, "", true))
				if err != nil {
					return
				}
				defer liveSubscription.Unsubscribe()

				i := externalStreamSessionAt

				// we have subscribed, enable notifications
				err = client.CallContext(ctx, nil, "stream_enableNotifications", sid, NewStream(externalStreamName, "", true))
				if err != nil {
					return
				}

				for {
					select {
					case hash := <-liveHashesChan:
						h := binary.BigEndian.Uint64(hash)
						if h != i {
							err = fmt.Errorf("expected live hash %d, got %d", i, h)
							return
						}
						i++
						if i > externalStreamMaxKeys {
							return
						}
					case err = <-liveSubscription.Err():
						return
					case <-ctx.Done():
						return
					}
				}
			}()

			go func() {
				if live && history == nil {
					close(historyErrC)
					return
				}

				var err error
				defer func() {
					historyErrC <- err
				}()

				// history stream
				historyHashesChan := make(chan []byte)
				historySubscription, err := client.Subscribe(ctx, "stream", historyHashesChan, "getHashes", sid, NewStream(externalStreamName, "", false))
				if err != nil {
					return
				}
				defer historySubscription.Unsubscribe()

				var i uint64
				historyTo := externalStreamMaxKeys
				if history != nil {
					i = history.From
					if history.To != 0 {
						historyTo = history.To
					}
				}

				// we have subscribed, enable notifications
				err = client.CallContext(ctx, nil, "stream_enableNotifications", sid, NewStream(externalStreamName, "", false))
				if err != nil {
					return
				}

				for {
					select {
					case hash := <-historyHashesChan:
						h := binary.BigEndian.Uint64(hash)
						if h != i {
							err = fmt.Errorf("expected history hash %d, got %d", i, h)
							return
						}
						i++
						if i > historyTo {
							return
						}
					case err = <-historySubscription.Err():
						return
					case <-ctx.Done():
						return
					}
				}
			}()

			if err := <-liveErrC; err != nil {
				return err
			}
			if err := <-historyErrC; err != nil {
				return err
			}

			return nil
		})
		return err
	}
	check := func(ctx context.Context, id discover.NodeID) (bool, error) {
		select {
		case err := <-errc:
			return false, err
		case <-ctx.Done():
			return false, ctx.Err()
		default:
		}
		return true, nil
	}

	conf.Step = &simulations.Step{
		Action:  action,
		Trigger: streamTesting.Trigger(10*time.Millisecond, quitC, sim.IDs[0]),
		Expect: &simulations.Expectation{
			Nodes: sim.IDs[1:1],
			Check: check,
		},
	}
	startedAt := time.Now()
	timeout := 300 * time.Second
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	result, err := sim.Run(ctx, conf)
	finishedAt := time.Now()
	if err != nil {
		t.Fatalf("Setting up simulation failed: %v", err)
	}
	if result.Error != nil {
		t.Fatalf("Simulation failed: %s", result.Error)
	}
	streamTesting.CheckResult(t, result, startedAt, finishedAt)
}
