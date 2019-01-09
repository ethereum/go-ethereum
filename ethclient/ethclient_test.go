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

package ethclient

import (
	"context"
	"fmt"
	"math/big"
	"reflect"
	"testing"
	"time"

	ethereum "github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/rpc"
)

// Verify that Client implements the ethereum interfaces.
var (
	_ = ethereum.ChainReader(&Client{})
	_ = ethereum.TransactionReader(&Client{})
	_ = ethereum.ChainStateReader(&Client{})
	_ = ethereum.ChainSyncReader(&Client{})
	_ = ethereum.ContractCaller(&Client{})
	_ = ethereum.GasEstimator(&Client{})
	_ = ethereum.GasPricer(&Client{})
	_ = ethereum.LogFilterer(&Client{})
	_ = ethereum.PendingStateReader(&Client{})
	// _ = ethereum.PendingStateEventer(&Client{})
	_ = ethereum.PendingContractCaller(&Client{})
)

func TestToFilterArg(t *testing.T) {
	blockHashErr := fmt.Errorf("cannot specify both BlockHash and FromBlock/ToBlock")
	addresses := []common.Address{
		common.HexToAddress("0xD36722ADeC3EdCB29c8e7b5a47f352D701393462"),
	}
	blockHash := common.HexToHash(
		"0xeb94bb7d78b73657a9d7a99792413f50c0a45c51fc62bdcb08a53f18e9a2b4eb",
	)

	for _, testCase := range []struct {
		name   string
		input  ethereum.FilterQuery
		output interface{}
		err    error
	}{
		{
			"without BlockHash",
			ethereum.FilterQuery{
				Addresses: addresses,
				FromBlock: big.NewInt(1),
				ToBlock:   big.NewInt(2),
				Topics:    [][]common.Hash{},
			},
			map[string]interface{}{
				"address":   addresses,
				"fromBlock": "0x1",
				"toBlock":   "0x2",
				"topics":    [][]common.Hash{},
			},
			nil,
		},
		{
			"with nil fromBlock and nil toBlock",
			ethereum.FilterQuery{
				Addresses: addresses,
				Topics:    [][]common.Hash{},
			},
			map[string]interface{}{
				"address":   addresses,
				"fromBlock": "0x0",
				"toBlock":   "latest",
				"topics":    [][]common.Hash{},
			},
			nil,
		},
		{
			"with blockhash",
			ethereum.FilterQuery{
				Addresses: addresses,
				BlockHash: &blockHash,
				Topics:    [][]common.Hash{},
			},
			map[string]interface{}{
				"address":   addresses,
				"blockHash": blockHash,
				"topics":    [][]common.Hash{},
			},
			nil,
		},
		{
			"with blockhash and from block",
			ethereum.FilterQuery{
				Addresses: addresses,
				BlockHash: &blockHash,
				FromBlock: big.NewInt(1),
				Topics:    [][]common.Hash{},
			},
			nil,
			blockHashErr,
		},
		{
			"with blockhash and to block",
			ethereum.FilterQuery{
				Addresses: addresses,
				BlockHash: &blockHash,
				ToBlock:   big.NewInt(1),
				Topics:    [][]common.Hash{},
			},
			nil,
			blockHashErr,
		},
		{
			"with blockhash and both from / to block",
			ethereum.FilterQuery{
				Addresses: addresses,
				BlockHash: &blockHash,
				FromBlock: big.NewInt(1),
				ToBlock:   big.NewInt(2),
				Topics:    [][]common.Hash{},
			},
			nil,
			blockHashErr,
		},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			output, err := toFilterArg(testCase.input)
			if (testCase.err == nil) != (err == nil) {
				t.Fatalf("expected error %v but got %v", testCase.err, err)
			}
			if testCase.err != nil {
				if testCase.err.Error() != err.Error() {
					t.Fatalf("expected error %v but got %v", testCase.err, err)
				}
			} else if !reflect.DeepEqual(testCase.output, output) {
				t.Fatalf("expected filter arg %v but got %v", testCase.output, output)
			}
		})
	}
}

func TestNewClient(t *testing.T) {
	type args struct {
		c *rpc.Client
	}
	tests := []struct {
		name string
		args args
		want *Client
	}{
		{
			"new Client",
			args{},
			&Client{},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := NewClient(tt.args.c); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("NewClient() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestClient_StorageAt(t *testing.T) {
	ctx := context.Background()
	ctx, cancel := context.WithTimeout(ctx, 100*time.Millisecond)
	defer cancel()

	type fields struct {
		c *rpc.Client
	}
	type args struct {
		ctx         context.Context
		account     common.Address
		key         common.Hash
		blockNumber *big.Int
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    []byte
		wantErr bool
	}{
		{
			"call StorageAt at future block",
			fields{
				c: &rpc.Client{},
			},
			args{
				ctx:         ctx,
				account:     common.HexToAddress("0x9f8f72aa9304c8b593d555f12ef6589cc3a579a2"),
				key:         common.HexToHash("0x0"),
				blockNumber: big.NewInt(1000000000),
			},
			[]byte(nil),
			true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ec := &Client{
				c: tt.fields.c,
			}
			got, err := ec.StorageAt(tt.args.ctx, tt.args.account, tt.args.key, tt.args.blockNumber)
			if (err != nil) != tt.wantErr {
				t.Errorf("Client.StorageAt() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Client.StorageAt() = %#v, want %#v", got, tt.want)
			}
		})
	}
}

func TestClient_TransactionCount(t *testing.T) {
	ctx := context.Background()
	ctx, cancel := context.WithTimeout(ctx, 100*time.Millisecond)
	defer cancel()

	type fields struct {
		c *rpc.Client
	}
	type args struct {
		ctx       context.Context
		blockHash common.Hash
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    uint
		wantErr bool
	}{
		{
			"call TransactionCount at future block",
			fields{
				c: &rpc.Client{},
			},
			args{
				ctx:       ctx,
				blockHash: common.HexToHash("0xfdea65c8e26263f6d9a1b5de9555d2931a33b825"),
			},
			0,
			true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ec := &Client{
				c: tt.fields.c,
			}
			got, err := ec.TransactionCount(tt.args.ctx, tt.args.blockHash)
			if (err != nil) != tt.wantErr {
				t.Errorf("Client.TransactionCount() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("Client.TransactionCount() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestClient_PendingTransactionCount(t *testing.T) {
	ctx := context.Background()
	ctx, cancel := context.WithTimeout(ctx, 100*time.Millisecond)
	defer cancel()

	type fields struct {
		c *rpc.Client
	}
	type args struct {
		ctx context.Context
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    uint
		wantErr bool
	}{
		{
			"call TransactionCount at pending block",
			fields{
				c: &rpc.Client{},
			},
			args{
				ctx: ctx,
			},
			0,
			true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ec := &Client{
				c: tt.fields.c,
			}
			got, err := ec.PendingTransactionCount(tt.args.ctx)
			if (err != nil) != tt.wantErr {
				t.Errorf("Client.PendingTransactionCount() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("Client.PendingTransactionCount() = %v, want %v", got, tt.want)
			}
		})
	}
}
