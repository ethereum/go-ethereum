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
	"encoding/json"
	"errors"
	"mime"
	"net/http"
	"strconv"
	"strings"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/params/forks"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/ethereum/go-ethereum/rpc"
)

var (
	urlHeaders        = "/eth/v1/exec/headers/"
	urlBlocks         = "/eth/v1/exec/blocks"
	urlBlockReceipts  = "/eth/v1/exec/block_receipts"
	urlTransaction    = "/eth/v1/exec/transaction"
	urlTxByIndex      = "/eth/v1/exec/transaction_by_index"
	urlReceiptByIndex = "/eth/v1/exec/receipt_by_index"
	urlState          = "/eth/v1/exec/state"
	urlCall           = "/eth/v1/exec/call"
	// Requires EIP-7745
	urlHistory    = "/eth/v1/exec/history"
	urlTxPosition = "/eth/v1/exec/transaction_position"
	urlLogs       = "/eth/v1/exec/logs"
)

type execApiServer struct {
	apiBackend                 Backend
	maxAmount, maxResponseSize int
}

func NewExecutionRestAPI(apiBackend Backend) func(mux *http.ServeMux, maxResponseSize int) {
	return func(mux *http.ServeMux, maxResponseSize int) {
		s := &execApiServer{
			apiBackend:      apiBackend,
			maxResponseSize: maxResponseSize,
		}
		mux.HandleFunc(urlHeaders, s.handleHeaders)
		mux.HandleFunc(urlBlocks, s.handleBlocks)
		mux.HandleFunc(urlBlockReceipts, s.handleBlockReceipts)
		mux.HandleFunc(urlTransaction, s.handleTransaction)
		mux.HandleFunc(urlTxByIndex, s.handleTxByIndex)
		mux.HandleFunc(urlReceiptByIndex, s.handleReceiptByIndex)
		mux.HandleFunc(urlState, s.handleState)
		mux.HandleFunc(urlCall, s.handleCall)
		// Requires EIP-7745
		mux.HandleFunc(urlHistory, s.handleHistory)
		mux.HandleFunc(urlTxPosition, s.handleTxPosition)
		mux.HandleFunc(urlLogs, s.handleLogs)
	}
}

type blockId struct {
	hash   common.Hash
	number uint64
}

func (b *blockId) isHash() bool {
	return b.hash != (common.Hash{})
}

func decodeBlockId(id string) (blockId, bool) {
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

func (s *execApiServer) handleHeaders(resp http.ResponseWriter, req *http.Request) {
	type headerResponse struct {
		Version string        `json:"version"`
		Data    *types.Header `json:"data"`
	}
	var (
		amount   int
		response []headerResponse
		binary   bool
		err      error
	)

	if mt, _, err := mime.ParseMediaType(req.Header.Get("accept")); err == nil {
		switch mt {
		case "application/json":
		case "application/octet-stream":
			binary = true
		default:
			http.Error(resp, "invalid accepted media type", http.StatusNotAcceptable)
		}
	}
	id, ok := decodeBlockId(req.URL.Path[len(urlHeaders):])
	if !ok {
		http.Error(resp, "invalid block id", http.StatusBadRequest)
		return
	}
	if s := req.URL.Query().Get("amount"); s != "" {
		amount, err = strconv.Atoi(s)
		if err != nil || amount <= 0 {
			http.Error(resp, "invalid amount", http.StatusBadRequest)
			return
		}
	} else {
		amount = 1
	}

	response = make([]headerResponse, amount)
	for i := amount - 1; i >= 0; i-- {
		if id.isHash() {
			response[i].Data, err = s.apiBackend.HeaderByHash(req.Context(), id.hash)
		} else {
			response[i].Data, err = s.apiBackend.HeaderByNumber(req.Context(), rpc.BlockNumber(id.number))
		}
		if errors.Is(err, context.Canceled) {
			http.Error(resp, "request timeout", http.StatusRequestTimeout)
			return
		}
		if response[i].Data == nil {
			http.Error(resp, "not available", http.StatusNotFound)
			return
		}
		response[i].Version = s.forkName(response[i].Data)
		if response[i].Data.Number.Uint64() == 0 {
			response = response[i:]
			break
		}
		id = blockId{hash: response[i].Data.ParentHash}
	}

	if binary {
		respRlp, err := rlp.EncodeToBytes(response)
		if err != nil {
			http.Error(resp, "response encoding error", http.StatusInternalServerError)
			return
		}
		resp.Header().Set("content-type", "application/octet-stream")
		resp.Write(respRlp)
	} else {
		respJson, err := json.Marshal(response)
		if err != nil {
			http.Error(resp, "response encoding error", http.StatusInternalServerError)
			return
		}
		resp.Header().Set("content-type", "application/json")
		resp.Write(respJson)
	}
}

func (s *execApiServer) handleBlocks(resp http.ResponseWriter, req *http.Request) { panic("TODO") }
func (s *execApiServer) handleBlockReceipts(resp http.ResponseWriter, req *http.Request) {
	panic("TODO")
}
func (s *execApiServer) handleTransaction(resp http.ResponseWriter, req *http.Request) { panic("TODO") }
func (s *execApiServer) handleTxByIndex(resp http.ResponseWriter, req *http.Request)   { panic("TODO") }
func (s *execApiServer) handleReceiptByIndex(resp http.ResponseWriter, req *http.Request) {
	panic("TODO")
}
func (s *execApiServer) handleState(resp http.ResponseWriter, req *http.Request)      { panic("TODO") }
func (s *execApiServer) handleCall(resp http.ResponseWriter, req *http.Request)       { panic("TODO") }
func (s *execApiServer) handleHistory(resp http.ResponseWriter, req *http.Request)    { panic("TODO") } // Requires EIP-7745
func (s *execApiServer) handleTxPosition(resp http.ResponseWriter, req *http.Request) { panic("TODO") } // Requires EIP-7745
func (s *execApiServer) handleLogs(resp http.ResponseWriter, req *http.Request)       { panic("TODO") } // Requires EIP-7745
