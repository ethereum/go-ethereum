package iblt

// Invertible Bloom Lookup Table from
// What’s the Difference?
// Efficient Set Reconciliation without Prior Context
// David Eppstein1 Michael T. Goodrich1 Frank Uyeda2 George Varghese
// https://www.ics.uci.edu/~eppstein/pubs/EppGooUye-SIGCOMM-11.pdf
// IBFL with N cells (N>=50) and K=4 can safely recover diffs of size at least N/2.
// For large N the space overhead is less than 1.3 (so we can decode diffs of size of n * 0.77).

import (
	"bytes"
	"encoding/binary"
	"encoding/gob"
	"errors"
	"log"

	"github.com/dchest/siphash"
)

// Filter is totally NOT thread-safe!
type Filter struct {
	mask      uint64
	keySums   []uint64
	valueSums []*bts
	counts    []int
	seen      bitset
	buf       []byte
	shift     uint16
	idx       []int
}

func (f *Filter) Clone() *Filter {
	r := &Filter{
		mask:      f.mask,
		keySums:   make([]uint64, f.N()),
		valueSums: make([]*bts, 0, f.N()),
		counts:    make([]int, f.N()),
		seen:      make(bitset, len(f.seen)),
		shift:     f.shift,
		idx:       make([]int, len(f.idx)),
	}

	copy(r.keySums, f.keySums)
	for _, v := range f.keySums {
		r.keySums = append(r.keySums, v)
	}
	for _, v := range f.valueSums {
		b := make([]byte, len(v.b))
		copy(b, v.b)
		r.valueSums = append(r.valueSums, &bts{b})
	}
	copy(r.counts, f.counts)
	return r
}

type serializableFilter struct {
	KeySums   []uint64
	ValueSums [][]byte
	Counts    []int
	K         int
}

func (f Filter) MarshalBinary() (data []byte, err error) {
	s := serializableFilter{
		KeySums:   f.keySums,
		Counts:    f.counts,
		ValueSums: make([][]byte, len(f.valueSums)),
		K:         len(f.idx),
	}
	for k, v := range f.valueSums {
		s.ValueSums[k] = v.b
	}
	buf := &bytes.Buffer{}
	err = gob.NewEncoder(buf).Encode(s)
	if err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func (f *Filter) UnmarshalBinary(data []byte) error {
	var s serializableFilter
	err := gob.NewDecoder(bytes.NewReader(data)).Decode(&s)
	if err != nil {
		return err
	}
	*f = *New(s.K, len(s.Counts))
	f.keySums = s.KeySums
	f.counts = s.Counts
	for k, v := range s.ValueSums {
		f.valueSums[k].b = v
	}
	return nil
}

// New constructs a new Filter.
func New(k, l int) *Filter {
	if k >= 10 || k < 1 {
		panic("k should be between 1 and 10")
	}
	if l/k < 2 {
		panic("l should be at least 2*k")
	}
	if l&(l-1) != 0 {
		panic("l should be a power of two")
	}
	var shift uint16
	for ll := l; ll != 0; ll >>= 1 {
		shift++
	}
	values := make([]*bts, l)
	for i := range values {
		values[i] = &bts{}
	}
	return &Filter{
		mask:      uint64(l - 1),
		keySums:   make([]uint64, l),
		valueSums: values,
		counts:    make([]int, l),
		seen:      newBitSet(l),
		shift:     shift,
		idx:       make([]int, k),
	}
}

func (f Filter) K() int {
	return len(f.idx)
}

func (f Filter) N() int {
	return len(f.counts)
}

func xor(a, b []byte) []byte {
	if len(b) > len(a) {
		return _xor(b, a)
	}
	return _xor(a, b)
}

func (f *Filter) getIdxHash(b []byte) ([]int, uint64) {
	hash := hash(b)
	return f.getIdx(hash), hash
}

func (f *Filter) getIdx(hash uint64) []int {
	f.seen.ClearAll()
	v := hash
	bits := uint16(64)
	for k := range f.idx {
		for {
			if bits < f.shift {
				v = xorShiftStarRound(&hash)
				bits = 64
			}
			pos := int(v & f.mask)
			v >>= f.shift
			bits -= f.shift
			if f.seen.Test(pos) {
				continue
			}
			f.seen.Set(pos)
			f.idx[k] = pos
			break
		}
	}
	return f.idx
}

func (f *Filter) Add(b []byte) {
	idx, hash := f.getIdxHash(b)
	enc := f.encode(b)
	for _, k := range idx {
		f.counts[k] = f.counts[k] + 1
		f.keySums[k] = f.keySums[k] ^ hash
		f.valueSums[k].xorInPlace(enc)
	}
}

func (f *Filter) Remove(b []byte) {
	idx, hash := f.getIdxHash(b)
	enc := f.encode(b)
	for _, k := range idx {
		f.counts[k] = f.counts[k] - 1
		f.keySums[k] = f.keySums[k] ^ hash
		f.valueSums[k].xorInPlace(enc)
	}
	// TODO: consider cleaning up pure cells:
	// the value could get long if a very long values were inserted and then removed.
}

type Diff struct {
	Added   [][]byte
	Removed [][]byte
}

func (f Filter) Diff(other Filter, diff *Filter) error {
	if f.shift != other.shift || f.shift != diff.shift {
		return errors.New("sizes should match")
	}
	if len(f.idx) != len(other.idx) || len(f.idx) != len(diff.idx) {
		return errors.New("ks should match")
	}
	for k := range f.counts {
		diff.counts[k] = f.counts[k] - other.counts[k]
		diff.keySums[k] = f.keySums[k] ^ other.keySums[k]
		diff.valueSums[k] = &bts{xor(f.valueSums[k].b, other.valueSums[k].b)}
	}
	return nil
}

// Inplace
func (f *Filter) Sub(other Filter) error {
	if f.shift != other.shift {
		return errors.New("sizes should match")
	}
	if len(f.idx) != len(other.idx) {
		return errors.New("ks should match")
	}
	for k := range f.counts {
		f.counts[k] = f.counts[k] - other.counts[k]
		f.keySums[k] = f.keySums[k] ^ other.keySums[k]
		f.valueSums[k].xorInPlace(other.valueSums[k].b)
	}
	return nil

}

// Decode is distructive!
// One can apply the diff produced (even in case of error) to restore the previous state.
func (f *Filter) Decode() (*Diff, error) {
	pure := make([]int, len(f.counts))
	numZ := 0
	diff := &Diff{}
	lp := 0
	for k := range f.counts {
		c := f.counts[k]
		switch c {
		case 0:
			if f.keySums[k] == 0 {
				numZ++
			}
		case -1, 1:
			_, err := f.decode(f.valueSums[k].b)
			if err != nil {
				continue
			}
			pure[lp] = k
			lp++
		}
	}
	head := 0
	tail := lp - 1
	for lp > 0 {
		// Deque pop
		lp--
		pos := pure[head]
		c := f.counts[pos]
		head++
		if head == len(f.counts) {
			head = 0
		}
		if c != 1 && c != -1 {
			continue
		}
		dec, err := f.decode(f.valueSums[pos].b)
		if err != nil {
			continue
		}
		h := hash(dec)
		if h == 0 || h != f.keySums[pos] {
			continue
		}
		if c == 1 {
			diff.Added = append(diff.Added, dec)
		} else {
			diff.Removed = append(diff.Removed, dec)
		}
		idx := f.getIdx(h)
		val := f.valueSums[pos].b
		numZ++
		for _, k := range idx {
			if k != pos {
				if f.keySums[k] == 0 && f.counts[k] == 0 {
					continue
				}
				f.keySums[k] = f.keySums[k] ^ h
				f.valueSums[k].xorInPlace(val)
				f.counts[k] = f.counts[k] - c
				c := f.counts[k]
				switch c {
				case 0:
					if f.keySums[k] == 0 {
						numZ++
					}
				case -1, 1:
					_, err := f.decode(f.valueSums[k].b)
					if err != nil {
						continue
					}

					// Deque push
					lp++
					tail++
					if tail == len(f.counts) {
						tail = 0
					}
					pure[tail] = k
				}
			}
		}
		f.keySums[pos] = 0
		f.counts[pos] = 0
	}
	var err error
	if numZ != len(f.counts) {
		err = errors.New("failed to decode")
	}
	return diff, err
}

func _xor(a, b []byte) []byte {
	// len(a) >= len(b)
	// TODO: use append to reduce allocations?
	r := make([]byte, len(a))
	for i, v := range b {
		r[i] = v ^ a[i]
	}
	copy(r[len(b):], a[len(b):])
	return r
}

type bts struct {
	b []byte
}

func (b *bts) xorInPlace(a []byte) {
	if len(b.b) < len(a) {
		b.b = _xor(a, b.b)
		return
	}
	for i, v := range a {
		b.b[i] = b.b[i] ^ v
	}
}

// Self-delimited encoding.
func (f *Filter) encode(b []byte) []byte {
	if len(b) >= 1<<16 {
		log.Panicln("len(b) is too large", len(b))
	}
	l := len(b) + 2
	if len(f.buf) < l {
		f.buf = make([]byte, l)
	}
	binary.LittleEndian.PutUint16(f.buf, uint16(len(b)))
	copy(f.buf[2:], b)
	return f.buf[:l]
}

func (f *Filter) decode(b []byte) ([]byte, error) {
	if len(b) == 0 {
		return nil, nil
	}
	if len(b) < 2 {
		return nil, errors.New("bad length")
	}
	l16 := binary.LittleEndian.Uint16(b)
	l := int(l16)
	if l+2 > len(b) {
		return nil, errors.New("too short")
	}
	return b[2 : l+2], nil
}

type bitset []uint64

func (b bitset) Set(i int) {
	b[i>>6] |= 1 << (uint32(i) & 63)
}

func (b bitset) Clear(i int) {
	b[i>>6] &= ^(1 << (uint32(i) & 63))
}

func (b bitset) Test(i int) bool {
	return (b[i>>6] & (1 << (uint32(i) & 63))) != 0
}

func (b bitset) ClearAll() {
	for i := range b {
		b[i] = 0
	}
}

func newBitSet(l int) bitset {
	return make([]uint64, (l+63)>>6)
}

func hash(v []byte) uint64 {
	return siphash.Hash(2, 57, v)
}

func xorShiftStarRound(x *uint64) uint64 {
	if *x == 0 {
		*x = 1
	}
	*x ^= (*x >> 12)
	*x ^= (*x << 25)
	*x ^= (*x >> 27)
	return *x * 2685821657736338717
}
