// Copyright 2024 The go-ethereum Authors
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

package rpc

import (
	"github.com/ethereum/go-ethereum/beacon/light/request"
	"github.com/ethereum/go-ethereum/beacon/light/sync"
	"github.com/ethereum/go-ethereum/beacon/types"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/rpc"
)

// RpcServer implements request.requestServer.
type RpcServer struct {
	client        *rpc.Client
	eventCallback func(event request.Event)
	lastId        uint64
}

// NewRpcServer creates a new RpcServer.
func NewRpcServer(client *rpc.Client) *RpcServer {
	return &RpcServer{client: client}
}

// Subscribe implements request.requestServer.
func (s *RpcServer) Subscribe(eventCallback func(event request.Event)) {
	s.eventCallback = eventCallback
}

// SendRequest implements request.requestServer.
func (s *RpcServer) SendRequest(id request.ID, req request.Request) {
	go func() {
		var (
			resp request.Response
			err  error
		)
		switch data := req.(type) {
		case lightclient.ReqHeader:
			var head *types.Header
			if data.Hash != (common.Hash{}) {
				err = s.client.CallContext(ctx, &head, "eth_getBlockByHash", data.Hash, false)
			} else {
				err = s.client.CallContext(ctx, &head, "eth_getBlockByNumber", data.Number, false)
			}
			if err == nil {
				resp = head
			}
		case lightclient.ReqBlock:
			var block *types.Block
			if data.Hash != (common.Hash{}) {
				block, err = s.getBlock(ctx, "eth_getBlockByHash", data.Hash, true)
			} else {
				block, err = s.getBlock(ctx, "eth_getBlockByNumber", data.Number, true)
			}
			if err == nil {
				resp = block
			}
		default:
		}
		if resp != nil {
			s.eventCallback(request.Event{Type: request.EvResponse, Data: request.RequestResponse{ID: id, Request: req, Response: resp}})
		} else {
			s.eventCallback(request.Event{Type: request.EvFail, Data: request.RequestResponse{ID: id, Request: req}})
		}
	}()
}

func (s *RpcServer) getBlock(ctx context.Context, method string, args ...interface{}) (*types.Block, error) {
	var raw json.RawMessage
	err := s.client.CallContext(ctx, &raw, method, args...)
	if err != nil {
		return nil, err
	}

	// Decode header and transactions.
	var head *types.Header
	if err := json.Unmarshal(raw, &head); err != nil {
		return nil, err
	}
	// When the block is not found, the API returns JSON null.
	if head == nil {
		return nil, ethereum.NotFound
	}

	var body rpcBlock
	if err := json.Unmarshal(raw, &body); err != nil {
		return nil, err
	}
	// Quick-verify transaction and uncle lists. This mostly helps with debugging the server.
	if head.UncleHash == types.EmptyUncleHash && len(body.UncleHashes) > 0 {
		return nil, errors.New("server returned non-empty uncle list but block header indicates no uncles")
	}
	if head.UncleHash != types.EmptyUncleHash && len(body.UncleHashes) == 0 {
		return nil, errors.New("server returned empty uncle list but block header indicates uncles")
	}
	if head.TxHash == types.EmptyTxsHash && len(body.Transactions) > 0 {
		return nil, errors.New("server returned non-empty transaction list but block header indicates no transactions")
	}
	if head.TxHash != types.EmptyTxsHash && len(body.Transactions) == 0 {
		return nil, errors.New("server returned empty transaction list but block header indicates transactions")
	}
	// Fill the sender cache of transactions in the block.
	txs := make([]*types.Transaction, len(body.Transactions))
	for i, tx := range body.Transactions {
		if tx.From != nil {
			setSenderFromServer(tx.tx, *tx.From, body.Hash)
		}
		txs[i] = tx.tx
	}
	return types.NewBlockWithHeader(head).WithBody(txs, nil).WithWithdrawals(body.Withdrawals), nil
}

// Unsubscribe implements request.requestServer.
func (s *RpcServer) Unsubscribe() {}
