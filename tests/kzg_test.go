// TODO: Migrate these to go-kzg/eth
package tests

import (
	"encoding/json"
	"io/ioutil"
	"math"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/vm"

	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/params"

	gokzg "github.com/protolambda/go-kzg"
	"github.com/protolambda/go-kzg/bls"
	"github.com/protolambda/go-kzg/eth"
)

// Helper: Long polynomial division for two polynomials in coefficient form
func polyLongDiv(dividend []bls.Fr, divisor []bls.Fr) []bls.Fr {
	a := make([]bls.Fr, len(dividend))
	for i := 0; i < len(a); i++ {
		bls.CopyFr(&a[i], &dividend[i])
	}
	aPos := len(a) - 1
	bPos := len(divisor) - 1
	diff := aPos - bPos
	out := make([]bls.Fr, diff+1)
	for diff >= 0 {
		quot := &out[diff]
		bls.DivModFr(quot, &a[aPos], &divisor[bPos])
		var tmp, tmp2 bls.Fr
		for i := bPos; i >= 0; i-- {
			// In steps: a[diff + i] -= b[i] * quot
			// tmp =  b[i] * quot
			bls.MulModFr(&tmp, quot, &divisor[i])
			// tmp2 = a[diff + i] - tmp
			bls.SubModFr(&tmp2, &a[diff+i], &tmp)
			// a[diff + i] = tmp2
			bls.CopyFr(&a[diff+i], &tmp2)
		}
		aPos -= 1
		diff -= 1
	}
	return out
}

// Helper: Compute proof for polynomial
func ComputeProof(poly []bls.Fr, x uint64, crsG1 []bls.G1Point) *bls.G1Point {
	// divisor = [-x, 1]
	divisor := [2]bls.Fr{}
	var tmp bls.Fr
	bls.AsFr(&tmp, x)
	bls.SubModFr(&divisor[0], &bls.ZERO, &tmp)
	bls.CopyFr(&divisor[1], &bls.ONE)
	//for i := 0; i < 2; i++ {
	//	fmt.Printf("div poly %d: %s\n", i, FrStr(&divisor[i]))
	//}
	// quot = poly / divisor
	quotientPolynomial := polyLongDiv(poly, divisor[:])
	//for i := 0; i < len(quotientPolynomial); i++ {
	//	fmt.Printf("quot poly %d: %s\n", i, FrStr(&quotientPolynomial[i]))
	//}

	// evaluate quotient poly at shared secret, in G1
	return bls.LinCombG1(crsG1[:len(quotientPolynomial)], quotientPolynomial)
}

// Test the go-kzg library for correctness
// Do the trusted setup, generate a polynomial, commit to it, make proof, verify proof.
func TestGoKzg(t *testing.T) {
	// Generate roots of unity
	fs := gokzg.NewFFTSettings(uint8(math.Log2(params.FieldElementsPerBlob)))

	// Create a CRS with `n` elements for `s`
	s := "1927409816240961209460912649124"
	kzgSetupG1, kzgSetupG2 := gokzg.GenerateTestingSetup(s, params.FieldElementsPerBlob)

	// Wrap it all up in KZG settings
	kzgSettings := gokzg.NewKZGSettings(fs, kzgSetupG1, kzgSetupG2)

	kzgSetupLagrange, err := fs.FFTG1(kzgSettings.SecretG1[:params.FieldElementsPerBlob], true)
	if err != nil {
		t.Fatal(err)
	}

	// Create testing polynomial (in coefficient form)
	polynomial := make([]bls.Fr, params.FieldElementsPerBlob)
	for i := uint64(0); i < params.FieldElementsPerBlob; i++ {
		bls.CopyFr(&polynomial[i], bls.RandomFr())
	}

	// Get polynomial in evaluation form
	evalPoly, err := fs.FFT(polynomial, false)
	if err != nil {
		t.Fatal(err)
	}

	// Get commitments to polynomial
	commitmentByCoeffs := kzgSettings.CommitToPoly(polynomial)
	commitmentByEval := gokzg.CommitToEvalPoly(kzgSetupLagrange, evalPoly)
	if !bls.EqualG1(commitmentByEval, commitmentByCoeffs) {
		t.Fatalf("expected commitments to be equal, but got:\nby eval: %s\nby coeffs: %s",
			commitmentByEval, commitmentByCoeffs)
	}

	// Create proof for testing
	x := uint64(17)
	proof := ComputeProof(polynomial, x, kzgSetupG1)

	// Get actual evaluation at x
	var xFr bls.Fr
	bls.AsFr(&xFr, x)
	var value bls.Fr
	bls.EvalPolyAt(&value, polynomial, &xFr)

	// Check proof against evaluation
	if !kzgSettings.CheckProofSingle(commitmentByEval, proof, &xFr, &value) {
		t.Fatal("could not verify proof")
	}
}

type JSONTestdataBlobs struct {
	KzgBlob1 string
	KzgBlob2 string
}

// Test the optimized VerifyBlobs function
func TestVerifyBlobs(t *testing.T) {
	data, err := ioutil.ReadFile("kzg_testdata/kzg_blobs.json")
	if err != nil {
		t.Fatal(err)
	}

	var jsonBlobs JSONTestdataBlobs
	err = json.Unmarshal(data, &jsonBlobs)
	if err != nil {
		t.Fatal(err)
	}

	// Pack all those bytes into two blobs
	var blob1 types.Blob
	var blob2 types.Blob
	for i := 0; i < params.FieldElementsPerBlob; i++ {
		// Be conservative and only pack 31 bytes per Fr element
		copy(blob1[i][:], jsonBlobs.KzgBlob1[i*31:(i+1)*31])
		copy(blob2[i][:], jsonBlobs.KzgBlob2[i*31:(i+1)*31])
	}
	// Compute KZG commitments for both of the blobs above
	kzg1, ok1 := eth.BlobToKZGCommitment(blob1)
	kzg2, ok2 := eth.BlobToKZGCommitment(blob2)
	if ok1 == false || ok2 == false {
		panic("failed to convert blobs")
	}

	// Create the dummy object with all that data we prepared
	blobData := types.BlobTxWrapData{
		BlobKzgs: []types.KZGCommitment{types.KZGCommitment(kzg1), types.KZGCommitment(kzg2)},
		Blobs:    []types.Blob{types.Blob(blob1), types.Blob(blob2)},
	}

	var hashes []common.Hash
	for i := 0; i < len(blobData.BlobKzgs); i++ {
		hashes = append(hashes, blobData.BlobKzgs[i].ComputeVersionedHash())
	}
	txData := &types.SignedBlobTx{
		Message: types.BlobTxMessage{
			BlobVersionedHashes: hashes,
		},
	}
	_, _, aggregatedProof, err := blobData.Blobs.ComputeCommitmentsAndAggregatedProof()
	if err != nil {
		t.Fatalf("bad CommitmentsAndAggregatedProof: %v", err)
	}
	wrapData := &types.BlobTxWrapData{
		BlobKzgs:           blobData.BlobKzgs,
		Blobs:              blobData.Blobs,
		KzgAggregatedProof: aggregatedProof,
	}
	tx := types.NewTx(txData, types.WithTxWrapData(wrapData))

	// Verify the blobs against the commitments!!
	err = tx.VerifyBlobs()
	if err != nil {
		t.Fatalf("bad verifyBlobs: %v", err)
	}

	// Now let's do a bad case:
	// mutate a single chunk of a single blob and VerifyBlobs() must fail
	wrapData.Blobs[0][42][1] = 0x42
	tx = types.NewTx(txData, types.WithTxWrapData(wrapData))
	err = tx.VerifyBlobs()
	if err == nil {
		t.Fatal("bad VerifyBlobs actually succeeded, expected error")
	}
}

// Helper: Create test vector for the PointEvaluation precompile
func TestPointEvaluationTestVector(t *testing.T) {
	// Create testing polynomial
	polynomial := make([]bls.Fr, params.FieldElementsPerBlob)
	for i := uint64(0); i < params.FieldElementsPerBlob; i++ {
		bls.CopyFr(&polynomial[i], bls.RandomFr())
	}

	// Create a commitment
	commitmentArray := eth.PolynomialToKZGCommitment(polynomial)

	// Create proof for testing
	x := uint64(0x42)
	xFr := new(bls.Fr)
	bls.AsFr(xFr, x)
	proofArray, err := eth.ComputeKZGProof(polynomial, xFr)

	// Get actual evaluation at x
	yFr := eth.EvaluatePolynomialInEvaluationForm(polynomial, xFr)
	yArray := bls.FrTo32(yFr)
	xArray := bls.FrTo32(xFr)

	// Verify kzg proof
	ok, err := eth.VerifyKZGProof(commitmentArray, xArray, yArray, proofArray)
	if err != nil {
		t.Fatal(err)
	}
	if !ok {
		t.Fatal("failed proof verification")
	}
	versionedHash := types.KZGCommitment(commitmentArray).ComputeVersionedHash()

	calldata := append(versionedHash[:], xArray[:]...)
	calldata = append(calldata, yArray[:]...)
	calldata = append(calldata, commitmentArray[:]...)
	calldata = append(calldata, proofArray[:]...)

	t.Logf("test-vector: %x", calldata)

	precompile := vm.PrecompiledContractsDanksharding[common.BytesToAddress([]byte{0x14})]
	if _, err := precompile.Run(calldata); err != nil {
		t.Fatalf("expected point verification to succeed")
	}
	// change a byte of the proof
	calldata[144+7] ^= 42
	if _, err := precompile.Run(calldata); err == nil {
		t.Fatalf("expected point verification to fail")
	}
}
