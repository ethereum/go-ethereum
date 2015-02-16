package qml

// #include <stdlib.h>
// #include "capi.h"
import "C"

import (
	"bytes"
	"fmt"
	"image/color"
	"reflect"
	"strings"
	"unicode"
	"unsafe"
)

var (
	intIs64 bool
	intDT   C.DataType

	ptrSize = C.size_t(unsafe.Sizeof(uintptr(0)))

	nilPtr     = unsafe.Pointer(uintptr(0))
	nilCharPtr = (*C.char)(nilPtr)

	typeString     = reflect.TypeOf("")
	typeBool       = reflect.TypeOf(false)
	typeInt        = reflect.TypeOf(int(0))
	typeInt64      = reflect.TypeOf(int64(0))
	typeInt32      = reflect.TypeOf(int32(0))
	typeFloat64    = reflect.TypeOf(float64(0))
	typeFloat32    = reflect.TypeOf(float32(0))
	typeIface      = reflect.TypeOf(new(interface{})).Elem()
	typeRGBA       = reflect.TypeOf(color.RGBA{})
	typeObjSlice   = reflect.TypeOf([]Object(nil))
	typeObject     = reflect.TypeOf([]Object(nil)).Elem()
	typePainter    = reflect.TypeOf(&Painter{})
	typeList       = reflect.TypeOf(&List{})
	typeMap        = reflect.TypeOf(&Map{})
	typeGenericMap = reflect.TypeOf(map[string]interface{}(nil))
)

func init() {
	var i int = 1<<31 - 1
	intIs64 = (i+1 > 0)
	if intIs64 {
		intDT = C.DTInt64
	} else {
		intDT = C.DTInt32
	}
}

// packDataValue packs the provided Go value into a C.DataValue for
// shiping into C++ land.
//
// For simple types (bool, int, etc) value is converted into a
// native C++ value. For anything else, including cases when value
// has a type that has an underlying simple type, the Go value itself
// is encapsulated into a C++ wrapper so that field access and method
// calls work.
//
// This must be run from the main GUI thread due to the cases where
// calling wrapGoValue is necessary.
func packDataValue(value interface{}, dvalue *C.DataValue, engine *Engine, owner valueOwner) {
	datap := unsafe.Pointer(&dvalue.data)
	if value == nil {
		dvalue.dataType = C.DTInvalid
		return
	}
	switch value := value.(type) {
	case string:
		dvalue.dataType = C.DTString
		cstr, cstrlen := unsafeStringData(value)
		*(**C.char)(datap) = cstr
		dvalue.len = cstrlen
	case bool:
		dvalue.dataType = C.DTBool
		*(*bool)(datap) = value
	case int:
		if value > 1<<31-1 {
			dvalue.dataType = C.DTInt64
			*(*int64)(datap) = int64(value)
		} else {
			dvalue.dataType = C.DTInt32
			*(*int32)(datap) = int32(value)
		}
	case int64:
		dvalue.dataType = C.DTInt64
		*(*int64)(datap) = value
	case int32:
		dvalue.dataType = C.DTInt32
		*(*int32)(datap) = value
	case uint64:
		dvalue.dataType = C.DTUint64
		*(*uint64)(datap) = value
	case uint32:
		dvalue.dataType = C.DTUint32
		*(*uint32)(datap) = value
	case float64:
		dvalue.dataType = C.DTFloat64
		*(*float64)(datap) = value
	case float32:
		dvalue.dataType = C.DTFloat32
		*(*float32)(datap) = value
	case *Common:
		dvalue.dataType = C.DTObject
		*(*unsafe.Pointer)(datap) = value.addr
	case color.RGBA:
		dvalue.dataType = C.DTColor
		*(*uint32)(datap) = uint32(value.A)<<24 | uint32(value.R)<<16 | uint32(value.G)<<8 | uint32(value.B)
	default:
		dvalue.dataType = C.DTObject
		if obj, ok := value.(Object); ok {
			*(*unsafe.Pointer)(datap) = obj.Common().addr
		} else {
			*(*unsafe.Pointer)(datap) = wrapGoValue(engine, value, owner)
		}
	}
}

// TODO Handle byte slices.

// unpackDataValue converts a value shipped by C++ into a native Go value.
//
// HEADS UP: This is considered safe to be run out of the main GUI thread.
//           If that changes, fix the call sites.
func unpackDataValue(dvalue *C.DataValue, engine *Engine) interface{} {
	datap := unsafe.Pointer(&dvalue.data)
	switch dvalue.dataType {
	case C.DTString:
		s := C.GoStringN(*(**C.char)(datap), dvalue.len)
		// TODO If we move all unpackDataValue calls to the GUI thread,
		// can we get rid of this allocation somehow?
		C.free(unsafe.Pointer(*(**C.char)(datap)))
		return s
	case C.DTBool:
		return *(*bool)(datap)
	case C.DTInt64:
		return *(*int64)(datap)
	case C.DTInt32:
		return int(*(*int32)(datap))
	case C.DTUint64:
		return *(*uint64)(datap)
	case C.DTUint32:
		return *(*uint32)(datap)
	case C.DTUintptr:
		return *(*uintptr)(datap)
	case C.DTFloat64:
		return *(*float64)(datap)
	case C.DTFloat32:
		return *(*float32)(datap)
	case C.DTColor:
		var c uint32 = *(*uint32)(datap)
		return color.RGBA{byte(c >> 16), byte(c >> 8), byte(c), byte(c >> 24)}
	case C.DTGoAddr:
		// ObjectByName also does this fold conversion, to have access
		// to the cvalue. Perhaps the fold should be returned.
		fold := (*(**valueFold)(datap))
		ensureEngine(engine.addr, unsafe.Pointer(fold))
		return fold.gvalue
	case C.DTInvalid:
		return nil
	case C.DTObject:
		// TODO Would be good to preserve identity on the Go side. See initGoType as well.
		obj := &Common{
			engine: engine,
			addr:   *(*unsafe.Pointer)(datap),
		}
		if len(converters) > 0 {
			// TODO Embed the type name in DataValue to drop these calls.
			typeName := obj.TypeName()
			if typeName == "PlainObject" {
				typeName = strings.TrimRight(obj.String("plainType"), "&*")
				if strings.HasPrefix(typeName, "const ") {
					typeName = typeName[6:]
				}
			}
			if f, ok := converters[typeName]; ok {
				return f(engine, obj)
			}
		}
		return obj
	case C.DTValueList, C.DTValueMap:
		var dvlist []C.DataValue
		var dvlisth = (*reflect.SliceHeader)(unsafe.Pointer(&dvlist))
		dvlisth.Data = uintptr(*(*unsafe.Pointer)(datap))
		dvlisth.Len = int(dvalue.len)
		dvlisth.Cap = int(dvalue.len)
		result := make([]interface{}, len(dvlist))
		for i := range result {
			result[i] = unpackDataValue(&dvlist[i], engine)
		}
		C.free(*(*unsafe.Pointer)(datap))
		if dvalue.dataType == C.DTValueList {
			return &List{result}
		} else {
			return &Map{result}
		}
	}
	panic(fmt.Sprintf("unsupported data type: %d", dvalue.dataType))
}

func dataTypeOf(typ reflect.Type) C.DataType {
	// Compare against the specific types rather than their kind.
	// Custom types may have methods that must be supported.
	switch typ {
	case typeString:
		return C.DTString
	case typeBool:
		return C.DTBool
	case typeInt:
		return intDT
	case typeInt64:
		return C.DTInt64
	case typeInt32:
		return C.DTInt32
	case typeFloat32:
		return C.DTFloat32
	case typeFloat64:
		return C.DTFloat64
	case typeIface:
		return C.DTAny
	case typeRGBA:
		return C.DTColor
	case typeObjSlice:
		return C.DTListProperty
	}
	return C.DTObject
}

var typeInfoSize = C.size_t(unsafe.Sizeof(C.GoTypeInfo{}))
var memberInfoSize = C.size_t(unsafe.Sizeof(C.GoMemberInfo{}))

var typeInfoCache = make(map[reflect.Type]*C.GoTypeInfo)

func appendLoweredName(buf []byte, name string) []byte {
	var last rune
	var lasti int
	for i, rune := range name {
		if !unicode.IsUpper(rune) {
			if lasti == 0 {
				last = unicode.ToLower(last)
			}
			buf = append(buf, string(last)...)
			buf = append(buf, name[i:]...)
			return buf
		}
		if i > 0 {
			buf = append(buf, string(unicode.ToLower(last))...)
		}
		lasti, last = i, rune
	}
	return append(buf, string(unicode.ToLower(last))...)
}

func typeInfo(v interface{}) *C.GoTypeInfo {
	vt := reflect.TypeOf(v)
	for vt.Kind() == reflect.Ptr {
		vt = vt.Elem()
	}

	typeInfo := typeInfoCache[vt]
	if typeInfo != nil {
		return typeInfo
	}

	typeInfo = (*C.GoTypeInfo)(C.malloc(typeInfoSize))
	typeInfo.typeName = C.CString(vt.Name())
	typeInfo.metaObject = nilPtr
	typeInfo.paint = (*C.GoMemberInfo)(nilPtr)

	var setters map[string]int
	var getters map[string]int

	// TODO Only do that if it's a struct?
	vtptr := reflect.PtrTo(vt)

	if vt.Kind() != reflect.Struct {
		panic(fmt.Sprintf("handling of %s (%#v) is incomplete; please report to the developers", vt, v))
	}

	numField := vt.NumField()
	numMethod := vtptr.NumMethod()
	privateFields := 0
	privateMethods := 0

	// struct { FooBar T; Baz T } => "fooBar\0baz\0"
	namesLen := 0
	for i := 0; i < numField; i++ {
		field := vt.Field(i)
		if field.PkgPath != "" {
			privateFields++
			continue
		}
		namesLen += len(field.Name) + 1
	}
	for i := 0; i < numMethod; i++ {
		method := vtptr.Method(i)
		if method.PkgPath != "" {
			privateMethods++
			continue
		}
		namesLen += len(method.Name) + 1

		// Track setters and getters.
		if len(method.Name) > 3 && method.Name[:3] == "Set" {
			if method.Type.NumIn() == 2 {
				if setters == nil {
					setters = make(map[string]int)
				}
				setters[method.Name[3:]] = i
			}
		} else if method.Type.NumIn() == 1 && method.Type.NumOut() == 1 {
			if getters == nil {
				getters = make(map[string]int)
			}
			getters[method.Name] = i
		}
	}
	names := make([]byte, 0, namesLen)
	for i := 0; i < numField; i++ {
		field := vt.Field(i)
		if field.PkgPath != "" {
			continue // not exported
		}
		names = appendLoweredName(names, field.Name)
		names = append(names, 0)
	}
	for i := 0; i < numMethod; i++ {
		method := vtptr.Method(i)
		if method.PkgPath != "" {
			continue // not exported
		}
		if _, ok := getters[method.Name]; !ok {
			continue
		}
		if _, ok := setters[method.Name]; !ok {
			delete(getters, method.Name)
			continue
		}
		// This is a getter method
		names = appendLoweredName(names, method.Name)
		names = append(names, 0)
	}
	for i := 0; i < numMethod; i++ {
		method := vtptr.Method(i)
		if method.PkgPath != "" {
			continue // not exported
		}
		if _, ok := getters[method.Name]; ok {
			continue // getter already handled above
		}
		names = appendLoweredName(names, method.Name)
		names = append(names, 0)
	}
	if len(names) != namesLen {
		panic("pre-allocated buffer size was wrong")
	}
	typeInfo.memberNames = C.CString(string(names))

	// Assemble information on members.
	membersLen := numField - privateFields + numMethod - privateMethods
	membersi := uintptr(0)
	mnamesi := uintptr(0)
	members := uintptr(C.malloc(memberInfoSize * C.size_t(membersLen)))
	mnames := uintptr(unsafe.Pointer(typeInfo.memberNames))
	for i := 0; i < numField; i++ {
		field := vt.Field(i)
		if field.PkgPath != "" {
			continue // not exported
		}
		memberInfo := (*C.GoMemberInfo)(unsafe.Pointer(members + uintptr(memberInfoSize)*membersi))
		memberInfo.memberName = (*C.char)(unsafe.Pointer(mnames + mnamesi))
		memberInfo.memberType = dataTypeOf(field.Type)
		memberInfo.reflectIndex = C.int(i)
		memberInfo.reflectGetIndex = -1
		memberInfo.reflectSetIndex = -1
		memberInfo.addrOffset = C.int(field.Offset)
		membersi += 1
		mnamesi += uintptr(len(field.Name)) + 1
		if methodIndex, ok := setters[field.Name]; ok {
			memberInfo.reflectSetIndex = C.int(methodIndex)
		}
	}
	for i := 0; i < numMethod; i++ {
		method := vtptr.Method(i)
		if method.PkgPath != "" {
			continue // not exported
		}
		if _, ok := getters[method.Name]; !ok {
			continue // not a getter
		}
		memberInfo := (*C.GoMemberInfo)(unsafe.Pointer(members + uintptr(memberInfoSize)*membersi))
		memberInfo.memberName = (*C.char)(unsafe.Pointer(mnames + mnamesi))
		memberInfo.memberType = dataTypeOf(method.Type.Out(0))
		memberInfo.reflectIndex = -1
		memberInfo.reflectGetIndex = C.int(getters[method.Name])
		memberInfo.reflectSetIndex = C.int(setters[method.Name])
		memberInfo.addrOffset = 0
		membersi += 1
		mnamesi += uintptr(len(method.Name)) + 1
	}
	for i := 0; i < numMethod; i++ {
		method := vtptr.Method(i)
		if method.PkgPath != "" {
			continue // not exported
		}
		if _, ok := getters[method.Name]; ok {
			continue // getter already handled above
		}
		memberInfo := (*C.GoMemberInfo)(unsafe.Pointer(members + uintptr(memberInfoSize)*membersi))
		memberInfo.memberName = (*C.char)(unsafe.Pointer(mnames + mnamesi))
		memberInfo.memberType = C.DTMethod
		memberInfo.reflectIndex = C.int(i)
		memberInfo.reflectGetIndex = -1
		memberInfo.reflectSetIndex = -1
		memberInfo.addrOffset = 0
		signature, result := methodQtSignature(method)
		// TODO The signature data might be embedded in the same array as the member names.
		memberInfo.methodSignature = C.CString(signature)
		memberInfo.resultSignature = C.CString(result)
		// TODO Sort out methods with a variable number of arguments.
		// It's called while bound, so drop the receiver.
		memberInfo.numIn = C.int(method.Type.NumIn() - 1)
		memberInfo.numOut = C.int(method.Type.NumOut())
		membersi += 1
		mnamesi += uintptr(len(method.Name)) + 1

		if method.Name == "Paint" && memberInfo.numIn == 1 && memberInfo.numOut == 0 && method.Type.In(1) == typePainter {
			typeInfo.paint = memberInfo
		}
	}
	typeInfo.members = (*C.GoMemberInfo)(unsafe.Pointer(members))
	typeInfo.membersLen = C.int(membersLen)

	typeInfo.fields = typeInfo.members
	typeInfo.fieldsLen = C.int(numField - privateFields + len(getters))
	typeInfo.methods = (*C.GoMemberInfo)(unsafe.Pointer(members + uintptr(memberInfoSize)*uintptr(typeInfo.fieldsLen)))
	typeInfo.methodsLen = C.int(numMethod - privateMethods - len(getters))

	if int(membersi) != membersLen {
		panic("used more space than allocated for member names")
	}
	if int(mnamesi) != namesLen {
		panic("allocated buffer doesn't match used space")
	}
	if typeInfo.fieldsLen+typeInfo.methodsLen != typeInfo.membersLen {
		panic("lengths are inconsistent")
	}

	typeInfoCache[vt] = typeInfo
	return typeInfo
}

func methodQtSignature(method reflect.Method) (signature, result string) {
	var buf bytes.Buffer
	for i, rune := range method.Name {
		if i == 0 {
			buf.WriteRune(unicode.ToLower(rune))
		} else {
			buf.WriteString(method.Name[i:])
			break
		}
	}
	buf.WriteByte('(')
	n := method.Type.NumIn()
	for i := 1; i < n; i++ {
		if i > 1 {
			buf.WriteByte(',')
		}
		buf.WriteString("QVariant")
	}
	buf.WriteByte(')')
	signature = buf.String()

	switch method.Type.NumOut() {
	case 0:
		// keep it as ""
	case 1:
		result = "QVariant"
	default:
		result = "QVariantList"
	}
	return
}

func hashable(value interface{}) (hashable bool) {
	defer func() { recover() }()
	return value == value
}

// unsafeString returns a Go string backed by C data.
//
// If the C data is deallocated or moved, the string will be
// invalid and will crash the program if used. As such, the
// resulting string must only be used inside the implementation
// of the qml package and while the life time of the C data
// is guaranteed.
func unsafeString(data *C.char, size C.int) string {
	var s string
	sh := (*reflect.StringHeader)(unsafe.Pointer(&s))
	sh.Data = uintptr(unsafe.Pointer(data))
	sh.Len = int(size)
	return s
}

// unsafeStringData returns a C string backed by Go data. The C
// string is NOT null-terminated, so its length must be taken
// into account.
//
// If the s Go string is garbage collected, the returned C data
// will be invalid and will crash the program if used. As such,
// the resulting data must only be used inside the implementation
// of the qml package and while the life time of the Go string
// is guaranteed.
func unsafeStringData(s string) (*C.char, C.int) {
	return *(**C.char)(unsafe.Pointer(&s)), C.int(len(s))
}

// unsafeBytesData returns a C string backed by Go data. The C
// string is NOT null-terminated, so its length must be taken
// into account.
//
// If the array backing the b Go slice is garbage collected, the
// returned C data will be invalid and will crash the program if
// used. As such, the resulting data must only be used inside the
// implementation of the qml package and while the life time of
// the Go array is guaranteed.
func unsafeBytesData(b []byte) (*C.char, C.int) {
	return *(**C.char)(unsafe.Pointer(&b)), C.int(len(b))
}
