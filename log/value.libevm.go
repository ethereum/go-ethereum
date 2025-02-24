// Copyright 2025 the libevm authors.
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

package log

import (
	"fmt"

	"golang.org/x/exp/slog"
)

// A Lazy function defers its execution until logging is performed.
type Lazy func() slog.Value

var _ slog.LogValuer = Lazy(nil)

// LogValue implements the [slog.LogValuer] interface.
func (l Lazy) LogValue() slog.Value {
	return l()
}

// TypeOf returns a Lazy function that reports the concrete type of `v` as
// determined with the `%T` [fmt] verb.
func TypeOf(v any) Lazy {
	return Lazy(func() slog.Value {
		return slog.StringValue(fmt.Sprintf("%T", v))
	})
}
