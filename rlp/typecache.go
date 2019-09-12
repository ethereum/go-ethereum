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
	decoder    decoder
	decoderErr error // error from makeDecoder
	writer     writer
	writerErr  error // error from makeWriter
}

// tags represents struct tags.
type tags struct {
	// rlp:"nil" controls whether empty input results in a nil pointer.
	nilOK bool

	// This controls whether nil pointers are encoded/decoded as empty strings
	// or empty lists.
	nilKind Kind

	// rlp:"tail" controls whether this field swallows additional list
	// elements. It can only be set for the last field, which must be
	// of slice type.
	tail bool

	// rlp:"-" ignores fields.
	ignored bool
}

// typekey is the key of a type in typeCache. It includes the struct tags because
// they might generate a different decoder.
type typekey struct {
	reflect.Type
	tags
}

type decoder func(*Stream, reflect.Value) error

type writer func(reflect.Value, *encbuf) error

func cachedDecoder(typ reflect.Type) (decoder, error) {
	info := cachedTypeInfo(typ, tags{})
	return info.decoder, info.decoderErr
}

func cachedWriter(typ reflect.Type) (writer, error) {
	info := cachedTypeInfo(typ, tags{})
	return info.writer, info.writerErr
}

func cachedTypeInfo(typ reflect.Type, tags tags) *typeinfo {
	typeCacheMutex.RLock()
	info := typeCache[typekey{typ, tags}]
	typeCacheMutex.RUnlock()
	if info != nil {
		return info
	}
	// not in the cache, need to generate info for this type.
	typeCacheMutex.Lock()
	defer typeCacheMutex.Unlock()
	return cachedTypeInfo1(typ, tags)
}

func cachedTypeInfo1(typ reflect.Type, tags tags) *typeinfo {
	key := typekey{typ, tags}
	info := typeCache[key]
	if info != nil {
		// another goroutine got the write lock first
		return info
	}
	// put a dummy value into the cache before generating.
	// if the generator tries to lookup itself, it will get
	// the dummy value and won't call itself recursively.
	info = new(typeinfo)
	typeCache[key] = info
	info.generate(typ, tags)
	return info
}

type field struct {
	index int
	info  *typeinfo
}

func structFields(typ reflect.Type) (fields []field, err error) {
	lastPublic := lastPublicField(typ)
	for i := 0; i < typ.NumField(); i++ {
		if f := typ.Field(i); f.PkgPath == "" { // exported
			tags, err := parseStructTag(typ, i, lastPublic)
			if err != nil {
				return nil, err
			}
			if tags.ignored {
				continue
			}
			info := cachedTypeInfo1(f.Type, tags)
			fields = append(fields, field{i, info})
		}
	}
	return fields, nil
}

type structFieldError struct {
	typ   reflect.Type
	field int
	err   error
}

func (e structFieldError) Error() string {
	return fmt.Sprintf("%v (struct field %v.%s)", e.err, e.typ, e.typ.Field(e.field).Name)
}

type structTagError struct {
	typ             reflect.Type
	field, tag, err string
}

func (e structTagError) Error() string {
	return fmt.Sprintf("rlp: invalid struct tag %q for %v.%s (%s)", e.tag, e.typ, e.field, e.err)
}

func parseStructTag(typ reflect.Type, fi, lastPublic int) (tags, error) {
	f := typ.Field(fi)
	var ts tags
	for _, t := range strings.Split(f.Tag.Get("rlp"), ",") {
		switch t = strings.TrimSpace(t); t {
		case "":
		case "-":
			ts.ignored = true
		case "nil", "nilString", "nilList":
			ts.nilOK = true
			if f.Type.Kind() != reflect.Ptr {
				return ts, structTagError{typ, f.Name, t, "field is not a pointer"}
			}
			switch t {
			case "nil":
				ts.nilKind = defaultNilKind(f.Type.Elem())
			case "nilString":
				ts.nilKind = String
			case "nilList":
				ts.nilKind = List
			}
		case "tail":
			ts.tail = true
			if fi != lastPublic {
				return ts, structTagError{typ, f.Name, t, "must be on last field"}
			}
			if f.Type.Kind() != reflect.Slice {
				return ts, structTagError{typ, f.Name, t, "field type is not slice"}
			}
		default:
			return ts, fmt.Errorf("rlp: unknown struct tag %q on %v.%s", t, typ, f.Name)
		}
	}
	return ts, nil
}

func lastPublicField(typ reflect.Type) int {
	last := 0
	for i := 0; i < typ.NumField(); i++ {
		if typ.Field(i).PkgPath == "" {
			last = i
		}
	}
	return last
}

func (i *typeinfo) generate(typ reflect.Type, tags tags) {
	i.decoder, i.decoderErr = makeDecoder(typ, tags)
	i.writer, i.writerErr = makeWriter(typ, tags)
}

// defaultNilKind determines whether a nil pointer to typ encodes/decodes
// as an empty string or empty list.
func defaultNilKind(typ reflect.Type) Kind {
	k := typ.Kind()
	if isUint(k) || k == reflect.String || k == reflect.Bool || isByteArray(typ) {
		return String
	}
	return List
}

func isUint(k reflect.Kind) bool {
	return k >= reflect.Uint && k <= reflect.Uintptr
}

func isByteArray(typ reflect.Type) bool {
	return (typ.Kind() == reflect.Slice || typ.Kind() == reflect.Array) && isByte(typ.Elem())
}
