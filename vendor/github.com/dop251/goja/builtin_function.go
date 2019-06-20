package goja

import (
	"fmt"
)

func (r *Runtime) builtin_Function(args []Value, proto *Object) *Object {
	src := "(function anonymous("
	if len(args) > 1 {
		for _, arg := range args[:len(args)-1] {
			src += arg.String() + ","
		}
		src = src[:len(src)-1]
	}
	body := ""
	if len(args) > 0 {
		body = args[len(args)-1].String()
	}
	src += "){" + body + "})"

	return r.toObject(r.eval(src, false, false, _undefined))
}

func (r *Runtime) functionproto_toString(call FunctionCall) Value {
	obj := r.toObject(call.This)
repeat:
	switch f := obj.self.(type) {
	case *funcObject:
		return newStringValue(f.src)
	case *nativeFuncObject:
		return newStringValue(fmt.Sprintf("function %s() { [native code] }", f.nameProp.get(call.This).ToString()))
	case *boundFuncObject:
		return newStringValue(fmt.Sprintf("function %s() { [native code] }", f.nameProp.get(call.This).ToString()))
	case *lazyObject:
		obj.self = f.create(obj)
		goto repeat
	}

	r.typeErrorResult(true, "Object is not a function")
	return nil
}

func (r *Runtime) toValueArray(a Value) []Value {
	obj := r.toObject(a)
	l := toUInt32(obj.self.getStr("length"))
	ret := make([]Value, l)
	for i := uint32(0); i < l; i++ {
		ret[i] = obj.self.get(valueInt(i))
	}
	return ret
}

func (r *Runtime) functionproto_apply(call FunctionCall) Value {
	f := r.toCallable(call.This)
	var args []Value
	if len(call.Arguments) >= 2 {
		args = r.toValueArray(call.Arguments[1])
	}
	return f(FunctionCall{
		This:      call.Argument(0),
		Arguments: args,
	})
}

func (r *Runtime) functionproto_call(call FunctionCall) Value {
	f := r.toCallable(call.This)
	var args []Value
	if len(call.Arguments) > 0 {
		args = call.Arguments[1:]
	}
	return f(FunctionCall{
		This:      call.Argument(0),
		Arguments: args,
	})
}

func (r *Runtime) boundCallable(target func(FunctionCall) Value, boundArgs []Value) func(FunctionCall) Value {
	var this Value
	var args []Value
	if len(boundArgs) > 0 {
		this = boundArgs[0]
		args = make([]Value, len(boundArgs)-1)
		copy(args, boundArgs[1:])
	} else {
		this = _undefined
	}
	return func(call FunctionCall) Value {
		a := append(args, call.Arguments...)
		return target(FunctionCall{
			This:      this,
			Arguments: a,
		})
	}
}

func (r *Runtime) boundConstruct(target func([]Value) *Object, boundArgs []Value) func([]Value) *Object {
	if target == nil {
		return nil
	}
	var args []Value
	if len(boundArgs) > 1 {
		args = make([]Value, len(boundArgs)-1)
		copy(args, boundArgs[1:])
	}
	return func(fargs []Value) *Object {
		a := append(args, fargs...)
		copy(a, args)
		return target(a)
	}
}

func (r *Runtime) functionproto_bind(call FunctionCall) Value {
	obj := r.toObject(call.This)
	f := obj.self
	var fcall func(FunctionCall) Value
	var construct func([]Value) *Object
repeat:
	switch ff := f.(type) {
	case *funcObject:
		fcall = ff.Call
		construct = ff.construct
	case *nativeFuncObject:
		fcall = ff.f
		construct = ff.construct
	case *boundFuncObject:
		f = &ff.nativeFuncObject
		goto repeat
	case *lazyObject:
		f = ff.create(obj)
		goto repeat
	default:
		r.typeErrorResult(true, "Value is not callable: %s", obj.ToString())
	}

	l := int(toUInt32(obj.self.getStr("length")))
	l -= len(call.Arguments) - 1
	if l < 0 {
		l = 0
	}

	v := &Object{runtime: r}

	ff := r.newNativeFuncObj(v, r.boundCallable(fcall, call.Arguments), r.boundConstruct(construct, call.Arguments), "", nil, l)
	v.self = &boundFuncObject{
		nativeFuncObject: *ff,
	}

	//ret := r.newNativeFunc(r.boundCallable(f, call.Arguments), nil, "", nil, l)
	//o := ret.self
	//o.putStr("caller", r.global.throwerProperty, false)
	//o.putStr("arguments", r.global.throwerProperty, false)
	return v
}

func (r *Runtime) initFunction() {
	o := r.global.FunctionPrototype.self
	o.(*nativeFuncObject).prototype = r.global.ObjectPrototype
	o._putProp("toString", r.newNativeFunc(r.functionproto_toString, nil, "toString", nil, 0), true, false, true)
	o._putProp("apply", r.newNativeFunc(r.functionproto_apply, nil, "apply", nil, 2), true, false, true)
	o._putProp("call", r.newNativeFunc(r.functionproto_call, nil, "call", nil, 1), true, false, true)
	o._putProp("bind", r.newNativeFunc(r.functionproto_bind, nil, "bind", nil, 1), true, false, true)

	r.global.Function = r.newNativeFuncConstruct(r.builtin_Function, "Function", r.global.FunctionPrototype, 1)
	r.addToGlobal("Function", r.global.Function)
}
