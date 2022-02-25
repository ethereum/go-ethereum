package tests

import (
	"encoding/hex"
	"fmt"
	"math"
	"strings"
	"testing"

	"github.com/ethereum/go-ethereum/params"

	"github.com/ethereum/go-ethereum/crypto/kzg"

	gokzg "github.com/protolambda/go-kzg"
	"github.com/protolambda/go-kzg/bls"
)

// Helper: invert the divisor, then multiply
func polyFactorDiv(dst *bls.Fr, a *bls.Fr, b *bls.Fr) {
	// TODO: use divmod instead.
	var tmp bls.Fr
	bls.InvModFr(&tmp, b)
	bls.MulModFr(dst, &tmp, a)
}

// Helper: Long polynomial division for two polynomials in coefficient form
func polyLongDiv(dividend []bls.Fr, divisor []bls.Fr) []bls.Fr {
	a := make([]bls.Fr, len(dividend), len(dividend))
	for i := 0; i < len(a); i++ {
		bls.CopyFr(&a[i], &dividend[i])
	}
	aPos := len(a) - 1
	bPos := len(divisor) - 1
	diff := aPos - bPos
	out := make([]bls.Fr, diff+1, diff+1)
	for diff >= 0 {
		quot := &out[diff]
		polyFactorDiv(quot, &a[aPos], &divisor[bPos])
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
	proof := ComputeProof(polynomial, x, kzg.CrsG1)

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
	proof := ComputeProof(polynomial, x, kzg.CrsG1)

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
	proof := ComputeProof(polynomial, x, kzg.CrsG1)

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
