package rpc

import (
	"bytes"
	"encoding/json"
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
)

func TestInvalidTypeError(t *testing.T) {
	err := NewInvalidTypeError("testField", "not string")
	expected := "invalid type on field testField: not string"

	if err.Error() != expected {
		t.Error(err.Error())
	}
}

func TestInsufficientParamsError(t *testing.T) {
	err := NewInsufficientParamsError(0, 1)
	expected := "insufficient params, want 1 have 0"

	if err.Error() != expected {
		t.Error(err.Error())
	}
}

func TestNotImplementedError(t *testing.T) {
	err := NewNotImplementedError("foo")
	expected := "foo method not implemented"

	if err.Error() != expected {
		t.Error(err.Error())
	}
}

func TestDecodeParamError(t *testing.T) {
	err := NewDecodeParamError("foo")
	expected := "could not decode, foo"

	if err.Error() != expected {
		t.Error(err.Error())
	}
}

func TestValidationError(t *testing.T) {
	err := NewValidationError("foo", "should be `bar`")
	expected := "foo not valid, should be `bar`"

	if err.Error() != expected {
		t.Error(err.Error())
	}
}

func TestHexdataMarshalNil(t *testing.T) {
	hd := newHexData([]byte{})
	hd.isNil = true
	v, _ := json.Marshal(hd)
	if string(v) != "null" {
		t.Errorf("Expected null, got %s", v)
	}
}

func TestHexnumMarshalNil(t *testing.T) {
	hn := newHexNum([]byte{})
	hn.isNil = true
	v, _ := json.Marshal(hn)
	if string(v) != "null" {
		t.Errorf("Expected null, got %s", v)
	}
}

func TestHexdataNil(t *testing.T) {
	v := newHexData(nil)
	if v.isNil != true {
		t.Errorf("Expected isNil to be true, but is %v", v.isNil)
	}
}

func TestHexdataPtrHash(t *testing.T) {
	in := common.Hash{0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20, 21, 22, 23, 24, 25, 26, 27, 28, 29, 30, 31}
	v := newHexData(&in)
	if bytes.Compare(in.Bytes(), v.data) != 0 {
		t.Errorf("Got % x expected % x", in, v.data)
	}
}

func TestHexdataPtrHashNil(t *testing.T) {
	var in *common.Hash
	in = nil
	v := newHexData(in)
	if !v.isNil {
		t.Errorf("Expect isNil to be true, but is %v", v.isNil)
	}
}

func TestHexdataPtrAddress(t *testing.T) {
	in := common.Address{0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19}
	v := newHexData(&in)
	if bytes.Compare(in.Bytes(), v.data) != 0 {
		t.Errorf("Got % x expected % x", in, v.data)
	}
}

func TestHexdataPtrAddressNil(t *testing.T) {
	var in *common.Address
	in = nil
	v := newHexData(in)
	if !v.isNil {
		t.Errorf("Expect isNil to be true, but is %v", v.isNil)
	}
}

func TestHexdataPtrBloom(t *testing.T) {
	in := types.Bloom{0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19}
	v := newHexData(&in)
	if bytes.Compare(in.Bytes(), v.data) != 0 {
		t.Errorf("Got % x expected % x", in, v.data)
	}
}

func TestHexdataPtrBloomNil(t *testing.T) {
	var in *types.Bloom
	in = nil
	v := newHexData(in)
	if !v.isNil {
		t.Errorf("Expect isNil to be true, but is %v", v.isNil)
	}
}

func TestHexdataBigintNil(t *testing.T) {
	var in *big.Int
	in = nil
	v := newHexData(in)
	if !v.isNil {
		t.Errorf("Expect isNil to be true, but is %v", v.isNil)
	}
}

func TestHexdataUint(t *testing.T) {
	var in = uint(16)
	var expected = []byte{0x10}
	v := newHexData(in)
	if bytes.Compare(expected, v.data) != 0 {
		t.Errorf("Expected % x got % x", expected, v.data)
	}
}

func TestHexdataInt8(t *testing.T) {
	var in = int8(16)
	var expected = []byte{0x10}
	v := newHexData(in)
	if bytes.Compare(expected, v.data) != 0 {
		t.Errorf("Expected % x got % x", expected, v.data)
	}
}

func TestHexdataUint8(t *testing.T) {
	var in = uint8(16)
	var expected = []byte{0x10}
	v := newHexData(in)
	if bytes.Compare(expected, v.data) != 0 {
		t.Errorf("Expected % x got % x", expected, v.data)
	}
}

func TestHexdataInt16(t *testing.T) {
	var in = int16(16)
	var expected = []byte{0x10}
	v := newHexData(in)
	if bytes.Compare(expected, v.data) != 0 {
		t.Errorf("Expected % x got % x", expected, v.data)
	}
}

func TestHexdataUint16(t *testing.T) {
	var in = uint16(16)
	var expected = []byte{0x0, 0x10}
	v := newHexData(in)
	if bytes.Compare(expected, v.data) != 0 {
		t.Errorf("Expected % x got % x", expected, v.data)
	}
}

func TestHexdataInt32(t *testing.T) {
	var in = int32(16)
	var expected = []byte{0x10}
	v := newHexData(in)
	if bytes.Compare(expected, v.data) != 0 {
		t.Errorf("Expected % x got % x", expected, v.data)
	}
}

func TestHexdataUint32(t *testing.T) {
	var in = uint32(16)
	var expected = []byte{0x0, 0x0, 0x0, 0x10}
	v := newHexData(in)
	if bytes.Compare(expected, v.data) != 0 {
		t.Errorf("Expected % x got % x", expected, v.data)
	}
}
