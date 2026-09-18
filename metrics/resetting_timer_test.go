package metrics

import (
	"testing"
	"time"
)

func TestResettingTimerCumulativeCountAndSum(t *testing.T) {
	timer := NewResettingTimer()

	// First batch of updates.
	timer.Update(10 * time.Millisecond)
	timer.Update(20 * time.Millisecond)
	timer.Update(30 * time.Millisecond)

	snap1 := timer.Snapshot()
	if have, want := snap1.Count(), 3; have != want {
		t.Fatalf("snap1 count: have %d, want %d", have, want)
	}
	if have, want := snap1.TotalCount(), int64(3); have != want {
		t.Fatalf("snap1 total count: have %d, want %d", have, want)
	}
	wantSum1 := int64(10*time.Millisecond + 20*time.Millisecond + 30*time.Millisecond)
	if have := snap1.TotalSum(); have != wantSum1 {
		t.Fatalf("snap1 total sum: have %d, want %d", have, wantSum1)
	}

	// Second batch of updates (after snapshot reset the values).
	timer.Update(40 * time.Millisecond)
	timer.Update(50 * time.Millisecond)

	snap2 := timer.Snapshot()
	// Per-window count should be 2.
	if have, want := snap2.Count(), 2; have != want {
		t.Fatalf("snap2 count: have %d, want %d", have, want)
	}
	// Cumulative count should be 5 (3 + 2).
	if have, want := snap2.TotalCount(), int64(5); have != want {
		t.Fatalf("snap2 total count: have %d, want %d", have, want)
	}
	// Cumulative sum should include both batches.
	wantSum2 := wantSum1 + int64(40*time.Millisecond+50*time.Millisecond)
	if have := snap2.TotalSum(); have != wantSum2 {
		t.Fatalf("snap2 total sum: have %d, want %d", have, wantSum2)
	}

	// Empty snapshot should still report the same cumulative totals.
	snap3 := timer.Snapshot()
	if have, want := snap3.Count(), 0; have != want {
		t.Fatalf("snap3 count: have %d, want %d", have, want)
	}
	if have, want := snap3.TotalCount(), int64(5); have != want {
		t.Fatalf("snap3 total count: have %d, want %d", have, want)
	}
	if have := snap3.TotalSum(); have != wantSum2 {
		t.Fatalf("snap3 total sum: have %d, want %d", have, wantSum2)
	}
}

func TestResettingSampleCumulativeCountAndSum(t *testing.T) {
	Enable()

	s := ResettingSample(NewUniformSample(100))

	// First batch.
	s.Update(10)
	s.Update(20)
	s.Update(30)

	snap1 := s.Snapshot()
	if have, want := snap1.Count(), int64(3); have != want {
		t.Fatalf("snap1 count: have %d, want %d", have, want)
	}
	if have, want := snap1.Sum(), int64(60); have != want {
		t.Fatalf("snap1 sum: have %d, want %d", have, want)
	}

	// Second batch.
	s.Update(40)
	s.Update(50)

	snap2 := s.Snapshot()
	// Count should be cumulative: 3 + 2 = 5.
	if have, want := snap2.Count(), int64(5); have != want {
		t.Fatalf("snap2 count: have %d, want %d", have, want)
	}
	// Sum should be cumulative: 60 + 90 = 150.
	if have, want := snap2.Sum(), int64(150); have != want {
		t.Fatalf("snap2 sum: have %d, want %d", have, want)
	}

	// Empty snapshot should still report cumulative totals.
	snap3 := s.Snapshot()
	if have, want := snap3.Count(), int64(5); have != want {
		t.Fatalf("snap3 count: have %d, want %d", have, want)
	}
	if have, want := snap3.Sum(), int64(150); have != want {
		t.Fatalf("snap3 sum: have %d, want %d", have, want)
	}
}

func TestResettingTimer(t *testing.T) {
	tests := []struct {
		values   []int64
		start    int
		end      int
		wantP50  float64
		wantP95  float64
		wantP99  float64
		wantMean float64
		wantMin  int64
		wantMax  int64
	}{
		{
			values:  []int64{},
			start:   1,
			end:     11,
			wantP50: 5.5, wantP95: 10, wantP99: 10,
			wantMin: 1, wantMax: 10, wantMean: 5.5,
		},
		{
			values:  []int64{},
			start:   1,
			end:     101,
			wantP50: 50.5, wantP95: 95.94999999999999, wantP99: 99.99,
			wantMin: 1, wantMax: 100, wantMean: 50.5,
		},
		{
			values:  []int64{1},
			start:   0,
			end:     0,
			wantP50: 1, wantP95: 1, wantP99: 1,
			wantMin: 1, wantMax: 1, wantMean: 1,
		},
		{
			values:  []int64{0},
			start:   0,
			end:     0,
			wantP50: 0, wantP95: 0, wantP99: 0,
			wantMin: 0, wantMax: 0, wantMean: 0,
		},
		{
			values:  []int64{},
			start:   0,
			end:     0,
			wantP50: 0, wantP95: 0, wantP99: 0,
			wantMin: 0, wantMax: 0, wantMean: 0,
		},
		{
			values:  []int64{1, 10},
			start:   0,
			end:     0,
			wantP50: 5.5, wantP95: 10, wantP99: 10,
			wantMin: 1, wantMax: 10, wantMean: 5.5,
		},
	}
	for i, tt := range tests {
		timer := NewResettingTimer()

		for i := tt.start; i < tt.end; i++ {
			tt.values = append(tt.values, int64(i))
		}

		for _, v := range tt.values {
			timer.Update(time.Duration(v))
		}
		snap := timer.Snapshot()

		ps := snap.Percentiles([]float64{0.50, 0.95, 0.99})

		if have, want := snap.Min(), tt.wantMin; have != want {
			t.Fatalf("%d: min: have %d, want %d", i, have, want)
		}
		if have, want := snap.Max(), tt.wantMax; have != want {
			t.Fatalf("%d: max: have %d, want %d", i, have, want)
		}
		if have, want := snap.Mean(), tt.wantMean; have != want {
			t.Fatalf("%d: mean: have %v, want %v", i, have, want)
		}
		if have, want := ps[0], tt.wantP50; have != want {
			t.Errorf("%d: p50: have %v, want %v", i, have, want)
		}
		if have, want := ps[1], tt.wantP95; have != want {
			t.Errorf("%d: p95: have %v, want %v", i, have, want)
		}
		if have, want := ps[2], tt.wantP99; have != want {
			t.Errorf("%d: p99: have %v, want %v", i, have, want)
		}
	}
}

func TestResettingTimerWithFivePercentiles(t *testing.T) {
	tests := []struct {
		values   []int64
		start    int
		end      int
		wantP05  float64
		wantP20  float64
		wantP50  float64
		wantP95  float64
		wantP99  float64
		wantMean float64
		wantMin  int64
		wantMax  int64
	}{
		{
			values:  []int64{},
			start:   1,
			end:     11,
			wantP05: 1, wantP20: 2.2, wantP50: 5.5, wantP95: 10, wantP99: 10,
			wantMin: 1, wantMax: 10, wantMean: 5.5,
		},
		{
			values:  []int64{},
			start:   1,
			end:     101,
			wantP05: 5.050000000000001, wantP20: 20.200000000000003, wantP50: 50.5, wantP95: 95.94999999999999, wantP99: 99.99,
			wantMin: 1, wantMax: 100, wantMean: 50.5,
		},
		{
			values:  []int64{1},
			start:   0,
			end:     0,
			wantP05: 1, wantP20: 1, wantP50: 1, wantP95: 1, wantP99: 1,
			wantMin: 1, wantMax: 1, wantMean: 1,
		},
		{
			values:  []int64{0},
			start:   0,
			end:     0,
			wantP05: 0, wantP20: 0, wantP50: 0, wantP95: 0, wantP99: 0,
			wantMin: 0, wantMax: 0, wantMean: 0,
		},
		{
			values:  []int64{},
			start:   0,
			end:     0,
			wantP05: 0, wantP20: 0, wantP50: 0, wantP95: 0, wantP99: 0,
			wantMin: 0, wantMax: 0, wantMean: 0,
		},
		{
			values:  []int64{1, 10},
			start:   0,
			end:     0,
			wantP05: 1, wantP20: 1, wantP50: 5.5, wantP95: 10, wantP99: 10,
			wantMin: 1, wantMax: 10, wantMean: 5.5,
		},
	}
	for ind, tt := range tests {
		timer := NewResettingTimer()

		for i := tt.start; i < tt.end; i++ {
			tt.values = append(tt.values, int64(i))
		}

		for _, v := range tt.values {
			timer.Update(time.Duration(v))
		}

		snap := timer.Snapshot()

		ps := snap.Percentiles([]float64{0.05, 0.20, 0.50, 0.95, 0.99})

		if tt.wantMin != snap.Min() {
			t.Errorf("%d: min: got %d, want %d", ind, snap.Min(), tt.wantMin)
		}

		if tt.wantMax != snap.Max() {
			t.Errorf("%d: max: got %d, want %d", ind, snap.Max(), tt.wantMax)
		}

		if tt.wantMean != snap.Mean() {
			t.Errorf("%d: mean: got %.2f, want %.2f", ind, snap.Mean(), tt.wantMean)
		}
		if tt.wantP05 != ps[0] {
			t.Errorf("%d: p05: got %v, want %v", ind, ps[0], tt.wantP05)
		}
		if tt.wantP20 != ps[1] {
			t.Errorf("%d: p20: got %v, want %v", ind, ps[1], tt.wantP20)
		}
		if tt.wantP50 != ps[2] {
			t.Errorf("%d: p50: got %v, want %v", ind, ps[2], tt.wantP50)
		}
		if tt.wantP95 != ps[3] {
			t.Errorf("%d: p95: got %v, want %v", ind, ps[3], tt.wantP95)
		}
		if tt.wantP99 != ps[4] {
			t.Errorf("%d: p99: got %v, want %v", ind, ps[4], tt.wantP99)
		}
	}
}
