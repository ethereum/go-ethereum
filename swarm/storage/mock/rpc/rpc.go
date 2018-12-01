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

// Package rpc implements an RPC client that connect to a centralized mock store.
// Centralazied mock store can be any other mock store implementation that is
// registered to Ethereum RPC server under mockStore name. Methods that defines
// mock.GlobalStore are the same that are used by RPC. Example:
//
//   server := rpc.NewServer()
//   server.RegisterName("mockStore", mem.NewGlobalStore())
package rpc

import (
	"fmt"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/ethereum/go-ethereum/swarm/log"
	"github.com/ethereum/go-ethereum/swarm/storage/mock"
)

// GlobalStore is rpc.Client that connects to a centralized mock store.
// Closing GlobalStore instance is required to release RPC client resources.
type GlobalStore struct {
	client *rpc.Client
}

// NewGlobalStore creates a new instance of GlobalStore.
func NewGlobalStore(client *rpc.Client) *GlobalStore {
	return &GlobalStore{
		client: client,
	}
}

// Close closes RPC client.
func (s *GlobalStore) Close() error {
	s.client.Close()
	return nil
}

// NewNodeStore returns a new instance of NodeStore that retrieves and stores
// chunk data only for a node with address addr.
func (s *GlobalStore) NewNodeStore(addr common.Address) *mock.NodeStore {
	return mock.NewNodeStore(addr, s)
}

// Get calls a Get method to RPC server.
func (s *GlobalStore) Get(addr common.Address, key []byte) (data []byte, err error) {
	err = s.client.Call(&data, "mockStore_get", addr, key)
	if err != nil && err.Error() == "not found" {
		// pass the mock package value of error instead an rpc error
		return data, mock.ErrNotFound
	}
	return data, err
}

// Put calls a Put method to RPC server.
func (s *GlobalStore) Put(addr common.Address, key []byte, data []byte) error {
	err := s.client.Call(nil, "mockStore_put", addr, key, data)
	return err
}

// Delete calls a Delete method to RPC server.
func (s *GlobalStore) Delete(addr common.Address, key []byte) error {
	err := s.client.Call(nil, "mockStore_delete", addr, key)
	return err
}

// HasKey calls a HasKey method to RPC server.
func (s *GlobalStore) HasKey(addr common.Address, key []byte) bool {
	var has bool
	if err := s.client.Call(&has, "mockStore_hasKey", addr, key); err != nil {
		log.Error(fmt.Sprintf("mock store HasKey: addr %s, key %064x: %v", addr, key, err))
		return false
	}
	return has
}
