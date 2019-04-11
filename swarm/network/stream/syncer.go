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
	"strconv"
	"time"

	"github.com/ethereum/go-ethereum/swarm/chunk"
	"github.com/ethereum/go-ethereum/swarm/storage"
)

const (
	BatchSize = 128
)

// SwarmSyncerServer implements an Server for history syncing on bins
// offered streams:
// * live request delivery with or without checkback
// * (live/non-live historical) chunk syncing per proximity bin
type SwarmSyncerServer struct {
	po       uint8
	netStore *storage.NetStore
	quit     chan struct{}
}

// NewSwarmSyncerServer is constructor for SwarmSyncerServer
func NewSwarmSyncerServer(po uint8, netStore *storage.NetStore) (*SwarmSyncerServer, error) {
	return &SwarmSyncerServer{
		po:       po,
		netStore: netStore,
		quit:     make(chan struct{}),
	}, nil
}

func RegisterSwarmSyncerServer(streamer *Registry, netStore *storage.NetStore) {
	streamer.RegisterServerFunc("SYNC", func(_ *Peer, t string, _ bool) (Server, error) {
		po, err := ParseSyncBinKey(t)
		if err != nil {
			return nil, err
		}
		return NewSwarmSyncerServer(po, netStore)
	})
	// streamer.RegisterServerFunc(stream, func(p *Peer) (Server, error) {
	// 	return NewOutgoingProvableSwarmSyncer(po, db)
	// })
}

// Close needs to be called on a stream server
func (s *SwarmSyncerServer) Close() {
	close(s.quit)
}

// GetData retrieves the actual chunk from netstore
func (s *SwarmSyncerServer) GetData(ctx context.Context, key []byte) ([]byte, error) {
	ch, err := s.netStore.Get(ctx, chunk.ModeGetSync, storage.Address(key))
	if err != nil {
		return nil, err
	}
	return ch.Data(), nil
}

// SessionIndex returns current storage bin (po) index.
func (s *SwarmSyncerServer) SessionIndex() (uint64, error) {
	return s.netStore.LastPullSubscriptionBinID(s.po)
}

// SetNextBatch retrieves the next batch of hashes from the localstore.
// It expects a range of bin IDs, both ends inclusive in syncing, and returns
// concatenated byte slice of chunk addresses and bin IDs of the first and
// the last one in that slice. The batch may have up to BatchSize number of
// chunk addresses. If at least one chunk is added to the batch and no new chunks
// are added in batchTimeout period, the batch will be returned. This function
// will block until new chunks are received from localstore pull subscription.
func (s *SwarmSyncerServer) SetNextBatch(from, to uint64) ([]byte, uint64, uint64, *HandoverProof, error) {
	descriptors, stop := s.netStore.SubscribePull(context.Background(), s.po, from, to)
	defer stop()

	const batchTimeout = 2 * time.Second

	var (
		batch        []byte
		batchSize    int
		batchStartID *uint64
		batchEndID   uint64
		timer        *time.Timer
		timerC       <-chan time.Time
	)
	defer func() {
		if timer != nil {
			timer.Stop()
		}
	}()

	for iterate := true; iterate; {
		select {
		case d, ok := <-descriptors:
			if !ok {
				iterate = false
				break
			}
			batch = append(batch, d.Address[:]...)
			// This is the most naive approach to label the chunk as synced
			// allowing it to be garbage collected. A proper way requires
			// validating that the chunk is successfully stored by the peer.
			err := s.netStore.Set(context.Background(), chunk.ModeSetSync, d.Address)
			if err != nil {
				return nil, 0, 0, nil, err
			}
			batchSize++
			if batchStartID == nil {
				// set batch start id only if
				// this is the first iteration
				batchStartID = &d.BinID
			}
			batchEndID = d.BinID
			if batchSize >= BatchSize {
				iterate = false
			}
			if timer == nil {
				timer = time.NewTimer(batchTimeout)
			} else {
				if !timer.Stop() {
					<-timer.C
				}
				timer.Reset(batchTimeout)
			}
			timerC = timer.C
		case <-timerC:
			// return batch if new chunks are not
			// received after some time
			iterate = false
		case <-s.quit:
			iterate = false
		}
	}
	if batchStartID == nil {
		// if batch start id is not set, return 0
		batchStartID = new(uint64)
	}
	return batch, *batchStartID, batchEndID, nil, nil
}

// SwarmSyncerClient
type SwarmSyncerClient struct {
	netStore *storage.NetStore
	peer     *Peer
	stream   Stream
}

// NewSwarmSyncerClient is a contructor for provable data exchange syncer
func NewSwarmSyncerClient(p *Peer, netStore *storage.NetStore, stream Stream) (*SwarmSyncerClient, error) {
	return &SwarmSyncerClient{
		netStore: netStore,
		peer:     p,
		stream:   stream,
	}, nil
}

// RegisterSwarmSyncerClient registers the client constructor function for
// to handle incoming sync streams
func RegisterSwarmSyncerClient(streamer *Registry, netStore *storage.NetStore) {
	streamer.RegisterClientFunc("SYNC", func(p *Peer, t string, live bool) (Client, error) {
		return NewSwarmSyncerClient(p, netStore, NewStream("SYNC", t, live))
	})
}

// NeedData
func (s *SwarmSyncerClient) NeedData(ctx context.Context, key []byte) (wait func(context.Context) error) {
	return s.netStore.FetchFunc(ctx, key)
}

// BatchDone
func (s *SwarmSyncerClient) BatchDone(stream Stream, from uint64, hashes []byte, root []byte) func() (*TakeoverProof, error) {
	// TODO: reenable this with putter/getter refactored code
	// if s.chunker != nil {
	// 	return func() (*TakeoverProof, error) { return s.TakeoverProof(stream, from, hashes, root) }
	// }
	return nil
}

func (s *SwarmSyncerClient) Close() {}

// base for parsing and formating sync bin key
// it must be 2 <= base <= 36
const syncBinKeyBase = 36

// FormatSyncBinKey returns a string representation of
// Kademlia bin number to be used as key for SYNC stream.
func FormatSyncBinKey(bin uint8) string {
	return strconv.FormatUint(uint64(bin), syncBinKeyBase)
}

// ParseSyncBinKey parses the string representation
// and returns the Kademlia bin number.
func ParseSyncBinKey(s string) (uint8, error) {
	bin, err := strconv.ParseUint(s, syncBinKeyBase, 8)
	if err != nil {
		return 0, err
	}
	return uint8(bin), nil
}
