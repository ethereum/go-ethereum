// Copyright 2024 The go-ethereum Authors
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

package native

import "testing"

func TestConvertErrorToParity(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string
	}{
		{
			name: "exact map key",
			in:   "max code size exceeded",
			want: "Out of gas",
		},
		{
			name: "wrapped map key",
			in:   "max code size exceeded: code size 32769 limit 32768",
			want: "Out of gas",
		},
		{
			name: "existing prefix rule",
			in:   "out of gas: not enough gas for reentrancy sentry",
			want: "Out of gas",
		},
		{
			name: "unknown error unchanged",
			in:   "some unknown error",
			want: "some unknown error",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			frame := &flatCallFrame{Error: tc.in}
			convertErrorToParity(frame)
			if frame.Error != tc.want {
				t.Fatalf("unexpected mapped error, got=%q want=%q", frame.Error, tc.want)
			}
		})
	}
}
