package parser

import (
	"regexp"
	"testing"
)

func TestRegExp(t *testing.T) {
	tt(t, func() {
		{
			// err
			test := func(input string, expect interface{}) {
				_, err := TransformRegExp(input)
				is(err, expect)
			}

			test("[", "Unterminated character class")

			test("(", "Unterminated group")

			test("(?=)", "re2: Invalid (?=) <lookahead>")

			test("(?=)", "re2: Invalid (?=) <lookahead>")

			test("(?!)", "re2: Invalid (?!) <lookahead>")

			// An error anyway
			test("(?=", "re2: Invalid (?=) <lookahead>")

			test("\\1", "re2: Invalid \\1 <backreference>")

			test("\\90", "re2: Invalid \\90 <backreference>")

			test("\\9123456789", "re2: Invalid \\9123456789 <backreference>")

			test("\\(?=)", "Unmatched ')'")

			test(")", "Unmatched ')'")
		}

		{
			// err
			test := func(input, expect string, expectErr interface{}) {
				output, err := TransformRegExp(input)
				is(output, expect)
				is(err, expectErr)
			}

			test("(?!)", "(?!)", "re2: Invalid (?!) <lookahead>")

			test(")", "", "Unmatched ')'")

			test("(?!))", "", "re2: Invalid (?!) <lookahead>")

			test("\\0", "\\0", nil)

			test("\\1", "\\1", "re2: Invalid \\1 <backreference>")

			test("\\9123456789", "\\9123456789", "re2: Invalid \\9123456789 <backreference>")
		}

		{
			// err
			test := func(input string, expect string) {
				result, err := TransformRegExp(input)
				is(err, nil)
				if is(result, expect) {
					_, err := regexp.Compile(result)
					if !is(err, nil) {
						t.Log(result)
					}
				}
			}

			test("", "")

			test("abc", "abc")

			test(`\abc`, `abc`)

			test(`\a\b\c`, `a\bc`)

			test(`\x`, `x`)

			test(`\c`, `c`)

			test(`\cA`, `\x01`)

			test(`\cz`, `\x1a`)

			test(`\ca`, `\x01`)

			test(`\cj`, `\x0a`)

			test(`\ck`, `\x0b`)

			test(`\+`, `\+`)

			test(`[\b]`, `[\x08]`)

			test(`\u0z01\x\undefined`, `u0z01xundefined`)

			test(`\\|'|\r|\n|\t|\u2028|\u2029`, `\\|'|\r|\n|\t|\x{2028}|\x{2029}`)

			test("]", "]")

			test("}", "}")

			test("%", "%")

			test("(%)", "(%)")

			test("(?:[%\\s])", "(?:[%\\s])")

			test("[[]", "[[]")

			test("\\101", "\\x41")

			test("\\51", "\\x29")

			test("\\051", "\\x29")

			test("\\175", "\\x7d")

			test("\\04", "\\x04")

			test(`<%([\s\S]+?)%>`, `<%([\s\S]+?)%>`)

			test(`(.)^`, "(.)^")

			test(`<%-([\s\S]+?)%>|<%=([\s\S]+?)%>|<%([\s\S]+?)%>|$`, `<%-([\s\S]+?)%>|<%=([\s\S]+?)%>|<%([\s\S]+?)%>|$`)

			test(`\$`, `\$`)

			test(`[G-b]`, `[G-b]`)

			test(`[G-b\0]`, `[G-b\0]`)
		}
	})
}

func TestTransformRegExp(t *testing.T) {
	tt(t, func() {
		pattern, err := TransformRegExp(`\s+abc\s+`)
		is(err, nil)
		is(pattern, `\s+abc\s+`)
		is(regexp.MustCompile(pattern).MatchString("\t abc def"), true)
	})
}
