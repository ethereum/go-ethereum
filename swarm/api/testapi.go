// Copyright 2016 The go-ethereum Authors
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

	"github.com/ethereum/go-ethereum/rpc"
	"github.com/ethereum/go-ethereum/swarm/network"
	"github.com/ethereum/go-ethereum/swarm/storage"
)

type Control struct {
	api  *API
	hive *network.Hive
}

func NewControl(api *API, hive *network.Hive) *Control {
	return &Control{api, hive}
}

func (c *Control) Hive() string {
	return c.hive.String()
}

// DebugAPI is a umbrella structure to provide additional debug API endpoints
type DebugAPI struct {
	netStore *storage.NetStore
}

func NewDebugAPI(nstore *storage.NetStore) *DebugAPI {
	return &DebugAPI{
		netStore: nstore,
	}
}

// HasChunk returns true if the underlying datastore has
// the chunk stored with the given address, false if it does not store it
func (dapi *DebugAPI) HasChunk(chunkAddress storage.Address) bool {
	return dapi.netStore.HasChunk(context.Background(), chunkAddress)
}

// The description for the DebugAPI to add to the APIs if the flag is set
func GetDebugAPIDesc(nstore *storage.NetStore) rpc.API {
	return rpc.API{
		Namespace: "debugapi",
		Version:   "1.0",
		Service:   NewDebugAPI(nstore),
		Public:    false,
	}
}
