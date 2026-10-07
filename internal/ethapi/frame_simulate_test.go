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

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/stretchr/testify/require"
)

func TestSimulateFrameTransaction(t *testing.T) {
	backend, args := frameRPCBackend(t, "0x602a60005260206000f3")
	results, err := NewBlockChainAPI(backend).SimulateV1(context.Background(), simOpts{ReturnFullTransactions: true, BlockStateCalls: []simBlock{{Calls: []TransactionArgs{args}}}}, nil)
	require.NoError(t, err)
	require.Len(t, results, 1)
	result := results[0].Calls[0]
	require.Equal(t, hexutil.Uint64(1), result.Status)
	require.Equal(t, args.From, result.Payer)
	require.Equal(t, common.LeftPadBytes([]byte{42}, 32), []byte(result.ReturnValue))
	require.Len(t, result.FrameResults, 2)
	require.Equal(t, result.ReturnValue, result.FrameResults[1].ReturnData)
	encoded, err := json.Marshal(results[0])
	require.NoError(t, err)
	var block struct {
		Transactions []struct {
			Type   string
			Frames []json.RawMessage
		}
		Calls []map[string]json.RawMessage
	}
	require.NoError(t, json.Unmarshal(encoded, &block))
	require.Equal(t, "0x6", block.Transactions[0].Type)
	require.Len(t, block.Transactions[0].Frames, 2)
	require.Contains(t, block.Calls[0], "frameResults")
	require.Contains(t, block.Calls[0], "payer")
	require.Equal(t, uint8(types.FrameTxType), results[0].Block.Transactions()[0].Type())
}

func TestSimulateFrameAtomicRollback(t *testing.T) {
	backend, args := frameRPCBackend(t, "0x36600e5760006000a060006000f35b602a60005260206000fd")
	first := args.Frames[1]
	first.Flags = 4
	second := first
	second.Data = hexutil.Bytes{1}
	third := args.Frames[1]
	fourth := third
	args.Frames = []FrameArgs{args.Frames[0], first, second, third, fourth}
	results, err := NewBlockChainAPI(backend).SimulateV1(context.Background(), simOpts{BlockStateCalls: []simBlock{{Calls: []TransactionArgs{args}}}}, nil)
	require.NoError(t, err)
	result := results[0].Calls[0]
	require.Zero(t, result.Status)
	require.Equal(t, common.LeftPadBytes([]byte{42}, 32), []byte(result.ReturnValue))
	require.Len(t, result.FrameResults, 5)
	for i, status := range []hexutil.Uint64{1, 1, 0, 2, 1} {
		require.Equal(t, status, result.FrameResults[i].Status)
	}
	require.Empty(t, result.FrameResults[1].Logs)
	require.Empty(t, result.FrameResults[2].Logs)
	require.Zero(t, result.FrameResults[3].GasUsed)
	require.Equal(t, 3, result.FrameResults[2].Error.Code)
	require.Len(t, result.Logs, 1)
	require.Equal(t, results[0].Block.Hash(), result.Logs[0].BlockHash)
	require.Len(t, result.FrameResults[4].Logs, 1)
	require.Equal(t, results[0].Block.Hash(), result.FrameResults[4].Logs[0].BlockHash)
}

func TestSimulateFrameDefaultVerifyOnly(t *testing.T) {
	backend, args := frameRPCBackend(t, "0x00")
	args.Frames = args.Frames[:1]
	result, err := NewBlockChainAPI(backend).SimulateV1(context.Background(), simOpts{BlockStateCalls: []simBlock{{Calls: []TransactionArgs{args}}}}, nil)
	require.NoError(t, err)
	require.Len(t, result[0].Calls[0].FrameResults, 1)
	require.Equal(t, hexutil.Uint64(1), result[0].Calls[0].Status)
}

func TestSimulateFrameTransferLogs(t *testing.T) {
	backend, args := frameRPCBackend(t, "0x00")
	value := hexutil.Big(*common.Big1)
	args.Frames[1].Value = &value
	result, err := NewBlockChainAPI(backend).SimulateV1(context.Background(), simOpts{TraceTransfers: true, BlockStateCalls: []simBlock{{Calls: []TransactionArgs{args}}}}, nil)
	require.NoError(t, err)
	require.Len(t, result[0].Calls[0].Logs, 2)
	require.Equal(t, transferAddress, result[0].Calls[0].Logs[0].Address)
	require.Equal(t, common.HexToAddress("0xfffffffffffffffffffffffffffffffffffffffe"), result[0].Calls[0].Logs[1].Address)
	require.Len(t, result[0].Calls[0].FrameResults[1].Logs, 2)
}
