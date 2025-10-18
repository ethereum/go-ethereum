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
	"errors"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/params/forks"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/gorilla/mux"
)

type execApiServer struct {
	apiBackend backend
}

func ExecutionAPI(server *Server, backend backend) API {
	api := execApiServer{apiBackend: backend}
	return func(router *mux.Router) {
		router.HandleFunc("/eth/v1/exec/headers/{blockid}", server.WrapHandler(api.handleHeaders, false, false, true)).Methods("GET")
		router.HandleFunc("/eth/v1/exec/blocks", server.WrapHandler(api.handleBlocks, false, false, true)).Methods("GET")
		router.HandleFunc("/eth/v1/exec/block_receipts", server.WrapHandler(api.handleBlockReceipts, false, false, true)).Methods("GET")
		router.HandleFunc("/eth/v1/exec/transaction", server.WrapHandler(api.handleTransaction, false, false, true)).Methods("GET")
		router.HandleFunc("/eth/v1/exec/transaction_by_index", server.WrapHandler(api.handleTxByIndex, false, false, true)).Methods("GET")
		router.HandleFunc("/eth/v1/exec/receipt_by_index", server.WrapHandler(api.handleReceiptByIndex, false, false, true)).Methods("GET")
		router.HandleFunc("/eth/v1/exec/state", server.WrapHandler(api.handleState, true, true, true)).Methods("POST")
		router.HandleFunc("/eth/v1/exec/call", server.WrapHandler(api.handleCall, true, true, true)).Methods("POST")
		router.HandleFunc("/eth/v1/exec/send_transaction", server.WrapHandler(api.handleSendTransaction, true, true, true)).Methods("POST")
		router.HandleFunc("/eth/v1/exec/history", server.WrapHandler(api.handleHistory, false, false, true)).Methods("GET")
		router.HandleFunc("/eth/v1/exec/transaction_position", server.WrapHandler(api.handleTxPosition, false, false, true)).Methods("GET")
		router.HandleFunc("/eth/v1/exec/logs", server.WrapHandler(api.handleLogs, false, false, true)).Methods("GET")
	}
}

type blockId struct {
	hash   common.Hash
	number uint64
}

func (b *blockId) isHash() bool {
	return b.hash != (common.Hash{})
}

func getBlockId(id string) (blockId, bool) {
	if hex, err := hexutil.Decode(id); err == nil {
		if len(hex) != common.HashLength {
			return blockId{}, false
		}
		var b blockId
		copy(b.hash[:], hex)
		return b, true
	}
	if number, err := strconv.ParseUint(id, 10, 64); err == nil {
		return blockId{number: number}, true
	}
	return blockId{}, false
}

// forkId returns the fork corresponding to the given header.
// Note that frontier thawing and difficulty bomb adjustments are ignored according
// to the API specification as they do not affect the interpretation of the
// returned data structures.
func (s *execApiServer) forkId(header *types.Header) forks.Fork {
	c := s.apiBackend.ChainConfig()
	switch {
	case header.Difficulty.Sign() == 0:
		return c.LatestFork(header.Time)
	case c.IsLondon(header.Number):
		return forks.London
	case c.IsBerlin(header.Number):
		return forks.Berlin
	case c.IsIstanbul(header.Number):
		return forks.Istanbul
	case c.IsPetersburg(header.Number):
		return forks.Petersburg
	case c.IsConstantinople(header.Number):
		return forks.Constantinople
	case c.IsByzantium(header.Number):
		return forks.Byzantium
	case c.IsEIP155(header.Number):
		return forks.SpuriousDragon
	case c.IsEIP150(header.Number):
		return forks.TangerineWhistle
	case c.IsDAOFork(header.Number):
		return forks.DAO
	case c.IsHomestead(header.Number):
		return forks.Homestead
	default:
		return forks.Frontier
	}
}

func (s *execApiServer) forkName(header *types.Header) string {
	return strings.ToLower(s.forkId(header).String())
}

func (s *execApiServer) handleHeaders(ctx context.Context, values url.Values, vars map[string]string, decodeBody func(*any) error) (any, string, int) {
	type headerResponse struct {
		Version string        `json:"version"`
		Data    *types.Header `json:"data"`
	}
	var (
		amount   int
		response []headerResponse
		err      error
	)
	id, ok := getBlockId(vars["blockid"])
	if !ok {
		return nil, "invalid block id", http.StatusBadRequest
	}
	if s := values.Get("amount"); s != "" {
		amount, err = strconv.Atoi(s)
		if err != nil || amount <= 0 {
			return nil, "invalid amount", http.StatusBadRequest
		}
	} else {
		amount = 1
	}

	response = make([]headerResponse, amount)
	for i := amount - 1; i >= 0; i-- {
		if id.isHash() {
			response[i].Data, err = s.apiBackend.HeaderByHash(ctx, id.hash)
		} else {
			response[i].Data, err = s.apiBackend.HeaderByNumber(ctx, rpc.BlockNumber(id.number))
		}
		if errors.Is(err, context.Canceled) {
			return nil, "request timeout", http.StatusRequestTimeout
		}
		if response[i].Data == nil {
			return nil, "not available", http.StatusNotFound
		}
		response[i].Version = s.forkName(response[i].Data)
		if response[i].Data.Number.Uint64() == 0 {
			response = response[i:]
			break
		}
		id = blockId{hash: response[i].Data.ParentHash}
	}
	return response, "", 0
}

func (s *execApiServer) handleBlocks(ctx context.Context, values url.Values, vars map[string]string, decodeBody func(*any) error) (any, string, int) {
	panic("TODO")
}
func (s *execApiServer) handleBlockReceipts(ctx context.Context, values url.Values, vars map[string]string, decodeBody func(*any) error) (any, string, int) {
	panic("TODO")
}
func (s *execApiServer) handleTransaction(ctx context.Context, values url.Values, vars map[string]string, decodeBody func(*any) error) (any, string, int) {
	panic("TODO")
}
func (s *execApiServer) handleTxByIndex(ctx context.Context, values url.Values, vars map[string]string, decodeBody func(*any) error) (any, string, int) {
	panic("TODO")
}
func (s *execApiServer) handleReceiptByIndex(ctx context.Context, values url.Values, vars map[string]string, decodeBody func(*any) error) (any, string, int) {
	panic("TODO")
}
func (s *execApiServer) handleState(ctx context.Context, values url.Values, vars map[string]string, decodeBody func(*any) error) (any, string, int) {
	panic("TODO")
}
func (s *execApiServer) handleCall(ctx context.Context, values url.Values, vars map[string]string, decodeBody func(*any) error) (any, string, int) {
	panic("TODO")
}
func (s *execApiServer) handleHistory(ctx context.Context, values url.Values, vars map[string]string, decodeBody func(*any) error) (any, string, int) {
	panic("TODO")
} // Requires EIP-7745
func (s *execApiServer) handleTxPosition(ctx context.Context, values url.Values, vars map[string]string, decodeBody func(*any) error) (any, string, int) {
	panic("TODO")
} // Requires EIP-7745
func (s *execApiServer) handleLogs(ctx context.Context, values url.Values, vars map[string]string, decodeBody func(*any) error) (any, string, int) {
	panic("TODO")
} // Requires EIP-7745
func (s *execApiServer) handleSendTransaction(ctx context.Context, values url.Values, vars map[string]string, decodeBody func(*any) error) (any, string, int) {
	panic("TODO")
}
