// Ported verbatim from github.com/QuarkChain/goquarkchain/serialize (byte-compatible).
// Modified from go-ethereum under GNU Lesser General Public License

package serialize

import (
	"fmt"
	"reflect"
	"strconv"
	"strings"
	"sync"
)

var (
	typeCacheMutex   sync.RWMutex
	typeCache        = make(map[reflect.Type]*typeinfo)
	fieldsCacheMutex sync.RWMutex
	fieldsCache      = make(map[reflect.Type][]field)
)

type typeinfo struct {
	serializer
	deserializer
}

// represents struct Tags
type Tags struct {
	// ser:"nil" controls whether empty input results in a nil pointer.
	NilOK bool
	// ser:"-" ignores fields.
	Ignored bool
	// bytesizeofslicelen: number
	ByteSizeOfSliceLen int
}

type deserializer func(*ByteBuffer, reflect.Value, Tags) error

type serializer func(reflect.Value, *[]byte, Tags) error

func cachedTypeInfo(typ reflect.Type) (*typeinfo, error) {
	typeCacheMutex.RLock()
	info := typeCache[typ]
	typeCacheMutex.RUnlock()
	if info != nil {
		return info, nil
	}

	typeCacheMutex.Lock()
	defer typeCacheMutex.Unlock()

	info, err := genTypeInfo(typ)
	if err != nil {
		return nil, err
	}

	typeCache[typ] = info
	return typeCache[typ], err
}

type field struct {
	index int
	info  *typeinfo
	tags  Tags
	name  string
}

func structFields(typ reflect.Type) (fields []field, err error) {
	fieldsCacheMutex.RLock()
	flds := fieldsCache[typ]
	fieldsCacheMutex.RUnlock()
	if flds != nil {
		return flds, nil
	}
	for i := 0; i < typ.NumField(); i++ {
		if f := typ.Field(i); f.PkgPath == "" { // exported
			tags, err := parseStructTag(typ, i)
			if err != nil {
				return nil, err
			}
			if tags.Ignored {
				continue
			}
			info, err := cachedTypeInfo(f.Type)
			if err != nil {
				return nil, err
			}
			fields = append(fields, field{i, info, tags, f.Name})
		}
	}
	fieldsCacheMutex.Lock()
	defer fieldsCacheMutex.Unlock()
	fieldsCache[typ] = fields

	return fields, nil
}

func parseStructTag(typ reflect.Type, fi int) (Tags, error) {
	f := typ.Field(fi)
	var ts Tags
	ts.ByteSizeOfSliceLen = 1
	for _, t := range strings.Split(f.Tag.Get("ser"), ",") {
		switch t = strings.TrimSpace(t); t {
		case "":
		case "-":
			ts.Ignored = true
		case "nil": // nil equal to optional in PyQuackChain
			ts.NilOK = true
		default:
			return ts, fmt.Errorf("ser: unknown struct tag %q on %v.%s", t, typ, f.Name)
		}
	}
	// bytesizeofslicelen use to specify the number of bytes used to save a slice len
	// only slice is useful
	if f.Type.Kind() == reflect.Slice {
		for _, t := range strings.Split(f.Tag.Get("bytesizeofslicelen"), ",") {
			t = strings.TrimSpace(t)
			if t != "" {
				num, err := strconv.Atoi(t)
				if err != nil {
					return ts, err
				}

				ts.ByteSizeOfSliceLen = num
			}
		}
	}

	return ts, nil
}

func genTypeInfo(typ reflect.Type) (info *typeinfo, err error) {
	info = new(typeinfo)
	if info.serializer, err = makeSerializer(typ); err != nil {
		return nil, err
	}
	if info.deserializer, err = makeDeserializer(typ); err != nil {
		return nil, err
	}
	return info, nil
}
