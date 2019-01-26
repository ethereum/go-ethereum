package common

import (
	"github.com/graph-gophers/graphql-go/errors"
)

// http://facebook.github.io/graphql/draft/#InputValueDefinition
type InputValue struct {
	Name    Ident
	Type    Type
	Default Literal
	Desc    string
	Loc     errors.Location
	TypeLoc errors.Location
}

type InputValueList []*InputValue

func (l InputValueList) Get(name string) *InputValue {
	for _, v := range l {
		if v.Name.Name == name {
			return v
		}
	}
	return nil
}

func ParseInputValue(l *Lexer) *InputValue {
	p := &InputValue{}
	p.Loc = l.Location()
	p.Desc = l.DescComment()
	p.Name = l.ConsumeIdentWithLoc()
	l.ConsumeToken(':')
	p.TypeLoc = l.Location()
	p.Type = ParseType(l)
	if l.Peek() == '=' {
		l.ConsumeToken('=')
		p.Default = ParseLiteral(l, true)
	}
	return p
}

type Argument struct {
	Name  Ident
	Value Literal
}

type ArgumentList []Argument

func (l ArgumentList) Get(name string) (Literal, bool) {
	for _, arg := range l {
		if arg.Name.Name == name {
			return arg.Value, true
		}
	}
	return nil, false
}

func (l ArgumentList) MustGet(name string) Literal {
	value, ok := l.Get(name)
	if !ok {
		panic("argument not found")
	}
	return value
}

func ParseArguments(l *Lexer) ArgumentList {
	var args ArgumentList
	l.ConsumeToken('(')
	for l.Peek() != ')' {
		name := l.ConsumeIdentWithLoc()
		l.ConsumeToken(':')
		value := ParseLiteral(l, false)
		args = append(args, Argument{Name: name, Value: value})
	}
	l.ConsumeToken(')')
	return args
}
