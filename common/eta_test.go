// Copyright 2025 The go-ethereum Authors
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

package common

import (
	"testing"
	"time"
)

func TestCalculateETA(t *testing.T) {
	type args struct {
		done    uint64
		left    uint64
		elapsed time.Duration
	}
	tests := []struct {
		name string
		args args
		want time.Duration
	}{
		{
			name: "zero done",
			args: args{done: 0, left: 100, elapsed: time.Second},
			want: 0,
		},
		{
			name: "zero elapsed",
			args: args{done: 1, left: 100, elapsed: 0},
			want: 0,
		},
		{
			name: "@Jolly23 's case",
			args: args{done: 16858580, left: 41802252, elapsed: 66179848 * time.Millisecond},
			want: 164098440 * time.Millisecond,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := CalculateETA(tt.args.done, tt.args.left, tt.args.elapsed)
			// Allow 1ms tolerance for rounding differences
			diff := got - tt.want
			if diff < 0 {
				diff = -diff
			}
			if diff > time.Millisecond {
				t.Errorf("CalculateETA() = %v, want %v", got, tt.want)
			}
		})
	}
}

// TestCalculateETA_SubMillisecond verifies that CalculateETA correctly handles
// operations completing in less than 1ms without returning zero or panicking.
func TestCalculateETA_SubMillisecond(t *testing.T) {
	// Simulate processing 1000 items in 500 microseconds
	done := uint64(1000)
	left := uint64(1000)
	elapsed := 500 * time.Microsecond
	
	eta := CalculateETA(done, left, elapsed)
	
	if eta == 0 {
		t.Fatalf("CalculateETA returned 0 for sub-millisecond elapsed time, expected non-zero ETA")
	}
	
	// At the same rate, processing another 1000 items should take ~500µs
	expectedETA := 500 * time.Microsecond
	tolerance := 100 * time.Microsecond
	
	if eta < expectedETA-tolerance || eta > expectedETA+tolerance {
		t.Errorf("CalculateETA = %v, expected approximately %v (within %v)", eta, expectedETA, tolerance)
	}
}

// TestCalculateETA_EdgeCases verifies correct behavior for boundary conditions
func TestCalculateETA_EdgeCases(t *testing.T) {
	tests := []struct {
		name     string
		done     uint64
		left     uint64
		elapsed  time.Duration
		expected time.Duration
	}{
		{"zero done", 0, 100, time.Second, 0},
		{"zero elapsed", 100, 100, 0, 0},
		{"zero left", 100, 0, time.Second, 0},
		{"instant completion", 1000, 1000, 1 * time.Nanosecond, time.Nanosecond},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := CalculateETA(tt.done, tt.left, tt.elapsed)
			if tt.expected == 0 && result != 0 {
				t.Errorf("expected 0, got %v", result)
			}
			if tt.expected != 0 && result == 0 {
				t.Errorf("expected non-zero, got 0")
			}
		})
	}
}
