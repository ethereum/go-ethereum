package abi

import (
	"bytes"
	"math/big"
	"reflect"
	"testing"
)

func TestNumberTypes(t *testing.T) {
	ubytes := make([]byte, 32)
	ubytes[31] = 1
	sbytesmin := []byte{1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1}

	unsigned := U256(big.NewInt(1))
	if !bytes.Equal(unsigned, ubytes) {
		t.Error("expected %x got %x", ubytes, unsigned)
	}

	signed := S256(big.NewInt(1))
	if !bytes.Equal(signed, ubytes) {
		t.Error("expected %x got %x", ubytes, unsigned)
	}

	signed = S256(big.NewInt(-1))
	if !bytes.Equal(signed, sbytesmin) {
		t.Error("expected %x got %x", ubytes, unsigned)
	}
}

func TestPackNumber(t *testing.T) {
	ubytes := make([]byte, 32)
	ubytes[31] = 1
	sbytesmin := []byte{1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1}
	maxunsigned := []byte{255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255}

	packed := packNum(reflect.ValueOf(1), IntTy)
	if !bytes.Equal(packed, ubytes) {
		t.Errorf("expected %x got %x", ubytes, packed)
	}
	packed = packNum(reflect.ValueOf(-1), IntTy)
	if !bytes.Equal(packed, sbytesmin) {
		t.Errorf("expected %x got %x", ubytes, packed)
	}
	packed = packNum(reflect.ValueOf(1), UintTy)
	if !bytes.Equal(packed, ubytes) {
		t.Errorf("expected %x got %x", ubytes, packed)
	}
	packed = packNum(reflect.ValueOf(-1), UintTy)
	if !bytes.Equal(packed, maxunsigned) {
		t.Errorf("expected %x got %x", maxunsigned, packed)
	}

	packed = packNum(reflect.ValueOf("string"), UintTy)
	if packed != nil {
		t.Errorf("expected 'string' to pack to nil. got %x instead", packed)
	}
}

func TestSigned(t *testing.T) {
	if isSigned(reflect.ValueOf(uint(10))) {
		t.Error()
	}

	if !isSigned(reflect.ValueOf(int(10))) {
		t.Error()
	}

	if !isSigned(reflect.ValueOf(big.NewInt(10))) {
		t.Error()
	}
}
