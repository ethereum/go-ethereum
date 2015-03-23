package rlp

import (
	"bytes"
	"encoding/hex"
	"fmt"
	"io"
	"math/big"
	"reflect"
	"testing"
)

func TestStreamKind(t *testing.T) {
	tests := []struct {
		input    string
		wantKind Kind
		wantLen  uint64
	}{
		{"00", Byte, 0},
		{"01", Byte, 0},
		{"7F", Byte, 0},
		{"80", String, 0},
		{"B7", String, 55},
		{"B800", String, 0},
		{"B90400", String, 1024},
		{"BA000400", String, 1024},
		{"BB00000400", String, 1024},
		{"BFFFFFFFFFFFFFFFFF", String, ^uint64(0)},
		{"C0", List, 0},
		{"C8", List, 8},
		{"F7", List, 55},
		{"F800", List, 0},
		{"F804", List, 4},
		{"F90400", List, 1024},
		{"FFFFFFFFFFFFFFFFFF", List, ^uint64(0)},
	}

	for i, test := range tests {
		// using plainReader to inhibit input limit errors.
		s := NewStream(newPlainReader(unhex(test.input)), 0)
		kind, len, err := s.Kind()
		if err != nil {
			t.Errorf("test %d: Kind returned error: %v", i, err)
			continue
		}
		if kind != test.wantKind {
			t.Errorf("test %d: kind mismatch: got %d, want %d", i, kind, test.wantKind)
		}
		if len != test.wantLen {
			t.Errorf("test %d: len mismatch: got %d, want %d", i, len, test.wantLen)
		}
	}
}

func TestNewListStream(t *testing.T) {
	ls := NewListStream(bytes.NewReader(unhex("0101010101")), 3)
	if k, size, err := ls.Kind(); k != List || size != 3 || err != nil {
		t.Errorf("Kind() returned (%v, %d, %v), expected (List, 3, nil)", k, size, err)
	}
	if size, err := ls.List(); size != 3 || err != nil {
		t.Errorf("List() returned (%d, %v), expected (3, nil)", size, err)
	}
	for i := 0; i < 3; i++ {
		if val, err := ls.Uint(); val != 1 || err != nil {
			t.Errorf("Uint() returned (%d, %v), expected (1, nil)", val, err)
		}
	}
	if err := ls.ListEnd(); err != nil {
		t.Errorf("ListEnd() returned %v, expected (3, nil)", err)
	}
}

func TestStreamErrors(t *testing.T) {
	withoutInputLimit := func(b []byte) *Stream {
		return NewStream(newPlainReader(b), 0)
	}
	withCustomInputLimit := func(limit uint64) func([]byte) *Stream {
		return func(b []byte) *Stream {
			return NewStream(bytes.NewReader(b), limit)
		}
	}

	type calls []string
	tests := []struct {
		string
		calls
		newStream func([]byte) *Stream // uses bytes.Reader if nil
		error
	}{
		{"C0", calls{"Bytes"}, nil, ErrExpectedString},
		{"C0", calls{"Uint"}, nil, ErrExpectedString},
		{"89000000000000000001", calls{"Uint"}, nil, errUintOverflow},
		{"00", calls{"List"}, nil, ErrExpectedList},
		{"80", calls{"List"}, nil, ErrExpectedList},
		{"C0", calls{"List", "Uint"}, nil, EOL},
		{"C8C9010101010101010101", calls{"List", "Kind"}, nil, ErrElemTooLarge},
		{"C3C2010201", calls{"List", "List", "Uint", "Uint", "ListEnd", "Uint"}, nil, EOL},
		{"00", calls{"ListEnd"}, nil, errNotInList},
		{"C401020304", calls{"List", "Uint", "ListEnd"}, nil, errNotAtEOL},

		// Expected EOF
		{"", calls{"Kind"}, nil, io.EOF},
		{"", calls{"Uint"}, nil, io.EOF},
		{"", calls{"List"}, nil, io.EOF},
		{"8105", calls{"Uint", "Uint"}, nil, io.EOF},
		{"C0", calls{"List", "ListEnd", "List"}, nil, io.EOF},

		// Input limit errors.
		{"81", calls{"Bytes"}, nil, ErrValueTooLarge},
		{"81", calls{"Uint"}, nil, ErrValueTooLarge},
		{"81", calls{"Raw"}, nil, ErrValueTooLarge},
		{"BFFFFFFFFFFFFFFFFFFF", calls{"Bytes"}, nil, ErrValueTooLarge},
		{"C801", calls{"List"}, nil, ErrValueTooLarge},

		// Test for input limit overflow. Since we are counting the limit
		// down toward zero in Stream.remaining, reading too far can overflow
		// remaining to a large value, effectively disabling the limit.
		{"C40102030401", calls{"Raw", "Uint"}, withCustomInputLimit(5), io.EOF},
		{"C4010203048102", calls{"Raw", "Uint"}, withCustomInputLimit(6), ErrValueTooLarge},

		// Check that the same calls are fine without a limit.
		{"C40102030401", calls{"Raw", "Uint"}, withoutInputLimit, nil},
		{"C4010203048102", calls{"Raw", "Uint"}, withoutInputLimit, nil},

		// Unexpected EOF. This only happens when there is
		// no input limit, so the reader needs to be 'dumbed down'.
		{"81", calls{"Bytes"}, withoutInputLimit, io.ErrUnexpectedEOF},
		{"81", calls{"Uint"}, withoutInputLimit, io.ErrUnexpectedEOF},
		{"BFFFFFFFFFFFFFFF", calls{"Bytes"}, withoutInputLimit, io.ErrUnexpectedEOF},
		{"C801", calls{"List", "Uint", "Uint"}, withoutInputLimit, io.ErrUnexpectedEOF},

		// This test verifies that the input position is advanced
		// correctly when calling Bytes for empty strings. Kind can be called
		// any number of times in between and doesn't advance.
		{"C3808080", calls{
			"List",  // enter the list
			"Bytes", // past first element

			"Kind", "Kind", "Kind", // this shouldn't advance

			"Bytes", // past second element

			"Kind", "Kind", // can't hurt to try

			"Bytes", // past final element
			"Bytes", // this one should fail
		}, nil, EOL},
	}

testfor:
	for i, test := range tests {
		if test.newStream == nil {
			test.newStream = func(b []byte) *Stream { return NewStream(bytes.NewReader(b), 0) }
		}
		s := test.newStream(unhex(test.string))
		rs := reflect.ValueOf(s)
		for j, call := range test.calls {
			fval := rs.MethodByName(call)
			ret := fval.Call(nil)
			err := "<nil>"
			if lastret := ret[len(ret)-1].Interface(); lastret != nil {
				err = lastret.(error).Error()
			}
			if j == len(test.calls)-1 {
				want := "<nil>"
				if test.error != nil {
					want = test.error.Error()
				}
				if err != want {
					t.Errorf("test %d: last call (%s) error mismatch\ngot:  %s\nwant: %s",
						i, call, err, test.error)
				}
			} else if err != "<nil>" {
				t.Errorf("test %d: call %d (%s) unexpected error: %q", i, j, call, err)
				continue testfor
			}
		}
	}
}

func TestStreamList(t *testing.T) {
	s := NewStream(bytes.NewReader(unhex("C80102030405060708")), 0)

	len, err := s.List()
	if err != nil {
		t.Fatalf("List error: %v", err)
	}
	if len != 8 {
		t.Fatalf("List returned invalid length, got %d, want 8", len)
	}

	for i := uint64(1); i <= 8; i++ {
		v, err := s.Uint()
		if err != nil {
			t.Fatalf("Uint error: %v", err)
		}
		if i != v {
			t.Errorf("Uint returned wrong value, got %d, want %d", v, i)
		}
	}

	if _, err := s.Uint(); err != EOL {
		t.Errorf("Uint error mismatch, got %v, want %v", err, EOL)
	}
	if err = s.ListEnd(); err != nil {
		t.Fatalf("ListEnd error: %v", err)
	}
}

func TestStreamRaw(t *testing.T) {
	s := NewStream(bytes.NewReader(unhex("C58401010101")), 0)
	s.List()

	want := unhex("8401010101")
	raw, err := s.Raw()
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(want, raw) {
		t.Errorf("raw mismatch: got %x, want %x", raw, want)
	}
}

func TestDecodeErrors(t *testing.T) {
	r := bytes.NewReader(nil)

	if err := Decode(r, nil); err != errDecodeIntoNil {
		t.Errorf("Decode(r, nil) error mismatch, got %q, want %q", err, errDecodeIntoNil)
	}

	var nilptr *struct{}
	if err := Decode(r, nilptr); err != errDecodeIntoNil {
		t.Errorf("Decode(r, nilptr) error mismatch, got %q, want %q", err, errDecodeIntoNil)
	}

	if err := Decode(r, struct{}{}); err != errNoPointer {
		t.Errorf("Decode(r, struct{}{}) error mismatch, got %q, want %q", err, errNoPointer)
	}

	expectErr := "rlp: type chan bool is not RLP-serializable"
	if err := Decode(r, new(chan bool)); err == nil || err.Error() != expectErr {
		t.Errorf("Decode(r, new(chan bool)) error mismatch, got %q, want %q", err, expectErr)
	}

	if err := Decode(r, new(uint)); err != io.EOF {
		t.Errorf("Decode(r, new(int)) error mismatch, got %q, want %q", err, io.EOF)
	}
}

type decodeTest struct {
	input string
	ptr   interface{}
	value interface{}
	error string
}

type simplestruct struct {
	A uint
	B string
}

type recstruct struct {
	I     uint
	Child *recstruct
}

var (
	veryBigInt = big.NewInt(0).Add(
		big.NewInt(0).Lsh(big.NewInt(0xFFFFFFFFFFFFFF), 16),
		big.NewInt(0xFFFF),
	)
)

var (
	sharedByteArray [5]byte
	sharedPtr       = new(*uint)
)

var decodeTests = []decodeTest{
	// integers
	{input: "05", ptr: new(uint32), value: uint32(5)},
	{input: "80", ptr: new(uint32), value: uint32(0)},
	{input: "8105", ptr: new(uint32), value: uint32(5)},
	{input: "820505", ptr: new(uint32), value: uint32(0x0505)},
	{input: "83050505", ptr: new(uint32), value: uint32(0x050505)},
	{input: "8405050505", ptr: new(uint32), value: uint32(0x05050505)},
	{input: "850505050505", ptr: new(uint32), error: "rlp: input string too long for uint32"},
	{input: "C0", ptr: new(uint32), error: "rlp: expected input string or byte for uint32"},

	// slices
	{input: "C0", ptr: new([]uint), value: []uint{}},
	{input: "C80102030405060708", ptr: new([]uint), value: []uint{1, 2, 3, 4, 5, 6, 7, 8}},

	// arrays
	{input: "C0", ptr: new([5]uint), value: [5]uint{}},
	{input: "C50102030405", ptr: new([5]uint), value: [5]uint{1, 2, 3, 4, 5}},
	{input: "C6010203040506", ptr: new([5]uint), error: "rlp: input list has too many elements for [5]uint"},

	// byte slices
	{input: "01", ptr: new([]byte), value: []byte{1}},
	{input: "80", ptr: new([]byte), value: []byte{}},
	{input: "8D6162636465666768696A6B6C6D", ptr: new([]byte), value: []byte("abcdefghijklm")},
	{input: "C0", ptr: new([]byte), value: []byte{}},
	{input: "C3010203", ptr: new([]byte), value: []byte{1, 2, 3}},

	{
		input: "C3820102",
		ptr:   new([]byte),
		error: "rlp: input string too long for uint8, decoding into ([]uint8)[0]",
	},

	// byte arrays
	{input: "01", ptr: new([5]byte), value: [5]byte{1}},
	{input: "80", ptr: new([5]byte), value: [5]byte{}},
	{input: "850102030405", ptr: new([5]byte), value: [5]byte{1, 2, 3, 4, 5}},
	{input: "C0", ptr: new([5]byte), value: [5]byte{}},
	{input: "C3010203", ptr: new([5]byte), value: [5]byte{1, 2, 3, 0, 0}},

	{
		input: "C3820102",
		ptr:   new([5]byte),
		error: "rlp: input string too long for uint8, decoding into ([5]uint8)[0]",
	},
	{
		input: "86010203040506",
		ptr:   new([5]byte),
		error: "rlp: input string too long for [5]uint8",
	},

	// byte array reuse (should be zeroed)
	{input: "850102030405", ptr: &sharedByteArray, value: [5]byte{1, 2, 3, 4, 5}},
	{input: "8101", ptr: &sharedByteArray, value: [5]byte{1}}, // kind: String
	{input: "850102030405", ptr: &sharedByteArray, value: [5]byte{1, 2, 3, 4, 5}},
	{input: "01", ptr: &sharedByteArray, value: [5]byte{1}}, // kind: Byte
	{input: "C3010203", ptr: &sharedByteArray, value: [5]byte{1, 2, 3, 0, 0}},
	{input: "C101", ptr: &sharedByteArray, value: [5]byte{1}}, // kind: List

	// zero sized byte arrays
	{input: "80", ptr: new([0]byte), value: [0]byte{}},
	{input: "C0", ptr: new([0]byte), value: [0]byte{}},
	{input: "01", ptr: new([0]byte), error: "rlp: input string too long for [0]uint8"},
	{input: "8101", ptr: new([0]byte), error: "rlp: input string too long for [0]uint8"},

	// strings
	{input: "00", ptr: new(string), value: "\000"},
	{input: "8D6162636465666768696A6B6C6D", ptr: new(string), value: "abcdefghijklm"},
	{input: "C0", ptr: new(string), error: "rlp: expected input string or byte for string"},

	// big ints
	{input: "01", ptr: new(*big.Int), value: big.NewInt(1)},
	{input: "89FFFFFFFFFFFFFFFFFF", ptr: new(*big.Int), value: veryBigInt},
	{input: "820001", ptr: new(big.Int), error: "rlp: canon int error appends zero's for *big.Int"},
	{input: "10", ptr: new(big.Int), value: *big.NewInt(16)}, // non-pointer also works
	{input: "C0", ptr: new(*big.Int), error: "rlp: expected input string or byte for *big.Int"},

	// structs
	{input: "C0", ptr: new(simplestruct), value: simplestruct{0, ""}},
	{input: "C105", ptr: new(simplestruct), value: simplestruct{5, ""}},
	{input: "C50583343434", ptr: new(simplestruct), value: simplestruct{5, "444"}},
	{
		input: "C501C302C103",
		ptr:   new(recstruct),
		value: recstruct{1, &recstruct{2, &recstruct{3, nil}}},
	},

	{
		input: "C3010101",
		ptr:   new(simplestruct),
		error: "rlp: input list has too many elements for rlp.simplestruct",
	},
	{
		input: "C501C3C00000",
		ptr:   new(recstruct),
		error: "rlp: expected input string or byte for uint, decoding into (rlp.recstruct).Child.I",
	},

	// pointers
	{input: "00", ptr: new(*uint), value: (*uint)(nil)},
	{input: "80", ptr: new(*uint), value: (*uint)(nil)},
	{input: "C0", ptr: new(*uint), value: (*uint)(nil)},
	{input: "07", ptr: new(*uint), value: uintp(7)},
	{input: "8108", ptr: new(*uint), value: uintp(8)},
	{input: "C109", ptr: new(*[]uint), value: &[]uint{9}},
	{input: "C58403030303", ptr: new(*[][]byte), value: &[][]byte{{3, 3, 3, 3}}},

	// check that input position is advanced also for empty values.
	{input: "C3808005", ptr: new([]*uint), value: []*uint{nil, nil, uintp(5)}},

	// pointer should be reset to nil
	{input: "05", ptr: sharedPtr, value: uintp(5)},
	{input: "80", ptr: sharedPtr, value: (*uint)(nil)},

	// interface{}
	{input: "00", ptr: new(interface{}), value: []byte{0}},
	{input: "01", ptr: new(interface{}), value: []byte{1}},
	{input: "80", ptr: new(interface{}), value: []byte{}},
	{input: "850505050505", ptr: new(interface{}), value: []byte{5, 5, 5, 5, 5}},
	{input: "C0", ptr: new(interface{}), value: []interface{}{}},
	{input: "C50183040404", ptr: new(interface{}), value: []interface{}{[]byte{1}, []byte{4, 4, 4}}},
	{
		input: "C3010203",
		ptr:   new([]io.Reader),
		error: "rlp: type io.Reader is not RLP-serializable",
	},
}

func uintp(i uint) *uint { return &i }

func runTests(t *testing.T, decode func([]byte, interface{}) error) {
	for i, test := range decodeTests {
		input, err := hex.DecodeString(test.input)
		if err != nil {
			t.Errorf("test %d: invalid hex input %q", i, test.input)
			continue
		}
		err = decode(input, test.ptr)
		if err != nil && test.error == "" {
			t.Errorf("test %d: unexpected Decode error: %v\ndecoding into %T\ninput %q",
				i, err, test.ptr, test.input)
			continue
		}
		if test.error != "" && fmt.Sprint(err) != test.error {
			t.Errorf("test %d: Decode error mismatch\ngot  %v\nwant %v\ndecoding into %T\ninput %q",
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

func TestDecodeWithByteReader(t *testing.T) {
	runTests(t, func(input []byte, into interface{}) error {
		return Decode(bytes.NewReader(input), into)
	})
}

// plainReader reads from a byte slice but does not
// implement ReadByte. It is also not recognized by the
// size validation. This is useful to test how the decoder
// behaves on a non-buffered input stream.
type plainReader []byte

func newPlainReader(b []byte) io.Reader {
	return (*plainReader)(&b)
}

func (r *plainReader) Read(buf []byte) (n int, err error) {
	if len(*r) == 0 {
		return 0, io.EOF
	}
	n = copy(buf, *r)
	*r = (*r)[n:]
	return n, nil
}

func TestDecodeWithNonByteReader(t *testing.T) {
	runTests(t, func(input []byte, into interface{}) error {
		return Decode(newPlainReader(input), into)
	})
}

func TestDecodeStreamReset(t *testing.T) {
	s := NewStream(nil, 0)
	runTests(t, func(input []byte, into interface{}) error {
		s.Reset(bytes.NewReader(input), 0)
		return s.Decode(into)
	})
}

type testDecoder struct{ called bool }

func (t *testDecoder) DecodeRLP(s *Stream) error {
	if _, err := s.Uint(); err != nil {
		return err
	}
	t.called = true
	return nil
}

func TestDecodeDecoder(t *testing.T) {
	var s struct {
		T1 testDecoder
		T2 *testDecoder
		T3 **testDecoder
	}
	if err := Decode(bytes.NewReader(unhex("C3010203")), &s); err != nil {
		t.Fatalf("Decode error: %v", err)
	}

	if !s.T1.called {
		t.Errorf("DecodeRLP was not called for (non-pointer) testDecoder")
	}

	if s.T2 == nil {
		t.Errorf("*testDecoder has not been allocated")
	} else if !s.T2.called {
		t.Errorf("DecodeRLP was not called for *testDecoder")
	}

	if s.T3 == nil || *s.T3 == nil {
		t.Errorf("**testDecoder has not been allocated")
	} else if !(*s.T3).called {
		t.Errorf("DecodeRLP was not called for **testDecoder")
	}
}

type byteDecoder byte

func (bd *byteDecoder) DecodeRLP(s *Stream) error {
	_, err := s.Uint()
	*bd = 255
	return err
}

func (bd byteDecoder) called() bool {
	return bd == 255
}

// This test verifies that the byte slice/byte array logic
// does not kick in for element types implementing Decoder.
func TestDecoderInByteSlice(t *testing.T) {
	var slice []byteDecoder
	if err := Decode(bytes.NewReader(unhex("C101")), &slice); err != nil {
		t.Errorf("unexpected Decode error %v", err)
	} else if !slice[0].called() {
		t.Errorf("DecodeRLP not called for slice element")
	}

	var array [1]byteDecoder
	if err := Decode(bytes.NewReader(unhex("C101")), &array); err != nil {
		t.Errorf("unexpected Decode error %v", err)
	} else if !array[0].called() {
		t.Errorf("DecodeRLP not called for array element")
	}
}

func ExampleDecode() {
	input, _ := hex.DecodeString("C90A1486666F6F626172")

	type example struct {
		A, B    uint
		private uint // private fields are ignored
		String  string
	}

	var s example
	err := Decode(bytes.NewReader(input), &s)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
	} else {
		fmt.Printf("Decoded value: %#v\n", s)
	}
	// Output:
	// Decoded value: rlp.example{A:0xa, B:0x14, private:0x0, String:"foobar"}
}

func ExampleStream() {
	input, _ := hex.DecodeString("C90A1486666F6F626172")
	s := NewStream(bytes.NewReader(input), 0)

	// Check what kind of value lies ahead
	kind, size, _ := s.Kind()
	fmt.Printf("Kind: %v size:%d\n", kind, size)

	// Enter the list
	if _, err := s.List(); err != nil {
		fmt.Printf("List error: %v\n", err)
		return
	}

	// Decode elements
	fmt.Println(s.Uint())
	fmt.Println(s.Uint())
	fmt.Println(s.Bytes())

	// Acknowledge end of list
	if err := s.ListEnd(); err != nil {
		fmt.Printf("ListEnd error: %v\n", err)
	}
	// Output:
	// Kind: List size:9
	// 10 <nil>
	// 20 <nil>
	// [102 111 111 98 97 114] <nil>
}

func BenchmarkDecode(b *testing.B) {
	enc := encodeTestSlice(90000)
	b.SetBytes(int64(len(enc)))
	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		var s []uint
		r := bytes.NewReader(enc)
		if err := Decode(r, &s); err != nil {
			b.Fatalf("Decode error: %v", err)
		}
	}
}

func BenchmarkDecodeIntSliceReuse(b *testing.B) {
	enc := encodeTestSlice(100000)
	b.SetBytes(int64(len(enc)))
	b.ReportAllocs()
	b.ResetTimer()

	var s []uint
	for i := 0; i < b.N; i++ {
		r := bytes.NewReader(enc)
		if err := Decode(r, &s); err != nil {
			b.Fatalf("Decode error: %v", err)
		}
	}
}

func encodeTestSlice(n uint) []byte {
	s := make([]uint, n)
	for i := uint(0); i < n; i++ {
		s[i] = i
	}
	b, err := EncodeToBytes(s)
	if err != nil {
		panic(fmt.Sprintf("encode error: %v", err))
	}
	return b
}

func unhex(str string) []byte {
	b, err := hex.DecodeString(str)
	if err != nil {
		panic(fmt.Sprintf("invalid hex string: %q", str))
	}
	return b
}
