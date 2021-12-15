package bor

import (
	"encoding/hex"
	"encoding/json"
	"io"
	"math/big"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/consensus/bor"
	"github.com/ethereum/go-ethereum/consensus/ethash"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/params"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/ethereum/go-ethereum/tests/bor/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"golang.org/x/crypto/sha3"
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
	sign(t, header, signerKey, init.genesis.Config.Bor)
	block = types.NewBlockWithHeader(header)
	_, err = chain.InsertChain([]*types.Block{block})
	assert.Equal(t,
		*err.(*bor.WrongDifficultyError),
		bor.WrongDifficultyError{Number: spanSize, Expected: expectedDifficulty, Actual: 3, Signer: addr.Bytes()})

	header.Difficulty = new(big.Int).SetUint64(expectedDifficulty)
	sign(t, header, signerKey, init.genesis.Config.Bor)
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

// TestEIP1559Transition tests the following:
//
// 1. A transaction whose gasFeeCap is greater than the baseFee is valid.
// 2. Gas accounting for access lists on EIP-1559 transactions is correct.
// 3. Only the transaction's tip will be received by the coinbase.
// 4. The transaction sender pays for both the tip and baseFee.
// 5. The coinbase receives only the partially realized tip when
//    gasFeeCap - gasTipCap < baseFee.
// 6. Legacy transaction behave as expected (e.g. gasPrice = gasFeeCap = gasTipCap).
func TestEIP1559Transition(t *testing.T) {
	var (
		aa = common.HexToAddress("0x000000000000000000000000000000000000aaaa")

		// Generate a canonical chain to act as the main dataset
		db     = rawdb.NewMemoryDatabase()
		engine = ethash.NewFaker()

		// A sender who makes transactions, has some funds
		key1, _ = crypto.HexToECDSA("b71c71a67e1177ad4e901695e1b4b9ee17ae16c6668d313eac2f96dbcda3f291")
		key2, _ = crypto.HexToECDSA("8a1f9a8f95be41cd7ccb6168179afb4504aefe388d1e14474d32c45c72ce7b7a")
		key3, _ = crypto.HexToECDSA("225171aed3793cba1c029832886d69785b7e77a54a44211226b447aa2d16b058")

		addr1 = crypto.PubkeyToAddress(key1.PublicKey)
		addr2 = crypto.PubkeyToAddress(key2.PublicKey)
		addr3 = crypto.PubkeyToAddress(key3.PublicKey)
		funds = new(big.Int).Mul(common.Big1, big.NewInt(params.Ether))
		gspec = &core.Genesis{
			Config: params.BorTestChainConfig,
			Alloc: core.GenesisAlloc{
				addr1: {Balance: funds},
				addr2: {Balance: funds},
				addr3: {Balance: funds},
				// The address 0xAAAA sloads 0x00 and 0x01
				aa: {
					Code: []byte{
						byte(vm.PC),
						byte(vm.PC),
						byte(vm.SLOAD),
						byte(vm.SLOAD),
					},
					Nonce:   0,
					Balance: big.NewInt(0),
				},
			},
		}
	)

	gspec.Config.BerlinBlock = common.Big0
	gspec.Config.LondonBlock = common.Big0
	genesis := gspec.MustCommit(db)
	signer := types.LatestSigner(gspec.Config)

	blocks, _ := core.GenerateChain(gspec.Config, genesis, engine, db, 1, func(i int, b *core.BlockGen) {
		b.SetCoinbase(common.Address{1})
		// One transaction to 0xAAAA
		accesses := types.AccessList{types.AccessTuple{
			Address:     aa,
			StorageKeys: []common.Hash{{0}},
		}}

		txdata := &types.DynamicFeeTx{
			ChainID:    gspec.Config.ChainID,
			Nonce:      0,
			To:         &aa,
			Gas:        30000,
			GasFeeCap:  newGwei(5),
			GasTipCap:  big.NewInt(2),
			AccessList: accesses,
			Data:       []byte{},
		}
		tx := types.NewTx(txdata)
		tx, _ = types.SignTx(tx, signer, key1)

		b.AddTx(tx)
	})

	diskdb := rawdb.NewMemoryDatabase()
	gspec.MustCommit(diskdb)

	chain, err := core.NewBlockChain(diskdb, nil, gspec.Config, engine, vm.Config{}, nil, nil)
	if err != nil {
		t.Fatalf("failed to create tester chain: %v", err)
	}
	if n, err := chain.InsertChain(blocks); err != nil {
		t.Fatalf("block %d: failed to insert into chain: %v", n, err)
	}

	block := chain.GetBlockByNumber(1)

	// 1+2: Ensure EIP-1559 access lists are accounted for via gas usage.
	expectedGas := params.TxGas + params.TxAccessListAddressGas + params.TxAccessListStorageKeyGas +
		vm.GasQuickStep*2 + params.WarmStorageReadCostEIP2929 + params.ColdSloadCostEIP2929
	if block.GasUsed() != expectedGas {
		t.Fatalf("incorrect amount of gas spent: expected %d, got %d", expectedGas, block.GasUsed())
	}

	state, _ := chain.State()

	// 3: Ensure that miner received only the tx's tip.
	actual := state.GetBalance(block.Coinbase())
	expected := new(big.Int).Add(
		new(big.Int).SetUint64(block.GasUsed()*block.Transactions()[0].GasTipCap().Uint64()),
		ethash.ConstantinopleBlockReward,
	)
	if actual.Cmp(expected) != 0 {
		t.Fatalf("miner balance incorrect: expected %d, got %d", expected, actual)
	}

	// check burnt contract balance
	actual = state.GetBalance(common.HexToAddress(params.BorTestChainConfig.Bor.CalculateBurntContract(block.NumberU64())))
	expected = new(big.Int).Mul(new(big.Int).SetUint64(block.GasUsed()), block.BaseFee())
	burntContractBalance := expected
	if actual.Cmp(expected) != 0 {
		t.Fatalf("burnt contract balance incorrect: expected %d, got %d", expected, actual)
	}

	// 4: Ensure the tx sender paid for the gasUsed * (tip + block baseFee).
	actual = new(big.Int).Sub(funds, state.GetBalance(addr1))
	expected = new(big.Int).SetUint64(block.GasUsed() * (block.Transactions()[0].GasTipCap().Uint64() + block.BaseFee().Uint64()))
	if actual.Cmp(expected) != 0 {
		t.Fatalf("sender balance incorrect: expected %d, got %d", expected, actual)
	}

	blocks, _ = core.GenerateChain(gspec.Config, block, engine, db, 1, func(i int, b *core.BlockGen) {
		b.SetCoinbase(common.Address{2})

		txdata := &types.LegacyTx{
			Nonce:    0,
			To:       &aa,
			Gas:      30000,
			GasPrice: newGwei(5),
		}
		tx := types.NewTx(txdata)
		tx, _ = types.SignTx(tx, signer, key2)

		b.AddTx(tx)
	})

	if n, err := chain.InsertChain(blocks); err != nil {
		t.Fatalf("block %d: failed to insert into chain: %v", n, err)
	}

	block = chain.GetBlockByNumber(2)
	state, _ = chain.State()
	effectiveTip := block.Transactions()[0].GasTipCap().Uint64() - block.BaseFee().Uint64()

	// 6+5: Ensure that miner received only the tx's effective tip.
	actual = state.GetBalance(block.Coinbase())
	expected = new(big.Int).Add(
		new(big.Int).SetUint64(block.GasUsed()*effectiveTip),
		ethash.ConstantinopleBlockReward,
	)
	if actual.Cmp(expected) != 0 {
		t.Fatalf("miner balance incorrect: expected %d, got %d", expected, actual)
	}

	// check burnt contract balance
	actual = state.GetBalance(common.HexToAddress(params.BorTestChainConfig.Bor.CalculateBurntContract(block.NumberU64())))
	expected = new(big.Int).Add(burntContractBalance, new(big.Int).Mul(new(big.Int).SetUint64(block.GasUsed()), block.BaseFee()))
	burntContractBalance = expected
	if actual.Cmp(expected) != 0 {
		t.Fatalf("burnt contract balance incorrect: expected %d, got %d", expected, actual)
	}

	// 4: Ensure the tx sender paid for the gasUsed * (effectiveTip + block baseFee).
	actual = new(big.Int).Sub(funds, state.GetBalance(addr2))
	expected = new(big.Int).SetUint64(block.GasUsed() * (effectiveTip + block.BaseFee().Uint64()))
	if actual.Cmp(expected) != 0 {
		t.Fatalf("sender balance incorrect: expected %d, got %d", expected, actual)
	}

	blocks, _ = core.GenerateChain(gspec.Config, block, engine, db, 1, func(i int, b *core.BlockGen) {
		b.SetCoinbase(common.Address{3})

		txdata := &types.LegacyTx{
			Nonce:    0,
			To:       &aa,
			Gas:      30000,
			GasPrice: newGwei(5),
		}
		tx := types.NewTx(txdata)
		tx, _ = types.SignTx(tx, signer, key3)

		b.AddTx(tx)

		accesses := types.AccessList{types.AccessTuple{
			Address:     aa,
			StorageKeys: []common.Hash{{0}},
		}}

		txdata2 := &types.DynamicFeeTx{
			ChainID:    gspec.Config.ChainID,
			Nonce:      1,
			To:         &aa,
			Gas:        30000,
			GasFeeCap:  newGwei(5),
			GasTipCap:  big.NewInt(2),
			AccessList: accesses,
			Data:       []byte{},
		}
		tx = types.NewTx(txdata2)
		tx, _ = types.SignTx(tx, signer, key3)

		b.AddTx(tx)

	})

	if n, err := chain.InsertChain(blocks); err != nil {
		t.Fatalf("block %d: failed to insert into chain: %v", n, err)
	}

	block = chain.GetBlockByNumber(3)
	state, _ = chain.State()

	// check burnt contract balance
	actual = state.GetBalance(common.HexToAddress(params.BorTestChainConfig.Bor.CalculateBurntContract(block.NumberU64())))
	burntAmount := new(big.Int).Mul(
		block.BaseFee(),
		big.NewInt(int64(block.GasUsed())),
	)
	expected = new(big.Int).Add(burntContractBalance, burntAmount)
	if actual.Cmp(expected) != 0 {
		t.Fatalf("burnt contract balance incorrect: expected %d, got %d", expected, actual)
	}
}

func newGwei(n int64) *big.Int {
	return new(big.Int).Mul(big.NewInt(n), big.NewInt(params.GWei))
}

func TestJaipurFork(t *testing.T) {
	init := buildEthereumInstance(t, rawdb.NewMemoryDatabase())
	chain := init.ethereum.BlockChain()
	engine := init.ethereum.Engine()
	_bor := engine.(*bor.Bor)
	db := init.ethereum.ChainDb()
	block := init.genesis.ToBlock(db)
	for i := uint64(1); i < sprintSize; i++ {
		block = buildNextBlock(t, _bor, chain, block, nil, init.genesis.Config.Bor)
		insertNewBlock(t, chain, block)
		if block.Number().Uint64() == init.genesis.Config.Bor.JaipurBlock-1 {
			assert.Equal(t, testSealHash(block.Header(), init.genesis.Config.Bor), bor.SealHash(block.Header(), init.genesis.Config.Bor))
		}
		if block.Number().Uint64() == init.genesis.Config.Bor.JaipurBlock {
			assert.Equal(t, testSealHash(block.Header(), init.genesis.Config.Bor), bor.SealHash(block.Header(), init.genesis.Config.Bor))
		}
	}
}

// SealHash returns the hash of a block prior to it being sealed.
func testSealHash(header *types.Header, c *params.BorConfig) (hash common.Hash) {
	hasher := sha3.NewLegacyKeccak256()
	testEncodeSigHeader(hasher, header, c)
	hasher.Sum(hash[:0])
	return hash
}

func testEncodeSigHeader(w io.Writer, header *types.Header, c *params.BorConfig) {
	enc := []interface{}{
		header.ParentHash,
		header.UncleHash,
		header.Coinbase,
		header.Root,
		header.TxHash,
		header.ReceiptHash,
		header.Bloom,
		header.Difficulty,
		header.Number,
		header.GasLimit,
		header.GasUsed,
		header.Time,
		header.Extra[:len(header.Extra)-65], // Yes, this will panic if extra is too short
		header.MixDigest,
		header.Nonce,
	}
	if c.IsJaipur(header.Number.Uint64()) {
		if header.BaseFee != nil {
			enc = append(enc, header.BaseFee)
		}
	}
	if err := rlp.Encode(w, enc); err != nil {
		panic("can't encode: " + err.Error())
	}
}
