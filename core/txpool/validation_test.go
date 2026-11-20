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
	"crypto/sha256"
	"errors"
	"math"
	"math/big"
	"strings"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/crypto/kzg4844"
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

// TestValidateTransactionBlobless tests that a blob transaction without any blob
// versioned hashes is rejected. The execution spec requires such a transaction to
// be invalid (NoBlobDataError) and the block validity path agrees with it
// (core.ErrMissingBlobHashes), but the stateless pool check used to accept it,
// because all the sidecar consistency checks passed vacuously.
func TestValidateTransactionBlobless(t *testing.T) {
	var blob kzg4844.Blob

	commitment, err := kzg4844.BlobToCommitment(&blob)
	if err != nil {
		t.Fatalf("failed to compute commitment: %v", err)
	}
	vhash := kzg4844.CalcBlobHashV1(sha256.New(), &commitment)
	proofs, err := kzg4844.ComputeCellProofs(&blob)
	if err != nil {
		t.Fatalf("failed to compute cell proofs: %v", err)
	}
	tests := []struct {
		name    string
		hashes  []common.Hash
		sidecar *types.BlobTxSidecar
		wantErr bool
	}{
		{
			// A blob transaction must carry at least one blob hash.
			name:    "no blob hashes",
			sidecar: &types.BlobTxSidecar{Version: types.BlobSidecarVersion1},
			wantErr: true,
		},
		{
			// Sanity check that a well-formed transaction is still accepted.
			name:   "single blob",
			hashes: []common.Hash{vhash},
			sidecar: &types.BlobTxSidecar{
				Version:     types.BlobSidecarVersion1,
				Blobs:       []kzg4844.Blob{blob},
				Commitments: []kzg4844.Commitment{commitment},
				Proofs:      proofs,
			},
		},
		{
			// Blobs are not relayed on eth/72, only cells are, so a transaction
			// without blob data but with a matching commitment and the full proof
			// set must remain acceptable.
			name:   "single blob without blob data",
			hashes: []common.Hash{vhash},
			sidecar: &types.BlobTxSidecar{
				Version:     types.BlobSidecarVersion1,
				Commitments: []kzg4844.Commitment{commitment},
				Proofs:      proofs,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tx := types.NewTx(&types.BlobTx{
				ChainID:    uint256.MustFromBig(params.MergedTestChainConfig.ChainID),
				GasTipCap:  uint256.NewInt(1),
				GasFeeCap:  uint256.NewInt(1),
				Gas:        21000,
				To:         common.Address{0x01},
				Value:      uint256.NewInt(0),
				BlobFeeCap: uint256.NewInt(params.BlobTxMinBlobGasprice),
				BlobHashes: tt.hashes,
			})
			key, err := crypto.GenerateKey()
			if err != nil {
				t.Fatal(err)
			}
			signer := types.LatestSigner(params.MergedTestChainConfig)
			tx, err = types.SignTx(tx.WithBlobTxSidecar(tt.sidecar), signer, key)
			if err != nil {
				t.Fatalf("failed to sign transaction: %v", err)
			}
			head := &types.Header{
				Number:     big.NewInt(1),
				GasLimit:   30_000_000,
				Time:       100_000_000, // all forks up to Osaka active
				Difficulty: big.NewInt(0),
			}
			opts := &ValidationOptions{
				Config:       params.MergedTestChainConfig,
				Accept:       0xff,
				MaxSize:      10 * 1024 * 1024,
				MaxBlobCount: params.BlobTxMaxBlobs,
				MinTip:       big.NewInt(0),
			}
			err = ValidateTransaction(tx, head, signer, opts)
			if !tt.wantErr {
				if err != nil {
					t.Fatalf("unexpected validation error: %v", err)
				}
				return
			}
			if err == nil {
				t.Fatal("expected validation error, got nil")
			}
			if !strings.Contains(err.Error(), "blobless blob transaction") {
				t.Fatalf("unexpected error: %v", err)
			}
		})
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
