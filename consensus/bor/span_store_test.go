package bor

import (
	"context"
	"fmt"
	"testing"

	"github.com/ethereum/go-ethereum/consensus/bor/clerk"
	"github.com/ethereum/go-ethereum/consensus/bor/heimdall/checkpoint"
	"github.com/ethereum/go-ethereum/consensus/bor/heimdall/milestone"
	"github.com/ethereum/go-ethereum/consensus/bor/heimdall/span"
	"github.com/stretchr/testify/require"
)

type MockHeimdallClient struct {
}

func (h *MockHeimdallClient) Span(ctx context.Context, spanID uint64) (*span.HeimdallSpan, error) {
	// Throw error for span id 100
	if spanID == 100 {
		return nil, fmt.Errorf("unable to fetch span")
	}

	// For everything else, return hardcoded span assuming length 6400 (except for span 0)
	if spanID == 0 {
		return &span.HeimdallSpan{
			Span: span.Span{
				ID:         0,
				StartBlock: 0,
				EndBlock:   255,
			},
		}, nil
	} else {
		return &span.HeimdallSpan{
			Span: span.Span{
				ID:         spanID,
				StartBlock: 6400*(spanID-1) + 256,
				EndBlock:   6400*spanID + 255,
			},
		}, nil
	}
}

func TestSpanStore_SpanById(t *testing.T) {
	spanStore := NewSpanStore(&MockHeimdallClient{}, nil, "1337")
	ctx := context.Background()

	type Testcase struct {
		id         uint64
		startBlock uint64
		endBlock   uint64
	}

	testcases := []Testcase{
		{id: 0, startBlock: 0, endBlock: 255},
		{id: 1, startBlock: 256, endBlock: 6655},
		{id: 2, startBlock: 6656, endBlock: 13055},
	}

	for _, tc := range testcases {
		t.Run("", func(t *testing.T) {
			span, err := spanStore.spanById(ctx, tc.id)
			require.NoError(t, err, "err in spanById for id=%d", tc.id)
			require.Equal(t, tc.id, span.ID, "invalid id in spanById for id=%d", tc.id)
			require.Equal(t, tc.startBlock, span.StartBlock, "invalid start block in spanById for id=%d", tc.id)
			require.Equal(t, tc.endBlock, span.EndBlock, "invalid end block in spanById for id=%d", tc.id)
		})
	}

	// Ensure cache is updated
	keys := spanStore.store.Keys()
	require.Len(t, keys, 3, "invalid length of keys in span store")

	// Ensure latest known span id is updated
	require.Equal(t, uint64(2), spanStore.latestKnownSpanId, "invalid latest known span id in span store")

	// Ask for a few more spans
	for i := spanStore.latestKnownSpanId; i <= 20; i++ {
		_, err := spanStore.spanById(ctx, i)
		require.NoError(t, err, "err in spanById for id=%d", i)
	}

	// Ensure cache is updated
	keys = spanStore.store.Keys()
	require.Len(t, keys, 10, "invalid length of keys in span store")

	// Ensure latest known span id is updated
	require.Equal(t, uint64(20), spanStore.latestKnownSpanId, "invalid latest known span id in span store")

	// Ensure we're still able to fetch old spans even though they're evicted from cache
	span, err := spanStore.spanById(ctx, 0)
	require.NoError(t, err, "err in spanById after eviction for id=0")
	require.Equal(t, uint64(0), span.ID, "invalid id in spanById after eviction for id=0")
	require.Equal(t, uint64(0), span.StartBlock, "invalid start block in spanById after eviction for id=0")
	require.Equal(t, uint64(255), span.EndBlock, "invalid end block in spanById after eviction for id=0")

	// Try fetching span 100 and ensure error is handled
	span, err = spanStore.spanById(ctx, 100)
	require.Error(t, err, "expected error in spanById for id=100")
	require.Nil(t, span, "expected nil span in spanById for id=100")

	// Ensure latest known span is still the old one
	require.Equal(t, uint64(20), spanStore.latestKnownSpanId, "invalid latest known span id in span store")
}

func TestSpanStore_SpanByBlockNumber(t *testing.T) {
	spanStore := NewSpanStore(&MockHeimdallClient{}, nil, "1337")
	ctx := context.Background()

	type Testcase struct {
		blockNumber uint64
		id          uint64
		startBlock  uint64
		endBlock    uint64
	}

	// Insert a few spans
	for i := spanStore.latestKnownSpanId; i < 3; i++ {
		_, err := spanStore.spanById(ctx, i)
		require.NoError(t, err, "err in spanById for id=%d", i)
	}

	// Ensure cache is updated
	keys := spanStore.store.Keys()
	require.Len(t, keys, 3, "invalid length of keys in span store")

	// Ensure latest known span id is updated
	require.Equal(t, uint64(2), spanStore.latestKnownSpanId, "invalid latest known span id in span store")

	// Ask for current and past spans via block number
	testcases := []Testcase{
		{blockNumber: 0, id: 0, startBlock: 0, endBlock: 255},
		{blockNumber: 1, id: 0, startBlock: 0, endBlock: 255},
		{blockNumber: 255, id: 0, startBlock: 0, endBlock: 255},
		{blockNumber: 256, id: 1, startBlock: 256, endBlock: 6655},
		{blockNumber: 257, id: 1, startBlock: 256, endBlock: 6655},
		{blockNumber: 6000, id: 1, startBlock: 256, endBlock: 6655},
		{blockNumber: 6655, id: 1, startBlock: 256, endBlock: 6655},
		{blockNumber: 6656, id: 2, startBlock: 6656, endBlock: 13055},
		{blockNumber: 10000, id: 2, startBlock: 6656, endBlock: 13055},
		{blockNumber: 13055, id: 2, startBlock: 6656, endBlock: 13055},
	}

	for _, tc := range testcases {
		t.Run("", func(t *testing.T) {
			span, err := spanStore.spanByBlockNumber(ctx, tc.blockNumber)
			require.NoError(t, err, "err in spanByBlockNumber for block=%d", tc.blockNumber)
			require.Equal(t, tc.id, span.ID, "invalid id in spanByBlockNumber for block=%d", tc.blockNumber)
			require.Equal(t, tc.startBlock, span.StartBlock, "invalid start block in spanByBlockNumber for block=%d", tc.blockNumber)
			require.Equal(t, tc.endBlock, span.EndBlock, "invalid end block in spanByBlockNumber for block=%d", tc.blockNumber)
		})
	}

	// Insert a few more spans to trigger eviction
	for i := spanStore.latestKnownSpanId; i <= 20; i++ {
		_, err := spanStore.spanById(ctx, i)
		require.NoError(t, err, "err in spanById for id=%d", i)
	}

	// Ensure cache is updated
	keys = spanStore.store.Keys()
	require.Len(t, keys, 10, "invalid length of keys in span store")

	// Ensure latest known span id is updated
	require.Equal(t, uint64(20), spanStore.latestKnownSpanId, "invalid latest known span id in span store")

	// Ask for current and past spans
	testcases = append(testcases, Testcase{blockNumber: 57856, id: 10, startBlock: 57856, endBlock: 64255})
	testcases = append(testcases, Testcase{blockNumber: 60000, id: 10, startBlock: 57856, endBlock: 64255})
	testcases = append(testcases, Testcase{blockNumber: 64255, id: 10, startBlock: 57856, endBlock: 64255})
	testcases = append(testcases, Testcase{blockNumber: 121856, id: 20, startBlock: 121856, endBlock: 128255})
	testcases = append(testcases, Testcase{blockNumber: 122000, id: 20, startBlock: 121856, endBlock: 128255})
	testcases = append(testcases, Testcase{blockNumber: 128255, id: 20, startBlock: 121856, endBlock: 128255})

	for _, tc := range testcases {
		t.Run("", func(t *testing.T) {
			span, err := spanStore.spanByBlockNumber(ctx, tc.blockNumber)
			require.NoError(t, err, "err in spanByBlockNumber for block=%d", tc.blockNumber)
			require.Equal(t, tc.id, span.ID, "invalid id in spanByBlockNumber for block=%d", tc.blockNumber)
			require.Equal(t, tc.startBlock, span.StartBlock, "invalid start block in spanByBlockNumber for block=%d", tc.blockNumber)
			require.Equal(t, tc.endBlock, span.EndBlock, "invalid end block in spanByBlockNumber for block=%d", tc.blockNumber)
		})
	}

	// Asking for a future span
	span, err := spanStore.spanByBlockNumber(ctx, 128256) // block 128256 belongs to span 21 (future span)
	require.NoError(t, err, "err in spanByBlockNumber for future block 128256")
	require.Equal(t, uint64(21), span.ID, "invalid id in spanByBlockNumber for future block 128256")
	require.Equal(t, uint64(128256), span.StartBlock, "invalid start block in spanByBlockNumber for future block 128256")
	require.Equal(t, uint64(134655), span.EndBlock, "invalid end block in spanByBlockNumber for future block 128256")
}

// Irrelevant to the tests above but necessary for interface compatibility
func (h *MockHeimdallClient) StateSyncEvents(ctx context.Context, fromID uint64, to int64) ([]*clerk.EventRecordWithTime, error) {
	panic("implement me")
}
func (h *MockHeimdallClient) FetchCheckpoint(ctx context.Context, number int64) (*checkpoint.Checkpoint, error) {
	panic("implement me")
}
func (h *MockHeimdallClient) FetchCheckpointCount(ctx context.Context) (int64, error) {
	panic("implement me")
}
func (h *MockHeimdallClient) FetchMilestone(ctx context.Context) (*milestone.Milestone, error) {
	panic("implement me")
}
func (h *MockHeimdallClient) FetchMilestoneCount(ctx context.Context) (int64, error) {
	panic("implement me")
}
func (h *MockHeimdallClient) FetchNoAckMilestone(ctx context.Context, milestoneID string) error {
	panic("implement me")
}
func (h *MockHeimdallClient) FetchLastNoAckMilestone(ctx context.Context) (string, error) {
	panic("implement me")
}
func (h *MockHeimdallClient) FetchMilestoneID(ctx context.Context, milestoneID string) error {
	panic("implement me")
}
func (h *MockHeimdallClient) Close() {
	panic("implement me")
}
