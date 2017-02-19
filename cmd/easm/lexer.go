package main

import (
	"fmt"
	"os"
	"strings"
	"unicode"
	"unicode/utf8"
)

// stateFn is used through the lifetime of the
// lexer to parse the different values at the
// current state.
type stateFn func(*Lexer) stateFn

// item is emitted when the lexer has discovered
// a new parsable item. These are delivered over
// the items channels of the lexer
type item struct {
	typ    itemType
	lineno int
	text   string
}

// itemType are the different types the lexer
// is able to parse and return.
type itemType int

const (
	eof              itemType = iota // end of file
	lineStart                        // emitted when a line starts
	lineEnd                          // emitted when a line ends
	invalidStatement                 // any invalid statement
	element                          // any element during element parsing
	label                            // label is emitted when a labal is found
	labelDef                         // label definition is emitted when a new label is found
	number                           // number is emitted when a number is found
	stringValue                      // stringValue is emitted when a string has been found

	Numbers            = "1234567890"                                           // characters representing any decimal number
	HexadecimalNumbers = Numbers + "aAbBcCdDeEfF"                               // characters representing any hexadecimal
	Alpha              = "abcdefghijklmnopqrstuwvxyzABCDEFGHIJKLMNOPQRSTUWVXYZ" // characters representing alphanumeric
)

// String implements stringer
func (it itemType) String() string {
	if int(it) > len(stringItemTypes) {
		return "invalid"
	}
	return stringItemTypes[it]
}

var stringItemTypes = []string{
	eof:              "EOF",
	invalidStatement: "invalid statement",
	element:          "element",
	lineEnd:          "end of line",
	lineStart:        "new line",
	label:            "label",
	labelDef:         "label definition",
	number:           "number",
	stringValue:      "string",
}

// Lexer is the basic construct for parsing
// source code and turning them in to tokens.
// Tokens are interpreted by the compiler.
type Lexer struct {
	input string // input contains the source code of the program

	items chan item // items is used to deliver tokens to the listener
	state stateFn   // the current state function

	lineno            int // current line number in the source file
	start, pos, width int // positions for lexing and returning value

	debug bool // flag for triggering debug output
}

// lex lexes the program by name with the given source. It returns a
// channel on which the items are delivered.
func lex(name string, source []byte, debug bool) <-chan item {
	ch := make(chan item)
	l := &Lexer{
		input: string(source),
		items: ch,
		state: lexLine,
		debug: debug,
	}
	go func() {
		l.emit(lineStart)
		for l.state != nil {
			l.state = l.state(l)
		}
		l.emit(eof)
		close(l.items)
	}()

	return ch
}

// next returns the next rune in the program's source.
func (l *Lexer) next() (rune rune) {
	if l.pos >= len(l.input) {
		l.width = 0
		return 0
	}
	rune, l.width = utf8.DecodeRuneInString(l.input[l.pos:])
	l.pos += l.width
	return rune
}

// backup backsup the last parsed element (multi-character)
func (l *Lexer) backup() {
	l.pos -= l.width
}

// peek returns the next rune but does not advance the seeker
func (l *Lexer) peek() rune {
	r := l.next()
	l.backup()
	return r
}

// ignore advances the seeker and ignores the value
func (l *Lexer) ignore() {
	l.start = l.pos
}

// Accepts checks whether the given input matches the next rune
func (l *Lexer) accept(valid string) bool {
	if strings.IndexRune(valid, l.next()) >= 0 {
		return true
	}

	l.backup()

	return false
}

// acceptRun will continue to advance the seeker until valid
// can no longer be met.
func (l *Lexer) acceptRun(valid string) {
	for strings.IndexRune(valid, l.next()) >= 0 {
	}
	l.backup()
}

// acceptRunUntil is the inverse of acceptRun and will continue
// to advance the seeker until the rune has been found.
func (l *Lexer) acceptRunUntil(until rune) bool {
	// Continues running until a rune is found
	for i := l.next(); strings.IndexRune(string(until), i) == -1; i = l.next() {
		if i == 0 {
			return false
		}
	}

	return true
}

// blob returns the current value
func (l *Lexer) blob() string {
	return l.input[l.start:l.pos]
}

// Emits a new item on to item channel for processing
func (l *Lexer) emit(t itemType) {
	item := item{t, l.lineno, l.blob()}

	if l.debug {
		fmt.Fprintf(os.Stderr, "%04d: (%-20v) %s\n", item.lineno, item.typ, item.text)
	}

	l.items <- item
	l.start = l.pos
}

// lexLine is state function for lexing lines
func lexLine(l *Lexer) stateFn {
	for {
		switch r := l.next(); {
		case r == '\n':
			l.emit(lineEnd)
			l.ignore()
			l.lineno++

			l.emit(lineStart)
		case isSpace(r):
			l.ignore()
		case isAlphaNumeric(r) || r == '_':
			return lexElement
		case isNumber(r):
			return lexNumber
		case r == '@':
			l.ignore()
			return lexLabel
		case r == '"':
			return lexInsideString
		default:
			return nil
		}
	}
}

// lexLabel parses the current label, emits and returns
// the lex text state function to advance the parsing
// process.
func lexLabel(l *Lexer) stateFn {
	l.acceptRun(Alpha + "_")

	l.emit(label)

	return lexLine
}

// lexInsideString lexes the inside of a string until
// until the state function finds the closing quote.
// It returns the lex text state function.
func lexInsideString(l *Lexer) stateFn {
	if l.acceptRunUntil('"') {
		l.emit(stringValue)
	}

	return lexLine
}

func lexNumber(l *Lexer) stateFn {
	acceptance := Numbers
	if l.accept("0") && l.accept("xX") {
		acceptance = HexadecimalNumbers
	}
	l.acceptRun(acceptance)

	l.emit(number)

	return lexLine
}

func lexElement(l *Lexer) stateFn {
	l.acceptRun(Alpha + "_" + Numbers)

	if l.peek() == ':' {
		l.emit(labelDef)

		l.accept(":")
		l.ignore()
	} else {
		l.emit(element)
	}
	return lexLine
}

func isAlphaNumeric(t rune) bool {
	return unicode.IsLetter(t)
}

func isSpace(t rune) bool {
	return unicode.IsSpace(t)
}

func isNumber(t rune) bool {
	return unicode.IsNumber(t)
}
