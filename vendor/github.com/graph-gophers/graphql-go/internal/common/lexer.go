package common

import (
	"fmt"
	"strings"
	"text/scanner"

	"github.com/graph-gophers/graphql-go/errors"
)

type syntaxError string

type Lexer struct {
	sc          *scanner.Scanner
	next        rune
	descComment string
}

type Ident struct {
	Name string
	Loc  errors.Location
}

func NewLexer(s string) *Lexer {
	sc := &scanner.Scanner{
		Mode: scanner.ScanIdents | scanner.ScanInts | scanner.ScanFloats | scanner.ScanStrings,
	}
	sc.Init(strings.NewReader(s))

	return &Lexer{sc: sc}
}

func (l *Lexer) CatchSyntaxError(f func()) (errRes *errors.QueryError) {
	defer func() {
		if err := recover(); err != nil {
			if err, ok := err.(syntaxError); ok {
				errRes = errors.Errorf("syntax error: %s", err)
				errRes.Locations = []errors.Location{l.Location()}
				return
			}
			panic(err)
		}
	}()

	f()
	return
}

func (l *Lexer) Peek() rune {
	return l.next
}

// Consume whitespace and tokens equivalent to whitespace (e.g. commas and comments).
//
// Consumed comment characters will build the description for the next type or field encountered.
// The description is available from `DescComment()`, and will be reset every time `Consume()` is
// executed.
func (l *Lexer) Consume() {
	l.descComment = ""
	for {
		l.next = l.sc.Scan()

		if l.next == ',' {
			// Similar to white space and line terminators, commas (',') are used to improve the
			// legibility of source text and separate lexical tokens but are otherwise syntactically and
			// semantically insignificant within GraphQL documents.
			//
			// http://facebook.github.io/graphql/draft/#sec-Insignificant-Commas
			continue
		}

		if l.next == '#' {
			// GraphQL source documents may contain single-line comments, starting with the '#' marker.
			//
			// A comment can contain any Unicode code point except `LineTerminator` so a comment always
			// consists of all code points starting with the '#' character up to but not including the
			// line terminator.

			l.consumeComment()
			continue
		}

		break
	}
}

func (l *Lexer) ConsumeIdent() string {
	name := l.sc.TokenText()
	l.ConsumeToken(scanner.Ident)
	return name
}

func (l *Lexer) ConsumeIdentWithLoc() Ident {
	loc := l.Location()
	name := l.sc.TokenText()
	l.ConsumeToken(scanner.Ident)
	return Ident{name, loc}
}

func (l *Lexer) ConsumeKeyword(keyword string) {
	if l.next != scanner.Ident || l.sc.TokenText() != keyword {
		l.SyntaxError(fmt.Sprintf("unexpected %q, expecting %q", l.sc.TokenText(), keyword))
	}
	l.Consume()
}

func (l *Lexer) ConsumeLiteral() *BasicLit {
	lit := &BasicLit{Type: l.next, Text: l.sc.TokenText()}
	l.Consume()
	return lit
}

func (l *Lexer) ConsumeToken(expected rune) {
	if l.next != expected {
		l.SyntaxError(fmt.Sprintf("unexpected %q, expecting %s", l.sc.TokenText(), scanner.TokenString(expected)))
	}
	l.Consume()
}

func (l *Lexer) DescComment() string {
	return l.descComment
}

func (l *Lexer) SyntaxError(message string) {
	panic(syntaxError(message))
}

func (l *Lexer) Location() errors.Location {
	return errors.Location{
		Line:   l.sc.Line,
		Column: l.sc.Column,
	}
}

// consumeComment consumes all characters from `#` to the first encountered line terminator.
// The characters are appended to `l.descComment`.
func (l *Lexer) consumeComment() {
	if l.next != '#' {
		return
	}

	// TODO: count and trim whitespace so we can dedent any following lines.
	if l.sc.Peek() == ' ' {
		l.sc.Next()
	}

	if l.descComment != "" {
		// TODO: use a bytes.Buffer or strings.Builder instead of this.
		l.descComment += "\n"
	}

	for {
		next := l.sc.Next()
		if next == '\r' || next == '\n' || next == scanner.EOF {
			break
		}

		// TODO: use a bytes.Buffer or strings.Build instead of this.
		l.descComment += string(next)
	}
}
