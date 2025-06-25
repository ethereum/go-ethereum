package metrics

import (
	"slices"
	"testing"
)

func TestMetricsSorting(t *testing.T) {
	var namedMetrics = []namedMetric{
		{name: "zzz"},
		{name: "bbb"},
		{name: "fff"},
		{name: "ggg"},
	}

	slices.SortFunc(namedMetrics, namedMetric.cmp)
	for i, name := range []string{"bbb", "fff", "ggg", "zzz"} {
		if namedMetrics[i].name != name {
			t.Fail()
		}
	}
}
