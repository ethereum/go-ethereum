package metrics

import (
	"testing"
	"time"
)

func TestResettingTimer(t *testing.T) {
	tests := []struct {
		values   []int64
		start    int
		end      int
		wantP50  int64
		wantP95  int64
		wantP99  int64
		wantMean float64
		wantMin  int64
		wantMax  int64
	}{
		{
			values:  []int64{},
			start:   1,
			end:     11,
			wantP50: 5, wantP95: 10, wantP99: 10,
			wantMin: 1, wantMax: 10, wantMean: 5.5,
		},
		{
			values:  []int64{},
			start:   1,
			end:     101,
			wantP50: 50, wantP95: 95, wantP99: 99,
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
			wantP50: 1, wantP95: 10, wantP99: 10,
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

		ps := snap.Percentiles([]float64{50, 95, 99})

		val := snap.Values()

		if len(val) > 0 {
			if tt.wantMin != val[0] {
				t.Fatalf("%d: min: got %d, want %d", ind, val[0], tt.wantMin)
			}

			if tt.wantMax != val[len(val)-1] {
				t.Fatalf("%d: max: got %d, want %d", ind, val[len(val)-1], tt.wantMax)
			}
		}

		if tt.wantMean != snap.Mean() {
			t.Fatalf("%d: mean: got %.2f, want %.2f", ind, snap.Mean(), tt.wantMean)
		}

		if tt.wantP50 != ps[0] {
			t.Fatalf("%d: p50: got %d, want %d", ind, ps[0], tt.wantP50)
		}

		if tt.wantP95 != ps[1] {
			t.Fatalf("%d: p95: got %d, want %d", ind, ps[1], tt.wantP95)
		}

		if tt.wantP99 != ps[2] {
			t.Fatalf("%d: p99: got %d, want %d", ind, ps[2], tt.wantP99)
		}
	}
}

func TestResettingTimerWithFivePercentiles(t *testing.T) {
	tests := []struct {
		values   []int64
		start    int
		end      int
		wantP05  int64
		wantP20  int64
		wantP50  int64
		wantP95  int64
		wantP99  int64
		wantMean float64
		wantMin  int64
		wantMax  int64
	}{
		{
			values:  []int64{},
			start:   1,
			end:     11,
			wantP05: 1, wantP20: 2, wantP50: 5, wantP95: 10, wantP99: 10,
			wantMin: 1, wantMax: 10, wantMean: 5.5,
		},
		{
			values:  []int64{},
			start:   1,
			end:     101,
			wantP05: 5, wantP20: 20, wantP50: 50, wantP95: 95, wantP99: 99,
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
			wantP05: 1, wantP20: 1, wantP50: 1, wantP95: 10, wantP99: 10,
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

		ps := snap.Percentiles([]float64{5, 20, 50, 95, 99})

		val := snap.Values()

		if len(val) > 0 {
			if tt.wantMin != val[0] {
				t.Fatalf("%d: min: got %d, want %d", ind, val[0], tt.wantMin)
			}

			if tt.wantMax != val[len(val)-1] {
				t.Fatalf("%d: max: got %d, want %d", ind, val[len(val)-1], tt.wantMax)
			}
		}

		if tt.wantMean != snap.Mean() {
			t.Fatalf("%d: mean: got %.2f, want %.2f", ind, snap.Mean(), tt.wantMean)
		}

		if tt.wantP05 != ps[0] {
			t.Fatalf("%d: p05: got %d, want %d", ind, ps[0], tt.wantP05)
		}

		if tt.wantP20 != ps[1] {
			t.Fatalf("%d: p20: got %d, want %d", ind, ps[1], tt.wantP20)
		}

		if tt.wantP50 != ps[2] {
			t.Fatalf("%d: p50: got %d, want %d", ind, ps[2], tt.wantP50)
		}

		if tt.wantP95 != ps[3] {
			t.Fatalf("%d: p95: got %d, want %d", ind, ps[3], tt.wantP95)
		}

		if tt.wantP99 != ps[4] {
			t.Fatalf("%d: p99: got %d, want %d", ind, ps[4], tt.wantP99)
		}
	}
}
