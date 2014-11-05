package ethutil

import (
	"bytes"
	"math/big"
	"testing"
)

func TestValueCmp(t *testing.T) {
	val1 := NewValue("hello")
	val2 := NewValue("world")
	if val1.Cmp(val2) {
		t.Error("Expected values not to be equal")
	}

	val3 := NewValue("hello")
	val4 := NewValue("hello")
	if !val3.Cmp(val4) {
		t.Error("Expected values to be equal")
	}
}

func TestValueTypes(t *testing.T) {
	str := NewValue("str")
	num := NewValue(1)
	inter := NewValue([]interface{}{1})
	byt := NewValue([]byte{1, 2, 3, 4})
	bigInt := NewValue(big.NewInt(10))

	if str.Str() != "str" {
		t.Errorf("expected Str to return 'str', got %s", str.Str())
	}

	if num.Uint() != 1 {
		t.Errorf("expected Uint to return '1', got %d", num.Uint())
	}

	interExp := []interface{}{1}
	if !NewValue(inter.Interface()).Cmp(NewValue(interExp)) {
		t.Errorf("expected Interface to return '%v', got %v", interExp, num.Interface())
	}

	bytExp := []byte{1, 2, 3, 4}
	if bytes.Compare(byt.Bytes(), bytExp) != 0 {
		t.Errorf("expected Bytes to return '%v', got %v", bytExp, byt.Bytes())
	}

	bigExp := big.NewInt(10)
	if bigInt.BigInt().Cmp(bigExp) != 0 {
		t.Errorf("expected BigInt to return '%v', got %v", bigExp, bigInt.BigInt())
	}
}

func TestIterator(t *testing.T) {
	value := NewValue([]interface{}{1, 2, 3})
	it := value.NewIterator()
	values := []uint64{1, 2, 3}
	i := 0
	for it.Next() {
		if values[i] != it.Value().Uint() {
			t.Errorf("Expected %d, got %d", values[i], it.Value().Uint())
		}
		i++
	}
}

func TestMath(t *testing.T) {
	a := NewValue(1)
	a.Add(1).Add(1)

	if !a.DeepCmp(NewValue(3)) {
		t.Error("Expected 3, got", a)
	}

	a = NewValue(2)
	a.Sub(1).Sub(1)
	if !a.DeepCmp(NewValue(0)) {
		t.Error("Expected 0, got", a)
	}
}

func TestString(t *testing.T) {
	data := "10"
	exp := int64(10)
	res := NewValue(data).Int()
	if res != exp {
		t.Errorf("Exprected %d Got res", exp, res)
	}
}
