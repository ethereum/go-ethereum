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

package adapters

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/p2p/simulations/pipes"
)

func TestTCPPipe(t *testing.T) {
	c1, c2, err := pipes.TCPPipe()
	if err != nil {
		t.Fatal(err)
	}

	done := make(chan struct{})

	go func() {
		msgs := 50
		size := 1024
		for i := 0; i < msgs; i++ {
			msg := make([]byte, size)
			_ = binary.PutUvarint(msg, uint64(i))

			_, err := c1.Write(msg)
			if err != nil {
				t.Fatal(err)
			}
		}

		for i := 0; i < msgs; i++ {
			msg := make([]byte, size)
			_ = binary.PutUvarint(msg, uint64(i))

			out := make([]byte, size)
			_, err := c2.Read(out)
			if err != nil {
				t.Fatal(err)
			}

			if !bytes.Equal(msg, out) {
				t.Fatalf("expected %#v, got %#v", msg, out)
			}
		}
		done <- struct{}{}
	}()

	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("test timeout")
	}
}

func TestTCPPipeBidirections(t *testing.T) {
	c1, c2, err := pipes.TCPPipe()
	if err != nil {
		t.Fatal(err)
	}

	done := make(chan struct{})

	go func() {
		msgs := 50
		size := 7
		for i := 0; i < msgs; i++ {
			msg := []byte(fmt.Sprintf("ping %02d", i))

			_, err := c1.Write(msg)
			if err != nil {
				t.Fatal(err)
			}
		}

		for i := 0; i < msgs; i++ {
			expected := []byte(fmt.Sprintf("ping %02d", i))

			out := make([]byte, size)
			_, err := c2.Read(out)
			if err != nil {
				t.Fatal(err)
			}

			if !bytes.Equal(expected, out) {
				t.Fatalf("expected %#v, got %#v", out, expected)
			} else {
				msg := []byte(fmt.Sprintf("pong %02d", i))
				_, err := c2.Write(msg)
				if err != nil {
					t.Fatal(err)
				}
			}
		}

		for i := 0; i < msgs; i++ {
			expected := []byte(fmt.Sprintf("pong %02d", i))

			out := make([]byte, size)
			_, err := c1.Read(out)
			if err != nil {
				t.Fatal(err)
			}

			if !bytes.Equal(expected, out) {
				t.Fatalf("expected %#v, got %#v", out, expected)
			}
		}
		done <- struct{}{}
	}()

	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("test timeout")
	}
}

func TestNetPipe(t *testing.T) {
	c1, c2, err := pipes.NetPipe()
	if err != nil {
		t.Fatal(err)
	}

	done := make(chan struct{})

	go func() {
		msgs := 50
		size := 1024
		// netPipe is blocking, so writes are emitted asynchronously
		go func() {
			for i := 0; i < msgs; i++ {
				msg := make([]byte, size)
				_ = binary.PutUvarint(msg, uint64(i))

				_, err := c1.Write(msg)
				if err != nil {
					t.Fatal(err)
				}
			}
		}()

		for i := 0; i < msgs; i++ {
			msg := make([]byte, size)
			_ = binary.PutUvarint(msg, uint64(i))

			out := make([]byte, size)
			_, err := c2.Read(out)
			if err != nil {
				t.Fatal(err)
			}

			if !bytes.Equal(msg, out) {
				t.Fatalf("expected %#v, got %#v", msg, out)
			}
		}

		done <- struct{}{}
	}()

	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("test timeout")
	}
}

func TestNetPipeBidirections(t *testing.T) {
	c1, c2, err := pipes.NetPipe()
	if err != nil {
		t.Fatal(err)
	}

	done := make(chan struct{})

	go func() {
		msgs := 1000
		size := 8
		pingTemplate := "ping %03d"
		pongTemplate := "pong %03d"

		// netPipe is blocking, so writes are emitted asynchronously
		go func() {
			for i := 0; i < msgs; i++ {
				msg := []byte(fmt.Sprintf(pingTemplate, i))

				_, err := c1.Write(msg)
				if err != nil {
					t.Fatal(err)
				}
			}
		}()

		// netPipe is blocking, so reads for pong are emitted asynchronously
		go func() {
			for i := 0; i < msgs; i++ {
				expected := []byte(fmt.Sprintf(pongTemplate, i))

				out := make([]byte, size)
				_, err := c1.Read(out)
				if err != nil {
					t.Fatal(err)
				}

				if !bytes.Equal(expected, out) {
					t.Fatalf("expected %#v, got %#v", expected, out)
				}
			}

			done <- struct{}{}
		}()

		// expect to read pings, and respond with pongs to the alternate connection
		for i := 0; i < msgs; i++ {
			expected := []byte(fmt.Sprintf(pingTemplate, i))

			out := make([]byte, size)
			_, err := c2.Read(out)
			if err != nil {
				t.Fatal(err)
			}

			if !bytes.Equal(expected, out) {
				t.Fatalf("expected %#v, got %#v", expected, out)
			} else {
				msg := []byte(fmt.Sprintf(pongTemplate, i))

				_, err := c2.Write(msg)
				if err != nil {
					t.Fatal(err)
				}
			}
		}
	}()

	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("test timeout")
	}
}
