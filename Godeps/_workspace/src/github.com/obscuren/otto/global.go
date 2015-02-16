package otto

import (
	"strconv"
	"time"
)

var (
	prototypeValueObject   = interface{}(nil)
	prototypeValueFunction = _functionObject{
		call: _nativeCallFunction{"", func(_ FunctionCall) Value {
			return UndefinedValue()
		}},
	}
	prototypeValueString = _stringASCII("")
	// TODO Make this just false?
	prototypeValueBoolean = Value{
		_valueType: valueBoolean,
		value:      false,
	}
	prototypeValueNumber = Value{
		_valueType: valueNumber,
		value:      0,
	}
	prototypeValueDate = _dateObject{
		epoch: 0,
		isNaN: false,
		time:  time.Unix(0, 0).UTC(),
		value: Value{
			_valueType: valueNumber,
			value:      0,
		},
	}
	prototypeValueRegExp = _regExpObject{
		regularExpression: nil,
		global:            false,
		ignoreCase:        false,
		multiline:         false,
		source:            "",
		flags:             "",
	}
)

func newContext() *_runtime {

	self := &_runtime{}

	self.GlobalEnvironment = self.newObjectEnvironment(nil, nil)
	self.GlobalObject = self.GlobalEnvironment.Object

	self.EnterGlobalExecutionContext()

	_newContext(self)

	self.eval = self.GlobalObject.property["eval"].value.(Value).value.(*_object)
	self.GlobalObject.prototype = self.Global.ObjectPrototype
	//self.parser = ast.NewParser()

	return self
}

func (runtime *_runtime) newBaseObject() *_object {
	self := newObject(runtime, "")
	return self
}

func (runtime *_runtime) newClassObject(class string) *_object {
	return newObject(runtime, class)
}

func (runtime *_runtime) newPrimitiveObject(class string, value Value) *_object {
	self := runtime.newClassObject(class)
	self.value = value
	return self
}

func (self *_object) primitiveValue() Value {
	switch value := self.value.(type) {
	case Value:
		return value
	case _stringObject:
		return toValue_string(value.String())
	}
	return Value{}
}

func (self *_object) hasPrimitive() bool {
	switch self.value.(type) {
	case Value, _stringObject:
		return true
	}
	return false
}

func (runtime *_runtime) newObject() *_object {
	self := runtime.newClassObject("Object")
	self.prototype = runtime.Global.ObjectPrototype
	return self
}

func (runtime *_runtime) newArray(length uint32) *_object {
	self := runtime.newArrayObject(length)
	self.prototype = runtime.Global.ArrayPrototype
	return self
}

func (runtime *_runtime) newArrayOf(valueArray []Value) *_object {
	self := runtime.newArray(uint32(len(valueArray)))
	for index, value := range valueArray {
		if value.isEmpty() {
			continue
		}
		self.defineProperty(strconv.FormatInt(int64(index), 10), value, 0111, false)
	}
	return self
}

func (runtime *_runtime) newString(value Value) *_object {
	self := runtime.newStringObject(value)
	self.prototype = runtime.Global.StringPrototype
	return self
}

func (runtime *_runtime) newBoolean(value Value) *_object {
	self := runtime.newBooleanObject(value)
	self.prototype = runtime.Global.BooleanPrototype
	return self
}

func (runtime *_runtime) newNumber(value Value) *_object {
	self := runtime.newNumberObject(value)
	self.prototype = runtime.Global.NumberPrototype
	return self
}

func (runtime *_runtime) newRegExp(patternValue Value, flagsValue Value) *_object {

	pattern := ""
	flags := ""
	if object := patternValue._object(); object != nil && object.class == "RegExp" {
		if flagsValue.IsDefined() {
			panic(newTypeError("Cannot supply flags when constructing one RegExp from another"))
		}
		regExp := object.regExpValue()
		pattern = regExp.source
		flags = regExp.flags
	} else {
		if patternValue.IsDefined() {
			pattern = toString(patternValue)
		}
		if flagsValue.IsDefined() {
			flags = toString(flagsValue)
		}
	}

	return runtime._newRegExp(pattern, flags)
}

func (runtime *_runtime) _newRegExp(pattern string, flags string) *_object {
	self := runtime.newRegExpObject(pattern, flags)
	self.prototype = runtime.Global.RegExpPrototype
	return self
}

// TODO Should (probably) be one argument, right? This is redundant
func (runtime *_runtime) newDate(epoch float64) *_object {
	self := runtime.newDateObject(epoch)
	self.prototype = runtime.Global.DatePrototype
	return self
}

func (runtime *_runtime) newError(name string, message Value) *_object {
	var self *_object
	switch name {
	case "EvalError":
		return runtime.newEvalError(message)
	case "TypeError":
		return runtime.newTypeError(message)
	case "RangeError":
		return runtime.newRangeError(message)
	case "ReferenceError":
		return runtime.newReferenceError(message)
	case "SyntaxError":
		return runtime.newSyntaxError(message)
	case "URIError":
		return runtime.newURIError(message)
	}

	self = runtime.newErrorObject(message)
	self.prototype = runtime.Global.ErrorPrototype
	if name != "" {
		self.defineProperty("name", toValue_string(name), 0111, false)
	}
	return self
}

func (runtime *_runtime) newNativeFunction(name string, _nativeFunction _nativeFunction) *_object {
	self := runtime.newNativeFunctionObject(name, _nativeFunction, 0)
	self.prototype = runtime.Global.FunctionPrototype
	prototype := runtime.newObject()
	self.defineProperty("prototype", toValue_object(prototype), 0100, false)
	prototype.defineProperty("constructor", toValue_object(self), 0100, false)
	return self
}

func (runtime *_runtime) newNodeFunction(node *_nodeFunctionLiteral, scopeEnvironment _environment) *_object {
	// TODO Implement 13.2 fully
	self := runtime.newNodeFunctionObject(node, scopeEnvironment)
	self.prototype = runtime.Global.FunctionPrototype
	prototype := runtime.newObject()
	self.defineProperty("prototype", toValue_object(prototype), 0100, false)
	prototype.defineProperty("constructor", toValue_object(self), 0101, false)
	return self
}
