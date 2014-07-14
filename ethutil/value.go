package ethutil

import (
	"bytes"
	"fmt"
	"math/big"
	"reflect"
)

// Data values are returned by the rlp decoder. The data values represents
// one item within the rlp data structure. It's responsible for all the casting
// It always returns something valid
type Value struct {
	Val  interface{}
	kind reflect.Value
}

func (val *Value) String() string {
	return fmt.Sprintf("%x", val.Val)
}

func NewValue(val interface{}) *Value {
	t := val
	if v, ok := val.(*Value); ok {
		t = v.Val
	}

	return &Value{Val: t}
}

func (val *Value) Type() reflect.Kind {
	return reflect.TypeOf(val.Val).Kind()
}

func (val *Value) IsNil() bool {
	return val.Val == nil
}

func (val *Value) Len() int {
	//return val.kind.Len()
	if data, ok := val.Val.([]interface{}); ok {
		return len(data)
	} else if data, ok := val.Val.([]byte); ok {
		return len(data)
	} else if data, ok := val.Val.(string); ok {
		return len(data)
	}

	return 0
}

func (val *Value) Raw() interface{} {
	return val.Val
}

func (val *Value) Interface() interface{} {
	return val.Val
}

func (val *Value) Uint() uint64 {
	if Val, ok := val.Val.(uint8); ok {
		return uint64(Val)
	} else if Val, ok := val.Val.(uint16); ok {
		return uint64(Val)
	} else if Val, ok := val.Val.(uint32); ok {
		return uint64(Val)
	} else if Val, ok := val.Val.(uint64); ok {
		return Val
	} else if Val, ok := val.Val.(int); ok {
		return uint64(Val)
	} else if Val, ok := val.Val.(uint); ok {
		return uint64(Val)
	} else if Val, ok := val.Val.([]byte); ok {
		return ReadVarint(bytes.NewReader(Val))
	}

	return 0
}

func (val *Value) Byte() byte {
	if Val, ok := val.Val.(byte); ok {
		return Val
	}

	return 0x0
}

func (val *Value) BigInt() *big.Int {
	if a, ok := val.Val.([]byte); ok {
		b := new(big.Int).SetBytes(a)

		return b
	} else if a, ok := val.Val.(*big.Int); ok {
		return a
	} else {
		return big.NewInt(int64(val.Uint()))
	}

	return big.NewInt(0)
}

func (val *Value) Str() string {
	if a, ok := val.Val.([]byte); ok {
		return string(a)
	} else if a, ok := val.Val.(string); ok {
		return a
	} else if a, ok := val.Val.(byte); ok {
		return string(a)
	}

	return ""
}

func (val *Value) Bytes() []byte {
	if a, ok := val.Val.([]byte); ok {
		return a
	} else if s, ok := val.Val.(byte); ok {
		return []byte{s}
	} else if s, ok := val.Val.(string); ok {
		return []byte(s)
	}

	return []byte{}
}

func (val *Value) Slice() []interface{} {
	if d, ok := val.Val.([]interface{}); ok {
		return d
	}

	return []interface{}{}
}

func (val *Value) SliceFrom(from int) *Value {
	slice := val.Slice()

	return NewValue(slice[from:])
}

func (val *Value) SliceTo(to int) *Value {
	slice := val.Slice()

	return NewValue(slice[:to])
}

func (val *Value) SliceFromTo(from, to int) *Value {
	slice := val.Slice()

	return NewValue(slice[from:to])
}

// TODO More type checking methods
func (val *Value) IsSlice() bool {
	return val.Type() == reflect.Slice
}

func (val *Value) IsStr() bool {
	return val.Type() == reflect.String
}

// Special list checking function. Something is considered
// a list if it's of type []interface{}. The list is usually
// used in conjunction with rlp decoded streams.
func (val *Value) IsList() bool {
	_, ok := val.Val.([]interface{})

	return ok
}

func (val *Value) IsEmpty() bool {
	return val.Val == nil || ((val.IsSlice() || val.IsStr()) && val.Len() == 0)
}

// Threat the value as a slice
func (val *Value) Get(idx int) *Value {
	if d, ok := val.Val.([]interface{}); ok {
		// Guard for oob
		if len(d) <= idx {
			return NewValue(nil)
		}

		if idx < 0 {
			return NewValue(nil)
		}

		return NewValue(d[idx])
	}

	// If this wasn't a slice you probably shouldn't be using this function
	return NewValue(nil)
}

func (self *Value) Copy() *Value {
	switch val := self.Val.(type) {
	case *big.Int:
		return NewValue(new(big.Int).Set(val))
	case []byte:
		return NewValue(CopyBytes(val))
	default:
		return NewValue(self.Val)
	}

	return nil
}

func (val *Value) Cmp(o *Value) bool {
	return reflect.DeepEqual(val.Val, o.Val)
}

func (val *Value) Encode() []byte {
	return Encode(val.Val)
}

// Assume that the data we have is encoded
func (self *Value) Decode() {
	v, _ := Decode(self.Bytes(), 0)
	self.Val = v
}

func NewValueFromBytes(data []byte) *Value {
	if len(data) != 0 {
		data, _ := Decode(data, 0)
		return NewValue(data)
	}

	return NewValue(nil)
}

// Value setters
func NewSliceValue(s interface{}) *Value {
	list := EmptyValue()

	if s != nil {
		if slice, ok := s.([]interface{}); ok {
			for _, val := range slice {
				list.Append(val)
			}
		} else if slice, ok := s.([]string); ok {
			for _, val := range slice {
				list.Append(val)
			}
		}
	}

	return list
}

func EmptyValue() *Value {
	return NewValue([]interface{}{})
}

func (val *Value) AppendList() *Value {
	list := EmptyValue()
	val.Val = append(val.Slice(), list)

	return list
}

func (val *Value) Append(v interface{}) *Value {
	val.Val = append(val.Slice(), v)

	return val
}

type ValueIterator struct {
	value        *Value
	currentValue *Value
	idx          int
}

func (val *Value) NewIterator() *ValueIterator {
	return &ValueIterator{value: val}
}

func (it *ValueIterator) Next() bool {
	if it.idx >= it.value.Len() {
		return false
	}

	it.currentValue = it.value.Get(it.idx)
	it.idx++

	return true
}

func (it *ValueIterator) Value() *Value {
	return it.currentValue
}

func (it *ValueIterator) Idx() int {
	return it.idx
}
