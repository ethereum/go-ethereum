package eth

import (
	"context"
	"math/big"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/consensus/bor"
	"github.com/ethereum/go-ethereum/consensus/bor/clerk"
	"github.com/ethereum/go-ethereum/consensus/bor/heimdall/checkpoint"
	"github.com/ethereum/go-ethereum/consensus/bor/heimdall/span"
)

type mockHeimdall struct {
	fetchCheckpoint      func(ctx context.Context, number int64) (*checkpoint.Checkpoint, error)
	fetchCheckpointCount func(ctx context.Context) (int64, error)
}

func (m *mockHeimdall) StateSyncEvents(ctx context.Context, fromID uint64, to int64) ([]*clerk.EventRecordWithTime, error) {
	return nil, nil
}
func (m *mockHeimdall) Span(ctx context.Context, spanID uint64) (*span.HeimdallSpan, error) {
	//nolint:nilnil
	return nil, nil
}
func (m *mockHeimdall) FetchCheckpoint(ctx context.Context, number int64) (*checkpoint.Checkpoint, error) {
	return m.fetchCheckpoint(ctx, number)
}
func (m *mockHeimdall) FetchCheckpointCount(ctx context.Context) (int64, error) {
	return m.fetchCheckpointCount(ctx)
}
func (m *mockHeimdall) Close() {}

func TestFetchWhitelistCheckpoints(t *testing.T) {
	t.Parallel()

	// create an empty ethHandler
	handler := &ethHandler{}

	// create a mock checkpoint verification function and use it to create a verifier
	verify := func(ctx context.Context, handler *ethHandler, checkpoint *checkpoint.Checkpoint) (string, error) {
		return "", nil
	}

	verifier := newCheckpointVerifier(verify)

	// Create a mock heimdall instance and use it for creating a bor instance
	var heimdall mockHeimdall

	bor := &bor.Bor{HeimdallClient: &heimdall}

	// create 20 mock checkpoints
	checkpoints := createMockCheckpoints(20)

	// create a mock fetch checkpoint function
	heimdall.fetchCheckpoint = func(_ context.Context, number int64) (*checkpoint.Checkpoint, error) {
		return checkpoints[number-1], nil // we're sure that number won't exceed 20
	}

	// create a background context
	ctx := context.Background()

	testCases := []struct {
		name        string
		first       bool
		count       int64
		length      int
		start       uint64
		end         uint64
		fetchErr    error
		expectedErr error
	}{
		{"fail to fetch checkpoint count", false, 0, 0, 0, 0, errCheckpointCount, errCheckpointCount},
		{"no checkpoints available", false, 0, 0, 0, 0, nil, errNoCheckpoint},
		{"fetch multiple checkpoints (count < 10)", true, 6, 6, 0, 6, nil, nil},
		{"fetch multiple checkpoints (count = 10)", true, 10, 10, 0, 10, nil, nil},
		{"fetch multiple checkpoints (count > 10)", true, 16, 10, 6, 16, nil, nil},
		{"fetch single checkpoint", false, 18, 1, 17, 18, nil, nil},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			heimdall.fetchCheckpointCount = getMockFetchCheckpointFn(tc.count, tc.fetchErr)
			blockNums, blockHashes, err := handler.fetchWhitelistCheckpoints(ctx, bor, verifier, tc.first)

			// Check if we have expected result
			require.Equal(t, tc.expectedErr, err)
			require.Equal(t, tc.length, len(blockNums))
			require.Equal(t, tc.length, len(blockHashes))
			validateBlockNumber(t, blockNums, checkpoints[tc.start:tc.end])
		})
	}
}

func validateBlockNumber(t *testing.T, blockNums []uint64, checkpoints []*checkpoint.Checkpoint) {
	t.Helper()

	for i, blockNum := range blockNums {
		require.Equal(t, blockNum, checkpoints[i].EndBlock.Uint64(), "expect block number in array to match with checkpoint")
	}
}

func getMockFetchCheckpointFn(number int64, err error) func(ctx context.Context) (int64, error) {
	return func(_ context.Context) (int64, error) {
		return number, err
	}
}

func createMockCheckpoints(count int) []*checkpoint.Checkpoint {
	var (
		checkpoints []*checkpoint.Checkpoint = make([]*checkpoint.Checkpoint, count)
		startBlock  int64                    = 257 // any number can be used
	)

	for i := 0; i < count; i++ {
		checkpoints[i] = &checkpoint.Checkpoint{
			Proposer:   common.Address{},
			StartBlock: big.NewInt(startBlock),
			EndBlock:   big.NewInt(startBlock + 255),
			RootHash:   common.Hash{},
			BorChainID: "137",
			Timestamp:  uint64(time.Now().Unix()),
		}
		startBlock += 256
	}

	return checkpoints
}
