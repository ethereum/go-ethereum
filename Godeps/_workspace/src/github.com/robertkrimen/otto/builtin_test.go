package otto

import (
	"testing"
)

func TestString_substr(t *testing.T) {
	tt(t, func() {
		test, _ := test()

		test(`
            [
                "abc".substr(0,1), // "a"
                "abc".substr(0,2), // "ab"
                "abc".substr(0,3), // "abc"
                "abc".substr(0,4), // "abc"
                "abc".substr(0,9), // "abc"
            ];
        `, "a,ab,abc,abc,abc")

		test(`
            [
                "abc".substr(1,1), // "b"
                "abc".substr(1,2), // "bc"
                "abc".substr(1,3), // "bc"
                "abc".substr(1,4), // "bc"
                "abc".substr(1,9), // "bc"
            ];
        `, "b,bc,bc,bc,bc")

		test(`
            [
                "abc".substr(2,1), // "c"
                "abc".substr(2,2), // "c"
                "abc".substr(2,3), // "c"
                "abc".substr(2,4), // "c"
                "abc".substr(2,9), // "c"
            ];
        `, "c,c,c,c,c")

		test(`
            [
                "abc".substr(3,1), // ""
                "abc".substr(3,2), // ""
                "abc".substr(3,3), // ""
                "abc".substr(3,4), // ""
                "abc".substr(3,9), // ""
            ];
        `, ",,,,")

		test(`
            [
                "abc".substr(0), // "abc"
                "abc".substr(1), // "bc"
                "abc".substr(2), // "c"
                "abc".substr(3), // ""
                "abc".substr(9), // ""
            ];
        `, "abc,bc,c,,")

		test(`
            [
                "abc".substr(-9), // "abc"
                "abc".substr(-3), // "abc"
                "abc".substr(-2), // "bc"
                "abc".substr(-1), // "c"
            ];
        `, "abc,abc,bc,c")

		test(`
            [
                "abc".substr(-9, 1), // "a"
                "abc".substr(-3, 1), // "a"
                "abc".substr(-2, 1), // "b"
                "abc".substr(-1, 1), // "c"
                "abc".substr(-1, 2), // "c"
            ];
        `, "a,a,b,c,c")

		test(`"abcd".substr(3, 5)`, "d")
	})
}

func Test_builtin_escape(t *testing.T) {
	tt(t, func() {
		is(builtin_escape("abc"), "abc")

		is(builtin_escape("="), "%3D")

		is(builtin_escape("abc=%+32"), "abc%3D%25+32")

		is(builtin_escape("世界"), "%u4E16%u754C")
	})
}

func Test_builtin_unescape(t *testing.T) {
	tt(t, func() {
		is(builtin_unescape("abc"), "abc")

		is(builtin_unescape("=%3D"), "==")

		is(builtin_unescape("abc%3D%25+32"), "abc=%+32")

		is(builtin_unescape("%u4E16%u754C"), "世界")
	})
}

func TestGlobal_escape(t *testing.T) {
	tt(t, func() {
		test, _ := test()

		test(`
            [
                escape("abc"),          // "abc"
                escape("="),            // "%3D"
                escape("abc=%+32"),     // "abc%3D%25+32"
                escape("\u4e16\u754c"), // "%u4E16%u754C"
            ];
        `, "abc,%3D,abc%3D%25+32,%u4E16%u754C")
	})
}

func TestGlobal_unescape(t *testing.T) {
	tt(t, func() {
		test, _ := test()

		test(`
            [
                unescape("abc"),          // "abc"
                unescape("=%3D"),         // "=="
                unescape("abc%3D%25+32"), // "abc=%+32"
                unescape("%u4E16%u754C"), // "世界"
            ];
        `, "abc,==,abc=%+32,世界")
	})
}
