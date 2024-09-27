package live

import (
	"testing"
)

func testGen(blknum uint64) ([]interface{}, error) {
	traces := make([]interface{}, 5)
	for i := range traces {
		traces[i] = int(blknum*5) + i
	}
	return traces, nil
}

func TestExportLimitedTraces(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		fromBlock uint64
		toBlock   uint64
		count     uint64
		after     uint64
		want      []int
	}{
		{
			name:      "Basic Test",
			fromBlock: 0,
			toBlock:   2,
			count:     10,
			after:     0,
			want: []int{
				0, 1, 2, 3, 4, // the first block
				5, 6, 7, 8, 9, // the second block
			},
		},
		{
			name:      "With After",
			fromBlock: 0,
			toBlock:   0,
			count:     3,
			after:     2,
			want: []int{
				2,    // the first block left
				3, 4, // the second block
			},
		},
		{
			name:      "With After",
			fromBlock: 0,
			toBlock:   0,
			count:     3,
			after:     2,
			want: []int{
				2,    // the first block left
				3, 4, // the second block
			},
		},

		{
			name:      "Not Enough Traces",
			fromBlock: 0,
			toBlock:   0,
			count:     10,
			after:     5,
			want:      nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			have, err := exportLimitedTraces(testGen, tt.fromBlock, tt.toBlock, tt.count, tt.after)
			if err != nil {
				t.Error(err)
				return
			}

			if hl, wl := len(have), len(tt.want); hl != wl {
				t.Errorf("test(%s) exported len not equal, have %d, want %d", tt.name, hl, wl)
				return
			}
			for i := range have {
				if h, w := have[i].(int), tt.want[i]; h != w {
					t.Errorf("test(%s) result not equal have %v want %v at index %d", tt.name, h, w, i)
				}
			}
		})
	}
}
