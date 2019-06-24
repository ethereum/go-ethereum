package resolvable

import (
	"fmt"
	"reflect"

	"github.com/graph-gophers/graphql-go/internal/common"
	"github.com/graph-gophers/graphql-go/internal/schema"
	"github.com/graph-gophers/graphql-go/introspection"
)

// Meta defines the details of the metadata schema for introspection.
type Meta struct {
	FieldSchema   Field
	FieldType     Field
	FieldTypename Field
	Schema        *Object
	Type          *Object
}

func newMeta(s *schema.Schema) *Meta {
	var err error
	b := newBuilder(s)

	metaSchema := s.Types["__Schema"].(*schema.Object)
	so, err := b.makeObjectExec(metaSchema.Name, metaSchema.Fields, nil, false, reflect.TypeOf(&introspection.Schema{}))
	if err != nil {
		panic(err)
	}

	metaType := s.Types["__Type"].(*schema.Object)
	t, err := b.makeObjectExec(metaType.Name, metaType.Fields, nil, false, reflect.TypeOf(&introspection.Type{}))
	if err != nil {
		panic(err)
	}

	if err := b.finish(); err != nil {
		panic(err)
	}

	fieldTypename := Field{
		Field: schema.Field{
			Name: "__typename",
			Type: &common.NonNull{OfType: s.Types["String"]},
		},
		TraceLabel: fmt.Sprintf("GraphQL field: __typename"),
	}

	fieldSchema := Field{
		Field: schema.Field{
			Name: "__schema",
			Type: s.Types["__Schema"],
		},
		TraceLabel: fmt.Sprintf("GraphQL field: __schema"),
	}

	fieldType := Field{
		Field: schema.Field{
			Name: "__type",
			Type: s.Types["__Type"],
		},
		TraceLabel: fmt.Sprintf("GraphQL field: __type"),
	}

	return &Meta{
		FieldSchema:   fieldSchema,
		FieldTypename: fieldTypename,
		FieldType:     fieldType,
		Schema:        so,
		Type:          t,
	}
}
