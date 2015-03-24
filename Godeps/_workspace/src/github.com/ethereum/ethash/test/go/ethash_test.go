package ethashTest

import (
	"bytes"
	"crypto/rand"
	"encoding/hex"
	"log"
	"math/big"
	"testing"

	"github.com/ethereum/ethash"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/ethdb"
)

func TestEthash(t *testing.T) {
	seedHash := make([]byte, 32)
	_, err := rand.Read(seedHash)
	if err != nil {
		panic(err)
	}

	db, err := ethdb.NewMemDatabase()
	if err != nil {
		panic(err)
	}

	blockProcessor, err := core.NewCanonical(5, db)
	if err != nil {
		panic(err)
	}

	log.Println("Block Number: ", blockProcessor.ChainManager().CurrentBlock().Number())

	e := ethash.New(blockProcessor.ChainManager())

	miningHash := make([]byte, 32)
	if _, err := rand.Read(miningHash); err != nil {
		panic(err)
	}
	diff := big.NewInt(10000)
	log.Println("difficulty", diff)

	nonce := uint64(0)

	ghash_full := e.FullHash(nonce, miningHash)
	log.Printf("ethash full (on nonce): %x %x\n", ghash_full, nonce)

	ghash_light := e.LightHash(nonce, miningHash)
	log.Printf("ethash light (on nonce): %x %x\n", ghash_light, nonce)

	if bytes.Compare(ghash_full, ghash_light) != 0 {
		t.Errorf("full: %x, light: %x", ghash_full, ghash_light)
	}
}

func TestGetSeedHash(t *testing.T) {
	seed0, err := ethash.GetSeedHash(0)
	if err != nil {
		t.Errorf("Failed to get seedHash for block 0: %v", err)
	}
	if bytes.Compare(seed0, []byte{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0}) != 0 {
		log.Printf("seedHash for block 0 should be 0s, was: %v\n", seed0)
	}
	seed1, err := ethash.GetSeedHash(30000)
	if err != nil {
		t.Error(err)
	}

	// From python:
	// > from pyethash import get_seedhash
	// > get_seedhash(30000)
	expectedSeed1, err := hex.DecodeString("290decd9548b62a8d60345a988386fc84ba6bc95484008f6362f93160ef3e563")
	if err != nil {
		t.Error(err)
	}

	if bytes.Compare(seed1, expectedSeed1) != 0 {
		log.Printf("seedHash for block 1 should be: %v,\nactual value: %v\n", expectedSeed1, seed1)
	}

}
