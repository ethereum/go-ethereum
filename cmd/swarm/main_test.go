// Copyright 2017 The go-ethereum Authors
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
	"testing"

	"github.com/ethereum/go-ethereum/swarm"
)

func TestParseFlagEnsEndpoint(t *testing.T) {
	for _, x := range []struct {
		description string
		value       string
		config      swarm.ENSClientConfig
	}{
		{
			description: "IPC endpoint",
			value:       "/data/testnet/geth.ipc",
			config: swarm.ENSClientConfig{
				Endpoint: "/data/testnet/geth.ipc",
			},
		},
		{
			description: "HTTP endpoint",
			value:       "http://127.0.0.1:1234",
			config: swarm.ENSClientConfig{
				Endpoint: "http://127.0.0.1:1234",
			},
		},
		{
			description: "WS endpoint",
			value:       "ws://127.0.0.1:1234",
			config: swarm.ENSClientConfig{
				Endpoint: "ws://127.0.0.1:1234",
			},
		},
		{
			description: "IPC Endpoint and TLD",
			value:       "test:/data/testnet/geth.ipc",
			config: swarm.ENSClientConfig{
				Endpoint: "/data/testnet/geth.ipc",
				TLD:      "test",
			},
		},
		{
			description: "HTTP endpoint and TLD",
			value:       "test:http://127.0.0.1:1234",
			config: swarm.ENSClientConfig{
				Endpoint: "http://127.0.0.1:1234",
				TLD:      "test",
			},
		},
		{
			description: "WS endpoint and TLD",
			value:       "test:ws://127.0.0.1:1234",
			config: swarm.ENSClientConfig{
				Endpoint: "ws://127.0.0.1:1234",
				TLD:      "test",
			},
		},
		{
			description: "IPC Endpoint and contract address",
			value:       "314159265dD8dbb310642f98f50C066173C1259b@/data/testnet/geth.ipc",
			config: swarm.ENSClientConfig{
				Endpoint:        "/data/testnet/geth.ipc",
				ContractAddress: "314159265dD8dbb310642f98f50C066173C1259b",
			},
		},
		{
			description: "HTTP endpoint and contract address",
			value:       "314159265dD8dbb310642f98f50C066173C1259b@http://127.0.0.1:1234",
			config: swarm.ENSClientConfig{
				Endpoint:        "http://127.0.0.1:1234",
				ContractAddress: "314159265dD8dbb310642f98f50C066173C1259b",
			},
		},
		{
			description: "WS endpoint and contract address",
			value:       "314159265dD8dbb310642f98f50C066173C1259b@ws://127.0.0.1:1234",
			config: swarm.ENSClientConfig{
				Endpoint:        "ws://127.0.0.1:1234",
				ContractAddress: "314159265dD8dbb310642f98f50C066173C1259b",
			},
		},
		{
			description: "IPC Endpoint, TLD and contract address",
			value:       "test:314159265dD8dbb310642f98f50C066173C1259b@/data/testnet/geth.ipc",
			config: swarm.ENSClientConfig{
				Endpoint:        "/data/testnet/geth.ipc",
				ContractAddress: "314159265dD8dbb310642f98f50C066173C1259b",
				TLD:             "test",
			},
		},
		{
			description: "HTTP endpoint, TLD and contract address",
			value:       "eth:314159265dD8dbb310642f98f50C066173C1259b@http://127.0.0.1:1234",
			config: swarm.ENSClientConfig{
				Endpoint:        "http://127.0.0.1:1234",
				ContractAddress: "314159265dD8dbb310642f98f50C066173C1259b",
				TLD:             "eth",
			},
		},
		{
			description: "WS endpoint, TLD and contract address",
			value:       "eth:314159265dD8dbb310642f98f50C066173C1259b@ws://127.0.0.1:1234",
			config: swarm.ENSClientConfig{
				Endpoint:        "ws://127.0.0.1:1234",
				ContractAddress: "314159265dD8dbb310642f98f50C066173C1259b",
				TLD:             "eth",
			},
		},
	} {
		t.Run(x.description, func(t *testing.T) {
			config := parseFlagEnsEndpoint(x.value)
			if config.Endpoint != x.config.Endpoint {
				t.Errorf("expected Endpoint %q, got %q", x.config.Endpoint, config.Endpoint)
			}
			if config.ContractAddress != x.config.ContractAddress {
				t.Errorf("expected ContractAddress %q, got %q", x.config.ContractAddress, config.ContractAddress)
			}
			if config.TLD != x.config.TLD {
				t.Errorf("expected TLD %q, got %q", x.config.TLD, config.TLD)
			}
		})
	}
}
