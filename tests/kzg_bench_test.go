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

func randomBlob() []bls.Fr {
	blob := make([]bls.Fr, params.FieldElementsPerBlob)
	for i := 0; i < len(blob); i++ {
		blob[i] = *bls.RandomFr()
	}
	return blob
}

func BenchmarkBlobToKzg(b *testing.B) {
	blob := randomBlob()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		kzg.BlobToKzg(blob)
	}
}

func BenchmarkVerifyBlobsWithoutKZGProof(b *testing.B) {
	var blobs [][]bls.Fr
	var commitments []*bls.G1Point
	for i := 0; i < 16; i++ {
		blob := randomBlob()
		blobs = append(blobs, blob)
		commitments = append(commitments, kzg.BlobToKzg(blob))
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		kzg.VerifyBlobsLegacy(commitments, blobs)
	}
}

func BenchmarkVerifyBlobs(b *testing.B) {
	blobs := make([]types.Blob, params.MaxBlobsPerTx)
	var commitments []types.KZGCommitment
	var hashes []common.Hash
	for i := 0; i < len(blobs); i++ {
		tmp := randomBlob()
		for j := range tmp {
			blobs[i][j] = bls.FrTo32(&tmp[j])
		}
		c, ok := blobs[i].ComputeCommitment()
		if !ok {
			b.Fatal("Could not compute commitment")
		}
		commitments = append(commitments, c)
		hashes = append(hashes, c.ComputeVersionedHash())
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

func BenchmarkVerifyKzgProof(b *testing.B) {
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
	commitment := kzg.BlobToKzg(evalPoly)

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
		if kzg.VerifyKzgProof(commitment, &xFr, &value, proof) != true {
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
			for i := 0; i < 10; i++ {
				var blobs []types.Blob
				var commitments []types.KZGCommitment
				var hashes []common.Hash
				for i := 0; i < params.MaxBlobsPerTx; i++ {
					var blobElements types.Blob
					blob := randomBlob()
					for j := range blob {
						blobElements[j] = bls.FrTo32(&blob[j])
					}
					blobs = append(blobs, blobElements)
					c, ok := blobElements.ComputeCommitment()
					if !ok {
						b.Fatal("Could not compute commitment")
					}
					commitments = append(commitments, c)
					hashes = append(hashes, c.ComputeVersionedHash())
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
	//runBenchmark(16)
}

func BenchmarkBatchVerifyWithoutKZGProofs(b *testing.B) {
	runBenchmark := func(siz int) {
		b.Run(fmt.Sprintf("%d", siz), func(b *testing.B) {
			var blobsSet [][][]bls.Fr
			var commitmentsSet [][]*bls.G1Point
			for i := 0; i < siz; i++ {
				var blobs [][]bls.Fr
				var commitments []*bls.G1Point
				for i := 0; i < params.MaxBlobsPerTx; i++ {
					blob := randomBlob()
					blobs = append(blobs, blob)
					commitments = append(commitments, kzg.BlobToKzg(blob))
				}
				blobsSet = append(blobsSet, blobs)
				commitmentsSet = append(commitmentsSet, commitments)
			}

			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				var batchVerify kzg.BlobsBatch
				for i := range blobsSet {
					if err := batchVerify.Join(commitmentsSet[i], blobsSet[i]); err != nil {
						b.Fatalf("unable to join: %v", err)
					}
				}
				if err := batchVerify.Verify(); err != nil {
					b.Fatalf("batch verify failed: %v", err)
				}
			}
		})
	}

	runBenchmark(2)
	runBenchmark(4)
	runBenchmark(8)
	runBenchmark(16)
}
