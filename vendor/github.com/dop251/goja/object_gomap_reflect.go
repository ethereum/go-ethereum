package goja

import "reflect"

type objectGoMapReflect struct {
	objectGoReflect

	keyType, valueType reflect.Type
}

func (o *objectGoMapReflect) init() {
	o.objectGoReflect.init()
	o.keyType = o.value.Type().Key()
	o.valueType = o.value.Type().Elem()
}

func (o *objectGoMapReflect) toKey(n Value) reflect.Value {
	key, err := o.val.runtime.toReflectValue(n, o.keyType)
	if err != nil {
		o.val.runtime.typeErrorResult(true, "map key conversion error: %v", err)
		panic("unreachable")
	}
	return key
}

func (o *objectGoMapReflect) strToKey(name string) reflect.Value {
	if o.keyType.Kind() == reflect.String {
		return reflect.ValueOf(name).Convert(o.keyType)
	}
	return o.toKey(newStringValue(name))
}

func (o *objectGoMapReflect) _get(n Value) Value {
	if v := o.value.MapIndex(o.toKey(n)); v.IsValid() {
		return o.val.runtime.ToValue(v.Interface())
	}

	return nil
}

func (o *objectGoMapReflect) _getStr(name string) Value {
	if v := o.value.MapIndex(o.strToKey(name)); v.IsValid() {
		return o.val.runtime.ToValue(v.Interface())
	}

	return nil
}

func (o *objectGoMapReflect) get(n Value) Value {
	if v := o._get(n); v != nil {
		return v
	}
	return o.objectGoReflect.get(n)
}

func (o *objectGoMapReflect) getStr(name string) Value {
	if v := o._getStr(name); v != nil {
		return v
	}
	return o.objectGoReflect.getStr(name)
}

func (o *objectGoMapReflect) getProp(n Value) Value {
	return o.get(n)
}

func (o *objectGoMapReflect) getPropStr(name string) Value {
	return o.getStr(name)
}

func (o *objectGoMapReflect) getOwnProp(name string) Value {
	if v := o._getStr(name); v != nil {
		return &valueProperty{
			value:      v,
			writable:   true,
			enumerable: true,
		}
	}
	return o.objectGoReflect.getOwnProp(name)
}

func (o *objectGoMapReflect) toValue(val Value, throw bool) (reflect.Value, bool) {
	v, err := o.val.runtime.toReflectValue(val, o.valueType)
	if err != nil {
		o.val.runtime.typeErrorResult(throw, "map value conversion error: %v", err)
		return reflect.Value{}, false
	}

	return v, true
}

func (o *objectGoMapReflect) put(key, val Value, throw bool) {
	k := o.toKey(key)
	v, ok := o.toValue(val, throw)
	if !ok {
		return
	}
	o.value.SetMapIndex(k, v)
}

func (o *objectGoMapReflect) putStr(name string, val Value, throw bool) {
	k := o.strToKey(name)
	v, ok := o.toValue(val, throw)
	if !ok {
		return
	}
	o.value.SetMapIndex(k, v)
}

func (o *objectGoMapReflect) _putProp(name string, value Value, writable, enumerable, configurable bool) Value {
	o.putStr(name, value, true)
	return value
}

func (o *objectGoMapReflect) defineOwnProperty(n Value, descr propertyDescr, throw bool) bool {
	name := n.String()
	if !o.val.runtime.checkHostObjectPropertyDescr(name, descr, throw) {
		return false
	}

	o.put(n, descr.Value, throw)
	return true
}

func (o *objectGoMapReflect) hasOwnPropertyStr(name string) bool {
	return o.value.MapIndex(o.strToKey(name)).IsValid()
}

func (o *objectGoMapReflect) hasOwnProperty(n Value) bool {
	return o.value.MapIndex(o.toKey(n)).IsValid()
}

func (o *objectGoMapReflect) hasProperty(n Value) bool {
	if o.hasOwnProperty(n) {
		return true
	}
	return o.objectGoReflect.hasProperty(n)
}

func (o *objectGoMapReflect) hasPropertyStr(name string) bool {
	if o.hasOwnPropertyStr(name) {
		return true
	}
	return o.objectGoReflect.hasPropertyStr(name)
}

func (o *objectGoMapReflect) delete(n Value, throw bool) bool {
	o.value.SetMapIndex(o.toKey(n), reflect.Value{})
	return true
}

func (o *objectGoMapReflect) deleteStr(name string, throw bool) bool {
	o.value.SetMapIndex(o.strToKey(name), reflect.Value{})
	return true
}

type gomapReflectPropIter struct {
	o         *objectGoMapReflect
	keys      []reflect.Value
	idx       int
	recursive bool
}

func (i *gomapReflectPropIter) next() (propIterItem, iterNextFunc) {
	for i.idx < len(i.keys) {
		key := i.keys[i.idx]
		v := i.o.value.MapIndex(key)
		i.idx++
		if v.IsValid() {
			return propIterItem{name: key.String(), enumerable: _ENUM_TRUE}, i.next
		}
	}

	if i.recursive {
		return i.o.objectGoReflect._enumerate(true)()
	}

	return propIterItem{}, nil
}

func (o *objectGoMapReflect) _enumerate(recusrive bool) iterNextFunc {
	r := &gomapReflectPropIter{
		o:         o,
		keys:      o.value.MapKeys(),
		recursive: recusrive,
	}
	return r.next
}

func (o *objectGoMapReflect) enumerate(all, recursive bool) iterNextFunc {
	return (&propFilterIter{
		wrapped: o._enumerate(recursive),
		all:     all,
		seen:    make(map[string]bool),
	}).next
}

func (o *objectGoMapReflect) equal(other objectImpl) bool {
	if other, ok := other.(*objectGoMapReflect); ok {
		return o.value.Interface() == other.value.Interface()
	}
	return false
}
