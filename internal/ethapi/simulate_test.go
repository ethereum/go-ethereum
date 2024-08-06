package ethapi

import (
	"fmt"
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
)

func TestSimulateSanitizeBlockOrder(t *testing.T) {
	for i, tc := range []struct {
		baseNumber  int
		blocks      []simBlock
		expectedLen int
		err         string
	}{
		{
			baseNumber:  10,
			blocks:      []simBlock{{}, {}, {}},
			expectedLen: 3,
		},
		{
			baseNumber:  10,
			blocks:      []simBlock{{BlockOverrides: &BlockOverrides{Number: newInt(13)}}, {}},
			expectedLen: 4,
		},
		{
			baseNumber:  10,
			blocks:      []simBlock{{BlockOverrides: &BlockOverrides{Number: newInt(11)}}, {BlockOverrides: &BlockOverrides{Number: newInt(14)}}, {}},
			expectedLen: 5,
		},
		{
			baseNumber: 10,
			blocks:     []simBlock{{BlockOverrides: &BlockOverrides{Number: newInt(13)}}, {BlockOverrides: &BlockOverrides{Number: newInt(12)}}},
			err:        "block numbers must be in order: 12 <= 13",
		},
	} {
		sim := &simulator{base: &types.Header{Number: big.NewInt(int64(tc.baseNumber))}}
		res, err := sim.sanitizeBlockOrder(tc.blocks)
		if err != nil {
			if err.Error() == tc.err {
				continue
			} else {
				t.Fatalf("testcase %d: error mismatch. Want '%s', have '%s'", i, tc.err, err.Error())
			}
		}
		if err == nil && tc.err != "" {
			t.Fatalf("testcase %d: expected err", i)
		}
		if len(res) != tc.expectedLen {
			fmt.Printf("res: %v\n", res)
			t.Errorf("testcase %d: mismatch number of blocks. Want %d, have %d", i, tc.expectedLen, len(res))
		}
		for bi, b := range res {
			if b.BlockOverrides == nil {
				t.Fatalf("testcase %d: block overrides nil", i)
			}
			if b.BlockOverrides.Number == nil {
				t.Fatalf("testcase %d: block number not set", i)
			}
			want := tc.baseNumber + bi + 1
			have := b.BlockOverrides.Number.ToInt().Uint64()
			if uint64(want) != have {
				t.Errorf("testcase %d: block number mismatch. Want %d, have %d", i, want, have)
			}
		}
	}
}

func newInt(n int64) *hexutil.Big {
	return (*hexutil.Big)(big.NewInt(n))
}
