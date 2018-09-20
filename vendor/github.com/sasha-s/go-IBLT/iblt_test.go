package iblt

import (
	"bytes"
	"encoding/gob"
	"fmt"
	"math/rand"
	"sort"
	"testing"

	"github.com/davecgh/go-spew/spew"
)

func TestIBLTSub(t *testing.T) {
	var tcs = []struct {
		a, b, add, remove []string
	}{
		{
			a:   []string{"z"},
			add: []string{"z"},
		},
		{
			a: []string{"z"},
			b: []string{"z"},
		},
		{
			a:      []string{"alpha", "beta", "gamma", "delta", "ε", "zeta", "η"},
			b:      []string{"α", "beta", "gamma", "δ", "epsilon", "zeta", "η"},
			add:    []string{"alpha", "delta", "ε"},
			remove: []string{"epsilon", "α", "δ"},
		},
	}
	for _, l := range []int{16, 32, 1024} {
		for _, tc := range tcs {
			f := New(3, l)
			f2 := New(3, l)
			for _, s := range tc.a {
				f.Add([]byte(s))
			}
			for _, s := range tc.b {
				f2.Add([]byte(s))
			}
			for i := 0; i < 10000; i++ {
				if i < 20 || rand.Intn(100) == 0 {
					buf := &bytes.Buffer{}
					err := gob.NewEncoder(buf).Encode(f)
					if err != nil {
						t.Error(err)
						continue
					}
					var i2 Filter
					err = gob.NewDecoder(buf).Decode(&i2)
					if err != nil {
						t.Error(err)
						continue
					}
					if spew.Sdump(f.keySums) != spew.Sdump(i2.keySums) {
						t.Error("decoded Filter is different, oops. keysums")
						spew.Dump(f.keySums, ">>>>", i2.keySums)
					}
					if spew.Sdump(f.counts) != spew.Sdump(i2.counts) {
						t.Error("decoded Filter is different, oops. counts")
						spew.Dump(f.counts, ">>>>", i2.counts)
					}
					if spew.Sprint(f.valueSums) != spew.Sprint(i2.valueSums) {
						t.Error("decoded Filter is different, oops. valueSums")
						spew.Println("", f.valueSums, "\n>>>>\n", i2.valueSums)
					}
					if f.K() != i2.K() {
						t.Errorf("k: expected %d, got %d", f.K(), i2.K())
					}
					if f.N() != i2.N() {
						t.Errorf("k: expected %d, got %d", f.N(), i2.N())
					}

				}
				v := []byte(fmt.Sprint(i))
				if rand.Intn(3) == 0 {
					f.Add(v)
					f2.Add(v)
				} else {
					f.Remove(v)
					f2.Remove(v)
				}
			}
			if err := f.Sub(*f2); err != nil {
				t.Error(spew.Sdump(tc), err)
				continue
			}
			r, err := f2.Decode()
			if err == nil {
				t.Error(spew.Sdump(tc), err, spew.Sdump(f2, r))
				continue
			}
			r, err = f.Decode()
			if err != nil {
				t.Error(spew.Sdump(tc), err, spew.Sdump(f))
				continue
			}

			sort.Strings(tc.add)
			sort.Strings(tc.remove)
			added := []string{}
			for _, s := range r.Added {
				added = append(added, string(s))
			}
			removed := []string{}
			for _, s := range r.Removed {
				removed = append(removed, string(s))
			}
			sort.Strings(added)
			sort.Strings(removed)
			if fmt.Sprint(tc.add) != fmt.Sprint(added) {
				t.Error(spew.Sdump(tc), "|got", added, "|expected", tc.add)
			}
			if fmt.Sprint(tc.remove) != fmt.Sprint(removed) {
				t.Error(spew.Sdump(tc), "|", removed, "|", tc.remove)
			}
		}
	}
}

func TestIBLT(t *testing.T) {
	var tcs = []struct {
		add, remove []string
	}{
		{
			add: []string{"z"},
		},
		{
			remove: []string{"z"},
		},
		{
			add:    []string{"alpha", "beta", "gamma", "delta"},
			remove: []string{"omega", "z", "p", "q"},
		},
	}
	for _, k := range []int{3, 5, 7} {
		for _, l := range []int{16, 32, 1024} {
			if k != 3 && l < 1024 {
				continue
			}
			for _, tc := range tcs {
				f := New(k, l)
				for _, s := range tc.add {
					f.Add([]byte(s))
				}
				for _, s := range tc.remove {
					f.Remove([]byte(s))
				}
				r, err := f.Decode()
				if err != nil {
					t.Error(spew.Sdump(tc), err)
				}
				sort.Strings(tc.add)
				sort.Strings(tc.remove)
				added := []string{}
				for _, s := range r.Added {
					added = append(added, string(s))
				}
				removed := []string{}
				for _, s := range r.Removed {
					removed = append(removed, string(s))
				}
				sort.Strings(added)
				sort.Strings(removed)
				if fmt.Sprint(tc.add) != fmt.Sprint(added) {
					t.Error(spew.Sdump(tc), "|", added, "|", tc.add)
				}
				if fmt.Sprint(tc.remove) != fmt.Sprint(removed) {
					t.Error(spew.Sdump(tc), "|", removed, "|", tc.remove)
				}
			}
		}
	}
}

func TestBitset(t *testing.T) {
	a := 1
	for l := 1; l < 15000; l += a {
		a += rand.Intn(100)
		fmt.Print(".")
		b := newBitSet(l)
		b2 := map[int]bool{}
		check := func() {
			for pos := 0; pos < l; pos++ {
				if b2[pos] != b.Test(pos) {
					t.Fatal(pos, b2[pos], b.Test(pos))
				}
			}
		}
		for k := 0; k < 10000; k++ {
			pos := rand.Intn(l)
			if rand.Intn(2) == 0 {
				b.Clear(pos)
				delete(b2, pos)
			} else {
				b.Set(pos)
				b2[pos] = true
			}
			if rand.Intn(50) == 0 {
				check()
			}
		}
		check()
		b.ClearAll()
		for pos := 0; pos < l; pos++ {
			if b.Test(pos) {
				t.Fatal(pos, b.Test(pos))
			}
		}
	}
}

func TestXOR(t *testing.T) {
	tcs := []struct {
		a, b, expected []byte
	}{
		{[]byte{0}, nil, []byte{0}},
		{nil, []byte{0}, []byte{0}},
		{[]byte{0xfa}, []byte{0xff}, []byte{5}},
		{[]byte{0xfa, 0xff}, []byte{0xff}, []byte{5, 0xff}},
		{[]byte{0xfa, 0xff}, []byte{0xff, 0xff, 1}, []byte{5, 0, 1}},
	}
	for _, tc := range tcs {
		actual := xor(tc.a, tc.b)
		if fmt.Sprint(actual) != fmt.Sprint(tc.expected) {
			t.Errorf("`%v` ^ `%v`: expected `%v`, got `%v`\n", tc.a, tc.b, tc.expected, actual)
		}
	}
	for _, tc := range tcs {
		clone := func(x []byte) []byte {
			y := make([]byte, len(x))
			copy(y, x)
			return y
		}
		a := bts{clone(tc.a)}
		a.xorInPlace(tc.b)
		if fmt.Sprint(a.b) != fmt.Sprint(tc.expected) {
			t.Errorf("`%v` ^ `%v`: expected `%v`, got `%v`\n", tc.a, tc.b, tc.expected, tc.a)
		}
	}
}

func random(n int) []byte {
	b := make([]byte, n)
	for i := range b {
		b[i] = byte(rand.Intn(256))
	}
	return b
}

func TestEncodeDecode(t *testing.T) {
	var f Filter
	for i := 0; i < 10000; i++ {
		l := rand.Intn(30)
		if i == 0 {
			l = 1<<16 - 1
		}
		b := random(l)
		encoded := f.encode(b)
		slack := rand.Intn(10)
		e2 := append(encoded, random(slack)...)
		decoded, _ := f.decode(e2)
		if fmt.Sprint(decoded) != fmt.Sprint(b) {
			t.Errorf("a := `%v`, a.endode() == `%v`, with slack: `%v`. a.endode().decode() ==`%v`\n", b, encoded, e2, decoded)
		}
	}
	defer func() {
		if recover() == nil {
			t.Error("expected Panic")
		}
	}()
	f.encode(random(1 << 16))
}
