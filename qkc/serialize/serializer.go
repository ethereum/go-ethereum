// Ported verbatim from github.com/QuarkChain/goquarkchain/serialize (byte-compatible).

package serialize

import (
	"encoding/binary"
	"errors"
	"fmt"
	"math/big"
	"reflect"
)

func Serialize(w *[]byte, val interface{}) error {
	return SerializeWithTags(w, val, Tags{ByteSizeOfSliceLen: 1})
}

func SerializeWithTags(w *[]byte, val interface{}, ts Tags) error {
	rval := reflect.ValueOf(val)
	ti, err := cachedTypeInfo(rval.Type())
	if err != nil {
		return err
	}

	return ti.serializer(rval, w, ts)
}

// SerializeToBytes returns the serialize result of val.
func SerializeToBytes(val interface{}) ([]byte, error) {
	w := make([]byte, 0, 512)
	if err := Serialize(&w, val); err != nil {
		return nil, err
	}

	return w, nil
}

func makeSerializer(typ reflect.Type) (serializer, error) {
	kind := typ.Kind()
	switch {
	//check Ptr first and add optional byte output if ts is nilok,
	//then get serializer for typ.Elem() which is not a ptr
	case kind == reflect.Ptr:
		return serializePtr, nil
	case kind != reflect.Ptr && reflect.PtrTo(typ).Implements(serializableInterface):
		return serializeSerializableInterface, nil
	case typ.AssignableTo(bigInt):
		return serializeBigIntNoPtr, nil
	case isUint(kind):
		return serializeUint, nil
	case kind == reflect.Bool:
		return serializeBool, nil
	case kind == reflect.String:
		return serializeString, nil
	case kind == reflect.Slice && isByte(typ.Elem()):
		return serializeByteSlice, nil
	case kind == reflect.Array && isByte(typ.Elem()):
		return serializeByteArray, nil
	case kind == reflect.Slice || kind == reflect.Array:
		return serializeList, nil
	case kind == reflect.Struct:
		return serializeStruct, nil
	default:
		return nil, fmt.Errorf("type %v is not serializable", typ)
	}
}

func serializeSerializableInterface(val reflect.Value, w *[]byte, tags Tags) error {
	if !val.CanAddr() {
		return fmt.Errorf("ser: unaddressable value of type %v, Serialize is pointer method", val.Type())
	}

	return val.Addr().Interface().(Serializable).Serialize(w)
}

func prefillByteArray(size int, barray []byte) ([]byte, error) {
	len := len(barray)
	if len > size {
		return nil, errors.New("barray len is larger then expected size")
	}
	if len == size {
		return barray, nil
	}

	bytes := make([]byte, size, size)
	var startIndex = size - len
	copy(bytes[startIndex:], barray)
	return bytes, nil
}

func serializeFixSizeBigUint(val *big.Int, size int, w *[]byte) error {
	if val == nil {
		bytes := make([]byte, size, size)
		*w = append(*w, bytes...)
		return nil
	}
	bytes, err := prefillByteArray(size, val.Bytes())
	if err == nil {
		*w = append(*w, bytes...)
	}

	return err
}

func serializeBigIntNoPtr(val reflect.Value, w *[]byte, ts Tags) error {
	i := val.Interface().(big.Int)
	return serializeBigInt(&i, w)
}

func serializeBigInt(i *big.Int, w *[]byte) error {
	var bytes []byte
	if cmp := i.Cmp(big.NewInt(0)); cmp == -1 {
		return fmt.Errorf("ser: cannot serialize negative *big.Int")
	} else if cmp > 0 {
		bytes = i.Bytes()
	}

	*w = append(*w, uint8(len(bytes)))
	*w = append(*w, bytes...)
	return nil
}

func serializeBool(val reflect.Value, w *[]byte, ts Tags) error {
	if val.Bool() {
		*w = append(*w, 0x01)
	} else {
		*w = append(*w, 0x00)
	}

	return nil
}

func serializeByteArray(val reflect.Value, w *[]byte, ts Tags) error {
	if !val.CanAddr() {
		// Slice requires the value to be addressable.
		// Make it addressable by copying.
		copy := reflect.New(val.Type()).Elem()
		copy.Set(val)
		val = copy
	}

	size := val.Len()
	slice := val.Slice(0, size).Bytes()

	*w = append(*w, slice...)
	return nil
}

func writeListLen(w *[]byte, len int, byteSizeOfSliceLen int) error {
	sizeBytes := make([]byte, byteSizeOfSliceLen)
	for i := byteSizeOfSliceLen - 1; i >= 0 && len != 0; i-- {
		sizeBytes[i] = byte(len)
		len = len >> 8
	}
	if len > 0 {
		return errors.New("barray len is larger then expected size")
	}

	*w = append(*w, sizeBytes...)
	return nil
}

// serializePrependedSizeBytes
func serializeByteSlice(val reflect.Value, w *[]byte, ts Tags) error {
	err := writeListLen(w, val.Len(), ts.ByteSizeOfSliceLen)
	if err != nil {
		return err
	}

	bytes := val.Bytes()
	*w = append(*w, bytes...)
	return nil
}

// PrependedSizeListSerializer
func serializeList(val reflect.Value, w *[]byte, ts Tags) error {
	typeinfo, err := cachedTypeInfo(val.Type().Elem())
	if err != nil {
		return err
	}

	if val.Kind() == reflect.Slice {
		err = writeListLen(w, val.Len(), ts.ByteSizeOfSliceLen)
		if err != nil {
			return err
		}
	}

	vlen := val.Len()
	for i := 0; i < vlen; i++ {
		if err := typeinfo.serializer(val.Index(i), w, Tags{ByteSizeOfSliceLen: 1}); err != nil {
			return err
		}
	}

	return nil
}

func serializeStruct(val reflect.Value, w *[]byte, ts Tags) error {
	fields, err := structFields(val.Type())
	if err != nil {
		return err
	}

	for _, f := range fields {
		if err := f.info.serializer(val.Field(f.index), w, f.tags); err != nil {
			return err
		}
	}

	return nil
}

func SerializeStructWithout(val reflect.Value, w *[]byte, excludeList map[string]bool) error {
	fields, err := structFields(val.Type())
	if err != nil {
		return err
	}

	for _, f := range fields {
		if excludeList != nil {
			if _, ok := excludeList[f.name]; ok {
				continue
			}
		}

		if err := f.info.serializer(val.Field(f.index), w, f.tags); err != nil {
			return err
		}
	}

	return nil
}

func serializeString(val reflect.Value, w *[]byte, ts Tags) error {
	s := val.String()

	sizeBytes := make([]byte, 4, 4)
	binary.BigEndian.PutUint32(sizeBytes, uint32(val.Len()))

	*w = append(*w, sizeBytes...)
	*w = append(*w, s...)
	return nil
}

func serializePtr(val reflect.Value, w *[]byte, ts Tags) error {
	typ := val.Type()
	typeinfo, err := cachedTypeInfo(typ.Elem())
	if err != nil {
		return err
	}
	switch {
	case val.IsNil() && ts.NilOK:
		*w = append(*w, 0)
		return nil
	case val.IsNil() && typ.Implements(serializableInterface):
		zero := reflect.New(typ.Elem())
		return typeinfo.serializer(zero.Elem(), w, ts)
	case val.IsNil():
		zero := reflect.Zero(typ.Elem())
		return typeinfo.serializer(zero, w, ts)
	default:
		if ts.NilOK {
			*w = append(*w, 1)
		}
		return typeinfo.serializer(val.Elem(), w, ts)
	}
}

func serializeUint(val reflect.Value, w *[]byte, ts Tags) error {
	kind := val.Type().Kind()
	value := val.Uint()
	switch kind {
	case reflect.Uint8:
		*w = append(*w, byte(value))
		return nil
	case reflect.Uint16:
		*w = append(*w, byte(value>>8), byte(value))
		break
	case reflect.Uint32:
		*w = append(*w, byte(value>>24), byte(value>>16), byte(value>>8), byte(value))
		return nil
	case reflect.Uint64:
		*w = append(*w, byte(value>>56), byte(value>>48), byte(value>>40), byte(value>>32),
			byte(value>>24), byte(value>>16), byte(value>>8), byte(value))
		return nil
	case reflect.Uint:
		//As Uint would be 32/64 bit, so
		var bi big.Int
		return serializeBigInt(bi.SetUint64(val.Uint()), w)
	default:
		return fmt.Errorf("ser: invalid Uint type: %s", val.Type().Name())
	}
	return nil
}
