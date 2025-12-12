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
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestFormat(t *testing.T) {
	tests := []struct {
		name         string
		from         any
		format       string
		wantContains []string
	}{
		{
			name:         "width",
			from:         42,
			format:       "%04d",
			wantContains: []string{"int", "0042"},
		},
		{
			name:         "precision",
			from:         float64(2),
			format:       "%.5f",
			wantContains: []string{"float64", "2.00000"},
		},
		{
			name:         "flag",
			from:         42,
			format:       "%+d",
			wantContains: []string{"int", "+42"},
		},
		{
			name:         "verb",
			from:         42,
			format:       "%x",
			wantContains: []string{"int", "2a"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := fmt.Sprintf(tt.format, fromAny(t, tt.from))
			for _, want := range tt.wantContains {
				assert.Containsf(t, got, want, "fmt.Sprintf(%q, From(%T[%[2]v]))", tt.format, tt.from)
			}
		})
	}
}

func fromAny(t *testing.T, x any) *Type {
	t.Helper()

	// Without this, the function will be From[any]().
	switch x := x.(type) {
	case int:
		return From(x).Type
	case float64:
		return From(x).Type
	default:
		t.Fatalf("Bad test setup: add type case for %T", x)
		return nil
	}
}
