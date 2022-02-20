package tests

import (
	"encoding/hex"
	"fmt"
	"github.com/ethereum/go-ethereum/params"
	"math"
	"strings"
	"testing"

	"github.com/ethereum/go-ethereum/crypto/kzg"

	gokzg "github.com/protolambda/go-kzg"
	"github.com/protolambda/go-kzg/bls"
)

func TestGoKzg(t *testing.T) {
	/// Test the go-kzg library for correctness
	/// Do the trusted setup, generate a polynomial, commit to it, make proof, verify proof.

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
	polynomial := make([]bls.Fr, params.FieldElementsPerBlob, params.FieldElementsPerBlob)
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
	proof := kzgSettings.ComputeProofSingle(polynomial, x)

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

func TestKzg(t *testing.T) {
	/// Test the geth KZG module (use our trusted setup instead of creating a new one)

	// First let's do some go-kzg preparations to be able to convert polynomial between coefficient and evaluation form
	fs := gokzg.NewFFTSettings(uint8(math.Log2(params.FieldElementsPerBlob)))

	// Create testing polynomial (in coefficient form)
	polynomial := make([]bls.Fr, params.FieldElementsPerBlob, params.FieldElementsPerBlob)
	for i := uint64(0); i < params.FieldElementsPerBlob; i++ {
		bls.CopyFr(&polynomial[i], bls.RandomFr())
	}

	// Get polynomial in evaluation form
	evalPoly, err := fs.FFT(polynomial, false)
	if err != nil {
		t.Fatal(err)
	}

	// Now let's start testing the kzg module
	// Create a commitment
	commitment := kzg.BlobToKzg(evalPoly)

	// Create proof for testing
	x := uint64(17)
	proof := kzg.ComputeProof(polynomial, x)

	// Get actual evaluation at x
	var xFr bls.Fr
	bls.AsFr(&xFr, x)
	var value bls.Fr
	bls.EvalPolyAt(&value, polynomial, &xFr)
	t.Log("value\n", bls.FrStr(&value))

	// Verify kzg proof
	if kzg.VerifyKzgProof(commitment, &xFr, &value, proof) != true {
		panic("failed proof verification")
	}
}

func TestBlobVerificationTestVector(t *testing.T) {
	data := []byte(strings.Repeat("HELPMELOVEME ", 10083))[:params.FieldElementsPerBlob*32]

	inputPoints := make([]bls.Fr, params.FieldElementsPerBlob, params.FieldElementsPerBlob)

	var inputPoint [32]byte
	for i := 0; i < params.FieldElementsPerBlob; i++ {
		copy(inputPoint[:32], data[i*32:(i+1)*32])
		bls.FrFrom32(&inputPoints[i], inputPoint)
	}

	commitment := kzg.BlobToKzg(inputPoints)
	versionedHash := kzg.KzgToVersionedHash(commitment)

	testVector := append(versionedHash[:], data[:]...)
	fmt.Printf("%s\n", hex.EncodeToString(testVector))
	fmt.Printf("%d\n", len(testVector))
}

func TestPointEvaluationTestVector(t *testing.T) {
	fs := gokzg.NewFFTSettings(uint8(math.Log2(params.FieldElementsPerBlob)))

	// Create testing polynomial
	polynomial := make([]bls.Fr, params.FieldElementsPerBlob, params.FieldElementsPerBlob)
	for i := uint64(0); i < params.FieldElementsPerBlob; i++ {
		bls.CopyFr(&polynomial[i], bls.RandomFr())
	}

	// Get polynomial in evaluation form
	evalPoly, err := fs.FFT(polynomial, false)
	if err != nil {
		t.Fatal(err)
	}

	// Create a commitment
	commitment := kzg.BlobToKzg(evalPoly)

	// Create proof for testing
	x := uint64(0x42)
	proof := kzg.ComputeProof(polynomial, x)

	// Get actual evaluation at x
	var xFr bls.Fr
	bls.AsFr(&xFr, x)
	var y bls.Fr
	bls.EvalPolyAt(&y, polynomial, &xFr)

	// Verify kzg proof
	if kzg.VerifyKzgProof(commitment, &xFr, &y, proof) != true {
		panic("failed proof verification")
	}

	versionedHash := kzg.KzgToVersionedHash(commitment)

	commitmentBytes := bls.ToCompressedG1(commitment)

	proofBytes := bls.ToCompressedG1(proof)

	xBytes := bls.FrTo32(&xFr)
	yBytes := bls.FrTo32(&y)

	testVector := append(versionedHash[:], xBytes[:]...)
	testVector = append(testVector, yBytes[:]...)
	testVector = append(testVector, commitmentBytes...)
	testVector = append(testVector, proofBytes...)
	fmt.Printf("%s\n", hex.EncodeToString(testVector))
	fmt.Printf("%d\n", len(testVector))
}
