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

// TestNewPoLTx_NegativeBlockNumber ensures negative block numbers are handled
// without panicking (uint64 wrap-around is expected).
func TestNewPoLTx_NegativeBlockNumber(t *testing.T) {
	chainID := big.NewInt(1)
	distributor := common.Address{}
	negBlock := big.NewInt(-1)
	baseFee := big.NewInt(1000000000)

	tx, err := NewPoLTx(chainID, distributor, negBlock, params.PoLTxGasLimit, baseFee, samplePubkey())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got, want := tx.Nonce(), negBlock.Uint64(); got != want {
		t.Fatalf("nonce mismatch: have %d, want %d", got, want)
	}
}

// TestIsPoLDistribution exercises positive and negative cases for the helper.
func TestIsPoLDistribution(t *testing.T) {
	distributor := common.HexToAddress("0x1000000000000000000000000000000000000001")
	baseFee := big.NewInt(1000000000)
	tx, err := NewPoLTx(big.NewInt(1), distributor, big.NewInt(0), params.PoLTxGasLimit, baseFee, samplePubkey())
	if err != nil {
		t.Fatalf("failed to build PoL tx: %v", err)
	}

	// Positive case.
	if !IsPoLDistribution(&distributor, tx.Data(), distributor) {
		t.Fatalf("expected IsPoLDistribution to return true for valid PoL call")
	}

	// Wrong address.
	otherAddr := common.HexToAddress("0x0200000000000000000000000000000000000002")
	if IsPoLDistribution(&otherAddr, tx.Data(), distributor) {
		t.Fatalf("expected false when distributor address mismatches")
	}

	// Too-short data.
	shortData := []byte{0x01, 0x02, 0x03}
	if IsPoLDistribution(&distributor, shortData, distributor) {
		t.Fatalf("expected false for data shorter than selector")
	}

	// Nil address.
	if IsPoLDistribution(nil, tx.Data(), distributor) {
		t.Fatalf("expected false when to==nil")
	}
}

// TestPoLTx_RawSignatureValues confirms that PoLTx reports no signature.
func TestPoLTx_RawSignatureValues(t *testing.T) {
	baseFee := big.NewInt(1000000000)
	tx, err := NewPoLTx(big.NewInt(1), common.Address{}, big.NewInt(0), params.PoLTxGasLimit, baseFee, samplePubkey())
	if err != nil {
		t.Fatalf("failed to create PoL tx: %v", err)
	}
	v, r, s := tx.RawSignatureValues()
	if v != nil || r != nil || s != nil {
		t.Fatalf("expected nil signature values, have v=%v r=%v s=%v", v, r, s)
	}
}
