// Copyright 2021-2022 The go-xpayments Authors
// This file is part of go-xpayments.

package metrics

import (
	"sort"
	"testing"
)

func TestMetricsSorting(t *testing.T) {
	var namedMetrics = namedMetricSlice{
		{name: "zzz"},
		{name: "bbb"},
		{name: "fff"},
		{name: "ggg"},
	}

	sort.Sort(namedMetrics)
	for i, name := range []string{"bbb", "fff", "ggg", "zzz"} {
		if namedMetrics[i].name != name {
			t.Fail()
		}
	}
}
