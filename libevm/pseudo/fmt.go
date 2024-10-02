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
	"fmt"
)

var _ = []fmt.Formatter{
	(*Type)(nil),
	(*Value[struct{}])(nil),
	(*concrete[struct{}])(nil),
}

// Format implements the [fmt.Formatter] interface.
func (t *Type) Format(s fmt.State, verb rune) {
	switch {
	case t == nil, t.val == nil:
		writeToFmtState(s, "<nil>[pseudo.Type[unknown]]")
	default:
		t.val.Format(s, verb)
	}
}

// Format implements the [fmt.Formatter] interface.
func (v *Value[T]) Format(s fmt.State, verb rune) { v.t.Format(s, verb) }

func (c *concrete[T]) Format(s fmt.State, verb rune) {
	switch {
	case c == nil:
		writeToFmtState(s, "<nil>[pseudo.Type[%T]]", concrete[T]{}.val)
	default:
		// Respects the original formatting directive. fmt all the way down!
		format := fmt.Sprintf("pseudo.Type[%%T]{%s}", fmt.FormatString(s, verb))
		writeToFmtState(s, format, c.val, c.val)
	}
}

func writeToFmtState(s fmt.State, format string, a ...any) {
	// There is no way to bubble errors out from a `fmt.Formatter`.
	_, _ = s.Write([]byte(fmt.Sprintf(format, a...)))
}
