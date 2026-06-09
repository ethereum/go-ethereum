// Ported verbatim from github.com/QuarkChain/goquarkchain/serialize (byte-compatible).

package serialize

import (
	"bytes"
	"errors"
	"fmt"
	"math/big"
	"testing"
)

type serializableStruct struct {
	val uint64
	err error
}

func (e *serializableStruct) Serialize(w *[]byte) error {
	if e.err != nil {
		return e.err
	} else {
		*w = append(*w, new(big.Int).SetUint64(e.val).Bytes()...)
	}
	return nil
}

func (e *serializableStruct) Deserialize(bb *ByteBuffer) error {
	if e == nil {

	} else if bb.Remaining() == 4 {
		e = nil
	} else {
		e.val = 1
	}

	return nil
}

type structForTest struct {
	From         *[]byte
	To           *[]byte `json:"to"                 ser:"nil"`
	IgnoredField int     `json:"ignore"             ser:"-"`
	privateField int     //private field will be Ignored
}

func newStructForTest(from, to *[]byte) structForTest {
	s := structForTest{from, to, 0, 0}
	return s
}

type testDataForSerialize struct {
	val           interface{}
	output, error string
}

func newUint128(val uint64) *Uint128 {
	ui := new(Uint128)
	ui.Value = new(big.Int).SetUint64(val)
	return ui
}

func newUint256(val uint64) *Uint256 {
	ui := new(Uint256)
	ui.Value = new(big.Int).SetUint64(val)
	return ui
}

type LargeBytes struct {
	Value []byte `bytesizeofslicelen:"4"`
}

var serdata = []testDataForSerialize{
	// booleans
	{val: true, output: "01"},
	{val: false, output: "00"},

	// integers
	{val: uint8(0), output: "00"},
	{val: uint8(128), output: "80"},
	{val: uint16(0), output: "0000"},
	{val: uint16(128), output: "0080"},
	{val: uint32(0), output: "00000000"},
	{val: uint32(128), output: "00000080"},
	{val: uint32(1024), output: "00000400"},
	{val: uint32(0xFFFFFFFF), output: "FFFFFFFF"},
	{val: uint64(0), output: "0000000000000000"},
	{val: uint64(128), output: "0000000000000080"},
	{val: uint64(0xFFFFFFFF), output: "00000000FFFFFFFF"},
	{val: uint64(0xFFFFFFFFFFFFFFFF), output: "FFFFFFFFFFFFFFFF"},
	{val: uint(0), output: "00"},
	{val: uint(128), output: "0180"},
	{val: uint(1024), output: "020400"},
	{val: uint(0xFFFFFFFF), output: "04FFFFFFFF"},

	{val: newUint128(0), output: "00000000000000000000000000000000"},
	{val: newUint128(128), output: "00000000000000000000000000000080"},
	{val: newUint128(0xFFFFFFFFFFFFFFFF), output: "0000000000000000FFFFFFFFFFFFFFFF"},
	{val: newUint256(0), output: "0000000000000000000000000000000000000000000000000000000000000000"},
	{val: newUint256(128), output: "0000000000000000000000000000000000000000000000000000000000000080"},
	{val: newUint256(0xFFFFFFFFFFFFFFFF), output: "000000000000000000000000000000000000000000000000FFFFFFFFFFFFFFFF"},

	// big integers (should match uint for small values)
	{val: big.NewInt(0), output: "00"},
	{val: big.NewInt(1), output: "0101"},
	{val: big.NewInt(128), output: "0180"},
	{val: big.NewInt(256), output: "020100"},
	{val: big.NewInt(1024), output: "020400"},
	{val: big.NewInt(0xFFFFFF), output: "03FFFFFF"},
	{val: big.NewInt(0xFFFFFFFFFFFFFF), output: "07FFFFFFFFFFFFFF"},
	{
		val:    big.NewInt(0).SetBytes(unhex("102030405060708090A0B0C0D0E0F2")),
		output: "0F102030405060708090A0B0C0D0E0F2",
	},
	{
		val:    big.NewInt(0).SetBytes(unhex("0100020003000400050006000700080009000A000B000C000D000E01")),
		output: "1C0100020003000400050006000700080009000A000B000C000D000E01",
	},

	// non-pointer big.Int
	{val: *big.NewInt(0), output: "00"},
	{val: *big.NewInt(0xFFFFFF), output: "03FFFFFF"},

	// negative ints are not supported
	{val: big.NewInt(-1), error: "ser: cannot serialize negative *big.Int"},

	// byte slices, strings
	{val: []byte{}, output: "00"},
	{val: []byte{0x7E}, output: "017E"},
	{val: []byte{0x80}, output: "0180"},
	{val: []byte{1, 2, 3}, output: "03010203"},

	{val: &LargeBytes{}, output: "00000000"},
	{val: &LargeBytes{[]byte{0x7E}}, output: "000000017E"},
	{val: &LargeBytes{[]byte{0x80}}, output: "0000000180"},
	{val: &LargeBytes{[]byte{1, 2, 3}}, output: "00000003010203"},

	{val: "", output: "00000000"},
	{val: "\x7E", output: "000000017E"},
	{val: "\x80", output: "0000000180"},
	{val: "dog", output: "00000003646F67"},

	// slices
	{val: []uint8{}, output: "00"},
	{val: []uint8{1, 2, 3}, output: "03010203"},
	{val: []uint32{}, output: "00"},
	{val: []uint32{1, 2, 3}, output: "03000000010000000200000003"},

	//Array
	{val: [3]uint8{1, 2, 3}, output: "010203"},
	{val: [3]uint32{1, 2, 3}, output: "000000010000000200000003"},

	// structs
	{val: newStructForTest(&[]byte{1, 2, 3}, nil), output: "0301020300"},
	{val: newStructForTest(&[]byte{1, 2, 3}, &[]byte{4, 5, 6}), output: "030102030103040506"},
	{val: newStructForTest(nil, &[]byte{4, 5, 6}), output: "000103040506"},

	// nil
	// as nilOk default value is false, serialize will use default
	// value to serialize instead of nil
	{val: (*uint)(nil), output: "00"},
	{val: (*string)(nil), output: "00000000"},
	{val: (*[]byte)(nil), output: "00"},
	{val: (*[10]byte)(nil), output: "00000000000000000000"},
	{val: (*big.Int)(nil), output: "00"},
	{val: (*[]string)(nil), output: "00"},
	{val: (*[10]string)(nil), output: "00000000000000000000000000000000000000000000000000000000000000000000000000000000"},
	{val: (*[]struct{ uint })(nil), output: "00"},

	// interfaces
	// Serializer
	{val: (*serializableStruct)(nil), output: ""},
	{val: &serializableStruct{val: 0xFFFF}, output: "FFFF"},
	{val: &serializableStruct{1, errors.New("test error")}, error: "test error"},

	// int is not support
	{val: int(0), error: "type int is not serializable"},
	{val: (*interface{})(nil), error: "type interface {} is not serializable"},
}

func runEncTests(t *testing.T, f func(val interface{}) ([]byte, error)) {
	for i, test := range serdata {
		output, err := f(test.val)
		if err != nil && test.error == "" {
			t.Errorf("test %d: unexpected error: %v\nvalue %#v\ntype %T",
				i, err, test.val, test.val)
			continue
		}
		if test.error != "" && fmt.Sprint(err) != test.error {
			t.Errorf("test %d: error mismatch\ngot   %v\nwant  %v\nvalue %#v\ntype  %T",
				i, err, test.error, test.val, test.val)
			continue
		}
		if err == nil && !bytes.Equal(output, unhex(test.output)) {
			t.Errorf("test %d: output mismatch:\ngot   %X\nwant  %s\nvalue %#v\ntype  %T",
				i, output, test.output, test.val, test.val)
		}
	}
}

func TestSerialize(t *testing.T) {
	runEncTests(t, func(val interface{}) ([]byte, error) {
		b := make([]byte, 0, 1)
		err := Serialize(&b, val)
		return b, err
	})
}

func TestSerializeToBytes(t *testing.T) {
	runEncTests(t, SerializeToBytes)
}
