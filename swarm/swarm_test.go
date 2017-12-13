// Copyright 2017 The go-ethereum Authors
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

package swarm

import (
	"testing"

	"github.com/ethereum/go-ethereum/common"
)

func TestParseEnsAPIAddress(t *testing.T) {
	for _, x := range []struct {
		description string
		value       string
		tld         string
		endpoint    string
		addr        common.Address
	}{
		{
			description: "IPC endpoint",
			value:       "/data/testnet/geth.ipc",
			endpoint:    "/data/testnet/geth.ipc",
		},
		{
			description: "HTTP endpoint",
			value:       "http://127.0.0.1:1234",
			endpoint:    "http://127.0.0.1:1234",
		},
		{
			description: "WS endpoint",
			value:       "ws://127.0.0.1:1234",
			endpoint:    "ws://127.0.0.1:1234",
		},
		{
			description: "IPC Endpoint and TLD",
			value:       "test:/data/testnet/geth.ipc",
			endpoint:    "/data/testnet/geth.ipc",
			tld:         "test",
		},
		{
			description: "HTTP endpoint and TLD",
			value:       "test:http://127.0.0.1:1234",
			endpoint:    "http://127.0.0.1:1234",
			tld:         "test",
		},
		{
			description: "WS endpoint and TLD",
			value:       "test:ws://127.0.0.1:1234",
			endpoint:    "ws://127.0.0.1:1234",
			tld:         "test",
		},
		{
			description: "IPC Endpoint and contract address",
			value:       "314159265dD8dbb310642f98f50C066173C1259b@/data/testnet/geth.ipc",
			endpoint:    "/data/testnet/geth.ipc",
			addr:        common.HexToAddress("314159265dD8dbb310642f98f50C066173C1259b"),
		},
		{
			description: "HTTP endpoint and contract address",
			value:       "314159265dD8dbb310642f98f50C066173C1259b@http://127.0.0.1:1234",
			endpoint:    "http://127.0.0.1:1234",
			addr:        common.HexToAddress("314159265dD8dbb310642f98f50C066173C1259b"),
		},
		{
			description: "WS endpoint and contract address",
			value:       "314159265dD8dbb310642f98f50C066173C1259b@ws://127.0.0.1:1234",
			endpoint:    "ws://127.0.0.1:1234",
			addr:        common.HexToAddress("314159265dD8dbb310642f98f50C066173C1259b"),
		},
		{
			description: "IPC Endpoint, TLD and contract address",
			value:       "test:314159265dD8dbb310642f98f50C066173C1259b@/data/testnet/geth.ipc",
			endpoint:    "/data/testnet/geth.ipc",
			addr:        common.HexToAddress("314159265dD8dbb310642f98f50C066173C1259b"),
			tld:         "test",
		},
		{
			description: "HTTP endpoint, TLD and contract address",
			value:       "eth:314159265dD8dbb310642f98f50C066173C1259b@http://127.0.0.1:1234",
			endpoint:    "http://127.0.0.1:1234",
			addr:        common.HexToAddress("314159265dD8dbb310642f98f50C066173C1259b"),
			tld:         "eth",
		},
		{
			description: "WS endpoint, TLD and contract address",
			value:       "eth:314159265dD8dbb310642f98f50C066173C1259b@ws://127.0.0.1:1234",
			endpoint:    "ws://127.0.0.1:1234",
			addr:        common.HexToAddress("314159265dD8dbb310642f98f50C066173C1259b"),
			tld:         "eth",
		},
	} {
		t.Run(x.description, func(t *testing.T) {
			tld, endpoint, addr := parseEnsAPIAddress(x.value)
			if endpoint != x.endpoint {
				t.Errorf("expected Endpoint %q, got %q", x.endpoint, endpoint)
			}
			if addr != x.addr {
				t.Errorf("expected ContractAddress %q, got %q", x.addr.String(), addr.String())
			}
			if tld != x.tld {
				t.Errorf("expected TLD %q, got %q", x.tld, tld)
			}
		})
	}
}
