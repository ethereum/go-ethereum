package goja

import "reflect"

const (
	classObject   = "Object"
	classArray    = "Array"
	classFunction = "Function"
	classNumber   = "Number"
	classString   = "String"
	classBoolean  = "Boolean"
	classError    = "Error"
	classRegExp   = "RegExp"
	classDate     = "Date"
)

type Object struct {
	runtime *Runtime
	self    objectImpl
}

type iterNextFunc func() (propIterItem, iterNextFunc)

type propertyDescr struct {
	Value Value

	Writable, Configurable, Enumerable Flag

	Getter, Setter Value
}

type objectImpl interface {
	sortable
	className() string
	get(Value) Value
	getProp(Value) Value
	getPropStr(string) Value
	getStr(string) Value
	getOwnProp(string) Value
	put(Value, Value, bool)
	putStr(string, Value, bool)
	hasProperty(Value) bool
	hasPropertyStr(string) bool
	hasOwnProperty(Value) bool
	hasOwnPropertyStr(string) bool
	_putProp(name string, value Value, writable, enumerable, configurable bool) Value
	defineOwnProperty(name Value, descr propertyDescr, throw bool) bool
	toPrimitiveNumber() Value
	toPrimitiveString() Value
	toPrimitive() Value
	assertCallable() (call func(FunctionCall) Value, ok bool)
	deleteStr(name string, throw bool) bool
	delete(name Value, throw bool) bool
	proto() *Object
	hasInstance(v Value) bool
	isExtensible() bool
	preventExtensions()
	enumerate(all, recusrive bool) iterNextFunc
	_enumerate(recursive bool) iterNextFunc
	export() interface{}
	exportType() reflect.Type
	equal(objectImpl) bool
}

type baseObject struct {
	class      string
	val        *Object
	prototype  *Object
	extensible bool

	values    map[string]Value
	propNames []string
}

type primitiveValueObject struct {
	baseObject
	pValue Value
}

func (o *primitiveValueObject) export() interface{} {
	return o.pValue.Export()
}

func (o *primitiveValueObject) exportType() reflect.Type {
	return o.pValue.ExportType()
}

type FunctionCall struct {
	This      Value
	Arguments []Value
}

type ConstructorCall struct {
	This      *Object
	Arguments []Value
}

func (f FunctionCall) Argument(idx int) Value {
	if idx < len(f.Arguments) {
		return f.Arguments[idx]
	}
	return _undefined
}

func (f ConstructorCall) Argument(idx int) Value {
	if idx < len(f.Arguments) {
		return f.Arguments[idx]
	}
	return _undefined
}

func (o *baseObject) init() {
	o.values = make(map[string]Value)
}

func (o *baseObject) className() string {
	return o.class
}

func (o *baseObject) getPropStr(name string) Value {
	if val := o.getOwnProp(name); val != nil {
		return val
	}
	if o.prototype != nil {
		return o.prototype.self.getPropStr(name)
	}
	return nil
}

func (o *baseObject) getProp(n Value) Value {
	return o.val.self.getPropStr(n.String())
}

func (o *baseObject) hasProperty(n Value) bool {
	return o.val.self.getProp(n) != nil
}

func (o *baseObject) hasPropertyStr(name string) bool {
	return o.val.self.getPropStr(name) != nil
}

func (o *baseObject) _getStr(name string) Value {
	p := o.getOwnProp(name)

	if p == nil && o.prototype != nil {
		p = o.prototype.self.getPropStr(name)
	}

	if p, ok := p.(*valueProperty); ok {
		return p.get(o.val)
	}

	return p
}

func (o *baseObject) getStr(name string) Value {
	p := o.val.self.getPropStr(name)
	if p, ok := p.(*valueProperty); ok {
		return p.get(o.val)
	}

	return p
}

func (o *baseObject) get(n Value) Value {
	return o.getStr(n.String())
}

func (o *baseObject) checkDeleteProp(name string, prop *valueProperty, throw bool) bool {
	if !prop.configurable {
		o.val.runtime.typeErrorResult(throw, "Cannot delete property '%s' of %s", name, o.val.ToString())
		return false
	}
	return true
}

func (o *baseObject) checkDelete(name string, val Value, throw bool) bool {
	if val, ok := val.(*valueProperty); ok {
		return o.checkDeleteProp(name, val, throw)
	}
	return true
}

func (o *baseObject) _delete(name string) {
	delete(o.values, name)
	for i, n := range o.propNames {
		if n == name {
			copy(o.propNames[i:], o.propNames[i+1:])
			o.propNames = o.propNames[:len(o.propNames)-1]
			break
		}
	}
}

func (o *baseObject) deleteStr(name string, throw bool) bool {
	if val, exists := o.values[name]; exists {
		if !o.checkDelete(name, val, throw) {
			return false
		}
		o._delete(name)
		return true
	}
	return true
}

func (o *baseObject) delete(n Value, throw bool) bool {
	return o.deleteStr(n.String(), throw)
}

func (o *baseObject) put(n Value, val Value, throw bool) {
	o.putStr(n.String(), val, throw)
}

func (o *baseObject) getOwnProp(name string) Value {
	v := o.values[name]
	if v == nil && name == "__proto" {
		return o.prototype
	}
	return v
}

func (o *baseObject) putStr(name string, val Value, throw bool) {
	if v, exists := o.values[name]; exists {
		if prop, ok := v.(*valueProperty); ok {
			if !prop.isWritable() {
				o.val.runtime.typeErrorResult(throw, "Cannot assign to read only property '%s'", name)
				return
			}
			prop.set(o.val, val)
			return
		}
		o.values[name] = val
		return
	}

	if name == "__proto__" {
		if !o.extensible {
			o.val.runtime.typeErrorResult(throw, "%s is not extensible", o.val)
			return
		}
		if val == _undefined || val == _null {
			o.prototype = nil
			return
		} else {
			if val, ok := val.(*Object); ok {
				o.prototype = val
			}
		}
		return
	}

	var pprop Value
	if proto := o.prototype; proto != nil {
		pprop = proto.self.getPropStr(name)
	}

	if pprop != nil {
		if prop, ok := pprop.(*valueProperty); ok {
			if !prop.isWritable() {
				o.val.runtime.typeErrorResult(throw)
				return
			}
			if prop.accessor {
				prop.set(o.val, val)
				return
			}
		}
	} else {
		if !o.extensible {
			o.val.runtime.typeErrorResult(throw)
			return
		}
	}

	o.values[name] = val
	o.propNames = append(o.propNames, name)
}

func (o *baseObject) hasOwnProperty(n Value) bool {
	v := o.values[n.String()]
	return v != nil
}

func (o *baseObject) hasOwnPropertyStr(name string) bool {
	v := o.values[name]
	return v != nil
}

func (o *baseObject) _defineOwnProperty(name, existingValue Value, descr propertyDescr, throw bool) (val Value, ok bool) {

	getterObj, _ := descr.Getter.(*Object)
	setterObj, _ := descr.Setter.(*Object)

	var existing *valueProperty

	if existingValue == nil {
		if !o.extensible {
			o.val.runtime.typeErrorResult(throw)
			return nil, false
		}
		existing = &valueProperty{}
	} else {
		if existing, ok = existingValue.(*valueProperty); !ok {
			existing = &valueProperty{
				writable:     true,
				enumerable:   true,
				configurable: true,
				value:        existingValue,
			}
		}

		if !existing.configurable {
			if descr.Configurable == FLAG_TRUE {
				goto Reject
			}
			if descr.Enumerable != FLAG_NOT_SET && descr.Enumerable.Bool() != existing.enumerable {
				goto Reject
			}
		}
		if existing.accessor && descr.Value != nil || !existing.accessor && (getterObj != nil || setterObj != nil) {
			if !existing.configurable {
				goto Reject
			}
		} else if !existing.accessor {
			if !existing.configurable {
				if !existing.writable {
					if descr.Writable == FLAG_TRUE {
						goto Reject
					}
					if descr.Value != nil && !descr.Value.SameAs(existing.value) {
						goto Reject
					}
				}
			}
		} else {
			if !existing.configurable {
				if descr.Getter != nil && existing.getterFunc != getterObj || descr.Setter != nil && existing.setterFunc != setterObj {
					goto Reject
				}
			}
		}
	}

	if descr.Writable == FLAG_TRUE && descr.Enumerable == FLAG_TRUE && descr.Configurable == FLAG_TRUE && descr.Value != nil {
		return descr.Value, true
	}

	if descr.Writable != FLAG_NOT_SET {
		existing.writable = descr.Writable.Bool()
	}
	if descr.Enumerable != FLAG_NOT_SET {
		existing.enumerable = descr.Enumerable.Bool()
	}
	if descr.Configurable != FLAG_NOT_SET {
		existing.configurable = descr.Configurable.Bool()
	}

	if descr.Value != nil {
		existing.value = descr.Value
		existing.getterFunc = nil
		existing.setterFunc = nil
	}

	if descr.Value != nil || descr.Writable != FLAG_NOT_SET {
		existing.accessor = false
	}

	if descr.Getter != nil {
		existing.getterFunc = propGetter(o.val, descr.Getter, o.val.runtime)
		existing.value = nil
		existing.accessor = true
	}

	if descr.Setter != nil {
		existing.setterFunc = propSetter(o.val, descr.Setter, o.val.runtime)
		existing.value = nil
		existing.accessor = true
	}

	if !existing.accessor && existing.value == nil {
		existing.value = _undefined
	}

	return existing, true

Reject:
	o.val.runtime.typeErrorResult(throw, "Cannot redefine property: %s", name.ToString())
	return nil, false

}

func (o *baseObject) defineOwnProperty(n Value, descr propertyDescr, throw bool) bool {
	name := n.String()
	existingVal := o.values[name]
	if v, ok := o._defineOwnProperty(n, existingVal, descr, throw); ok {
		o.values[name] = v
		if existingVal == nil {
			o.propNames = append(o.propNames, name)
		}
		return true
	}
	return false
}

func (o *baseObject) _put(name string, v Value) {
	if _, exists := o.values[name]; !exists {
		o.propNames = append(o.propNames, name)
	}

	o.values[name] = v
}

func (o *baseObject) _putProp(name string, value Value, writable, enumerable, configurable bool) Value {
	if writable && enumerable && configurable {
		o._put(name, value)
		return value
	} else {
		p := &valueProperty{
			value:        value,
			writable:     writable,
			enumerable:   enumerable,
			configurable: configurable,
		}
		o._put(name, p)
		return p
	}
}

func (o *baseObject) tryPrimitive(methodName string) Value {
	if method, ok := o.getStr(methodName).(*Object); ok {
		if call, ok := method.self.assertCallable(); ok {
			v := call(FunctionCall{
				This: o.val,
			})
			if _, fail := v.(*Object); !fail {
				return v
			}
		}
	}
	return nil
}

func (o *baseObject) toPrimitiveNumber() Value {
	if v := o.tryPrimitive("valueOf"); v != nil {
		return v
	}

	if v := o.tryPrimitive("toString"); v != nil {
		return v
	}

	o.val.runtime.typeErrorResult(true, "Could not convert %v to primitive", o)
	return nil
}

func (o *baseObject) toPrimitiveString() Value {
	if v := o.tryPrimitive("toString"); v != nil {
		return v
	}

	if v := o.tryPrimitive("valueOf"); v != nil {
		return v
	}

	o.val.runtime.typeErrorResult(true, "Could not convert %v to primitive", o)
	return nil
}

func (o *baseObject) toPrimitive() Value {
	return o.toPrimitiveNumber()
}

func (o *baseObject) assertCallable() (func(FunctionCall) Value, bool) {
	return nil, false
}

func (o *baseObject) proto() *Object {
	return o.prototype
}

func (o *baseObject) isExtensible() bool {
	return o.extensible
}

func (o *baseObject) preventExtensions() {
	o.extensible = false
}

func (o *baseObject) sortLen() int64 {
	return toLength(o.val.self.getStr("length"))
}

func (o *baseObject) sortGet(i int64) Value {
	return o.val.self.get(intToValue(i))
}

func (o *baseObject) swap(i, j int64) {
	ii := intToValue(i)
	jj := intToValue(j)

	x := o.val.self.get(ii)
	y := o.val.self.get(jj)

	o.val.self.put(ii, y, false)
	o.val.self.put(jj, x, false)
}

func (o *baseObject) export() interface{} {
	m := make(map[string]interface{})

	for item, f := o.enumerate(false, false)(); f != nil; item, f = f() {
		v := item.value
		if v == nil {
			v = o.getStr(item.name)
		}
		if v != nil {
			m[item.name] = v.Export()
		} else {
			m[item.name] = nil
		}
	}
	return m
}

func (o *baseObject) exportType() reflect.Type {
	return reflectTypeMap
}

type enumerableFlag int

const (
	_ENUM_UNKNOWN enumerableFlag = iota
	_ENUM_FALSE
	_ENUM_TRUE
)

type propIterItem struct {
	name       string
	value      Value // set only when enumerable == _ENUM_UNKNOWN
	enumerable enumerableFlag
}

type objectPropIter struct {
	o         *baseObject
	propNames []string
	recursive bool
	idx       int
}

type propFilterIter struct {
	wrapped iterNextFunc
	all     bool
	seen    map[string]bool
}

func (i *propFilterIter) next() (propIterItem, iterNextFunc) {
	for {
		var item propIterItem
		item, i.wrapped = i.wrapped()
		if i.wrapped == nil {
			return propIterItem{}, nil
		}

		if !i.seen[item.name] {
			i.seen[item.name] = true
			if !i.all {
				if item.enumerable == _ENUM_FALSE {
					continue
				}
				if item.enumerable == _ENUM_UNKNOWN {
					if prop, ok := item.value.(*valueProperty); ok {
						if !prop.enumerable {
							continue
						}
					}
				}
			}
			return item, i.next
		}
	}
}

func (i *objectPropIter) next() (propIterItem, iterNextFunc) {
	for i.idx < len(i.propNames) {
		name := i.propNames[i.idx]
		i.idx++
		prop := i.o.values[name]
		if prop != nil {
			return propIterItem{name: name, value: prop}, i.next
		}
	}

	if i.recursive && i.o.prototype != nil {
		return i.o.prototype.self._enumerate(i.recursive)()
	}
	return propIterItem{}, nil
}

func (o *baseObject) _enumerate(recursive bool) iterNextFunc {
	propNames := make([]string, len(o.propNames))
	copy(propNames, o.propNames)
	return (&objectPropIter{
		o:         o,
		propNames: propNames,
		recursive: recursive,
	}).next
}

func (o *baseObject) enumerate(all, recursive bool) iterNextFunc {
	return (&propFilterIter{
		wrapped: o._enumerate(recursive),
		all:     all,
		seen:    make(map[string]bool),
	}).next
}

func (o *baseObject) equal(other objectImpl) bool {
	// Rely on parent reference comparison
	return false
}

func (o *baseObject) hasInstance(v Value) bool {
	o.val.runtime.typeErrorResult(true, "Expecting a function in instanceof check, but got %s", o.val.ToString())
	panic("Unreachable")
}
