// Copyright 2014 The go-ethereum Authors
// This file is part of the go-ethereum library.
//
// go-ethereum is free software: you can redistribute it and/or modify
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

package filter

import (
	"testing"
	"time"
)

func TestFilters(t *testing.T) {
	var success bool
	var failure bool

	fm := New()
	fm.Start()
	fm.Install(Generic{
		Str1: "hello",
		Fn: func(data interface{}) {
			success = data.(bool)
		},
	})
	fm.Install(Generic{
		Str1: "hello1",
		Str2: "hello",
		Fn: func(data interface{}) {
			failure = true
		},
	})
	fm.Notify(Generic{Str1: "hello"}, true)
	fm.Stop()

	time.Sleep(10 * time.Millisecond) // yield to the notifier

	if !success {
		t.Error("expected 'hello' to be posted")
	}

	if failure {
		t.Error("hello1 was triggered")
	}
}
