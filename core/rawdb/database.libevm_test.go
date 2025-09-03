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

package rawdb

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSkipFreezers(t *testing.T) {
	db := NewMemoryDatabase()

	tests := []struct {
		skipFreezers bool
		wantErr      error
	}{
		{
			skipFreezers: false,
			wantErr:      errNotSupported,
		},
		{
			skipFreezers: true,
			wantErr:      nil,
		},
	}

	for _, tt := range tests {
		var opts []InspectDatabaseOption
		if tt.skipFreezers {
			opts = append(opts, WithSkipFreezers())
		}
		assert.ErrorIsf(t, InspectDatabase(db, nil, nil, opts...), tt.wantErr, "InspectDatabase(%T, nil, nil, [WithSkipFreezers = %t])", db, tt.skipFreezers)
	}
}
