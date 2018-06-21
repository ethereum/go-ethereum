// Copyright 2018 The go-ethereum Authors
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
package priorityqueue

import (
	"context"
	"sync"
	"testing"
)

func TestPriorityQueue(t *testing.T) {
	var results []string
	wg := sync.WaitGroup{}
	pq := New(3, 2)
	wg.Add(1)
	go pq.Run(context.Background(), func(v interface{}) {
		results = append(results, v.(string))
		wg.Done()
	})
	pq.Push(context.Background(), "2.0", 2)
	wg.Wait()
	if results[0] != "2.0" {
		t.Errorf("expected first result %q, got %q", "2.0", results[0])
	}

Loop:
	for i, tc := range []struct {
		priorities []int
		values     []string
		results    []string
		errors     []error
	}{
		{
			priorities: []int{0},
			values:     []string{""},
			results:    []string{""},
		},
		{
			priorities: []int{0, 1},
			values:     []string{"0.0", "1.0"},
			results:    []string{"1.0", "0.0"},
		},
		{
			priorities: []int{1, 0},
			values:     []string{"1.0", "0.0"},
			results:    []string{"1.0", "0.0"},
		},
		{
			priorities: []int{0, 1, 1},
			values:     []string{"0.0", "1.0", "1.1"},
			results:    []string{"1.0", "1.1", "0.0"},
		},
		{
			priorities: []int{0, 0, 0},
			values:     []string{"0.0", "0.0", "0.1"},
			errors:     []error{nil, nil, errContention},
		},
	} {
		var results []string
		wg := sync.WaitGroup{}
		pq := New(3, 2)
		wg.Add(len(tc.values))
		for j, value := range tc.values {
			err := pq.Push(nil, value, tc.priorities[j])
			if tc.errors != nil && err != tc.errors[j] {
				t.Errorf("expected push error %v, got %v", tc.errors[j], err)
				continue Loop
			}
			if err != nil {
				continue Loop
			}
		}
		go pq.Run(context.Background(), func(v interface{}) {
			results = append(results, v.(string))
			wg.Done()
		})
		wg.Wait()
		for k, result := range tc.results {
			if results[k] != result {
				t.Errorf("test case %v: expected %v element %q, got %q", i, k, result, results[k])
			}
		}
	}
}
