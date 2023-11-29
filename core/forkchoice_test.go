package core

import (
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/consensus/ethash"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/params"
	"github.com/ethereum/go-ethereum/trie"

	"github.com/stretchr/testify/require"
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

// nolint: tparallel
func TestForkChoice(t *testing.T) {
	t.Parallel()

	// Create mocks for forker
	getTd := func(hash common.Hash, number uint64) *big.Int {
		if number <= 2 {
			return big.NewInt(int64(number))
		}

		return big.NewInt(0)
	}
	mockChainReader := newChainReaderFake(getTd)
	mockForker := NewForkChoice(mockChainReader, nil, nil)

	createHeader := func(number int64, extra []byte) *types.Header {
		return &types.Header{
			Number: big.NewInt(number),
			Extra:  extra,
		}
	}

	// Create headers for different cases
	headerA := createHeader(1, []byte("A"))
	headerB := createHeader(2, []byte("B"))
	headerC := createHeader(3, []byte("C"))
	headerD := createHeader(4, []byte("D")) // 0x96b0f70c01f4d2b1ee2df5b0202c099776f24c9375ffc89d94b880007633961b (hash)
	headerE := createHeader(4, []byte("E")) // 0xdc0acf54354ff86194baeaab983098a49a40218cffcc77a583726fc06c429685 (hash)

	testCases := []struct {
		name     string
		current  *types.Header
		incoming *types.Header
		want     bool
	}{
		{"tdd(incoming) > tdd(current)", headerA, headerB, true},
		{"tdd(current) > tdd(incoming)", headerB, headerA, false},
		{"tdd(current) = tdd(incoming), number(incoming) > number(current)", headerC, headerD, false},
		{"tdd(current) = tdd(incoming), number(current) > number(incoming)", headerD, headerC, true},
		{"tdd(current) = tdd(incoming), number(current) = number(incoming), hash(current) > hash(incoming)", headerE, headerD, false},
		{"tdd(current) = tdd(incoming), number(current) = number(incoming), hash(incoming) > hash(current)", headerD, headerE, true},
	}

	// nolint: paralleltest
	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			res, err := mockForker.ReorgNeeded(tc.current, tc.incoming)
			require.Equal(t, tc.want, res, tc.name)
			require.NoError(t, err, tc.name)
		})
	}
}

func TestPastChainInsert(t *testing.T) {
	t.Parallel()

	var (
		db    = rawdb.NewMemoryDatabase()
		gspec = &Genesis{BaseFee: big.NewInt(params.InitialBaseFee), Config: params.AllEthashProtocolChanges}
	)

	_, _ = gspec.Commit(db, trie.NewDatabase(db))

	hc, err := NewHeaderChain(db, gspec.Config, ethash.NewFaker(), func() bool { return false })
	if err != nil {
		t.Fatal(err)
	}

	// Create mocks for forker
	getTd := func(hash common.Hash, number uint64) *big.Int {
		return big.NewInt(int64(number))
	}
	validate := func(currentHeader *types.Header, chain []*types.Header) (bool, error) {
		// Put all explicit conditions here
		// If canonical chain is empty, and we're importing a chain of 64 blocks
		if currentHeader.Number.Uint64() == uint64(0) && len(chain) == 64 {
			return true, nil
		}
		// If canonical chain is of len 64, and we're importing a past chain from 54-64, then accept it
		if currentHeader.Number.Uint64() == uint64(64) && chain[0].Number.Uint64() == 55 && len(chain) == 10 {
			return true, nil
		}

		return false, nil
	}
	mockChainReader := newChainReaderFake(getTd)
	mockChainValidator := newChainValidatorFake(validate)
	mockForker := NewForkChoice(mockChainReader, nil, mockChainValidator)

	// chain A: G->A1->A2...A64
	genDb, chainA := makeHeaderChainWithGenesis(gspec, 64, ethash.NewFaker(), 10)

	// Inserting 64 headers on an empty chain
	// expecting 1 write status with no error
	testInsert(t, hc, chainA, CanonStatTy, nil, mockForker)

	// The current chain is: G->A1->A2...A64
	// chain B: G->A1->A2...A44->B45->B46...B64
	chainB := makeHeaderChain(gspec.Config, chainA[43], 20, ethash.NewFaker(), genDb, 10)

	// The current chain is: G->A1->A2...A64
	// chain C: G->A1->A2...A54->C55->C56...C64
	chainC := makeHeaderChain(gspec.Config, chainA[53], 10, ethash.NewFaker(), genDb, 10)

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
		db    = rawdb.NewMemoryDatabase()
		gspec = &Genesis{BaseFee: big.NewInt(params.InitialBaseFee), Config: params.AllEthashProtocolChanges}
	)

	_, _ = gspec.Commit(db, trie.NewDatabase(db))

	hc, err := NewHeaderChain(db, gspec.Config, ethash.NewFaker(), func() bool { return false })
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
	genDb, chainA := makeHeaderChainWithGenesis(gspec, 64, ethash.NewFaker(), 10)

	// Inserting 64 headers on an empty chain
	// expecting 1 write status with no error
	testInsert(t, hc, chainA, CanonStatTy, nil, mockForker)

	// The current chain is: G->A1->A2...A64
	// chain B: G->A1->A2...A64->B65->B66...B84
	chainB := makeHeaderChain(gspec.Config, chainA[63], 20, ethash.NewFaker(), genDb, 10)

	// Inserting 20 headers on the canonical chain
	// expecting 0 write status with no error
	testInsert(t, hc, chainB, SideStatTy, nil, mockForker)

	// The current chain is: G->A1->A2...A64
	// chain C: G->A1->A2...A64->C65->C66...C74
	chainC := makeHeaderChain(gspec.Config, chainA[63], 10, ethash.NewFaker(), genDb, 10)

	// Inserting 10 headers on the canonical chain
	// expecting 0 write status with no error
	testInsert(t, hc, chainC, CanonStatTy, nil, mockForker)
}

func TestOverlappingChainInsert(t *testing.T) {
	t.Parallel()

	var (
		db    = rawdb.NewMemoryDatabase()
		gspec = &Genesis{BaseFee: big.NewInt(params.InitialBaseFee), Config: params.AllEthashProtocolChanges}
	)

	_, _ = gspec.Commit(db, trie.NewDatabase(db))

	hc, err := NewHeaderChain(db, gspec.Config, ethash.NewFaker(), func() bool { return false })
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
	genDb, chainA := makeHeaderChainWithGenesis(gspec, 64, ethash.NewFaker(), 10)

	// Inserting 64 headers on an empty chain
	// expecting 1 write status with no error
	testInsert(t, hc, chainA, CanonStatTy, nil, mockForker)

	// The current chain is: G->A1->A2...A64
	// chain B: G->A1->A2...A54->B55->B56...B84
	chainB := makeHeaderChain(gspec.Config, chainA[53], 30, ethash.NewFaker(), genDb, 10)

	// Inserting 20 blocks on canonical chain
	// expecting 2 write status with no error
	testInsert(t, hc, chainB, SideStatTy, nil, mockForker)

	// The current chain is: G->A1->A2...A64
	// chain C: G->A1->A2...A54->C55->C56...C74
	chainC := makeHeaderChain(gspec.Config, chainA[53], 20, ethash.NewFaker(), genDb, 10)

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
func (w *chainValidatorFake) IsValidPeer(fetchHeadersByNumber func(number uint64, amount int, skip int, reverse bool) ([]*types.Header, []common.Hash, error)) (bool, error) {
	return true, nil
}
func (w *chainValidatorFake) IsValidChain(current *types.Header, headers []*types.Header) (bool, error) {
	return w.validate(current, headers)
}
func (w *chainValidatorFake) ProcessCheckpoint(endBlockNum uint64, endBlockHash common.Hash) {}
func (w *chainValidatorFake) ProcessMilestone(endBlockNum uint64, endBlockHash common.Hash)  {}
func (w *chainValidatorFake) ProcessFutureMilestone(num uint64, hash common.Hash) {
}
func (w *chainValidatorFake) GetWhitelistedCheckpoint() (bool, uint64, common.Hash) {
	return false, 0, common.Hash{}
}

func (w *chainValidatorFake) GetWhitelistedMilestone() (bool, uint64, common.Hash) {
	return false, 0, common.Hash{}
}
func (w *chainValidatorFake) PurgeWhitelistedCheckpoint() {}
func (w *chainValidatorFake) PurgeWhitelistedMilestone()  {}
func (w *chainValidatorFake) GetCheckpoints(current, sidechainHeader *types.Header, sidechainCheckpoints []*types.Header) (map[uint64]*types.Header, error) {
	return map[uint64]*types.Header{}, nil
}
func (w *chainValidatorFake) LockMutex(endBlockNum uint64) bool {
	return false
}
func (w *chainValidatorFake) UnlockMutex(doLock bool, milestoneId string, endBlockNum uint64, endBlockHash common.Hash) {
}
func (w *chainValidatorFake) UnlockSprint(endBlockNum uint64) {
}
func (w *chainValidatorFake) RemoveMilestoneID(milestoneId string) {
}
func (w *chainValidatorFake) GetMilestoneIDsList() []string {
	return nil
}
