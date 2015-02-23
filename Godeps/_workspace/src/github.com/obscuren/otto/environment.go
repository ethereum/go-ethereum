package otto

import (
	"fmt"
)

// _environment

type _environment interface {
	HasBinding(string) bool

	CreateMutableBinding(string, bool)
	SetMutableBinding(string, Value, bool)
	// SetMutableBinding with Lazy CreateMutableBinding(..., true)
	SetValue(string, Value, bool)

	GetBindingValue(string, bool) Value
	GetValue(string, bool) Value // GetBindingValue
	DeleteBinding(string) bool
	ImplicitThisValue() *_object

	Outer() _environment

	newReference(string, bool) _reference
	clone(clone *_clone) _environment
	runtimeOf() *_runtime
}

// _functionEnvironment

type _functionEnvironment struct {
	_declarativeEnvironment
	arguments           *_object
	indexOfArgumentName map[string]string
}

func (runtime *_runtime) newFunctionEnvironment(outer _environment) *_functionEnvironment {
	return &_functionEnvironment{
		_declarativeEnvironment: _declarativeEnvironment{
			runtime:  runtime,
			outer:    outer,
			property: map[string]_declarativeProperty{},
		},
	}
}

func (self0 _functionEnvironment) clone(clone *_clone) _environment {
	return &_functionEnvironment{
		*(self0._declarativeEnvironment.clone(clone).(*_declarativeEnvironment)),
		clone.object(self0.arguments),
		self0.indexOfArgumentName,
	}
}

func (self _functionEnvironment) runtimeOf() *_runtime {
	return self._declarativeEnvironment.runtimeOf()
}

// _objectEnvironment

type _objectEnvironment struct {
	runtime     *_runtime
	outer       _environment
	Object      *_object
	ProvideThis bool
}

func (self *_objectEnvironment) runtimeOf() *_runtime {
	return self.runtime
}

func (runtime *_runtime) newObjectEnvironment(object *_object, outer _environment) *_objectEnvironment {
	if object == nil {
		object = runtime.newBaseObject()
		object.class = "environment"
	}
	return &_objectEnvironment{
		runtime: runtime,
		outer:   outer,
		Object:  object,
	}
}

func (self0 *_objectEnvironment) clone(clone *_clone) _environment {
	self1, exists := clone.objectEnvironment(self0)
	if exists {
		return self1
	}
	*self1 = _objectEnvironment{
		clone.runtime,
		clone.environment(self0.outer),
		clone.object(self0.Object),
		self0.ProvideThis,
	}
	return self1
}

func (self *_objectEnvironment) HasBinding(name string) bool {
	return self.Object.hasProperty(name)
}

func (self *_objectEnvironment) CreateMutableBinding(name string, deletable bool) {
	if self.Object.hasProperty(name) {
		panic(hereBeDragons())
	}
	mode := _propertyMode(0111)
	if !deletable {
		mode = _propertyMode(0110)
	}
	// TODO False?
	self.Object.defineProperty(name, UndefinedValue(), mode, false)
}

func (self *_objectEnvironment) SetMutableBinding(name string, value Value, strict bool) {
	self.Object.put(name, value, strict)
}

func (self *_objectEnvironment) SetValue(name string, value Value, throw bool) {
	if !self.HasBinding(name) {
		self.CreateMutableBinding(name, true) // Configurable by default
	}
	self.SetMutableBinding(name, value, throw)
}

func (self *_objectEnvironment) GetBindingValue(name string, strict bool) Value {
	if self.Object.hasProperty(name) {
		return self.Object.get(name)
	}
	if strict {
		panic(newReferenceError("Not Defined", name))
	}
	return UndefinedValue()
}

func (self *_objectEnvironment) GetValue(name string, throw bool) Value {
	return self.GetBindingValue(name, throw)
}

func (self *_objectEnvironment) DeleteBinding(name string) bool {
	return self.Object.delete(name, false)
}

func (self *_objectEnvironment) ImplicitThisValue() *_object {
	if self.ProvideThis {
		return self.Object
	}
	return nil
}

func (self *_objectEnvironment) Outer() _environment {
	return self.outer
}

func (self *_objectEnvironment) newReference(name string, strict bool) _reference {
	return newPropertyReference(self.Object, name, strict)
}

// _declarativeEnvironment

func (runtime *_runtime) newDeclarativeEnvironment(outer _environment) *_declarativeEnvironment {
	return &_declarativeEnvironment{
		runtime:  runtime,
		outer:    outer,
		property: map[string]_declarativeProperty{},
	}
}

func (self0 *_declarativeEnvironment) clone(clone *_clone) _environment {
	self1, exists := clone.declarativeEnvironment(self0)
	if exists {
		return self1
	}
	property := make(map[string]_declarativeProperty, len(self0.property))
	for index, value := range self0.property {
		property[index] = clone.declarativeProperty(value)
	}
	*self1 = _declarativeEnvironment{
		clone.runtime,
		clone.environment(self0.outer),
		property,
	}
	return self1
}

type _declarativeProperty struct {
	value     Value
	mutable   bool
	deletable bool
	readable  bool
}

type _declarativeEnvironment struct {
	runtime  *_runtime
	outer    _environment
	property map[string]_declarativeProperty
}

func (self *_declarativeEnvironment) HasBinding(name string) bool {
	_, exists := self.property[name]
	return exists
}

func (self *_declarativeEnvironment) runtimeOf() *_runtime {
	return self.runtime
}

func (self *_declarativeEnvironment) CreateMutableBinding(name string, deletable bool) {
	_, exists := self.property[name]
	if exists {
		panic(fmt.Errorf("CreateMutableBinding: %s: already exists", name))
	}
	self.property[name] = _declarativeProperty{
		value:     UndefinedValue(),
		mutable:   true,
		deletable: deletable,
		readable:  false,
	}
}

func (self *_declarativeEnvironment) SetMutableBinding(name string, value Value, strict bool) {
	property, exists := self.property[name]
	if !exists {
		panic(fmt.Errorf("SetMutableBinding: %s: missing", name))
	}
	if property.mutable {
		property.value = value
		self.property[name] = property
	} else {
		typeErrorResult(strict)
	}
}

func (self *_declarativeEnvironment) SetValue(name string, value Value, throw bool) {
	if !self.HasBinding(name) {
		self.CreateMutableBinding(name, false) // NOT deletable by default
	}
	self.SetMutableBinding(name, value, throw)
}

func (self *_declarativeEnvironment) GetBindingValue(name string, strict bool) Value {
	property, exists := self.property[name]
	if !exists {
		panic(fmt.Errorf("GetBindingValue: %s: missing", name))
	}
	if !property.mutable && !property.readable {
		if strict {
			panic(newTypeError())
		}
		return UndefinedValue()
	}
	return property.value
}

func (self *_declarativeEnvironment) GetValue(name string, throw bool) Value {
	return self.GetBindingValue(name, throw)
}

func (self *_declarativeEnvironment) DeleteBinding(name string) bool {
	property, exists := self.property[name]
	if !exists {
		return true
	}
	if !property.deletable {
		return false
	}
	delete(self.property, name)
	return true
}

func (self *_declarativeEnvironment) ImplicitThisValue() *_object {
	return nil
}

func (self *_declarativeEnvironment) Outer() _environment {
	return self.outer
}

func (self *_declarativeEnvironment) newReference(name string, strict bool) _reference {
	return newEnvironmentReference(self, name, strict, nil)
}
