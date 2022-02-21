package kzg

import (
	"encoding/json"

	"github.com/ethereum/go-ethereum/crypto"

	"github.com/ethereum/go-ethereum/params"
	"github.com/protolambda/go-kzg/bls"
)

var crsG2 []bls.G2Point
var crsLagrange []bls.G1Point
var CrsG1 []bls.G1Point // only used in tests (for proof creation)

// Convert polynomial in evaluation form to KZG commitment
func BlobToKzg(eval []bls.Fr) *bls.G1Point {
	return bls.LinCombG1(crsLagrange, eval)
}

// Verify a KZG proof
func VerifyKzgProof(commitment *bls.G1Point, x *bls.Fr, y *bls.Fr, proof *bls.G1Point) bool {
	// Verify the pairing equation
	var xG2 bls.G2Point
	bls.MulG2(&xG2, &bls.GenG2, x)
	var sMinuxX bls.G2Point
	bls.SubG2(&sMinuxX, &crsG2[1], &xG2)
	var yG1 bls.G1Point
	bls.MulG1(&yG1, &bls.GenG1, y)
	var commitmentMinusY bls.G1Point
	bls.SubG1(&commitmentMinusY, commitment, &yG1)

	// This trick may be applied in the BLS-lib specific code:
	//
	// e([commitment - y], [1]) = e([proof],  [s - x])
	//    equivalent to
	// e([commitment - y]^(-1), [1]) * e([proof],  [s - x]) = 1_T
	//
	return bls.PairingsVerify(&commitmentMinusY, &bls.GenG2, proof, &sMinuxX)
}

func KzgToVersionedHash(commitment *bls.G1Point) [32]byte {
	h := crypto.Keccak256Hash(bls.ToCompressedG1(commitment))
	h[0] = byte(params.BlobCommitmentVersionKZG)
	return h
}

type JSONTrustedSetup struct {
	SetupG1       []bls.G1Point
	SetupG2       []bls.G2Point
	SetupLagrange []bls.G1Point
}

// Initialize KZG subsystem (load the trusted setup data)
func init() {
	var parsedSetup = JSONTrustedSetup{}

	// TODO: This is dirty. KZG setup should be loaded using an actual config file directive
	err := json.Unmarshal([]byte(KZGSetupStr), &parsedSetup)
	if err != nil {
		panic(err)
	}

	crsG2 = parsedSetup.SetupG2
	crsLagrange = parsedSetup.SetupLagrange
	CrsG1 = parsedSetup.SetupG1
}
