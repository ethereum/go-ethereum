package trie

import (
	"bytes"
	"testing"

	"github.com/scroll-tech/go-ethereum/common"
)

// grep from `feat/snap`
func reverseBitInPlace(b []byte) {
	var v [8]uint8
	for i := 0; i < len(b); i++ {
		for j := 0; j < 8; j++ {
			v[j] = (b[i] >> j) & 1
		}
		var tmp uint8 = 0
		for j := 0; j < 8; j++ {
			tmp |= v[8-j-1] << j
		}
		b[i] = tmp
	}
}

func reverseBytesInPlace(b []byte) {
	for i, j := 0, len(b)-1; i < j; i, j = i+1, j-1 {
		b[i], b[j] = b[j], b[i]
	}
}

func TestBitReverse(t *testing.T) {

	for _, testBytes := range [][]byte{
		common.FromHex("7b908cce3bc16abb3eac5dff6c136856526f15225f74ce860a2bec47912a5492"),
		common.FromHex("fac65cd2ad5e301083d0310dd701b5faaff1364cbe01cdbfaf4ec3609bb4149e"),
		common.FromHex("55791f6ec2f83fee512a2d3d4b505784fdefaea89974e10440d01d62a18a298a"),
		common.FromHex("5ab775b64d86a8058bb71c3c765d0f2158c14bbeb9cb32a65eda793a7e95e30f"),
		common.FromHex("ccb464abf67804538908c62431b3a6788e8dc6dee62aff9bfe6b10136acfceac"),
		common.FromHex("b908adff17a5aa9d6787324c39014a74b04cef7fba6a92aeb730f48da1ca665d"),
	} {

		b1 := bitReverse(testBytes)
		reverseBitInPlace(testBytes)
		reverseBytesInPlace(testBytes)
		if !bytes.Equal(b1, testBytes) {
			t.Errorf("unexpected bit reversed %x vs %x", b1, testBytes)
		}
	}

}

func TestBitDoubleReverse(t *testing.T) {

	for _, testBytes := range [][]byte{
		common.FromHex("7b908cce3bc16abb3eac5dff6c136856526f15225f74ce860a2bec47912a5492"),
		common.FromHex("fac65cd2ad5e301083d0310dd701b5faaff1364cbe01cdbfaf4ec3609bb4149e"),
		common.FromHex("55791f6ec2f83fee512a2d3d4b505784fdefaea89974e10440d01d62a18a298a"),
		common.FromHex("5ab775b64d86a8058bb71c3c765d0f2158c14bbeb9cb32a65eda793a7e95e30f"),
		common.FromHex("ccb464abf67804538908c62431b3a6788e8dc6dee62aff9bfe6b10136acfceac"),
		common.FromHex("b908adff17a5aa9d6787324c39014a74b04cef7fba6a92aeb730f48da1ca665d"),
	} {

		b := bitReverse(bitReverse(testBytes))
		if !bytes.Equal(b, testBytes) {
			t.Errorf("unexpected double bit reversed %x vs %x", b, testBytes)
		}
	}

}
