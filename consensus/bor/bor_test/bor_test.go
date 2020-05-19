package bortest

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math/big"
	"testing"
	"time"

	"github.com/maticnetwork/bor/consensus/bor"
	"github.com/maticnetwork/bor/core/rawdb"

	"github.com/maticnetwork/bor/crypto"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"github.com/maticnetwork/bor/core/types"

	"github.com/maticnetwork/bor/mocks"
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
	to := int64(block.Header().Time)

	// Insert sprintSize # of blocks so that span is fetched at the start of a new sprint
	for i := uint64(1); i <= spanSize; i++ {
		block = buildNextBlock(t, _bor, chain, block, nil, init.genesis.Config.Bor)
		insertNewBlock(t, chain, block)
	}

	assert.True(t, h.AssertCalled(t, "FetchWithRetry", spanPath, ""))
	assert.True(t, h.AssertCalled(t, "FetchWithRetry", clerkPath, fmt.Sprintf(clerkQueryParams, 1, to, 1)))
	validators, err := _bor.GetCurrentValidators(sprintSize, spanSize) // check validator set at the first block of new span
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
	// read heimdall api response from file
	res = stateSyncEventsPayload(t)
	var _eventRecords []*bor.EventRecordWithTime
	if err := json.Unmarshal(res.Result, &_eventRecords); err != nil {
		t.Fatalf("%s", err)
	}

	// use that as a sample to generate bor.stateFetchLimit events
	eventRecords := generateFakeStateSyncEvents(_eventRecords[0], 50)
	_res, _ := json.Marshal(eventRecords)
	response := bor.ResponseWithHeight{Height: "0"}
	if err := json.Unmarshal(_res, &response.Result); err != nil {
		t.Fatalf("%s", err)
	}

	// at # sprintSize, events are fetched for the interval [from, (block-sprint).Time)
	from := 1
	to := int64(chain.GetHeaderByNumber(0).Time)
	page := 1
	query1Params := fmt.Sprintf(clerkQueryParams, from, to, page)
	h.On("FetchWithRetry", clerkPath, query1Params).Return(&response, nil)

	page = 2
	query2Params := fmt.Sprintf(clerkQueryParams, from, to, page)
	h.On("FetchWithRetry", clerkPath, query2Params).Return(&bor.ResponseWithHeight{}, nil)
	_bor.SetHeimdallClient(h)

	block = buildNextBlock(t, _bor, chain, block, nil, init.genesis.Config.Bor)
	insertNewBlock(t, chain, block)

	assert.True(t, h.AssertCalled(t, "FetchWithRetry", spanPath, ""))
	assert.True(t, h.AssertCalled(t, "FetchWithRetry", clerkPath, query1Params))
	assert.True(t, h.AssertCalled(t, "FetchWithRetry", clerkPath, query2Params))
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
	h.On("FetchWithRetry", mock.AnythingOfType("string"), mock.AnythingOfType("string")).Return(stateSyncEventsPayload(t), nil)
	return h, heimdallSpan
}

func generateFakeStateSyncEvents(sample *bor.EventRecordWithTime, count int) []*bor.EventRecordWithTime {
	events := make([]*bor.EventRecordWithTime, count)
	event := *sample
	event.ID = 0
	event.Time = time.Now()
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
