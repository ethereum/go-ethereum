package rlp

import (
	"fmt"
	"math/big"
	"reflect"
	"sync"
)

type decoder func(*Stream, reflect.Value) error

type typeinfo struct {
	decoder
}

var (
	typeCacheMutex sync.RWMutex
	typeCache      = make(map[reflect.Type]*typeinfo)
)

func cachedTypeInfo(typ reflect.Type) (*typeinfo, error) {
	typeCacheMutex.RLock()
	info := typeCache[typ]
	typeCacheMutex.RUnlock()
	if info != nil {
		return info, nil
	}
	// not in the cache, need to generate info for this type.
	typeCacheMutex.Lock()
	defer typeCacheMutex.Unlock()
	return cachedTypeInfo1(typ)
}

func cachedTypeInfo1(typ reflect.Type) (*typeinfo, error) {
	info := typeCache[typ]
	if info != nil {
		// another goroutine got the write lock first
		return info, nil
	}
	// put a dummmy value into the cache before generating.
	// if the generator tries to lookup itself, it will get
	// the dummy value and won't call itself recursively.
	typeCache[typ] = new(typeinfo)
	info, err := genTypeInfo(typ)
	if err != nil {
		// remove the dummy value if the generator fails
		delete(typeCache, typ)
		return nil, err
	}
	*typeCache[typ] = *info
	return typeCache[typ], err
}

var (
	decoderInterface = reflect.TypeOf(new(Decoder)).Elem()
	bigInt           = reflect.TypeOf(big.Int{})
)

func genTypeInfo(typ reflect.Type) (info *typeinfo, err error) {
	info = new(typeinfo)
	kind := typ.Kind()
	switch {
	case typ.Implements(decoderInterface):
		info.decoder = decodeDecoder
	case kind != reflect.Ptr && reflect.PtrTo(typ).Implements(decoderInterface):
		info.decoder = decodeDecoderNoPtr
	case typ.AssignableTo(reflect.PtrTo(bigInt)):
		info.decoder = decodeBigInt
	case typ.AssignableTo(bigInt):
		info.decoder = decodeBigIntNoPtr
	case isInteger(kind):
		info.decoder = makeNumDecoder(typ)
	case kind == reflect.String:
		info.decoder = decodeString
	case kind == reflect.Slice || kind == reflect.Array:
		info.decoder, err = makeListDecoder(typ)
	case kind == reflect.Struct:
		info.decoder, err = makeStructDecoder(typ)
	case kind == reflect.Ptr:
		info.decoder, err = makePtrDecoder(typ)
	case kind == reflect.Interface && typ.NumMethod() == 0:
		info.decoder = decodeInterface
	default:
		err = fmt.Errorf("rlp: type %v is not RLP-serializable", typ)
	}
	return info, err
}

func isInteger(k reflect.Kind) bool {
	return k >= reflect.Int && k <= reflect.Uintptr
}
