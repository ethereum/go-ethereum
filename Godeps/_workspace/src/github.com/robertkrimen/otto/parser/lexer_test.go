package parser

import (
	"../terst"
	"testing"

	"github.com/robertkrimen/otto/file"
	"github.com/robertkrimen/otto/token"
)

var tt = terst.Terst
var is = terst.Is

func TestLexer(t *testing.T) {
	tt(t, func() {
		setup := func(src string) *_parser {
			parser := newParser("", src)
			return parser
		}

		test := func(src string, test ...interface{}) {
			parser := setup(src)
			for len(test) > 0 {
				tkn, literal, idx := parser.scan()
				if len(test) > 0 {
					is(tkn, test[0].(token.Token))
					test = test[1:]
				}
				if len(test) > 0 {
					is(literal, test[0].(string))
					test = test[1:]
				}
				if len(test) > 0 {
					// FIXME terst, Fix this so that cast to file.Idx is not necessary?
					is(idx, file.Idx(test[0].(int)))
					test = test[1:]
				}
			}
		}

		test("",
			token.EOF, "", 1,
		)

		test("1",
			token.NUMBER, "1", 1,
			token.EOF, "", 2,
		)

		test(".0",
			token.NUMBER, ".0", 1,
			token.EOF, "", 3,
		)

		test("abc",
			token.IDENTIFIER, "abc", 1,
			token.EOF, "", 4,
		)

		test("abc(1)",
			token.IDENTIFIER, "abc", 1,
			token.LEFT_PARENTHESIS, "", 4,
			token.NUMBER, "1", 5,
			token.RIGHT_PARENTHESIS, "", 6,
			token.EOF, "", 7,
		)

		test(".",
			token.PERIOD, "", 1,
			token.EOF, "", 2,
		)

		test("===.",
			token.STRICT_EQUAL, "", 1,
			token.PERIOD, "", 4,
			token.EOF, "", 5,
		)

		test(">>>=.0",
			token.UNSIGNED_SHIFT_RIGHT_ASSIGN, "", 1,
			token.NUMBER, ".0", 5,
			token.EOF, "", 7,
		)

		test(">>>=0.0.",
			token.UNSIGNED_SHIFT_RIGHT_ASSIGN, "", 1,
			token.NUMBER, "0.0", 5,
			token.PERIOD, "", 8,
			token.EOF, "", 9,
		)

		test("\"abc\"",
			token.STRING, "\"abc\"", 1,
			token.EOF, "", 6,
		)

		test("abc = //",
			token.IDENTIFIER, "abc", 1,
			token.ASSIGN, "", 5,
			token.EOF, "", 9,
		)

		test("abc = 1 / 2",
			token.IDENTIFIER, "abc", 1,
			token.ASSIGN, "", 5,
			token.NUMBER, "1", 7,
			token.SLASH, "", 9,
			token.NUMBER, "2", 11,
			token.EOF, "", 12,
		)

		test("xyzzy = 'Nothing happens.'",
			token.IDENTIFIER, "xyzzy", 1,
			token.ASSIGN, "", 7,
			token.STRING, "'Nothing happens.'", 9,
			token.EOF, "", 27,
		)

		test("abc = !false",
			token.IDENTIFIER, "abc", 1,
			token.ASSIGN, "", 5,
			token.NOT, "", 7,
			token.BOOLEAN, "false", 8,
			token.EOF, "", 13,
		)

		test("abc = !!true",
			token.IDENTIFIER, "abc", 1,
			token.ASSIGN, "", 5,
			token.NOT, "", 7,
			token.NOT, "", 8,
			token.BOOLEAN, "true", 9,
			token.EOF, "", 13,
		)

		test("abc *= 1",
			token.IDENTIFIER, "abc", 1,
			token.MULTIPLY_ASSIGN, "", 5,
			token.NUMBER, "1", 8,
			token.EOF, "", 9,
		)

		test("if 1 else",
			token.IF, "if", 1,
			token.NUMBER, "1", 4,
			token.ELSE, "else", 6,
			token.EOF, "", 10,
		)

		test("null",
			token.NULL, "null", 1,
			token.EOF, "", 5,
		)

		test(`"\u007a\x79\u000a\x78"`,
			token.STRING, "\"\\u007a\\x79\\u000a\\x78\"", 1,
			token.EOF, "", 23,
		)

		test(`"[First line \
Second line \
 Third line\
.     ]"
	`,
			token.STRING, "\"[First line \\\nSecond line \\\n Third line\\\n.     ]\"", 1,
			token.EOF, "", 53,
		)

		test("/",
			token.SLASH, "", 1,
			token.EOF, "", 2,
		)

		test("var abc = \"abc\uFFFFabc\"",
			token.VAR, "var", 1,
			token.IDENTIFIER, "abc", 5,
			token.ASSIGN, "", 9,
			token.STRING, "\"abc\uFFFFabc\"", 11,
			token.EOF, "", 22,
		)

		test(`'\t' === '\r'`,
			token.STRING, "'\\t'", 1,
			token.STRICT_EQUAL, "", 6,
			token.STRING, "'\\r'", 10,
			token.EOF, "", 14,
		)

		test(`var \u0024 = 1`,
			token.VAR, "var", 1,
			token.IDENTIFIER, "$", 5,
			token.ASSIGN, "", 12,
			token.NUMBER, "1", 14,
			token.EOF, "", 15,
		)

		test("10e10000",
			token.NUMBER, "10e10000", 1,
			token.EOF, "", 9,
		)

		test(`var if var class`,
			token.VAR, "var", 1,
			token.IF, "if", 5,
			token.VAR, "var", 8,
			token.KEYWORD, "class", 12,
			token.EOF, "", 17,
		)

		test(`-0`,
			token.MINUS, "", 1,
			token.NUMBER, "0", 2,
			token.EOF, "", 3,
		)

		test(`.01`,
			token.NUMBER, ".01", 1,
			token.EOF, "", 4,
		)

		test(`.01e+2`,
			token.NUMBER, ".01e+2", 1,
			token.EOF, "", 7,
		)

		test(";",
			token.SEMICOLON, "", 1,
			token.EOF, "", 2,
		)

		test(";;",
			token.SEMICOLON, "", 1,
			token.SEMICOLON, "", 2,
			token.EOF, "", 3,
		)

		test("//",
			token.EOF, "", 3,
		)

		test(";;//",
			token.SEMICOLON, "", 1,
			token.SEMICOLON, "", 2,
			token.EOF, "", 5,
		)

		test("1",
			token.NUMBER, "1", 1,
		)

		test("12 123",
			token.NUMBER, "12", 1,
			token.NUMBER, "123", 4,
		)

		test("1.2 12.3",
			token.NUMBER, "1.2", 1,
			token.NUMBER, "12.3", 5,
		)

		test("/ /=",
			token.SLASH, "", 1,
			token.QUOTIENT_ASSIGN, "", 3,
		)

		test(`"abc"`,
			token.STRING, `"abc"`, 1,
		)

		test(`'abc'`,
			token.STRING, `'abc'`, 1,
		)

		test("++",
			token.INCREMENT, "", 1,
		)

		test(">",
			token.GREATER, "", 1,
		)

		test(">=",
			token.GREATER_OR_EQUAL, "", 1,
		)

		test(">>",
			token.SHIFT_RIGHT, "", 1,
		)

		test(">>=",
			token.SHIFT_RIGHT_ASSIGN, "", 1,
		)

		test(">>>",
			token.UNSIGNED_SHIFT_RIGHT, "", 1,
		)

		test(">>>=",
			token.UNSIGNED_SHIFT_RIGHT_ASSIGN, "", 1,
		)

		test("1 \"abc\"",
			token.NUMBER, "1", 1,
			token.STRING, "\"abc\"", 3,
		)

		test(",",
			token.COMMA, "", 1,
		)

		test("1, \"abc\"",
			token.NUMBER, "1", 1,
			token.COMMA, "", 2,
			token.STRING, "\"abc\"", 4,
		)

		test("new abc(1, 3.14159);",
			token.NEW, "new", 1,
			token.IDENTIFIER, "abc", 5,
			token.LEFT_PARENTHESIS, "", 8,
			token.NUMBER, "1", 9,
			token.COMMA, "", 10,
			token.NUMBER, "3.14159", 12,
			token.RIGHT_PARENTHESIS, "", 19,
			token.SEMICOLON, "", 20,
		)

		test("1 == \"1\"",
			token.NUMBER, "1", 1,
			token.EQUAL, "", 3,
			token.STRING, "\"1\"", 6,
		)

		test("1\n[]\n",
			token.NUMBER, "1", 1,
			token.LEFT_BRACKET, "", 3,
			token.RIGHT_BRACKET, "", 4,
		)

		test("1\ufeff[]\ufeff",
			token.NUMBER, "1", 1,
			token.LEFT_BRACKET, "", 5,
			token.RIGHT_BRACKET, "", 6,
		)

		// ILLEGAL

		test(`3ea`,
			token.ILLEGAL, "3e", 1,
			token.IDENTIFIER, "a", 3,
			token.EOF, "", 4,
		)

		test(`3in`,
			token.ILLEGAL, "3", 1,
			token.IN, "in", 2,
			token.EOF, "", 4,
		)

		test("\"Hello\nWorld\"",
			token.ILLEGAL, "", 1,
			token.IDENTIFIER, "World", 8,
			token.ILLEGAL, "", 13,
			token.EOF, "", 14,
		)

		test("\u203f = 10",
			token.ILLEGAL, "", 1,
			token.ASSIGN, "", 5,
			token.NUMBER, "10", 7,
			token.EOF, "", 9,
		)

		test(`"\x0G"`,
			token.STRING, "\"\\x0G\"", 1,
			token.EOF, "", 7,
		)

	})
}
