package otto

import (
	"errors"
	"fmt"
	"math"
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
	debugger     func(*Otto)
	random       func() float64

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

// convertNumeric converts numeric parameter val from js to that of type t if it is safe to do so, otherwise it panics.
// This allows literals (int64), bitwise values (int32) and the general form (float64) of javascript numerics to be passed as parameters to go functions easily.
func convertNumeric(val reflect.Value, t reflect.Type) reflect.Value {
	if val.Kind() == t.Kind() {
		return val
	}

	if val.Kind() == reflect.Interface {
		val = reflect.ValueOf(val.Interface())
	}

	switch val.Kind() {
	case reflect.Float32, reflect.Float64:
		f64 := val.Float()
		switch t.Kind() {
		case reflect.Float64:
			return reflect.ValueOf(f64)
		case reflect.Float32:
			if reflect.Zero(t).OverflowFloat(f64) {
				panic("converting float64 to float32 would overflow")
			}

			return val.Convert(t)
		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64, reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
			i64 := int64(f64)
			if float64(i64) != f64 {
				panic(fmt.Sprintf("converting %v to %v would cause loss of precision", val.Type(), t))
			}

			// The float represents an integer
			val = reflect.ValueOf(i64)
		default:
			panic(fmt.Sprintf("cannot convert %v to %v", val.Type(), t))
		}
	}

	switch val.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		i64 := val.Int()
		switch t.Kind() {
		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
			if reflect.Zero(t).OverflowInt(i64) {
				panic(fmt.Sprintf("converting %v to %v would overflow", val.Type(), t))
			}
			return val.Convert(t)
		case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
			if i64 < 0 {
				panic(fmt.Sprintf("converting %v to %v would underflow", val.Type(), t))
			}
			if reflect.Zero(t).OverflowUint(uint64(i64)) {
				panic(fmt.Sprintf("converting %v to %v would overflow", val.Type(), t))
			}
			return val.Convert(t)
		}

	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		u64 := val.Uint()
		switch t.Kind() {
		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
			if u64 > math.MaxInt64 || reflect.Zero(t).OverflowInt(int64(u64)) {
				panic(fmt.Sprintf("converting %v to %v would overflow", val.Type(), t))
			}
			return val.Convert(t)
		case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
			if reflect.Zero(t).OverflowUint(u64) {
				panic(fmt.Sprintf("converting %v to %v would overflow", val.Type(), t))
			}
			return val.Convert(t)
		}
	}

	panic(fmt.Sprintf("unsupported type %v for numeric conversion", val.Type()))
}

// callParamConvert converts request val to type t if possible.
// If the conversion fails due to overflow or type miss-match then it panics.
// If no conversion is known then the original value is returned.
func callParamConvert(val reflect.Value, t reflect.Type) reflect.Value {
	if val.Kind() == reflect.Interface {
		val = reflect.ValueOf(val.Interface())
	}

	switch t.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Float32, reflect.Float64:
		if val.Kind() == t.Kind() {
			// Types already match
			return val
		}
		return convertNumeric(val, t)
	case reflect.Slice:
		if val.Kind() != reflect.Slice {
			// Conversion from none slice type to slice not possible
			panic(fmt.Sprintf("cannot use %v as type %v", val, t))
		}
	default:
		// No supported conversion
		return val
	}

	elemType := t.Elem()
	switch elemType.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Float32, reflect.Float64, reflect.Slice:
		// Attempt to convert to slice of the type t
		s := reflect.MakeSlice(reflect.SliceOf(elemType), val.Len(), val.Len())
		for i := 0; i < val.Len(); i++ {
			s.Index(i).Set(callParamConvert(val.Index(i), elemType))
		}

		return s
	}

	// Not a slice type we can convert
	return val
}

// callSliceRequired returns true if CallSlice is required instead of Call.
func callSliceRequired(param reflect.Type, val reflect.Value) bool {
	vt := val.Type()
	for param.Kind() == reflect.Slice {
		if val.Kind() == reflect.Interface {
			val = reflect.ValueOf(val.Interface())
			vt = val.Type()
		}

		if vt.Kind() != reflect.Slice {
			return false
		}

		vt = vt.Elem()
		if val.Kind() != reflect.Invalid {
			if val.Len() > 0 {
				val = val.Index(0)
			} else {
				val = reflect.Value{}
			}
		}
		param = param.Elem()
	}

	return true
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
					argsCount := len(call.ArgumentList)
					in := make([]reflect.Value, argsCount)
					t := value.Type()
					callSlice := false
					paramsCount := t.NumIn()
					lastParam := paramsCount - 1
					lastArg := argsCount - 1
					isVariadic := t.IsVariadic()
					for i, value := range call.ArgumentList {
						var paramType reflect.Type
						if isVariadic && i == lastArg && argsCount == paramsCount {
							// Variadic functions last parameter and parameter numbers match incoming args
							paramType = t.In(lastArg)
							val := reflect.ValueOf(value.export())
							callSlice = callSliceRequired(paramType, val)
							if callSlice {
								in[i] = callParamConvert(reflect.ValueOf(value.export()), paramType)
								continue
							}
						}

						if i >= lastParam {
							if isVariadic {
								paramType = t.In(lastParam).Elem()
							} else {
								paramType = t.In(lastParam)
							}
						} else {
							paramType = t.In(i)
						}
						in[i] = callParamConvert(reflect.ValueOf(value.export()), paramType)
					}

					var out []reflect.Value
					if callSlice {
						out = value.CallSlice(in)
					} else {
						out = value.Call(in)
					}

					l := len(out)
					switch l {
					case 0:
						return Value{}
					case 1:
						return self.toValue(out[0].Interface())
					}

					// Return an array of the values to emulate multi value return.
					// In the future this can be used along side destructuring assignment.
					s := make([]interface{}, l)
					for i, v := range out {
						s[i] = self.toValue(v.Interface())
					}
					return self.toValue(s)
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

func (self *_runtime) cmpl_runOrEval(src interface{}, eval bool) (Value, error) {
	result := Value{}
	cmpl_program, program, err := self.parseSource(src)
	if err != nil {
		return result, err
	}
	if cmpl_program == nil {
		cmpl_program = cmpl_parse(program)
	}
	err = catchPanic(func() {
		result = self.cmpl_evaluate_nodeProgram(cmpl_program, eval)
	})
	switch result.kind {
	case valueEmpty:
		result = Value{}
	case valueReference:
		result = result.resolve()
	}
	return result, err
}

func (self *_runtime) cmpl_run(src interface{}) (Value, error) {
	return self.cmpl_runOrEval(src, false)
}

func (self *_runtime) cmpl_eval(src interface{}) (Value, error) {
	return self.cmpl_runOrEval(src, true)
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
