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

// Package test provides functions that are used for testing
// GlobalStorer implementations.
package test

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"strconv"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/swarm/storage"
	"github.com/ethereum/go-ethereum/swarm/storage/mock"
)

// MockStore creates NodeStore instances from provided GlobalStorer,
// each one with a unique address, stores different chunks on them
// and checks if they are retrievable or not on all nodes.
// Attribute n defines the number of NodeStores that will be created.
func MockStore(t *testing.T, globalStore mock.GlobalStorer, n int) {
	t.Run("GlobalStore", func(t *testing.T) {
		addrs := make([]common.Address, n)
		for i := 0; i < n; i++ {
			addrs[i] = common.HexToAddress(strconv.FormatInt(int64(i)+1, 16))
		}

		for i, addr := range addrs {
			chunkAddr := storage.Address(append(addr[:], []byte(strconv.FormatInt(int64(i)+1, 16))...))
			data := []byte(strconv.FormatInt(int64(i)+1, 16))
			data = append(data, make([]byte, 4096-len(data))...)
			globalStore.Put(addr, chunkAddr, data)

			for _, cAddr := range addrs {
				cData, err := globalStore.Get(cAddr, chunkAddr)
				if cAddr == addr {
					if err != nil {
						t.Fatalf("get data from store %s key %s: %v", cAddr.Hex(), chunkAddr.Hex(), err)
					}
					if !bytes.Equal(data, cData) {
						t.Fatalf("data on store %s: expected %x, got %x", cAddr.Hex(), data, cData)
					}
					if !globalStore.HasKey(cAddr, chunkAddr) {
						t.Fatalf("expected key %s on global store for node %s, but it was not found", chunkAddr.Hex(), cAddr.Hex())
					}
				} else {
					if err != mock.ErrNotFound {
						t.Fatalf("expected error from store %s: %v, got %v", cAddr.Hex(), mock.ErrNotFound, err)
					}
					if len(cData) > 0 {
						t.Fatalf("data on store %s: expected nil, got %x", cAddr.Hex(), cData)
					}
					if globalStore.HasKey(cAddr, chunkAddr) {
						t.Fatalf("not expected key %s on global store for node %s, but it was found", chunkAddr.Hex(), cAddr.Hex())
					}
				}
			}
		}
		t.Run("delete", func(t *testing.T) {
			chunkAddr := storage.Address([]byte("1234567890abcd"))
			for _, addr := range addrs {
				err := globalStore.Put(addr, chunkAddr, []byte("data"))
				if err != nil {
					t.Fatalf("put data to store %s key %s: %v", addr.Hex(), chunkAddr.Hex(), err)
				}
			}
			firstNodeAddr := addrs[0]
			if err := globalStore.Delete(firstNodeAddr, chunkAddr); err != nil {
				t.Fatalf("delete from store %s key %s: %v", firstNodeAddr.Hex(), chunkAddr.Hex(), err)
			}
			for i, addr := range addrs {
				_, err := globalStore.Get(addr, chunkAddr)
				if i == 0 {
					if err != mock.ErrNotFound {
						t.Errorf("get data from store %s key %s: expected mock.ErrNotFound error, got %v", addr.Hex(), chunkAddr.Hex(), err)
					}
				} else {
					if err != nil {
						t.Errorf("get data from store %s key %s: %v", addr.Hex(), chunkAddr.Hex(), err)
					}
				}
			}
		})
	})

	t.Run("NodeStore", func(t *testing.T) {
		nodes := make(map[common.Address]*mock.NodeStore)
		for i := 0; i < n; i++ {
			addr := common.HexToAddress(strconv.FormatInt(int64(i)+1, 16))
			nodes[addr] = globalStore.NewNodeStore(addr)
		}

		i := 0
		for addr, store := range nodes {
			i++
			chunkAddr := storage.Address(append(addr[:], []byte(fmt.Sprintf("%x", i))...))
			data := []byte(strconv.FormatInt(int64(i)+1, 16))
			data = append(data, make([]byte, 4096-len(data))...)
			store.Put(chunkAddr, data)

			for cAddr, cStore := range nodes {
				cData, err := cStore.Get(chunkAddr)
				if cAddr == addr {
					if err != nil {
						t.Fatalf("get data from store %s key %s: %v", cAddr.Hex(), chunkAddr.Hex(), err)
					}
					if !bytes.Equal(data, cData) {
						t.Fatalf("data on store %s: expected %x, got %x", cAddr.Hex(), data, cData)
					}
					if !globalStore.HasKey(cAddr, chunkAddr) {
						t.Fatalf("expected key %s on global store for node %s, but it was not found", chunkAddr.Hex(), cAddr.Hex())
					}
				} else {
					if err != mock.ErrNotFound {
						t.Fatalf("expected error from store %s: %v, got %v", cAddr.Hex(), mock.ErrNotFound, err)
					}
					if len(cData) > 0 {
						t.Fatalf("data on store %s: expected nil, got %x", cAddr.Hex(), cData)
					}
					if globalStore.HasKey(cAddr, chunkAddr) {
						t.Fatalf("not expected key %s on global store for node %s, but it was found", chunkAddr.Hex(), cAddr.Hex())
					}
				}
			}
		}
		t.Run("delete", func(t *testing.T) {
			chunkAddr := storage.Address([]byte("1234567890abcd"))
			var chosenStore *mock.NodeStore
			for addr, store := range nodes {
				if chosenStore == nil {
					chosenStore = store
				}
				err := store.Put(chunkAddr, []byte("data"))
				if err != nil {
					t.Fatalf("put data to store %s key %s: %v", addr.Hex(), chunkAddr.Hex(), err)
				}
			}
			if err := chosenStore.Delete(chunkAddr); err != nil {
				t.Fatalf("delete key %s: %v", chunkAddr.Hex(), err)
			}
			for addr, store := range nodes {
				_, err := store.Get(chunkAddr)
				if store == chosenStore {
					if err != mock.ErrNotFound {
						t.Errorf("get data from store %s key %s: expected mock.ErrNotFound error, got %v", addr.Hex(), chunkAddr.Hex(), err)
					}
				} else {
					if err != nil {
						t.Errorf("get data from store %s key %s: %v", addr.Hex(), chunkAddr.Hex(), err)
					}
				}
			}
		})
	})
}

// MockStoreListings tests global store methods Keys, Nodes, NodeKeys and KeyNodes.
// It uses a provided globalstore to put chunks for n number of node addresses
// and to validate that methods are returning the right responses.
func MockStoreListings(t *testing.T, globalStore mock.GlobalStorer, n int) {
	addrs := make([]common.Address, n)
	for i := 0; i < n; i++ {
		addrs[i] = common.HexToAddress(strconv.FormatInt(int64(i)+1, 16))
	}
	type chunk struct {
		key  []byte
		data []byte
	}
	const chunksPerNode = 5
	keys := make([][]byte, n*chunksPerNode)
	for i := 0; i < n*chunksPerNode; i++ {
		b := make([]byte, 8)
		binary.BigEndian.PutUint64(b, uint64(i))
		keys[i] = b
	}

	// keep track of keys on every node
	nodeKeys := make(map[common.Address][][]byte)
	// keep track of nodes that store particular key
	keyNodes := make(map[string][]common.Address)
	for i := 0; i < chunksPerNode; i++ {
		// put chunks for every address
		for j := 0; j < n; j++ {
			addr := addrs[j]
			key := keys[(i*n)+j]
			err := globalStore.Put(addr, key, []byte("data"))
			if err != nil {
				t.Fatal(err)
			}
			nodeKeys[addr] = append(nodeKeys[addr], key)
			keyNodes[string(key)] = append(keyNodes[string(key)], addr)
		}

		// test Keys method
		var startKey []byte
		var gotKeys [][]byte
		for {
			keys, err := globalStore.Keys(startKey, 0)
			if err != nil {
				t.Fatal(err)
			}
			gotKeys = append(gotKeys, keys.Keys...)
			if keys.Next == nil {
				break
			}
			startKey = keys.Next
		}
		wantKeys := keys[:(i+1)*n]
		if fmt.Sprint(gotKeys) != fmt.Sprint(wantKeys) {
			t.Fatalf("got #%v keys %v, want %v", i+1, gotKeys, wantKeys)
		}

		// test Nodes method
		var startNode *common.Address
		var gotNodes []common.Address
		for {
			nodes, err := globalStore.Nodes(startNode, 0)
			if err != nil {
				t.Fatal(err)
			}
			gotNodes = append(gotNodes, nodes.Addrs...)
			if nodes.Next == nil {
				break
			}
			startNode = nodes.Next
		}
		wantNodes := addrs
		if fmt.Sprint(gotNodes) != fmt.Sprint(wantNodes) {
			t.Fatalf("got #%v nodes %v, want %v", i+1, gotNodes, wantNodes)
		}

		// test NodeKeys method
		for addr, wantKeys := range nodeKeys {
			var startKey []byte
			var gotKeys [][]byte
			for {
				keys, err := globalStore.NodeKeys(addr, startKey, 0)
				if err != nil {
					t.Fatal(err)
				}
				gotKeys = append(gotKeys, keys.Keys...)
				if keys.Next == nil {
					break
				}
				startKey = keys.Next
			}
			if fmt.Sprint(gotKeys) != fmt.Sprint(wantKeys) {
				t.Fatalf("got #%v %s node keys %v, want %v", i+1, addr.Hex(), gotKeys, wantKeys)
			}
		}

		// test KeyNodes method
		for key, wantNodes := range keyNodes {
			var startNode *common.Address
			var gotNodes []common.Address
			for {
				nodes, err := globalStore.KeyNodes([]byte(key), startNode, 0)
				if err != nil {
					t.Fatal(err)
				}
				gotNodes = append(gotNodes, nodes.Addrs...)
				if nodes.Next == nil {
					break
				}
				startNode = nodes.Next
			}
			if fmt.Sprint(gotNodes) != fmt.Sprint(wantNodes) {
				t.Fatalf("got #%v %x key nodes %v, want %v", i+1, []byte(key), gotNodes, wantNodes)
			}
		}
	}
}

// ImportExport saves chunks to the outStore, exports them to the tar archive,
// imports tar archive to the inStore and checks if all chunks are imported correctly.
func ImportExport(t *testing.T, outStore, inStore mock.GlobalStorer, n int) {
	exporter, ok := outStore.(mock.Exporter)
	if !ok {
		t.Fatal("outStore does not implement mock.Exporter")
	}
	importer, ok := inStore.(mock.Importer)
	if !ok {
		t.Fatal("inStore does not implement mock.Importer")
	}
	addrs := make([]common.Address, n)
	for i := 0; i < n; i++ {
		addrs[i] = common.HexToAddress(strconv.FormatInt(int64(i)+1, 16))
	}

	for i, addr := range addrs {
		chunkAddr := storage.Address(append(addr[:], []byte(strconv.FormatInt(int64(i)+1, 16))...))
		data := []byte(strconv.FormatInt(int64(i)+1, 16))
		data = append(data, make([]byte, 4096-len(data))...)
		outStore.Put(addr, chunkAddr, data)
	}

	r, w := io.Pipe()
	defer r.Close()

	exportErrChan := make(chan error)
	go func() {
		defer w.Close()

		_, err := exporter.Export(w)
		exportErrChan <- err
	}()

	if _, err := importer.Import(r); err != nil {
		t.Fatalf("import: %v", err)
	}

	if err := <-exportErrChan; err != nil {
		t.Fatalf("export: %v", err)
	}

	for i, addr := range addrs {
		chunkAddr := storage.Address(append(addr[:], []byte(strconv.FormatInt(int64(i)+1, 16))...))
		data := []byte(strconv.FormatInt(int64(i)+1, 16))
		data = append(data, make([]byte, 4096-len(data))...)
		for _, cAddr := range addrs {
			cData, err := inStore.Get(cAddr, chunkAddr)
			if cAddr == addr {
				if err != nil {
					t.Fatalf("get data from store %s key %s: %v", cAddr.Hex(), chunkAddr.Hex(), err)
				}
				if !bytes.Equal(data, cData) {
					t.Fatalf("data on store %s: expected %x, got %x", cAddr.Hex(), data, cData)
				}
				if !inStore.HasKey(cAddr, chunkAddr) {
					t.Fatalf("expected key %s on global store for node %s, but it was not found", chunkAddr.Hex(), cAddr.Hex())
				}
			} else {
				if err != mock.ErrNotFound {
					t.Fatalf("expected error from store %s: %v, got %v", cAddr.Hex(), mock.ErrNotFound, err)
				}
				if len(cData) > 0 {
					t.Fatalf("data on store %s: expected nil, got %x", cAddr.Hex(), cData)
				}
				if inStore.HasKey(cAddr, chunkAddr) {
					t.Fatalf("not expected key %s on global store for node %s, but it was found", chunkAddr.Hex(), cAddr.Hex())
				}
			}
		}
	}
}
