package otto

import (
	"fmt"
	"math"
	"strings"
	"testing"
)

func TestGlobal(t *testing.T) {
	tt(t, func() {
		test, vm := test()

		runtime := vm.vm.runtime

		{
			call := func(object interface{}, src string, argumentList ...interface{}) Value {
				var tgt *Object
				switch object := object.(type) {
				case Value:
					tgt = object.Object()
				case *Object:
					tgt = object
				case *_object:
					tgt = toValue_object(object).Object()
				default:
					panic("Here be dragons.")
				}
				value, err := tgt.Call(src, argumentList...)
				is(err, nil)
				return value
			}

			// FIXME enterGlobalScope
			if false {
				value := runtime.scope.lexical.getBinding("Object", false)._object().call(UndefinedValue(), []Value{toValue(runtime.newObject())}, false, nativeFrame)
				is(value.IsObject(), true)
				is(value, "[object Object]")
				is(value._object().prototype == runtime.global.ObjectPrototype, true)
				is(value._object().prototype == runtime.global.Object.get("prototype")._object(), true)
				is(value._object().get("toString"), "function toString() { [native code] }")
				is(call(value.Object(), "hasOwnProperty", "hasOwnProperty"), false)

				is(call(value._object().get("toString")._object().prototype, "toString"), "function () { [native code] }") // TODO Is this right?
				is(value._object().get("toString")._object().get("toString"), "function toString() { [native code] }")
				is(value._object().get("toString")._object().get("toString")._object(), "function toString() { [native code] }")

				is(call(value._object(), "propertyIsEnumerable", "isPrototypeOf"), false)
				value._object().put("xyzzy", toValue_string("Nothing happens."), false)
				is(call(value, "propertyIsEnumerable", "isPrototypeOf"), false)
				is(call(value, "propertyIsEnumerable", "xyzzy"), true)
				is(value._object().get("xyzzy"), "Nothing happens.")

				is(call(runtime.scope.lexical.getBinding("Object", false), "isPrototypeOf", value), false)
				is(call(runtime.scope.lexical.getBinding("Object", false)._object().get("prototype"), "isPrototypeOf", value), true)
				is(call(runtime.scope.lexical.getBinding("Function", false), "isPrototypeOf", value), false)

				is(runtime.newObject().prototype == runtime.global.Object.get("prototype")._object(), true)

				abc := runtime.newBoolean(toValue_bool(true))
				is(toValue_object(abc), "true") // TODO Call primitive?

				//def := runtime.localGet("Boolean")._object().Construct(UndefinedValue(), []Value{})
				//is(def, "false") // TODO Call primitive?
			}
		}

		test(`new Number().constructor == Number`, true)

		test(`this.hasOwnProperty`, "function hasOwnProperty() { [native code] }")

		test(`eval.length === 1`, true)
		test(`eval.prototype === undefined`, true)
		test(`raise: new eval()`, "TypeError: function eval() { [native code] } is not a constructor")

		test(`
            [
                [ delete undefined, undefined ],
                [ delete NaN, NaN ],
                [ delete Infinity, Infinity ],
            ];
        `, "false,,false,NaN,false,Infinity")

		test(`
            Object.getOwnPropertyNames(Function('return this')()).sort();
        `, "Array,Boolean,Date,Error,EvalError,Function,Infinity,JSON,Math,NaN,Number,Object,RangeError,ReferenceError,RegExp,String,SyntaxError,TypeError,URIError,console,decodeURI,decodeURIComponent,encodeURI,encodeURIComponent,escape,eval,isFinite,isNaN,parseFloat,parseInt,undefined,unescape")

		// __defineGetter__,__defineSetter__,__lookupGetter__,__lookupSetter__,constructor,hasOwnProperty,isPrototypeOf,propertyIsEnumerable,toLocaleString,toString,valueOf
		test(`
            Object.getOwnPropertyNames(Object.prototype).sort();
        `, "constructor,hasOwnProperty,isPrototypeOf,propertyIsEnumerable,toLocaleString,toString,valueOf")

		// arguments,caller,length,name,prototype
		test(`
            Object.getOwnPropertyNames(EvalError).sort();
        `, "length,prototype")

		test(`
            var abc = [];
            var def = [EvalError, RangeError, ReferenceError, SyntaxError, TypeError, URIError];
            for (constructor in def) {
                abc.push(def[constructor] === def[constructor].prototype.constructor);
            }
            def = [Array, Boolean, Date, Function, Number, Object, RegExp, String, SyntaxError];
            for (constructor in def) {
                abc.push(def[constructor] === def[constructor].prototype.constructor);
            }
            abc;
        `, "true,true,true,true,true,true,true,true,true,true,true,true,true,true,true")

		test(`
            [ Array.prototype.constructor === Array, Array.constructor === Function ];
        `, "true,true")

		test(`
            [ Number.prototype.constructor === Number, Number.constructor === Function ];
        `, "true,true")

		test(`
            [ Function.prototype.constructor === Function, Function.constructor === Function ];
        `, "true,true")
	})
}

func TestGlobalLength(t *testing.T) {
	tt(t, func() {
		test, _ := test()

		test(`
            [ Object.length, Function.length, RegExp.length, Math.length ];
        `, "1,1,2,")
	})
}

func TestGlobalError(t *testing.T) {
	tt(t, func() {
		test, _ := test()

		test(`
            [ TypeError.length, TypeError(), TypeError("Nothing happens.") ];
        `, "1,TypeError,TypeError: Nothing happens.")

		test(`
            [ URIError.length, URIError(), URIError("Nothing happens.") ];
        `, "1,URIError,URIError: Nothing happens.")
	})
}

func TestGlobalReadOnly(t *testing.T) {
	tt(t, func() {
		test, _ := test()

		test(`Number.POSITIVE_INFINITY`, math.Inf(1))

		test(`
            Number.POSITIVE_INFINITY = 1;
        `, 1)

		test(`Number.POSITIVE_INFINITY`, math.Inf(1))

		test(`
            Number.POSITIVE_INFINITY = 1;
            Number.POSITIVE_INFINITY;
        `, math.Inf(1))
	})
}

func Test_isNaN(t *testing.T) {
	tt(t, func() {
		test, _ := test()

		test(`isNaN(0)`, false)
		test(`isNaN("Xyzzy")`, true)
		test(`isNaN()`, true)
		test(`isNaN(NaN)`, true)
		test(`isNaN(Infinity)`, false)

		test(`isNaN.length === 1`, true)
		test(`isNaN.prototype === undefined`, true)
	})
}

func Test_isFinite(t *testing.T) {
	tt(t, func() {
		test, _ := test()

		test(`isFinite(0)`, true)
		test(`isFinite("Xyzzy")`, false)
		test(`isFinite()`, false)
		test(`isFinite(NaN)`, false)
		test(`isFinite(Infinity)`, false)
		test(`isFinite(new Number(451));`, true)

		test(`isFinite.length === 1`, true)
		test(`isFinite.prototype === undefined`, true)
	})
}

func Test_parseInt(t *testing.T) {
	tt(t, func() {
		test, _ := test()

		test(`parseInt("0")`, 0)
		test(`parseInt("11")`, 11)
		test(`parseInt(" 11")`, 11)
		test(`parseInt("11 ")`, 11)
		test(`parseInt(" 11 ")`, 11)
		test(`parseInt(" 11\n")`, 11)
		test(`parseInt(" 11\n", 16)`, 17)

		test(`parseInt("Xyzzy")`, _NaN)

		test(`parseInt(" 0x11\n", 16)`, 17)
		test(`parseInt("0x0aXyzzy", 16)`, 10)
		test(`parseInt("0x1", 0)`, 1)
		test(`parseInt("0x10000000000000000000", 16)`, float64(75557863725914323419136))

		test(`parseInt.length === 2`, true)
		test(`parseInt.prototype === undefined`, true)
	})
}

func Test_parseFloat(t *testing.T) {
	tt(t, func() {
		test, _ := test()

		test(`parseFloat("0")`, 0)
		test(`parseFloat("11")`, 11)
		test(`parseFloat(" 11")`, 11)
		test(`parseFloat("11 ")`, 11)
		test(`parseFloat(" 11 ")`, 11)
		test(`parseFloat(" 11\n")`, 11)
		test(`parseFloat(" 11\n", 16)`, 11)
		test(`parseFloat("11.1")`, 11.1)

		test(`parseFloat("Xyzzy")`, _NaN)

		test(`parseFloat(" 0x11\n", 16)`, 0)
		test(`parseFloat("0x0a")`, 0)
		test(`parseFloat("0x0aXyzzy")`, 0)
		test(`parseFloat("Infinity")`, _Infinity)
		test(`parseFloat("infinity")`, _NaN)
		test(`parseFloat("0x")`, 0)
		test(`parseFloat("11x")`, 11)
		test(`parseFloat("Infinity1")`, _Infinity)

		test(`parseFloat.length === 1`, true)
		test(`parseFloat.prototype === undefined`, true)
	})
}

func Test_encodeURI(t *testing.T) {
	tt(t, func() {
		test, _ := test()

		test(`encodeURI("http://example.com/ Nothing happens.")`, "http://example.com/%20Nothing%20happens.")
		test(`encodeURI("http://example.com/ _^#")`, "http://example.com/%20_%5E#")
		test(`encodeURI(String.fromCharCode("0xE000"))`, "%EE%80%80")
		test(`encodeURI(String.fromCharCode("0xFFFD"))`, "%EF%BF%BD")
		test(`raise: encodeURI(String.fromCharCode("0xDC00"))`, "URIError: URI malformed")

		test(`encodeURI.length === 1`, true)
		test(`encodeURI.prototype === undefined`, true)
	})
}

func Test_encodeURIComponent(t *testing.T) {
	tt(t, func() {
		test, _ := test()

		test(`encodeURIComponent("http://example.com/ Nothing happens.")`, "http%3A%2F%2Fexample.com%2F%20Nothing%20happens.")
		test(`encodeURIComponent("http://example.com/ _^#")`, "http%3A%2F%2Fexample.com%2F%20_%5E%23")
	})
}

func Test_decodeURI(t *testing.T) {
	tt(t, func() {
		test, _ := test()

		test(`decodeURI(encodeURI("http://example.com/ Nothing happens."))`, "http://example.com/ Nothing happens.")
		test(`decodeURI(encodeURI("http://example.com/ _^#"))`, "http://example.com/ _^#")
		test(`raise: decodeURI("http://example.com/ _^#%")`, "URIError: URI malformed")
		test(`raise: decodeURI("%DF%7F")`, "URIError: URI malformed")
		for _, check := range strings.Fields("+ %3B %2F %3F %3A %40 %26 %3D %2B %24 %2C %23") {
			test(fmt.Sprintf(`decodeURI("%s")`, check), check)
		}

		test(`decodeURI.length === 1`, true)
		test(`decodeURI.prototype === undefined`, true)
	})
}

func Test_decodeURIComponent(t *testing.T) {
	tt(t, func() {
		test, _ := test()

		test(`decodeURIComponent(encodeURI("http://example.com/ Nothing happens."))`, "http://example.com/ Nothing happens.")
		test(`decodeURIComponent(encodeURI("http://example.com/ _^#"))`, "http://example.com/ _^#")

		test(`decodeURIComponent.length === 1`, true)
		test(`decodeURIComponent.prototype === undefined`, true)

		test(`
        var global = Function('return this')();
        var abc = Object.getOwnPropertyDescriptor(global, "decodeURIComponent");
        [ abc.value === global.decodeURIComponent, abc.writable, abc.enumerable, abc.configurable ];
    `, "true,true,false,true")
	})
}

func TestGlobal_skipEnumeration(t *testing.T) {
	tt(t, func() {
		test, _ := test()

		test(`
            var found = [];
            for (var test in this) {
                if (false ||
                    test === 'NaN' ||
                    test === 'undefined' ||
                    test === 'Infinity' ||
                    false) {
                    found.push(test)
                }
            }
            found.length;
        `, 0)

		test(`
            var found = [];
            for (var test in this) {
                if (false ||
                    test === 'Object' ||
                    test === 'Function' ||
                    test === 'String' ||
                    test === 'Number' ||
                    test === 'Array' ||
                    test === 'Boolean' ||
                    test === 'Date' ||
                    test === 'RegExp' ||
                    test === 'Error' ||
                    test === 'EvalError' ||
                    test === 'RangeError' ||
                    test === 'ReferenceError' ||
                    test === 'SyntaxError' ||
                    test === 'TypeError' ||
                    test === 'URIError' ||
                    false) {
                    found.push(test)
                }
            }
            found.length;
        `, 0)
	})
}
