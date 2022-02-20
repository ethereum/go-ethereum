package tests

import (
	"encoding/hex"
	"fmt"
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
	fs := gokzg.NewFFTSettings(uint8(math.Log2(kzg.CHUNKS_PER_BLOB)))

	// Create a CRS with `n` elements for `s`
	s := "1927409816240961209460912649124"
	kzg_setup_g1, kzg_setup_g2 := gokzg.GenerateTestingSetup(s, kzg.CHUNKS_PER_BLOB)

	// Wrap it all up in KZG settings
	kzg_settings := gokzg.NewKZGSettings(fs, kzg_setup_g1, kzg_setup_g2)

	kzg_setup_lagrange, err := fs.FFTG1(kzg_settings.SecretG1[:kzg.CHUNKS_PER_BLOB], true)
	if err != nil {
		t.Fatal(err)
	}

	// Create testing polynomial (in coefficient form)
	polynomial := make([]bls.Fr, kzg.CHUNKS_PER_BLOB, kzg.CHUNKS_PER_BLOB)
	for i := uint64(0); i < kzg.CHUNKS_PER_BLOB; i++ {
		bls.CopyFr(&polynomial[i], bls.RandomFr())
	}

	// Get polynomial in evaluation form
	evalPoly, err := fs.FFT(polynomial, false)
	if err != nil {
		t.Fatal(err)
	}

	// Get commitments to polynomial
	commitmentByCoeffs := kzg_settings.CommitToPoly(polynomial)
	commitmentByEval := gokzg.CommitToEvalPoly(kzg_setup_lagrange, evalPoly)
	if !bls.EqualG1(commitmentByEval, commitmentByCoeffs) {
		t.Fatalf("expected commitments to be equal, but got:\nby eval: %s\nby coeffs: %s",
			commitmentByEval, commitmentByCoeffs)
	}

	// Create proof for testing
	x := uint64(17)
	proof := kzg_settings.ComputeProofSingle(polynomial, x)

	// Get actual evaluation at x
	var x_fr bls.Fr
	bls.AsFr(&x_fr, x)
	var value bls.Fr
	bls.EvalPolyAt(&value, polynomial, &x_fr)

	// Check proof against evaluation
	if !kzg_settings.CheckProofSingle(commitmentByEval, proof, &x_fr, &value) {
		t.Fatal("could not verify proof")
	}
}

func TestKzg(t *testing.T) {
	/// Test the geth KZG module (use our trusted setup instead of creating a new one)

	// First let's do some go-kzg preparations to be able to convert polynomial between coefficient and evaluation form
	fs := gokzg.NewFFTSettings(uint8(math.Log2(kzg.CHUNKS_PER_BLOB)))

	// Create testing polynomial (in coefficient form)
	polynomial := make([]bls.Fr, kzg.CHUNKS_PER_BLOB, kzg.CHUNKS_PER_BLOB)
	for i := uint64(0); i < kzg.CHUNKS_PER_BLOB; i++ {
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
	var x_fr bls.Fr
	bls.AsFr(&x_fr, x)
	var value bls.Fr
	bls.EvalPolyAt(&value, polynomial, &x_fr)
	t.Log("value\n", bls.FrStr(&value))

	// Verify kzg proof
	if kzg.VerifyKzgProof(*commitment, x_fr, value, *proof) != true {
		panic("failed proof verification")
	}
}

func TestBlobVerificationTestVector(t *testing.T) {
	data := []byte(strings.Repeat("HELPMELOVEME ", 10083))[:kzg.CHUNKS_PER_BLOB*32]

	inputPoints := make([]bls.Fr, kzg.CHUNKS_PER_BLOB, kzg.CHUNKS_PER_BLOB)

	var inputPoint [32]byte
	for i := 0; i < kzg.CHUNKS_PER_BLOB; i++ {
		copy(inputPoint[:32], data[i*32:(i+1)*32])
		bls.FrFrom32(&inputPoints[i], inputPoint)
	}

	commitment := kzg.BlobToKzg(inputPoints)
	versioned_hash := kzg.KzgToVersionedHash(*commitment)

	test_vector = append(versioned_hash[:], data[:]...)
	fmt.Printf("%s\n", hex.EncodeToString(test_vector))
	fmt.Printf("%d\n", len(test_vector))
}
