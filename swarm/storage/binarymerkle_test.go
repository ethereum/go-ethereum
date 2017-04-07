package storage

import (
	"fmt"
	"log"
	"math/rand"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/crypto/sha3"
)

func TestBuildBMT(t *testing.T) {
	for n := 0; n <= 4096; n += 1 {
		fmt.Println("chunksize", n)
		testBuildBMTprv(n, t)
	}
}

func testBuildBMTprv(n int, t *testing.T) {

	data := make([]byte, n)
	tdata := testDataReader(n)
	tdata.Read(data)

	var tree *BTree
	var r *Root
	var count int
	var err1 error
	start := time.Now()
	tree, r, count, err1 = BuildBMT(sha3.NewKeccak256, data, true)
	elapsed := time.Since(start)
	log.Printf("n=%d took %s", n, elapsed)

	if err1 != nil {
		fmt.Println(tree, r, count, err1)
		return
	}
	// for i := 0; i < count; i++ {
	// 	p, err := tree.InclusionProof(i)
	// 	if err != nil {
	// 		fmt.Println("proof failed ", i, err.Error())
	// 		continue
	// 	}
	// 	ok, err := r.CheckProof(sha3.NewKeccak256, p.proof, i)
	//
	// 	if !ok || (err != nil) {
	// 		t.Errorf("proof %d failed", i)
	// 	}
	// }

	offset := rand.Intn(n)
	length := rand.Intn((n-offset+1)-1) + 1
	p, err := tree.GetInclusionProofs(offset, length)
	if err != nil {
		t.Errorf("proof %d failed %s", offset, err)
		return

	}

	ok, err := r.CheckProofs(sha3.NewKeccak256, p)

	if !ok || (err != nil) {
		t.Errorf("proof  failed %s", err)
	} else {
		fmt.Println("proofs ok for offset", offset, "lenght", length, "chunksize", n)
	}

	// ok, err := r.CheckProof(sha3.NewKeccak256, p.proof, i)
	//
	// if !ok || (err != nil) {
	// 	t.Errorf("proof %d failed", i)
	// }

	fmt.Println("done")
}

func benchmarkBuildBMT(n int, t *testing.B) {
	//t.ReportAllocs()
	tdata := testDataReader(n)
	data := make([]byte, n)
	tdata.Read(data)

	//reader := bytes.NewReader(data)

	var tree *BTree
	var r *Root
	var count int
	var err1 error
	//	blocks := splitData(data, 32)
	t.ReportAllocs()
	t.ResetTimer()
	for i := 0; i < t.N; i++ {

		tree, r, count, err1 = BuildBMT(sha3.NewKeccak256, data, false)

		if err1 != nil {
			fmt.Println(err1, tree, r, count)
			return
		}
	}
}

func benchmarkSHA3(n int, t *testing.B) {

	data := make([]byte, n)
	tdata := testDataReader(n)
	tdata.Read(data)
	hashFunc = sha3.NewKeccak256

	t.ReportAllocs()
	t.ResetTimer()

	h := hashFunc()
	for i := 0; i < t.N; i++ {

		h.Reset()
		h.Write(data)
		//binary.Write(h, binary.LittleEndian, count)
		h.Sum(nil)

	}

}

func BenchmarkBuildBMT_4k(t *testing.B)   { benchmarkBuildBMT(4096, t) }
func BenchmarkBuildBMT_2k(t *testing.B)   { benchmarkBuildBMT(4096/2, t) }
func BenchmarkBuildBMT_1k(t *testing.B)   { benchmarkBuildBMT(4096/4, t) }
func BenchmarkBuildBMT_512b(t *testing.B) { benchmarkBuildBMT(4096/8, t) }
func BenchmarkBuildBMT_256b(t *testing.B) { benchmarkBuildBMT(4096/16, t) }
func BenchmarkBuildBMT_128b(t *testing.B) { benchmarkBuildBMT(4096/64, t) }

func BenchmarkBuildSHA3_4k(t *testing.B)   { benchmarkSHA3(4096, t) }
func BenchmarkBuildSHA3_2k(t *testing.B)   { benchmarkSHA3(4096/2, t) }
func BenchmarkBuildSHA3_1k(t *testing.B)   { benchmarkSHA3(4096/4, t) }
func BenchmarkBuildSHA3_512b(t *testing.B) { benchmarkSHA3(4096/8, t) }
func BenchmarkBuildSHA3_256b(t *testing.B) { benchmarkSHA3(4096/16, t) }

func BenchmarkBuildNagiBinaryMerkle_4k(t *testing.B) {
	n := 4096
	data := make([]byte, n)
	tdata := testDataReader(n)
	tdata.Read(data)
	hashFunc = sha3.NewKeccak256

	t.ReportAllocs()
	t.ResetTimer()

	//h := hashFunc()
	for i := 0; i < t.N; i++ {

		BinaryMerkle(data, sha3.NewKeccak256)

	}

}

//func BenchmarkBinaryMerkleTree(t *testing.B) { benchmarkBMT(4096, t) }

// This implementation does not take advantage of any paralellisms and uses
// far more memory than necessary, but it is easy to see that it is correct.
// It can be used for generating test cases for optimized implementations.

func BinaryMerkle(chunk []byte, hasher Hasher) []byte {
	hash := hasher()
	section := 2 * hash.Size()
	l := len(chunk)
	if l > section {
		n := l / section
		r := l - n*section
		hash.Write(chunk[0:r])
		next := hash.Sum(nil)
		for r < l {
			hash.Reset()
			hash.Write(chunk[r : r+section])
			next = hash.Sum(next)
			r += section
		}
		return BinaryMerkle(next, hasher)
	} else {
		hash.Write(chunk)
		return hash.Sum(nil)
	}
}
