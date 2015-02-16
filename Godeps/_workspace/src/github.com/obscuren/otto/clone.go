package otto

import (
	"fmt"
)

type _clone struct {
	runtime *_runtime
	stash   struct {
		object                 map[*_object]*_object
		objectEnvironment      map[*_objectEnvironment]*_objectEnvironment
		declarativeEnvironment map[*_declarativeEnvironment]*_declarativeEnvironment
	}
}

func (runtime *_runtime) clone() *_runtime {

	self := &_runtime{}
	clone := &_clone{
		runtime: self,
	}
	clone.stash.object = make(map[*_object]*_object)
	clone.stash.objectEnvironment = make(map[*_objectEnvironment]*_objectEnvironment)
	clone.stash.declarativeEnvironment = make(map[*_declarativeEnvironment]*_declarativeEnvironment)

	globalObject := clone.object(runtime.GlobalObject)
	self.GlobalEnvironment = self.newObjectEnvironment(globalObject, nil)
	self.GlobalObject = globalObject
	self.Global = _global{
		clone.object(runtime.Global.Object),
		clone.object(runtime.Global.Function),
		clone.object(runtime.Global.Array),
		clone.object(runtime.Global.String),
		clone.object(runtime.Global.Boolean),
		clone.object(runtime.Global.Number),
		clone.object(runtime.Global.Math),
		clone.object(runtime.Global.Date),
		clone.object(runtime.Global.RegExp),
		clone.object(runtime.Global.Error),
		clone.object(runtime.Global.EvalError),
		clone.object(runtime.Global.TypeError),
		clone.object(runtime.Global.RangeError),
		clone.object(runtime.Global.ReferenceError),
		clone.object(runtime.Global.SyntaxError),
		clone.object(runtime.Global.URIError),
		clone.object(runtime.Global.JSON),

		clone.object(runtime.Global.ObjectPrototype),
		clone.object(runtime.Global.FunctionPrototype),
		clone.object(runtime.Global.ArrayPrototype),
		clone.object(runtime.Global.StringPrototype),
		clone.object(runtime.Global.BooleanPrototype),
		clone.object(runtime.Global.NumberPrototype),
		clone.object(runtime.Global.DatePrototype),
		clone.object(runtime.Global.RegExpPrototype),
		clone.object(runtime.Global.ErrorPrototype),
		clone.object(runtime.Global.EvalErrorPrototype),
		clone.object(runtime.Global.TypeErrorPrototype),
		clone.object(runtime.Global.RangeErrorPrototype),
		clone.object(runtime.Global.ReferenceErrorPrototype),
		clone.object(runtime.Global.SyntaxErrorPrototype),
		clone.object(runtime.Global.URIErrorPrototype),
	}

	self.EnterGlobalExecutionContext()

	self.eval = self.GlobalObject.property["eval"].value.(Value).value.(*_object)
	self.GlobalObject.prototype = self.Global.ObjectPrototype

	return self
}
func (clone *_clone) object(self0 *_object) *_object {
	if self1, exists := clone.stash.object[self0]; exists {
		return self1
	}
	self1 := &_object{}
	clone.stash.object[self0] = self1
	return self0.objectClass.clone(self0, self1, clone)
}

func (clone *_clone) declarativeEnvironment(self0 *_declarativeEnvironment) (*_declarativeEnvironment, bool) {
	if self1, exists := clone.stash.declarativeEnvironment[self0]; exists {
		return self1, true
	}
	self1 := &_declarativeEnvironment{}
	clone.stash.declarativeEnvironment[self0] = self1
	return self1, false
}

func (clone *_clone) objectEnvironment(self0 *_objectEnvironment) (*_objectEnvironment, bool) {
	if self1, exists := clone.stash.objectEnvironment[self0]; exists {
		return self1, true
	}
	self1 := &_objectEnvironment{}
	clone.stash.objectEnvironment[self0] = self1
	return self1, false
}

func (clone *_clone) value(self0 Value) Value {
	self1 := self0
	switch value := self0.value.(type) {
	case *_object:
		self1.value = clone.object(value)
	}
	return self1
}

func (clone *_clone) valueArray(self0 []Value) []Value {
	self1 := make([]Value, len(self0))
	for index, value := range self0 {
		self1[index] = clone.value(value)
	}
	return self1
}

func (clone *_clone) environment(self0 _environment) _environment {
	if self0 == nil {
		return nil
	}
	return self0.clone(clone)
}

func (clone *_clone) property(self0 _property) _property {
	self1 := self0
	if value, valid := self0.value.(Value); valid {
		self1.value = clone.value(value)
	} else {
		panic(fmt.Errorf("self0.value.(Value) != true"))
	}
	return self1
}

func (clone *_clone) declarativeProperty(self0 _declarativeProperty) _declarativeProperty {
	self1 := self0
	self1.value = clone.value(self0.value)
	return self1
}

func (clone *_clone) callFunction(self0 _callFunction) _callFunction {
	if self0 == nil {
		return nil
	}
	return self0.clone(clone)
}
