// Copyright 2019 The go-ethereum Authors
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

package api

import (
	"context"

	"github.com/ethereum/go-ethereum/swarm/network"
	"github.com/ethereum/go-ethereum/swarm/storage"
)

type Inspector struct {
	api      *API
	hive     *network.Hive
	netStore *storage.NetStore
}

func NewInspector(api *API, hive *network.Hive, netStore *storage.NetStore) *Inspector {
	return &Inspector{api, hive, netStore}
}

// Hive prints the kademlia table
func (inspector *Inspector) Hive() string {
	return inspector.hive.String()
}

type HasInfo struct {
	Addr string `json:"address"`
	Has  bool   `json:"has"`
}

// Has checks whether each chunk address is present in the underlying datastore,
// the bool in the returned structs indicates if the underlying datastore has
// the chunk stored with the given address (true), or not (false)
func (inspector *Inspector) Has(chunkAddresses []storage.Address) []HasInfo {
	results := make([]HasInfo, 0)
	for _, addr := range chunkAddresses {
		res := HasInfo{}
		res.Addr = addr.String()
		res.Has = inspector.netStore.Has(context.Background(), addr)
		results = append(results, res)
	}
	return results
}
