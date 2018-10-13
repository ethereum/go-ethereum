package query

import (
	"fmt"
	"text/scanner"

	"github.com/graph-gophers/graphql-go/errors"
	"github.com/graph-gophers/graphql-go/internal/common"
)

type Document struct {
	Operations OperationList
	Fragments  FragmentList
}

type OperationList []*Operation

func (l OperationList) Get(name string) *Operation {
	for _, f := range l {
		if f.Name.Name == name {
			return f
		}
	}
	return nil
}

type FragmentList []*FragmentDecl

func (l FragmentList) Get(name string) *FragmentDecl {
	for _, f := range l {
		if f.Name.Name == name {
			return f
		}
	}
	return nil
}

type Operation struct {
	Type       OperationType
	Name       common.Ident
	Vars       common.InputValueList
	Selections []Selection
	Directives common.DirectiveList
	Loc        errors.Location
}

type OperationType string

const (
	Query        OperationType = "QUERY"
	Mutation                   = "MUTATION"
	Subscription               = "SUBSCRIPTION"
)

type Fragment struct {
	On         common.TypeName
	Selections []Selection
}

type FragmentDecl struct {
	Fragment
	Name       common.Ident
	Directives common.DirectiveList
	Loc        errors.Location
}

type Selection interface {
	isSelection()
}

type Field struct {
	Alias           common.Ident
	Name            common.Ident
	Arguments       common.ArgumentList
	Directives      common.DirectiveList
	Selections      []Selection
	SelectionSetLoc errors.Location
}

type InlineFragment struct {
	Fragment
	Directives common.DirectiveList
	Loc        errors.Location
}

type FragmentSpread struct {
	Name       common.Ident
	Directives common.DirectiveList
	Loc        errors.Location
}

func (Field) isSelection()          {}
func (InlineFragment) isSelection() {}
func (FragmentSpread) isSelection() {}

func Parse(queryString string) (*Document, *errors.QueryError) {
	l := common.NewLexer(queryString)

	var doc *Document
	err := l.CatchSyntaxError(func() { doc = parseDocument(l) })
	if err != nil {
		return nil, err
	}

	return doc, nil
}

func parseDocument(l *common.Lexer) *Document {
	d := &Document{}
	l.Consume()
	for l.Peek() != scanner.EOF {
		if l.Peek() == '{' {
			op := &Operation{Type: Query, Loc: l.Location()}
			op.Selections = parseSelectionSet(l)
			d.Operations = append(d.Operations, op)
			continue
		}

		loc := l.Location()
		switch x := l.ConsumeIdent(); x {
		case "query":
			op := parseOperation(l, Query)
			op.Loc = loc
			d.Operations = append(d.Operations, op)

		case "mutation":
			d.Operations = append(d.Operations, parseOperation(l, Mutation))

		case "subscription":
			d.Operations = append(d.Operations, parseOperation(l, Subscription))

		case "fragment":
			frag := parseFragment(l)
			frag.Loc = loc
			d.Fragments = append(d.Fragments, frag)

		default:
			l.SyntaxError(fmt.Sprintf(`unexpected %q, expecting "fragment"`, x))
		}
	}
	return d
}

func parseOperation(l *common.Lexer, opType OperationType) *Operation {
	op := &Operation{Type: opType}
	op.Name.Loc = l.Location()
	if l.Peek() == scanner.Ident {
		op.Name = l.ConsumeIdentWithLoc()
	}
	op.Directives = common.ParseDirectives(l)
	if l.Peek() == '(' {
		l.ConsumeToken('(')
		for l.Peek() != ')' {
			loc := l.Location()
			l.ConsumeToken('$')
			iv := common.ParseInputValue(l)
			iv.Loc = loc
			op.Vars = append(op.Vars, iv)
		}
		l.ConsumeToken(')')
	}
	op.Selections = parseSelectionSet(l)
	return op
}

func parseFragment(l *common.Lexer) *FragmentDecl {
	f := &FragmentDecl{}
	f.Name = l.ConsumeIdentWithLoc()
	l.ConsumeKeyword("on")
	f.On = common.TypeName{Ident: l.ConsumeIdentWithLoc()}
	f.Directives = common.ParseDirectives(l)
	f.Selections = parseSelectionSet(l)
	return f
}

func parseSelectionSet(l *common.Lexer) []Selection {
	var sels []Selection
	l.ConsumeToken('{')
	for l.Peek() != '}' {
		sels = append(sels, parseSelection(l))
	}
	l.ConsumeToken('}')
	return sels
}

func parseSelection(l *common.Lexer) Selection {
	if l.Peek() == '.' {
		return parseSpread(l)
	}
	return parseField(l)
}

func parseField(l *common.Lexer) *Field {
	f := &Field{}
	f.Alias = l.ConsumeIdentWithLoc()
	f.Name = f.Alias
	if l.Peek() == ':' {
		l.ConsumeToken(':')
		f.Name = l.ConsumeIdentWithLoc()
	}
	if l.Peek() == '(' {
		f.Arguments = common.ParseArguments(l)
	}
	f.Directives = common.ParseDirectives(l)
	if l.Peek() == '{' {
		f.SelectionSetLoc = l.Location()
		f.Selections = parseSelectionSet(l)
	}
	return f
}

func parseSpread(l *common.Lexer) Selection {
	loc := l.Location()
	l.ConsumeToken('.')
	l.ConsumeToken('.')
	l.ConsumeToken('.')

	f := &InlineFragment{Loc: loc}
	if l.Peek() == scanner.Ident {
		ident := l.ConsumeIdentWithLoc()
		if ident.Name != "on" {
			fs := &FragmentSpread{
				Name: ident,
				Loc:  loc,
			}
			fs.Directives = common.ParseDirectives(l)
			return fs
		}
		f.On = common.TypeName{Ident: l.ConsumeIdentWithLoc()}
	}
	f.Directives = common.ParseDirectives(l)
	f.Selections = parseSelectionSet(l)
	return f
}
