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
	"time"

	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/node"
	"github.com/ethereum/go-ethereum/p2p"
	"github.com/ethereum/go-ethereum/p2p/discover"
	"github.com/ethereum/go-ethereum/p2p/simulations/adapters"
	"github.com/ethereum/go-ethereum/swarm/network"
	"github.com/ethereum/go-ethereum/swarm/network/simulation"
	"github.com/ethereum/go-ethereum/swarm/state"
	"github.com/ethereum/go-ethereum/swarm/storage"
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
	externalStreamName := "externalStream"
	externalStreamSessionAt := uint64(50)
	externalStreamMaxKeys := uint64(100)

	sim := simulation.New(map[string]simulation.ServiceFunc{
		"intervalsStreamer": func(ctx *adapters.ServiceContext, bucket *sync.Map) (s node.Service, cleanup func(), err error) {

			id := ctx.Config.ID
			addr := network.NewAddrFromNodeID(id)
			store, datadir, err := createTestLocalStorageForID(id, addr)
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
	_, err := sim.AddNodesAndConnectChain(nodes)
	if err != nil {
		t.Fatal(err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Second)
	defer cancel()

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

		liveErrC := make(chan error)
		historyErrC := make(chan error)

		if _, err := sim.WaitTillHealthy(ctx, 2); err != nil {
			log.Error("WaitKademlia error: %v", "err", err)
			return err
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
			liveHashesChan, err = getHashes(ctx, registry, storer, NewStream(externalStreamName, "", true))
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
			historyHashesChan, err = getHashes(ctx, registry, storer, NewStream(externalStreamName, "", false))
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
				case <-ctx.Done():
					return
				}
			}
		}()

		err = registry.Subscribe(storer, NewStream(externalStreamName, "", live), history, Top)
		if err != nil {
			return err
		}
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

func getHashes(ctx context.Context, r *Registry, peerID discover.NodeID, s Stream) (chan []byte, error) {
	peer := r.getPeer(peerID)

	client, err := peer.getClient(ctx, s)
	if err != nil {
		return nil, err
	}

	c := client.Client.(*testExternalClient)

	return c.hashes, nil
}

func enableNotifications(r *Registry, peerID discover.NodeID, s Stream) error {
	peer := r.getPeer(peerID)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	client, err := peer.getClient(ctx, s)
	if err != nil {
		return err
	}

	close(client.Client.(*testExternalClient).enableNotificationsC)

	return nil
}

type testExternalClient struct {
	hashes               chan []byte
	db                   *storage.DBAPI
	enableNotificationsC chan struct{}
}

func newTestExternalClient(db *storage.DBAPI) *testExternalClient {
	return &testExternalClient{
		hashes:               make(chan []byte),
		db:                   db,
		enableNotificationsC: make(chan struct{}),
	}
}

func (c *testExternalClient) NeedData(ctx context.Context, hash []byte) func() {
	chunk, _ := c.db.GetOrCreateRequest(ctx, hash)
	if chunk.ReqC == nil {
		return nil
	}
	c.hashes <- hash
	//NOTE: This was failing on go1.9.x with a deadlock.
	//Sometimes this function would just block
	//It is commented now, but it may be well worth after the chunk refactor
	//to re-enable this and see if the problem has been addressed
	/*
		return func() {
			return chunk.WaitToStore()
		}
	*/
	return nil
}

func (c *testExternalClient) BatchDone(Stream, uint64, []byte, []byte) func() (*TakeoverProof, error) {
	return nil
}

func (c *testExternalClient) Close() {}

const testExternalServerBatchSize = 10

type testExternalServer struct {
	t         string
	keyFunc   func(key []byte, index uint64)
	sessionAt uint64
	maxKeys   uint64
}

func newTestExternalServer(t string, sessionAt, maxKeys uint64, keyFunc func(key []byte, index uint64)) *testExternalServer {
	if keyFunc == nil {
		keyFunc = binary.BigEndian.PutUint64
	}
	return &testExternalServer{
		t:         t,
		keyFunc:   keyFunc,
		sessionAt: sessionAt,
		maxKeys:   maxKeys,
	}
}

func (s *testExternalServer) SetNextBatch(from uint64, to uint64) ([]byte, uint64, uint64, *HandoverProof, error) {
	if from == 0 && to == 0 {
		from = s.sessionAt
		to = s.sessionAt + testExternalServerBatchSize
	}
	if to-from > testExternalServerBatchSize {
		to = from + testExternalServerBatchSize - 1
	}
	if from >= s.maxKeys && to > s.maxKeys {
		return nil, 0, 0, nil, io.EOF
	}
	if to > s.maxKeys {
		to = s.maxKeys
	}
	b := make([]byte, HashSize*(to-from+1))
	for i := from; i <= to; i++ {
		s.keyFunc(b[(i-from)*HashSize:(i-from+1)*HashSize], i)
	}
	return b, from, to, nil, nil
}

func (s *testExternalServer) GetData(context.Context, []byte) ([]byte, error) {
	return make([]byte, 4096), nil
}

func (s *testExternalServer) Close() {}
