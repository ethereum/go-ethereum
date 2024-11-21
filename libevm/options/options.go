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

// Package options provides a generic mechanism for defining configuration of
// arbitrary types.
package options

// An Option configures values of arbitrary type.
type Option[T any] interface {
	Configure(*T)
}

// As applies Options to a zero-value T, which it then returns.
func As[T any](opts ...Option[T]) *T {
	var t T
	for _, o := range opts {
		o.Configure(&t)
	}
	return &t
}

// A Func converts a function into an [Option], using itself as the Configure
// method.
type Func[T any] func(*T)

var _ Option[struct{}] = Func[struct{}](nil)

// Configure implements the [Option] interface.
func (f Func[T]) Configure(t *T) { f(t) }
