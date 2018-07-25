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
	"os"
	"sync"
	"testing"

	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/node"
	"github.com/ethereum/go-ethereum/p2p"
	"github.com/ethereum/go-ethereum/p2p/simulations/adapters"
	"github.com/ethereum/go-ethereum/swarm/network"
	"github.com/ethereum/go-ethereum/swarm/network/simulation"
	"github.com/ethereum/go-ethereum/swarm/state"
	"github.com/ethereum/go-ethereum/swarm/storage"
)

var (
	externalStreamName             = "externalStream"
	externalStreamSessionAt uint64 = 50
	externalStreamMaxKeys   uint64 = 100
)

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

	sim := simulation.New(map[string]simulation.ServiceFunc{
		"intervalsStreamer": func(ctx *adapters.ServiceContext, bucket *sync.Map) (s node.Service, cleanup func(), err error) {

			id := ctx.Config.ID
			addr := network.NewAddrFromNodeID(id)
			store, datadir, err := createTestLocalStorageForId(id, addr)
			if err != nil {
				return nil, nil, err
			}
			bucket.Store(bucketKeyStore, store)
			cleanup = func() {
				store.Close()
				os.RemoveAll(datadir)
			}
			localStore := store.(*storage.LocalStore)
			db := storage.NewDBAPI(localStore)
			kad := network.NewKademlia(addr.Over(), network.NewKadParams())
			delivery := NewDelivery(kad, db)

			r := NewRegistry(addr, delivery, db, state.NewInmemoryStore(), &RegistryOptions{
				SkipCheck: skipCheck,
			})
			bucket.Store(bucketKeyRegistry, r)

			r.RegisterClientFunc(externalStreamName, func(p *Peer, t string, live bool) (Client, error) {
				return newTestExternalClient(db), nil
			})
			r.RegisterServerFunc(externalStreamName, func(p *Peer, t string, live bool) (Server, error) {
				return newTestExternalServer(t, externalStreamSessionAt, externalStreamMaxKeys, nil), nil
			})

			fileStore := storage.NewFileStore(localStore, storage.NewFileStoreParams())
			bucket.Store(bucketKeyFileStore, fileStore)

			return r, cleanup, nil

		},
	})
	defer sim.Close()

	log.Info("Adding nodes to simulation")
	_, err := sim.AddNodesAndConnectFull(nodes)
	if err != nil {
		t.Fatal(err)
	}

	ctx := context.Background()

	result := sim.Run(ctx, func(ctx context.Context, sim *simulation.Simulation) error {
		nodeIDs := sim.UpNodeIDs()
		storer := nodeIDs[0]
		checker := nodeIDs[1]

		item, ok := sim.NodeItem(storer, bucketKeyFileStore)
		if !ok {
			return fmt.Errorf("No filestore")
		}
		fileStore := item.(*storage.FileStore)

		size := chunkCount * chunkSize
		_, wait, err := fileStore.Store(ctx, io.LimitReader(crand.Reader, int64(size)), int64(size), false)
		if err != nil {
			log.Error("Store error: %v", "err", err)
			t.Fatal(err)
		}
		err = wait(ctx)
		if err != nil {
			log.Error("Wait error: %v", "err", err)
			t.Fatal(err)
		}

		item, ok = sim.NodeItem(checker, bucketKeyRegistry)
		if !ok {
			return fmt.Errorf("No registry")
		}
		registry := item.(*Registry)
		err = registry.Subscribe(storer, NewStream(externalStreamName, "", live), history, Top)
		if err != nil {
			return err
		}

		liveErrC := make(chan error)
		historyErrC := make(chan error)

		if *waitKademlia {
			if _, err := sim.WaitTillHealthy(ctx, 2); err != nil {
				log.Error("WaitKademlia error: %v", "err", err)
				return err
			}
		}

		log.Debug("Watching for disconnections")
		disconnections := sim.PeerEvents(
			context.Background(),
			sim.NodeIDs(),
			simulation.NewPeerEventsFilter().Type(p2p.PeerEventTypeDrop),
		)

		go func() {
			for d := range disconnections {
				if d.Error != nil {
					log.Error("peer drop", "node", d.NodeID, "peer", d.Event.Peer)
					t.Fatal(d.Error)
				}
			}
		}()

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
			var liveHashesChan chan []byte
			liveHashesChan, err = getHashes(registry, ctx, storer, NewStream(externalStreamName, "", true))
			if err != nil {
				log.Error("Subscription error: %v", "err", err)
				return
			}
			i := externalStreamSessionAt

			// we have subscribed, enable notifications
			err = enableNotifications(registry, storer, NewStream(externalStreamName, "", true))
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
				//case err = <-liveSubscription.Err():
				//	return
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
			var historyHashesChan chan []byte
			historyHashesChan, err = getHashes(registry, ctx, storer, NewStream(externalStreamName, "", false))
			if err != nil {
				return
			}

			var i uint64
			historyTo := externalStreamMaxKeys
			if history != nil {
				i = history.From
				if history.To != 0 {
					historyTo = history.To
				}
			}

			// we have subscribed, enable notifications
			err = enableNotifications(registry, storer, NewStream(externalStreamName, "", false))
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
				//case err = <-historySubscription.Err():
				//	return
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

	if result.Error != nil {
		t.Fatal(result.Error)
	}
}
