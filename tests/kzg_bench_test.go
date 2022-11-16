// TODO: Migrate these to crypto/kzg
package tests

import (
	"fmt"
	"math"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto/kzg"
	"github.com/ethereum/go-ethereum/params"
	"github.com/holiman/uint256"
	gokzg "github.com/protolambda/go-kzg"
	"github.com/protolambda/go-kzg/bls"
	"github.com/protolambda/ztyp/view"
)

func randomBlob() kzg.Polynomial {
	blob := make(kzg.Polynomial, params.FieldElementsPerBlob)
	for i := 0; i < len(blob); i++ {
		blob[i] = *bls.RandomFr()
	}
	return blob
}

func BenchmarkBlobToKzg(b *testing.B) {
	blob := randomBlob()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		kzg.PolynomialToKZGCommitment(blob)
	}
}

func BenchmarkVerifyBlobs(b *testing.B) {
	blobs := make([]types.Blob, params.MaxBlobsPerBlock)
	var commitments []types.KZGCommitment
	var hashes []common.Hash
	for i := 0; i < len(blobs); i++ {
		tmp := randomBlob()
		for j := range tmp {
			blobs[i][j] = bls.FrTo32(&tmp[j])
		}
		frs, ok := kzg.BlobToPolynomial(blobs[i])
		if !ok {
			b.Fatal("Could not compute commitment")
		}
		c := types.KZGCommitment(kzg.PolynomialToKZGCommitment(frs))
		commitments = append(commitments, c)
		h := common.Hash(kzg.KZGToVersionedHash(kzg.KZGCommitment(c)))
		hashes = append(hashes, h)
	}
	txData := &types.SignedBlobTx{
		Message: types.BlobTxMessage{
			ChainID:             view.Uint256View(*uint256.NewInt(1)),
			Nonce:               view.Uint64View(0),
			Gas:                 view.Uint64View(123457),
			GasTipCap:           view.Uint256View(*uint256.NewInt(42)),
			GasFeeCap:           view.Uint256View(*uint256.NewInt(10)),
			BlobVersionedHashes: hashes,
		},
	}
	_, _, aggregatedProof, err := types.Blobs(blobs).ComputeCommitmentsAndAggregatedProof()
	if err != nil {
		b.Fatal(err)
	}
	wrapData := &types.BlobTxWrapData{
		BlobKzgs:           commitments,
		Blobs:              blobs,
		KzgAggregatedProof: aggregatedProof,
	}
	tx := types.NewTx(txData, types.WithTxWrapData(wrapData))

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if err := tx.VerifyBlobs(); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkVerifyKZGProof(b *testing.B) {
	// First let's do some go-kzg preparations to be able to convert polynomial between coefficient and evaluation form
	fs := gokzg.NewFFTSettings(uint8(math.Log2(params.FieldElementsPerBlob)))

	// Create testing polynomial (in coefficient form)
	polynomial := make([]bls.Fr, params.FieldElementsPerBlob)
	for i := uint64(0); i < params.FieldElementsPerBlob; i++ {
		bls.CopyFr(&polynomial[i], bls.RandomFr())
	}

	// Get polynomial in evaluation form
	evalPoly, err := fs.FFT(polynomial, false)
	if err != nil {
		b.Fatal(err)
	}

	// Now let's start testing the kzg module
	// Create a commitment
	k := kzg.PolynomialToKZGCommitment(evalPoly)
	commitment, _ := bls.FromCompressedG1(k[:])

	// Create proof for testing
	x := uint64(17)
	proof := ComputeProof(polynomial, x, kzg.KzgSetupG1)

	// Get actual evaluation at x
	var xFr bls.Fr
	bls.AsFr(&xFr, x)
	var value bls.Fr
	bls.EvalPolyAt(&value, polynomial, &xFr)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// Verify kzg proof
		if kzg.VerifyKZGProofFromPoints(commitment, &xFr, &value, proof) != true {
			b.Fatal("failed proof verification")
		}
	}
}

func BenchmarkVerifyMultiple(b *testing.B) {
	runBenchmark := func(siz int) {
		b.Run(fmt.Sprintf("%d", siz), func(b *testing.B) {
			var blobsSet [][]types.Blob
			var commitmentsSet [][]types.KZGCommitment
			var hashesSet [][]common.Hash
			for i := 0; i < siz; i++ {
				var blobs []types.Blob
				var commitments []types.KZGCommitment
				var hashes []common.Hash
				for i := 0; i < params.MaxBlobsPerBlock; i++ {
					var blobElements types.Blob
					blob := randomBlob()
					for j := range blob {
						blobElements[j] = bls.FrTo32(&blob[j])
					}
					blobs = append(blobs, blobElements)
					c := types.KZGCommitment(kzg.PolynomialToKZGCommitment(blob))
					commitments = append(commitments, c)
					h := common.Hash(kzg.KZGToVersionedHash(kzg.KZGCommitment(c)))
					hashes = append(hashes, h)
				}
				blobsSet = append(blobsSet, blobs)
				commitmentsSet = append(commitmentsSet, commitments)
				hashesSet = append(hashesSet, hashes)
			}

			var txs []*types.Transaction
			for i := range blobsSet {
				blobs := blobsSet[i]
				commitments := commitmentsSet[i]
				hashes := hashesSet[i]

				txData := &types.SignedBlobTx{
					Message: types.BlobTxMessage{
						ChainID:             view.Uint256View(*uint256.NewInt(1)),
						Nonce:               view.Uint64View(0),
						Gas:                 view.Uint64View(123457),
						GasTipCap:           view.Uint256View(*uint256.NewInt(42)),
						GasFeeCap:           view.Uint256View(*uint256.NewInt(10)),
						BlobVersionedHashes: hashes,
					},
				}
				_, _, aggregatedProof, err := types.Blobs(blobs).ComputeCommitmentsAndAggregatedProof()
				if err != nil {
					b.Fatal(err)
				}
				wrapData := &types.BlobTxWrapData{
					BlobKzgs:           commitments,
					Blobs:              blobs,
					KzgAggregatedProof: aggregatedProof,
				}
				txs = append(txs, types.NewTx(txData, types.WithTxWrapData(wrapData)))
			}

			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				for _, tx := range txs {
					if err := tx.VerifyBlobs(); err != nil {
						b.Fatal(err)
					}
				}
			}
		})
	}

	//runBenchmark(2)
	//runBenchmark(4)
	runBenchmark(8)
	runBenchmark(16)
}
