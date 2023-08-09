package types

import (
	"crypto/ecdsa"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/crypto/kzg4844"
	"github.com/holiman/uint256"
)

// This test verifies that the transaction hash is not affected by presence of a BlobTxSidecar.
func TestBlobTxHashing(t *testing.T) {
	key, _ := crypto.GenerateKey()
	withBlobs := createEmptyBlobTx(key, true)
	withBlobsStripped := withBlobs.WithoutBlobTxSidecar()
	withoutBlobs := createEmptyBlobTx(key, false)

	hash := withBlobs.Hash()
	size := withBlobs.Size()
	t.Log("tx hash:", hash)
	t.Log("tx size:", size)

	if h := withBlobsStripped.Hash(); h != hash {
		t.Fatal("wrong tx hash after WithoutBlobTxSidecar:", h)
	}
	if h := withoutBlobs.Hash(); h != hash {
		t.Fatal("wrong tx hash on tx created without sidecar:", h)
	}
}

var (
	emptyBlob          = kzg4844.Blob{}
	emptyBlobCommit, _ = kzg4844.BlobToCommitment(emptyBlob)
	emptyBlobProof, _  = kzg4844.ComputeBlobProof(emptyBlob, emptyBlobCommit)
)

func createEmptyBlobTx(key *ecdsa.PrivateKey, withSidecar bool) *Transaction {
	sidecar := &BlobTxSidecar{
		Blobs:       []kzg4844.Blob{emptyBlob},
		Commitments: []kzg4844.Commitment{emptyBlobCommit},
		Proofs:      []kzg4844.Proof{emptyBlobProof},
	}
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
	signer := NewCancunSigner(blobtx.ChainID.ToBig())
	return MustSignNewTx(key, signer, blobtx)
}
