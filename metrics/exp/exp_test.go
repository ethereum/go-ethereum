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

package exp

import (
	"encoding/json"
	"io"
	"net/http/httptest"
	"runtime/metrics"
	"testing"

	gethmetrics "github.com/ethereum/go-ethereum/metrics"
)

// Tests that /debug/metrics stays valid JSON when a runtime histogram has bucket
// edges but no samples yet. This is the shape the Go runtime reports for
// /gc/pauses:seconds before the first GC pause, and permanently when GC is
// disabled with --gogc=-1.
func TestExpHandlerEmptyRuntimeHistogram(t *testing.T) {
	r := gethmetrics.NewRegistry()
	hist := gethmetrics.RuntimeHistogramFromData(1e9, &metrics.Float64Histogram{
		Counts:  []uint64{0, 0, 0},
		Buckets: []float64{0, 1, 2, 3},
	})
	if err := r.Register("system/memory/pauses", hist); err != nil {
		t.Fatal(err)
	}

	rec := httptest.NewRecorder()
	ExpHandler(r).ServeHTTP(rec, httptest.NewRequest("GET", "/debug/metrics", nil))

	body, err := io.ReadAll(rec.Result().Body)
	if err != nil {
		t.Fatal(err)
	}
	var parsed map[string]any
	if err := json.Unmarshal(body, &parsed); err != nil {
		t.Fatalf("invalid JSON from /debug/metrics: %v\n%s", err, body)
	}
	if have, want := parsed["system/memory/pauses.mean"], float64(0); have != want {
		t.Fatalf("wrong mean: have %v, want %v", have, want)
	}
}
