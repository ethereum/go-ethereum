package kzg

import (
	"encoding/json"

	gokzg "github.com/protolambda/go-kzg"
	"github.com/protolambda/go-kzg/bls"
)

const CHUNKS_PER_BLOB = 4096

var kzg_settings gokzg.KZGSettings
var lagrange_crs []bls.G1Point

type JSONTrustedSetup struct {
	SetupG1       []bls.G1Point
	SetupG2       []bls.G2Point
	SetupLagrange []bls.G1Point
}

func BlobToKzg(eval []bls.Fr) *bls.G1Point {
	// Convert polynomial in evaluation form to KZG commitment

	// XXX evaluation points?
	return bls.LinCombG1(lagrange_crs, eval)
}

func VerifyKzgProof(commitment bls.G1Point, x bls.Fr, y bls.Fr, proof bls.G1Point) bool {
	return kzg_settings.CheckProofSingle(&commitment, &proof, &x, &y)
}

func ComputeProof(polyCoeff []bls.Fr, x uint64) *bls.G1Point {
	return kzg_settings.ComputeProofSingle(polyCoeff, x)
}

// Initialize KZG subsystem (load the trusted setup data)
func init() {
	var parsedSetup = JSONTrustedSetup{}

	// TODO: This is dirty. KZG setup should be loaded using an actual config file directive
	err := json.Unmarshal([]byte(KZGSetupStr), &parsedSetup)
	if err != nil {
		panic(err)
	}

	kzg_settings.SecretG1 = parsedSetup.SetupG1
	kzg_settings.SecretG2 = parsedSetup.SetupG2
	lagrange_crs = parsedSetup.SetupLagrange
}
