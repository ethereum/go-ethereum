// Copyright 2020 The go-ethereum Authors
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

// Package utesting provides a standalone replacement for package testing.
//
// This package exists because package testing cannot easily be embedded into a
// standalone go program. It provides an API that mirrors the standard library
// testing API.
package utesting

import (
	"bytes"
	"fmt"
	"io"
	"regexp"
	"runtime"
	"sync"
	"time"
)

// Test represents a single test.
type Test struct {
	Name string
	Fn   func(*T)
}

// Result is the result of a test execution.
type Result struct {
	Name     string
	Failed   bool
	Output   string
	Duration time.Duration
}

// MatchTests returns the tests whose name matches a regular expression.
func MatchTests(tests []Test, expr string) []Test {
	var results []Test
	re, err := regexp.Compile(expr)
	if err != nil {
		return nil
	}
	for _, test := range tests {
		if re.MatchString(test.Name) {
			results = append(results, test)
		}
	}
	return results
}

// RunTests executes all given tests in order and returns their results.
// If the report writer is non-nil, a test report is written to it in real time.
func RunTests(tests []Test, report io.Writer) []Result {
	results := make([]Result, len(tests))
	for i, test := range tests {
		start := time.Now()
		results[i].Name = test.Name
		results[i].Failed, results[i].Output = Run(test)
		results[i].Duration = time.Since(start)
		if report != nil {
			printResult(results[i], report)
		}
	}
	return results
}

func printResult(r Result, w io.Writer) {
	pd := r.Duration.Truncate(100 * time.Microsecond)
	if r.Failed {
		fmt.Fprintf(w, "-- FAIL %s (%v)\n", r.Name, pd)
		fmt.Fprintln(w, r.Output)
	} else {
		fmt.Fprintf(w, "-- OK %s (%v)\n", r.Name, pd)
	}
}

// CountFailures returns the number of failed tests in the result slice.
func CountFailures(rr []Result) int {
	count := 0
	for _, r := range rr {
		if r.Failed {
			count++
		}
	}
	return count
}

// Run executes a single test.
func Run(test Test) (bool, string) {
	t := new(T)
	done := make(chan struct{})
	go func() {
		defer close(done)
		defer func() {
			if err := recover(); err != nil {
				buf := make([]byte, 4096)
				i := runtime.Stack(buf, false)
				t.Logf("panic: %v\n\n%s", err, buf[:i])
				t.Fail()
			}
		}()
		test.Fn(t)
	}()
	<-done
	return t.failed, t.output.String()
}

// T is the value given to the test function. The test can signal failures
// and log output by calling methods on this object.
type T struct {
	mu     sync.Mutex
	failed bool
	output bytes.Buffer
}

// FailNow marks the test as having failed and stops its execution by calling
// runtime.Goexit (which then runs all deferred calls in the current goroutine).
func (t *T) FailNow() {
	t.Fail()
	runtime.Goexit()
}

// Fail marks the test as having failed but continues execution.
func (t *T) Fail() {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.failed = true
}

// Failed reports whether the test has failed.
func (t *T) Failed() bool {
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.failed
}

// Log formats its arguments using default formatting, analogous to Println, and records
// the text in the error log.
func (t *T) Log(vs ...interface{}) {
	t.mu.Lock()
	defer t.mu.Unlock()
	fmt.Fprintln(&t.output, vs...)
}

// Logf formats its arguments according to the format, analogous to Printf, and records
// the text in the error log. A final newline is added if not provided.
func (t *T) Logf(format string, vs ...interface{}) {
	t.mu.Lock()
	defer t.mu.Unlock()
	if len(format) == 0 || format[len(format)-1] != '\n' {
		format += "\n"
	}
	fmt.Fprintf(&t.output, format, vs...)
}

// Error is equivalent to Log followed by Fail.
func (t *T) Error(vs ...interface{}) {
	t.Log(vs...)
	t.Fail()
}

// Errorf is equivalent to Logf followed by Fail.
func (t *T) Errorf(format string, vs ...interface{}) {
	t.Logf(format, vs...)
	t.Fail()
}

// Fatal is equivalent to Log followed by FailNow.
func (t *T) Fatal(vs ...interface{}) {
	t.Log(vs...)
	t.FailNow()
}

// Fatalf is equivalent to Logf followed by FailNow.
func (t *T) Fatalf(format string, vs ...interface{}) {
	t.Logf(format, vs...)
	t.FailNow()
}
