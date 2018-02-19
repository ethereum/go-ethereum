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
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/node"
	"github.com/ethereum/go-ethereum/p2p/discover"
	"github.com/ethereum/go-ethereum/p2p/simulations"
	"github.com/ethereum/go-ethereum/p2p/simulations/adapters"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/ethereum/go-ethereum/swarm/network"
	"github.com/ethereum/go-ethereum/swarm/network/stream/intervals"
	streamTesting "github.com/ethereum/go-ethereum/swarm/network/stream/testing"
	"github.com/ethereum/go-ethereum/swarm/storage"
)

var externalStreamName = "externalStream"

func newIntervalsStreamerService(ctx *adapters.ServiceContext) (node.Service, error) {
	id := ctx.Config.ID
	addr := toAddr(id)
	kad := network.NewKademlia(addr.Over(), network.NewKadParams())
	store := stores[id].(*storage.LocalStore)
	db := storage.NewDBAPI(store)
	delivery := NewDelivery(kad, db)
	deliveries[id] = delivery
	netStore := storage.NewNetStore(store, nil)
	hashesChan := make(chan []byte) // this chanel is only for one client, in need for more clients, create a map
	r := NewRegistry(addr, delivery, netStore, intervals.NewMemStore(), defaultSkipCheck)

	r.RegisterClientFunc(externalStreamName, func(p *Peer, t []byte, live bool) (Client, error) {
		return newTestExternalClient(t, hashesChan), nil
	})
	r.RegisterServerFunc(externalStreamName, func(p *Peer, t []byte, live bool) (Server, error) {
		return newTestExternalServer(t), nil
	})

	go func() {
		waitPeerErrC <- waitForPeers(r, 1*time.Second, peerCount(id))
	}()
	return &TestExternalRegistry{r, hashesChan}, nil
}

func XTestIntervals(t *testing.T) {
	nodes := 2
	chunkCount := dataChunkCount
	skipCheck := false

	defaultSkipCheck = skipCheck
	toAddr = network.NewAddrFromNodeID
	conf := &streamTesting.RunConfig{
		Adapter:   *adapter,
		NodeCount: nodes,
		ConnLevel: 1,
		ToAddr:    toAddr,
		Services:  services,
	}

	sim, teardown, err := streamTesting.NewSimulation(conf)
	defer teardown()
	if err != nil {
		t.Fatal(err)
	}

	peerCount = func(id discover.NodeID) int {
		return 1
	}

	dpa := storage.NewDPA(sim.Stores[0], storage.NewChunkerParams())
	dpa.Start()
	size := chunkCount * chunkSize
	_, wait, err := dpa.Store(io.LimitReader(crand.Reader, int64(size)), int64(size))
	wait()
	defer dpa.Stop()
	if err != nil {
		t.Fatal(err)
	}

	errc := make(chan error, 1)
	waitPeerErrC = make(chan error)
	quitC := make(chan struct{})

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

		liveHashesChan := make(chan []byte)
		historyHashesChan := make(chan []byte)
		id := sim.IDs[1]
		err := sim.CallClient(id, func(client *rpc.Client) error {
			err := streamTesting.WatchDisconnections(id, client, errc, quitC)
			if err != nil {
				return err
			}
			ctx, cancel := context.WithTimeout(ctx, 1*time.Second)
			defer cancel()
			sid := sim.IDs[0]
			err = client.CallContext(ctx, nil, "stream_subscribeStream", sid, NewStream(externalStreamName, nil, true), &Range{From: 0, To: 5}, Top)

			if err != nil {
				return err
			}
			// live stream
			_, err = client.Subscribe(ctx, "stream_getHashes", liveHashesChan, sid, NewStream(externalStreamName, nil, true))
			if err != nil {
				return err
			}
			// history stream
			_, err = client.Subscribe(ctx, "stream_getHashes", historyHashesChan, sid, NewStream(externalStreamName, nil, false))
			return err
		})
		if err != nil {
			return err
		}

		go func() {
			for i := uint64(0); i < 5; i++ {
				h := binary.BigEndian.Uint64(<-historyHashesChan)
				if h != i {
					errc <- fmt.Errorf("")
				}
			}
		}()
		return nil
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
			Nodes: sim.IDs[0:1],
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
