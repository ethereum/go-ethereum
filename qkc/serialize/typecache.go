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

// encodesZeroBytes reports whether a value of typ, serialized as a list element
// (default tags, i.e. no nil marker), can encode to zero bytes — e.g. struct{},
// a struct whose fields are all ignored/unexported, a zero-length array, or a
// pointer/array/struct recursively composed of those. genTypeInfo rejects slices
// of such elements: they carry no per-element byte cost, which both makes the
// bytes-remaining bound in deserializeList reject a valid round-trip and leaves
// that bound unable to limit allocation.
//
// It mirrors makeSerializer's dispatch and must stay lock-free (no cachedTypeInfo
// / structFields), as it runs inside genTypeInfo while typeCacheMutex is held.
// The visited set breaks reference cycles, conservatively treating an in-progress
// type as non-zero. Serializable types are assumed non-zero, since their custom
// encoding can't be inspected statically (QKC's all write >= 1 byte).
func encodesZeroBytes(typ reflect.Type, visited map[reflect.Type]bool) bool {
	if visited[typ] {
		return false
	}
	// Mark typ only for the current recursion path and unmark on return, so the
	// set tracks ancestors, not every type ever seen. Without the delete, the
	// same zero-byte type used as repeated sibling fields (e.g.
	// struct{ A empty; B empty }) would hit a stale mark on its second occurrence
	// and be misclassified as non-zero, breaking serialize/deserialize symmetry.
	// A type still on the path is a genuine cycle, kept conservatively as non-zero.
	visited[typ] = true
	defer delete(visited, typ)

	switch {
	case typ.Kind() == reflect.Ptr:
		// A list element pointer carries no nil marker, so it costs exactly what
		// its element costs.
		return encodesZeroBytes(typ.Elem(), visited)
	case reflect.PtrTo(typ).Implements(serializableInterface):
		return false
	case typ.AssignableTo(bigInt):
		return false
	case isUint(typ.Kind()):
		return false
	case typ.Kind() == reflect.Bool:
		return false
	case typ.Kind() == reflect.String:
		return false
	case typ.Kind() == reflect.Slice:
		// A slice always writes a length prefix (>= 1 byte).
		return false
	case typ.Kind() == reflect.Array:
		if typ.Len() == 0 {
			return true
		}
		if isByte(typ.Elem()) {
			return false
		}
		return encodesZeroBytes(typ.Elem(), visited)
	case typ.Kind() == reflect.Struct:
		for i := 0; i < typ.NumField(); i++ {
			f := typ.Field(i)
			if f.PkgPath != "" { // unexported fields are skipped by the codec
				continue
			}
			tags, err := parseStructTag(typ, i)
			if err != nil || tags.Ignored {
				continue
			}
			if tags.NilOK {
				return false // a nil marker byte is always written
			}
			if !encodesZeroBytes(f.Type, visited) {
				return false
			}
		}
		return true
	default:
		return false
	}
}

func genTypeInfo(typ reflect.Type) (info *typeinfo, err error) {
	info = new(typeinfo)
	if info.serializer, err = makeSerializer(typ); err != nil {
		return nil, err
	}
	if info.deserializer, err = makeDeserializer(typ); err != nil {
		return nil, err
	}
	// A slice whose element encodes to zero bytes is unsupported: such a list
	// serializes to just its length prefix, so deserializeList's bytes-remaining
	// bound would reject the valid round-trip (and could not bound allocation
	// anyway). Reject it symmetrically here so serialize and deserialize agree.
	// Byte slices are exempt (a byte is one byte); arrays are exempt (their length
	// is fixed by the type, not read from input, so there is nothing to bound).
	if typ.Kind() == reflect.Slice && !isByte(typ.Elem()) && encodesZeroBytes(typ.Elem(), make(map[reflect.Type]bool)) {
		return nil, fmt.Errorf("ser: list element type %v encodes to zero bytes, which is unsupported", typ.Elem())
	}
	return info, nil
}
