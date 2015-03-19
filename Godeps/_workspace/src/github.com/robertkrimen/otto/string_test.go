package otto

import (
	"testing"
)

func TestString(t *testing.T) {
	tt(t, func() {
		test, _ := test()

		test(`
            abc = (new String("xyzzy")).length;
            def = new String().length;
            ghi = new String("Nothing happens.").length;
        `)
		test("abc", 5)
		test("def", 0)
		test("ghi", 16)
		test(`"".length`, 0)
		test(`"a\uFFFFbc".length`, 4)
		test(`String(+0)`, "0")
		test(`String(-0)`, "0")
		test(`""+-0`, "0")
		test(`
            var abc = Object.getOwnPropertyDescriptor(String, "prototype");
            [   [ typeof String.prototype ],
                [ abc.writable, abc.enumerable, abc.configurable ] ];
        `, "object,false,false,false")
	})
}

func TestString_charAt(t *testing.T) {
	tt(t, func() {
		test, _ := test()

		test(`
            abc = "xyzzy".charAt(0)
            def = "xyzzy".charAt(11)
        `)
		test("abc", "x")
		test("def", "")
	})
}

func TestString_charCodeAt(t *testing.T) {
	tt(t, func() {
		test, _ := test()

		test(`
            abc = "xyzzy".charCodeAt(0)
            def = "xyzzy".charCodeAt(11)
        `)
		test("abc", 120)
		test("def", _NaN)
	})
}

func TestString_fromCharCode(t *testing.T) {
	tt(t, func() {
		test, _ := test()

		test(`String.fromCharCode()`, []uint16{})
		test(`String.fromCharCode(88, 121, 122, 122, 121)`, []uint16{88, 121, 122, 122, 121}) // FIXME terst, Double-check these...
		test(`String.fromCharCode("88", 121, 122, 122.05, 121)`, []uint16{88, 121, 122, 122, 121})
		test(`String.fromCharCode("88", 121, 122, NaN, 121)`, []uint16{88, 121, 122, 0, 121})
		test(`String.fromCharCode("0x21")`, []uint16{33})
		test(`String.fromCharCode(-1).charCodeAt(0)`, 65535)
		test(`String.fromCharCode(65535).charCodeAt(0)`, 65535)
		test(`String.fromCharCode(65534).charCodeAt(0)`, 65534)
		test(`String.fromCharCode(4294967295).charCodeAt(0)`, 65535)
		test(`String.fromCharCode(4294967294).charCodeAt(0)`, 65534)
		test(`String.fromCharCode(0x0024) === "$"`, true)
	})
}

func TestString_concat(t *testing.T) {
	tt(t, func() {
		test, _ := test()

		test(`"".concat()`, "")
		test(`"".concat("abc", "def")`, "abcdef")
		test(`"".concat("abc", undefined, "def")`, "abcundefineddef")
	})
}

func TestString_indexOf(t *testing.T) {
	tt(t, func() {
		test, _ := test()

		test(`"".indexOf("")`, 0)
		test(`"".indexOf("", 11)`, 0)
		test(`"abc".indexOf("")`, 0)
		test(`"abc".indexOf("", 11)`, 3)
		test(`"abc".indexOf("a")`, 0)
		test(`"abc".indexOf("bc")`, 1)
		test(`"abc".indexOf("bc", 11)`, -1)
		test(`"$$abcdabcd".indexOf("ab", function(){return -Infinity;}())`, 2)
		test(`"$$abcdabcd".indexOf("ab", function(){return NaN;}())`, 2)

		test(`
            var abc = {toString:function(){return "\u0041B";}}
            var def = {valueOf:function(){return true;}}
            var ghi = "ABB\u0041BABAB";
            var jkl;
            with(ghi) {
                jkl = indexOf(abc, def);
            }
            jkl;
        `, 3)
	})
}

func TestString_lastIndexOf(t *testing.T) {
	tt(t, func() {
		test, _ := test()

		test(`"".lastIndexOf("")`, 0)
		test(`"".lastIndexOf("", 11)`, 0)
		test(`"abc".lastIndexOf("")`, 3)
		test(`"abc".lastIndexOf("", 11)`, 3)
		test(`"abc".lastIndexOf("a")`, 0)
		test(`"abc".lastIndexOf("bc")`, 1)
		test(`"abc".lastIndexOf("bc", 11)`, 1)
		test(`"abc".lastIndexOf("bc", 0)`, -1)
		test(`"abc".lastIndexOf("abcabcabc", 2)`, -1)
		test(`"abc".lastIndexOf("abc", 0)`, 0)
		test(`"abc".lastIndexOf("abc", 1)`, 0)
		test(`"abc".lastIndexOf("abc", 2)`, 0)
		test(`"abc".lastIndexOf("abc", 3)`, 0)

		test(`
            abc = new Object(true);
            abc.lastIndexOf = String.prototype.lastIndexOf;
            abc.lastIndexOf(true, false);
        `, 0)
	})
}

func TestString_match(t *testing.T) {
	tt(t, func() {
		test, _ := test()

		test(`"abc____abc_abc___".match(/__abc/)`, "__abc")
		test(`"abc___abc_abc__abc__abc".match(/abc/g)`, "abc,abc,abc,abc,abc")
		test(`"abc____abc_abc___".match(/__abc/g)`, "__abc")
		test(`
            abc = /abc/g
            "abc___abc_abc__abc__abc".match(abc)
        `, "abc,abc,abc,abc,abc")
		test(`abc.lastIndex`, 23)
	})
}

func TestString_replace(t *testing.T) {
	tt(t, func() {
		test, _ := test()

		test(`"abc_abc".replace(/abc/, "$&123")`, "abc123_abc")
		test(`"abc_abc".replace(/abc/g, "$&123")`, "abc123_abc123")
		test(`"abc_abc_".replace(/abc/g, "$&123")`, "abc123_abc123_")
		test(`"_abc_abc_".replace(/abc/g, "$&123")`, "_abc123_abc123_")
		test(`"abc".replace(/abc/, "$&123")`, "abc123")
		test(`"abc_".replace(/abc/, "$&123")`, "abc123_")
		test("\"^abc$\".replace(/abc/, \"$`def\")", "^^def$")
		test("\"^abc$\".replace(/abc/, \"def$`\")", "^def^$")
		test(`"_abc_abd_".replace(/ab(c|d)/g, "$1")`, "_c_d_")
		test(`
            "_abc_abd_".replace(/ab(c|d)/g, function(){
            })
        `, "_undefined_undefined_")

		test(`"b".replace(/(a)?(b)?/, "_$1_")`, "__")
		test(`
            "b".replace(/(a)?(b)?/, function(a, b, c, d, e, f){
                return [a, b, c, d, e, f]
            })
        `, "b,,b,0,b,")

		test(`
            var abc = 'She sells seashells by the seashore.';
            var def = /sh/;
            [ abc.replace(def, "$'" + 'sch') ];
        `, "She sells seaells by the seashore.schells by the seashore.")
	})
}

func TestString_search(t *testing.T) {
	tt(t, func() {
		test, _ := test()

		test(`"abc".search(/abc/)`, 0)
		test(`"abc".search(/def/)`, -1)
		test(`"abc".search(/c$/)`, 2)
		test(`"abc".search(/$/)`, 3)
	})
}

func TestString_split(t *testing.T) {
	tt(t, func() {
		test, _ := test()

		test(`"abc".split("", 1)`, "a")
		test(`"abc".split("", 2)`, "a,b")
		test(`"abc".split("", 3)`, "a,b,c")
		test(`"abc".split("", 4)`, "a,b,c")
		test(`"abc".split("", 11)`, "a,b,c")
		test(`"abc".split("", 0)`, "")
		test(`"abc".split("")`, "a,b,c")

		test(`"abc".split(undefined)`, "abc")

		test(`"__1__3_1__2__".split("_")`, ",,1,,3,1,,2,,")

		test(`"__1__3_1__2__".split(/_/)`, ",,1,,3,1,,2,,")

		test(`"ab".split(/a*/)`, ",b")

		test(`_ = "A<B>bold</B>and<CODE>coded</CODE>".split(/<(\/)?([^<>]+)>/)`, "A,,B,bold,/,B,and,,CODE,coded,/,CODE,")
		test(`_.length`, 13)
		test(`_[1] === undefined`, true)
		test(`_[12] === ""`, true)

		test(`
            var abc = new String("one-1 two-2 three-3");
            var def = abc.split(new RegExp);

            [ def.constructor === Array, abc.length, def.length, def.join('') ];
        `, "true,19,19,one-1 two-2 three-3")
	})
}

func TestString_slice(t *testing.T) {
	tt(t, func() {
		test, _ := test()

		test(`"abc".slice()`, "abc")
		test(`"abc".slice(0)`, "abc")
		test(`"abc".slice(0,11)`, "abc")
		test(`"abc".slice(0,-1)`, "ab")
		test(`"abc".slice(-1,11)`, "c")
		test(`abc = "abc"; abc.slice(abc.length+1, 0)`, "")
	})
}

func TestString_substring(t *testing.T) {
	tt(t, func() {
		test, _ := test()

		test(`"abc".substring()`, "abc")
		test(`"abc".substring(0)`, "abc")
		test(`"abc".substring(0,11)`, "abc")
		test(`"abc".substring(11,0)`, "abc")
		test(`"abc".substring(0,-1)`, "")
		test(`"abc".substring(-1,11)`, "abc")
		test(`"abc".substring(11,1)`, "bc")
		test(`"abc".substring(1)`, "bc")
		test(`"abc".substring(Infinity, Infinity)`, "")
	})
}

func TestString_toCase(t *testing.T) {
	tt(t, func() {
		test, _ := test()

		test(`"abc".toLowerCase()`, "abc")
		test(`"ABC".toLowerCase()`, "abc")
		test(`"abc".toLocaleLowerCase()`, "abc")
		test(`"ABC".toLocaleLowerCase()`, "abc")
		test(`"abc".toUpperCase()`, "ABC")
		test(`"ABC".toUpperCase()`, "ABC")
		test(`"abc".toLocaleUpperCase()`, "ABC")
		test(`"ABC".toLocaleUpperCase()`, "ABC")
	})
}

func Test_floatToString(t *testing.T) {
	tt(t, func() {
		test, _ := test()

		test(`String(-1234567890)`, "-1234567890")
		test(`-+String(-(-1234567890))`, -1234567890)
		test(`String(-1e128)`, "-1e+128")
		test(`String(0.12345)`, "0.12345")
		test(`String(-0.00000012345)`, "-1.2345e-7")
		test(`String(0.0000012345)`, "0.0000012345")
		test(`String(1000000000000000000000)`, "1e+21")
		test(`String(1e21)`, "1e+21")
		test(`String(1E21)`, "1e+21")
		test(`String(-1000000000000000000000)`, "-1e+21")
		test(`String(-1e21)`, "-1e+21")
		test(`String(-1E21)`, "-1e+21")
		test(`String(0.0000001)`, "1e-7")
		test(`String(1e-7)`, "1e-7")
		test(`String(1E-7)`, "1e-7")
		test(`String(-0.0000001)`, "-1e-7")
		test(`String(-1e-7)`, "-1e-7")
		test(`String(-1E-7)`, "-1e-7")
	})
}

func TestString_indexing(t *testing.T) {
	tt(t, func() {
		test, _ := test()

		// Actually a test of stringToArrayIndex, under the hood.
		test(`
            abc = new String("abc");
            index = Math.pow(2, 32);
            [ abc.length, abc[index], abc[index+1], abc[index+2], abc[index+3] ];
        `, "3,,,,")
	})
}

func TestString_trim(t *testing.T) {
	tt(t, func() {
		test, _ := test()

		test(`'    \n abc   \t \n'.trim();`, "abc")
		test(`"		abc\u000B".trim()`, "abc")
		test(`"abc ".trim()`, "abc")
		test(`
            var a = "\u180Eabc \u000B "
            var b = a.trim()
            a.length + b.length
        `, 10)
	})
}

func TestString_trimLeft(t *testing.T) {
	tt(t, func() {
		test, _ := test()

		test(`"		abc\u000B".trimLeft()`, "abc\u000B")
		test(`"abc ".trimLeft()`, "abc ")
		test(`
            var a = "\u180Eabc \u000B "
            var b = a.trimLeft()
            a.length + b.length
        `, 13)
	})
}

func TestString_trimRight(t *testing.T) {
	tt(t, func() {
		test, _ := test()

		test(`"		abc\u000B".trimRight()`, "		abc")
		test(`" abc ".trimRight()`, " abc")
		test(`
            var a = "\u180Eabc \u000B "
            var b = a.trimRight()
            a.length + b.length
        `, 11)
	})
}

func TestString_localeCompare(t *testing.T) {
	tt(t, func() {
		test, _ := test()

		test(`'a'.localeCompare('c');`, -1)
		test(`'c'.localeCompare('a');`, 1)
		test(`'a'.localeCompare('a');`, 0)
	})
}
