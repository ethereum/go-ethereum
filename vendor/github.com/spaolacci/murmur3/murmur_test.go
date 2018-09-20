package murmur3

import (
	"fmt"
	"strconv"
	"testing"
)

var data = []struct {
	seed  uint32
	h32   uint32
	h64_1 uint64
	h64_2 uint64
	s     string
}{
	{0x00, 0x00000000, 0x0000000000000000, 0x0000000000000000, ""},
	{0x00, 0x248bfa47, 0xcbd8a7b341bd9b02, 0x5b1e906a48ae1d19, "hello"},
	{0x00, 0x149bbb7f, 0x342fac623a5ebc8e, 0x4cdcbc079642414d, "hello, world"},
	{0x00, 0xe31e8a70, 0xb89e5988b737affc, 0x664fc2950231b2cb, "19 Jan 2038 at 3:14:07 AM"},
	{0x00, 0xd5c48bfc, 0xcd99481f9ee902c9, 0x695da1a38987b6e7, "The quick brown fox jumps over the lazy dog."},

	{0x01, 0x514e28b7, 0x4610abe56eff5cb5, 0x51622daa78f83583, ""},
	{0x01, 0xbb4abcad, 0xa78ddff5adae8d10, 0x128900ef20900135, "hello"},
	{0x01, 0x6f5cb2e9, 0x8b95f808840725c6, 0x1597ed5422bd493b, "hello, world"},
	{0x01, 0xf50e1f30, 0x2a929de9c8f97b2f, 0x56a41d99af43a2db, "19 Jan 2038 at 3:14:07 AM"},
	{0x01, 0x846f6a36, 0xfb3325171f9744da, 0xaaf8b92a5f722952, "The quick brown fox jumps over the lazy dog."},

	{0x2a, 0x087fcd5c, 0xf02aa77dfa1b8523, 0xd1016610da11cbb9, ""},
	{0x2a, 0xe2dbd2e1, 0xc4b8b3c960af6f08, 0x2334b875b0efbc7a, "hello"},
	{0x2a, 0x7ec7c6c2, 0xb91864d797caa956, 0xd5d139a55afe6150, "hello, world"},
	{0x2a, 0x58f745f6, 0xfd8f19ebdc8c6b6a, 0xd30fdc310fa08ff9, "19 Jan 2038 at 3:14:07 AM"},
	{0x2a, 0xc02d1434, 0x74f33c659cda5af7, 0x4ec7a891caf316f0, "The quick brown fox jumps over the lazy dog."},
}

func TestRefStrings(t *testing.T) {
	for _, elem := range data {

		h32 := New32WithSeed(elem.seed)
		h32.Write([]byte(elem.s))
		if v := h32.Sum32(); v != elem.h32 {
			t.Errorf("[Hash32] key: '%s', seed: '%d': 0x%x (want 0x%x)", elem.s, elem.seed, v, elem.h32)
		}

		h32.Reset()
		h32.Write([]byte(elem.s))
		target := fmt.Sprintf("%08x", elem.h32)
		if p := fmt.Sprintf("%x", h32.Sum(nil)); p != target {
			t.Errorf("[Hash32] key: '%s', seed: '%d': %s (want %s)", elem.s, elem.seed, p, target)
		}

		if v := Sum32WithSeed([]byte(elem.s), elem.seed); v != elem.h32 {
			t.Errorf("[Hash32] key '%s', seed: '%d': 0x%x (want 0x%x)", elem.s, elem.seed, v, elem.h32)
		}

		h64 := New64WithSeed(elem.seed)
		h64.Write([]byte(elem.s))
		if v := h64.Sum64(); v != elem.h64_1 {
			t.Errorf("'[Hash64] key: '%s', seed: '%d': 0x%x (want 0x%x)", elem.s, elem.seed, v, elem.h64_1)
		}

		h64.Reset()
		h64.Write([]byte(elem.s))
		target = fmt.Sprintf("%016x", elem.h64_1)
		if p := fmt.Sprintf("%x", h64.Sum(nil)); p != target {
			t.Errorf("[Hash64] key: '%s', seed: '%d': %s (want %s)", elem.s, elem.seed, p, target)
		}

		if v := Sum64WithSeed([]byte(elem.s), elem.seed); v != elem.h64_1 {
			t.Errorf("[Hash64] key: '%s', seed: '%d': 0x%x (want 0x%x)", elem.s, elem.seed, v, elem.h64_1)
		}

		h128 := New128WithSeed(elem.seed)

		h128.Write([]byte(elem.s))
		if v1, v2 := h128.Sum128(); v1 != elem.h64_1 || v2 != elem.h64_2 {
			t.Errorf("[Hash128] key: '%s', seed: '%d': 0x%x-0x%x (want 0x%x-0x%x)", elem.s, elem.seed, v1, v2, elem.h64_1, elem.h64_2)
		}

		h128.Reset()
		h128.Write([]byte(elem.s))
		target = fmt.Sprintf("%016x%016x", elem.h64_1, elem.h64_2)
		if p := fmt.Sprintf("%x", h128.Sum(nil)); p != target {
			t.Errorf("[Hash128] key: '%s', seed: '%d': %s (want %s)", elem.s, elem.seed, p, target)
		}

		if v1, v2 := Sum128WithSeed([]byte(elem.s), elem.seed); v1 != elem.h64_1 || v2 != elem.h64_2 {
			t.Errorf("[Hash128] key: '%s', seed: '%d': 0x%x-0x%x (want 0x%x-0x%x)", elem.s, elem.seed, v1, v2, elem.h64_1, elem.h64_2)
		}
	}
}

func TestIncremental(t *testing.T) {
	for _, elem := range data {
		h32 := New32WithSeed(elem.seed)
		h128 := New128WithSeed(elem.seed)
		var i, j int
		for k := len(elem.s); i < k; i = j {
			j = 2*i + 3
			if j > k {
				j = k
			}
			s := elem.s[i:j]
			print(s + "|")
			h32.Write([]byte(s))
			h128.Write([]byte(s))
		}
		println()
		if v := h32.Sum32(); v != elem.h32 {
			t.Errorf("[Hash32] key: '%s', seed: '%d': 0x%x (want 0x%x)", elem.s, elem.seed, v, elem.h32)
		}
		if v1, v2 := h128.Sum128(); v1 != elem.h64_1 || v2 != elem.h64_2 {
			t.Errorf("[Hash128] key: '%s', seed: '%d': 0x%x-0x%x (want 0x%x-0x%x)", elem.s, elem.seed, v1, v2, elem.h64_1, elem.h64_2)
		}
	}
}

func Benchmark32(b *testing.B) {
	buf := make([]byte, 8192)
	for length := 1; length <= cap(buf); length *= 2 {
		b.Run(strconv.Itoa(length), func(b *testing.B) {
			buf = buf[:length]
			b.SetBytes(int64(length))
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				Sum32(buf)
			}
		})
	}
}

func BenchmarkPartial32(b *testing.B) {
	buf := make([]byte, 128)
	for length := 8; length <= cap(buf); length *= 2 {
		b.Run(strconv.Itoa(length), func(b *testing.B) {
			buf = buf[:length]
			b.SetBytes(int64(length))

			start := (32 / 8) / 2
			chunks := 7
			k := length / chunks
			tail := (length - start) % k

			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				hasher := New32()
				hasher.Write(buf[0:start])

				for j := start; j+k <= length; j += k {
					hasher.Write(buf[j : j+k])
				}

				hasher.Write(buf[length-tail:])
				hasher.Sum32()
			}
		})
	}
}

func Benchmark64(b *testing.B) {
	buf := make([]byte, 8192)
	for length := 1; length <= cap(buf); length *= 2 {
		b.Run(strconv.Itoa(length), func(b *testing.B) {
			buf = buf[:length]
			b.SetBytes(int64(length))
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				Sum64(buf)
			}
		})
	}
}

func Benchmark128(b *testing.B) {
	buf := make([]byte, 8192)
	for length := 1; length <= cap(buf); length *= 2 {
		b.Run(strconv.Itoa(length), func(b *testing.B) {
			buf = buf[:length]
			b.SetBytes(int64(length))
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				Sum128(buf)
			}
		})
	}
}
