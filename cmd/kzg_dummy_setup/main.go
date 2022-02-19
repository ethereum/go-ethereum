// **NEVER EVER** use the output of this script in production
//
// Do a dummy KZG trusted setup ceremony for testing purposes
// Dump a json file with the results
//
// **NEVER EVER** use the output of this script in production

package main

import (
	"encoding/json"
	"io/ioutil"
	"math"

	gokzg "github.com/protolambda/go-kzg"
	"github.com/protolambda/go-kzg/bls"
)

const CHUNKS_PER_BLOB = 4096

type JSONTrustedSetup struct {
	SetupG1       []bls.G1Point
	SetupG2       []bls.G2Point
	SetupLagrange []bls.G1Point
}

func main() {
	// Generate roots of unity
	fs := gokzg.NewFFTSettings(uint8(math.Log2(CHUNKS_PER_BLOB)))

	// Create a CRS for `s` with CHUNKS_PER_BLOB elements
	s := "1927409816240961209460912649124"
	kzg_setup_g1, kzg_setup_g2 := gokzg.GenerateTestingSetup(s, CHUNKS_PER_BLOB)

	// Also create the lagrange CRS
	kzg_setup_lagrange, err := fs.FFTG1(kzg_setup_g1[:CHUNKS_PER_BLOB], true)
	if err != nil {
		panic(err)
	}

	var trusted_setup = JSONTrustedSetup{}
	trusted_setup.SetupG1 = kzg_setup_g1
	trusted_setup.SetupG2 = kzg_setup_g2
	trusted_setup.SetupLagrange = kzg_setup_lagrange

	json_trusted_setup, _ := json.Marshal(trusted_setup)

	err = ioutil.WriteFile("kzg_trusted_setup.json", json_trusted_setup, 0644)
	if err != nil {
		panic(err)
	}
}
