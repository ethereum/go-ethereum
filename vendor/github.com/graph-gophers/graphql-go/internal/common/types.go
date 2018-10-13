package common

import (
	"github.com/graph-gophers/graphql-go/errors"
)

type Type interface {
	Kind() string
	String() string
}

type List struct {
	OfType Type
}

type NonNull struct {
	OfType Type
}

type TypeName struct {
	Ident
}

func (*List) Kind() string     { return "LIST" }
func (*NonNull) Kind() string  { return "NON_NULL" }
func (*TypeName) Kind() string { panic("TypeName needs to be resolved to actual type") }

func (t *List) String() string    { return "[" + t.OfType.String() + "]" }
func (t *NonNull) String() string { return t.OfType.String() + "!" }
func (*TypeName) String() string  { panic("TypeName needs to be resolved to actual type") }

func ParseType(l *Lexer) Type {
	t := parseNullType(l)
	if l.Peek() == '!' {
		l.ConsumeToken('!')
		return &NonNull{OfType: t}
	}
	return t
}

func parseNullType(l *Lexer) Type {
	if l.Peek() == '[' {
		l.ConsumeToken('[')
		ofType := ParseType(l)
		l.ConsumeToken(']')
		return &List{OfType: ofType}
	}

	return &TypeName{Ident: l.ConsumeIdentWithLoc()}
}

type Resolver func(name string) Type

func ResolveType(t Type, resolver Resolver) (Type, *errors.QueryError) {
	switch t := t.(type) {
	case *List:
		ofType, err := ResolveType(t.OfType, resolver)
		if err != nil {
			return nil, err
		}
		return &List{OfType: ofType}, nil
	case *NonNull:
		ofType, err := ResolveType(t.OfType, resolver)
		if err != nil {
			return nil, err
		}
		return &NonNull{OfType: ofType}, nil
	case *TypeName:
		refT := resolver(t.Name)
		if refT == nil {
			err := errors.Errorf("Unknown type %q.", t.Name)
			err.Rule = "KnownTypeNames"
			err.Locations = []errors.Location{t.Loc}
			return nil, err
		}
		return refT, nil
	default:
		return t, nil
	}
}
