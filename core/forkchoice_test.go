package core

import (
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/consensus/ethash"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/params"
)

// chainValidatorFake is a mock for the chain validator service
type chainValidatorFake struct {
	validate func(currentHeader *types.Header, chain []*types.Header) (bool, error)
}

// chainReaderFake is a mock for the chain reader service
type chainReaderFake struct {
	getTd func(hash common.Hash, number uint64) *big.Int
}

func newChainValidatorFake(validate func(currentHeader *types.Header, chain []*types.Header) (bool, error)) *chainValidatorFake {
	return &chainValidatorFake{validate: validate}
}

func newChainReaderFake(getTd func(hash common.Hash, number uint64) *big.Int) *chainReaderFake {
	return &chainReaderFake{getTd: getTd}
}

func TestPastChainInsert(t *testing.T) {
	t.Parallel()

	var (
		db      = rawdb.NewMemoryDatabase()
		genesis = (&Genesis{BaseFee: big.NewInt(params.InitialBaseFee)}).MustCommit(db)
	)

	hc, err := NewHeaderChain(db, params.AllEthashProtocolChanges, ethash.NewFaker(), func() bool { return false })
	if err != nil {
		t.Fatal(err)
	}

	// Create mocks for forker
	getTd := func(hash common.Hash, number uint64) *big.Int {
		return big.NewInt(int64(number))
	}
	validate := func(currentHeader *types.Header, chain []*types.Header) (bool, error) {
		// Put all explicit conditions here
		// If canonical chain is empty and we're importing a chain of 64 blocks
		if currentHeader.Number.Uint64() == uint64(0) && len(chain) == 64 {
			return true, nil
		}
		// If canonical chain is of len 64 and we're importing a past chain from 54-64, then accept it
		if currentHeader.Number.Uint64() == uint64(64) && chain[0].Number.Uint64() == 55 && len(chain) == 10 {
			return true, nil
		}

		return false, nil
	}
	mockChainReader := newChainReaderFake(getTd)
	mockChainValidator := newChainValidatorFake(validate)
	mockForker := NewForkChoice(mockChainReader, nil, mockChainValidator)

	// chain A: G->A1->A2...A64
	chainA := makeHeaderChain(genesis.Header(), 64, ethash.NewFaker(), db, 10)

	// Inserting 64 headers on an empty chain
	// expecting 1 write status with no error
	testInsert(t, hc, chainA, CanonStatTy, nil, mockForker)

	// The current chain is: G->A1->A2...A64
	// chain B: G->A1->A2...A44->B45->B46...B64
	chainB := makeHeaderChain(chainA[43], 20, ethash.NewFaker(), db, 10)

	// The current chain is: G->A1->A2...A64
	// chain C: G->A1->A2...A54->C55->C56...C64
	chainC := makeHeaderChain(chainA[53], 10, ethash.NewFaker(), db, 10)

	// Update the function to consider chainC with higher difficulty
	getTd = func(hash common.Hash, number uint64) *big.Int {
		td := big.NewInt(int64(number))
		if hash == chainB[len(chainB)-1].Hash() || hash == chainC[len(chainC)-1].Hash() {
			td = big.NewInt(65)
		}

		return td
	}
	mockChainReader = newChainReaderFake(getTd)
	mockForker = NewForkChoice(mockChainReader, nil, mockChainValidator)

	// Inserting 20 blocks from chainC on canonical chain
	// expecting 2 write status with no error
	testInsert(t, hc, chainB, SideStatTy, nil, mockForker)

	// Inserting 10 blocks from chainB on canonical chain
	// expecting 1 write status with no error
	testInsert(t, hc, chainC, CanonStatTy, nil, mockForker)
}

func TestFutureChainInsert(t *testing.T) {
	t.Parallel()

	var (
		db      = rawdb.NewMemoryDatabase()
		genesis = (&Genesis{BaseFee: big.NewInt(params.InitialBaseFee)}).MustCommit(db)
	)

	hc, err := NewHeaderChain(db, params.AllEthashProtocolChanges, ethash.NewFaker(), func() bool { return false })
	if err != nil {
		t.Fatal(err)
	}

	// Create mocks for forker
	getTd := func(hash common.Hash, number uint64) *big.Int {
		return big.NewInt(int64(number))
	}
	validate := func(currentHeader *types.Header, chain []*types.Header) (bool, error) {
		// Put all explicit conditions here
		// If canonical chain is empty and we're importing a chain of 64 blocks
		if currentHeader.Number.Uint64() == uint64(0) && len(chain) == 64 {
			return true, nil
		}
		// If length of future chains > some value, they should not be accepted
		if currentHeader.Number.Uint64() == uint64(64) && len(chain) <= 10 {
			return true, nil
		}

		return false, nil
	}
	mockChainReader := newChainReaderFake(getTd)
	mockChainValidator := newChainValidatorFake(validate)
	mockForker := NewForkChoice(mockChainReader, nil, mockChainValidator)

	// chain A: G->A1->A2...A64
	chainA := makeHeaderChain(genesis.Header(), 64, ethash.NewFaker(), db, 10)

	// Inserting 64 headers on an empty chain
	// expecting 1 write status with no error
	testInsert(t, hc, chainA, CanonStatTy, nil, mockForker)

	// The current chain is: G->A1->A2...A64
	// chain B: G->A1->A2...A64->B65->B66...B84
	chainB := makeHeaderChain(chainA[63], 20, ethash.NewFaker(), db, 10)

	// Inserting 20 headers on the canonical chain
	// expecting 0 write status with no error
	testInsert(t, hc, chainB, SideStatTy, nil, mockForker)

	// The current chain is: G->A1->A2...A64
	// chain C: G->A1->A2...A64->C65->C66...C74
	chainC := makeHeaderChain(chainA[63], 10, ethash.NewFaker(), db, 10)

	// Inserting 10 headers on the canonical chain
	// expecting 0 write status with no error
	testInsert(t, hc, chainC, CanonStatTy, nil, mockForker)
}

func TestOverlappingChainInsert(t *testing.T) {
	t.Parallel()

	var (
		db      = rawdb.NewMemoryDatabase()
		genesis = (&Genesis{BaseFee: big.NewInt(params.InitialBaseFee)}).MustCommit(db)
	)

	hc, err := NewHeaderChain(db, params.AllEthashProtocolChanges, ethash.NewFaker(), func() bool { return false })
	if err != nil {
		t.Fatal(err)
	}

	// Create mocks for forker
	getTd := func(hash common.Hash, number uint64) *big.Int {
		return big.NewInt(int64(number))
	}
	validate := func(currentHeader *types.Header, chain []*types.Header) (bool, error) {
		// Put all explicit conditions here
		// If canonical chain is empty and we're importing a chain of 64 blocks
		if currentHeader.Number.Uint64() == uint64(0) && len(chain) == 64 {
			return true, nil
		}
		// If length of chain is > some fixed value then don't accept it
		if currentHeader.Number.Uint64() == uint64(64) && len(chain) <= 20 {
			return true, nil
		}

		return false, nil
	}
	mockChainReader := newChainReaderFake(getTd)
	mockChainValidator := newChainValidatorFake(validate)
	mockForker := NewForkChoice(mockChainReader, nil, mockChainValidator)

	// chain A: G->A1->A2...A64
	chainA := makeHeaderChain(genesis.Header(), 64, ethash.NewFaker(), db, 10)

	// Inserting 64 headers on an empty chain
	// expecting 1 write status with no error
	testInsert(t, hc, chainA, CanonStatTy, nil, mockForker)

	// The current chain is: G->A1->A2...A64
	// chain B: G->A1->A2...A54->B55->B56...B84
	chainB := makeHeaderChain(chainA[53], 30, ethash.NewFaker(), db, 10)

	// Inserting 20 blocks on canonical chain
	// expecting 2 write status with no error
	testInsert(t, hc, chainB, SideStatTy, nil, mockForker)

	// The current chain is: G->A1->A2...A64
	// chain C: G->A1->A2...A54->C55->C56...C74
	chainC := makeHeaderChain(chainA[53], 20, ethash.NewFaker(), db, 10)

	// Inserting 10 blocks on canonical chain
	// expecting 1 write status with no error
	testInsert(t, hc, chainC, CanonStatTy, nil, mockForker)
}

// Mock chain reader functions
func (c *chainReaderFake) Config() *params.ChainConfig {
	return &params.ChainConfig{TerminalTotalDifficulty: nil}
}
func (c *chainReaderFake) GetTd(hash common.Hash, number uint64) *big.Int {
	return c.getTd(hash, number)
}

// Mock chain validator functions
func (w *chainValidatorFake) IsValidPeer(remoteHeader *types.Header, fetchHeadersByNumber func(number uint64, amount int, skip int, reverse bool) ([]*types.Header, []common.Hash, error)) (bool, error) {
	return true, nil
}
func (w *chainValidatorFake) IsValidChain(current *types.Header, headers []*types.Header) (bool, error) {
	return w.validate(current, headers)
}
func (w *chainValidatorFake) ProcessCheckpoint(endBlockNum uint64, endBlockHash common.Hash) {}
func (w *chainValidatorFake) GetCheckpointWhitelist() map[uint64]common.Hash {
	return nil
}
func (w *chainValidatorFake) PurgeCheckpointWhitelist() {}
func (w *chainValidatorFake) GetCheckpoints(current, sidechainHeader *types.Header, sidechainCheckpoints []*types.Header) (map[uint64]*types.Header, error) {
	return map[uint64]*types.Header{}, nil
}
