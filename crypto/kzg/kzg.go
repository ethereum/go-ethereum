package kzg

import (
	"encoding/json"
	"io/ioutil"
	"os"

	gokzg "github.com/protolambda/go-kzg"
	"github.com/protolambda/go-kzg/bls"
)

const CHUNKS_PER_BLOB = 256

// XXX Do we also need the roots of unity in here?
var kzg_settings gokzg.KZGSettings
var lagrange_crs []bls.G1Point

type JSONTrustedSetup struct {
	SetupG1       []string
	SetupG2       []string
	SetupLagrange []string
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

func init() {
	var parsedSetup = JSONTrustedSetup{}

	// XXXX
	jsonFile, err := os.Open("/home/f/Computers/eth/go-ethereum/crypto/kzg/kzg_trusted_setup.json")
	if err != nil {
		panic(err)
	}
	defer jsonFile.Close()

	byteValue, _ := ioutil.ReadAll(jsonFile)

	err = json.Unmarshal(byteValue, &parsedSetup)
	if err != nil {
		panic(err)
	}

	// for i := uint64(0); i < CHUNKS_PER_BLOB; i++ {
	// 	var tmpG1 bls.G1Point
	// 	var tmpG2 bls.G2Point
	// 	tmpG1.SetString(parsedSetup.SetupG1[i])
	// 	tmpG2.SetString(parsedSetup.SetupG2[i])
	// 	kzg_settings.SecretG1 = append(kzg_settings.SecretG1, tmpG1)
	// 	kzg_settings.SecretG2 = append(kzg_settings.SecretG2, tmpG2)
	// 	tmpG1.SetString(parsedSetup.SetupLagrange[i])
	// 	lagrange_crs = append(lagrange_crs, tmpG1)
	// }
}
