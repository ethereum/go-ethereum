// Copyright 2026 the libevm authors.
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

package ethtest

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/exp/slog"

	"github.com/ava-labs/libevm/log"
)

type tbRecorder struct {
	testing.TB
	logged, errored []string
}

func (r *tbRecorder) Logf(format string, a ...any) {
	r.logged = append(r.logged, fmt.Sprintf(format, a...))
}

func (r *tbRecorder) Errorf(format string, a ...any) {
	r.errored = append(r.errored, fmt.Sprintf(format, a...))
}

func TestTBLogHandler(t *testing.T) {
	got := &tbRecorder{}
	l := log.NewLogger(NewTBLogHandler(got, slog.LevelDebug))

	l.Debug("Cockroach")            // Logf
	l.Info("Hello", "who", "world") // Logf
	l.Warn("Smoke")                 // Errorf
	l.Error("Fire")                 // Errorf
	// Crit will call os.Exit(1) so we don't test it.

	require.Len(t, got.logged, 2, "Logf() calls")
	require.Len(t, got.errored, 2, "Errorf() calls")

	// Check simplest elements without being brittle about exact formatting
	// See https://testing.googleblog.com/2015/01/testing-on-toilet-change-detector-tests.html.
	assert.Contains(t, got.logged[0], "Cockroach")
	assert.Contains(t, got.logged[1], "Hello")
	assert.Contains(t, got.logged[1], "who")
	assert.Contains(t, got.logged[1], "world")

	assert.Contains(t, got.errored[0], "Smoke")
	assert.Contains(t, got.errored[1], "Fire")
}
