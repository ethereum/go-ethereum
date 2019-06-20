package goja

import (
	"reflect"
	"strconv"
)

type objectGoSliceReflect struct {
	objectGoReflect
	lengthProp valueProperty
}

func (o *objectGoSliceReflect) init() {
	o.objectGoReflect.init()
	o.class = classArray
	o.prototype = o.val.runtime.global.ArrayPrototype
	o.lengthProp.writable = false
	o._setLen()
	o.baseObject._put("length", &o.lengthProp)
}

func (o *objectGoSliceReflect) _setLen() {
	o.lengthProp.value = intToValue(int64(o.value.Len()))
}

func (o *objectGoSliceReflect) _has(n Value) bool {
	if idx := toIdx(n); idx >= 0 {
		return idx < int64(o.value.Len())
	}
	return false
}

func (o *objectGoSliceReflect) _hasStr(name string) bool {
	if idx := strToIdx(name); idx >= 0 {
		return idx < int64(o.value.Len())
	}
	return false
}

func (o *objectGoSliceReflect) getIdx(idx int64) Value {
	if idx < int64(o.value.Len()) {
		return o.val.runtime.ToValue(o.value.Index(int(idx)).Interface())
	}
	return nil
}

func (o *objectGoSliceReflect) _get(n Value) Value {
	if idx := toIdx(n); idx >= 0 {
		return o.getIdx(idx)
	}
	return nil
}

func (o *objectGoSliceReflect) _getStr(name string) Value {
	if idx := strToIdx(name); idx >= 0 {
		return o.getIdx(idx)
	}
	return nil
}

func (o *objectGoSliceReflect) get(n Value) Value {
	if v := o._get(n); v != nil {
		return v
	}
	return o.objectGoReflect.get(n)
}

func (o *objectGoSliceReflect) getProp(n Value) Value {
	if v := o._get(n); v != nil {
		return v
	}
	return o.objectGoReflect.getProp(n)
}

func (o *objectGoSliceReflect) getPropStr(name string) Value {
	if v := o._getStr(name); v != nil {
		return v
	}
	return o.objectGoReflect.getPropStr(name)
}

func (o *objectGoSliceReflect) getOwnProp(name string) Value {
	if v := o._getStr(name); v != nil {
		return v
	}
	return o.objectGoReflect.getOwnProp(name)
}

func (o *objectGoSliceReflect) putIdx(idx int64, v Value, throw bool) {
	if idx >= int64(o.value.Len()) {
		o.val.runtime.typeErrorResult(throw, "Cannot extend a Go reflect slice")
		return
	}
	val, err := o.val.runtime.toReflectValue(v, o.value.Type().Elem())
	if err != nil {
		o.val.runtime.typeErrorResult(throw, "Go type conversion error: %v", err)
		return
	}
	o.value.Index(int(idx)).Set(val)
}

func (o *objectGoSliceReflect) put(n Value, val Value, throw bool) {
	if idx := toIdx(n); idx >= 0 {
		o.putIdx(idx, val, throw)
		return
	}
	// TODO: length
	o.objectGoReflect.put(n, val, throw)
}

func (o *objectGoSliceReflect) putStr(name string, val Value, throw bool) {
	if idx := strToIdx(name); idx >= 0 {
		o.putIdx(idx, val, throw)
		return
	}
	// TODO: length
	o.objectGoReflect.putStr(name, val, throw)
}

func (o *objectGoSliceReflect) hasProperty(n Value) bool {
	if o._has(n) {
		return true
	}
	return o.objectGoReflect.hasProperty(n)
}

func (o *objectGoSliceReflect) hasPropertyStr(name string) bool {
	if o._hasStr(name) {
		return true
	}
	return o.objectGoReflect.hasOwnPropertyStr(name)
}

func (o *objectGoSliceReflect) hasOwnProperty(n Value) bool {
	if o._has(n) {
		return true
	}
	return o.objectGoReflect.hasOwnProperty(n)
}

func (o *objectGoSliceReflect) hasOwnPropertyStr(name string) bool {
	if o._hasStr(name) {
		return true
	}
	return o.objectGoReflect.hasOwnPropertyStr(name)
}

func (o *objectGoSliceReflect) _putProp(name string, value Value, writable, enumerable, configurable bool) Value {
	o.putStr(name, value, false)
	return value
}

func (o *objectGoSliceReflect) defineOwnProperty(name Value, descr propertyDescr, throw bool) bool {
	if !o.val.runtime.checkHostObjectPropertyDescr(name.String(), descr, throw) {
		return false
	}
	o.put(name, descr.Value, throw)
	return true
}

func (o *objectGoSliceReflect) toPrimitiveNumber() Value {
	return o.toPrimitiveString()
}

func (o *objectGoSliceReflect) toPrimitiveString() Value {
	return o.val.runtime.arrayproto_join(FunctionCall{
		This: o.val,
	})
}

func (o *objectGoSliceReflect) toPrimitive() Value {
	return o.toPrimitiveString()
}

func (o *objectGoSliceReflect) deleteStr(name string, throw bool) bool {
	if idx := strToIdx(name); idx >= 0 && idx < int64(o.value.Len()) {
		o.value.Index(int(idx)).Set(reflect.Zero(o.value.Type().Elem()))
		return true
	}
	return o.objectGoReflect.deleteStr(name, throw)
}

func (o *objectGoSliceReflect) delete(name Value, throw bool) bool {
	if idx := toIdx(name); idx >= 0 && idx < int64(o.value.Len()) {
		o.value.Index(int(idx)).Set(reflect.Zero(o.value.Type().Elem()))
		return true
	}
	return true
}

type gosliceReflectPropIter struct {
	o          *objectGoSliceReflect
	recursive  bool
	idx, limit int
}

func (i *gosliceReflectPropIter) next() (propIterItem, iterNextFunc) {
	if i.idx < i.limit && i.idx < i.o.value.Len() {
		name := strconv.Itoa(i.idx)
		i.idx++
		return propIterItem{name: name, enumerable: _ENUM_TRUE}, i.next
	}

	if i.recursive {
		return i.o.prototype.self._enumerate(i.recursive)()
	}

	return propIterItem{}, nil
}

func (o *objectGoSliceReflect) enumerate(all, recursive bool) iterNextFunc {
	return (&propFilterIter{
		wrapped: o._enumerate(recursive),
		all:     all,
		seen:    make(map[string]bool),
	}).next
}

func (o *objectGoSliceReflect) _enumerate(recursive bool) iterNextFunc {
	return (&gosliceReflectPropIter{
		o:         o,
		recursive: recursive,
		limit:     o.value.Len(),
	}).next
}

func (o *objectGoSliceReflect) equal(other objectImpl) bool {
	if other, ok := other.(*objectGoSliceReflect); ok {
		return o.value.Interface() == other.value.Interface()
	}
	return false
}

func (o *objectGoSliceReflect) sortLen() int64 {
	return int64(o.value.Len())
}

func (o *objectGoSliceReflect) sortGet(i int64) Value {
	return o.get(intToValue(i))
}

func (o *objectGoSliceReflect) swap(i, j int64) {
	ii := intToValue(i)
	jj := intToValue(j)
	x := o.get(ii)
	y := o.get(jj)

	o.put(ii, y, false)
	o.put(jj, x, false)
}
