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

import (
	"reflect"

	"github.com/ethereum/go-ethereum/rlp"
)

// Reflection is used as a last resort in pseudo types so is limited to this
// file to avoid being seen as the norm. If you are adding to this file, please
// try to achieve the same results with type parameters.

func (c *concrete[T]) isZero() bool {
	// The alternative would require that T be comparable, which would bubble up
	// and invade the rest of the code base.
	return reflect.ValueOf(c.val).IsZero()
}

func (c *concrete[T]) equal(t *Type) bool {
	d, ok := t.val.(*concrete[T])
	if !ok {
		return false
	}
	switch v := any(c.val).(type) {
	case EqualityChecker[T]:
		return v.Equal(d.val)
	default:
		// See rationale for reflection in [concrete.isZero].
		return reflect.DeepEqual(c.val, d.val)
	}
}

func (c *concrete[T]) DecodeRLP(s *rlp.Stream) error {
	switch v := reflect.ValueOf(c.val); v.Kind() {
	case reflect.Pointer:
		if v.IsNil() {
			el := v.Type().Elem()
			c.val = reflect.New(el).Interface().(T) //nolint:forcetypeassert // Invariant scoped to the last few lines of code so simple to verify
		}
		return s.Decode(c.val)
	default:
		return s.Decode(&c.val)
	}
}
