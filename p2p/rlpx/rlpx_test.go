// Copyright 2015 The go-ethereum Authors
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

package rlpx

import (
	"bytes"
	"crypto/ecdsa"
	"fmt"
	"io"
	"io/ioutil"
	"net"
	"sync"
	"testing"

	"github.com/ethereum/go-ethereum/crypto"
)

func TestSequentialTransfer(t *testing.T) {
	var (
		p1, p2 = net.Pipe()
		k1, k2 = newkey(), newkey()
		sc     = Server(p1, &Config{Key: k1})
		cc     = Client(p2, &k1.PublicKey, &Config{Key: k2})
	)
	run(t, rig{
		"server": func() error { return testProtoReaders(t, sc, 1) },
		"client": func() error { return testProtoWriters(t, cc, 1) },
	})
}

func TestConcurrentTransfer(t *testing.T) {
	var (
		p1, p2 = net.Pipe()
		k1, k2 = newkey(), newkey()
		sc     = Server(p1, &Config{Key: k1})
		cc     = Client(p2, &k1.PublicKey, &Config{Key: k2})
	)
	run(t, rig{
		"server": func() error { return testProtoReaders(t, cc, 10) },
		"client": func() error { return testProtoWriters(t, sc, 10) },
	})
}

func TestConcurrentTransferReadError(t *testing.T) {
	var (
		p1, p2        = net.Pipe()
		k1, k2        = newkey(), newkey()
		sc            = Server(p1, &Config{Key: k1})
		cc            = Client(p2, &k1.PublicKey, &Config{Key: k2})
		badPacketSize = uint32(16 * 8 * 1024)
	)
	run(t, rig{
		"client": func() error { return testProtoWriters(t, sc, 10) },
		"server": func() error { return testProtoReaders(t, cc, 10) },

		// This sends a bad frame after a two sane ones.
		"badFrameWrite": func() error {
			if err := cc.Handshake(); err != nil {
				return fmt.Errorf("handshake error: %v", err)
			}
			heads := []interface{}{
				chunkStartHeader{Protocol: 11, ContextID: 1, TotalSize: badPacketSize},
				regularHeader{Protocol: 11, ContextID: 1},
				// The bad frame is a chunk start header with a context id
				// that is already in use.
				chunkStartHeader{Protocol: 11, ContextID: 1, TotalSize: 22},
			}
			for i, h := range heads {
				buf := makeFrameWriteBuffer()
				buf.Write(make([]byte, 1024))
				if err := cc.sendFrame(h, buf); err != nil {
					return fmt.Errorf("error sending frame %d: %v", i, err)
				}
			}
			return nil
		},

		// The other end should receive an error from Read.
		"badFrameRead": func() error {
			proto := sc.Protocol(11)
			_, r, err := proto.ReadPacket()
			if err != nil {
				return fmt.Errorf("unexpected ReadPacket error: %v", err)
			}
			_, err = io.CopyN(ioutil.Discard, r, int64(badPacketSize))
			if err == nil {
				return fmt.Errorf("no error received")
			}
			if err != errUnexpectedChunkStart {
				return fmt.Errorf("wrong error: got %q want %q", err, errUnexpectedChunkStart)
			}
			// TODO: shouldn't all transfers fail?
			return nil
		},
	})
}

func testProtoWriters(t *testing.T, conn *Conn, nprotos uint16) error {
	defer conn.Close()
	writers := rig{}
	for i := uint16(0); i < nprotos; i++ {
		i := i
		writers[fmt.Sprint("protocol ", i)] = func() error { return testWriter(t, conn.Protocol(i)) }
	}
	run(t, writers)
	return nil
}

func testProtoReaders(t *testing.T, conn *Conn, nprotos uint16) error {
	defer conn.Close()
	readers := rig{}
	for i := uint16(0); i < nprotos; i++ {
		i := i
		readers[fmt.Sprint("protocol ", i)] = func() error { return testReader(t, conn.Protocol(i)) }
	}
	run(t, readers)
	return nil
}

func testWriter(t *testing.T, p *Protocol) error {
	for size := 1; size < 8*1024*1024; size *= 2 {
		if err := sendBytes(p, make([]byte, size)); err != nil {
			return fmt.Errorf("error sending %d bytes: %v", size, err)
		}
	}
	return nil
}

func testReader(t *testing.T, p *Protocol) error {
	for size := 1; size < 8*1024*1024; size *= 2 {
		len, r, err := p.ReadPacket()
		if err != nil {
			return fmt.Errorf("ReadPacket error with size %d: %v", size, err)
		}
		if len != uint32(size) {
			return fmt.Errorf("len mismatch, got %d want %d", len, size)
		}
		if n, err := io.CopyN(ioutil.Discard, r, int64(size)); err != nil {
			return fmt.Errorf("body read error at %d of %d bytes: %v", n, size, err)
		}
	}
	return nil
}

type rig map[string]func() error

func run(t *testing.T, rig rig) {
	var wg sync.WaitGroup
	wg.Add(len(rig))
	for name, fn := range rig {
		name, fn := name, fn
		go func() {
			if err := fn(); err != nil {
				t.Error(name, err)
			}
			wg.Done()
		}()
	}
	wg.Wait()
}

func sendBytes(p *Protocol, data []byte) error {
	return p.SendPacket(uint32(len(data)), bytes.NewReader(data))
}

func newkey() *ecdsa.PrivateKey {
	key, err := crypto.GenerateKey()
	if err != nil {
		panic("couldn't generate key: " + err.Error())
	}
	return key
}
