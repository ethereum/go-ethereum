package otto

import (
	"errors"
	"reflect"
	"sync"

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
	global       _global
	globalObject *_object
	globalStash  *_objectStash
	scope        *_scope
	otto         *Otto
	eval         *_object // The builtin eval, for determine indirect versus direct invocation

	labels []string // FIXME
	lck    sync.Mutex
}

func (self *_runtime) enterScope(scope *_scope) {
	scope.outer = self.scope
	self.scope = scope
}

func (self *_runtime) leaveScope() {
	self.scope = self.scope.outer
}

// FIXME This is used in two places (cloning)
func (self *_runtime) enterGlobalScope() {
	self.enterScope(newScope(self.globalStash, self.globalStash, self.globalObject))
}

func (self *_runtime) enterFunctionScope(outer _stash, this Value) *_fnStash {
	if outer == nil {
		outer = self.globalStash
	}
	stash := self.newFunctionStash(outer)
	var thisObject *_object
	switch this.kind {
	case valueUndefined, valueNull:
		thisObject = self.globalObject
	default:
		thisObject = self.toObject(this)
	}
	self.enterScope(newScope(stash, stash, thisObject))
	return stash
}

func (self *_runtime) putValue(reference _reference, value Value) {
	name := reference.putValue(value)
	if name != "" {
		// Why? -- If reference.base == nil
		// strict = false
		self.globalObject.defineProperty(name, value, 0111, false)
	}
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
				tryValue = toValue_object(self.newError(caught.name, caught.messageValue()))
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

// toObject

func (self *_runtime) toObject(value Value) *_object {
	switch value.kind {
	case valueEmpty, valueUndefined, valueNull:
		panic(self.panicTypeError())
	case valueBoolean:
		return self.newBoolean(value)
	case valueString:
		return self.newString(value)
	case valueNumber:
		return self.newNumber(value)
	case valueObject:
		return value._object()
	}
	panic(self.panicTypeError())
}

func (self *_runtime) objectCoerce(value Value) (*_object, error) {
	switch value.kind {
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
	panic(self.panicTypeError())
}

func checkObjectCoercible(rt *_runtime, value Value) {
	isObject, mustCoerce := testObjectCoercible(value)
	if !isObject && !mustCoerce {
		panic(rt.panicTypeError())
	}
}

// testObjectCoercible

func testObjectCoercible(value Value) (isObject bool, mustCoerce bool) {
	switch value.kind {
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

func (self *_runtime) safeToValue(value interface{}) (Value, error) {
	result := Value{}
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
		// FIXME We should really figure out what can come here.
		// This catch-all is ugly.
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
				// TODO Maybe cache this?
				return toValue_object(self.newNativeFunction("", func(call FunctionCall) Value {
					in := make([]reflect.Value, len(call.ArgumentList))
					for i, value := range call.ArgumentList {
						in[i] = reflect.ValueOf(value.export())
					}

					out := value.Call(in)
					if len(out) == 1 {
						return self.toValue(out[0].Interface())
					} else if len(out) == 0 {
						return Value{}
					}

					panic(call.runtime.panicTypeError())
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
	self.prototype = runtime.global.ArrayPrototype
	return self
}

func (runtime *_runtime) newGoArray(value reflect.Value) *_object {
	self := runtime.newGoArrayObject(value)
	self.prototype = runtime.global.ArrayPrototype
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
	result := Value{}
	cmpl_program, program, err := self.parseSource(src)
	if err != nil {
		return result, err
	}
	if cmpl_program == nil {
		cmpl_program = cmpl_parse(program)
	}
	err = catchPanic(func() {
		result = self.cmpl_evaluate_nodeProgram(cmpl_program, false)
	})
	switch result.kind {
	case valueEmpty:
		result = Value{}
	case valueReference:
		result = result.resolve()
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
				panic(self.panicReferenceError(err.Message))
			}
			panic(self.panicSyntaxError(err.Message))
		}
	}
	panic(self.panicSyntaxError(err.Error()))
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
