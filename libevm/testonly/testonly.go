// Copyright 2024 the libevm authors.
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

// Package testonly enforces functionality that MUST be limited to tests.
package testonly

import (
	"runtime"
	"strings"
)

// OrPanic runs `fn` i.f.f. called from within a testing environment.
func OrPanic(fn func()) {
	pc := make([]uintptr, 64)
	runtime.Callers(0, pc)
	frames := runtime.CallersFrames(pc)
	for {
		f, more := frames.Next()
		if strings.Contains(f.File, "/testing/") || strings.HasSuffix(f.File, "_test.go") {
			fn()
			return
		}
		if !more {
			panic("no _test.go file in call stack")
		}
	}
}
