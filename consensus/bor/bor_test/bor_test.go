package bortest

import (
	"encoding/hex"
	"math/big"
	"testing"

	"github.com/maticnetwork/bor/common"
	"github.com/maticnetwork/bor/consensus/bor"
	"github.com/maticnetwork/bor/core/rawdb"
	"github.com/maticnetwork/bor/crypto"
	"github.com/stretchr/testify/assert"

	"github.com/maticnetwork/bor/core/types"

	"github.com/maticnetwork/bor/mocks"
)

func TestCommitSpan(t *testing.T) {
	init := buildEthereumInstance(t, rawdb.NewMemoryDatabase())
	chain := init.ethereum.BlockChain()
	engine := init.ethereum.Engine()
	_bor := engine.(*bor.Bor)

	// Mock HeimdallClient.FetchWithRetry to return span data from span.json
	res, heimdallSpan := loadSpanFromFile(t)
	h := &mocks.IHeimdallClient{}
	h.On("FetchWithRetry", "bor", "span", "1").Return(res, nil)
	_bor.SetHeimdallClient(h)

	db := init.ethereum.ChainDb()
	block := init.genesis.ToBlock(db)
	// Insert sprintSize # of blocks so that span is fetched at the start of a new sprint
	for i := uint64(1); i <= sprintSize; i++ {
		block = buildNextBlock(t, _bor, chain, block, nil, init.genesis.Config.Bor)
		insertNewBlock(t, chain, block)
	}

	// FetchWithRetry is invoked 2 times
	// 1. bor.FinalizeAndAssemble to prepare a new block when calling buildNextBlock
	// 2. bor.Finalize via(bc.insertChain => bc.processor.Process)
	assert.True(t, h.AssertCalled(t, "FetchWithRetry", "bor", "span", "1"))
	validators, err := _bor.GetCurrentValidators(sprintSize, 256) // new span starts at 256
	if err != nil {
		t.Fatalf("%s", err)
	}

	assert.Equal(t, 3, len(validators))
	for i, validator := range validators {
		assert.Equal(t, validator.Address.Bytes(), heimdallSpan.SelectedProducers[i].Address.Bytes())
		assert.Equal(t, validator.VotingPower, heimdallSpan.SelectedProducers[i].VotingPower)
	}
}

func TestIsValidatorAction(t *testing.T) {
	init := buildEthereumInstance(t, rawdb.NewMemoryDatabase())
	chain := init.ethereum.BlockChain()
	engine := init.ethereum.Engine()
	_bor := engine.(*bor.Bor)

	proposeStateData, _ := hex.DecodeString("ede01f170000000000000000000000000000000000000000000000000000000000000000")
	proposeSpanData, _ := hex.DecodeString("4b0e4d17")
	var tx *types.Transaction
	tx = types.NewTransaction(
		0,
		common.HexToAddress(chain.Config().Bor.StateReceiverContract),
		big.NewInt(0), 0, big.NewInt(0),
		proposeStateData,
	)
	assert.True(t, _bor.IsValidatorAction(chain, addr, tx))

	tx = types.NewTransaction(
		0,
		common.HexToAddress(chain.Config().Bor.ValidatorContract),
		big.NewInt(0), 0, big.NewInt(0),
		proposeSpanData,
	)
	assert.True(t, _bor.IsValidatorAction(chain, addr, tx))

	res, heimdallSpan := loadSpanFromFile(t)
	h := &mocks.IHeimdallClient{}
	h.On("FetchWithRetry", "bor", "span", "1").Return(res, nil)
	_bor.SetHeimdallClient(h)

	db := init.ethereum.ChainDb()
	block := init.genesis.ToBlock(db)

	for i := uint64(1); i <= spanSize; i++ {
		block = buildNextBlock(t, _bor, chain, block, nil, init.genesis.Config.Bor)
		insertNewBlock(t, chain, block)
	}

	for _, validator := range heimdallSpan.SelectedProducers {
		_addr := validator.Address
		tx = types.NewTransaction(
			0,
			common.HexToAddress(chain.Config().Bor.StateReceiverContract),
			big.NewInt(0), 0, big.NewInt(0),
			proposeStateData,
		)
		assert.True(t, _bor.IsValidatorAction(chain, _addr, tx))

		tx = types.NewTransaction(
			0,
			common.HexToAddress(chain.Config().Bor.ValidatorContract),
			big.NewInt(0), 0, big.NewInt(0),
			proposeSpanData,
		)
		assert.True(t, _bor.IsValidatorAction(chain, _addr, tx))
	}
}

func TestOutOfTurnSigning(t *testing.T) {
	init := buildEthereumInstance(t, rawdb.NewMemoryDatabase())
	chain := init.ethereum.BlockChain()
	engine := init.ethereum.Engine()
	_bor := engine.(*bor.Bor)

	res, _ := loadSpanFromFile(t)
	h := &mocks.IHeimdallClient{}
	h.On("FetchWithRetry", "bor", "span", "1").Return(res, nil)
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
