package otto

import (
	"encoding/json"
)

type _objectClass struct {
	getOwnProperty    func(*_object, string) *_property
	getProperty       func(*_object, string) *_property
	get               func(*_object, string) Value
	canPut            func(*_object, string) bool
	put               func(*_object, string, Value, bool)
	hasProperty       func(*_object, string) bool
	hasOwnProperty    func(*_object, string) bool
	defineOwnProperty func(*_object, string, _property, bool) bool
	delete            func(*_object, string, bool) bool
	enumerate         func(*_object, bool, func(string) bool)
	clone             func(*_object, *_object, *_clone) *_object
	marshalJSON       func(*_object) json.Marshaler
}

func objectEnumerate(self *_object, all bool, each func(string) bool) {
	for _, name := range self.propertyOrder {
		if all || self.property[name].enumerable() {
			if !each(name) {
				return
			}
		}
	}
}

var (
	_classObject,
	_classArray,
	_classString,
	_classArguments,
	_classGoStruct,
	_classGoMap,
	_classGoArray,
	_classGoSlice,
	_ *_objectClass
)

func init() {
	_classObject = &_objectClass{
		objectGetOwnProperty,
		objectGetProperty,
		objectGet,
		objectCanPut,
		objectPut,
		objectHasProperty,
		objectHasOwnProperty,
		objectDefineOwnProperty,
		objectDelete,
		objectEnumerate,
		objectClone,
		nil,
	}

	_classArray = &_objectClass{
		objectGetOwnProperty,
		objectGetProperty,
		objectGet,
		objectCanPut,
		objectPut,
		objectHasProperty,
		objectHasOwnProperty,
		arrayDefineOwnProperty,
		objectDelete,
		objectEnumerate,
		objectClone,
		nil,
	}

	_classString = &_objectClass{
		stringGetOwnProperty,
		objectGetProperty,
		objectGet,
		objectCanPut,
		objectPut,
		objectHasProperty,
		objectHasOwnProperty,
		objectDefineOwnProperty,
		objectDelete,
		stringEnumerate,
		objectClone,
		nil,
	}

	_classArguments = &_objectClass{
		argumentsGetOwnProperty,
		objectGetProperty,
		argumentsGet,
		objectCanPut,
		objectPut,
		objectHasProperty,
		objectHasOwnProperty,
		argumentsDefineOwnProperty,
		argumentsDelete,
		objectEnumerate,
		objectClone,
		nil,
	}

	_classGoStruct = &_objectClass{
		goStructGetOwnProperty,
		objectGetProperty,
		objectGet,
		goStructCanPut,
		goStructPut,
		objectHasProperty,
		objectHasOwnProperty,
		objectDefineOwnProperty,
		objectDelete,
		goStructEnumerate,
		objectClone,
		goStructMarshalJSON,
	}

	_classGoMap = &_objectClass{
		goMapGetOwnProperty,
		objectGetProperty,
		objectGet,
		objectCanPut,
		objectPut,
		objectHasProperty,
		objectHasOwnProperty,
		goMapDefineOwnProperty,
		goMapDelete,
		goMapEnumerate,
		objectClone,
		nil,
	}

	_classGoArray = &_objectClass{
		goArrayGetOwnProperty,
		objectGetProperty,
		objectGet,
		objectCanPut,
		objectPut,
		objectHasProperty,
		objectHasOwnProperty,
		goArrayDefineOwnProperty,
		goArrayDelete,
		goArrayEnumerate,
		objectClone,
		nil,
	}

	_classGoSlice = &_objectClass{
		goSliceGetOwnProperty,
		objectGetProperty,
		objectGet,
		objectCanPut,
		objectPut,
		objectHasProperty,
		objectHasOwnProperty,
		goSliceDefineOwnProperty,
		goSliceDelete,
		goSliceEnumerate,
		objectClone,
		nil,
	}
}

// Allons-y

// 8.12.1
func objectGetOwnProperty(self *_object, name string) *_property {
	// Return a _copy_ of the property
	property, exists := self._read(name)
	if !exists {
		return nil
	}
	return &property
}

// 8.12.2
func objectGetProperty(self *_object, name string) *_property {
	property := self.getOwnProperty(name)
	if property != nil {
		return property
	}
	if self.prototype != nil {
		return self.prototype.getProperty(name)
	}
	return nil
}

// 8.12.3
func objectGet(self *_object, name string) Value {
	property := self.getProperty(name)
	if property != nil {
		return property.get(self)
	}
	return UndefinedValue()
}

// 8.12.4
func objectCanPut(self *_object, name string) bool {
	canPut, _, _ := _objectCanPut(self, name)
	return canPut
}

func _objectCanPut(self *_object, name string) (canPut bool, property *_property, setter *_object) {
	property = self.getOwnProperty(name)
	if property != nil {
		switch propertyValue := property.value.(type) {
		case Value:
			canPut = property.writable()
			return
		case _propertyGetSet:
			setter = propertyValue[1]
			canPut = setter != nil
			return
		default:
			panic(newTypeError())
		}
	}

	if self.prototype == nil {
		return self.extensible, nil, nil
	}

	property = self.prototype.getProperty(name)
	if property == nil {
		return self.extensible, nil, nil
	}

	switch propertyValue := property.value.(type) {
	case Value:
		if !self.extensible {
			return false, nil, nil
		}
		return property.writable(), nil, nil
	case _propertyGetSet:
		setter = propertyValue[1]
		canPut = setter != nil
		return
	default:
		panic(newTypeError())
	}

	return false, nil, nil
}

// 8.12.5
func objectPut(self *_object, name string, value Value, throw bool) {

	if true {
		// Shortcut...
		//
		// So, right now, every class is using objectCanPut and every class
		// is using objectPut.
		//
		// If that were to no longer be the case, we would have to have
		// something to detect that here, so that we do not use an
		// incompatible canPut routine
		canPut, property, setter := _objectCanPut(self, name)
		if !canPut {
			typeErrorResult(throw)
		} else if setter != nil {
			setter.callSet(toValue(self), value)
		} else if property != nil {
			property.value = value
			self.defineOwnProperty(name, *property, throw)
		} else {
			self.defineProperty(name, value, 0111, throw)
		}
		return
	}

	// The long way...
	//
	// Right now, code should never get here, see above
	if !self.canPut(name) {
		typeErrorResult(throw)
		return
	}

	property := self.getOwnProperty(name)
	if property == nil {
		property = self.getProperty(name)
		if property != nil {
			if getSet, isAccessor := property.value.(_propertyGetSet); isAccessor {
				getSet[1].callSet(toValue(self), value)
				return
			}
		}
		self.defineProperty(name, value, 0111, throw)
	} else {
		switch propertyValue := property.value.(type) {
		case Value:
			property.value = value
			self.defineOwnProperty(name, *property, throw)
		case _propertyGetSet:
			if propertyValue[1] != nil {
				propertyValue[1].callSet(toValue(self), value)
				return
			}
			if throw {
				panic(newTypeError())
			}
		default:
			panic(newTypeError())
		}
	}
}

// 8.12.6
func objectHasProperty(self *_object, name string) bool {
	return self.getProperty(name) != nil
}

func objectHasOwnProperty(self *_object, name string) bool {
	return self.getOwnProperty(name) != nil
}

// 8.12.9
func objectDefineOwnProperty(self *_object, name string, descriptor _property, throw bool) bool {
	property, exists := self._read(name)
	{
		if !exists {
			if !self.extensible {
				goto Reject
			}
			if newGetSet, isAccessor := descriptor.value.(_propertyGetSet); isAccessor {
				if newGetSet[0] == &_nilGetSetObject {
					newGetSet[0] = nil
				}
				if newGetSet[1] == &_nilGetSetObject {
					newGetSet[1] = nil
				}
				descriptor.value = newGetSet
			}
			self._write(name, descriptor.value, descriptor.mode)
			return true
		}
		if descriptor.isEmpty() {
			return true
		}

		// TODO Per 8.12.9.6 - We should shortcut here (returning true) if
		// the current and new (define) properties are the same

		configurable := property.configurable()
		if !configurable {
			if descriptor.configurable() {
				goto Reject
			}
			// Test that, if enumerable is set on the property descriptor, then it should
			// be the same as the existing property
			if descriptor.enumerateSet() && descriptor.enumerable() != property.enumerable() {
				goto Reject
			}
		}
		value, isDataDescriptor := property.value.(Value)
		getSet, _ := property.value.(_propertyGetSet)
		if descriptor.isGenericDescriptor() {
			// GenericDescriptor
		} else if isDataDescriptor != descriptor.isDataDescriptor() {
			// DataDescriptor <=> AccessorDescriptor
			if !configurable {
				goto Reject
			}
		} else if isDataDescriptor && descriptor.isDataDescriptor() {
			// DataDescriptor <=> DataDescriptor
			if !configurable {
				if !property.writable() && descriptor.writable() {
					goto Reject
				}
				if !property.writable() {
					if descriptor.value != nil && !sameValue(value, descriptor.value.(Value)) {
						goto Reject
					}
				}
			}
		} else {
			// AccessorDescriptor <=> AccessorDescriptor
			newGetSet, _ := descriptor.value.(_propertyGetSet)
			presentGet, presentSet := true, true
			if newGetSet[0] == &_nilGetSetObject {
				// Present, but nil
				newGetSet[0] = nil
			} else if newGetSet[0] == nil {
				// Missing, not even nil
				newGetSet[0] = getSet[0]
				presentGet = false
			}
			if newGetSet[1] == &_nilGetSetObject {
				// Present, but nil
				newGetSet[1] = nil
			} else if newGetSet[1] == nil {
				// Missing, not even nil
				newGetSet[1] = getSet[1]
				presentSet = false
			}
			if !configurable {
				if (presentGet && (getSet[0] != newGetSet[0])) || (presentSet && (getSet[1] != newGetSet[1])) {
					goto Reject
				}
			}
			descriptor.value = newGetSet
		}
		{
			// This section will preserve attributes of
			// the original property, if necessary
			value1 := descriptor.value
			if value1 == nil {
				value1 = property.value
			} else if newGetSet, isAccessor := descriptor.value.(_propertyGetSet); isAccessor {
				if newGetSet[0] == &_nilGetSetObject {
					newGetSet[0] = nil
				}
				if newGetSet[1] == &_nilGetSetObject {
					newGetSet[1] = nil
				}
				value1 = newGetSet
			}
			mode1 := descriptor.mode
			if mode1&0222 != 0 {
				// TODO Factor this out into somewhere testable
				// (Maybe put into switch ...)
				mode0 := property.mode
				if mode1&0200 != 0 {
					if descriptor.isDataDescriptor() {
						mode1 &= ^0200 // Turn off "writable" missing
						mode1 |= (mode0 & 0100)
					}
				}
				if mode1&020 != 0 {
					mode1 |= (mode0 & 010)
				}
				if mode1&02 != 0 {
					mode1 |= (mode0 & 01)
				}
				mode1 &= 0311 // 0311 to preserve the non-setting on "writable"
			}
			self._write(name, value1, mode1)
		}
		return true
	}
Reject:
	if throw {
		panic(newTypeError())
	}
	return false
}

func objectDelete(self *_object, name string, throw bool) bool {
	property_ := self.getOwnProperty(name)
	if property_ == nil {
		return true
	}
	if property_.configurable() {
		self._delete(name)
		return true
	}
	return typeErrorResult(throw)
}

func objectClone(self0 *_object, self1 *_object, clone *_clone) *_object {
	*self1 = *self0

	self1.runtime = clone.runtime
	if self1.prototype != nil {
		self1.prototype = clone.object(self0.prototype)
	}
	self1.property = make(map[string]_property, len(self0.property))
	self1.propertyOrder = make([]string, len(self0.propertyOrder))
	copy(self1.propertyOrder, self0.propertyOrder)
	for index, property := range self0.property {
		self1.property[index] = clone.property(property)
	}

	switch value := self0.value.(type) {
	case _functionObject:
		self1.value = value.clone(clone)
	case _argumentsObject:
		self1.value = value.clone(clone)
	}

	return self1
}
