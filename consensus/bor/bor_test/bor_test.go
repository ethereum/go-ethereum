package bortest

import (
	"encoding/hex"
	"fmt"
	"math/big"
	"testing"

	"github.com/maticnetwork/bor/consensus/bor"
	"github.com/maticnetwork/bor/core/rawdb"
	"github.com/maticnetwork/bor/crypto"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"github.com/maticnetwork/bor/core/types"

	"github.com/maticnetwork/bor/mocks"
)

func TestInsertingSpanSizeBlocks(t *testing.T) {
	init := buildEthereumInstance(t, rawdb.NewMemoryDatabase())
	chain := init.ethereum.BlockChain()
	engine := init.ethereum.Engine()
	_bor := engine.(*bor.Bor)
	h, heimdallSpan := getMockedHeimdallClient(t)
	_bor.SetHeimdallClient(h)

	db := init.ethereum.ChainDb()
	block := init.genesis.ToBlock(db)
	var to int64

	// Insert sprintSize # of blocks so that span is fetched at the start of a new sprint
	for i := uint64(1); i <= spanSize; i++ {
		block = buildNextBlock(t, _bor, chain, block, nil, init.genesis.Config.Bor)
		insertNewBlock(t, chain, block)
		if i == sprintSize-1 {
			// at # sprintSize, events are fetched for the internal [from, (block-1).Time)
			to = int64(block.Header().Time)
		}
	}

	assert.True(t, h.AssertCalled(t, "FetchWithRetry", "bor", "span", "1"))
	query := fmt.Sprintf("clerk/event-record/list?from-time=%d&to-time=%d&page=1&limit=50", 1, to)
	assert.True(t, h.AssertCalled(t, "FetchWithRetry", query))
	validators, err := _bor.GetCurrentValidators(sprintSize, spanSize) // new span starts at 256
	if err != nil {
		t.Fatalf("%s", err)
	}

	assert.Equal(t, 3, len(validators))
	for i, validator := range validators {
		assert.Equal(t, validator.Address.Bytes(), heimdallSpan.SelectedProducers[i].Address.Bytes())
		assert.Equal(t, validator.VotingPower, heimdallSpan.SelectedProducers[i].VotingPower)
	}
}

func TestOutOfTurnSigning(t *testing.T) {
	init := buildEthereumInstance(t, rawdb.NewMemoryDatabase())
	chain := init.ethereum.BlockChain()
	engine := init.ethereum.Engine()
	_bor := engine.(*bor.Bor)
	h, _ := getMockedHeimdallClient(t)
	_bor.SetHeimdallClient(h)

	db := init.ethereum.ChainDb()
	block := init.genesis.ToBlock(db)

	for i := uint64(1); i < spanSize; i++ {
		block = buildNextBlock(t, _bor, chain, block, nil, init.genesis.Config.Bor)
		insertNewBlock(t, chain, block)
	}

	// insert spanSize-th block
	// This account is one the out-of-turn validators for 1st (0-indexed) span
	signer := "c8deb0bea5c41afe8e37b4d1bd84e31adff11b09c8c96ff4b605003cce067cd9"
	signerKey, _ := hex.DecodeString(signer)
	key, _ = crypto.HexToECDSA(signer)
	addr = crypto.PubkeyToAddress(key.PublicKey)
	expectedSuccessionNumber := 2

	block = buildNextBlock(t, _bor, chain, block, signerKey, init.genesis.Config.Bor)
	_, err := chain.InsertChain([]*types.Block{block})
	assert.Equal(t,
		*err.(*bor.BlockTooSoonError),
		bor.BlockTooSoonError{Number: spanSize, Succession: expectedSuccessionNumber})

	expectedDifficulty := uint64(3 - expectedSuccessionNumber) // len(validators) - succession
	header := block.Header()
	header.Time += (bor.CalcProducerDelay(header.Number.Uint64(), expectedSuccessionNumber, init.genesis.Config.Bor) -
		bor.CalcProducerDelay(header.Number.Uint64(), 0, init.genesis.Config.Bor))
	sign(t, header, signerKey)
	block = types.NewBlockWithHeader(header)
	_, err = chain.InsertChain([]*types.Block{block})
	assert.Equal(t,
		*err.(*bor.WrongDifficultyError),
		bor.WrongDifficultyError{Number: spanSize, Expected: expectedDifficulty, Actual: 3, Signer: addr.Bytes()})

	header.Difficulty = new(big.Int).SetUint64(expectedDifficulty)
	sign(t, header, signerKey)
	block = types.NewBlockWithHeader(header)
	_, err = chain.InsertChain([]*types.Block{block})
	assert.Nil(t, err)
}

func TestSignerNotFound(t *testing.T) {
	init := buildEthereumInstance(t, rawdb.NewMemoryDatabase())
	chain := init.ethereum.BlockChain()
	engine := init.ethereum.Engine()
	_bor := engine.(*bor.Bor)
	h, _ := getMockedHeimdallClient(t)
	_bor.SetHeimdallClient(h)

	db := init.ethereum.ChainDb()
	block := init.genesis.ToBlock(db)

	// random signer account that is not a part of the validator set
	signer := "3714d99058cd64541433d59c6b391555b2fd9b54629c2b717a6c9c00d1127b6b"
	signerKey, _ := hex.DecodeString(signer)
	key, _ = crypto.HexToECDSA(signer)
	addr = crypto.PubkeyToAddress(key.PublicKey)

	block = buildNextBlock(t, _bor, chain, block, signerKey, init.genesis.Config.Bor)
	_, err := chain.InsertChain([]*types.Block{block})
	assert.Equal(t,
		*err.(*bor.UnauthorizedSignerError),
		bor.UnauthorizedSignerError{Number: 0, Signer: addr.Bytes()})
}

func getMockedHeimdallClient(t *testing.T) (*mocks.IHeimdallClient, *bor.HeimdallSpan) {
	res, heimdallSpan := loadSpanFromFile(t)
	h := &mocks.IHeimdallClient{}
	h.On("FetchWithRetry", "bor", "span", "1").Return(res, nil)

	res = stateSyncEventsPayload(t)
	// query := fmt.Sprintf("clerk/event-record/list?from-time=%d&to-time=%d&page=1&limit=50", 1, 1589709047)
	h.On("FetchWithRetry", mock.AnythingOfType("string")).Return(res, nil)
	return h, heimdallSpan
}
