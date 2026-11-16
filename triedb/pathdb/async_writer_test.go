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

package pathdb

import (
	"errors"
	"testing"
)

// TestAsyncWriterOrder ensures the scheduled tasks run in order and are all
// done once the wait returns.
func TestAsyncWriterOrder(t *testing.T) {
	w := newAsyncWriter()
	defer w.close()

	var done []int
	for i := 0; i < 1000; i++ {
		if err := w.submit(func() error {
			done = append(done, i)
			return nil
		}); err != nil {
			t.Fatalf("Failed to submit task: %v", err)
		}
	}
	if err := w.wait(); err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if len(done) != 1000 {
		t.Fatalf("Unexpected task count, want: 1000, got: %d", len(done))
	}
	for i, n := range done {
		if n != i {
			t.Fatalf("Unexpected task order at %d: %d", i, n)
		}
	}
}

// TestAsyncWriterError ensures the error of a failed task is reported by the
// wait and rejects the following tasks.
func TestAsyncWriterError(t *testing.T) {
	w := newAsyncWriter()
	defer w.close()

	failure := errors.New("disk full")
	if err := w.submit(func() error { return failure }); err != nil {
		t.Fatalf("Failed to submit task: %v", err)
	}
	if err := w.wait(); !errors.Is(err, failure) {
		t.Fatalf("Unexpected error, want: %v, got: %v", failure, err)
	}
	if err := w.submit(func() error { return nil }); !errors.Is(err, failure) {
		t.Fatalf("Unexpected submit error, want: %v, got: %v", failure, err)
	}
}

// TestAsyncWriterClose ensures the close waits for the scheduled tasks, can
// be repeated, and rejects the following tasks.
func TestAsyncWriterClose(t *testing.T) {
	w := newAsyncWriter()

	var ran bool
	if err := w.submit(func() error {
		ran = true
		return nil
	}); err != nil {
		t.Fatalf("Failed to submit task: %v", err)
	}
	if err := w.close(); err != nil {
		t.Fatalf("Failed to close writer: %v", err)
	}
	if !ran {
		t.Fatal("Scheduled task is not done before close")
	}
	if err := w.close(); err != nil {
		t.Fatalf("Failed to close writer again: %v", err)
	}
	if err := w.submit(func() error { return nil }); !errors.Is(err, errWriterClosed) {
		t.Fatalf("Unexpected submit error, want: %v, got: %v", errWriterClosed, err)
	}
}
