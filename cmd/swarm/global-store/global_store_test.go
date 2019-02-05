// Copyright 2019 The go-ethereum Authors
// This file is part of go-ethereum.
//
// go-ethereum is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// go-ethereum is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU General Public License for more details.
//
// You should have received a copy of the GNU General Public License
// along with go-ethereum. If not, see <http://www.gnu.org/licenses/>.

package main

import (
	"context"
	"io/ioutil"
	"net"
	"net/http"
	"os"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/rpc"
	mockRPC "github.com/ethereum/go-ethereum/swarm/storage/mock/rpc"
)

// TestHTTP_InMemory tests in-memory global store that exposes
// HTTP server.
func TestHTTP_InMemory(t *testing.T) {
	testHTTP(t, true)
}

// TestHTTP_Database tests global store with persisted database
// that exposes HTTP server.
func TestHTTP_Database(t *testing.T) {
	dir, err := ioutil.TempDir("", "swarm-global-store-")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(dir)

	// create a fresh global store
	testHTTP(t, true, "--dir", dir)

	// check if data saved by the previous global store instance
	testHTTP(t, false, "--dir", dir)
}

// testWebsocket starts global store binary with HTTP server
// and validates that it can store and retrieve data.
// If put is false, no data will be stored, only retrieved,
// giving the possibility to check if data is present in the
// storage directory.
func testHTTP(t *testing.T, put bool, args ...string) {
	addr := findFreeTCPAddress(t)
	testCmd := runGlobalStore(t, append([]string{"http", "--addr", addr}, args...)...)
	defer testCmd.Interrupt()

	client, err := rpc.DialHTTP("http://" + addr)
	if err != nil {
		t.Fatal(err)
	}

	// wait until global store process is started as
	// rpc.DialHTTP is actually not connecting
	for i := 0; i < 1000; i++ {
		_, err = http.DefaultClient.Get("http://" + addr)
		if err == nil {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	if err != nil {
		t.Fatal(err)
	}

	store := mockRPC.NewGlobalStore(client)
	defer store.Close()

	node := store.NewNodeStore(common.HexToAddress("123abc"))

	wantKey := "key"
	wantValue := "value"

	if put {
		err = node.Put([]byte(wantKey), []byte(wantValue))
		if err != nil {
			t.Fatal(err)
		}
	}

	gotValue, err := node.Get([]byte(wantKey))
	if err != nil {
		t.Fatal(err)
	}

	if string(gotValue) != wantValue {
		t.Errorf("got value %s for key %s, want %s", string(gotValue), wantKey, wantValue)
	}
}

// TestWebsocket_InMemory tests in-memory global store that exposes
// WebSocket server.
func TestWebsocket_InMemory(t *testing.T) {
	testWebsocket(t, true)
}

// TestWebsocket_Database tests global store with persisted database
// that exposes HTTP server.
func TestWebsocket_Database(t *testing.T) {
	dir, err := ioutil.TempDir("", "swarm-global-store-")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(dir)

	// create a fresh global store
	testWebsocket(t, true, "--dir", dir)

	// check if data saved by the previous global store instance
	testWebsocket(t, false, "--dir", dir)
}

// testWebsocket starts global store binary with WebSocket server
// and validates that it can store and retrieve data.
// If put is false, no data will be stored, only retrieved,
// giving the possibility to check if data is present in the
// storage directory.
func testWebsocket(t *testing.T, put bool, args ...string) {
	addr := findFreeTCPAddress(t)
	testCmd := runGlobalStore(t, append([]string{"ws", "--addr", addr}, args...)...)
	defer testCmd.Interrupt()

	var client *rpc.Client
	var err error
	// wait until global store process is started
	for i := 0; i < 1000; i++ {
		client, err = rpc.DialWebsocket(context.Background(), "ws://"+addr, "")
		if err == nil {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	if err != nil {
		t.Fatal(err)
	}

	store := mockRPC.NewGlobalStore(client)
	defer store.Close()

	node := store.NewNodeStore(common.HexToAddress("123abc"))

	wantKey := "key"
	wantValue := "value"

	if put {
		err = node.Put([]byte(wantKey), []byte(wantValue))
		if err != nil {
			t.Fatal(err)
		}
	}

	gotValue, err := node.Get([]byte(wantKey))
	if err != nil {
		t.Fatal(err)
	}

	if string(gotValue) != wantValue {
		t.Errorf("got value %s for key %s, want %s", string(gotValue), wantKey, wantValue)
	}
}

// findFreeTCPAddress returns a local address (IP:Port) to which
// global store can listen on.
func findFreeTCPAddress(t *testing.T) (addr string) {
	t.Helper()

	listener, err := net.Listen("tcp", "")
	if err != nil {
		t.Fatal(err)
	}
	defer listener.Close()

	return listener.Addr().String()
}
