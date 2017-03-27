package storage

import (
	"fmt"
	"log"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/crypto/sha3"
)

//will test the hash (hash.Hash) interface
func TestBuildBMT2(t *testing.T) {

	// Grab some data to make the tree out of, and partition
	data := make([]byte, 4096)
	tdata := testDataReader(4096)
	tdata.Read(data)
	var thashFunc = MakeHashFunc("BMTSHA3")
	var h = thashFunc()
	h.Reset()
	h.Write(data)
	var key = h.Sum(nil)

	fmt.Println(key)

	// for i := 0; i < count; i++ {
	// 	p := tree.InclusionProof(i)
	//
	// 	fmt.Println(p)
	//
	// 	ok := r.CheckProof(sha3.NewKeccak256, p, i)
	// 	if !ok {
	// 		t.Errorf("proof %d failed", i)
	// 	}
	// }
	// fmt.Println("done")
}

func TestBuildBMTStress(t *testing.T) {

	for i := 1; i < 4096; i++ {
		BuildBMTwithGivenDataLen(i, t)
	}
}

func TestBuildBMT(t *testing.T) {
	BuildBMTwithGivenDataLen(4096, t)
}

func BuildBMTwithGivenDataLen(datalen int, t *testing.T) {

	// Grab some data to make the tree out of, and partition
	fmt.Println("dl", datalen)
	data := make([]byte, datalen)
	tdata := testDataReader(datalen)
	tdata.Read(data)
	fmt.Println(data)

	start := time.Now()

	tree, r, count, err1 := BuildBMT(sha3.NewKeccak256, data, 32)

	elapsed := time.Since(start)
	log.Printf("Binomial took %s", elapsed)

	if err1 != nil {
		fmt.Println(err1)
		return
	}

	fmt.Println(tree.Root(), count)

	for i := 0; i < count; i++ {
		p := tree.InclusionProof(i)

		fmt.Println(p)

		ok, err := r.CheckProof(sha3.NewKeccak256, p, i)

		if !ok || (err != nil) {
			t.Errorf("proof %d failed", i)
		}
	}
	fmt.Println("done")
}
