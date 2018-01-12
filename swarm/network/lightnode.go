// Copyright 2017 The go-ethereum Authors
// This file is part of the go-ethereum library.d
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

package network

import (
	"errors"

	"github.com/ethereum/go-ethereum/swarm/storage"
)

// RemoteReader implements IncomingStreamer
type RemoteSectionReader struct {
	db            *DbAccess
	start         uint64
	end           uint64
	hashes        chan []byte
	currentHashes []byte
	currentData   []byte
	quit          chan struct{}
	root          []byte
}

// NewRemoteReader is the constructor for RemoteReader
func NewRemoteSectionReader(root []byte, db *DbAccess) *RemoteSectionReader {
	return &RemoteSectionReader{
		db:     db,
		root:   root,
		hashes: make(chan []byte),
		quit:   make(chan struct{}),
	}
}

func (r *RemoteSectionReader) NeedData(key []byte) func() {
	chunk, created := r.db.getOrCreateRequest(storage.Key(key))
	// TODO: we may want to request from this peer anyway even if the request exists
	if chunk.ReqC == nil || !created {
		return nil
	}
	return func() {
		select {
		case <-chunk.ReqC:
		case <-r.quit:
		}
	}
}

func (r *RemoteSectionReader) BatchDone(s string, from uint64, hashes []byte, root []byte) func() (*TakeoverProof, error) {
	r.hashes <- hashes
	return nil
}

func (r *RemoteSectionReader) Read(b []byte) (n int64, err error) {
	l := int64(len(b))
	m := int64(len(r.currentData))
	if m > l {
		m = l
	}
	copy(b, r.currentData[:m])
	if m == l {
		r.currentData = r.currentData[m:]
		return l, nil
	}
	var end bool
	for i := 0; !end && i < len(r.currentHashes); i += HashSize {
		hash := r.currentHashes[i : i+HashSize]
		chunk, err := r.db.get(hash)
		if err != nil {
			return n, err
		}
		m := chunk.Size
		if n+m > l {
			m = l - n
			end = true
		}
		copy(b[n:], chunk.SData[:m])
		n += int64(m)
	}

	for {
		select {
		case <-r.quit:
			return n, errors.New("aborted")
		case hashes := <-r.hashes:
			var i int
			for ; !end && i < len(hashes); i += HashSize {
				hash := hashes[i : i+HashSize]
				chunk, err := r.db.get(hash)
				if err != nil {
					return n, err
				}
				m := chunk.Size
				if n+m > l {
					m = l - n
					end = true

				}
				copy(b[n:], chunk.SData[:m])
				n += m
			}
			hashes = hashes[i:]
		}
	}
}

// RemoteSectionServer implements OutgoingStreamer
type RemoteSectionServer struct {
	// quit chan struct{}
	root []byte
	db   *DbAccess
	r    *storage.LazyChunkReader
}

// NewRemoteReader is the constructor for RemoteReader
func NewRemoteSectionServer(db *DbAccess, r *storage.LazyChunkReader) *RemoteSectionServer {
	return &RemoteSectionServer{
		db: db,
		r:  r,
	}
}

// GetData retrieves the actual chunk from localstore
func (s *RemoteSectionServer) GetData(key []byte) []byte {
	chunk, err := s.db.get(storage.Key(key))
	if err != nil {
		return nil
	}
	return chunk.SData
}

// GetBatch retrieves the next batch of hashes from the dbstore
func (s *RemoteSectionServer) SetNextBatch(from, to uint64) ([]byte, uint64, uint64, *HandoverProof, error) {
	if to > from+batchSize {
		to = from + batchSize
	}
	batch := make([]byte, (to-from)*HashSize)
	s.r.ReadAt(batch, int64(from))
	return batch, from, to, nil, nil
}

// RegisterRemoteSectionReader registers RemoteSectionReader on light downstream node
func RegisterRemoteSectionReader(s *Streamer, db *DbAccess) {
	s.RegisterIncomingStreamer("REMOTE_SECTION", func(p *StreamerPeer, t []byte) (IncomingStreamer, error) {
		return NewRemoteSectionReader(t, db), nil
	})
}

// RegisterRemoteSectionServer registers RemoteSectionServer outgoing streamer on
// upstream light server node
func RegisterRemoteSectionServer(s *Streamer, db *DbAccess, rf func([]byte) *storage.LazyChunkReader) {
	s.RegisterOutgoingStreamer("REMOTE_SECTION", func(p *StreamerPeer, t []byte) (OutgoingStreamer, error) {
		r := rf(t)
		return NewRemoteSectionServer(db, r), nil
	})
}

// RegisterRemoteDownloader registers RemoteDownloader incoming streamer
// on downstream light  node
// func RegisterRemoteDownloader(s *Streamer, db *DbAccess) {
// 	s.RegisterIncomingStreamer("REMOTE_DOWNLOADER", func(p *StreamerPeer, t []byte) (IncomingStreamer, error) {
// 		return NewRemoteDownloader(t, db), nil
// 	})
// }
//
// // RegisterRemoteDownloadServer registers RemoteDownloadServer outgoing streamer on
// // upstream light server node
// func RegisterRemoteDownloadServer(s *Streamer, db *DbAccess, rf func([]byte) *storage.LazyChunkReader) {
// 	s.RegisterOutgoingStreamer("REMOTE_DOWNLOADER", func(p *StreamerPeer, t []byte) (OutgoingStreamer, error) {
// 		r := rf(t)
// 		return NewRemoteDownloadServer(db, r), nil
// 	})
// }
