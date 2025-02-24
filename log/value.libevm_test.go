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
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
	"golang.org/x/exp/slog"
)

func TestTypeOf(t *testing.T) {
	type foo struct{}

	tests := map[any]string{
		nil:         "<nil>",
		int(0):      "int",
		int(1):      "int",
		uint(0):     "uint",
		foo{}:       "log.foo",
		(*foo)(nil): "*log.foo",
	}

	for in, want := range tests {
		got := TypeOf(in).LogValue()
		assert.Equalf(t, want, got.String(), "TypeOf(%T(%[1]v))", in, in)
	}
}

func TestLazy(t *testing.T) {
	const (
		key        = "theKey"
		val        = "theVal"
		wantLogged = key + "=" + val
	)

	var gotNumEvaluations int
	fn := Lazy(func() slog.Value {
		gotNumEvaluations++
		return slog.StringValue(val)
	})

	var out bytes.Buffer
	log := slog.New(slog.NewTextHandler(&out, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))
	log.Info("", key, fn)
	log.Debug("", "not evaluated", fn)

	assert.Containsf(t, out.String(), wantLogged, "evaluation of %T function is logged", fn)
	assert.Equalf(t, 1, gotNumEvaluations, "number of evaluations of %T function", fn)
}
