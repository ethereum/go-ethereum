package storage

import (
	"fmt"
	"testing"

	"github.com/ethereum/go-ethereum/crypto/sha3"
)

func TestBuildBMT3(t *testing.T) {

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

func TestBuildBMT(t *testing.T) {

	// Grab some data to make the tree out of, and partition
	data := make([]byte, 4096)
	tdata := testDataReader(4096)
	tdata.Read(data)
	tree, r, count, err1 := BuildBMT(sha3.NewKeccak256, data, 1)

	if err1 != nil {
		fmt.Println(err1)
		return
	}

	fmt.Println(tree.Root())

	//var ok bool

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

func TestBuildBMT2(t *testing.T) {

	data := make([]byte, 4096)
	tdata := testDataReader(4096)
	tdata.Read(data)

	fmt.Println(len(data))
	blocks := splitData(data, 2)

	count := len(blocks)

	//	t.Errorf("GetCount() != %d (was )", count)

	tree := Build(blocks)
	err1 := tree.Validate()
	if err1 != nil {
		t.Errorf("%s", err1)
	}
	if tree.Count() != uint64(count) {
		t.Errorf("GetCount() != %d (was %d)", count, tree.Count())
	}

	r := Root{uint64(count), tree.Root()}

	fmt.Println(tree.Root())

	for i := 0; i < count; i++ {
		p := tree.InclusionProof(i)

		fmt.Println(p)

		ok, err := r.CheckProof(sha3.NewKeccak256, p, i)
		if !ok || (err != nil) {
			t.Errorf("proof %d failed", i)
		}
	}
	//t.Errorf("proof ok")
	// TODO: check wrong proofs fail

}
