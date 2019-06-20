package goja

import (
	"reflect"
	"strconv"
)

type objectGoMapSimple struct {
	baseObject
	data map[string]interface{}
}

func (o *objectGoMapSimple) init() {
	o.baseObject.init()
	o.prototype = o.val.runtime.global.ObjectPrototype
	o.class = classObject
	o.extensible = true
}

func (o *objectGoMapSimple) _get(n Value) Value {
	return o._getStr(n.String())
}

func (o *objectGoMapSimple) _getStr(name string) Value {
	v, exists := o.data[name]
	if !exists {
		return nil
	}
	return o.val.runtime.ToValue(v)
}

func (o *objectGoMapSimple) get(n Value) Value {
	return o.getStr(n.String())
}

func (o *objectGoMapSimple) getProp(n Value) Value {
	return o.getPropStr(n.String())
}

func (o *objectGoMapSimple) getPropStr(name string) Value {
	if v := o._getStr(name); v != nil {
		return v
	}
	return o.baseObject.getPropStr(name)
}

func (o *objectGoMapSimple) getStr(name string) Value {
	if v := o._getStr(name); v != nil {
		return v
	}
	return o.baseObject._getStr(name)
}

func (o *objectGoMapSimple) getOwnProp(name string) Value {
	if v := o._getStr(name); v != nil {
		return v
	}
	return o.baseObject.getOwnProp(name)
}

func (o *objectGoMapSimple) put(n Value, val Value, throw bool) {
	o.putStr(n.String(), val, throw)
}

func (o *objectGoMapSimple) _hasStr(name string) bool {
	_, exists := o.data[name]
	return exists
}

func (o *objectGoMapSimple) _has(n Value) bool {
	return o._hasStr(n.String())
}

func (o *objectGoMapSimple) putStr(name string, val Value, throw bool) {
	if o.extensible || o._hasStr(name) {
		o.data[name] = val.Export()
	} else {
		o.val.runtime.typeErrorResult(throw, "Host object is not extensible")
	}
}

func (o *objectGoMapSimple) hasProperty(n Value) bool {
	if o._has(n) {
		return true
	}
	return o.baseObject.hasProperty(n)
}

func (o *objectGoMapSimple) hasPropertyStr(name string) bool {
	if o._hasStr(name) {
		return true
	}
	return o.baseObject.hasOwnPropertyStr(name)
}

func (o *objectGoMapSimple) hasOwnProperty(n Value) bool {
	return o._has(n)
}

func (o *objectGoMapSimple) hasOwnPropertyStr(name string) bool {
	return o._hasStr(name)
}

func (o *objectGoMapSimple) _putProp(name string, value Value, writable, enumerable, configurable bool) Value {
	o.putStr(name, value, false)
	return value
}

func (o *objectGoMapSimple) defineOwnProperty(name Value, descr propertyDescr, throw bool) bool {
	if descr.Getter != nil || descr.Setter != nil {
		o.val.runtime.typeErrorResult(throw, "Host objects do not support accessor properties")
		return false
	}
	o.put(name, descr.Value, throw)
	return true
}

/*
func (o *objectGoMapSimple) toPrimitiveNumber() Value {
	return o.toPrimitiveString()
}

func (o *objectGoMapSimple) toPrimitiveString() Value {
	return stringObjectObject
}

func (o *objectGoMapSimple) toPrimitive() Value {
	return o.toPrimitiveString()
}

func (o *objectGoMapSimple) assertCallable() (call func(FunctionCall) Value, ok bool) {
	return nil, false
}
*/

func (o *objectGoMapSimple) deleteStr(name string, throw bool) bool {
	delete(o.data, name)
	return true
}

func (o *objectGoMapSimple) delete(name Value, throw bool) bool {
	return o.deleteStr(name.String(), throw)
}

type gomapPropIter struct {
	o         *objectGoMapSimple
	propNames []string
	recursive bool
	idx       int
}

func (i *gomapPropIter) next() (propIterItem, iterNextFunc) {
	for i.idx < len(i.propNames) {
		name := i.propNames[i.idx]
		i.idx++
		if _, exists := i.o.data[name]; exists {
			return propIterItem{name: name, enumerable: _ENUM_TRUE}, i.next
		}
	}

	if i.recursive {
		return i.o.prototype.self._enumerate(true)()
	}

	return propIterItem{}, nil
}

func (o *objectGoMapSimple) enumerate(all, recursive bool) iterNextFunc {
	return (&propFilterIter{
		wrapped: o._enumerate(recursive),
		all:     all,
		seen:    make(map[string]bool),
	}).next
}

func (o *objectGoMapSimple) _enumerate(recursive bool) iterNextFunc {
	propNames := make([]string, len(o.data))
	i := 0
	for key, _ := range o.data {
		propNames[i] = key
		i++
	}
	return (&gomapPropIter{
		o:         o,
		propNames: propNames,
		recursive: recursive,
	}).next
}

func (o *objectGoMapSimple) export() interface{} {
	return o.data
}

func (o *objectGoMapSimple) exportType() reflect.Type {
	return reflectTypeMap
}

func (o *objectGoMapSimple) equal(other objectImpl) bool {
	if other, ok := other.(*objectGoMapSimple); ok {
		return o == other
	}
	return false
}

func (o *objectGoMapSimple) sortLen() int64 {
	return int64(len(o.data))
}

func (o *objectGoMapSimple) sortGet(i int64) Value {
	return o.getStr(strconv.FormatInt(i, 10))
}

func (o *objectGoMapSimple) swap(i, j int64) {
	ii := strconv.FormatInt(i, 10)
	jj := strconv.FormatInt(j, 10)
	x := o.getStr(ii)
	y := o.getStr(jj)

	o.putStr(ii, y, false)
	o.putStr(jj, x, false)
}
