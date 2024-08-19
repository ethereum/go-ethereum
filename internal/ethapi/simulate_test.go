package ethapi

import (
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
)

func TestSimulateSanitizeBlockOrder(t *testing.T) {
	type result struct {
		number    uint64
		timestamp uint64
	}
	for i, tc := range []struct {
		baseNumber    int
		baseTimestamp uint64
		blocks        []simBlock
		expected      []result
		err           string
	}{
		{
			baseNumber:    10,
			baseTimestamp: 50,
			blocks:        []simBlock{{}, {}, {}},
			expected:      []result{{number: 11, timestamp: 51}, {number: 12, timestamp: 52}, {number: 13, timestamp: 53}},
		},
		{
			baseNumber:    10,
			baseTimestamp: 50,
			blocks:        []simBlock{{BlockOverrides: &BlockOverrides{Number: newInt(13), Time: newUint64(70)}}, {}},
			expected:      []result{{number: 11, timestamp: 51}, {number: 12, timestamp: 52}, {number: 13, timestamp: 70}, {number: 14, timestamp: 71}},
		},
		{
			baseNumber:    10,
			baseTimestamp: 50,
			blocks:        []simBlock{{BlockOverrides: &BlockOverrides{Number: newInt(11)}}, {BlockOverrides: &BlockOverrides{Number: newInt(14)}}, {}},
			expected:      []result{{number: 11, timestamp: 51}, {number: 12, timestamp: 52}, {number: 13, timestamp: 53}, {number: 14, timestamp: 54}, {number: 15, timestamp: 55}},
		},
		{
			baseNumber:    10,
			baseTimestamp: 50,
			blocks:        []simBlock{{BlockOverrides: &BlockOverrides{Number: newInt(13)}}, {BlockOverrides: &BlockOverrides{Number: newInt(12)}}},
			err:           "block numbers must be in order: 12 <= 13",
		},
		{
			baseNumber:    10,
			baseTimestamp: 50,
			blocks:        []simBlock{{BlockOverrides: &BlockOverrides{Number: newInt(13), Time: newUint64(52)}}},
			err:           "block timestamps must be in order: 52 <= 52",
		},
		{
			baseNumber:    10,
			baseTimestamp: 50,
			blocks:        []simBlock{{BlockOverrides: &BlockOverrides{Number: newInt(11), Time: newUint64(60)}}, {BlockOverrides: &BlockOverrides{Number: newInt(12), Time: newUint64(55)}}},
			err:           "block timestamps must be in order: 55 <= 60",
		},
		{
			baseNumber:    10,
			baseTimestamp: 50,
			blocks:        []simBlock{{BlockOverrides: &BlockOverrides{Number: newInt(11), Time: newUint64(60)}}, {BlockOverrides: &BlockOverrides{Number: newInt(13), Time: newUint64(61)}}},
			err:           "block timestamps must be in order: 61 <= 61",
		},
	} {
		sim := &simulator{base: &types.Header{Number: big.NewInt(int64(tc.baseNumber)), Time: tc.baseTimestamp}}
		res, err := sim.sanitizeChain(tc.blocks)
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
		if len(res) != len(tc.expected) {
			t.Errorf("testcase %d: mismatch number of blocks. Want %d, have %d", i, len(tc.expected), len(res))
		}
		for bi, b := range res {
			if b.BlockOverrides == nil {
				t.Fatalf("testcase %d: block overrides nil", i)
			}
			if b.BlockOverrides.Number == nil {
				t.Fatalf("testcase %d: block number not set", i)
			}
			if b.BlockOverrides.Time == nil {
				t.Fatalf("testcase %d: block time not set", i)
			}
			if (uint64)(*b.BlockOverrides.Time) != tc.expected[bi].timestamp {
				t.Errorf("testcase %d: block timestamp mismatch. Want %d, have %d", i, tc.expected[bi].timestamp, (uint64)(*b.BlockOverrides.Time))
			}
			have := b.BlockOverrides.Number.ToInt().Uint64()
			if have != tc.expected[bi].number {
				t.Errorf("testcase %d: block number mismatch. Want %d, have %d", i, tc.expected[bi].number, have)
			}
		}
	}
}

func newInt(n int64) *hexutil.Big {
	return (*hexutil.Big)(big.NewInt(n))
}
