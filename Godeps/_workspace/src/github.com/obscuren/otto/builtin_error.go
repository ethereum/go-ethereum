package otto

func (runtime *_runtime) newEvalError(message Value) *_object {
	self := runtime.newErrorObject(message)
	self.prototype = runtime.Global.EvalErrorPrototype
	return self
}

func builtinEvalError(call FunctionCall) Value {
	return toValue_object(call.runtime.newEvalError(call.Argument(0)))
}

func builtinNewEvalError(self *_object, _ Value, argumentList []Value) Value {
	return toValue_object(self.runtime.newEvalError(valueOfArrayIndex(argumentList, 0)))
}

func (runtime *_runtime) newTypeError(message Value) *_object {
	self := runtime.newErrorObject(message)
	self.prototype = runtime.Global.TypeErrorPrototype
	return self
}

func builtinTypeError(call FunctionCall) Value {
	return toValue_object(call.runtime.newTypeError(call.Argument(0)))
}

func builtinNewTypeError(self *_object, _ Value, argumentList []Value) Value {
	return toValue_object(self.runtime.newTypeError(valueOfArrayIndex(argumentList, 0)))
}

func (runtime *_runtime) newRangeError(message Value) *_object {
	self := runtime.newErrorObject(message)
	self.prototype = runtime.Global.RangeErrorPrototype
	return self
}

func builtinRangeError(call FunctionCall) Value {
	return toValue_object(call.runtime.newRangeError(call.Argument(0)))
}

func builtinNewRangeError(self *_object, _ Value, argumentList []Value) Value {
	return toValue_object(self.runtime.newRangeError(valueOfArrayIndex(argumentList, 0)))
}

func (runtime *_runtime) newURIError(message Value) *_object {
	self := runtime.newErrorObject(message)
	self.prototype = runtime.Global.URIErrorPrototype
	return self
}

func (runtime *_runtime) newReferenceError(message Value) *_object {
	self := runtime.newErrorObject(message)
	self.prototype = runtime.Global.ReferenceErrorPrototype
	return self
}

func builtinReferenceError(call FunctionCall) Value {
	return toValue_object(call.runtime.newReferenceError(call.Argument(0)))
}

func builtinNewReferenceError(self *_object, _ Value, argumentList []Value) Value {
	return toValue_object(self.runtime.newReferenceError(valueOfArrayIndex(argumentList, 0)))
}

func (runtime *_runtime) newSyntaxError(message Value) *_object {
	self := runtime.newErrorObject(message)
	self.prototype = runtime.Global.SyntaxErrorPrototype
	return self
}

func builtinSyntaxError(call FunctionCall) Value {
	return toValue_object(call.runtime.newSyntaxError(call.Argument(0)))
}

func builtinNewSyntaxError(self *_object, _ Value, argumentList []Value) Value {
	return toValue_object(self.runtime.newSyntaxError(valueOfArrayIndex(argumentList, 0)))
}

func builtinURIError(call FunctionCall) Value {
	return toValue_object(call.runtime.newURIError(call.Argument(0)))
}

func builtinNewURIError(self *_object, _ Value, argumentList []Value) Value {
	return toValue_object(self.runtime.newURIError(valueOfArrayIndex(argumentList, 0)))
}
