package bortest

import (
	"encoding/hex"
	"math/big"
	"testing"

	"github.com/maticnetwork/bor/common"
	"github.com/maticnetwork/bor/consensus/bor"
	"github.com/maticnetwork/bor/core/rawdb"
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
	// FetchWithRetry is invoked 3 times
	// 1. bor.FinalizeAndAssemble to prepare a new block when calling insertNewBlock
	// 2. bor.Finalize via(bc.insertChain => bc.processor.Process)
	// 3. bor.FinalizeAndAssemble via worker.commit
	h.On("FetchWithRetry", "bor", "span", "1").Return(res, nil).Times(3)
	_bor.SetHeimdallClient(h)

	db := init.ethereum.ChainDb()
	statedb, err := chain.State()
	if err != nil {
		t.Fatalf("%s", err)
	}
	_key, _ := hex.DecodeString(privKey)

	block := init.genesis.ToBlock(db)
	// Insert sprintSize # of blocks so that span is fetched at the start of a new sprint
	for i := uint64(1); i <= sprintSize; i++ {
		header := buildMinimalNextHeader(t, block, init.genesis.Config.Bor)
		block = insertNewBlock(t, _bor, chain, header, statedb, _key)
	}

	assert.True(t, h.AssertNumberOfCalls(t, "FetchWithRetry", 3))
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
	statedb, err := chain.State()
	if err != nil {
		t.Fatalf("%s", err)
	}
	_key, _ := hex.DecodeString(privKey)

	block := init.genesis.ToBlock(db)
	for i := uint64(1); i <= spanSize; i++ {
		header := buildMinimalNextHeader(t, block, init.genesis.Config.Bor)
		block = insertNewBlock(t, _bor, chain, header, statedb, _key)
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
