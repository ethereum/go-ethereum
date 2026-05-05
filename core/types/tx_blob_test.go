// Copyright 2023 The go-ethereum Authors
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

package types

import (
	"bytes"
	"crypto/ecdsa"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/crypto/kzg4844"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/holiman/uint256"
)

// This test verifies that tx.Hash() is not affected by presence of a BlobTxSidecar.
func TestBlobTxHashing(t *testing.T) {
	key, _ := crypto.GenerateKey()
	withBlobs := createEmptyBlobTx(key, true)
	withBlobsStripped := withBlobs.WithoutBlobTxSidecar()
	withoutBlobs := createEmptyBlobTx(key, false)

	hash := withBlobs.Hash()
	t.Log("tx hash:", hash)

	if h := withBlobsStripped.Hash(); h != hash {
		t.Fatal("wrong tx hash after WithoutBlobTxSidecar:", h)
	}
	if h := withoutBlobs.Hash(); h != hash {
		t.Fatal("wrong tx hash on tx created without sidecar:", h)
	}
}

// This test verifies that tx.Size() takes BlobTxSidecar into account.
func TestBlobTxSize(t *testing.T) {
	key, _ := crypto.GenerateKey()
	withBlobs := createEmptyBlobTx(key, true)
	withBlobsStripped := withBlobs.WithoutBlobTxSidecar()
	withoutBlobs := createEmptyBlobTx(key, false)

	withBlobsEnc, _ := withBlobs.MarshalBinary()
	withoutBlobsEnc, _ := withoutBlobs.MarshalBinary()

	size := withBlobs.Size()
	t.Log("size with blobs:", size)

	sizeNoBlobs := withoutBlobs.Size()
	t.Log("size without blobs:", sizeNoBlobs)

	if size != uint64(len(withBlobsEnc)) {
		t.Error("wrong size with blobs:", size, "encoded length:", len(withBlobsEnc))
	}
	if sizeNoBlobs != uint64(len(withoutBlobsEnc)) {
		t.Error("wrong size without blobs:", sizeNoBlobs, "encoded length:", len(withoutBlobsEnc))
	}
	if sizeNoBlobs >= size {
		t.Error("size without blobs >= size with blobs")
	}
	if sz := withBlobsStripped.Size(); sz != sizeNoBlobs {
		t.Fatal("wrong size on tx after WithoutBlobTxSidecar:", sz)
	}
}

var (
	emptyBlob          = new(kzg4844.Blob)
	emptyBlobCommit, _ = kzg4844.BlobToCommitment(emptyBlob)
	emptyBlobProof, _  = kzg4844.ComputeBlobProof(emptyBlob, emptyBlobCommit)
)

func createEmptyBlobTx(key *ecdsa.PrivateKey, withSidecar bool) *Transaction {
	blobtx := createEmptyBlobTxInner(withSidecar)
	signer := NewCancunSigner(blobtx.ChainID.ToBig())
	return MustSignNewTx(key, signer, blobtx)
}

// TestEncodeForNetwork verifies that EncodeForNetwork produces output identical
// to rlp.EncodeToBytes on the original transaction, for both V0 and V1 sidecars.
func TestEncodeForNetwork(t *testing.T) {
	t.Run("v0", func(t *testing.T) { testEncodeForNetwork(t, BlobSidecarVersion0) })
	t.Run("v1", func(t *testing.T) { testEncodeForNetwork(t, BlobSidecarVersion1) })
}

func testEncodeForNetwork(t *testing.T, version byte) {
	key, _ := crypto.GenerateKey()
	tx := createEmptyBlobTx(key, true)
	if version == BlobSidecarVersion1 {
		if err := tx.BlobTxSidecar().ToV1(); err != nil {
			t.Fatalf("failed to convert sidecar to v1: %v", err)
		}
	}

	wantRLP, err := rlp.EncodeToBytes(tx)
	if err != nil {
		t.Fatalf("failed to encode tx: %v", err)
	}

	sc := tx.BlobTxSidecar()
	ptx := &BlobTxForPool{
		Tx:          tx.WithoutBlobTxSidecar(),
		Version:     sc.Version,
		Commitments: sc.Commitments,
		Proofs:      sc.Proofs,
		Blobs:       sc.Blobs,
	}
	storedRLP, err := rlp.EncodeToBytes(ptx)
	if err != nil {
		t.Fatalf("failed to encode BlobTxForPool: %v", err)
	}

	gotRLP, err := EncodeForNetwork(storedRLP)
	if err != nil {
		t.Fatalf("EncodeForNetwork failed: %v", err)
	}

	if !bytes.Equal(gotRLP, wantRLP) {
		t.Fatalf("network encoding mismatch (version %d): got %d bytes, want %d bytes", version, len(gotRLP), len(wantRLP))
	}
}

func createEmptyBlobTxInner(withSidecar bool) *BlobTx {
	sidecar := NewBlobTxSidecar(BlobSidecarVersion0, []kzg4844.Blob{*emptyBlob}, []kzg4844.Commitment{emptyBlobCommit}, []kzg4844.Proof{emptyBlobProof})
	blobtx := &BlobTx{
		ChainID:    uint256.NewInt(1),
		Nonce:      5,
		GasTipCap:  uint256.NewInt(22),
		GasFeeCap:  uint256.NewInt(5),
		Gas:        25000,
		To:         common.Address{0x03, 0x04, 0x05},
		Value:      uint256.NewInt(99),
		Data:       make([]byte, 50),
		BlobFeeCap: uint256.NewInt(15),
		BlobHashes: sidecar.BlobHashes(),
	}
	if withSidecar {
		blobtx.Sidecar = sidecar
	}
	return blobtx
}
