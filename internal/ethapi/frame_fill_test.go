// Copyright 2026 The go-ethereum Authors
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

package ethapi

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/ethereum/go-ethereum/core/types"
	"github.com/stretchr/testify/require"
)

func TestFillFrameTransaction(t *testing.T) {
	backend, args := frameRPCBackend(t, "0x602a60005260206000f3")
	result, err := NewTransactionAPI(backend, new(AddrLocker)).FillTransaction(context.Background(), args)
	require.NoError(t, err)
	require.Equal(t, uint8(types.FrameTxType), result.Tx.Type())
	encoded, err := json.Marshal(result)
	require.NoError(t, err)
	var response struct {
		Tx  map[string]json.RawMessage `json:"tx"`
		Raw *string                    `json:"raw"`
	}
	require.NoError(t, json.Unmarshal(encoded, &response))
	require.Nil(t, response.Raw, "placeholder requests have no signed wire encoding")
	require.JSONEq(t, `"0x6"`, string(response.Tx["type"]))
	for _, field := range []string{"chainId", "nonce", "from", "maxFeePerGas", "maxPriorityFeePerGas", "maxFeePerBlobGas"} {
		require.NotEmpty(t, response.Tx[field])
	}
	require.NotContains(t, response.Tx, "sender")
	var filled TransactionArgs
	txJSON, err := json.Marshal(response.Tx)
	require.NoError(t, err)
	require.NoError(t, json.Unmarshal(txJSON, &filled))
	require.Len(t, filled.Frames, 2)
	require.Greater(t, uint64(*filled.Frames[0].ExecutionGas), uint64(0))
	require.Zero(t, uint64(*filled.Frames[0].StateGas))
	require.Zero(t, uint64(*filled.Frames[1].StateGas))
	require.Nil(t, filled.Signatures[0].Signature)
	output, err := NewBlockChainAPI(backend).Call(context.Background(), filled, nil, nil, nil)
	require.NoError(t, err)
	require.Equal(t, byte(42), output[len(output)-1])
}
