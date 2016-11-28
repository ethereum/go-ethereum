package network

import (
	"errors"
	"fmt"
	"math/rand"
	"runtime"
	"sync"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/logger/glog"
)

func init() {
	glog.SetV(4)
	glog.SetToStderr(true)
}

type testAddr struct {
	*BinAddr
	i int
}

func NewTestAddr(s string, i int) *testAddr {
	return &testAddr{NewBinAddr(s), i}
}

func str(v PbVal) string {
	if v == nil {
		return ""
	}
	return v.(*testAddr).String()
}

func indexes(t *PbTree) (i []int, pos []int) {
	t.Each(func(v PbVal, po int) bool {
		a := v.(*testAddr)
		i = append(i, a.i)
		pos = append(pos, po)
		return true
	})
	return i, pos
}

func add(t *PbTree, n int, values ...string) {
	for i, val := range values {
		t.Add(NewTestAddr(val, i+n))
	}
}

func (self *testAddr) Prefix(val PbVal, pos int) (po int, eq bool) {
	return self.BinAddr.Prefix(val.(*testAddr).BinAddr, pos)
}

// func RandomBinAddr()
func TestPbTreeAdd(t *testing.T) {
	n := NewPbTree(NewTestAddr("001111", 0), 0)
	// Pin set correctly
	exp := "001111"
	got := str(n.Pin())
	if got != exp {
		t.Fatalf("incorrect pinned value. Expected %v, got %v", exp, got)
	}
	// check size
	goti := n.Size()
	expi := 1
	if goti != expi {
		t.Fatalf("incorrect number of elements in PbTree. Expected %v, got %v", expi, goti)
	}

	add(n, 1, "011111", "001111", "011111", "000111")
	// check size
	goti = n.Size()
	expi = 3
	if goti != expi {
		t.Fatalf("incorrect number of elements in PbTree. Expected %v, got %v", expi, goti)
	}
	inds, pos := indexes(n)
	got = fmt.Sprintf("%v", inds)
	exp = "[3 4 2]"
	if got != exp {
		t.Fatalf("incorrect indexes in iteration over PbTree. Expected %v, got %v", exp, got)
	}
	got = fmt.Sprintf("%v", pos)
	exp = "[1 2 0]"
	if got != exp {
		t.Fatalf("incorrect po-s in iteration over PbTree. Expected %v, got %v", exp, got)
	}
}

// func RandomBinAddr()
func TestPbTreeRemove(t *testing.T) {
	n := NewPbTree(NewTestAddr("001111", 0), 0)
	n.Remove(NewTestAddr("001111", 0))
	exp := ""
	got := str(n.Pin())
	if got != exp {
		t.Fatalf("incorrect pinned value. Expected %v, got %v", exp, got)
	}
	add(n, 1, "000000", "011111", "001111", "000111")
	n.Remove(NewTestAddr("001111", 0))
	goti := n.Size()
	expi := 3
	if goti != expi {
		t.Fatalf("incorrect number of elements in PbTree. Expected %v, got %v", expi, goti)
	}
	inds, pos := indexes(n)
	got = fmt.Sprintf("%v", inds)
	exp = "[2 4 1]"
	if got != exp {
		t.Fatalf("incorrect indexes in iteration over PbTree. Expected %v, got %v", exp, got)
	}
	got = fmt.Sprintf("%v", pos)
	exp = "[1 3 0]"
	if got != exp {
		t.Fatalf("incorrect po-s in iteration over PbTree. Expected %v, got %v", exp, got)
	}
	// remove again
	n.Remove(NewTestAddr("001111", 0))
	inds, pos = indexes(n)
	got = fmt.Sprintf("%v", inds)
	exp = "[2 4 1]"
	if got != exp {
		t.Fatalf("incorrect indexes in iteration over PbTree. Expected %v, got %v", exp, got)
	}

}

func checkPo(val PbVal) func(PbVal, int) error {
	return func(v PbVal, po int) error {
		// check the po
		exp, _ := val.Prefix(v, 0)
		if po != exp {
			return fmt.Errorf("incorrect prox order for item %v in neighbour iteration for %v. Expected %v, got %v", v, val, exp, po)
		}
		return nil
	}
}

func checkOrder(val PbVal) func(PbVal, int) error {
	var pos int = keylen
	return func(v PbVal, po int) error {
		if pos < po {
			return fmt.Errorf("incorrect order for item %v in neighbour iteration for %v. PO %v > %v (previous max)", v, val, po, pos)
		}
		pos = po
		return nil
	}
}

func checkValues(m map[string]bool, val PbVal) func(PbVal, int) error {
	return func(v PbVal, po int) error {
		duplicate, ok := m[v.String()]
		if !ok {
			return fmt.Errorf("alien value %v", v)
		}
		if duplicate {
			return fmt.Errorf("duplicate value returned: %v", v)
		}
		m[v.String()] = true
		return nil
	}
}

var errNoCount = errors.New("not count")

func testPbTreeEachNeighbour(n *PbTree, val PbVal, expCount int, fs ...func(PbVal, int) error) error {
	var err error
	var count int
	n.EachNeighbour(val, func(v PbVal, po int) bool {
		for _, f := range fs {
			err = f(v, po)
			if err != nil {
				return err.Error() == errNoCount.Error()
			}
		}
		count++
		if count == expCount {
			return false
		}
		return true
	})
	if err == nil && count < expCount {
		return fmt.Errorf("not enough neighbours returned, expected %v, got %v", expCount, count)
	}
	return err
}

const (
	maxEachNeighbourTests = 500
	maxEachNeighbour      = 4
	keylen                = 4
)

func randomTestAddr(n int, i int) *testAddr {
	v := RandomAddress().Bin()[:n]
	return NewTestAddr(v, i)
}

func TestPbTreeMerge(t *testing.T) {
	for i := 0; i < maxEachNeighbourTests; i++ {
		max0 := rand.Intn(maxEachNeighbour) + 1
		max1 := rand.Intn(maxEachNeighbour) + 1
		n0 := NewPbTree(nil, 0)
		n1 := NewPbTree(nil, 0)
		m := make(map[string]bool)
		for j := 0; j < max0; {
			v := randomTestAddr(keylen, j)
			_, found := n0.Add(v)
			if !found {
				glog.V(4).Infof("%v: add %v", j, v)
				m[v.String()] = false
				j++
			}
		}
		expAdded := 0

		for j := 0; j < max1; {
			v := randomTestAddr(keylen, j)
			_, found := n1.Add(v)
			glog.V(4).Infof("%v: add %v", j, v)
			if !found {
				j++
			}
			_, found = m[v.String()]
			if !found {
				expAdded++
				glog.V(4).Infof("%v: newly added %v", j-1, v)
				m[v.String()] = false
			}
		}
		expSize := len(m)

		glog.V(4).Infof("%v-0: pin: %v, size: %v", i, n0.Pin(), max0)
		glog.V(4).Infof("%v-1: pin: %v, size: %v", i, n1.Pin(), max1)
		glog.V(4).Infof("%v: %v", i, expSize)
		added := n0.Merge(n1)
		size := n0.Size()
		if expSize != size {
			t.Fatalf("incorrect number of elements in merged pbTree, expected %v, got %v\n%v", expSize, size, n0)
		}
		if expAdded != added {
			t.Fatalf("incorrect number of added elements in merged pbTree, expected %v, got %v", expAdded, added)
		}
		for k, _ := range m {
			_, found := n0.Add(NewTestAddr(k, 0))
			if !found {
				t.Fatalf("merged pbTree missing element %v", k)
			}
		}
	}
}

func TestPbTreeEachNeighbourSync(t *testing.T) {
	for i := 0; i < maxEachNeighbourTests; i++ {
		max := rand.Intn(maxEachNeighbour/2) + maxEachNeighbour/2
		pin := randomTestAddr(keylen, 0)
		n := NewPbTree(pin, 0)
		m := make(map[string]bool)
		m[pin.String()] = false
		for j := 1; j <= max; j++ {
			v := randomTestAddr(keylen, j)
			n.Add(v)
			m[v.String()] = false
		}

		size := n.Size()
		if size < 2 {
			continue
		}
		count := rand.Intn(size/2) + size/2
		val := randomTestAddr(keylen, max+1)
		glog.V(4).Infof("%v: pin: %v, size: %v, val: %v, count: %v", i, n.Pin(), size, val, count)
		err := testPbTreeEachNeighbour(n, val, count, checkPo(val), checkOrder(val), checkValues(m, val))
		if err != nil {
			t.Fatal(err)
		}
		minPoFound := keylen
		maxPoNotFound := 0
		for k, found := range m {
			po, _ := val.Prefix(NewTestAddr(k, 0), 0)
			if found {
				if po < minPoFound {
					minPoFound = po
				}
			} else {
				if po > maxPoNotFound {
					maxPoNotFound = po
				}
			}
		}
		if minPoFound < maxPoNotFound {
			t.Fatalf("incorrect neighbours returned: found one with PO %v < there was one not found with PO %v", minPoFound, maxPoNotFound)
		}
	}
}

func TestPbTreeEachNeighbourAsync(t *testing.T) {
	for i := 0; i < maxEachNeighbourTests; i++ {
		max := rand.Intn(maxEachNeighbour/2) + maxEachNeighbour/2
		n := NewPbTree(randomTestAddr(keylen, 0), 0)
		var size int = 1
		for j := 1; j <= max; j++ {
			v := randomTestAddr(keylen, j)
			_, found := n.Add(v)
			if !found {
				size++
			}
		}
		if size != n.Size() {
			t.Fatal(n)
		}
		if size < 2 {
			continue
		}
		count := rand.Intn(size/2) + size/2
		val := randomTestAddr(keylen, max+1)

		mu := sync.Mutex{}
		m := make(map[string]bool)
		maxPos := rand.Intn(keylen)
		glog.V(5).Infof("%v: pin: %v, size: %v, val: %v, count: %v, maxPos: %v", i, n.Pin(), size, val, count, maxPos)
		msize := 0
		remember := func(v PbVal, po int) error {
			// mu.Lock()
			// defer mu.Unlock()
			if po > maxPos {
				// glog.V(4).Infof("NOT ADD %v", v)
				return errNoCount
			}
			// glog.V(4).Infof("ADD %v, %v", v, msize)
			m[v.String()] = true
			msize++
			return nil
		}
		if i == 0 {
			continue
		}
		err := testPbTreeEachNeighbour(n, val, count, remember)
		if err != nil {
			glog.V(6).Info(err)
		}
		d := 0
		forget := func(v PbVal, po int) {
			mu.Lock()
			defer mu.Unlock()
			d++
			// glog.V(4).Infof("DEL %v", v)
			delete(m, v.String())
		}

		n.EachNeighbourAsync(val, count, maxPos, forget, true)
		if d != msize {
			t.Fatalf("incorrect number of neighbour calls in async iterator. expected %v, got %v", msize, d)
		}
		if len(m) != 0 {

			t.Fatalf("incorrect neighbour calls in async iterator. %v items missed:\n%v", len(m), n)
		}
	}
}

func benchmarkEachNeighbourSync(t *testing.B, max, count int, d time.Duration) {
	t.ReportAllocs()
	pin := randomTestAddr(keylen, 0)
	n := NewPbTree(pin, 0)
	for j := 1; j <= max; {
		v := randomTestAddr(keylen, j)
		_, found := n.Add(v)
		if !found {
			j++
		}
	}
	t.ResetTimer()
	for i := 0; i < t.N; i++ {
		val := randomTestAddr(keylen, max+1)
		m := 0
		n.EachNeighbour(val, func(v PbVal, po int) bool {
			time.Sleep(d)
			m++
			if m == count {
				return false
			}
			return true
		})
	}
	t.StopTimer()
	stats := new(runtime.MemStats)
	runtime.ReadMemStats(stats)
	// fmt.Println(stats.Sys)
}

func benchmarkEachNeighbourAsync(t *testing.B, max, count int, d time.Duration) {
	t.ReportAllocs()
	pin := randomTestAddr(keylen, 0)
	n := NewPbTree(pin, 0)
	for j := 1; j <= max; {
		v := randomTestAddr(keylen, j)
		_, found := n.Add(v)
		if !found {
			j++
		}
	}
	t.ResetTimer()
	for i := 0; i < t.N; i++ {
		val := randomTestAddr(keylen, max+1)
		n.EachNeighbourAsync(val, count, keylen, func(v PbVal, po int) {
			time.Sleep(d)
		}, true)
	}
	t.StopTimer()
	stats := new(runtime.MemStats)
	runtime.ReadMemStats(stats)
	// fmt.Println(stats.Sys)
}

func BenchmarkEachNeighbourSync_3_1_0(t *testing.B) {
	benchmarkEachNeighbourSync(t, 1000, 10, 1*time.Microsecond)
}
func BenchmarkEachNeighboursAsync_3_1_0(t *testing.B) {
	benchmarkEachNeighbourAsync(t, 1000, 10, 1*time.Microsecond)
}
func BenchmarkEachNeighbourSync_3_2_0(t *testing.B) {
	benchmarkEachNeighbourSync(t, 1000, 100, 1*time.Microsecond)
}
func BenchmarkEachNeighboursAsync_3_2_0(t *testing.B) {
	benchmarkEachNeighbourAsync(t, 1000, 100, 1*time.Microsecond)
}
func BenchmarkEachNeighbourSync_3_3_0(t *testing.B) {
	benchmarkEachNeighbourSync(t, 1000, 1000, 1*time.Microsecond)
}
func BenchmarkEachNeighboursAsync_3_3_0(t *testing.B) {
	benchmarkEachNeighbourAsync(t, 1000, 1000, 1*time.Microsecond)
}

func BenchmarkEachNeighbourSync_3_1_1(t *testing.B) {
	benchmarkEachNeighbourSync(t, 1000, 10, 2*time.Microsecond)
}
func BenchmarkEachNeighboursAsync_3_1_1(t *testing.B) {
	benchmarkEachNeighbourAsync(t, 1000, 10, 2*time.Microsecond)
}
func BenchmarkEachNeighbourSync_3_2_1(t *testing.B) {
	benchmarkEachNeighbourSync(t, 1000, 100, 2*time.Microsecond)
}
func BenchmarkEachNeighboursAsync_3_2_1(t *testing.B) {
	benchmarkEachNeighbourAsync(t, 1000, 100, 2*time.Microsecond)
}
func BenchmarkEachNeighbourSync_3_3_1(t *testing.B) {
	benchmarkEachNeighbourSync(t, 1000, 1000, 2*time.Microsecond)
}
func BenchmarkEachNeighboursAsync_3_3_1(t *testing.B) {
	benchmarkEachNeighbourAsync(t, 1000, 1000, 2*time.Microsecond)
}

func BenchmarkEachNeighbourSync_3_1_2(t *testing.B) {
	benchmarkEachNeighbourSync(t, 1000, 10, 4*time.Microsecond)
}
func BenchmarkEachNeighboursAsync_3_1_2(t *testing.B) {
	benchmarkEachNeighbourAsync(t, 1000, 10, 4*time.Microsecond)
}
func BenchmarkEachNeighbourSync_3_2_2(t *testing.B) {
	benchmarkEachNeighbourSync(t, 1000, 100, 4*time.Microsecond)
}
func BenchmarkEachNeighboursAsync_3_2_2(t *testing.B) {
	benchmarkEachNeighbourAsync(t, 1000, 100, 4*time.Microsecond)
}
func BenchmarkEachNeighbourSync_3_3_2(t *testing.B) {
	benchmarkEachNeighbourSync(t, 1000, 1000, 4*time.Microsecond)
}
func BenchmarkEachNeighboursAsync_3_3_2(t *testing.B) {
	benchmarkEachNeighbourAsync(t, 1000, 1000, 4*time.Microsecond)
}

func BenchmarkEachNeighbourSync_3_1_3(t *testing.B) {
	benchmarkEachNeighbourSync(t, 1000, 10, 8*time.Microsecond)
}
func BenchmarkEachNeighboursAsync_3_1_3(t *testing.B) {
	benchmarkEachNeighbourAsync(t, 1000, 10, 8*time.Microsecond)
}
func BenchmarkEachNeighbourSync_3_2_3(t *testing.B) {
	benchmarkEachNeighbourSync(t, 1000, 100, 8*time.Microsecond)
}
func BenchmarkEachNeighboursAsync_3_2_3(t *testing.B) {
	benchmarkEachNeighbourAsync(t, 1000, 100, 8*time.Microsecond)
}
func BenchmarkEachNeighbourSync_3_3_3(t *testing.B) {
	benchmarkEachNeighbourSync(t, 1000, 1000, 8*time.Microsecond)
}
func BenchmarkEachNeighboursAsync_3_3_3(t *testing.B) {
	benchmarkEachNeighbourAsync(t, 1000, 1000, 8*time.Microsecond)
}

func BenchmarkEachNeighbourSync_3_1_4(t *testing.B) {
	benchmarkEachNeighbourSync(t, 1000, 10, 16*time.Microsecond)
}
func BenchmarkEachNeighboursAsync_3_1_4(t *testing.B) {
	benchmarkEachNeighbourAsync(t, 1000, 10, 16*time.Microsecond)
}
func BenchmarkEachNeighbourSync_3_2_4(t *testing.B) {
	benchmarkEachNeighbourSync(t, 1000, 100, 16*time.Microsecond)
}
func BenchmarkEachNeighboursAsync_3_2_4(t *testing.B) {
	benchmarkEachNeighbourAsync(t, 1000, 100, 16*time.Microsecond)
}
func BenchmarkEachNeighbourSync_3_3_4(t *testing.B) {
	benchmarkEachNeighbourSync(t, 1000, 1000, 16*time.Microsecond)
}
func BenchmarkEachNeighboursAsync_3_3_4(t *testing.B) {
	benchmarkEachNeighbourAsync(t, 1000, 1000, 16*time.Microsecond)
}
