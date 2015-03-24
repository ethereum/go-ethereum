package common

import (
	"bytes"
	"fmt"
	"io"
	"math/big"
	"reflect"
	"strconv"

	"github.com/ethereum/go-ethereum/rlp"
)

// Value can hold values of certain basic types and provides ways to
// convert between types without bothering to check whether the
// conversion is actually meaningful.
//
// It currently supports the following types:
//
//    - int{,8,16,32,64}
//    - uint{,8,16,32,64}
//    - *big.Int
//    - []byte, string
//    - []interface{}
//
// Value is useful whenever you feel that Go's types limit your
// ability to express yourself. In these situations, use Value and
// forget about this strong typing nonsense.
type Value struct{ Val interface{} }

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
	if data, ok := val.Val.([]interface{}); ok {
		return len(data)
	}

	return len(val.Bytes())
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
	} else if Val, ok := val.Val.(float32); ok {
		return uint64(Val)
	} else if Val, ok := val.Val.(float64); ok {
		return uint64(Val)
	} else if Val, ok := val.Val.(int); ok {
		return uint64(Val)
	} else if Val, ok := val.Val.(uint); ok {
		return uint64(Val)
	} else if Val, ok := val.Val.([]byte); ok {
		return new(big.Int).SetBytes(Val).Uint64()
	} else if Val, ok := val.Val.(*big.Int); ok {
		return Val.Uint64()
	}

	return 0
}

func (val *Value) Int() int64 {
	if Val, ok := val.Val.(int8); ok {
		return int64(Val)
	} else if Val, ok := val.Val.(int16); ok {
		return int64(Val)
	} else if Val, ok := val.Val.(int32); ok {
		return int64(Val)
	} else if Val, ok := val.Val.(int64); ok {
		return Val
	} else if Val, ok := val.Val.(int); ok {
		return int64(Val)
	} else if Val, ok := val.Val.(float32); ok {
		return int64(Val)
	} else if Val, ok := val.Val.(float64); ok {
		return int64(Val)
	} else if Val, ok := val.Val.([]byte); ok {
		return new(big.Int).SetBytes(Val).Int64()
	} else if Val, ok := val.Val.(*big.Int); ok {
		return Val.Int64()
	} else if Val, ok := val.Val.(string); ok {
		n, _ := strconv.Atoi(Val)
		return int64(n)
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
	} else if a, ok := val.Val.(string); ok {
		return Big(a)
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
	} else if s, ok := val.Val.(*big.Int); ok {
		return s.Bytes()
	} else {
		return big.NewInt(val.Int()).Bytes()
	}

	return []byte{}
}

func (val *Value) Err() error {
	if err, ok := val.Val.(error); ok {
		return err
	}

	return nil
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

func (self *Value) IsErr() bool {
	_, ok := self.Val.(error)
	return ok
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

func (self *Value) DeepCmp(o *Value) bool {
	return bytes.Compare(self.Bytes(), o.Bytes()) == 0
}

func (self *Value) DecodeRLP(s *rlp.Stream) error {
	var v interface{}
	if err := s.Decode(&v); err != nil {
		return err
	}
	self.Val = v
	return nil
}

func (self *Value) EncodeRLP(w io.Writer) error {
	if self == nil {
		w.Write(rlp.EmptyList)
		return nil
	} else {
		return rlp.Encode(w, self.Val)
	}
}

// NewValueFromBytes decodes RLP data.
// The contained value will be nil if data contains invalid RLP.
func NewValueFromBytes(data []byte) *Value {
	v := new(Value)
	if len(data) != 0 {
		if err := rlp.DecodeBytes(data, v); err != nil {
			v.Val = nil
		}
	}
	return v
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

const (
	valOpAdd = iota
	valOpDiv
	valOpMul
	valOpPow
	valOpSub
)

// Math stuff
func (self *Value) doOp(op int, other interface{}) *Value {
	left := self.BigInt()
	right := NewValue(other).BigInt()

	switch op {
	case valOpAdd:
		self.Val = left.Add(left, right)
	case valOpDiv:
		self.Val = left.Div(left, right)
	case valOpMul:
		self.Val = left.Mul(left, right)
	case valOpPow:
		self.Val = left.Exp(left, right, Big0)
	case valOpSub:
		self.Val = left.Sub(left, right)
	}

	return self
}

func (self *Value) Add(other interface{}) *Value {
	return self.doOp(valOpAdd, other)
}

func (self *Value) Sub(other interface{}) *Value {
	return self.doOp(valOpSub, other)
}

func (self *Value) Div(other interface{}) *Value {
	return self.doOp(valOpDiv, other)
}

func (self *Value) Mul(other interface{}) *Value {
	return self.doOp(valOpMul, other)
}

func (self *Value) Pow(other interface{}) *Value {
	return self.doOp(valOpPow, other)
}

type ValueIterator struct {
	value        *Value
	currentValue *Value
	idx          int
}

func (val *Value) NewIterator() *ValueIterator {
	return &ValueIterator{value: val}
}

func (it *ValueIterator) Len() int {
	return it.value.Len()
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
	return it.idx - 1
}
