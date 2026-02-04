// Copyright 2025 The go-ethereum Authors
// This file is part of go-ethereum.
//
// go-ethereum is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// go-ethereum is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU General Public License for more details.
//
// You should have received a copy of the GNU General Public License
// along with go-ethereum. If not, see <http://www.gnu.org/licenses/>.

package main

import "testing"

func TestConvert(t *testing.T) {
	tests := []struct {
		value string
		from  string
		to    string
		want  string
	}{
		// Wei to ether
		{"1000000000000000000", "wei", "ether", "1"},
		{"500000000000000000", "wei", "ether", "0.5"},
		{"1", "wei", "ether", "0.000000000000000001"},

		// Ether to wei
		{"1", "ether", "wei", "1000000000000000000"},
		{"0.5", "ether", "wei", "500000000000000000"},
		{"1.5", "ether", "wei", "1500000000000000000"},

		// Gwei conversions
		{"1000000000", "gwei", "ether", "1"},
		{"1", "ether", "gwei", "1000000000"},
		{"1000000000", "wei", "gwei", "1"},
		{"1", "gwei", "wei", "1000000000"},

		// Identity conversions
		{"42", "wei", "wei", "42"},
		{"1", "ether", "ether", "1"},
		{"100", "gwei", "gwei", "100"},

		// Case insensitivity
		{"1", "ETHER", "WEI", "1000000000000000000"},
		{"1", "Gwei", "Wei", "1000000000"},
	}
	for _, tt := range tests {
		t.Run(tt.value+"_"+tt.from+"_to_"+tt.to, func(t *testing.T) {
			got, err := convert(tt.value, tt.from, tt.to)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tt.want {
				t.Errorf("convert(%q, %q, %q) = %q, want %q", tt.value, tt.from, tt.to, got, tt.want)
			}
		})
	}
}

func TestConvertErrors(t *testing.T) {
	tests := []struct {
		value string
		from  string
		to    string
	}{
		{"abc", "wei", "ether"},
		{"1", "foo", "ether"},
		{"1", "wei", "bar"},
	}
	for _, tt := range tests {
		t.Run(tt.value+"_"+tt.from+"_to_"+tt.to, func(t *testing.T) {
			_, err := convert(tt.value, tt.from, tt.to)
			if err == nil {
				t.Fatal("expected error, got nil")
			}
		})
	}
}
