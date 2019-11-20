package goja

import (
	"fmt"
	"go/ast"
	"reflect"
)

// JsonEncodable allows custom JSON encoding by JSON.stringify()
// Note that if the returned value itself also implements JsonEncodable, it won't have any effect.
type JsonEncodable interface {
	JsonEncodable() interface{}
}

// FieldNameMapper provides custom mapping between Go and JavaScript property names.
type FieldNameMapper interface {
	// FieldName returns a JavaScript name for the given struct field in the given type.
	// If this method returns "" the field becomes hidden.
	FieldName(t reflect.Type, f reflect.StructField) string

	// MethodName returns a JavaScript name for the given method in the given type.
	// If this method returns "" the method becomes hidden.
	MethodName(t reflect.Type, m reflect.Method) string
}

type reflectFieldInfo struct {
	Index     []int
	Anonymous bool
}

type reflectTypeInfo struct {
	Fields                  map[string]reflectFieldInfo
	Methods                 map[string]int
	FieldNames, MethodNames []string
}

type objectGoReflect struct {
	baseObject
	origValue, value reflect.Value

	valueTypeInfo, origValueTypeInfo *reflectTypeInfo

	toJson func() interface{}
}

func (o *objectGoReflect) init() {
	o.baseObject.init()
	switch o.value.Kind() {
	case reflect.Bool:
		o.class = classBoolean
		o.prototype = o.val.runtime.global.BooleanPrototype
	case reflect.String:
		o.class = classString
		o.prototype = o.val.runtime.global.StringPrototype
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64,
		reflect.Float32, reflect.Float64:

		o.class = classNumber
		o.prototype = o.val.runtime.global.NumberPrototype
	default:
		o.class = classObject
		o.prototype = o.val.runtime.global.ObjectPrototype
	}

	o.baseObject._putProp("toString", o.val.runtime.newNativeFunc(o.toStringFunc, nil, "toString", nil, 0), true, false, true)
	o.baseObject._putProp("valueOf", o.val.runtime.newNativeFunc(o.valueOfFunc, nil, "valueOf", nil, 0), true, false, true)

	o.valueTypeInfo = o.val.runtime.typeInfo(o.value.Type())
	o.origValueTypeInfo = o.val.runtime.typeInfo(o.origValue.Type())

	if j, ok := o.origValue.Interface().(JsonEncodable); ok {
		o.toJson = j.JsonEncodable
	}
}

func (o *objectGoReflect) toStringFunc(call FunctionCall) Value {
	return o.toPrimitiveString()
}

func (o *objectGoReflect) valueOfFunc(call FunctionCall) Value {
	return o.toPrimitive()
}

func (o *objectGoReflect) get(n Value) Value {
	return o.getStr(n.String())
}

func (o *objectGoReflect) _getField(jsName string) reflect.Value {
	if info, exists := o.valueTypeInfo.Fields[jsName]; exists {
		v := o.value.FieldByIndex(info.Index)
		if info.Anonymous {
			v = v.Addr()
		}
		return v
	}

	return reflect.Value{}
}

func (o *objectGoReflect) _getMethod(jsName string) reflect.Value {
	if idx, exists := o.origValueTypeInfo.Methods[jsName]; exists {
		return o.origValue.Method(idx)
	}

	return reflect.Value{}
}

func (o *objectGoReflect) _get(name string) Value {
	if o.value.Kind() == reflect.Struct {
		if v := o._getField(name); v.IsValid() {
			return o.val.runtime.ToValue(v.Interface())
		}
	}

	if v := o._getMethod(name); v.IsValid() {
		return o.val.runtime.ToValue(v.Interface())
	}

	return nil
}

func (o *objectGoReflect) getStr(name string) Value {
	if v := o._get(name); v != nil {
		return v
	}
	return o.baseObject._getStr(name)
}

func (o *objectGoReflect) getProp(n Value) Value {
	name := n.String()
	if p := o.getOwnProp(name); p != nil {
		return p
	}
	return o.baseObject.getOwnProp(name)
}

func (o *objectGoReflect) getPropStr(name string) Value {
	if v := o.getOwnProp(name); v != nil {
		return v
	}
	return o.baseObject.getPropStr(name)
}

func (o *objectGoReflect) getOwnProp(name string) Value {
	if o.value.Kind() == reflect.Struct {
		if v := o._getField(name); v.IsValid() {
			return &valueProperty{
				value:      o.val.runtime.ToValue(v.Interface()),
				writable:   v.CanSet(),
				enumerable: true,
			}
		}
	}

	if v := o._getMethod(name); v.IsValid() {
		return &valueProperty{
			value:      o.val.runtime.ToValue(v.Interface()),
			enumerable: true,
		}
	}

	return nil
}

func (o *objectGoReflect) put(n Value, val Value, throw bool) {
	o.putStr(n.String(), val, throw)
}

func (o *objectGoReflect) putStr(name string, val Value, throw bool) {
	if !o._put(name, val, throw) {
		o.val.runtime.typeErrorResult(throw, "Cannot assign to property %s of a host object", name)
	}
}

func (o *objectGoReflect) _put(name string, val Value, throw bool) bool {
	if o.value.Kind() == reflect.Struct {
		if v := o._getField(name); v.IsValid() {
			if !v.CanSet() {
				o.val.runtime.typeErrorResult(throw, "Cannot assign to a non-addressable or read-only property %s of a host object", name)
				return false
			}
			vv, err := o.val.runtime.toReflectValue(val, v.Type())
			if err != nil {
				o.val.runtime.typeErrorResult(throw, "Go struct conversion error: %v", err)
				return false
			}
			v.Set(vv)
			return true
		}
	}
	return false
}

func (o *objectGoReflect) _putProp(name string, value Value, writable, enumerable, configurable bool) Value {
	if o._put(name, value, false) {
		return value
	}
	return o.baseObject._putProp(name, value, writable, enumerable, configurable)
}

func (r *Runtime) checkHostObjectPropertyDescr(name string, descr propertyDescr, throw bool) bool {
	if descr.Getter != nil || descr.Setter != nil {
		r.typeErrorResult(throw, "Host objects do not support accessor properties")
		return false
	}
	if descr.Writable == FLAG_FALSE {
		r.typeErrorResult(throw, "Host object field %s cannot be made read-only", name)
		return false
	}
	if descr.Configurable == FLAG_TRUE {
		r.typeErrorResult(throw, "Host object field %s cannot be made configurable", name)
		return false
	}
	return true
}

func (o *objectGoReflect) defineOwnProperty(n Value, descr propertyDescr, throw bool) bool {
	if o.value.Kind() == reflect.Struct {
		name := n.String()
		if v := o._getField(name); v.IsValid() {
			if !o.val.runtime.checkHostObjectPropertyDescr(name, descr, throw) {
				return false
			}
			val := descr.Value
			if val == nil {
				val = _undefined
			}
			vv, err := o.val.runtime.toReflectValue(val, v.Type())
			if err != nil {
				o.val.runtime.typeErrorResult(throw, "Go struct conversion error: %v", err)
				return false
			}
			v.Set(vv)
			return true
		}
	}

	return o.baseObject.defineOwnProperty(n, descr, throw)
}

func (o *objectGoReflect) _has(name string) bool {
	if o.value.Kind() == reflect.Struct {
		if v := o._getField(name); v.IsValid() {
			return true
		}
	}
	if v := o._getMethod(name); v.IsValid() {
		return true
	}
	return false
}

func (o *objectGoReflect) hasProperty(n Value) bool {
	name := n.String()
	if o._has(name) {
		return true
	}
	return o.baseObject.hasProperty(n)
}

func (o *objectGoReflect) hasPropertyStr(name string) bool {
	if o._has(name) {
		return true
	}
	return o.baseObject.hasPropertyStr(name)
}

func (o *objectGoReflect) hasOwnProperty(n Value) bool {
	return o._has(n.String())
}

func (o *objectGoReflect) hasOwnPropertyStr(name string) bool {
	return o._has(name)
}

func (o *objectGoReflect) _toNumber() Value {
	switch o.value.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return intToValue(o.value.Int())
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return intToValue(int64(o.value.Uint()))
	case reflect.Bool:
		if o.value.Bool() {
			return intToValue(1)
		} else {
			return intToValue(0)
		}
	case reflect.Float32, reflect.Float64:
		return floatToValue(o.value.Float())
	}
	return nil
}

func (o *objectGoReflect) _toString() Value {
	switch o.value.Kind() {
	case reflect.String:
		return newStringValue(o.value.String())
	case reflect.Bool:
		if o.value.Interface().(bool) {
			return stringTrue
		} else {
			return stringFalse
		}
	}
	switch v := o.value.Interface().(type) {
	case fmt.Stringer:
		return newStringValue(v.String())
	}
	return stringObjectObject
}

func (o *objectGoReflect) toPrimitiveNumber() Value {
	if v := o._toNumber(); v != nil {
		return v
	}
	return o._toString()
}

func (o *objectGoReflect) toPrimitiveString() Value {
	if v := o._toNumber(); v != nil {
		return v.ToString()
	}
	return o._toString()
}

func (o *objectGoReflect) toPrimitive() Value {
	if o.prototype == o.val.runtime.global.NumberPrototype {
		return o.toPrimitiveNumber()
	}
	return o.toPrimitiveString()
}

func (o *objectGoReflect) deleteStr(name string, throw bool) bool {
	if o._has(name) {
		o.val.runtime.typeErrorResult(throw, "Cannot delete property %s from a Go type")
		return false
	}
	return o.baseObject.deleteStr(name, throw)
}

func (o *objectGoReflect) delete(name Value, throw bool) bool {
	return o.deleteStr(name.String(), throw)
}

type goreflectPropIter struct {
	o         *objectGoReflect
	idx       int
	recursive bool
}

func (i *goreflectPropIter) nextField() (propIterItem, iterNextFunc) {
	names := i.o.valueTypeInfo.FieldNames
	if i.idx < len(names) {
		name := names[i.idx]
		i.idx++
		return propIterItem{name: name, enumerable: _ENUM_TRUE}, i.nextField
	}

	i.idx = 0
	return i.nextMethod()
}

func (i *goreflectPropIter) nextMethod() (propIterItem, iterNextFunc) {
	names := i.o.origValueTypeInfo.MethodNames
	if i.idx < len(names) {
		name := names[i.idx]
		i.idx++
		return propIterItem{name: name, enumerable: _ENUM_TRUE}, i.nextMethod
	}

	if i.recursive {
		return i.o.baseObject._enumerate(true)()
	}

	return propIterItem{}, nil
}

func (o *objectGoReflect) _enumerate(recursive bool) iterNextFunc {
	r := &goreflectPropIter{
		o:         o,
		recursive: recursive,
	}
	if o.value.Kind() == reflect.Struct {
		return r.nextField
	}
	return r.nextMethod
}

func (o *objectGoReflect) enumerate(all, recursive bool) iterNextFunc {
	return (&propFilterIter{
		wrapped: o._enumerate(recursive),
		all:     all,
		seen:    make(map[string]bool),
	}).next
}

func (o *objectGoReflect) export() interface{} {
	return o.origValue.Interface()
}

func (o *objectGoReflect) exportType() reflect.Type {
	return o.origValue.Type()
}

func (o *objectGoReflect) equal(other objectImpl) bool {
	if other, ok := other.(*objectGoReflect); ok {
		return o.value.Interface() == other.value.Interface()
	}
	return false
}

func (r *Runtime) buildFieldInfo(t reflect.Type, index []int, info *reflectTypeInfo) {
	n := t.NumField()
	for i := 0; i < n; i++ {
		field := t.Field(i)
		name := field.Name
		if !ast.IsExported(name) {
			continue
		}
		if r.fieldNameMapper != nil {
			name = r.fieldNameMapper.FieldName(t, field)
		}

		if name != "" {
			if inf, exists := info.Fields[name]; !exists {
				info.FieldNames = append(info.FieldNames, name)
			} else {
				if len(inf.Index) <= len(index) {
					continue
				}
			}
		}

		if name != "" || field.Anonymous {
			idx := make([]int, len(index)+1)
			copy(idx, index)
			idx[len(idx)-1] = i

			if name != "" {
				info.Fields[name] = reflectFieldInfo{
					Index:     idx,
					Anonymous: field.Anonymous,
				}
			}
			if field.Anonymous {
				typ := field.Type
				for typ.Kind() == reflect.Ptr {
					typ = typ.Elem()
				}
				if typ.Kind() == reflect.Struct {
					r.buildFieldInfo(typ, idx, info)
				}
			}
		}
	}
}

func (r *Runtime) buildTypeInfo(t reflect.Type) (info *reflectTypeInfo) {
	info = new(reflectTypeInfo)
	if t.Kind() == reflect.Struct {
		info.Fields = make(map[string]reflectFieldInfo)
		n := t.NumField()
		info.FieldNames = make([]string, 0, n)
		r.buildFieldInfo(t, nil, info)
	}

	info.Methods = make(map[string]int)
	n := t.NumMethod()
	info.MethodNames = make([]string, 0, n)
	for i := 0; i < n; i++ {
		method := t.Method(i)
		name := method.Name
		if !ast.IsExported(name) {
			continue
		}
		if r.fieldNameMapper != nil {
			name = r.fieldNameMapper.MethodName(t, method)
			if name == "" {
				continue
			}
		}

		if _, exists := info.Methods[name]; !exists {
			info.MethodNames = append(info.MethodNames, name)
		}

		info.Methods[name] = i
	}
	return
}

func (r *Runtime) typeInfo(t reflect.Type) (info *reflectTypeInfo) {
	var exists bool
	if info, exists = r.typeInfoCache[t]; !exists {
		info = r.buildTypeInfo(t)
		if r.typeInfoCache == nil {
			r.typeInfoCache = make(map[reflect.Type]*reflectTypeInfo)
		}
		r.typeInfoCache[t] = info
	}

	return
}

// SetFieldNameMapper sets a custom field name mapper for Go types. It can be called at any time, however
// the mapping for any given value is fixed at the point of creation.
// Setting this to nil restores the default behaviour which is all exported fields and methods are mapped to their
// original unchanged names.
func (r *Runtime) SetFieldNameMapper(mapper FieldNameMapper) {
	r.fieldNameMapper = mapper
	r.typeInfoCache = nil
}
