package common

import (
	"strconv"
	"strings"
	"text/scanner"

	"github.com/graph-gophers/graphql-go/errors"
)

type Literal interface {
	Value(vars map[string]interface{}) interface{}
	String() string
	Location() errors.Location
}

type BasicLit struct {
	Type rune
	Text string
	Loc  errors.Location
}

func (lit *BasicLit) Value(vars map[string]interface{}) interface{} {
	switch lit.Type {
	case scanner.Int:
		value, err := strconv.ParseInt(lit.Text, 10, 32)
		if err != nil {
			panic(err)
		}
		return int32(value)

	case scanner.Float:
		value, err := strconv.ParseFloat(lit.Text, 64)
		if err != nil {
			panic(err)
		}
		return value

	case scanner.String:
		value, err := strconv.Unquote(lit.Text)
		if err != nil {
			panic(err)
		}
		return value

	case scanner.Ident:
		switch lit.Text {
		case "true":
			return true
		case "false":
			return false
		default:
			return lit.Text
		}

	default:
		panic("invalid literal")
	}
}

func (lit *BasicLit) String() string {
	return lit.Text
}

func (lit *BasicLit) Location() errors.Location {
	return lit.Loc
}

type ListLit struct {
	Entries []Literal
	Loc     errors.Location
}

func (lit *ListLit) Value(vars map[string]interface{}) interface{} {
	entries := make([]interface{}, len(lit.Entries))
	for i, entry := range lit.Entries {
		entries[i] = entry.Value(vars)
	}
	return entries
}

func (lit *ListLit) String() string {
	entries := make([]string, len(lit.Entries))
	for i, entry := range lit.Entries {
		entries[i] = entry.String()
	}
	return "[" + strings.Join(entries, ", ") + "]"
}

func (lit *ListLit) Location() errors.Location {
	return lit.Loc
}

type ObjectLit struct {
	Fields []*ObjectLitField
	Loc    errors.Location
}

type ObjectLitField struct {
	Name  Ident
	Value Literal
}

func (lit *ObjectLit) Value(vars map[string]interface{}) interface{} {
	fields := make(map[string]interface{}, len(lit.Fields))
	for _, f := range lit.Fields {
		fields[f.Name.Name] = f.Value.Value(vars)
	}
	return fields
}

func (lit *ObjectLit) String() string {
	entries := make([]string, 0, len(lit.Fields))
	for _, f := range lit.Fields {
		entries = append(entries, f.Name.Name+": "+f.Value.String())
	}
	return "{" + strings.Join(entries, ", ") + "}"
}

func (lit *ObjectLit) Location() errors.Location {
	return lit.Loc
}

type NullLit struct {
	Loc errors.Location
}

func (lit *NullLit) Value(vars map[string]interface{}) interface{} {
	return nil
}

func (lit *NullLit) String() string {
	return "null"
}

func (lit *NullLit) Location() errors.Location {
	return lit.Loc
}

type Variable struct {
	Name string
	Loc  errors.Location
}

func (v Variable) Value(vars map[string]interface{}) interface{} {
	return vars[v.Name]
}

func (v Variable) String() string {
	return "$" + v.Name
}

func (v *Variable) Location() errors.Location {
	return v.Loc
}

func ParseLiteral(l *Lexer, constOnly bool) Literal {
	loc := l.Location()
	switch l.Peek() {
	case '$':
		if constOnly {
			l.SyntaxError("variable not allowed")
			panic("unreachable")
		}
		l.ConsumeToken('$')
		return &Variable{l.ConsumeIdent(), loc}

	case scanner.Int, scanner.Float, scanner.String, scanner.Ident:
		lit := l.ConsumeLiteral()
		if lit.Type == scanner.Ident && lit.Text == "null" {
			return &NullLit{loc}
		}
		lit.Loc = loc
		return lit
	case '-':
		l.ConsumeToken('-')
		lit := l.ConsumeLiteral()
		lit.Text = "-" + lit.Text
		lit.Loc = loc
		return lit
	case '[':
		l.ConsumeToken('[')
		var list []Literal
		for l.Peek() != ']' {
			list = append(list, ParseLiteral(l, constOnly))
		}
		l.ConsumeToken(']')
		return &ListLit{list, loc}

	case '{':
		l.ConsumeToken('{')
		var fields []*ObjectLitField
		for l.Peek() != '}' {
			name := l.ConsumeIdentWithLoc()
			l.ConsumeToken(':')
			value := ParseLiteral(l, constOnly)
			fields = append(fields, &ObjectLitField{name, value})
		}
		l.ConsumeToken('}')
		return &ObjectLit{fields, loc}

	default:
		l.SyntaxError("invalid value")
		panic("unreachable")
	}
}
