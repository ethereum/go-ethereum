package goja

import (
	"fmt"
)

func (r *Runtime) builtin_Object(args []Value, proto *Object) *Object {
	if len(args) > 0 {
		arg := args[0]
		if arg != _undefined && arg != _null {
			return arg.ToObject(r)
		}
	}
	return r.NewObject()
}

func (r *Runtime) object_getPrototypeOf(call FunctionCall) Value {
	o := call.Argument(0).ToObject(r)
	p := o.self.proto()
	if p == nil {
		return _null
	}
	return p
}

func (r *Runtime) object_getOwnPropertyDescriptor(call FunctionCall) Value {
	obj := call.Argument(0).ToObject(r)
	propName := call.Argument(1).String()
	desc := obj.self.getOwnProp(propName)
	if desc == nil {
		return _undefined
	}
	var writable, configurable, enumerable, accessor bool
	var get, set *Object
	var value Value
	if v, ok := desc.(*valueProperty); ok {
		writable = v.writable
		configurable = v.configurable
		enumerable = v.enumerable
		accessor = v.accessor
		value = v.value
		get = v.getterFunc
		set = v.setterFunc
	} else {
		writable = true
		configurable = true
		enumerable = true
		value = desc
	}

	ret := r.NewObject()
	o := ret.self
	if !accessor {
		o.putStr("value", value, false)
		o.putStr("writable", r.toBoolean(writable), false)
	} else {
		if get != nil {
			o.putStr("get", get, false)
		} else {
			o.putStr("get", _undefined, false)
		}
		if set != nil {
			o.putStr("set", set, false)
		} else {
			o.putStr("set", _undefined, false)
		}
	}
	o.putStr("enumerable", r.toBoolean(enumerable), false)
	o.putStr("configurable", r.toBoolean(configurable), false)

	return ret
}

func (r *Runtime) object_getOwnPropertyNames(call FunctionCall) Value {
	// ES6
	obj := call.Argument(0).ToObject(r)
	// obj := r.toObject(call.Argument(0))

	var values []Value
	for item, f := obj.self.enumerate(true, false)(); f != nil; item, f = f() {
		values = append(values, newStringValue(item.name))
	}
	return r.newArrayValues(values)
}

func (r *Runtime) toPropertyDescr(v Value) (ret propertyDescr) {
	if o, ok := v.(*Object); ok {
		descr := o.self

		ret.Value = descr.getStr("value")

		if p := descr.getStr("writable"); p != nil {
			ret.Writable = ToFlag(p.ToBoolean())
		}
		if p := descr.getStr("enumerable"); p != nil {
			ret.Enumerable = ToFlag(p.ToBoolean())
		}
		if p := descr.getStr("configurable"); p != nil {
			ret.Configurable = ToFlag(p.ToBoolean())
		}

		ret.Getter = descr.getStr("get")
		ret.Setter = descr.getStr("set")

		if ret.Getter != nil && ret.Getter != _undefined {
			if _, ok := r.toObject(ret.Getter).self.assertCallable(); !ok {
				r.typeErrorResult(true, "getter must be a function")
			}
		}

		if ret.Setter != nil && ret.Setter != _undefined {
			if _, ok := r.toObject(ret.Setter).self.assertCallable(); !ok {
				r.typeErrorResult(true, "setter must be a function")
			}
		}

		if (ret.Getter != nil || ret.Setter != nil) && (ret.Value != nil || ret.Writable != FLAG_NOT_SET) {
			r.typeErrorResult(true, "Invalid property descriptor. Cannot both specify accessors and a value or writable attribute")
			return
		}
	} else {
		r.typeErrorResult(true, "Property description must be an object: %s", v.String())
	}

	return
}

func (r *Runtime) _defineProperties(o *Object, p Value) {
	type propItem struct {
		name string
		prop propertyDescr
	}
	props := p.ToObject(r)
	var list []propItem
	for item, f := props.self.enumerate(false, false)(); f != nil; item, f = f() {
		list = append(list, propItem{
			name: item.name,
			prop: r.toPropertyDescr(props.self.getStr(item.name)),
		})
	}
	for _, prop := range list {
		o.self.defineOwnProperty(newStringValue(prop.name), prop.prop, true)
	}
}

func (r *Runtime) object_create(call FunctionCall) Value {
	var proto *Object
	if arg := call.Argument(0); arg != _null {
		if o, ok := arg.(*Object); ok {
			proto = o
		} else {
			r.typeErrorResult(true, "Object prototype may only be an Object or null: %s", arg.String())
		}
	}
	o := r.newBaseObject(proto, classObject).val

	if props := call.Argument(1); props != _undefined {
		r._defineProperties(o, props)
	}

	return o
}

func (r *Runtime) object_defineProperty(call FunctionCall) (ret Value) {
	if obj, ok := call.Argument(0).(*Object); ok {
		descr := r.toPropertyDescr(call.Argument(2))
		obj.self.defineOwnProperty(call.Argument(1), descr, true)
		ret = call.Argument(0)
	} else {
		r.typeErrorResult(true, "Object.defineProperty called on non-object")
	}
	return
}

func (r *Runtime) object_defineProperties(call FunctionCall) Value {
	obj := r.toObject(call.Argument(0))
	r._defineProperties(obj, call.Argument(1))
	return obj
}

func (r *Runtime) object_seal(call FunctionCall) Value {
	// ES6
	arg := call.Argument(0)
	if obj, ok := arg.(*Object); ok {
		descr := propertyDescr{
			Writable:     FLAG_TRUE,
			Enumerable:   FLAG_TRUE,
			Configurable: FLAG_FALSE,
		}
		for item, f := obj.self.enumerate(true, false)(); f != nil; item, f = f() {
			v := obj.self.getOwnProp(item.name)
			if prop, ok := v.(*valueProperty); ok {
				if !prop.configurable {
					continue
				}
				prop.configurable = false
			} else {
				descr.Value = v
				obj.self.defineOwnProperty(newStringValue(item.name), descr, true)
				//obj.self._putProp(item.name, v, true, true, false)
			}
		}
		obj.self.preventExtensions()
		return obj
	}
	return arg
}

func (r *Runtime) object_freeze(call FunctionCall) Value {
	arg := call.Argument(0)
	if obj, ok := arg.(*Object); ok {
		descr := propertyDescr{
			Writable:     FLAG_FALSE,
			Enumerable:   FLAG_TRUE,
			Configurable: FLAG_FALSE,
		}
		for item, f := obj.self.enumerate(true, false)(); f != nil; item, f = f() {
			v := obj.self.getOwnProp(item.name)
			if prop, ok := v.(*valueProperty); ok {
				prop.configurable = false
				if prop.value != nil {
					prop.writable = false
				}
			} else {
				descr.Value = v
				obj.self.defineOwnProperty(newStringValue(item.name), descr, true)
			}
		}
		obj.self.preventExtensions()
		return obj
	} else {
		// ES6 behavior
		return arg
	}
}

func (r *Runtime) object_preventExtensions(call FunctionCall) (ret Value) {
	arg := call.Argument(0)
	if obj, ok := arg.(*Object); ok {
		obj.self.preventExtensions()
		return obj
	}
	// ES6
	//r.typeErrorResult(true, "Object.preventExtensions called on non-object")
	//panic("Unreachable")
	return arg
}

func (r *Runtime) object_isSealed(call FunctionCall) Value {
	if obj, ok := call.Argument(0).(*Object); ok {
		if obj.self.isExtensible() {
			return valueFalse
		}
		for item, f := obj.self.enumerate(true, false)(); f != nil; item, f = f() {
			prop := obj.self.getOwnProp(item.name)
			if prop, ok := prop.(*valueProperty); ok {
				if prop.configurable {
					return valueFalse
				}
			} else {
				return valueFalse
			}
		}
	} else {
		// ES6
		//r.typeErrorResult(true, "Object.isSealed called on non-object")
		return valueTrue
	}
	return valueTrue
}

func (r *Runtime) object_isFrozen(call FunctionCall) Value {
	if obj, ok := call.Argument(0).(*Object); ok {
		if obj.self.isExtensible() {
			return valueFalse
		}
		for item, f := obj.self.enumerate(true, false)(); f != nil; item, f = f() {
			prop := obj.self.getOwnProp(item.name)
			if prop, ok := prop.(*valueProperty); ok {
				if prop.configurable || prop.value != nil && prop.writable {
					return valueFalse
				}
			} else {
				return valueFalse
			}
		}
	} else {
		// ES6
		//r.typeErrorResult(true, "Object.isFrozen called on non-object")
		return valueTrue
	}
	return valueTrue
}

func (r *Runtime) object_isExtensible(call FunctionCall) Value {
	if obj, ok := call.Argument(0).(*Object); ok {
		if obj.self.isExtensible() {
			return valueTrue
		}
		return valueFalse
	} else {
		// ES6
		//r.typeErrorResult(true, "Object.isExtensible called on non-object")
		return valueFalse
	}
}

func (r *Runtime) object_keys(call FunctionCall) Value {
	// ES6
	obj := call.Argument(0).ToObject(r)
	//if obj, ok := call.Argument(0).(*valueObject); ok {
	var keys []Value
	for item, f := obj.self.enumerate(false, false)(); f != nil; item, f = f() {
		keys = append(keys, newStringValue(item.name))
	}
	return r.newArrayValues(keys)
	//} else {
	//	r.typeErrorResult(true, "Object.keys called on non-object")
	//}
	//return nil
}

func (r *Runtime) objectproto_hasOwnProperty(call FunctionCall) Value {
	p := call.Argument(0).String()
	o := call.This.ToObject(r)
	if o.self.hasOwnPropertyStr(p) {
		return valueTrue
	} else {
		return valueFalse
	}
}

func (r *Runtime) objectproto_isPrototypeOf(call FunctionCall) Value {
	if v, ok := call.Argument(0).(*Object); ok {
		o := call.This.ToObject(r)
		for {
			v = v.self.proto()
			if v == nil {
				break
			}
			if v == o {
				return valueTrue
			}
		}
	}
	return valueFalse
}

func (r *Runtime) objectproto_propertyIsEnumerable(call FunctionCall) Value {
	p := call.Argument(0).ToString()
	o := call.This.ToObject(r)
	pv := o.self.getOwnProp(p.String())
	if pv == nil {
		return valueFalse
	}
	if prop, ok := pv.(*valueProperty); ok {
		if !prop.enumerable {
			return valueFalse
		}
	}
	return valueTrue
}

func (r *Runtime) objectproto_toString(call FunctionCall) Value {
	switch o := call.This.(type) {
	case valueNull:
		return stringObjectNull
	case valueUndefined:
		return stringObjectUndefined
	case *Object:
		return newStringValue(fmt.Sprintf("[object %s]", o.self.className()))
	default:
		obj := call.This.ToObject(r)
		return newStringValue(fmt.Sprintf("[object %s]", obj.self.className()))
	}
}

func (r *Runtime) objectproto_toLocaleString(call FunctionCall) Value {
	return call.This.ToObject(r).ToString()
}

func (r *Runtime) objectproto_valueOf(call FunctionCall) Value {
	return call.This.ToObject(r)
}

func (r *Runtime) initObject() {
	o := r.global.ObjectPrototype.self
	o._putProp("toString", r.newNativeFunc(r.objectproto_toString, nil, "toString", nil, 0), true, false, true)
	o._putProp("toLocaleString", r.newNativeFunc(r.objectproto_toLocaleString, nil, "toLocaleString", nil, 0), true, false, true)
	o._putProp("valueOf", r.newNativeFunc(r.objectproto_valueOf, nil, "valueOf", nil, 0), true, false, true)
	o._putProp("hasOwnProperty", r.newNativeFunc(r.objectproto_hasOwnProperty, nil, "hasOwnProperty", nil, 1), true, false, true)
	o._putProp("isPrototypeOf", r.newNativeFunc(r.objectproto_isPrototypeOf, nil, "isPrototypeOf", nil, 1), true, false, true)
	o._putProp("propertyIsEnumerable", r.newNativeFunc(r.objectproto_propertyIsEnumerable, nil, "propertyIsEnumerable", nil, 1), true, false, true)

	r.global.Object = r.newNativeFuncConstruct(r.builtin_Object, classObject, r.global.ObjectPrototype, 1)
	o = r.global.Object.self
	o._putProp("defineProperty", r.newNativeFunc(r.object_defineProperty, nil, "defineProperty", nil, 3), true, false, true)
	o._putProp("defineProperties", r.newNativeFunc(r.object_defineProperties, nil, "defineProperties", nil, 2), true, false, true)
	o._putProp("getOwnPropertyDescriptor", r.newNativeFunc(r.object_getOwnPropertyDescriptor, nil, "getOwnPropertyDescriptor", nil, 2), true, false, true)
	o._putProp("getPrototypeOf", r.newNativeFunc(r.object_getPrototypeOf, nil, "getPrototypeOf", nil, 1), true, false, true)
	o._putProp("getOwnPropertyNames", r.newNativeFunc(r.object_getOwnPropertyNames, nil, "getOwnPropertyNames", nil, 1), true, false, true)
	o._putProp("create", r.newNativeFunc(r.object_create, nil, "create", nil, 2), true, false, true)
	o._putProp("seal", r.newNativeFunc(r.object_seal, nil, "seal", nil, 1), true, false, true)
	o._putProp("freeze", r.newNativeFunc(r.object_freeze, nil, "freeze", nil, 1), true, false, true)
	o._putProp("preventExtensions", r.newNativeFunc(r.object_preventExtensions, nil, "preventExtensions", nil, 1), true, false, true)
	o._putProp("isSealed", r.newNativeFunc(r.object_isSealed, nil, "isSealed", nil, 1), true, false, true)
	o._putProp("isFrozen", r.newNativeFunc(r.object_isFrozen, nil, "isFrozen", nil, 1), true, false, true)
	o._putProp("isExtensible", r.newNativeFunc(r.object_isExtensible, nil, "isExtensible", nil, 1), true, false, true)
	o._putProp("keys", r.newNativeFunc(r.object_keys, nil, "keys", nil, 1), true, false, true)

	r.addToGlobal("Object", r.global.Object)
}
