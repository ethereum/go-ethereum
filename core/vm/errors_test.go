// Copyright 2014 The go-ethereum Authors
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

package vm

import (
	"testing"
)

func TestGasError(t *testing.T) {
	err := NewGasError("test operation", 1000, 500)
	expected := "gas error in test operation: required 1000, available 500"
	if err.Error() != expected {
		t.Errorf("GasError.Error() = %v, want %v", err.Error(), expected)
	}
}

func TestStackError(t *testing.T) {
	err := NewStackError("test operation", 3, 1, []uint64{1, 2, 3})
	expected := "stack error in test operation: required 3 items, available 1"
	if err.Error() != expected {
		t.Errorf("StackError.Error() = %v, want %v", err.Error(), expected)
	}
}

func TestMemoryError(t *testing.T) {
	err := NewMemoryError("test operation", 100, 50, 0)
	expected := "memory error in test operation: requested 100 bytes at offset 0, available 50"
	if err.Error() != expected {
		t.Errorf("MemoryError.Error() = %v, want %v", err.Error(), expected)
	}
}

func TestErrorTypes(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected string
	}{
		{
			name:     "GasError",
			err:      NewGasError("ADD", 1000, 500),
			expected: "gas error in ADD: required 1000, available 500",
		},
		{
			name:     "StackError",
			err:      NewStackError("MUL", 2, 1, nil),
			expected: "stack error in MUL: required 2 items, available 1",
		},
		{
			name:     "MemoryError",
			err:      NewMemoryError("MLOAD", 32, 16, 0),
			expected: "memory error in MLOAD: requested 32 bytes at offset 0, available 16",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.err.Error() != tt.expected {
				t.Errorf("Error() = %v, want %v", tt.err.Error(), tt.expected)
			}
		})
	}
} 