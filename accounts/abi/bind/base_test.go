// Copyright 2019 The go-ethereum Authors
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

package bind_test

import (
	"context"
	"math/big"
	"testing"

	"github.com/XinFinOrg/XDPoSChain"
	"github.com/XinFinOrg/XDPoSChain/accounts/abi"
	"github.com/XinFinOrg/XDPoSChain/accounts/abi/bind"
	"github.com/XinFinOrg/XDPoSChain/common"
	"github.com/XinFinOrg/XDPoSChain/core/types"
	"github.com/stretchr/testify/assert"
)

func mockSign(addr common.Address, tx *types.Transaction) (*types.Transaction, error) { return tx, nil }

type mockTransactor struct {
	baseFee                *big.Int
	gasTipCap              *big.Int
	gasPrice               *big.Int
	suggestGasTipCapCalled bool
	suggestGasPriceCalled  bool
}

func (mt *mockTransactor) HeaderByNumber(ctx context.Context, number *big.Int) (*types.Header, error) {
	return &types.Header{BaseFee: mt.baseFee}, nil
}

func (mt *mockTransactor) PendingCodeAt(ctx context.Context, account common.Address) ([]byte, error) {
	return []byte{1}, nil
}

func (mt *mockTransactor) PendingNonceAt(ctx context.Context, account common.Address) (uint64, error) {
	return 0, nil
}

func (mt *mockTransactor) SuggestGasPrice(ctx context.Context) (*big.Int, error) {
	mt.suggestGasPriceCalled = true
	return mt.gasPrice, nil
}

func (mt *mockTransactor) SuggestGasTipCap(ctx context.Context) (*big.Int, error) {
	mt.suggestGasTipCapCalled = true
	return mt.gasTipCap, nil
}

func (mt *mockTransactor) EstimateGas(ctx context.Context, call XDPoSChain.CallMsg) (gas uint64, err error) {
	return 0, nil
}

func (mt *mockTransactor) SendTransaction(ctx context.Context, tx *types.Transaction) error {
	return nil
}

type mockCaller struct {
	codeAtBlockNumber         *big.Int
	callContractBlockNumber   *big.Int
	pendingCodeAtCalled       bool
	pendingCallContractCalled bool
}

func (mc *mockCaller) CodeAt(ctx context.Context, contract common.Address, blockNumber *big.Int) ([]byte, error) {
	mc.codeAtBlockNumber = blockNumber
	return []byte{1, 2, 3}, nil
}

func (mc *mockCaller) CallContract(ctx context.Context, call XDPoSChain.CallMsg, blockNumber *big.Int) ([]byte, error) {
	mc.callContractBlockNumber = blockNumber
	return nil, nil
}

func (mc *mockCaller) PendingCodeAt(ctx context.Context, contract common.Address) ([]byte, error) {
	mc.pendingCodeAtCalled = true
	return nil, nil
}

func (mc *mockCaller) PendingCallContract(ctx context.Context, call XDPoSChain.CallMsg) ([]byte, error) {
	mc.pendingCallContractCalled = true
	return nil, nil
}
func TestPassingBlockNumber(t *testing.T) {

	mc := &mockCaller{}

	bc := bind.NewBoundContract(common.HexToAddress("0x0"), abi.ABI{
		Methods: map[string]abi.Method{
			"something": {
				Name:    "something",
				Outputs: abi.Arguments{},
			},
		},
	}, mc, nil, nil)

	bc.Call(&bind.CallOpts{}, nil, "something")

	bc.Call(&bind.CallOpts{}, nil, "something")

	if mc.callContractBlockNumber != nil {
		t.Fatalf("CallContract() was passed a block number when it should not have been")
	}

	if mc.codeAtBlockNumber != nil {
		t.Fatalf("CodeAt() was passed a block number when it should not have been")
	}

	bc.Call(&bind.CallOpts{Pending: true}, nil, "something")

	if !mc.pendingCallContractCalled {
		t.Fatalf("CallContract() was not passed the block number")
	}

	if !mc.pendingCodeAtCalled {
		t.Fatalf("CodeAt() was not passed the block number")
	}
}

func TestTransactGasFee(t *testing.T) {
	assert := assert.New(t)

	// GasTipCap and GasFeeCap
	// When opts.GasTipCap and opts.GasFeeCap are nil
	mt := &mockTransactor{baseFee: big.NewInt(100), gasTipCap: big.NewInt(5)}
	bc := bind.NewBoundContract(common.Address{}, abi.ABI{}, nil, mt, nil)
	opts := &bind.TransactOpts{Signer: mockSign}
	tx, err := bc.Transact(opts, "")
	assert.Nil(err)
	assert.Equal(big.NewInt(5), tx.GasTipCap())
	assert.Equal(big.NewInt(205), tx.GasFeeCap())
	assert.Nil(opts.GasTipCap)
	assert.Nil(opts.GasFeeCap)
	assert.True(mt.suggestGasTipCapCalled)

	// Second call to Transact should use latest suggested GasTipCap
	mt.gasTipCap = big.NewInt(6)
	mt.suggestGasTipCapCalled = false
	tx, err = bc.Transact(opts, "")
	assert.Nil(err)
	assert.Equal(big.NewInt(6), tx.GasTipCap())
	assert.Equal(big.NewInt(206), tx.GasFeeCap())
	assert.True(mt.suggestGasTipCapCalled)

	// GasPrice
	// When opts.GasPrice is nil
	mt = &mockTransactor{gasPrice: big.NewInt(5)}
	bc = bind.NewBoundContract(common.Address{}, abi.ABI{}, nil, mt, nil)
	opts = &bind.TransactOpts{Signer: mockSign}
	tx, err = bc.Transact(opts, "")
	assert.Nil(err)
	assert.Equal(big.NewInt(5), tx.GasPrice())
	assert.Nil(opts.GasPrice)
	assert.True(mt.suggestGasPriceCalled)

	// Second call to Transact should use latest suggested GasPrice
	mt.gasPrice = big.NewInt(6)
	mt.suggestGasPriceCalled = false
	tx, err = bc.Transact(opts, "")
	assert.Nil(err)
	assert.Equal(big.NewInt(6), tx.GasPrice())
	assert.True(mt.suggestGasPriceCalled)
}
