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

package txpool

import (
	"crypto/ecdsa"
	"errors"
	"math"
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/params"
	"github.com/holiman/uint256"
)

func TestValidateTransactionEIP2681(t *testing.T) {
	key, err := crypto.GenerateKey()
	if err != nil {
		t.Fatal(err)
	}

	head := &types.Header{
		Number:     big.NewInt(1),
		GasLimit:   5000000,
		Time:       1,
		Difficulty: big.NewInt(1),
	}

	signer := types.LatestSigner(params.TestChainConfig)

	// Create validation options
	opts := &ValidationOptions{
		Config:       params.TestChainConfig,
		Accept:       0xFF, // Accept all transaction types
		MaxSize:      32 * 1024,
		MaxBlobCount: 6,
		MinTip:       big.NewInt(0),
	}

	tests := []struct {
		name    string
		nonce   uint64
		wantErr error
	}{
		{
			name:    "normal nonce",
			nonce:   42,
			wantErr: nil,
		},
		{
			name:    "max allowed nonce (2^64-2)",
			nonce:   math.MaxUint64 - 1,
			wantErr: nil,
		},
		{
			name:    "EIP-2681 nonce overflow (2^64-1)",
			nonce:   math.MaxUint64,
			wantErr: core.ErrNonceMax,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tx := createTestTransaction(key, tt.nonce)
			err := ValidateTransaction(tx, head, signer, opts)

			if tt.wantErr == nil {
				if err != nil {
					t.Errorf("ValidateTransaction() error = %v, wantErr nil", err)
				}
			} else {
				if err == nil {
					t.Errorf("ValidateTransaction() error = nil, wantErr %v", tt.wantErr)
				} else if !errors.Is(err, tt.wantErr) {
					t.Errorf("ValidateTransaction() error = %v, wantErr %v", err, tt.wantErr)
				}
			}
		})
	}
}

// TestValidateFrameTransactionBlobs checks that the pool admits frame
// transactions but rejects those carrying blob hashes, since it has no way to
// hold their sidecars or account for their blob gas when building a block.
func TestValidateFrameTransactionBlobs(t *testing.T) {
	config := *params.MergedTestChainConfig
	config.AmsterdamTime = new(uint64)
	config.BogotaTime = new(uint64)

	head := &types.Header{
		Number:     big.NewInt(1),
		GasLimit:   60_000_000,
		Time:       1,
		Difficulty: common.Big0,
	}
	signer := types.LatestSigner(&config)
	opts := &ValidationOptions{
		Config:  &config,
		Accept:  1 << types.FrameTxType,
		MaxSize: 128 * 1024,
		MinTip:  big.NewInt(0),
	}
	frameTx := func(blobHashes []common.Hash) *types.Transaction {
		sender := common.Address{0xaa}
		target := common.Address{0xbb}
		return types.NewTx(&types.FrameTx{
			ChainID: uint256.MustFromBig(config.ChainID),
			Sender:  sender,
			Frames: []types.Frame{
				{Mode: types.ModeVerify, Flags: types.ApproveExecutionAndPayment, GasLimits: types.Limits{Execution: 5000}, Value: uint256.NewInt(0)},
				{Mode: types.ModeSender, Target: &target, GasLimits: types.Limits{Execution: 30000}, Value: uint256.NewInt(0)},
			},
			Signatures: types.SignatureList{{Scheme: types.FrameTxSchemeSecp256k1, Signer: sender.Bytes(), Signature: make([]byte, 65)}},
			Fees: types.Fees{
				MaxPriorityFeePerGas: uint256.NewInt(1),
				MaxFeePerGas:         uint256.NewInt(100),
				MaxFeePerBlobGas:     uint256.NewInt(100),
			},
			BlobVersionedHashes: blobHashes,
		})
	}
	if err := ValidateTransaction(frameTx(nil), head, signer, opts); err != nil {
		t.Fatalf("frame tx without blobs rejected: %v", err)
	}
	blobHash := common.Hash{0x01}
	if err := ValidateTransaction(frameTx([]common.Hash{blobHash}), head, signer, opts); !errors.Is(err, core.ErrTxTypeNotSupported) {
		t.Fatalf("frame tx with blobs: have %v, want %v", err, core.ErrTxTypeNotSupported)
	}
}

// createTestTransaction creates a basic transaction for testing
func createTestTransaction(key *ecdsa.PrivateKey, nonce uint64) *types.Transaction {
	to := common.HexToAddress("0x0000000000000000000000000000000000000001")

	txdata := &types.LegacyTx{
		Nonce:    nonce,
		To:       &to,
		Value:    big.NewInt(1000),
		Gas:      21000,
		GasPrice: big.NewInt(1),
		Data:     nil,
	}

	tx := types.NewTx(txdata)
	signedTx, _ := types.SignTx(tx, types.HomesteadSigner{}, key)
	return signedTx
}
