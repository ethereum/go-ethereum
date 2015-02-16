package otto

import (
	"fmt"
)

// Object

func builtinObject(call FunctionCall) Value {
	value := call.Argument(0)
	switch value._valueType {
	case valueUndefined, valueNull:
		return toValue_object(call.runtime.newObject())
	}

	return toValue_object(call.runtime.toObject(value))
}

func builtinNewObject(self *_object, _ Value, argumentList []Value) Value {
	value := valueOfArrayIndex(argumentList, 0)
	switch value._valueType {
	case valueNull, valueUndefined:
	case valueNumber, valueString, valueBoolean:
		return toValue_object(self.runtime.toObject(value))
	case valueObject:
		return value
	default:
	}
	return toValue_object(self.runtime.newObject())
}

func builtinObject_valueOf(call FunctionCall) Value {
	return toValue_object(call.thisObject())
}

func builtinObject_hasOwnProperty(call FunctionCall) Value {
	propertyName := toString(call.Argument(0))
	thisObject := call.thisObject()
	return toValue_bool(thisObject.hasOwnProperty(propertyName))
}

func builtinObject_isPrototypeOf(call FunctionCall) Value {
	value := call.Argument(0)
	if !value.IsObject() {
		return FalseValue()
	}
	prototype := call.toObject(value).prototype
	thisObject := call.thisObject()
	for prototype != nil {
		if thisObject == prototype {
			return TrueValue()
		}
		prototype = prototype.prototype
	}
	return FalseValue()
}

func builtinObject_propertyIsEnumerable(call FunctionCall) Value {
	propertyName := toString(call.Argument(0))
	thisObject := call.thisObject()
	property := thisObject.getOwnProperty(propertyName)
	if property != nil && property.enumerable() {
		return TrueValue()
	}
	return FalseValue()
}

func builtinObject_toString(call FunctionCall) Value {
	result := ""
	if call.This.IsUndefined() {
		result = "[object Undefined]"
	} else if call.This.IsNull() {
		result = "[object Null]"
	} else {
		result = fmt.Sprintf("[object %s]", call.thisObject().class)
	}
	return toValue_string(result)
}

func builtinObject_toLocaleString(call FunctionCall) Value {
	toString := call.thisObject().get("toString")
	if !toString.isCallable() {
		panic(newTypeError())
	}
	return toString.call(call.This)
}

func builtinObject_getPrototypeOf(call FunctionCall) Value {
	objectValue := call.Argument(0)
	object := objectValue._object()
	if object == nil {
		panic(newTypeError())
	}

	if object.prototype == nil {
		return NullValue()
	}

	return toValue_object(object.prototype)
}

func builtinObject_getOwnPropertyDescriptor(call FunctionCall) Value {
	objectValue := call.Argument(0)
	object := objectValue._object()
	if object == nil {
		panic(newTypeError())
	}

	name := toString(call.Argument(1))
	descriptor := object.getOwnProperty(name)
	if descriptor == nil {
		return UndefinedValue()
	}
	return toValue_object(call.runtime.fromPropertyDescriptor(*descriptor))
}

func builtinObject_defineProperty(call FunctionCall) Value {
	objectValue := call.Argument(0)
	object := objectValue._object()
	if object == nil {
		panic(newTypeError())
	}
	name := toString(call.Argument(1))
	descriptor := toPropertyDescriptor(call.Argument(2))
	object.defineOwnProperty(name, descriptor, true)
	return objectValue
}

func builtinObject_defineProperties(call FunctionCall) Value {
	objectValue := call.Argument(0)
	object := objectValue._object()
	if object == nil {
		panic(newTypeError())
	}

	properties := call.runtime.toObject(call.Argument(1))
	properties.enumerate(false, func(name string) bool {
		descriptor := toPropertyDescriptor(properties.get(name))
		object.defineOwnProperty(name, descriptor, true)
		return true
	})

	return objectValue
}

func builtinObject_create(call FunctionCall) Value {
	prototypeValue := call.Argument(0)
	if !prototypeValue.IsNull() && !prototypeValue.IsObject() {
		panic(newTypeError())
	}

	object := call.runtime.newObject()
	object.prototype = prototypeValue._object()

	propertiesValue := call.Argument(1)
	if propertiesValue.IsDefined() {
		properties := call.runtime.toObject(propertiesValue)
		properties.enumerate(false, func(name string) bool {
			descriptor := toPropertyDescriptor(properties.get(name))
			object.defineOwnProperty(name, descriptor, true)
			return true
		})
	}

	return toValue_object(object)
}

func builtinObject_isExtensible(call FunctionCall) Value {
	object := call.Argument(0)
	if object := object._object(); object != nil {
		return toValue_bool(object.extensible)
	}
	panic(newTypeError())
}

func builtinObject_preventExtensions(call FunctionCall) Value {
	object := call.Argument(0)
	if object := object._object(); object != nil {
		object.extensible = false
	} else {
		panic(newTypeError())
	}
	return object
}

func builtinObject_isSealed(call FunctionCall) Value {
	object := call.Argument(0)
	if object := object._object(); object != nil {
		if object.extensible {
			return toValue_bool(false)
		}
		result := true
		object.enumerate(true, func(name string) bool {
			property := object.getProperty(name)
			if property.configurable() {
				result = false
			}
			return true
		})
		return toValue_bool(result)
	}
	panic(newTypeError())
}

func builtinObject_seal(call FunctionCall) Value {
	object := call.Argument(0)
	if object := object._object(); object != nil {
		object.enumerate(true, func(name string) bool {
			if property := object.getOwnProperty(name); nil != property && property.configurable() {
				property.configureOff()
				object.defineOwnProperty(name, *property, true)
			}
			return true
		})
		object.extensible = false
	} else {
		panic(newTypeError())
	}
	return object
}

func builtinObject_isFrozen(call FunctionCall) Value {
	object := call.Argument(0)
	if object := object._object(); object != nil {
		if object.extensible {
			return toValue_bool(false)
		}
		result := true
		object.enumerate(true, func(name string) bool {
			property := object.getProperty(name)
			if property.configurable() || property.writable() {
				result = false
			}
			return true
		})
		return toValue_bool(result)
	}
	panic(newTypeError())
}

func builtinObject_freeze(call FunctionCall) Value {
	object := call.Argument(0)
	if object := object._object(); object != nil {
		object.enumerate(true, func(name string) bool {
			if property, update := object.getOwnProperty(name), false; nil != property {
				if property.isDataDescriptor() && property.writable() {
					property.writeOff()
					update = true
				}
				if property.configurable() {
					property.configureOff()
					update = true
				}
				if update {
					object.defineOwnProperty(name, *property, true)
				}
			}
			return true
		})
		object.extensible = false
	} else {
		panic(newTypeError())
	}
	return object
}

func builtinObject_keys(call FunctionCall) Value {
	if object, keys := call.Argument(0)._object(), []Value(nil); nil != object {
		object.enumerate(false, func(name string) bool {
			keys = append(keys, toValue_string(name))
			return true
		})
		return toValue_object(call.runtime.newArrayOf(keys))
	}
	panic(newTypeError())
}

func builtinObject_getOwnPropertyNames(call FunctionCall) Value {
	if object, propertyNames := call.Argument(0)._object(), []Value(nil); nil != object {
		object.enumerate(true, func(name string) bool {
			if object.hasOwnProperty(name) {
				propertyNames = append(propertyNames, toValue_string(name))
			}
			return true
		})
		return toValue_object(call.runtime.newArrayOf(propertyNames))
	}
	panic(newTypeError())
}
