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
	"errors"
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/consensus/beacon"
	"github.com/ethereum/go-ethereum/consensus/ethash"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/internal/ethapi/override"
	"github.com/ethereum/go-ethereum/params"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/stretchr/testify/require"
)

func frameRPCBackend(t *testing.T, code string) (*testBackend, TransactionArgs) {
	t.Helper()
	sender := newTestAccount().addr
	target := common.HexToAddress("0x2222222222222222222222222222222222222222")
	config := *params.MergedTestChainConfig
	config.AmsterdamTime, config.BogotaTime = new(uint64), new(uint64)
	genesis := &core.Genesis{Config: &config, Difficulty: common.Big0, GasLimit: 30000000, Alloc: types.GenesisAlloc{
		sender: {Balance: big.NewInt(params.Ether)},
		target: {Balance: big.NewInt(1), Code: common.FromHex(code)},
	}}
	backend := newTestBackend(t, 0, genesis, beacon.New(ethash.NewFaker()), nil)
	var args TransactionArgs
	require.NoError(t, json.Unmarshal([]byte(`{"type":"0x6","from":"0x1111111111111111111111111111111111111111","frames":[{"mode":"0x1","flags":"0x3","stateGas":"0x0"},{"mode":"0x2","target":"0x2222222222222222222222222222222222222222","stateGas":"0x0"}],"signatures":[{"scheme":"0x1","signer":"0x","msg":"0x","signature":null}]}`), &args))
	args.From = &sender
	return backend, args
}

func TestFrameCall(t *testing.T) {
	backend, args := frameRPCBackend(t, "0x602a60005260206000f3")
	api := NewBlockChainAPI(backend)
	result, err := api.Call(context.Background(), args, nil, nil, nil)
	require.NoError(t, err)
	require.Equal(t, common.LeftPadBytes([]byte{42}, 32), []byte(result))
	require.Nil(t, args.Frames[0].ExecutionGas, "preparation must not modify the caller's request")
	estimate, err := api.EstimateGas(context.Background(), args, nil, nil, nil)
	require.NoError(t, err)
	require.Greater(t, uint64(estimate), params.FrameTxIntrinsicGas)
	acl, err := api.CreateAccessList(context.Background(), args, nil, nil)
	require.NoError(t, err)
	require.Empty(t, acl.Error)
	require.Greater(t, uint64(acl.GasUsed), uint64(0))

	zero := hexutil.Uint64(0)
	args.Frames[1].ExecutionGas = &zero
	_, err = api.Call(context.Background(), args, nil, nil, nil)
	require.ErrorIs(t, err, vm.ErrOutOfGas)
}

func TestFrameCallRevert(t *testing.T) {
	backend, args := frameRPCBackend(t, "0x602a60005260206000fd")
	api := NewBlockChainAPI(backend)
	_, err := api.Call(context.Background(), args, nil, nil, nil)
	var revert *revertError
	require.ErrorAs(t, err, &revert)
	require.Equal(t, hexutil.Encode(common.LeftPadBytes([]byte{42}, 32)), revert.ErrorData())
	_, err = api.EstimateGas(context.Background(), args, nil, nil, nil)
	require.ErrorAs(t, err, &revert)
}

func TestFrameSignaturePlaceholderIsolation(t *testing.T) {
	backend, args := frameRPCBackend(t, "0x00")
	db, header, err := backend.StateAndHeaderByNumberOrHash(context.Background(), rpc.BlockNumberOrHashWithNumber(rpc.LatestBlockNumber))
	require.NoError(t, err)
	block := core.NewEVMBlockContext(header, NewChainContext(context.Background(), backend), nil)
	require.NoError(t, args.prepareFrames(context.Background(), backend, db, header, &block, nil, backend.RPCGasCap()))
	msg := args.ToMessage(header.BaseFee, true)
	msg.SkipTransactionChecks = false
	block.BaseFee = new(big.Int)
	evm := backend.GetEVM(context.Background(), db, header, &vm.Config{NoBaseFee: true}, &block)
	defer evm.Release()
	_, err = core.ApplyMessage(evm, msg, nil)
	require.ErrorIs(t, err, types.ErrFrameTxInvalidSignature)

	invalid := hexutil.Bytes(make([]byte, 65))
	args.Signatures[0].Signature = &invalid
	_, err = NewBlockChainAPI(backend).Call(context.Background(), args, nil, nil, nil)
	require.True(t, errors.Is(err, types.ErrFrameTxInvalidSignature), "%v", err)
}

func TestFrameRequestValidation(t *testing.T) {
	for _, input := range []string{
		`{"type":"0x6","frames":[]}`,
		`{"frames":[{}]}`,
		`{"frames":[{"mode":"0x3"}]}`,
		`{"frames":[{"mode":"0x1","flags":"0x5"}]}`,
		`{"type":"0x2","frames":[{"mode":"0x1"}]}`,
		`{"frames":[{"mode":"0x1"}],"signatures":[{"scheme":"0x1","signature":"0x01"}]}`,
		`{"frames":[{"mode":"0x1"}],"signatures":[{"scheme":"0x0","signer":"0x1111111111111111111111111111111111111111"}]}`,
	} {
		t.Run(input, func(t *testing.T) {
			var args TransactionArgs
			require.NoError(t, json.Unmarshal([]byte(input), &args))
			require.Error(t, args.CallDefaults(1000000, big.NewInt(1), big.NewInt(1)))
		})
	}
}

func TestFrameCompleteSignature(t *testing.T) {
	backend, args := frameRPCBackend(t, "0x602a60005260206000f3")
	gas := hexutil.Uint64(50000)
	for i := range args.Frames {
		args.Frames[i].ExecutionGas = &gas
	}
	require.NoError(t, args.CallDefaults(backend.RPCGasCap(), backend.CurrentHeader().BaseFee, backend.ChainConfig().ChainID))
	hash := args.ToMessage(backend.CurrentHeader().BaseFee, true).FrameSigHash
	sig, err := crypto.Sign(hash.Bytes(), newTestAccount().key)
	require.NoError(t, err)
	vrs := hexutil.Bytes(append([]byte{sig[64]}, sig[:64]...))
	args.Signatures[0].Signature = &vrs
	output, err := NewBlockChainAPI(backend).Call(context.Background(), args, nil, nil, nil)
	require.NoError(t, err)
	require.Equal(t, byte(42), output[len(output)-1])
}

func TestFrameGasPreparationBalance(t *testing.T) {
	backend, args := frameRPCBackend(t, "0x602a60005260206000f3")
	args.MaxFeePerGas = (*hexutil.Big)(big.NewInt(params.GWei))
	args.MaxPriorityFeePerGas = new(hexutil.Big)
	overrides := override.StateOverride{args.from(): {Balance: (*hexutil.Big)(big.NewInt(1000000000000000))}}
	output, err := NewBlockChainAPI(backend).Call(context.Background(), args, nil, &overrides, nil)
	require.NoError(t, err)
	require.Equal(t, byte(42), output[len(output)-1])
}

func TestFrameNullGas(t *testing.T) {
	for _, input := range []string{`{"mode":"0x1","executionGas":null}`, `{"mode":"0x1","stateGas":null}`} {
		var frame FrameArgs
		require.Error(t, json.Unmarshal([]byte(input), &frame))
	}
}

func TestFrameStateGas(t *testing.T) {
	backend, args := frameRPCBackend(t, "0x600160005500")
	args.Frames[1].StateGas = nil
	_, err := NewBlockChainAPI(backend).Call(context.Background(), args, nil, nil, nil)
	require.NoError(t, err)
	zero := hexutil.Uint64(0)
	args.Frames[1].StateGas = &zero
	_, err = NewBlockChainAPI(backend).Call(context.Background(), args, nil, nil, nil)
	require.ErrorIs(t, err, vm.ErrOutOfGas)
}

func TestFrameSignatureSchemes(t *testing.T) {
	backend, args := frameRPCBackend(t, "0x00")
	var extra []FrameSignatureArgs
	require.NoError(t, json.Unmarshal([]byte(`[{"scheme":"0x2","signer":"0x","msg":"0x"},{"scheme":"0x0"},{"scheme":"0x0","signature":"0x"}]`), &extra))
	args.Signatures = append(args.Signatures, extra...)
	_, err := NewBlockChainAPI(backend).Call(context.Background(), args, nil, nil, nil)
	require.NoError(t, err)
}

func TestFrameGasCap(t *testing.T) {
	backend, args := frameRPCBackend(t, "0x00")
	gas := hexutil.Uint64(backend.RPCGasCap())
	args.Frames[0].ExecutionGas = &gas
	_, err := NewBlockChainAPI(backend).Call(context.Background(), args, nil, nil, nil)
	require.ErrorContains(t, err, "exceeds allowance")
}

func TestFrameEmptySignaturePlaceholder(t *testing.T) {
	backend, args := frameRPCBackend(t, "0x602a60005260206000f3")
	empty := hexutil.Bytes{}
	args.Signatures[0].Signature = &empty
	var extra []FrameSignatureArgs
	require.NoError(t, json.Unmarshal([]byte(`[{"scheme":"0x2","signature":"0x"},{"scheme":"0x0","signature":"0x"}]`), &extra))
	args.Signatures = append(args.Signatures, extra...)
	output, err := NewBlockChainAPI(backend).Call(context.Background(), args, nil, nil, nil)
	require.NoError(t, err)
	require.Equal(t, byte(42), output[len(output)-1])
}
