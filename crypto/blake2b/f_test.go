package blake2b

import (
	"encoding/binary"
	"encoding/hex"
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
				t.Errorf("Unexpected result\nExpected: [%v]\nActual:   [%v]\n", test.hOut, h)
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

// toEthereumTestCase transforms F test vector into test vector format used by
// go-ethereum precompiles
func toEthereumTestCase(vector testVector) {
	var memory [213]byte

	// 4 bytes for rounds
	binary.BigEndian.PutUint32(memory[0:4], uint32(vector.rounds))

	// for h (512 bits = 64 bytes)
	for i := 0; i < 8; i++ {
		offset := 4 + i*8
		binary.LittleEndian.PutUint64(memory[offset:offset+8], vector.hIn[i])

	}

	// for m (1024 bits = 128 bytes)
	for i := 0; i < 16; i++ {
		offset := 68 + i*8
		binary.LittleEndian.PutUint64(memory[offset:offset+8], vector.m[i])
	}

	// 8 bytes for t[0], 8 bytes for t[1]
	binary.LittleEndian.PutUint64(memory[196:204], vector.c[0])
	binary.LittleEndian.PutUint64(memory[204:212], vector.c[1])

	// 1 byte for f
	if vector.f {
		memory[212] = 1
	}

	fmt.Printf("input: \"%v\"\n", hex.EncodeToString(memory[:]))

	var result [64]byte

	binary.LittleEndian.PutUint64(result[0:8], vector.hOut[0])
	binary.LittleEndian.PutUint64(result[8:16], vector.hOut[1])
	binary.LittleEndian.PutUint64(result[16:24], vector.hOut[2])
	binary.LittleEndian.PutUint64(result[24:32], vector.hOut[3])

	binary.LittleEndian.PutUint64(result[32:40], vector.hOut[4])
	binary.LittleEndian.PutUint64(result[40:48], vector.hOut[5])
	binary.LittleEndian.PutUint64(result[48:56], vector.hOut[6])
	binary.LittleEndian.PutUint64(result[56:64], vector.hOut[7])

	fmt.Printf("expected: \"%v\"\n", hex.EncodeToString(result[:]))
}
