// Copyright 2025 Berachain Foundation
// This file is part of the bera-geth library.
//
// The bera-geth library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The bera-geth library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the bera-geth library. If not, see <http://www.gnu.org/licenses/>.

package types

import (
	"bytes"
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/params"
)

// samplePubkey returns a deterministic 48-byte pubkey for tests.
func samplePubkey() *common.Pubkey {
	var pk common.Pubkey
	for i := 0; i < common.PubkeyLength; i++ {
		pk[i] = byte(i)
	}
	return &pk
}

// TestNewPoLTx_DataPacking verifies that NewPoLTx produces the expected calldata
// (function selector + ABI-encoded pubkey) and that the transaction fields are
// wired up correctly.
func TestNewPoLTx_DataPacking(t *testing.T) {
	chainID := big.NewInt(1)
	distributor := common.HexToAddress("0x000000000000000000000000000000000000dEaD")
	blockNum := big.NewInt(123)
	pubkey := samplePubkey()
	baseFee := big.NewInt(1000000000)

	tx, err := NewPoLTx(chainID, distributor, blockNum, params.PoLTxGasLimit, baseFee, pubkey)
	if err != nil {
		t.Fatalf("NewPoLTx returned error: %v", err)
	}
	if tx.Type() != PoLTxType {
		t.Fatalf("unexpected tx type: have %d, want %d", tx.Type(), PoLTxType)
	}

	expectedData, err := getDistributeForData(pubkey)
	if err != nil {
		t.Fatalf("getDistributeForData failed: %v", err)
	}
	if !bytes.Equal(tx.Data(), expectedData) {
		t.Fatalf("calldata mismatch\n have: %x\n want: %x", tx.Data(), expectedData)
	}
	// Extra sanity: first 4 bytes must equal the method selector.
	if !bytes.Equal(tx.Data()[:4], distributeForMethod.ID) {
		t.Fatalf("calldata selector mismatch")
	}
}

// TestNewPoLTx_NilPubkey verifies that NewPoLTx returns an error when pubkey is nil.
func TestNewPoLTx_NilPubkey(t *testing.T) {
	chainID := big.NewInt(1)
	distributor := common.HexToAddress("0x000000000000000000000000000000000000dEaD")
	blockNum := big.NewInt(123)
	baseFee := big.NewInt(1000000000)

	// Call NewPoLTx with nil pubkey
	tx, err := NewPoLTx(chainID, distributor, blockNum, params.PoLTxGasLimit, baseFee, nil)
	if err == nil {
		t.Fatalf("expected error for nil pubkey, but got nil")
	}
	if tx != nil {
		t.Fatalf("expected nil transaction when error occurs, but got %v", tx)
	}
	expectedErr := "pubkey cannot be nil for PoL transaction"
	if err.Error() != expectedErr {
		t.Fatalf("error message mismatch: have %q, want %q", err.Error(), expectedErr)
	}
}

// TestNewPoLTx_InvalidBlockNumber ensures that block numbers <= 0 are rejected.
func TestNewPoLTx_InvalidBlockNumber(t *testing.T) {
	chainID := big.NewInt(1)
	distributor := common.Address{}
	baseFee := big.NewInt(1000000000)
	expectedErr := "PoL tx must only be created for a block number greater than 0"

	testCases := []struct {
		name        string
		blockNumber *big.Int
	}{
		{"negative block number", big.NewInt(-1)},
		{"zero block number", big.NewInt(0)},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			tx, err := NewPoLTx(chainID, distributor, tc.blockNumber, params.PoLTxGasLimit, baseFee, samplePubkey())
			if err == nil {
				t.Fatalf("expected error for block number %v, but got nil", tc.blockNumber)
			}
			if tx != nil {
				t.Fatalf("expected nil transaction when error occurs, but got %v", tx)
			}
			if err.Error() != expectedErr {
				t.Fatalf("error message mismatch: have %q, want %q", err.Error(), expectedErr)
			}
		})
	}

	// Test that positive block numbers work correctly
	t.Run("positive block number", func(t *testing.T) {
		positiveBlock := big.NewInt(123)
		tx, err := NewPoLTx(chainID, distributor, positiveBlock, params.PoLTxGasLimit, baseFee, samplePubkey())
		if err != nil {
			t.Fatalf("unexpected error for positive block number: %v", err)
		}
		if tx == nil {
			t.Fatalf("expected transaction for positive block number, but got nil")
		}
		// Nonce should be blockNumber - 1 (as per the implementation)
		expectedNonce := positiveBlock.Uint64() - 1
		if got := tx.Nonce(); got != expectedNonce {
			t.Fatalf("nonce mismatch: have %d, want %d", got, expectedNonce)
		}
	})
}

// TestIsPoLDistribution exercises positive and negative cases for the helper.
func TestIsPoLDistribution(t *testing.T) {
	distributor := common.HexToAddress("0x1000000000000000000000000000000000000001")
	baseFee := big.NewInt(1000000000)
	tx, err := NewPoLTx(big.NewInt(1), distributor, big.NewInt(1), params.PoLTxGasLimit, baseFee, samplePubkey())
	if err != nil {
		t.Fatalf("failed to build PoL tx: %v", err)
	}

	// Positive case.
	if !IsPoLDistribution(params.SystemAddress, &distributor, tx.Data(), distributor) {
		t.Fatalf("expected IsPoLDistribution to return true for valid PoL call")
	}

	// Wrong address.
	otherAddr := common.HexToAddress("0x0200000000000000000000000000000000000002")
	if IsPoLDistribution(params.SystemAddress, &otherAddr, tx.Data(), distributor) {
		t.Fatalf("expected false when distributor address mismatches")
	}

	// Too-short data.
	shortData := []byte{0x01, 0x02, 0x03}
	if IsPoLDistribution(params.SystemAddress, &distributor, shortData, distributor) {
		t.Fatalf("expected false for data shorter than selector")
	}

	// Nil address.
	if IsPoLDistribution(params.SystemAddress, nil, tx.Data(), distributor) {
		t.Fatalf("expected false when to==nil")
	}

	// Wrong from address.
	if IsPoLDistribution(distributor, &distributor, tx.Data(), distributor) {
		t.Fatalf("expected false when from address mismatches")
	}
}

// TestPoLTx_RawSignatureValues confirms that PoLTx reports no signature.
func TestPoLTx_RawSignatureValues(t *testing.T) {
	baseFee := big.NewInt(1000000000)
	tx, err := NewPoLTx(big.NewInt(1), common.Address{}, big.NewInt(1), params.PoLTxGasLimit, baseFee, samplePubkey())
	if err != nil {
		t.Fatalf("failed to create PoL tx: %v", err)
	}
	v, r, s := tx.RawSignatureValues()
	if v.Sign() != 0 || r.Sign() != 0 || s.Sign() != 0 {
		t.Fatalf("expected 0 signature values, have v=%v r=%v s=%v", v, r, s)
	}
}
