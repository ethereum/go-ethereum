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

package forks

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestForkDepends(t *testing.T) {
	assert.True(t, Cancun.Requires(London))
	assert.True(t, Cancun.Requires(Frontier))
	assert.False(t, London.Requires(Cancun))
}

func TestForkName(t *testing.T) {
	assert.Equal(t, "Cancun", Cancun.String())
	assert.Equal(t, "cancun", Cancun.ConfigName())

	f, ok := ForkByName("Cancun")
	if !ok {
		t.Fatal("cancun fork not found by name")
	}
	if f != Cancun {
		t.Fatal("wrong fork found by name cancun")
	}

	f, ok = ForkByConfigName("cancun")
	if !ok {
		t.Fatal("cancun fork not found by name")
	}
	if f != Cancun {
		t.Fatal("wrong fork found by name cancun")
	}
}
