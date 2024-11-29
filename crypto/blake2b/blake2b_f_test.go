package blake2b

import (
	"encoding/binary"
	"fmt"
	"reflect"
	"testing"
)

func TestF(t *testing.T) {
	for i, test := range testVectorsF {
		t.Run(fmt.Sprintf("test vector %v", i), func(t *testing.T) {
			//toEthereumTestCase(test)

			h := test.hIn
			F(&h, test.m, test.c, test.f, test.rounds)

			if !reflect.DeepEqual(test.hOut, h) {
				t.Errorf("Unexpected result\nExpected: [%#x]\nActual:   [%#x]\n", test.hOut, h)
			}
		})
	}
}

type testVector struct {
	hIn    [8]uint64
	m      [16]uint64
	c      [2]uint64
	f      bool
	rounds uint32
	hOut   [8]uint64
}

// https://tools.ietf.org/html/rfc7693#appendix-A
var testVectorsF = []testVector{
	{
		hIn: [8]uint64{
			0x6a09e667f2bdc948, 0xbb67ae8584caa73b,
			0x3c6ef372fe94f82b, 0xa54ff53a5f1d36f1,
			0x510e527fade682d1, 0x9b05688c2b3e6c1f,
			0x1f83d9abfb41bd6b, 0x5be0cd19137e2179,
		},
		m: [16]uint64{
			0x0000000000636261, 0x0000000000000000, 0x0000000000000000,
			0x0000000000000000, 0x0000000000000000, 0x0000000000000000,
			0x0000000000000000, 0x0000000000000000, 0x0000000000000000,
			0x0000000000000000, 0x0000000000000000, 0x0000000000000000,
			0x0000000000000000, 0x0000000000000000, 0x0000000000000000,
			0x0000000000000000,
		},
		c:      [2]uint64{3, 0},
		f:      true,
		rounds: 12,
		hOut: [8]uint64{
			0x0D4D1C983FA580BA, 0xE9F6129FB697276A, 0xB7C45A68142F214C,
			0xD1A2FFDB6FBB124B, 0x2D79AB2A39C5877D, 0x95CC3345DED552C2,
			0x5A92F1DBA88AD318, 0x239900D4ED8623B9,
		},
	},
}

func Fuzz(f *testing.F) {
	f.Fuzz(func(t *testing.T, data []byte) {
		fuzz(data)
	})
}

func fuzz(data []byte) {
	// Make sure the data confirms to the input model
	if len(data) != 211 {
		return
	}
	// Parse everything and call all the implementations
	var (
		rounds = binary.BigEndian.Uint16(data[0:2])

		h [8]uint64
		m [16]uint64
		t [2]uint64
		f uint64
	)

	for i := 0; i < 8; i++ {
		offset := 2 + i*8
		h[i] = binary.LittleEndian.Uint64(data[offset : offset+8])
	}
	for i := 0; i < 16; i++ {
		offset := 66 + i*8
		m[i] = binary.LittleEndian.Uint64(data[offset : offset+8])
	}
	t[0] = binary.LittleEndian.Uint64(data[194:202])
	t[1] = binary.LittleEndian.Uint64(data[202:210])

	if data[210]%2 == 1 { // Avoid spinning the fuzzer to hit 0/1
		f = 0xFFFFFFFFFFFFFFFF
	}

	// Run the blake2b compression on all instruction sets and cross reference
	want := h
	fGeneric(&want, &m, t[0], t[1], f, uint64(rounds))

	have := h
	fSSE4(&have, &m, t[0], t[1], f, uint64(rounds))
	if have != want {
		panic("SSE4 mismatches generic algo")
	}
	have = h
	fAVX(&have, &m, t[0], t[1], f, uint64(rounds))
	if have != want {
		panic("AVX mismatches generic algo")
	}
	have = h
	fAVX2(&have, &m, t[0], t[1], f, uint64(rounds))
	if have != want {
		panic("AVX2 mismatches generic algo")
	}
}
