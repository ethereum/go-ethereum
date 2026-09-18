// Copyright 2023 The go-ethereum Authors
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

package prometheus

import (
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/metrics"
	"github.com/ethereum/go-ethereum/metrics/internal"
)

func TestMain(m *testing.M) {
	metrics.Enable()
	os.Exit(m.Run())
}

func TestCollector(t *testing.T) {
	var (
		c    = newCollector()
		want string
	)
	internal.ExampleMetrics().Each(func(name string, i interface{}) {
		c.Add(name, i)
	})
	if wantB, err := os.ReadFile("./testdata/prometheus.want"); err != nil {
		t.Fatal(err)
	} else {
		want = string(wantB)
	}
	if have := c.buff.String(); have != want {
		t.Logf("have\n%v", have)
		t.Logf("have vs want:\n%v", findFirstDiffPos(have, want))
		t.Fatalf("unexpected collector output")
	}
}

func TestResettingTimerCumulativePrometheus(t *testing.T) {
	registry := metrics.NewRegistry()
	timer := metrics.NewRegisteredResettingTimer("test/resetting", registry)

	// First batch of updates.
	timer.Update(10 * time.Millisecond)
	timer.Update(20 * time.Millisecond)

	// First scrape.
	c1 := newCollector()
	registry.Each(func(name string, i interface{}) {
		c1.Add(name, i)
	})
	out1 := c1.buff.String()
	if !strings.Contains(out1, "test_resetting_count 2") {
		t.Fatalf("first scrape should have count 2, got:\n%s", out1)
	}

	// Second batch.
	timer.Update(30 * time.Millisecond)

	// Second scrape - count should be cumulative (3, not 1).
	c2 := newCollector()
	registry.Each(func(name string, i interface{}) {
		c2.Add(name, i)
	})
	out2 := c2.buff.String()
	if !strings.Contains(out2, "test_resetting_count 3") {
		t.Fatalf("second scrape should have cumulative count 3, got:\n%s", out2)
	}

	// Third scrape with no new updates - count should stay at 3.
	c3 := newCollector()
	registry.Each(func(name string, i interface{}) {
		c3.Add(name, i)
	})
	out3 := c3.buff.String()
	// With no new events and totalCount > 0, we still need to report.
	if !strings.Contains(out3, "test_resetting_count 3") {
		t.Fatalf("third scrape should still report cumulative count 3, got:\n%s", out3)
	}
}

func findFirstDiffPos(a, b string) string {
	yy := strings.Split(b, "\n")
	for i, x := range strings.Split(a, "\n") {
		if i >= len(yy) {
			return fmt.Sprintf("have:%d: %s\nwant:%d: <EOF>", i, x, i)
		}
		if y := yy[i]; x != y {
			return fmt.Sprintf("have:%d: %s\nwant:%d: %s", i, x, i, y)
		}
	}
	return ""
}
