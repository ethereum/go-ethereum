package schema

import (
	"fmt"
	"text/scanner"

	"github.com/graph-gophers/graphql-go/errors"
	"github.com/graph-gophers/graphql-go/internal/common"
)

// Schema represents a GraphQL service's collective type system capabilities.
// A schema is defined in terms of the types and directives it supports as well as the root
// operation types for each kind of operation: `query`, `mutation`, and `subscription`.
//
// For a more formal definition, read the relevant section in the specification:
//
// http://facebook.github.io/graphql/draft/#sec-Schema
type Schema struct {
	// EntryPoints determines the place in the type system where `query`, `mutation`, and
	// `subscription` operations begin.
	//
	// http://facebook.github.io/graphql/draft/#sec-Root-Operation-Types
	//
	// NOTE: The specification refers to this concept as "Root Operation Types".
	// TODO: Rename the `EntryPoints` field to `RootOperationTypes` to align with spec terminology.
	EntryPoints map[string]NamedType

	// Types are the fundamental unit of any GraphQL schema.
	// There are six kinds of named types, and two wrapping types.
	//
	// http://facebook.github.io/graphql/draft/#sec-Types
	Types map[string]NamedType

	// TODO: Type extensions?
	// http://facebook.github.io/graphql/draft/#sec-Type-Extensions

	// Directives are used to annotate various parts of a GraphQL document as an indicator that they
	// should be evaluated differently by a validator, executor, or client tool such as a code
	// generator.
	//
	// http://facebook.github.io/graphql/draft/#sec-Type-System.Directives
	Directives map[string]*DirectiveDecl

	entryPointNames map[string]string
	objects         []*Object
	unions          []*Union
	enums           []*Enum
}

// Resolve a named type in the schema by its name.
func (s *Schema) Resolve(name string) common.Type {
	return s.Types[name]
}

// NamedType represents a type with a name.
//
// http://facebook.github.io/graphql/draft/#NamedType
type NamedType interface {
	common.Type
	TypeName() string
	Description() string
}

// Scalar types represent primitive leaf values (e.g. a string or an integer) in a GraphQL type
// system.
//
// GraphQL responses take the form of a hierarchical tree; the leaves on these trees are GraphQL
// scalars.
//
// http://facebook.github.io/graphql/draft/#sec-Scalars
type Scalar struct {
	Name string
	Desc string
	// TODO: Add a list of directives?
}

// Object types represent a list of named fields, each of which yield a value of a specific type.
//
// GraphQL queries are hierarchical and composed, describing a tree of information.
// While Scalar types describe the leaf values of these hierarchical types, Objects describe the
// intermediate levels.
//
// http://facebook.github.io/graphql/draft/#sec-Objects
type Object struct {
	Name       string
	Interfaces []*Interface
	Fields     FieldList
	Desc       string
	// TODO: Add a list of directives?

	interfaceNames []string
}

// Interface types represent a list of named fields and their arguments.
//
// GraphQL objects can then implement these interfaces which requires that the object type will
// define all fields defined by those interfaces.
//
// http://facebook.github.io/graphql/draft/#sec-Interfaces
type Interface struct {
	Name          string
	PossibleTypes []*Object
	Fields        FieldList // NOTE: the spec refers to this as `FieldsDefinition`.
	Desc          string
	// TODO: Add a list of directives?
}

// Union types represent objects that could be one of a list of GraphQL object types, but provides no
// guaranteed fields between those types.
//
// They also differ from interfaces in that object types declare what interfaces they implement, but
// are not aware of what unions contain them.
//
// http://facebook.github.io/graphql/draft/#sec-Unions
type Union struct {
	Name          string
	PossibleTypes []*Object // NOTE: the spec refers to this as `UnionMemberTypes`.
	Desc          string
	// TODO: Add a list of directives?

	typeNames []string
}

// Enum types describe a set of possible values.
//
// Like scalar types, Enum types also represent leaf values in a GraphQL type system.
//
// http://facebook.github.io/graphql/draft/#sec-Enums
type Enum struct {
	Name   string
	Values []*EnumValue // NOTE: the spec refers to this as `EnumValuesDefinition`.
	Desc   string
	// TODO: Add a list of directives?
}

// EnumValue types are unique values that may be serialized as a string: the name of the
// represented value.
//
// http://facebook.github.io/graphql/draft/#EnumValueDefinition
type EnumValue struct {
	Name       string
	Directives common.DirectiveList
	Desc       string
	// TODO: Add a list of directives?
}

// InputObject types define a set of input fields; the input fields are either scalars, enums, or
// other input objects.
//
// This allows arguments to accept arbitrarily complex structs.
//
// http://facebook.github.io/graphql/draft/#sec-Input-Objects
type InputObject struct {
	Name   string
	Desc   string
	Values common.InputValueList
	// TODO: Add a list of directives?
}

// FieldsList is a list of an Object's Fields.
//
// http://facebook.github.io/graphql/draft/#FieldsDefinition
type FieldList []*Field

// Get iterates over the field list, returning a pointer-to-Field when the field name matches the
// provided `name` argument.
// Returns nil when no field was found by that name.
func (l FieldList) Get(name string) *Field {
	for _, f := range l {
		if f.Name == name {
			return f
		}
	}
	return nil
}

// Names returns a string slice of the field names in the FieldList.
func (l FieldList) Names() []string {
	names := make([]string, len(l))
	for i, f := range l {
		names[i] = f.Name
	}
	return names
}

// http://facebook.github.io/graphql/draft/#sec-Type-System.Directives
type DirectiveDecl struct {
	Name string
	Desc string
	Locs []string
	Args common.InputValueList
}

func (*Scalar) Kind() string      { return "SCALAR" }
func (*Object) Kind() string      { return "OBJECT" }
func (*Interface) Kind() string   { return "INTERFACE" }
func (*Union) Kind() string       { return "UNION" }
func (*Enum) Kind() string        { return "ENUM" }
func (*InputObject) Kind() string { return "INPUT_OBJECT" }

func (t *Scalar) String() string      { return t.Name }
func (t *Object) String() string      { return t.Name }
func (t *Interface) String() string   { return t.Name }
func (t *Union) String() string       { return t.Name }
func (t *Enum) String() string        { return t.Name }
func (t *InputObject) String() string { return t.Name }

func (t *Scalar) TypeName() string      { return t.Name }
func (t *Object) TypeName() string      { return t.Name }
func (t *Interface) TypeName() string   { return t.Name }
func (t *Union) TypeName() string       { return t.Name }
func (t *Enum) TypeName() string        { return t.Name }
func (t *InputObject) TypeName() string { return t.Name }

func (t *Scalar) Description() string      { return t.Desc }
func (t *Object) Description() string      { return t.Desc }
func (t *Interface) Description() string   { return t.Desc }
func (t *Union) Description() string       { return t.Desc }
func (t *Enum) Description() string        { return t.Desc }
func (t *InputObject) Description() string { return t.Desc }

// Field is a conceptual function which yields values.
// http://facebook.github.io/graphql/draft/#FieldDefinition
type Field struct {
	Name       string
	Args       common.InputValueList // NOTE: the spec refers to this as `ArgumentsDefinition`.
	Type       common.Type
	Directives common.DirectiveList
	Desc       string
}

// New initializes an instance of Schema.
func New() *Schema {
	s := &Schema{
		entryPointNames: make(map[string]string),
		Types:           make(map[string]NamedType),
		Directives:      make(map[string]*DirectiveDecl),
	}
	for n, t := range Meta.Types {
		s.Types[n] = t
	}
	for n, d := range Meta.Directives {
		s.Directives[n] = d
	}
	return s
}

// Parse the schema string.
func (s *Schema) Parse(schemaString string) error {
	l := common.NewLexer(schemaString)

	err := l.CatchSyntaxError(func() { parseSchema(s, l) })
	if err != nil {
		return err
	}

	for _, t := range s.Types {
		if err := resolveNamedType(s, t); err != nil {
			return err
		}
	}
	for _, d := range s.Directives {
		for _, arg := range d.Args {
			t, err := common.ResolveType(arg.Type, s.Resolve)
			if err != nil {
				return err
			}
			arg.Type = t
		}
	}

	s.EntryPoints = make(map[string]NamedType)
	for key, name := range s.entryPointNames {
		t, ok := s.Types[name]
		if !ok {
			if !ok {
				return errors.Errorf("type %q not found", name)
			}
		}
		s.EntryPoints[key] = t
	}

	for _, obj := range s.objects {
		obj.Interfaces = make([]*Interface, len(obj.interfaceNames))
		for i, intfName := range obj.interfaceNames {
			t, ok := s.Types[intfName]
			if !ok {
				return errors.Errorf("interface %q not found", intfName)
			}
			intf, ok := t.(*Interface)
			if !ok {
				return errors.Errorf("type %q is not an interface", intfName)
			}
			obj.Interfaces[i] = intf
			intf.PossibleTypes = append(intf.PossibleTypes, obj)
		}
	}

	for _, union := range s.unions {
		union.PossibleTypes = make([]*Object, len(union.typeNames))
		for i, name := range union.typeNames {
			t, ok := s.Types[name]
			if !ok {
				return errors.Errorf("object type %q not found", name)
			}
			obj, ok := t.(*Object)
			if !ok {
				return errors.Errorf("type %q is not an object", name)
			}
			union.PossibleTypes[i] = obj
		}
	}

	for _, enum := range s.enums {
		for _, value := range enum.Values {
			if err := resolveDirectives(s, value.Directives); err != nil {
				return err
			}
		}
	}

	return nil
}

func resolveNamedType(s *Schema, t NamedType) error {
	switch t := t.(type) {
	case *Object:
		for _, f := range t.Fields {
			if err := resolveField(s, f); err != nil {
				return err
			}
		}
	case *Interface:
		for _, f := range t.Fields {
			if err := resolveField(s, f); err != nil {
				return err
			}
		}
	case *InputObject:
		if err := resolveInputObject(s, t.Values); err != nil {
			return err
		}
	}
	return nil
}

func resolveField(s *Schema, f *Field) error {
	t, err := common.ResolveType(f.Type, s.Resolve)
	if err != nil {
		return err
	}
	f.Type = t
	if err := resolveDirectives(s, f.Directives); err != nil {
		return err
	}
	return resolveInputObject(s, f.Args)
}

func resolveDirectives(s *Schema, directives common.DirectiveList) error {
	for _, d := range directives {
		dirName := d.Name.Name
		dd, ok := s.Directives[dirName]
		if !ok {
			return errors.Errorf("directive %q not found", dirName)
		}
		for _, arg := range d.Args {
			if dd.Args.Get(arg.Name.Name) == nil {
				return errors.Errorf("invalid argument %q for directive %q", arg.Name.Name, dirName)
			}
		}
		for _, arg := range dd.Args {
			if _, ok := d.Args.Get(arg.Name.Name); !ok {
				d.Args = append(d.Args, common.Argument{Name: arg.Name, Value: arg.Default})
			}
		}
	}
	return nil
}

func resolveInputObject(s *Schema, values common.InputValueList) error {
	for _, v := range values {
		t, err := common.ResolveType(v.Type, s.Resolve)
		if err != nil {
			return err
		}
		v.Type = t
	}
	return nil
}

func parseSchema(s *Schema, l *common.Lexer) {
	l.Consume()

	for l.Peek() != scanner.EOF {
		desc := l.DescComment()
		switch x := l.ConsumeIdent(); x {

		case "schema":
			l.ConsumeToken('{')
			for l.Peek() != '}' {
				name := l.ConsumeIdent()
				l.ConsumeToken(':')
				typ := l.ConsumeIdent()
				s.entryPointNames[name] = typ
			}
			l.ConsumeToken('}')

		case "type":
			obj := parseObjectDef(l)
			obj.Desc = desc
			s.Types[obj.Name] = obj
			s.objects = append(s.objects, obj)

		case "interface":
			iface := parseInterfaceDef(l)
			iface.Desc = desc
			s.Types[iface.Name] = iface

		case "union":
			union := parseUnionDef(l)
			union.Desc = desc
			s.Types[union.Name] = union
			s.unions = append(s.unions, union)

		case "enum":
			enum := parseEnumDef(l)
			enum.Desc = desc
			s.Types[enum.Name] = enum
			s.enums = append(s.enums, enum)

		case "input":
			input := parseInputDef(l)
			input.Desc = desc
			s.Types[input.Name] = input

		case "scalar":
			name := l.ConsumeIdent()
			s.Types[name] = &Scalar{Name: name, Desc: desc}

		case "directive":
			directive := parseDirectiveDef(l)
			directive.Desc = desc
			s.Directives[directive.Name] = directive

		default:
			// TODO: Add support for type extensions.
			l.SyntaxError(fmt.Sprintf(`unexpected %q, expecting "schema", "type", "enum", "interface", "union", "input", "scalar" or "directive"`, x))
		}
	}
}

func parseObjectDef(l *common.Lexer) *Object {
	object := &Object{Name: l.ConsumeIdent()}

	if l.Peek() == scanner.Ident {
		l.ConsumeKeyword("implements")

		for l.Peek() != '{' {
			if l.Peek() == '&' {
				l.ConsumeToken('&')
			}

			object.interfaceNames = append(object.interfaceNames, l.ConsumeIdent())
		}
	}

	l.ConsumeToken('{')
	object.Fields = parseFieldsDef(l)
	l.ConsumeToken('}')

	return object
}

func parseInterfaceDef(l *common.Lexer) *Interface {
	i := &Interface{Name: l.ConsumeIdent()}

	l.ConsumeToken('{')
	i.Fields = parseFieldsDef(l)
	l.ConsumeToken('}')

	return i
}

func parseUnionDef(l *common.Lexer) *Union {
	union := &Union{Name: l.ConsumeIdent()}

	l.ConsumeToken('=')
	union.typeNames = []string{l.ConsumeIdent()}
	for l.Peek() == '|' {
		l.ConsumeToken('|')
		union.typeNames = append(union.typeNames, l.ConsumeIdent())
	}

	return union
}

func parseInputDef(l *common.Lexer) *InputObject {
	i := &InputObject{}
	i.Name = l.ConsumeIdent()
	l.ConsumeToken('{')
	for l.Peek() != '}' {
		i.Values = append(i.Values, common.ParseInputValue(l))
	}
	l.ConsumeToken('}')
	return i
}

func parseEnumDef(l *common.Lexer) *Enum {
	enum := &Enum{Name: l.ConsumeIdent()}

	l.ConsumeToken('{')
	for l.Peek() != '}' {
		v := &EnumValue{
			Desc:       l.DescComment(),
			Name:       l.ConsumeIdent(),
			Directives: common.ParseDirectives(l),
		}

		enum.Values = append(enum.Values, v)
	}
	l.ConsumeToken('}')
	return enum
}

func parseDirectiveDef(l *common.Lexer) *DirectiveDecl {
	l.ConsumeToken('@')
	d := &DirectiveDecl{Name: l.ConsumeIdent()}

	if l.Peek() == '(' {
		l.ConsumeToken('(')
		for l.Peek() != ')' {
			v := common.ParseInputValue(l)
			d.Args = append(d.Args, v)
		}
		l.ConsumeToken(')')
	}

	l.ConsumeKeyword("on")

	for {
		loc := l.ConsumeIdent()
		d.Locs = append(d.Locs, loc)
		if l.Peek() != '|' {
			break
		}
		l.ConsumeToken('|')
	}
	return d
}

func parseFieldsDef(l *common.Lexer) FieldList {
	var fields FieldList
	for l.Peek() != '}' {
		f := &Field{}
		f.Desc = l.DescComment()
		f.Name = l.ConsumeIdent()
		if l.Peek() == '(' {
			l.ConsumeToken('(')
			for l.Peek() != ')' {
				f.Args = append(f.Args, common.ParseInputValue(l))
			}
			l.ConsumeToken(')')
		}
		l.ConsumeToken(':')
		f.Type = common.ParseType(l)
		f.Directives = common.ParseDirectives(l)
		fields = append(fields, f)
	}
	return fields
}
