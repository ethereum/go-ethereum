package common

import (
	"context"
	"reflect"
	"sort"
	"testing"
)

func TestWithLabels(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name     string
		initial  []string
		new      []string
		expected []string
	}{
		{
			"nil-nil",
			nil,
			nil,
			nil,
		},

		{
			"nil-something",
			nil,
			[]string{"one", "two"},
			[]string{"one", "two"},
		},

		{
			"something-nil",
			[]string{"one", "two"},
			nil,
			[]string{"one", "two"},
		},

		{
			"something-something",
			[]string{"one", "two"},
			[]string{"three", "four"},
			[]string{"one", "two", "three", "four"},
		},

		// deduplication
		{
			"with duplicates nil-something",
			nil,
			[]string{"one", "two", "one"},
			[]string{"one", "two"},
		},

		{
			"with duplicates something-nil",
			[]string{"one", "two", "one"},
			nil,
			[]string{"one", "two"},
		},

		{
			"with duplicates something-something",
			[]string{"one", "two"},
			[]string{"three", "one"},
			[]string{"one", "two", "three"},
		},

		{
			"with duplicates something-something",
			[]string{"one", "two", "three"},
			[]string{"three", "four", "two"},
			[]string{"one", "two", "three", "four"},
		},
	}

	for _, c := range cases {
		c := c

		t.Run(c.name, func(t *testing.T) {
			t.Parallel()

			ctx := context.Background()

			ctx = WithLabels(ctx, c.initial...)
			ctx = WithLabels(ctx, c.new...)

			got := Labels(ctx)

			if len(got) != len(c.expected) {
				t.Errorf("case %s. expected %v, got %v", c.name, c.expected, got)

				return
			}

			gotSorted := sort.StringSlice(got)
			gotSorted.Sort()

			expectedSorted := sort.StringSlice(c.expected)
			expectedSorted.Sort()

			if !reflect.DeepEqual(gotSorted, expectedSorted) {
				t.Errorf("case %s. expected %v, got %v", c.name, expectedSorted, gotSorted)
			}
		})
	}
}
