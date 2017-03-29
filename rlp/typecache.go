// Copyright 2014 The go-ethereum Authors
// This file is part of the go-ethereum library.
//
// The go-ethereum library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The go-ethereum library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the go-ethereum library. If not, see <http://www.gnu.org/licenses/>.

package rlp

import (
	"fmt"
	"reflect"
	"strings"
	"sync"
)

var (
	typeCacheMutex sync.RWMutex
	typeCache      = make(map[typekey]*typeinfo)
)

type typeinfo struct {
	decoder
	writer
}

// represents struct tags
type tags struct {
	// rlp:"nil" controls whether empty input results in a nil pointer.
	nilOK bool
	// rlp:"tail" controls whether this field swallows additional list
	// elements. It can only be set for the last field, which must be
	// of slice type.
	tail bool
	// rlp:"-" ignores fields.
	ignored bool
}

type typekey struct {
	reflect.Type
	// the key must include the struct tags because they
	// might generate a different decoder.
	tags
}

type decoder func(*Stream, reflect.Value) error

type writer func(reflect.Value, *encbuf) error

func cachedTypeInfo(typ reflect.Type, tags tags) (*typeinfo, error) {
	typeCacheMutex.RLock()
	info := typeCache[typekey{typ, tags}]
	typeCacheMutex.RUnlock()
	if info != nil {
		return info, nil
	}
	// not in the cache, need to generate info for this type.
	typeCacheMutex.Lock()
	defer typeCacheMutex.Unlock()
	return cachedTypeInfo1(typ, tags)
}

func cachedTypeInfo1(typ reflect.Type, tags tags) (*typeinfo, error) {
	key := typekey{typ, tags}
	info := typeCache[key]
	if info != nil {
		// another goroutine got the write lock first
		return info, nil
	}
	// put a dummmy value into the cache before generating.
	// if the generator tries to lookup itself, it will get
	// the dummy value and won't call itself recursively.
	typeCache[key] = new(typeinfo)
	info, err := genTypeInfo(typ, tags)
	if err != nil {
		// remove the dummy value if the generator fails
		delete(typeCache, key)
		return nil, err
	}
	*typeCache[key] = *info
	return typeCache[key], err
}

type field struct {
	index int
	info  *typeinfo
}

func structFields(typ reflect.Type) (fields []field, err error) {
	for i := 0; i < typ.NumField(); i++ {
		if f := typ.Field(i); f.PkgPath == "" { // exported
			tags, err := parseStructTag(typ, i)
			if err != nil {
				return nil, err
			}
			if tags.ignored {
				continue
			}
			info, err := cachedTypeInfo1(f.Type, tags)
			if err != nil {
				return nil, err
			}
			fields = append(fields, field{i, info})
		}
	}
	return fields, nil
}

func parseStructTag(typ reflect.Type, fi int) (tags, error) {
	f := typ.Field(fi)
	var ts tags
	for _, t := range strings.Split(f.Tag.Get("rlp"), ",") {
		switch t = strings.TrimSpace(t); t {
		case "":
		case "-":
			ts.ignored = true
		case "nil":
			ts.nilOK = true
		case "tail":
			ts.tail = true
			if fi != typ.NumField()-1 {
				return ts, fmt.Errorf(`rlp: invalid struct tag "tail" for %v.%s (must be on last field)`, typ, f.Name)
			}
			if f.Type.Kind() != reflect.Slice {
				return ts, fmt.Errorf(`rlp: invalid struct tag "tail" for %v.%s (field type is not slice)`, typ, f.Name)
			}
		default:
			return ts, fmt.Errorf("rlp: unknown struct tag %q on %v.%s", t, typ, f.Name)
		}
	}
	return ts, nil
}

func genTypeInfo(typ reflect.Type, tags tags) (info *typeinfo, err error) {
	info = new(typeinfo)
	if info.decoder, err = makeDecoder(typ, tags); err != nil {
		return nil, err
	}
	if info.writer, err = makeWriter(typ, tags); err != nil {
		return nil, err
	}
	return info, nil
}

func isUint(k reflect.Kind) bool {
	return k >= reflect.Uint && k <= reflect.Uintptr
}
