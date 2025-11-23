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

package miner

import (
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/txpool"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/params"
)

func TestBundleValidation(t *testing.T) {
	bundle := &Bundle{
		Txs:          []*types.Transaction{},
		MinTimestamp: 100,
		MaxTimestamp: 200,
		TargetBlock:  1000,
	}

	// Test timestamp validation
	if err := bundle.ValidateTimestamp(50); err != ErrBundleTimestampTooEarly {
		t.Errorf("Expected ErrBundleTimestampTooEarly, got %v", err)
	}

	if err := bundle.ValidateTimestamp(150); err != nil {
		t.Errorf("Expected no error for valid timestamp, got %v", err)
	}

	if err := bundle.ValidateTimestamp(250); err != ErrBundleTimestampTooLate {
		t.Errorf("Expected ErrBundleTimestampTooLate, got %v", err)
	}
}

func TestBundleCanRevert(t *testing.T) {
	bundle := &Bundle{
		Txs:          []*types.Transaction{},
		RevertingTxs: []int{1, 3, 5},
	}

	testCases := []struct {
		index    int
		expected bool
	}{
		{0, false},
		{1, true},
		{2, false},
		{3, true},
		{4, false},
		{5, true},
		{6, false},
	}

	for _, tc := range testCases {
		result := bundle.CanRevert(tc.index)
		if result != tc.expected {
			t.Errorf("CanRevert(%d) = %v, want %v", tc.index, result, tc.expected)
		}
	}
}

func TestDefaultOrderingStrategy(t *testing.T) {
	strategy := &DefaultOrderingStrategy{}

	// Create some test transactions
	key, _ := crypto.GenerateKey()
	signer := types.LatestSigner(params.TestChainConfig)

	tx1 := types.MustSignNewTx(key, signer, &types.LegacyTx{
		Nonce:    0,
		GasPrice: big.NewInt(1),
		Gas:      21000,
		To:       &common.Address{1},
		Value:    big.NewInt(1),
	})

	tx2 := types.MustSignNewTx(key, signer, &types.LegacyTx{
		Nonce:    1,
		GasPrice: big.NewInt(2),
		Gas:      21000,
		To:       &common.Address{2},
		Value:    big.NewInt(2),
	})

	bundle := &Bundle{
		Txs:         []*types.Transaction{tx1, tx2},
		TargetBlock: 0,
	}

	pending := make(map[common.Address][]*txpool.LazyTransaction)
	header := &types.Header{
		Number: big.NewInt(1),
		Time:   100,
	}

	txs, err := strategy.OrderTransactions(pending, []*Bundle{bundle}, nil, header)
	if err != nil {
		t.Fatalf("OrderTransactions failed: %v", err)
	}

	if len(txs) != 2 {
		t.Errorf("Expected 2 transactions, got %d", len(txs))
	}
}

