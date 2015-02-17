package otto

import (
	"github.com/robertkrimen/otto/ast"
)

type _reference interface {
	GetBase() interface{}      // GetBase
	GetName() string           // GetReferencedName
	IsStrict() bool            // IsStrictReference
	IsUnresolvable() bool      // IsUnresolvableReference
	IsPropertyReference() bool // IsPropertyReference
	GetValue() Value           // GetValue
	PutValue(Value) bool       // PutValue
	Delete() bool
}

// Reference

type _referenceDefault struct {
	name   string
	strict bool
}

func (self _referenceDefault) GetName() string {
	return self.name
}

func (self _referenceDefault) IsStrict() bool {
	return self.strict
}

// PropertyReference

type _propertyReference struct {
	_referenceDefault
	Base *_object
}

func newPropertyReference(base *_object, name string, strict bool) *_propertyReference {
	return &_propertyReference{
		Base: base,
		_referenceDefault: _referenceDefault{
			name:   name,
			strict: strict,
		},
	}
}

func (self *_propertyReference) GetBase() interface{} {
	return self.Base
}

func (self *_propertyReference) IsUnresolvable() bool {
	return self.Base == nil
}

func (self *_propertyReference) IsPropertyReference() bool {
	return true
}

func (self *_propertyReference) GetValue() Value {
	if self.Base == nil {
		panic(newReferenceError("notDefined", self.name))
	}
	return self.Base.get(self.name)
}

func (self *_propertyReference) PutValue(value Value) bool {
	if self.Base == nil {
		return false
	}
	self.Base.put(self.name, value, self.IsStrict())
	return true
}

func (self *_propertyReference) Delete() bool {
	if self.Base == nil {
		// TODO Throw an error if strict
		return true
	}
	return self.Base.delete(self.name, self.IsStrict())
}

// ArgumentReference

func newArgumentReference(base *_object, name string, strict bool) *_propertyReference {
	if base == nil {
		panic(hereBeDragons())
	}
	return newPropertyReference(base, name, strict)
}

type _environmentReference struct {
	_referenceDefault
	Base _environment
	node ast.Node
}

func newEnvironmentReference(base _environment, name string, strict bool, node ast.Node) *_environmentReference {
	return &_environmentReference{
		Base: base,
		_referenceDefault: _referenceDefault{
			name:   name,
			strict: strict,
		},
		node: node,
	}
}

func (self *_environmentReference) GetBase() interface{} {
	return self.Base
}

func (self *_environmentReference) IsUnresolvable() bool {
	return self.Base == nil // The base (an environment) will never be nil
}

func (self *_environmentReference) IsPropertyReference() bool {
	return false
}

func (self *_environmentReference) GetValue() Value {
	if self.Base == nil {
		// This should never be reached, but just in case
	}
	return self.Base.GetValue(self.name, self.IsStrict())
}

func (self *_environmentReference) PutValue(value Value) bool {
	if self.Base == nil {
		// This should never be reached, but just in case
		return false
	}
	self.Base.SetValue(self.name, value, self.IsStrict())
	return true
}

func (self *_environmentReference) Delete() bool {
	if self.Base == nil {
		// This should never be reached, but just in case
		return false
	}
	return self.Base.DeleteBinding(self.name)
}

// getIdentifierReference

func getIdentifierReference(environment _environment, name string, strict bool) _reference {
	if environment == nil {
		return newPropertyReference(nil, name, strict)
	}
	if environment.HasBinding(name) {
		return environment.newReference(name, strict)
	}
	return getIdentifierReference(environment.Outer(), name, strict)
}
