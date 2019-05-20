package otto

import (
	"testing"
)

func TestFunction(t *testing.T) {
	tt(t, func() {
		test, _ := test()

		test(`
            var abc = Object.getOwnPropertyDescriptor(Function, "prototype");
            [   [ typeof Function.prototype, typeof Function.prototype.length, Function.prototype.length ],
                [ abc.writable, abc.enumerable, abc.configurable ] ];
        `, "function,number,0,false,false,false")
	})
}

func Test_argumentList2parameterList(t *testing.T) {
	tt(t, func() {
		is(argumentList2parameterList([]Value{toValue("abc, def"), toValue("ghi")}), []string{"abc", "def", "ghi"})
	})
}

func TestFunction_new(t *testing.T) {
	tt(t, func() {
		test, _ := test()

		test(`raise:
            new Function({});
        `, "SyntaxError: Unexpected identifier")

		test(`
            var abc = Function("def, ghi", "jkl", "return def+ghi+jkl");
            [ typeof abc, abc instanceof Function, abc("ab", "ba", 1) ];
        `, "function,true,abba1")

		test(`raise:
            var abc = {
                toString: function() { throw 1; }
            };
            var def = {
                toString: function() { throw 2; }
            };
            var ghi = new Function(abc, def);
            ghi;
        `, "1")

		// S15.3.2.1_A3_T10
		test(`raise:
            var abc = {
                toString: function() { return "z;x"; }
            };
            var def = "return this";
            var ghi = new Function(abc, def);
            ghi;
        `, "SyntaxError: Unexpected token ;")

		test(`raise:
            var abc;
            var def = "return true";
            var ghi = new Function(null, def);
            ghi;
        `, "SyntaxError: Unexpected token null")
	})
}

func TestFunction_apply(t *testing.T) {
	tt(t, func() {
		test, _ := test()

		test(`Function.prototype.apply.length`, 2)
		test(`String.prototype.substring.apply("abc", [1, 11])`, "bc")
	})
}

func TestFunction_call(t *testing.T) {
	tt(t, func() {
		test, _ := test()

		test(`Function.prototype.call.length`, 1)
		test(`String.prototype.substring.call("abc", 1, 11)`, "bc")
	})
}

func TestFunctionArguments(t *testing.T) {
	tt(t, func() {
		test, _ := test()

		// Should not be able to delete arguments
		test(`
            function abc(def, arguments){
                delete def;
                return def;
            }
            abc(1);
        `, 1)

		// Again, should not be able to delete arguments
		test(`
            function abc(def){
                delete def;
                return def;
            }
            abc(1);
        `, 1)

		// Test typeof of a function argument
		test(`
            function abc(def, ghi, jkl){
                return typeof jkl
            }
            abc("1st", "2nd", "3rd", "4th", "5th");
        `, "string")

		test(`
            function abc(def, ghi, jkl){
                arguments[0] = 3.14;
                arguments[1] = 'Nothing happens';
                arguments[2] = 42;
                if (3.14 === def && 'Nothing happens' === ghi && 42 === jkl)
                    return true;
            }
            abc(-1, 4.2, 314);
        `, true)
	})
}

func TestFunctionDeclarationInFunction(t *testing.T) {
	tt(t, func() {
		test, _ := test()

		// Function declarations happen AFTER parameter/argument declarations
		// That is, a function declared within a function will shadow/overwrite
		// declared parameters

		test(`
            function abc(def){
                return def;
                function def(){
                    return 1;
                }
            }
            typeof abc();
        `, "function")
	})
}

func TestArguments_defineOwnProperty(t *testing.T) {
	tt(t, func() {
		test, _ := test()

		test(`
            var abc;
            var def = true;
            var ghi = {};
            (function (a, b, c) {
                Object.defineProperty(arguments, "0", {
                    value: 42,
                    writable: false,
                    enumerable: false,
                    configurable: false
                });
                Object.defineProperty(arguments, "1", {
                    value: 3.14,
                    configurable: true,
                    enumerable: true
                });
                abc = Object.getOwnPropertyDescriptor(arguments, "0");
                for (var name in arguments) {
                    ghi[name] = (ghi[name] || 0) + 1;
                    if (name === "0") {
                        def = false;
                    }
                }
            }(0, 1, 2));
            [ abc.value, abc.writable, abc.enumerable, abc.configurable, def, ghi["1"] ];
        `, "42,false,false,false,true,1")
	})
}

func TestFunction_bind(t *testing.T) {
	tt(t, func() {
		test, _ := test()

		defer mockUTC()()

		test(`
            abc = function(){
                return "abc";
            };
            def = abc.bind();
            [ typeof def.prototype, typeof def.hasOwnProperty, def.hasOwnProperty("caller"), def.hasOwnProperty("arguments"), def() ];
        `, "object,function,true,true,abc")

		test(`
            abc = function(){
                return arguments[1];
            };
            def = abc.bind(undefined, "abc");
            ghi = abc.bind(undefined, "abc", "ghi");
            [ def(), def("def"), ghi("def") ];
        `, ",def,ghi")

		test(`
            var abc = function () {};
            var ghi;
            try {
                Object.defineProperty(Function.prototype, "xyzzy", {
                    value: 1001,
                    writable: true,
                    enumerable: true,
                    configurable: true
                });
                var def = abc.bind({});
                ghi = !def.hasOwnProperty("xyzzy") && ghi.xyzzy === 1001;
            } finally {
                delete Function.prototype.xyzzy;
            }
            [ ghi ];
        `, "true")

		test(`
            var abc = function (def, ghi) {};
            var jkl = abc.bind({});
            var mno = abc.bind({}, 1, 2);
            [ jkl.length, mno.length ];
        `, "2,0")

		test(`raise:
            Math.bind();
        `, "TypeError: 'bind' is not a function")

		test(`
            function construct(fn, arguments) {
                var bound = Function.prototype.bind.apply(fn, [null].concat(arguments));
                return new bound();
            }
            var abc = construct(Date, [1957, 4, 27]);
            Object.prototype.toString.call(abc);
        `, "[object Date]")

		test(`
            var fn = function (x, y, z) {
                var result = {};
                result.abc = x + y + z;
                result.def = arguments[0] === "a" && arguments.length === 3;
                return result;
            };
            var newFn = Function.prototype.bind.call(fn, {}, "a", "b", "c");
            var result = new newFn();
            [ result.hasOwnProperty("abc"), result.hasOwnProperty("def"), result.abc, result.def ];
        `, "true,true,abc,true")

		test(`
            abc = function(){
                return "abc";
            };
            def = abc.bind();
            def.toString();
        `, "function () { [native code] }")
	})
}

func TestFunction_toString(t *testing.T) {
	tt(t, func() {
		test, _ := test()

		test(`raise:
            Function.prototype.toString.call(undefined);
        `, "TypeError")

		test(`
            abc = function()   {       return -1    ;
}
            1;
            abc.toString();
        `, "function()   {       return -1    ;\n}")
	})
}
