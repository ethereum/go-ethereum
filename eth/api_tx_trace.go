// Copyright 2022 The go-ethereum Authors
// // This file is part of the go-ethereum library.
// //
// // The go-ethereum library is free software: you can redistribute it and/or modify
// // it under the terms of the GNU Lesser General Public License as published by
// // the Free Software Foundation, either version 3 of the License, or
// // (at your option) any later version.
// //
// // The go-ethereum library is distributed in the hope that it will be useful,
// // but WITHOUT ANY WARRANTY; without even the implied warranty of
// // MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// // GNU Lesser General Public License for more details.
// //
// // You should have received a copy of the GNU Lesser General Public License
// // along with the go-ethereum library. If not, see <http://www.gnu.org/licenses/>.
//
// // Package ethapi implements the general Ethereum API functions.

package eth

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/rawdb"
)

// PublicTxTraceAPI provides an API to tracing transaction or block information.
// // It offers only methods that operate on public data that is freely available to anyone.
type PublicTxTraceAPI struct {
	e *Ethereum
}

// NewPublicTxTraceAPI creates a new trace API.
func NewPublicTxTraceAPI(e *Ethereum) *PublicTxTraceAPI {
	return &PublicTxTraceAPI{e: e}
}

// Transaction trace_transaction function returns transaction traces.
func (api *PublicTxTraceAPI) Transaction(ctx context.Context, txHash common.Hash) (interface{}, error) {
	if api.e.blockchain == nil {
		return []byte{}, fmt.Errorf("blockchain corruput")
	}
	traceDb := api.e.blockchain.TxTraceDB()
	raw := rawdb.ReadTxTrace(traceDb, txHash)
	if bytes.Equal(raw, []byte{}) { // empty response
		return nil, fmt.Errorf("trace result of tx {%#v} not found in tracedb", txHash)
	}
	var res interface{}
	err := json.Unmarshal(raw, &res)
	if err != nil {
		return nil, err
	}
	return res, nil
}
