// Copyright 2022 The go-ethereum Authors
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

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNameFilter(t *testing.T) {
	t.Parallel()
	_, err := newNameFilter("Foo")
	require.Error(t, err)
	_, err = newNameFilter("too/many:colons:Foo")
	require.Error(t, err)

	f, err := newNameFilter("a/path:A", "*:B", "c/path:*")
	require.NoError(t, err)

	for _, tt := range []struct {
		name  string
		match bool
	}{
		{"a/path:A", true},
		{"unknown/path:A", false},
		{"a/path:X", false},
		{"unknown/path:X", false},
		{"any/path:B", true},
		{"c/path:X", true},
		{"c/path:foo:B", false},
	} {
		match := f.Matches(tt.name)
		if tt.match {
			assert.True(t, match, "expected match")
		} else {
			assert.False(t, match, "expected no match")
		}
	}
}
