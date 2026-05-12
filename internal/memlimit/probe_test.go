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

package memlimit

import "testing"

// TestLimitSmoke asserts that Limit() returns a non-zero value on
// any host the test suite could plausibly run on. This catches
// regressions in the gopsutil fallback path that the OS-specific
// tests (which use a fake reader) would miss.
func TestLimitSmoke(t *testing.T) {
	bytes, src := Limit()
	if bytes == 0 {
		t.Errorf("Limit() returned 0 bytes (source=%s); expected non-zero on any sane host", src)
	}
}
