package resolvable

import (
	"context"
	"fmt"
	"reflect"
	"strings"

	"github.com/graph-gophers/graphql-go/internal/common"
	"github.com/graph-gophers/graphql-go/internal/exec/packer"
	"github.com/graph-gophers/graphql-go/internal/schema"
)

type Schema struct {
	schema.Schema
	Query    Resolvable
	Mutation Resolvable
	Resolver reflect.Value
}

type Resolvable interface {
	isResolvable()
}

type Object struct {
	Name           string
	Fields         map[string]*Field
	TypeAssertions map[string]*TypeAssertion
}

type Field struct {
	schema.Field
	TypeName    string
	MethodIndex int
	HasContext  bool
	HasError    bool
	ArgsPacker  *packer.StructPacker
	ValueExec   Resolvable
	TraceLabel  string
}

type TypeAssertion struct {
	MethodIndex int
	TypeExec    Resolvable
}

type List struct {
	Elem Resolvable
}

type Scalar struct{}

func (*Object) isResolvable() {}
func (*List) isResolvable()   {}
func (*Scalar) isResolvable() {}

func ApplyResolver(s *schema.Schema, resolver interface{}) (*Schema, error) {
	b := newBuilder(s)

	var query, mutation Resolvable

	if t, ok := s.EntryPoints["query"]; ok {
		if err := b.assignExec(&query, t, reflect.TypeOf(resolver)); err != nil {
			return nil, err
		}
	}

	if t, ok := s.EntryPoints["mutation"]; ok {
		if err := b.assignExec(&mutation, t, reflect.TypeOf(resolver)); err != nil {
			return nil, err
		}
	}

	if err := b.finish(); err != nil {
		return nil, err
	}

	return &Schema{
		Schema:   *s,
		Resolver: reflect.ValueOf(resolver),
		Query:    query,
		Mutation: mutation,
	}, nil
}

type execBuilder struct {
	schema        *schema.Schema
	resMap        map[typePair]*resMapEntry
	packerBuilder *packer.Builder
}

type typePair struct {
	graphQLType  common.Type
	resolverType reflect.Type
}

type resMapEntry struct {
	exec    Resolvable
	targets []*Resolvable
}

func newBuilder(s *schema.Schema) *execBuilder {
	return &execBuilder{
		schema:        s,
		resMap:        make(map[typePair]*resMapEntry),
		packerBuilder: packer.NewBuilder(),
	}
}

func (b *execBuilder) finish() error {
	for _, entry := range b.resMap {
		for _, target := range entry.targets {
			*target = entry.exec
		}
	}

	return b.packerBuilder.Finish()
}

func (b *execBuilder) assignExec(target *Resolvable, t common.Type, resolverType reflect.Type) error {
	k := typePair{t, resolverType}
	ref, ok := b.resMap[k]
	if !ok {
		ref = &resMapEntry{}
		b.resMap[k] = ref
		var err error
		ref.exec, err = b.makeExec(t, resolverType)
		if err != nil {
			return err
		}
	}
	ref.targets = append(ref.targets, target)
	return nil
}

func (b *execBuilder) makeExec(t common.Type, resolverType reflect.Type) (Resolvable, error) {
	var nonNull bool
	t, nonNull = unwrapNonNull(t)

	switch t := t.(type) {
	case *schema.Object:
		return b.makeObjectExec(t.Name, t.Fields, nil, nonNull, resolverType)

	case *schema.Interface:
		return b.makeObjectExec(t.Name, t.Fields, t.PossibleTypes, nonNull, resolverType)

	case *schema.Union:
		return b.makeObjectExec(t.Name, nil, t.PossibleTypes, nonNull, resolverType)
	}

	if !nonNull {
		if resolverType.Kind() != reflect.Ptr {
			return nil, fmt.Errorf("%s is not a pointer", resolverType)
		}
		resolverType = resolverType.Elem()
	}

	switch t := t.(type) {
	case *schema.Scalar:
		return makeScalarExec(t, resolverType)

	case *schema.Enum:
		return &Scalar{}, nil

	case *common.List:
		if resolverType.Kind() != reflect.Slice {
			return nil, fmt.Errorf("%s is not a slice", resolverType)
		}
		e := &List{}
		if err := b.assignExec(&e.Elem, t.OfType, resolverType.Elem()); err != nil {
			return nil, err
		}
		return e, nil

	default:
		panic("invalid type: " + t.String())
	}
}

func makeScalarExec(t *schema.Scalar, resolverType reflect.Type) (Resolvable, error) {
	implementsType := false
	switch r := reflect.New(resolverType).Interface().(type) {
	case *int32:
		implementsType = (t.Name == "Int")
	case *float64:
		implementsType = (t.Name == "Float")
	case *string:
		implementsType = (t.Name == "String")
	case *bool:
		implementsType = (t.Name == "Boolean")
	case packer.Unmarshaler:
		implementsType = r.ImplementsGraphQLType(t.Name)
	}
	if !implementsType {
		return nil, fmt.Errorf("can not use %s as %s", resolverType, t.Name)
	}
	return &Scalar{}, nil
}

func (b *execBuilder) makeObjectExec(typeName string, fields schema.FieldList, possibleTypes []*schema.Object, nonNull bool, resolverType reflect.Type) (*Object, error) {
	if !nonNull {
		if resolverType.Kind() != reflect.Ptr && resolverType.Kind() != reflect.Interface {
			return nil, fmt.Errorf("%s is not a pointer or interface", resolverType)
		}
	}

	methodHasReceiver := resolverType.Kind() != reflect.Interface

	Fields := make(map[string]*Field)
	for _, f := range fields {
		methodIndex := findMethod(resolverType, f.Name)
		if methodIndex == -1 {
			hint := ""
			if findMethod(reflect.PtrTo(resolverType), f.Name) != -1 {
				hint = " (hint: the method exists on the pointer type)"
			}
			return nil, fmt.Errorf("%s does not resolve %q: missing method for field %q%s", resolverType, typeName, f.Name, hint)
		}

		m := resolverType.Method(methodIndex)
		fe, err := b.makeFieldExec(typeName, f, m, methodIndex, methodHasReceiver)
		if err != nil {
			return nil, fmt.Errorf("%s\n\treturned by (%s).%s", err, resolverType, m.Name)
		}
		Fields[f.Name] = fe
	}

	typeAssertions := make(map[string]*TypeAssertion)
	for _, impl := range possibleTypes {
		methodIndex := findMethod(resolverType, "To"+impl.Name)
		if methodIndex == -1 {
			return nil, fmt.Errorf("%s does not resolve %q: missing method %q to convert to %q", resolverType, typeName, "To"+impl.Name, impl.Name)
		}
		if resolverType.Method(methodIndex).Type.NumOut() != 2 {
			return nil, fmt.Errorf("%s does not resolve %q: method %q should return a value and a bool indicating success", resolverType, typeName, "To"+impl.Name)
		}
		a := &TypeAssertion{
			MethodIndex: methodIndex,
		}
		if err := b.assignExec(&a.TypeExec, impl, resolverType.Method(methodIndex).Type.Out(0)); err != nil {
			return nil, err
		}
		typeAssertions[impl.Name] = a
	}

	return &Object{
		Name:           typeName,
		Fields:         Fields,
		TypeAssertions: typeAssertions,
	}, nil
}

var contextType = reflect.TypeOf((*context.Context)(nil)).Elem()
var errorType = reflect.TypeOf((*error)(nil)).Elem()

func (b *execBuilder) makeFieldExec(typeName string, f *schema.Field, m reflect.Method, methodIndex int, methodHasReceiver bool) (*Field, error) {
	in := make([]reflect.Type, m.Type.NumIn())
	for i := range in {
		in[i] = m.Type.In(i)
	}
	if methodHasReceiver {
		in = in[1:] // first parameter is receiver
	}

	hasContext := len(in) > 0 && in[0] == contextType
	if hasContext {
		in = in[1:]
	}

	var argsPacker *packer.StructPacker
	if len(f.Args) > 0 {
		if len(in) == 0 {
			return nil, fmt.Errorf("must have parameter for field arguments")
		}
		var err error
		argsPacker, err = b.packerBuilder.MakeStructPacker(f.Args, in[0])
		if err != nil {
			return nil, err
		}
		in = in[1:]
	}

	if len(in) > 0 {
		return nil, fmt.Errorf("too many parameters")
	}

	if m.Type.NumOut() > 2 {
		return nil, fmt.Errorf("too many return values")
	}

	hasError := m.Type.NumOut() == 2
	if hasError {
		if m.Type.Out(1) != errorType {
			return nil, fmt.Errorf(`must have "error" as its second return value`)
		}
	}

	fe := &Field{
		Field:       *f,
		TypeName:    typeName,
		MethodIndex: methodIndex,
		HasContext:  hasContext,
		ArgsPacker:  argsPacker,
		HasError:    hasError,
		TraceLabel:  fmt.Sprintf("GraphQL field: %s.%s", typeName, f.Name),
	}
	if err := b.assignExec(&fe.ValueExec, f.Type, m.Type.Out(0)); err != nil {
		return nil, err
	}
	return fe, nil
}

func findMethod(t reflect.Type, name string) int {
	for i := 0; i < t.NumMethod(); i++ {
		if strings.EqualFold(stripUnderscore(name), stripUnderscore(t.Method(i).Name)) {
			return i
		}
	}
	return -1
}

func unwrapNonNull(t common.Type) (common.Type, bool) {
	if nn, ok := t.(*common.NonNull); ok {
		return nn.OfType, true
	}
	return t, false
}

func stripUnderscore(s string) string {
	return strings.Replace(s, "_", "", -1)
}
