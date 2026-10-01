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
	"crypto/ecdsa"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/crypto/kzg4844"
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

func TestBlobTxSidecarToV1(t *testing.T) {

	// Standard case, converting from v0 to v1
	t.Run("V0ToV1", func(t *testing.T) {
		sidecar := NewBlobTxSidecar(
			BlobSidecarVersion0,
			[]kzg4844.Blob{*emptyBlob},
			[]kzg4844.Commitment{emptyBlobCommit},
			[]kzg4844.Proof{emptyBlobProof},
		)

		if err := sidecar.ToV1(); err != nil {
			t.Fatalf("failed: %v", err)
		}

		if sidecar.Version != BlobSidecarVersion1 {
			t.Errorf("expected version %d, got %d", BlobSidecarVersion1, sidecar.Version)
		}

		// Version 1 should have 128 cell proofs per blob
		expectedProofs := len(sidecar.Blobs) * kzg4844.CellProofsPerBlob
		if len(sidecar.Proofs) != expectedProofs {
			t.Errorf("expected %d proofs, got %d", expectedProofs, len(sidecar.Proofs))
		}
	})

	// Already v1 so its a noop
	t.Run("AlreadyV1", func(t *testing.T) {
		cellProofs, err := kzg4844.ComputeCellProofs(emptyBlob)
		if err != nil {
			t.Fatalf("ComputeCellProofs failed: %v", err)
		}

		sidecar := NewBlobTxSidecar(
			BlobSidecarVersion1,
			[]kzg4844.Blob{*emptyBlob},
			[]kzg4844.Commitment{emptyBlobCommit},
			cellProofs,
		)

		originalProofs := len(sidecar.Proofs)

		if err := sidecar.ToV1(); err != nil {
			t.Fatalf("failed: %v", err)
		}

		if sidecar.Version != BlobSidecarVersion1 {
			t.Errorf("expected version %d, got %d", BlobSidecarVersion1, sidecar.Version)
		}

		if len(sidecar.Proofs) != originalProofs {
			t.Errorf("proofs were modified: expected %d, got %d", originalProofs, len(sidecar.Proofs))
		}
	})

	// Invalid version should return error
	t.Run("InvalidVersion", func(t *testing.T) {
		invalidVersion := byte(2)
		sidecar := NewBlobTxSidecar(
			invalidVersion,
			[]kzg4844.Blob{*emptyBlob},
			[]kzg4844.Commitment{emptyBlobCommit},
			[]kzg4844.Proof{emptyBlobProof},
		)

		err := sidecar.ToV1()
		if err == nil {
			t.Errorf("Invalid version %d should return error, but got nil", invalidVersion)
		}

		// The version shouldn't change on error
		if sidecar.Version != invalidVersion {
			t.Errorf("version changed from %d to %d on error", invalidVersion, sidecar.Version)
		}
	})
}
