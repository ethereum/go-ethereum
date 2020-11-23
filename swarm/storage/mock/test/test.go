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
	})
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

	go func() {
		defer w.Close()
		if _, err := exporter.Export(w); err != nil {
			t.Fatalf("export: %v", err)
		}
	}()

	if _, err := importer.Import(r); err != nil {
		t.Fatalf("import: %v", err)
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
