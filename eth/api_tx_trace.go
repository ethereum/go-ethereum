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
	"fmt"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/rlp"

	txtracelib "github.com/DeBankDeFi/etherlib/pkg/txtracev2"
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

	raw, err := api.e.blockchain.TxTraceStore().ReadTxTrace(ctx, txHash)
	if err != nil {
		return []byte{}, err
	}

	if bytes.Equal(raw, []byte{}) { // empty response
		return nil, fmt.Errorf("trace result of tx {%#v} not found in tracedb", txHash)
	}

	flatten := new(txtracelib.ActionTraceList)
	err = rlp.DecodeBytes(raw, flatten)
	if err != nil {
		return nil, fmt.Errorf("failed to decode rlp flatten traces: %v", err)
	}

	return *flatten, nil
}
