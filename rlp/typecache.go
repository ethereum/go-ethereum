package rlp

import (
	"reflect"
	"sync"
)

var (
	typeCacheMutex sync.RWMutex
	typeCache      = make(map[reflect.Type]*typeinfo)
)

type typeinfo struct {
	decoder
	writer
}

type decoder func(*Stream, reflect.Value) error

type writer func(reflect.Value, *encbuf) error

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

func structFields(typ reflect.Type) (fields []field, err error) {
	for i := 0; i < typ.NumField(); i++ {
		if f := typ.Field(i); f.PkgPath == "" { // exported
			info, err := cachedTypeInfo1(f.Type)
			if err != nil {
				return nil, err
			}
			fields = append(fields, field{i, info})
		}
	}
	return fields, nil
}

func genTypeInfo(typ reflect.Type) (info *typeinfo, err error) {
	info = new(typeinfo)
	if info.decoder, err = makeDecoder(typ); err != nil {
		return nil, err
	}
	if info.writer, err = makeWriter(typ); err != nil {
		return nil, err
	}
	return info, nil
}

func isUint(k reflect.Kind) bool {
	return k >= reflect.Uint && k <= reflect.Uintptr
}
