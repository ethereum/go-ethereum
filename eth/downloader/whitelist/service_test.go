package whitelist

import (
	"errors"
	"math/big"
	"testing"

	"gotest.tools/assert"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
)

// NewMockService creates a new mock whitelist service
func NewMockService(maxCapacity uint) *Service {
	return &Service{
		checkpointWhitelist: make(map[uint64]common.Hash),
		checkpointOrder:     []uint64{},
		maxCapacity:         maxCapacity,
	}
}

// TestWhitelistCheckpoint checks the checkpoint whitelist map queue mechanism
func TestWhitelistCheckpoint(t *testing.T) {
	t.Parallel()

	s := NewMockService(10)
	for i := 0; i < 10; i++ {
		s.enqueueCheckpointWhitelist(uint64(i), common.Hash{})
	}
	assert.Equal(t, s.length(), 10, "expected 10 items in whitelist")

	s.enqueueCheckpointWhitelist(11, common.Hash{})
	s.dequeueCheckpointWhitelist()
	assert.Equal(t, s.length(), 10, "expected 10 items in whitelist")
}

// TestIsValidChain checks che IsValidChain function in isolation
// for different cases by providing a mock fetchHeadersByNumber function
func TestIsValidChain(t *testing.T) {
	t.Parallel()

	s := NewMockService(10)

	// case1: no checkpoint whitelist, should consider the chain as valid
	res, err := s.IsValidChain(nil, nil)
	assert.NilError(t, err, "expected no error")
	assert.Equal(t, res, true, "expected chain to be valid")

	// add checkpoint entries and mock fetchHeadersByNumber function
	s.ProcessCheckpoint(uint64(0), common.Hash{})
	s.ProcessCheckpoint(uint64(1), common.Hash{})

	assert.Equal(t, s.length(), 2, "expected 2 items in whitelist")

	// create a false function, returning absolutely nothing
	falseFetchHeadersByNumber := func(number uint64, amount int, skip int, reverse bool) ([]*types.Header, []common.Hash, error) {
		return nil, nil, nil
	}

	// case2: false fetchHeadersByNumber function provided, should consider the chain as invalid
	// and throw `ErrNoRemoteCheckoint` error
	res, err = s.IsValidChain(nil, falseFetchHeadersByNumber)
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	if !errors.Is(err, ErrNoRemoteCheckoint) {
		t.Fatalf("expected error ErrNoRemoteCheckoint, got %v", err)
	}

	assert.Equal(t, res, false, "expected chain to be invalid")

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

	res, err = s.IsValidChain(nil, fetchHeadersByNumber)
	assert.NilError(t, err, "expected no error")
	assert.Equal(t, res, true, "expected chain to be valid")

	// add one more checkpoint whitelist entry
	s.ProcessCheckpoint(uint64(2), common.Hash{})
	assert.Equal(t, s.length(), 3, "expected 3 items in whitelist")

	// case4: correct fetchHeadersByNumber function provided with wrong header
	// for block number 2. Should consider the chain as invalid and throw an error
	res, err = s.IsValidChain(nil, fetchHeadersByNumber)
	assert.Equal(t, err, ErrCheckpointMismatch, "expected checkpoint mismatch error")
	assert.Equal(t, res, false, "expected chain to be invalid")
}
