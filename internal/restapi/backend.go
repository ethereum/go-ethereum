// Copyright 2025 The go-ethereum Authors
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

package restapi

import (
	"context"

	"github.com/ethereum/go-ethereum/beacon/blsync"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/params"
	"github.com/ethereum/go-ethereum/rest"
	"github.com/ethereum/go-ethereum/rpc"
)

type Backend interface {
	HeaderByNumber(ctx context.Context, number rpc.BlockNumber) (*types.Header, error)
	HeaderByHash(ctx context.Context, hash common.Hash) (*types.Header, error)
	ChainConfig() *params.ChainConfig
	Blsync() *blsync.Client
}

func GetAPIs(apiBackend Backend) []rest.API {
	apis := []rest.API{
		{
			Namespace: "exec",
			Register:  NewExecutionRestAPI(apiBackend),
		},
	}
	if apiBackend.Blsync() != nil {
		apis = append(apis, rest.API{
			Namespace: "beacon",
			Register:  apiBackend.Blsync().NewAPIServer(),
		})
	}
	return apis
}
