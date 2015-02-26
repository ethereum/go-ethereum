package otto

import (
	"errors"
	"reflect"

	"github.com/robertkrimen/otto/ast"
	"github.com/robertkrimen/otto/parser"
)

type _global struct {
	Object         *_object // Object( ... ), new Object( ... ) - 1 (length)
	Function       *_object // Function( ... ), new Function( ... ) - 1
	Array          *_object // Array( ... ), new Array( ... ) - 1
	String         *_object // String( ... ), new String( ... ) - 1
	Boolean        *_object // Boolean( ... ), new Boolean( ... ) - 1
	Number         *_object // Number( ... ), new Number( ... ) - 1
	Math           *_object
	Date           *_object // Date( ... ), new Date( ... ) - 7
	RegExp         *_object // RegExp( ... ), new RegExp( ... ) - 2
	Error          *_object // Error( ... ), new Error( ... ) - 1
	EvalError      *_object
	TypeError      *_object
	RangeError     *_object
	ReferenceError *_object
	SyntaxError    *_object
	URIError       *_object
	JSON           *_object

	ObjectPrototype         *_object // Object.prototype
	FunctionPrototype       *_object // Function.prototype
	ArrayPrototype          *_object // Array.prototype
	StringPrototype         *_object // String.prototype
	BooleanPrototype        *_object // Boolean.prototype
	NumberPrototype         *_object // Number.prototype
	DatePrototype           *_object // Date.prototype
	RegExpPrototype         *_object // RegExp.prototype
	ErrorPrototype          *_object // Error.prototype
	EvalErrorPrototype      *_object
	TypeErrorPrototype      *_object
	RangeErrorPrototype     *_object
	ReferenceErrorPrototype *_object
	SyntaxErrorPrototype    *_object
	URIErrorPrototype       *_object
}

type _runtime struct {
	Stack [](*_executionContext)

	GlobalObject      *_object
	GlobalEnvironment *_objectEnvironment

	Global _global

	eval *_object // The builtin eval, for determine indirect versus direct invocation

	Otto *Otto

	labels []string // FIXME
}

func (self *_runtime) EnterGlobalExecutionContext() {
	self.EnterExecutionContext(newExecutionContext(self.GlobalEnvironment, self.GlobalEnvironment, self.GlobalObject))
}

func (self *_runtime) EnterExecutionContext(scope *_executionContext) {
	self.Stack = append(self.Stack, scope)
}

func (self *_runtime) LeaveExecutionContext() {
	self.Stack = self.Stack[:len(self.Stack)-1]
}

func (self *_runtime) _executionContext(depth int) *_executionContext {
	if depth == 0 {
		return self.Stack[len(self.Stack)-1]
	}
	if len(self.Stack)-1+depth >= 0 {
		return self.Stack[len(self.Stack)-1+depth]
	}
	return nil
}

func (self *_runtime) EnterFunctionExecutionContext(function *_object, this Value) *_functionEnvironment {
	scopeEnvironment := function.functionValue().call.ScopeEnvironment()
	if scopeEnvironment == nil {
		scopeEnvironment = self.GlobalEnvironment
	}
	environment := self.newFunctionEnvironment(scopeEnvironment)
	var thisObject *_object
	switch this._valueType {
	case valueUndefined, valueNull:
		thisObject = self.GlobalObject
	default:
		thisObject = self.toObject(this)
	}
	self.EnterExecutionContext(newExecutionContext(environment, environment, thisObject))
	return environment
}

func (self *_runtime) EnterEvalExecutionContext(call FunctionCall) {
	// Skip the current function lexical/variable environment, which is of the function execution context call
	// to eval (the global execution context). Instead, execute in the context of where the eval was called,
	// which is essentially dynamic scoping
	parent := self._executionContext(-1)
	new := newExecutionContext(parent.LexicalEnvironment, parent.VariableEnvironment, parent.this)
	// FIXME Make passing through of self.GlobalObject more general? Whenever newExecutionContext is passed a nil object?
	new.eval = true
	self.EnterExecutionContext(new)
}

func (self *_runtime) GetValue(value Value) Value {
	if value.isReference() {
		return value.reference().GetValue()
	}
	return value
}

func (self *_runtime) PutValue(reference _reference, value Value) {
	if !reference.PutValue(value) {
		// Why? -- If reference.Base == nil
		strict := false
		self.GlobalObject.defineProperty(reference.GetName(), value, 0111, strict)
	}
}

func (self *_runtime) Call(function *_object, this Value, argumentList []Value, evalHint bool) Value {
	// Pass eval boolean through to EnterFunctionExecutionContext for further testing
	_functionEnvironment := self.EnterFunctionExecutionContext(function, this)
	defer func() {
		self.LeaveExecutionContext()
	}()

	if evalHint {
		evalHint = function == self.eval // If evalHint is true, then it IS a direct eval
	}
	callValue := function.functionValue().call.Dispatch(function, _functionEnvironment, self, this, argumentList, evalHint)
	if value, valid := callValue.value.(_result); valid {
		return value.value
	}
	return callValue
}

func (self *_runtime) tryCatchEvaluate(inner func() Value) (tryValue Value, exception bool) {
	// resultValue = The value of the block (e.g. the last statement)
	// throw = Something was thrown
	// throwValue = The value of what was thrown
	// other = Something that changes flow (return, break, continue) that is not a throw
	// Otherwise, some sort of unknown panic happened, we'll just propagate it
	defer func() {
		if caught := recover(); caught != nil {
			if exception, ok := caught.(*_exception); ok {
				caught = exception.eject()
			}
			switch caught := caught.(type) {
			case _error:
				exception = true
				tryValue = toValue_object(self.newError(caught.Name, caught.MessageValue()))
			//case *_syntaxError:
			//    exception = true
			//    tryValue = toValue_object(self.newError("SyntaxError", toValue_string(caught.Message)))
			case Value:
				exception = true
				tryValue = caught
			default:
				panic(caught)
			}
		}
	}()

	tryValue = inner()
	return
}

// _executionContext Proxy

func (self *_runtime) localGet(name string) Value {
	return self._executionContext(0).getValue(name)
}

func (self *_runtime) localSet(name string, value Value) {
	self._executionContext(0).setValue(name, value, false)
}

func (self *_runtime) VariableEnvironment() _environment {
	return self._executionContext(0).VariableEnvironment
}

func (self *_runtime) LexicalEnvironment() _environment {
	return self._executionContext(0).LexicalEnvironment
}

// toObject

func (self *_runtime) toObject(value Value) *_object {
	switch value._valueType {
	case valueEmpty, valueUndefined, valueNull:
		panic(newTypeError())
	case valueBoolean:
		return self.newBoolean(value)
	case valueString:
		return self.newString(value)
	case valueNumber:
		return self.newNumber(value)
	case valueObject:
		return value._object()
	}
	panic(newTypeError())
}

func (self *_runtime) objectCoerce(value Value) (*_object, error) {
	switch value._valueType {
	case valueUndefined:
		return nil, errors.New("undefined")
	case valueNull:
		return nil, errors.New("null")
	case valueBoolean:
		return self.newBoolean(value), nil
	case valueString:
		return self.newString(value), nil
	case valueNumber:
		return self.newNumber(value), nil
	case valueObject:
		return value._object(), nil
	}
	panic(newTypeError())
}

func checkObjectCoercible(value Value) {
	isObject, mustCoerce := testObjectCoercible(value)
	if !isObject && !mustCoerce {
		panic(newTypeError())
	}
}

// testObjectCoercible

func testObjectCoercible(value Value) (isObject bool, mustCoerce bool) {
	switch value._valueType {
	case valueReference, valueEmpty, valueNull, valueUndefined:
		return false, false
	case valueNumber, valueString, valueBoolean:
		isObject = false
		mustCoerce = true
	case valueObject:
		isObject = true
		mustCoerce = false
	}
	return
}

func (self *_runtime) ToValue(value interface{}) (Value, error) {
	result := UndefinedValue()
	err := catchPanic(func() {
		result = self.toValue(value)
	})
	return result, err
}

func (self *_runtime) toValue(value interface{}) Value {
	switch value := value.(type) {
	case Value:
		return value
	case func(FunctionCall) Value:
		return toValue_object(self.newNativeFunction("", value))
	case _nativeFunction:
		return toValue_object(self.newNativeFunction("", value))
	case Object, *Object, _object, *_object:
		// Nothing happens.
		// FIXME
	default:
		{
			value := reflect.ValueOf(value)
			switch value.Kind() {
			case reflect.Ptr:
				switch reflect.Indirect(value).Kind() {
				case reflect.Struct:
					return toValue_object(self.newGoStructObject(value))
				case reflect.Array:
					return toValue_object(self.newGoArray(value))
				}
			case reflect.Func:
				return toValue_object(self.newNativeFunction("", func(call FunctionCall) Value {
					args := make([]reflect.Value, len(call.ArgumentList))
					for i, a := range call.ArgumentList {
						args[i] = reflect.ValueOf(a.export())
					}

					retvals := value.Call(args)
					if len(retvals) > 1 {
						panic(newTypeError())
					} else if len(retvals) == 1 {
						return toValue(retvals[0].Interface())
					}
					return UndefinedValue()
				}))
			case reflect.Struct:
				return toValue_object(self.newGoStructObject(value))
			case reflect.Map:
				return toValue_object(self.newGoMapObject(value))
			case reflect.Slice:
				return toValue_object(self.newGoSlice(value))
			case reflect.Array:
				return toValue_object(self.newGoArray(value))
			}
		}
	}
	return toValue(value)
}

func (runtime *_runtime) newGoSlice(value reflect.Value) *_object {
	self := runtime.newGoSliceObject(value)
	self.prototype = runtime.Global.ArrayPrototype
	return self
}

func (runtime *_runtime) newGoArray(value reflect.Value) *_object {
	self := runtime.newGoArrayObject(value)
	self.prototype = runtime.Global.ArrayPrototype
	return self
}

func (runtime *_runtime) parse(filename string, src interface{}) (*ast.Program, error) {
	return parser.ParseFile(nil, filename, src, 0)
}

func (runtime *_runtime) cmpl_parse(filename string, src interface{}) (*_nodeProgram, error) {
	program, err := parser.ParseFile(nil, filename, src, 0)
	if err != nil {
		return nil, err
	}
	return cmpl_parse(program), nil
}

func (self *_runtime) parseSource(src interface{}) (*_nodeProgram, *ast.Program, error) {
	switch src := src.(type) {
	case *ast.Program:
		return nil, src, nil
	case *Script:
		return src.program, nil, nil
	}
	program, err := self.parse("", src)
	return nil, program, err
}

func (self *_runtime) cmpl_run(src interface{}) (Value, error) {
	result := UndefinedValue()
	cmpl_program, program, err := self.parseSource(src)
	if err != nil {
		return result, err
	}
	if cmpl_program == nil {
		cmpl_program = cmpl_parse(program)
	}
	err = catchPanic(func() {
		result = self.cmpl_evaluate_nodeProgram(cmpl_program)
	})
	switch result._valueType {
	case valueEmpty:
		result = UndefinedValue()
	case valueReference:
		result = self.GetValue(result)
	}
	return result, err
}

func (self *_runtime) parseThrow(err error) {
	if err == nil {
		return
	}
	switch err := err.(type) {
	case parser.ErrorList:
		{
			err := err[0]
			if err.Message == "Invalid left-hand side in assignment" {
				panic(newReferenceError(err.Message))
			}
			panic(newSyntaxError(err.Message))
		}
	}
	panic(newSyntaxError(err.Error()))
}

func (self *_runtime) parseOrThrow(source string) *ast.Program {
	program, err := self.parse("", source)
	self.parseThrow(err) // Will panic/throw appropriately
	return program
}

func (self *_runtime) cmpl_parseOrThrow(source string) *_nodeProgram {
	program, err := self.cmpl_parse("", source)
	self.parseThrow(err) // Will panic/throw appropriately
	return program
}
