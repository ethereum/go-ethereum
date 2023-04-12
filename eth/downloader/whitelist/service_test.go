package whitelist

import (
	"errors"
	"fmt"
	"math/big"
	"reflect"
	"sort"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
)

// NewMockService creates a new mock whitelist service
func NewMockService(maxCapacity uint, checkpointInterval uint64) *Service {
	return &Service{
		checkpointWhitelist: make(map[uint64]common.Hash),
		checkpointOrder:     []uint64{},
		maxCapacity:         maxCapacity,
		checkpointInterval:  checkpointInterval,
	}
}

// TestWhitelistCheckpoint checks the checkpoint whitelist map queue mechanism
func TestWhitelistCheckpoint(t *testing.T) {
	t.Parallel()

	s := NewMockService(10, 10)
	for i := 0; i < 10; i++ {
		s.enqueueCheckpointWhitelist(uint64(i), common.Hash{})
	}
	require.Equal(t, s.length(), 10, "expected 10 items in whitelist")

	s.enqueueCheckpointWhitelist(11, common.Hash{})
	s.dequeueCheckpointWhitelist()
	require.Equal(t, s.length(), 10, "expected 10 items in whitelist")
}

// TestIsValidPeer checks the IsValidPeer function in isolation
// for different cases by providing a mock fetchHeadersByNumber function
func TestIsValidPeer(t *testing.T) {
	t.Parallel()

	s := NewMockService(10, 10)

	// case1: no checkpoint whitelist, should consider the chain as valid
	res, err := s.IsValidPeer(nil, nil)
	require.NoError(t, err, "expected no error")
	require.Equal(t, res, true, "expected chain to be valid")

	// add checkpoint entries and mock fetchHeadersByNumber function
	s.ProcessCheckpoint(uint64(0), common.Hash{})
	s.ProcessCheckpoint(uint64(1), common.Hash{})

	require.Equal(t, s.length(), 2, "expected 2 items in whitelist")

	// create a false function, returning absolutely nothing
	falseFetchHeadersByNumber := func(number uint64, amount int, skip int, reverse bool) ([]*types.Header, []common.Hash, error) {
		return nil, nil, nil
	}

	// case2: false fetchHeadersByNumber function provided, should consider the chain as invalid
	// and throw `ErrNoRemoteCheckoint` error
	res, err = s.IsValidPeer(nil, falseFetchHeadersByNumber)
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	if !errors.Is(err, ErrNoRemoteCheckoint) {
		t.Fatalf("expected error ErrNoRemoteCheckoint, got %v", err)
	}

	require.Equal(t, res, false, "expected chain to be invalid")

	// case3: correct fetchHeadersByNumber function provided, should consider the chain as valid
	// create a mock function, returning a the required header
	fetchHeadersByNumber := func(number uint64, _ int, _ int, _ bool) ([]*types.Header, []common.Hash, error) {
		hash := common.Hash{}
		header := types.Header{Number: big.NewInt(0)}

		switch number {
		case 0:
			return []*types.Header{&header}, []common.Hash{hash}, nil
		case 1:
			header.Number = big.NewInt(1)
			return []*types.Header{&header}, []common.Hash{hash}, nil
		case 2:
			header.Number = big.NewInt(1) // sending wrong header for misamatch
			return []*types.Header{&header}, []common.Hash{hash}, nil
		default:
			return nil, nil, errors.New("invalid number")
		}
	}

	res, err = s.IsValidPeer(nil, fetchHeadersByNumber)
	require.NoError(t, err, "expected no error")
	require.Equal(t, res, true, "expected chain to be valid")

	// add one more checkpoint whitelist entry
	s.ProcessCheckpoint(uint64(2), common.Hash{})
	require.Equal(t, s.length(), 3, "expected 3 items in whitelist")

	// case4: correct fetchHeadersByNumber function provided with wrong header
	// for block number 2. Should consider the chain as invalid and throw an error
	res, err = s.IsValidPeer(nil, fetchHeadersByNumber)
	require.Equal(t, err, ErrCheckpointMismatch, "expected checkpoint mismatch error")
	require.Equal(t, res, false, "expected chain to be invalid")
}

// TestIsValidChain checks the IsValidChain function in isolation
// for different cases by providing a mock current header and chain
func TestIsValidChain(t *testing.T) {
	t.Parallel()

	s := NewMockService(10, 10)
	chainA := createMockChain(1, 20) // A1->A2...A19->A20
	// case1: no checkpoint whitelist, should consider the chain as valid
	res, err := s.IsValidChain(nil, chainA)
	require.Equal(t, res, true, "expected chain to be valid")
	require.Equal(t, err, nil, "expected error to be nil")

	tempChain := createMockChain(21, 22) // A21->A22

	// add mock checkpoint entries
	s.ProcessCheckpoint(tempChain[0].Number.Uint64(), tempChain[0].Hash())
	s.ProcessCheckpoint(tempChain[1].Number.Uint64(), tempChain[1].Hash())

	require.Equal(t, s.length(), 2, "expected 2 items in whitelist")

	// case2: We're behind the oldest whitelisted block entry, should consider
	// the chain as valid as we're still far behind the latest blocks
	res, err = s.IsValidChain(chainA[len(chainA)-1], chainA)
	require.Equal(t, res, true, "expected chain to be valid")
	require.Equal(t, err, nil, "expected error to be nil")

	// Clear checkpoint whitelist and add blocks A5 and A15 in whitelist
	s.PurgeCheckpointWhitelist()
	s.ProcessCheckpoint(chainA[5].Number.Uint64(), chainA[5].Hash())
	s.ProcessCheckpoint(chainA[15].Number.Uint64(), chainA[15].Hash())

	require.Equal(t, s.length(), 2, "expected 2 items in whitelist")

	// case3: Try importing a past chain having valid checkpoint, should
	// consider the chain as valid
	res, err = s.IsValidChain(chainA[len(chainA)-1], chainA)
	require.Equal(t, res, true, "expected chain to be valid")
	require.Equal(t, err, nil, "expected error to be nil")

	// Clear checkpoint whitelist and mock blocks in whitelist
	tempChain = createMockChain(20, 20) // A20

	s.PurgeCheckpointWhitelist()
	s.ProcessCheckpoint(tempChain[0].Number.Uint64(), tempChain[0].Hash())

	require.Equal(t, s.length(), 1, "expected 1 items in whitelist")

	// case4: Try importing a past chain having invalid checkpoint
	res, _ = s.IsValidChain(chainA[len(chainA)-1], chainA)
	require.Equal(t, res, false, "expected chain to be invalid")
	// Not checking error here because we return nil in case of checkpoint mismatch

	// create a future chain to be imported of length <= `checkpointInterval`
	chainB := createMockChain(21, 30) // B21->B22...B29->B30

	// case5: Try importing a future chain (1)
	res, err = s.IsValidChain(chainA[len(chainA)-1], chainB)
	require.Equal(t, res, true, "expected chain to be valid")
	require.Equal(t, err, nil, "expected error to be nil")

	// create a future chain to be imported of length > `checkpointInterval`
	chainB = createMockChain(21, 40) // C21->C22...C39->C40

	// Note: Earlier, it used to reject future chains longer than some threshold.
	// That check is removed for now.

	// case6: Try importing a future chain (2)
	res, err = s.IsValidChain(chainA[len(chainA)-1], chainB)
	require.Equal(t, res, true, "expected chain to be valid")
	require.Equal(t, err, nil, "expected error to be nil")
}

func TestSplitChain(t *testing.T) {
	t.Parallel()

	type Result struct {
		pastStart    uint64
		pastEnd      uint64
		futureStart  uint64
		futureEnd    uint64
		pastLength   int
		futureLength int
	}

	// Current chain is at block: X
	// Incoming chain is represented as [N, M]
	testCases := []struct {
		name    string
		current uint64
		chain   []*types.Header
		result  Result
	}{
		{name: "X = 10, N = 11, M = 20", current: uint64(10), chain: createMockChain(11, 20), result: Result{futureStart: 11, futureEnd: 20, futureLength: 10}},
		{name: "X = 10, N = 13, M = 20", current: uint64(10), chain: createMockChain(13, 20), result: Result{futureStart: 13, futureEnd: 20, futureLength: 8}},
		{name: "X = 10, N = 2, M = 10", current: uint64(10), chain: createMockChain(2, 10), result: Result{pastStart: 2, pastEnd: 10, pastLength: 9}},
		{name: "X = 10, N = 2, M = 9", current: uint64(10), chain: createMockChain(2, 9), result: Result{pastStart: 2, pastEnd: 9, pastLength: 8}},
		{name: "X = 10, N = 2, M = 8", current: uint64(10), chain: createMockChain(2, 8), result: Result{pastStart: 2, pastEnd: 8, pastLength: 7}},
		{name: "X = 10, N = 5, M = 15", current: uint64(10), chain: createMockChain(5, 15), result: Result{pastStart: 5, pastEnd: 10, pastLength: 6, futureStart: 11, futureEnd: 15, futureLength: 5}},
		{name: "X = 10, N = 10, M = 20", current: uint64(10), chain: createMockChain(10, 20), result: Result{pastStart: 10, pastEnd: 10, pastLength: 1, futureStart: 11, futureEnd: 20, futureLength: 10}},
	}
	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			past, future := splitChain(tc.current, tc.chain)
			require.Equal(t, len(past), tc.result.pastLength)
			require.Equal(t, len(future), tc.result.futureLength)

			if len(past) > 0 {
				// Check if we have expected block/s
				require.Equal(t, past[0].Number.Uint64(), tc.result.pastStart)
				require.Equal(t, past[len(past)-1].Number.Uint64(), tc.result.pastEnd)
			}

			if len(future) > 0 {
				// Check if we have expected block/s
				require.Equal(t, future[0].Number.Uint64(), tc.result.futureStart)
				require.Equal(t, future[len(future)-1].Number.Uint64(), tc.result.futureEnd)
			}
		})
	}
}

//nolint:gocognit
func TestSplitChainProperties(t *testing.T) {
	t.Parallel()

	// Current chain is at block: X
	// Incoming chain is represented as [N, M]

	currentChain := []int{0, 1, 2, 3, 10, 100} // blocks starting from genesis
	blockDiffs := []int{0, 1, 2, 3, 4, 5, 9, 10, 11, 12, 90, 100, 101, 102}

	caseParams := make(map[int]map[int]map[int]struct{}) // X -> N -> M

	for _, current := range currentChain {
		// past cases only + past to current
		for _, diff := range blockDiffs {
			from := current - diff

			// use int type for everything to not care about underflow
			if from < 0 {
				continue
			}

			for _, diff := range blockDiffs {
				to := current - diff

				if to >= from {
					addTestCaseParams(caseParams, current, from, to)
				}
			}
		}

		// future only + current to future
		for _, diff := range blockDiffs {
			from := current + diff

			if from < 0 {
				continue
			}

			for _, diff := range blockDiffs {
				to := current + diff

				if to >= from {
					addTestCaseParams(caseParams, current, from, to)
				}
			}
		}

		// past-current-future
		for _, diff := range blockDiffs {
			from := current - diff

			if from < 0 {
				continue
			}

			for _, diff := range blockDiffs {
				to := current + diff

				if to >= from {
					addTestCaseParams(caseParams, current, from, to)
				}
			}
		}
	}

	type testCase struct {
		current     int
		remoteStart int
		remoteEnd   int
	}

	var ts []testCase

	// X -> N -> M
	for x, nm := range caseParams {
		for n, mMap := range nm {
			for m := range mMap {
				ts = append(ts, testCase{x, n, m})
			}
		}
	}

	//nolint:paralleltest
	for i, tc := range ts {
		tc := tc

		name := fmt.Sprintf("test case: index = %d, X = %d, N = %d, M = %d", i, tc.current, tc.remoteStart, tc.remoteEnd)

		t.Run(name, func(t *testing.T) {
			t.Parallel()

			chain := createMockChain(uint64(tc.remoteStart), uint64(tc.remoteEnd))

			past, future := splitChain(uint64(tc.current), chain)

			// properties
			if len(past) > 0 {
				// Check if the chain is ordered
				isOrdered := sort.SliceIsSorted(past, func(i, j int) bool {
					return past[i].Number.Uint64() < past[j].Number.Uint64()
				})

				require.True(t, isOrdered, "an ordered past chain expected: %v", past)

				isSequential := sort.SliceIsSorted(past, func(i, j int) bool {
					return past[i].Number.Uint64() == past[j].Number.Uint64()-1
				})

				require.True(t, isSequential, "a sequential past chain expected: %v", past)

				// Check if current block >= past chain's last block
				require.Equal(t, past[len(past)-1].Number.Uint64() <= uint64(tc.current), true)
			}

			if len(future) > 0 {
				// Check if the chain is ordered
				isOrdered := sort.SliceIsSorted(future, func(i, j int) bool {
					return future[i].Number.Uint64() < future[j].Number.Uint64()
				})

				require.True(t, isOrdered, "an ordered future chain expected: %v", future)

				isSequential := sort.SliceIsSorted(future, func(i, j int) bool {
					return future[i].Number.Uint64() == future[j].Number.Uint64()-1
				})

				require.True(t, isSequential, "a sequential future chain expected: %v", future)

				// Check if future chain's first block > current block
				require.Equal(t, future[len(future)-1].Number.Uint64() > uint64(tc.current), true)
			}

			// Check if both chains are continuous
			if len(past) > 0 && len(future) > 0 {
				require.Equal(t, past[len(past)-1].Number.Uint64(), future[0].Number.Uint64()-1)
			}

			// Check if we get the original chain on appending both
			gotChain := append(past, future...)
			require.Equal(t, reflect.DeepEqual(gotChain, chain), true)
		})
	}
}

// createMockChain returns a chain with dummy headers
// starting from `start` to `end` (inclusive)
func createMockChain(start, end uint64) []*types.Header {
	var (
		i     uint64
		idx   uint64
		chain []*types.Header = make([]*types.Header, end-start+1)
	)

	for i = start; i <= end; i++ {
		header := &types.Header{
			Number: big.NewInt(int64(i)),
			Time:   uint64(time.Now().UnixMicro()) + i,
		}
		chain[idx] = header
		idx++
	}

	return chain
}

// mXNM should be initialized
func addTestCaseParams(mXNM map[int]map[int]map[int]struct{}, x, n, m int) {
	//nolint:ineffassign
	mNM, ok := mXNM[x]
	if !ok {
		mNM = make(map[int]map[int]struct{})
		mXNM[x] = mNM
	}

	//nolint:ineffassign
	_, ok = mNM[n]
	if !ok {
		mM := make(map[int]struct{})
		mNM[n] = mM
	}

	mXNM[x][n][m] = struct{}{}
}
