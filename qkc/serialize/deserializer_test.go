// Ported verbatim from github.com/QuarkChain/goquarkchain/serialize (byte-compatible).

package serialize

import (
	"encoding/hex"
	"fmt"
	"math/big"
	"reflect"
	"strings"
	"testing"
)

type testDataForDeserialize struct {
	input string
	ptr   interface{}
	value interface{}
	error string
}

type simplestruct struct {
	A uint
	B string
}

var (
	veryBigInt = big.NewInt(0).Add(
		big.NewInt(0).Lsh(big.NewInt(0xFFFFFFFFFFFFFF), 16),
		big.NewInt(0xFFFF),
	)
)

type hasIgnoredField struct {
	A uint
	B uint `ser:"-"`
	C uint32
}

var deserdata = []testDataForDeserialize{
	// booleans
	{input: "01", ptr: new(bool), value: true},
	{input: "00", ptr: new(bool), value: false},
	{input: "02", ptr: new(bool), error: "deser: invalid boolean value: 2"},

	// integers

	{input: "00", ptr: new(uint8), value: uint8(0)},
	{input: "05", ptr: new(uint8), value: uint8(5)},
	{input: "0000", ptr: new(uint16), value: uint16(0)},
	{input: "0005", ptr: new(uint16), value: uint16(5)},
	{input: "05", ptr: new(uint16), error: "deser: buffer is shorter than expected"},
	{input: "00000000", ptr: new(uint32), value: uint32(0)},
	{input: "00000505", ptr: new(uint32), value: uint32(0x0505)},
	{input: "05050505", ptr: new(uint32), value: uint32(0x05050505)},
	{input: "0000000000000000", ptr: new(uint64), value: uint64(0)},
	{input: "0000000000000505", ptr: new(uint64), value: uint64(0x0505)},
	{input: "0505050505050505", ptr: new(uint64), value: uint64(0x0505050505050505)},
	{input: "0100", ptr: new(uint), value: uint(0)},
	{input: "020505", ptr: new(uint), value: uint(0x0505)},
	{input: "080505050505050505", ptr: new(uint), value: uint(0x0505050505050505)},

	{input: "00000000000000000000000000000001", ptr: new(Uint128), value: *newUint128(1)},
	{input: "00000000000000000000000000000080", ptr: new(Uint128), value: *newUint128(128)},
	{input: "0000000000000000FFFFFFFFFFFFFFFF", ptr: new(Uint128), value: *newUint128(0xFFFFFFFFFFFFFFFF)},
	{input: "05", ptr: new(Uint128), error: "deser: buffer is shorter than expected"},
	{input: "0000000000000000000000000000000000000000000000000000000000000001", ptr: new(Uint256), value: *newUint256(1)},
	{input: "0000000000000000000000000000000000000000000000000000000000000080", ptr: new(Uint256), value: *newUint256(128)},
	{input: "000000000000000000000000000000000000000000000000FFFFFFFFFFFFFFFF", ptr: new(Uint256), value: *newUint256(0xFFFFFFFFFFFFFFFF)},
	{input: "05", ptr: new(Uint256), error: "deser: buffer is shorter than expected"},

	//uint slices
	{input: "00", ptr: new([]uint), value: []uint{}},
	{input: "080102030405060708", ptr: new([]uint8), value: []uint8{1, 2, 3, 4, 5, 6, 7, 8}},
	{input: "080000000100000002000000030000000400000005000000060000000700000008", ptr: new([]uint32), value: []uint32{1, 2, 3, 4, 5, 6, 7, 8}},
	{input: "050102", ptr: new([]uint8), error: "deser: buffer is shorter than expected"},

	// arrays
	{input: "0102030405", ptr: new([5]uint8), value: [5]uint8{1, 2, 3, 4, 5}},
	{input: "0000000100000002000000030000000400000005", ptr: new([5]uint32), value: [5]uint32{1, 2, 3, 4, 5}},
	{input: "", ptr: new([0]uint), value: [0]uint{}}, // zero sized arrays
	{input: "0102", ptr: new([5]uint8), error: "deser: buffer is shorter than expected"},

	// byte slices
	{input: "0101", ptr: new([]byte), value: []byte{1}},
	{input: "00", ptr: new([]byte), value: []byte{}},
	{input: "0D6162636465666768696A6B6C6D", ptr: new([]byte), value: []byte("abcdefghijklm")},
	{input: "0D0102", ptr: new([]byte), error: "deser: buffer is shorter than expected"},

	// byte slices, strings
	{input: "00", ptr: new([]byte), value: []byte{}},
	{input: "017E", ptr: new([]byte), value: []byte{0x7E}},
	{input: "0180", ptr: new([]byte), value: []byte{0x80}},
	{input: "03010203", ptr: new([]byte), value: []byte{1, 2, 3}},
	{input: "04010203", ptr: new([]byte), error: "deser: buffer is shorter than expected"},

	//SerializableList interface
	{input: "00000000", ptr: new(LargeBytes), value: LargeBytes{[]byte{}}},
	{input: "000000017E", ptr: new(LargeBytes), value: LargeBytes{[]byte{0x7E}}},
	{input: "0000000180", ptr: new(LargeBytes), value: LargeBytes{[]byte{0x80}}},
	{input: "00000003010203", ptr: new(LargeBytes), value: LargeBytes{[]byte{1, 2, 3}}},
	{input: "03010203", ptr: new(LargeBytes), error: "deser: buffer is shorter than expected for serialize.LargeBytes.Value"},

	// byte arrays
	{input: "00", ptr: new([0]byte), value: [0]byte{}},
	{input: "02", ptr: new([1]byte), value: [1]byte{2}},
	{input: "80", ptr: new([1]byte), value: [1]byte{128}},
	{input: "0102030405", ptr: new([5]byte), value: [5]byte{1, 2, 3, 4, 5}},

	// strings
	{input: "0000000100", ptr: new(string), value: "\000"},
	{input: "0000000D6162636465666768696A6B6C6D", ptr: new(string), value: "abcdefghijklm"},
	{input: "0D6162636465666768696A6B6C6D", ptr: new(string), error: "deser: buffer is shorter than expected"},

	// big ints
	{input: "0101", ptr: new(*big.Int), value: big.NewInt(1)},
	{input: "09FFFFFFFFFFFFFFFFFF", ptr: new(*big.Int), value: veryBigInt},
	{input: "0110", ptr: new(big.Int), value: *big.NewInt(16)}, // non-pointer also works
	{input: "0210", ptr: new(big.Int), error: "deser: buffer is shorter than expected"},

	// structs
	{input: "0301020300", ptr: new(structForTest), value: newStructForTest(&[]byte{1, 2, 3}, nil)},
	{input: "030102030103040506", ptr: new(structForTest), value: newStructForTest(&[]byte{1, 2, 3}, &[]byte{4, 5, 6})},
	// To present (canonical marker 01) but its slice buffer is truncated. (The
	// presence marker must be exactly 0 or 1; non-canonical markers like 03 are
	// rejected — covered by TestDeserializeNilMarkerCanonical.)
	{input: "0301020301040506", ptr: new(structForTest), error: "deser: buffer is shorter than expected for serialize.structForTest.To"},

	// structs
	{
		input: "010500000003343434",
		ptr:   new(simplestruct),
		value: simplestruct{5, "444"},
	},

	// struct tag "-"
	{
		input: "010100000002",
		ptr:   new(hasIgnoredField),
		value: hasIgnoredField{A: 1, C: 2},
	},

	// pointers
	{input: "0100", ptr: new(*[]byte), value: &[]byte{0}},
	{input: "0100", ptr: new(*uint), value: uintp(0)},
	{input: "0107", ptr: new(*uint), value: uintp(7)},
	{input: "0180", ptr: new(*uint), value: uintp(0x80)},
	{input: "010109", ptr: new(*[]uint), value: &[]uint{9}},
	{input: "010403030303", ptr: new(*[][]byte), value: &[][]byte{{3, 3, 3, 3}}},

	// do not support interface{}, need to know the real type
	{input: "02", ptr: new(int), error: "type int is not serializable"},
	{input: "00", ptr: new(interface{}), error: "type interface {} is not serializable"},
}

func uintp(i uint) *uint { return &i }

func runTests(t *testing.T, deserialize func([]byte, interface{}) error) {
	for i, test := range deserdata {
		input, err := hex.DecodeString(test.input)
		if err != nil {
			t.Errorf("test %d: invalid hex input %q", i, test.input)
			continue
		}
		err = deserialize(input, test.ptr)
		if err != nil && test.error == "" {
			t.Errorf("test %d: unexpected Deserialize error: %v\ndecoding into %T\ninput %q",
				i, err, test.ptr, test.input)
			continue
		}
		if test.error != "" && fmt.Sprint(err) != test.error {
			t.Errorf("test %d: Deserialize error mismatch\ngot  %v\nwant %v\ndecoding into %T\ninput %q",
				i, err, test.error, test.ptr, test.input)
			continue
		}
		deref := reflect.ValueOf(test.ptr).Elem().Interface()
		if err == nil && !reflect.DeepEqual(deref, test.value) {
			t.Errorf("test %d: value mismatch\ngot  %#v\nwant %#v\ndecoding into %T\ninput %q",
				i, deref, test.value, test.ptr, test.input)
		}
	}
}

func TestDeserialize(t *testing.T) {
	runTests(t, func(input []byte, into interface{}) error {
		return Deserialize(NewByteBuffer(input), into)
	})
}

func ExampleDeserialize() {
	input, _ := hex.DecodeString("010a0000001400000006666F6F626172")

	type example struct {
		A       uint
		B       uint32
		private uint // private fields are Ignored
		String  string
	}

	var s example
	err := Deserialize(NewByteBuffer(input), &s)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
	} else {
		fmt.Printf("Deserialized value: %#v\n", s)
	}
	// Output:
	// Deserialized value: serialize.example{A:0xa, B:0x14, private:0x0, String:"foobar"}
}

func ExampleDeserialize_structTagNilAndIgnore() {
	// In this example, we'll use the "nil" struct tag to change
	// how a pointer-typed field is deserialized. The input contains an RLP
	// list of one element, an empty string.
	input := []byte{0x00, 0x01, 0x03, 0x04, 0x05, 0x06}

	s := new(structForTest)
	Deserialize(NewByteBuffer(input), &s)
	fmt.Printf("From = %v\n", *s.From)
	fmt.Printf("To = %v\n", *s.To)

	// Output:
	// From = []
	// To = [4 5 6]
}

func ExampleDeserialize_structTagNil() {
	// In this example, we'll use the "nil" struct tag to change
	// how a pointer-typed field is deserialized. The input contains an RLP
	// list of one element, an empty string.
	input := []byte{0x00}

	// This type uses the normal rules.
	// The empty input string is deserialized as a pointer to an empty Go string.
	var normalRules struct {
		String *[]byte
	}
	Deserialize(NewByteBuffer(input), &normalRules)
	fmt.Printf("normal: String = %v\n", *normalRules.String)

	// This type uses the struct tag.
	// The empty input string is deserialized as a nil pointer.
	var withEmptyOK struct {
		String *[]byte `ser:"nil"`
	}
	Deserialize(NewByteBuffer(input), &withEmptyOK)
	fmt.Printf("with nil tag: String = %v\n", withEmptyOK.String)

	// Output:
	// normal: String = []
	// with nil tag: String = <nil>
}

func BenchmarkDeserialize(b *testing.B) {
	enc := encodeTestSlice(90000)
	b.SetBytes(int64(len(enc)))
	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		var s []uint
		bb := NewByteBuffer(enc)
		if err := Deserialize(bb, &s); err != nil {
			b.Fatalf("Deserialize error: %v", err)
		}
	}
}

func BenchmarkDeserializeIntSliceReuse(b *testing.B) {
	enc := encodeTestSlice(100000)
	b.SetBytes(int64(len(enc)))
	b.ReportAllocs()
	b.ResetTimer()

	var s []uint
	for i := 0; i < b.N; i++ {
		bb := NewByteBuffer(enc)
		if err := Deserialize(bb, &s); err != nil {
			b.Fatalf("Deserialize error: %v", err)
		}
	}
}

func encodeTestSlice(n uint) []byte {
	s := make([]uint, n)
	for i := uint(0); i < n; i++ {
		s[i] = i
	}
	b, err := SerializeToBytes(s)
	if err != nil {
		panic(fmt.Sprintf("encode error: %v", err))
	}
	return b
}

func unhex(str string) []byte {
	b, err := hex.DecodeString(strings.Replace(str, " ", "", -1))
	if err != nil {
		panic(fmt.Sprintf("invalid hex string: %q", str))
	}
	return b
}
