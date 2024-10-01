// Copyright 2024 the libevm authors.
//
// The libevm additions to go-ethereum are free software: you can redistribute
// them and/or modify them under the terms of the GNU Lesser General Public License
// as published by the Free Software Foundation, either version 3 of the License,
// or (at your option) any later version.
//
// The libevm additions are distributed in the hope that they will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the GNU Lesser
// General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the go-ethereum library. If not, see
// <http://www.gnu.org/licenses/>.

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
