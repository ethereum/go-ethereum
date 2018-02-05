// Copyright 2018 The go-ethereum Authors
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

package light

import (
	"errors"

	"github.com/ethereum/go-ethereum/swarm/network/stream"
	"github.com/ethereum/go-ethereum/swarm/storage"
)

// RemoteReader implements IncomingStreamer
type RemoteSectionReader struct {
	db            *storage.DBAPI
	start         uint64
	end           uint64
	hashes        chan []byte
	currentHashes []byte
	currentData   []byte
	quit          chan struct{}
	root          []byte
}

// NewRemoteReader is the constructor for RemoteReader
func NewRemoteSectionReader(root []byte, db *storage.DBAPI) *RemoteSectionReader {
	return &RemoteSectionReader{
		db:     db,
		root:   root,
		hashes: make(chan []byte),
		quit:   make(chan struct{}),
	}
}

func (r *RemoteSectionReader) NeedData(key []byte) func() {
	chunk, created := r.db.GetOrCreateRequest(storage.Key(key))
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

func (r *RemoteSectionReader) BatchDone(s string, from uint64, hashes []byte, root []byte) func() (*stream.TakeoverProof, error) {
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
	for i := 0; !end && i < len(r.currentHashes); i += stream.HashSize {
		hash := r.currentHashes[i : i+stream.HashSize]
		chunk, err := r.db.Get(hash)
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
			for ; !end && i < len(hashes); i += stream.HashSize {
				hash := hashes[i : i+stream.HashSize]
				chunk, err := r.db.Get(hash)
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

func (r *RemoteSectionReader) Close() {}

// RemoteSectionServer implements OutgoingStreamer
type RemoteSectionServer struct {
	// quit chan struct{}
	root []byte
	db   *storage.DBAPI
	r    *storage.LazyChunkReader
}

// NewRemoteReader is the constructor for RemoteReader
func NewRemoteSectionServer(db *storage.DBAPI, r *storage.LazyChunkReader) *RemoteSectionServer {
	return &RemoteSectionServer{
		db: db,
		r:  r,
	}
}

// GetData retrieves the actual chunk from localstore
func (s *RemoteSectionServer) GetData(key []byte) ([]byte, error) {
	chunk, err := s.db.Get(storage.Key(key))
	if err != nil {
		return nil, err
	}
	return chunk.SData, nil
}

// GetBatch retrieves the next batch of hashes from the dbstore
func (s *RemoteSectionServer) SetNextBatch(from, to uint64) ([]byte, uint64, uint64, *stream.HandoverProof, error) {
	if to > from+stream.BatchSize {
		to = from + stream.BatchSize
	}
	batch := make([]byte, (to-from)*stream.HashSize)
	s.r.ReadAt(batch, int64(from))
	return batch, from, to, nil, nil
}

func (s *RemoteSectionServer) Close() {}

// RegisterRemoteSectionReader registers RemoteSectionReader on light downstream node
func RegisterRemoteSectionReader(s *stream.Registry, db *storage.DBAPI) {
	s.RegisterClientFunc("REMOTE_SECTION", func(p *stream.Peer, t []byte) (stream.Client, error) {
		return NewRemoteSectionReader(t, db), nil
	})
}

// RegisterRemoteSectionServer registers RemoteSectionServer outgoing streamer on
// upstream light server node
func RegisterRemoteSectionServer(s *stream.Registry, db *storage.DBAPI, rf func([]byte) *storage.LazyChunkReader) {
	s.RegisterServerFunc("REMOTE_SECTION", func(p *stream.Peer, t []byte) (stream.Server, error) {
		r := rf(t)
		return NewRemoteSectionServer(db, r), nil
	})
}

// RegisterRemoteDownloader registers RemoteDownloader incoming streamer
// on downstream light  node
// func RegisterRemoteDownloader(s *Streamer, db *storage.DBAPI) {
// 	s.RegisterIncomingStreamer("REMOTE_DOWNLOADER", func(p *stream.Peer, t []byte) (IncomingStreamer, error) {
// 		return NewRemoteDownloader(t, db), nil
// 	})
// }
//
// // RegisterRemoteDownloadServer registers RemoteDownloadServer outgoing streamer on
// // upstream light server node
// func RegisterRemoteDownloadServer(s *Streamer, db *storage.DBAPI, rf func([]byte) *storage.LazyChunkReader) {
// 	s.RegisterOutgoingStreamer("REMOTE_DOWNLOADER", func(p *stream.Peer, t []byte) (OutgoingStreamer, error) {
// 		r := rf(t)
// 		return NewRemoteDownloadServer(db, r), nil
// 	})
// }
