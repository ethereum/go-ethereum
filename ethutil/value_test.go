package ethutil

import (
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

	if str.Str() != "str" {
		t.Errorf("expected Str to return 'str', got %s", str.Str())
	}

	if num.Uint() != 1 {
		t.Errorf("expected Uint to return '1', got %d", num.Uint())
	}

	exp := []interface{}{1}
	if !NewValue(inter.Interface()).Cmp(NewValue(exp)) {
		t.Errorf("expected Interface to return '%v', got %v", exp, num.Interface())
	}
}
