// Copyright 2021 The go-ethereum Authors
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

package ethstats

import (
	"context"
	"math/big"
	"strconv"
	"testing"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/event"
	"github.com/ethereum/go-ethereum/rpc"
)

func TestParseEthstatsURL(t *testing.T) {
	cases := []struct {
		url              string
		node, pass, host string
	}{
		{
			url:  `"debug meowsbits":mypass@ws://mordor.dash.fault.dev:3000`,
			node: "debug meowsbits", pass: "mypass", host: "ws://mordor.dash.fault.dev:3000",
		},
		{
			url:  `"debug @meowsbits":mypass@ws://mordor.dash.fault.dev:3000`,
			node: "debug @meowsbits", pass: "mypass", host: "ws://mordor.dash.fault.dev:3000",
		},
		{
			url:  `"debug: @meowsbits":mypass@ws://mordor.dash.fault.dev:3000`,
			node: "debug: @meowsbits", pass: "mypass", host: "ws://mordor.dash.fault.dev:3000",
		},
		{
			url:  `name:@ws://mordor.dash.fault.dev:3000`,
			node: "name", pass: "", host: "ws://mordor.dash.fault.dev:3000",
		},
		{
			url:  `name@ws://mordor.dash.fault.dev:3000`,
			node: "name", pass: "", host: "ws://mordor.dash.fault.dev:3000",
		},
		{
			url:  `:mypass@ws://mordor.dash.fault.dev:3000`,
			node: "", pass: "mypass", host: "ws://mordor.dash.fault.dev:3000",
		},
		{
			url:  `:@ws://mordor.dash.fault.dev:3000`,
			node: "", pass: "", host: "ws://mordor.dash.fault.dev:3000",
		},
	}

	for i, c := range cases {
		parts, err := parseEthstatsURL(c.url)
		if err != nil {
			t.Fatal(err)
		}

		node, pass, host := parts[0], parts[1], parts[2]

		// unquote because the value provided will be used as a CLI flag value, so unescaped quotes will be removed
		nodeUnquote, err := strconv.Unquote(node)
		if err == nil {
			node = nodeUnquote
		}

		if node != c.node {
			t.Errorf("case=%d mismatch node value, got: %v ,want: %v", i, node, c.node)
		}

		if pass != c.pass {
			t.Errorf("case=%d mismatch pass value, got: %v ,want: %v", i, pass, c.pass)
		}

		if host != c.host {
			t.Errorf("case=%d mismatch host value, got: %v ,want: %v", i, host, c.host)
		}
	}
}

// MockBackend is a mock implementation of the backend interface
type MockFullNodeBackend struct{}

func (m *MockFullNodeBackend) SubscribeChainHeadEvent(ch chan<- core.ChainHeadEvent) event.Subscription {
	return nil
}

func (m *MockFullNodeBackend) SubscribeNewTxsEvent(ch chan<- core.NewTxsEvent) event.Subscription {
	return nil
}

func (m *MockFullNodeBackend) CurrentHeader() *types.Header {
	return &types.Header{}
}

func (m *MockFullNodeBackend) HeaderByNumber(ctx context.Context, number rpc.BlockNumber) (*types.Header, error) {
	return nil, nil
}

func (m *MockFullNodeBackend) GetTd(ctx context.Context, hash common.Hash) *big.Int {
	return big.NewInt(0)
}

func (m *MockFullNodeBackend) Stats() (pending int, queued int) {
	return 0, 0
}

func (m *MockFullNodeBackend) SyncProgress() ethereum.SyncProgress {
	return ethereum.SyncProgress{}
}

func (m *MockFullNodeBackend) SubscribeChain2HeadEvent(ch chan<- core.Chain2HeadEvent) event.Subscription {
	return nil
}

func (m *MockFullNodeBackend) BlockByNumber(ctx context.Context, number rpc.BlockNumber) (*types.Block, error) {
	return nil, nil
}

func (m *MockFullNodeBackend) CurrentBlock() *types.Header {
	return &types.Header{Number: big.NewInt(1)}
}

func (m *MockFullNodeBackend) SuggestGasTipCap(ctx context.Context) (*big.Int, error) {
	return big.NewInt(0), nil
}

func TestAssembleBlockStats_NilBlock(t *testing.T) {
	mockBackend := &MockFullNodeBackend{}
	service := &Service{
		backend: mockBackend,
	}

	result := service.assembleBlockStats(nil)

	if result != nil {
		t.Errorf("Expected nil, got %v", result)
	}
}
