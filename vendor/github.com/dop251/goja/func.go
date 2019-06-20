package goja

import "reflect"

type baseFuncObject struct {
	baseObject

	nameProp, lenProp valueProperty
}

type funcObject struct {
	baseFuncObject

	stash *stash
	prg   *Program
	src   string
}

type nativeFuncObject struct {
	baseFuncObject

	f         func(FunctionCall) Value
	construct func(args []Value) *Object
}

type boundFuncObject struct {
	nativeFuncObject
}

func (f *nativeFuncObject) export() interface{} {
	return f.f
}

func (f *nativeFuncObject) exportType() reflect.Type {
	return reflect.TypeOf(f.f)
}

func (f *funcObject) getPropStr(name string) Value {
	switch name {
	case "prototype":
		if _, exists := f.values["prototype"]; !exists {
			return f.addPrototype()
		}
	}

	return f.baseObject.getPropStr(name)
}

func (f *funcObject) addPrototype() Value {
	proto := f.val.runtime.NewObject()
	proto.self._putProp("constructor", f.val, true, false, true)
	return f._putProp("prototype", proto, true, false, false)
}

func (f *funcObject) getProp(n Value) Value {
	return f.getPropStr(n.String())
}

func (f *funcObject) hasOwnProperty(n Value) bool {
	if r := f.baseObject.hasOwnProperty(n); r {
		return true
	}

	name := n.String()
	if name == "prototype" {
		return true
	}
	return false
}

func (f *funcObject) hasOwnPropertyStr(name string) bool {
	if r := f.baseObject.hasOwnPropertyStr(name); r {
		return true
	}

	if name == "prototype" {
		return true
	}
	return false
}

func (f *funcObject) construct(args []Value) *Object {
	proto := f.getStr("prototype")
	var protoObj *Object
	if p, ok := proto.(*Object); ok {
		protoObj = p
	} else {
		protoObj = f.val.runtime.global.ObjectPrototype
	}
	obj := f.val.runtime.newBaseObject(protoObj, classObject).val
	ret := f.Call(FunctionCall{
		This:      obj,
		Arguments: args,
	})

	if ret, ok := ret.(*Object); ok {
		return ret
	}
	return obj
}

func (f *funcObject) Call(call FunctionCall) Value {
	vm := f.val.runtime.vm
	pc := vm.pc

	vm.stack.expand(vm.sp + len(call.Arguments) + 1)
	vm.stack[vm.sp] = f.val
	vm.sp++
	if call.This != nil {
		vm.stack[vm.sp] = call.This
	} else {
		vm.stack[vm.sp] = _undefined
	}
	vm.sp++
	for _, arg := range call.Arguments {
		if arg != nil {
			vm.stack[vm.sp] = arg
		} else {
			vm.stack[vm.sp] = _undefined
		}
		vm.sp++
	}

	vm.pc = -1
	vm.pushCtx()
	vm.args = len(call.Arguments)
	vm.prg = f.prg
	vm.stash = f.stash
	vm.pc = 0
	vm.run()
	vm.pc = pc
	vm.halt = false
	return vm.pop()
}

func (f *funcObject) export() interface{} {
	return f.Call
}

func (f *funcObject) exportType() reflect.Type {
	return reflect.TypeOf(f.Call)
}

func (f *funcObject) assertCallable() (func(FunctionCall) Value, bool) {
	return f.Call, true
}

func (f *baseFuncObject) init(name string, length int) {
	f.baseObject.init()

	f.nameProp.configurable = true
	f.nameProp.value = newStringValue(name)
	f._put("name", &f.nameProp)

	f.lenProp.configurable = true
	f.lenProp.value = valueInt(length)
	f._put("length", &f.lenProp)
}

func (f *baseFuncObject) hasInstance(v Value) bool {
	if v, ok := v.(*Object); ok {
		o := f.val.self.getStr("prototype")
		if o1, ok := o.(*Object); ok {
			for {
				v = v.self.proto()
				if v == nil {
					return false
				}
				if o1 == v {
					return true
				}
			}
		} else {
			f.val.runtime.typeErrorResult(true, "prototype is not an object")
		}
	}

	return false
}

func (f *nativeFuncObject) defaultConstruct(ccall func(ConstructorCall) *Object, args []Value) *Object {
	proto := f.getStr("prototype")
	var protoObj *Object
	if p, ok := proto.(*Object); ok {
		protoObj = p
	} else {
		protoObj = f.val.runtime.global.ObjectPrototype
	}
	obj := f.val.runtime.newBaseObject(protoObj, classObject).val
	ret := ccall(ConstructorCall{
		This:      obj,
		Arguments: args,
	})

	if ret != nil {
		return ret
	}
	return obj
}

func (f *nativeFuncObject) assertCallable() (func(FunctionCall) Value, bool) {
	if f.f != nil {
		return f.f, true
	}
	return nil, false
}

func (f *boundFuncObject) getProp(n Value) Value {
	return f.getPropStr(n.String())
}

func (f *boundFuncObject) getPropStr(name string) Value {
	if name == "caller" || name == "arguments" {
		//f.runtime.typeErrorResult(true, "'caller' and 'arguments' are restricted function properties and cannot be accessed in this context.")
		return f.val.runtime.global.throwerProperty
	}
	return f.nativeFuncObject.getPropStr(name)
}

func (f *boundFuncObject) delete(n Value, throw bool) bool {
	return f.deleteStr(n.String(), throw)
}

func (f *boundFuncObject) deleteStr(name string, throw bool) bool {
	if name == "caller" || name == "arguments" {
		return true
	}
	return f.nativeFuncObject.deleteStr(name, throw)
}

func (f *boundFuncObject) putStr(name string, val Value, throw bool) {
	if name == "caller" || name == "arguments" {
		f.val.runtime.typeErrorResult(true, "'caller' and 'arguments' are restricted function properties and cannot be accessed in this context.")
	}
	f.nativeFuncObject.putStr(name, val, throw)
}

func (f *boundFuncObject) put(n Value, val Value, throw bool) {
	f.putStr(n.String(), val, throw)
}
