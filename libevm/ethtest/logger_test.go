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
	tests := []struct {
		name    string
		level   slog.Level
		wantLog []string
		wantErr []string
	}{
		{
			name:    "warn_level",
			level:   slog.LevelWarn,
			wantLog: []string{"Cockroach", "Hello"},
			wantErr: []string{"Smoke", "Fire"},
		},
		{
			name:    "error_level",
			level:   slog.LevelError,
			wantLog: []string{"Cockroach", "Hello", "Smoke"},
			wantErr: []string{"Fire"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := &tbRecorder{}
			l := log.NewLogger(NewTBLogHandler(got, tt.level))

			l.Debug("Cockroach")
			l.Info("Hello", "who", "world")
			l.Warn("Smoke")
			l.Error("Fire")
			// Crit will call os.Exit(1) so we don't test it.

			require.Len(t, got.logged, len(tt.wantLog), "Logf() calls")
			require.Len(t, got.errored, len(tt.wantErr), "Errorf() calls")

			for i, want := range tt.wantLog {
				assert.Contains(t, got.logged[i], want, "Logf()[%d]", i)
			}
			for i, want := range tt.wantErr {
				assert.Contains(t, got.errored[i], want, "Errorf()[%d]", i)
			}
		})
	}
}
