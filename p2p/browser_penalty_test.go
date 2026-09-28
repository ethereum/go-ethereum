// Copyright 2026 The go-ethereum Authors
// This file is part of the go-ethereum library.
//
// The go-ethereum library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The go-ethereum library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the go-ethereum library. If not, see <http://www.gnu.org/licenses/>.

package p2p

import (
	"errors"
	"io"
	"testing"
)

func TestPenalizeBrowser(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want bool
	}{
		{"invalid message", DiscProtocolError, true},
		{"idle timeout", DiscReadTimeout, true},
		{"clean single-shot", errProtocolReturned, false},
		{"quitting", DiscQuitting, false},
		{"remote requested", DiscRequested, false},
		{"network error reason", DiscNetworkError, false},
		{"dropped connection", io.EOF, false},
		{"plain error", errors.New("boom"), false},
		{"no error", nil, false},
	}
	for _, tt := range tests {
		if got := penalizeBrowser(tt.err); got != tt.want {
			t.Errorf("penalizeBrowser(%s) = %v, want %v", tt.name, got, tt.want)
		}
	}
}
