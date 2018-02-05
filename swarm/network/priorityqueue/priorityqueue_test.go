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
