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

package abigen

import (
	"strings"
	"testing"
)

func TestNormalizeBytecode(t *testing.T) {
	placeholder := "__$0123456789abcdef0123456789abcdef01$__"
	tests := []struct {
		name  string
		input string
		want  string
		ok    bool
	}{
		{"empty", "", "", true},
		{"hex", "0x6000aBcD", "6000aBcD", true},
		{"outer whitespace", "  0x6000\n", "6000", true},
		{"library placeholder", "60" + placeholder + "00", "60" + placeholder + "00", true},
		{"quote", "6000\"", "", false},
		{"embedded whitespace", "60 00", "", false},
		{"malformed placeholder", "60__$xyz$__00", "", false},
		{"unterminated placeholder", "60__$abcd", "", false},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, err := normalizeBytecode(test.input)
			if test.ok {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				if got != test.want {
					t.Fatalf("got %q, want %q", got, test.want)
				}
			} else if err == nil {
				t.Fatalf("expected invalid bytecode to be rejected, got %q", got)
			}
		})
	}
}

func TestBindRejectsInvalidBytecode(t *testing.T) {
	bad := "6000\";\nvar injected = true\nvar _ = \""
	if _, err := Bind([]string{"Test"}, []string{"[]"}, []string{bad}, nil, "test", nil, nil); err == nil || !strings.Contains(err.Error(), "invalid bytecode") {
		t.Fatalf("Bind accepted invalid bytecode, err %v", err)
	}
	if _, err := BindV2([]string{"Test"}, []string{"[]"}, []string{bad}, "test", nil, nil); err == nil || !strings.Contains(err.Error(), "invalid bytecode") {
		t.Fatalf("BindV2 accepted invalid bytecode, err %v", err)
	}
}
