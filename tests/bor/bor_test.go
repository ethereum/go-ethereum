package bor

import (
	"encoding/hex"
	"encoding/json"
	"math/big"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"github.com/ethereum/go-ethereum/consensus/bor"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/tests/bor/mocks"
)

var (
	spanPath         = "bor/span/1"
	clerkPath        = "clerk/event-record/list"
	clerkQueryParams = "from-time=%d&to-time=%d&page=%d&limit=50"
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
	// to := int64(block.Header().Time)

	// Insert sprintSize # of blocks so that span is fetched at the start of a new sprint
	for i := uint64(1); i <= spanSize; i++ {
		block = buildNextBlock(t, _bor, chain, block, nil, init.genesis.Config.Bor)
		insertNewBlock(t, chain, block)
	}

	assert.True(t, h.AssertCalled(t, "FetchWithRetry", spanPath, ""))
	validators, err := _bor.GetCurrentValidators(block.Hash(), spanSize) // check validator set at the first block of new span
	if err != nil {
		t.Fatalf("%s", err)
	}

	assert.Equal(t, 3, len(validators))
	for i, validator := range validators {
		assert.Equal(t, validator.Address.Bytes(), heimdallSpan.SelectedProducers[i].Address.Bytes())
		assert.Equal(t, validator.VotingPower, heimdallSpan.SelectedProducers[i].VotingPower)
	}
}

func TestFetchStateSyncEvents(t *testing.T) {
	init := buildEthereumInstance(t, rawdb.NewMemoryDatabase())
	chain := init.ethereum.BlockChain()
	engine := init.ethereum.Engine()
	_bor := engine.(*bor.Bor)

	// A. Insert blocks for 0th sprint
	db := init.ethereum.ChainDb()
	block := init.genesis.ToBlock(db)
	// Insert sprintSize # of blocks so that span is fetched at the start of a new sprint
	for i := uint64(1); i < sprintSize; i++ {
		block = buildNextBlock(t, _bor, chain, block, nil, init.genesis.Config.Bor)
		insertNewBlock(t, chain, block)
	}

	// B. Before inserting 1st block of the next sprint, mock heimdall deps
	// B.1 Mock /bor/span/1
	res, _ := loadSpanFromFile(t)
	h := &mocks.IHeimdallClient{}
	h.On("FetchWithRetry", spanPath, "").Return(res, nil)

	// B.2 Mock State Sync events
	fromID := uint64(1)
	// at # sprintSize, events are fetched for [fromID, (block-sprint).Time)
	to := int64(chain.GetHeaderByNumber(0).Time)
	eventCount := 50

	sample := getSampleEventRecord(t)
	sample.Time = time.Unix(to-int64(eventCount+1), 0) // last event.Time will be just < to
	eventRecords := generateFakeStateSyncEvents(sample, eventCount)
	h.On("FetchStateSyncEvents", fromID, to).Return(eventRecords, nil)
	_bor.SetHeimdallClient(h)

	block = buildNextBlock(t, _bor, chain, block, nil, init.genesis.Config.Bor)
	insertNewBlock(t, chain, block)

	assert.True(t, h.AssertCalled(t, "FetchWithRetry", spanPath, ""))
	assert.True(t, h.AssertCalled(t, "FetchStateSyncEvents", fromID, to))
}

func TestFetchStateSyncEvents_2(t *testing.T) {
	init := buildEthereumInstance(t, rawdb.NewMemoryDatabase())
	chain := init.ethereum.BlockChain()
	engine := init.ethereum.Engine()
	_bor := engine.(*bor.Bor)

	// Mock /bor/span/1
	res, _ := loadSpanFromFile(t)
	h := &mocks.IHeimdallClient{}
	h.On("FetchWithRetry", spanPath, "").Return(res, nil)

	// Mock State Sync events
	// at # sprintSize, events are fetched for [fromID, (block-sprint).Time)
	fromID := uint64(1)
	to := int64(chain.GetHeaderByNumber(0).Time)
	sample := getSampleEventRecord(t)

	// First query will be from [id=1, (block-sprint).Time]
	// Insert 5 events in this time range
	eventRecords := []*bor.EventRecordWithTime{
		buildStateEvent(sample, 1, 3), // id = 1, time = 1
		buildStateEvent(sample, 2, 1), // id = 2, time = 3
		buildStateEvent(sample, 3, 2), // id = 3, time = 2
		// event with id 5 is missing
		buildStateEvent(sample, 4, 5), // id = 4, time = 5
		buildStateEvent(sample, 6, 4), // id = 6, time = 4
	}
	h.On("FetchStateSyncEvents", fromID, to).Return(eventRecords, nil)
	_bor.SetHeimdallClient(h)

	// Insert blocks for 0th sprint
	db := init.ethereum.ChainDb()
	block := init.genesis.ToBlock(db)
	for i := uint64(1); i <= sprintSize; i++ {
		block = buildNextBlock(t, _bor, chain, block, nil, init.genesis.Config.Bor)
		insertNewBlock(t, chain, block)
	}
	assert.True(t, h.AssertCalled(t, "FetchWithRetry", spanPath, ""))
	assert.True(t, h.AssertCalled(t, "FetchStateSyncEvents", fromID, to))
	lastStateID, _ := _bor.GenesisContractsClient.LastStateId(sprintSize)
	// state 6 was not written
	assert.Equal(t, uint64(4), lastStateID.Uint64())

	//
	fromID = uint64(5)
	to = int64(chain.GetHeaderByNumber(sprintSize).Time)
	eventRecords = []*bor.EventRecordWithTime{
		buildStateEvent(sample, 5, 7),
		buildStateEvent(sample, 6, 4),
	}
	h.On("FetchStateSyncEvents", fromID, to).Return(eventRecords, nil)
	for i := sprintSize + 1; i <= spanSize; i++ {
		block = buildNextBlock(t, _bor, chain, block, nil, init.genesis.Config.Bor)
		insertNewBlock(t, chain, block)
	}
	assert.True(t, h.AssertCalled(t, "FetchStateSyncEvents", fromID, to))
	lastStateID, _ = _bor.GenesisContractsClient.LastStateId(spanSize)
	assert.Equal(t, uint64(6), lastStateID.Uint64())
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
	h.On("FetchWithRetry", "bor/span/1", "").Return(res, nil)
	h.On(
		"FetchStateSyncEvents",
		mock.AnythingOfType("uint64"),
		mock.AnythingOfType("int64")).Return([]*bor.EventRecordWithTime{getSampleEventRecord(t)}, nil)
	return h, heimdallSpan
}

func generateFakeStateSyncEvents(sample *bor.EventRecordWithTime, count int) []*bor.EventRecordWithTime {
	events := make([]*bor.EventRecordWithTime, count)
	event := *sample
	event.ID = 1
	events[0] = &bor.EventRecordWithTime{}
	*events[0] = event
	for i := 1; i < count; i++ {
		event.ID = uint64(i)
		event.Time = event.Time.Add(1 * time.Second)
		events[i] = &bor.EventRecordWithTime{}
		*events[i] = event
	}
	return events
}

func buildStateEvent(sample *bor.EventRecordWithTime, id uint64, timeStamp int64) *bor.EventRecordWithTime {
	event := *sample
	event.ID = id
	event.Time = time.Unix(timeStamp, 0)
	return &event
}

func getSampleEventRecord(t *testing.T) *bor.EventRecordWithTime {
	res := stateSyncEventsPayload(t)
	var _eventRecords []*bor.EventRecordWithTime
	if err := json.Unmarshal(res.Result, &_eventRecords); err != nil {
		t.Fatalf("%s", err)
	}
	_eventRecords[0].Time = time.Unix(1, 0)
	return _eventRecords[0]
}
