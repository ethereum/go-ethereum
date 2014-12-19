// test bench for the package blockhash

package blockhash

import (
	//	"fmt"
	"math"
	"math/rand"
	"testing"
)

func maketest(l int) []byte {

	r := rand.New(rand.NewSource(int64(l)))

	test := make([]byte, l)
	for i := 0; i < l; i++ {
		test[i] = byte(r.Intn(256))
	}

	return test
}

func cmptest(a, b []byte) bool {

	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

const testcnt = 10

func testlen(i int) int {

	/*	if i == 79 {
		return 16777217
	}*/

	return int(0.5 + math.Exp2(3.0+float64(i)/5))
}

func TestBlockHashStorage(t *testing.T) {
	t.Logf("Creating DBStorage...")

	dbstore := new(dpaDBStorage)
	dbstore.Init(nil)
	go dbstore.Run()

	t.Logf("Creating MemStorage...")

	memstore := new(dpaMemStorage)
	memstore.Init(&dbstore.dpaStorage)
	go memstore.Run()

	t.Logf("Storing test vectors...")

	test := make([][]byte, testcnt)
	hash := make([]HashType, testcnt)
	for i := 0; i < testcnt; i++ {
		test[i] = maketest(testlen(i))
		//t.Logf("Test[%d] = %x", i, test[i])
		hash[i] = GetDPAroot(test[i], &memstore.dpaStorage)
		//t.Logf("Hash[%d] = %x", i, hash[i])
	}

	t.Logf("Retrieving test vectors...")

	rnd := rand.New(rand.NewSource(0))

	for i := 0; i < testcnt; i++ {

		tt := GetDPAdata(hash[i], &memstore.dpaStorage) // get the whole vector with byte array wrapper

		sr := GetDPAreader(hash[i], &memstore.dpaStorage)
		size := int(sr.Size())
		pos := rnd.Intn(size - 1)
		slen := rnd.Intn(size-1-pos) + 1
		sr.Seek(int64(pos), 0)
		br, _ := sr.Read(tt[pos : pos+slen]) // re-read a random section

		if (br == slen) && cmptest(test[i], tt) {
			t.Logf("Test case %d passed (test vector length %d)", i, len(tt))
		} else {
			t.Errorf("Test case %d failed", i)
			if size < 20 {
				t.Errorf("pos = %d  slen = %d  br = %d  vector = %x instead of %x", pos, slen, br, tt, test[i])
			}
		}
	}

}
