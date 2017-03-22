package storage

import (
	"fmt"
	"io/ioutil"
	"testing"

	"github.com/ethereum/go-ethereum/crypto/sha3"
)

func TestGetHeight(t *testing.T) {
	data := [][2]int{
		{0, 0},
		{1, 1},
		{2, 2},
		{3, 3},
		{4, 3},
		{255, 9},
		{256, 9},
		{257, 10},
	}
	for _, v := range data {
		h := GetHeight(uint64(v[0]))
		if !(v[1] == h) {
			t.Errorf("GetHeight(%d)!=%d (was %d)", v[0], v[1], h)
		}
	}
}

func TestGetHeight2(t *testing.T) {
	for i := 1; i < 1000; i++ {
		h := GetHeight(uint64(i))
		upperBound := 1 << uint(h-1)
		lowerBound := (1 << uint(h-2)) + 1
		if i < lowerBound {
			t.Errorf("GetHeight(%d) too high: %d", i, h)
		}
		if i > upperBound {
			t.Errorf("GetHeight(%d) too low: %d", i, h)
		}
	}
}

func TestBuildBMT(t *testing.T) {

	// Grab some data to make the tree out of, and partition
	data, err := ioutil.ReadFile("binarymerkle_test.go") // assume testdata exists
	if err != nil {
		fmt.Println(err)
		return
	}

	tree, r, count, err1 := BuildBMT(sha3.NewKeccak256, data, 32)

	switch err1 {
	case -1:

		t.Errorf("BMT Validation error")
		return
	case -2:
		t.Errorf("BMT leaf count validation error")
		return
	case 0:
		fmt.Println("Build BMT OK")
	}

	fmt.Println(tree.Root())

	for i := 0; i < count; i++ {
		p := tree.InclusionProof(i)

		fmt.Println(p)

		ok := r.CheckProof(sha3.NewKeccak256, p, i)
		if !ok {
			t.Errorf("proof %d failed", i)
		}
	}
}

func TestBuildBMT2(t *testing.T) {

	// Grab some data to make the tree out of, and partition
	data, err := ioutil.ReadFile("binarymerkle_test.go") // assume testdata exists
	if err != nil {
		fmt.Println(err)
		return
	}

	fmt.Println(len(data))
	blocks := splitData(data, 32)

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

		ok := r.CheckProof(sha3.NewKeccak256, p, i)
		if !ok {
			t.Errorf("proof %d failed", i)
		}
	}
	//t.Errorf("proof ok")
	// TODO: check wrong proofs fail

}
