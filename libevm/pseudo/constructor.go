package pseudo

// A Constructor returns newly constructed [Type] instances for a pre-registered
// concrete type.
type Constructor interface {
	Zero() *Type
	NewPointer() *Type
	NilPointer() *Type
}

// NewConstructor returns a [Constructor] that builds `T` [Type] instances.
func NewConstructor[T any]() Constructor {
	return ctor[T]{}
}

type ctor[T any] struct{}

func (ctor[T]) Zero() *Type       { return Zero[T]().Type }
func (ctor[T]) NilPointer() *Type { return Zero[*T]().Type }

func (ctor[T]) NewPointer() *Type {
	var x T
	return From(&x).Type
}
