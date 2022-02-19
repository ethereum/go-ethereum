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

	"github.com/ethereum/go-ethereum/crypto/kzg"

	gokzg "github.com/protolambda/go-kzg"
	"github.com/protolambda/go-kzg/bls"
)

const CHUNKS_PER_BLOB = 4096

func main() {
	// Generate roots of unity
	fs := gokzg.NewFFTSettings(uint8(math.Log2(CHUNKS_PER_BLOB)))

	// Create a CRS with `n` elements for `s`
	s := "1927409816240961209460912649124"
	kzg_setup_g1, kzg_setup_g2 := gokzg.GenerateTestingSetup(s, CHUNKS_PER_BLOB)

	kzg_setup_lagrange, err := fs.FFTG1(kzg_setup_g1[:CHUNKS_PER_BLOB], true)
	if err != nil {
		panic(err)
	}

	var trusted_setup = kzg.JSONTrustedSetup{}
	for i := uint64(0); i < CHUNKS_PER_BLOB; i++ {
		trusted_setup.SetupG1 = append(trusted_setup.SetupG1, bls.StrG1(&kzg_setup_g1[i]))
		trusted_setup.SetupG2 = append(trusted_setup.SetupG2, bls.StrG2(&kzg_setup_g2[i]))
		trusted_setup.SetupLagrange = append(trusted_setup.SetupLagrange, bls.StrG1(&kzg_setup_lagrange[i]))
	}

	json_trusted_setup, _ := json.Marshal(trusted_setup)

	err = ioutil.WriteFile("kzg_trusted_setup.json", json_trusted_setup, 0644)
	if err != nil {
		panic(err)
	}

}
