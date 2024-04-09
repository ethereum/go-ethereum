//go:build integration
// +build integration

package bor

import (
	"context"
	"crypto/ecdsa"
	"encoding/hex"
	"fmt"
	"io"
	"math/big"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/sha3"

	"github.com/ethereum/go-ethereum/accounts"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/fdlimit"
	"github.com/ethereum/go-ethereum/consensus/bor"
	"github.com/ethereum/go-ethereum/consensus/bor/clerk"
	"github.com/ethereum/go-ethereum/consensus/bor/heimdall/checkpoint"
	"github.com/ethereum/go-ethereum/consensus/bor/heimdall/milestone"
	"github.com/ethereum/go-ethereum/consensus/bor/valset"
	"github.com/ethereum/go-ethereum/consensus/ethash"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/eth"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/node"
	"github.com/ethereum/go-ethereum/p2p/enode"
	"github.com/ethereum/go-ethereum/params"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/ethereum/go-ethereum/tests/bor/mocks"
	"github.com/ethereum/go-ethereum/trie"
)

var (
	// addr1 = 0x71562b71999873DB5b286dF957af199Ec94617F7
	pkey1, _ = crypto.HexToECDSA("b71c71a67e1177ad4e901695e1b4b9ee17ae16c6668d313eac2f96dbcda3f291")
	// addr2 = 0x9fB29AAc15b9A4B7F17c3385939b007540f4d791
	pkey2, _ = crypto.HexToECDSA("9b28f36fbd67381120752d6172ecdcf10e06ab2d9a1367aac00cdcd6ac7855d3")
)

func TestValidatorWentOffline(t *testing.T) {

	log.SetDefault(log.NewLogger(log.NewTerminalHandlerWithLevel(os.Stderr, log.LevelInfo, true)))
	fdlimit.Raise(2048)

	// Generate a batch of accounts to seal and fund with
	faucets := make([]*ecdsa.PrivateKey, 128)
	for i := 0; i < len(faucets); i++ {
		faucets[i], _ = crypto.GenerateKey()
	}

	// Create an Ethash network based off of the Ropsten config
	// Generate a batch of accounts to seal and fund with
	genesis := InitGenesis(t, faucets, "./testdata/genesis_2val.json", 8)

	var (
		stacks []*node.Node
		nodes  []*eth.Ethereum
		enodes []*enode.Node
	)
	for i := 0; i < 2; i++ {
		// Start the node and wait until it's up
		stack, ethBackend, err := InitMiner(genesis, keys[i], true)
		if err != nil {
			panic(err)
		}
		defer stack.Close()

		for stack.Server().NodeInfo().Ports.Listener == 0 {
			time.Sleep(250 * time.Millisecond)
		}
		// Connect the node to all the previous ones
		for _, n := range enodes {
			stack.Server().AddPeer(n)
		}
		// Start tracking the node and its enode
		stacks = append(stacks, stack)
		nodes = append(nodes, ethBackend)
		enodes = append(enodes, stack.Server().Self())
	}

	// Iterate over all the nodes and start mining
	time.Sleep(3 * time.Second)
	for _, node := range nodes {
		if err := node.StartMining(); err != nil {
			panic(err)
		}
	}

	for {

		// for block 1 to 8, the primary validator is node0
		// for block 9 to 16, the primary validator is node1
		// for block 17 to 24, the primary validator is node0
		// for block 25 to 32, the primary validator is node1
		blockHeaderVal0 := nodes[0].BlockChain().CurrentHeader()

		// we remove peer connection between node0 and node1
		if blockHeaderVal0.Number.Uint64() == 9 {
			stacks[0].Server().RemovePeer(enodes[1])
		}

		// here, node1 is the primary validator, node0 will sign out-of-turn

		// we add peer connection between node1 and node0
		if blockHeaderVal0.Number.Uint64() == 14 {
			stacks[0].Server().AddPeer(enodes[1])
		}

		// reorg happens here, node1 has higher difficulty, it will replace blocks by node0

		if blockHeaderVal0.Number.Uint64() == 30 {

			break
		}

	}

	// check block 10 miner ; expected author is node1 signer
	blockHeaderVal0 := nodes[0].BlockChain().GetHeaderByNumber(10)
	blockHeaderVal1 := nodes[1].BlockChain().GetHeaderByNumber(10)
	authorVal0, err := nodes[0].Engine().Author(blockHeaderVal0)
	if err != nil {
		log.Error("Error in getting author", "err", err)
	}
	authorVal1, err := nodes[1].Engine().Author(blockHeaderVal1)
	if err != nil {
		log.Error("Error in getting author", "err", err)
	}

	// check both nodes have the same block 10
	assert.Equal(t, authorVal0, authorVal1)

	// check node0 has block mined by node1
	assert.Equal(t, authorVal0, nodes[1].AccountManager().Accounts()[0])

	// check node1 has block mined by node1
	assert.Equal(t, authorVal1, nodes[1].AccountManager().Accounts()[0])

	// check block 11 miner ; expected author is node1 signer
	blockHeaderVal0 = nodes[0].BlockChain().GetHeaderByNumber(11)
	blockHeaderVal1 = nodes[1].BlockChain().GetHeaderByNumber(11)
	authorVal0, err = nodes[0].Engine().Author(blockHeaderVal0)
	if err != nil {
		log.Error("Error in getting author", "err", err)
	}
	authorVal1, err = nodes[1].Engine().Author(blockHeaderVal1)
	if err != nil {
		log.Error("Error in getting author", "err", err)
	}

	// check both nodes have the same block 11
	assert.Equal(t, authorVal0, authorVal1)

	// check node0 has block mined by node1
	assert.Equal(t, authorVal0, nodes[1].AccountManager().Accounts()[0])

	// check node1 has block mined by node1
	assert.Equal(t, authorVal1, nodes[1].AccountManager().Accounts()[0])

	// check block 12 miner ; expected author is node1 signer
	blockHeaderVal0 = nodes[0].BlockChain().GetHeaderByNumber(12)
	blockHeaderVal1 = nodes[1].BlockChain().GetHeaderByNumber(12)
	authorVal0, err = nodes[0].Engine().Author(blockHeaderVal0)
	if err != nil {
		log.Error("Error in getting author", "err", err)
	}
	authorVal1, err = nodes[1].Engine().Author(blockHeaderVal1)
	if err != nil {
		log.Error("Error in getting author", "err", err)
	}

	// check both nodes have the same block 12
	assert.Equal(t, authorVal0, authorVal1)

	// check node0 has block mined by node1
	assert.Equal(t, authorVal0, nodes[1].AccountManager().Accounts()[0])

	// check node1 has block mined by node1
	assert.Equal(t, authorVal1, nodes[1].AccountManager().Accounts()[0])

	// check block 17 miner ; expected author is node0 signer
	blockHeaderVal0 = nodes[0].BlockChain().GetHeaderByNumber(17)
	blockHeaderVal1 = nodes[1].BlockChain().GetHeaderByNumber(17)
	authorVal0, err = nodes[0].Engine().Author(blockHeaderVal0)
	if err != nil {
		log.Error("Error in getting author", "err", err)
	}
	authorVal1, err = nodes[1].Engine().Author(blockHeaderVal1)
	if err != nil {
		log.Error("Error in getting author", "err", err)
	}

	// check both nodes have the same block 17
	assert.Equal(t, authorVal0, authorVal1)

	// check node0 has block mined by node1
	assert.Equal(t, authorVal0, nodes[0].AccountManager().Accounts()[0])

	// check node1 has block mined by node1
	assert.Equal(t, authorVal1, nodes[0].AccountManager().Accounts()[0])
}

func TestForkWithBlockTime(t *testing.T) {

	cases := []struct {
		name          string
		sprint        map[string]uint64
		blockTime     map[string]uint64
		change        uint64
		producerDelay map[string]uint64
		forkExpected  bool
	}{
		{
			name: "No fork after 2 sprints with producer delay = max block time",
			sprint: map[string]uint64{
				"0": 128,
			},
			blockTime: map[string]uint64{
				"0":   5,
				"128": 2,
				"256": 8,
			},
			change: 2,
			producerDelay: map[string]uint64{
				"0": 8,
			},
			forkExpected: false,
		},
		{
			name: "No Fork after 1 sprint producer delay = max block time",
			sprint: map[string]uint64{
				"0": 64,
			},
			blockTime: map[string]uint64{
				"0":  5,
				"64": 2,
			},
			change: 1,
			producerDelay: map[string]uint64{
				"0": 5,
			},
			forkExpected: false,
		},
		{
			name: "Fork after 4 sprints with producer delay < max block time",
			sprint: map[string]uint64{
				"0": 16,
			},
			blockTime: map[string]uint64{
				"0":  2,
				"64": 5,
			},
			change: 4,
			producerDelay: map[string]uint64{
				"0": 4,
			},
			forkExpected: true,
		},
	}

	// Create an Ethash network based off of the Ropsten config
	// Generate a batch of accounts to seal and fund with
	faucets := make([]*ecdsa.PrivateKey, 128)
	for i := 0; i < len(faucets); i++ {
		faucets[i], _ = crypto.GenerateKey()
	}
	genesis := InitGenesis(t, faucets, "./testdata/genesis_2val.json", 8)

	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {

			genesis.Config.Bor.Sprint = test.sprint
			genesis.Config.Bor.Period = test.blockTime
			genesis.Config.Bor.BackupMultiplier = test.blockTime
			genesis.Config.Bor.ProducerDelay = test.producerDelay

			stacks, nodes, _ := setupMiner(t, 2, genesis)

			defer func() {
				for _, stack := range stacks {
					stack.Close()
				}
			}()

			// Iterate over all the nodes and start mining
			for _, node := range nodes {
				if err := node.StartMining(); err != nil {
					t.Fatal("Error occurred while starting miner", "node", node, "error", err)
				}
			}
			var wg sync.WaitGroup
			blockHeaders := make([]*types.Header, 2)
			ticker := time.NewTicker(time.Duration(test.blockTime["0"]) * time.Second)
			defer ticker.Stop()

			for i := 0; i < 2; i++ {
				wg.Add(1)

				go func(i int) {
					defer wg.Done()

					for range ticker.C {
						blockHeaders[i] = nodes[i].BlockChain().GetHeaderByNumber(test.sprint["0"]*test.change + 10)
						if blockHeaders[i] != nil {
							break
						}
					}

				}(i)
			}

			wg.Wait()

			// Before the end of sprint
			blockHeaderVal0 := nodes[0].BlockChain().GetHeaderByNumber(test.sprint["0"] - 1)
			blockHeaderVal1 := nodes[1].BlockChain().GetHeaderByNumber(test.sprint["0"] - 1)
			assert.Equal(t, blockHeaderVal0.Hash(), blockHeaderVal1.Hash())
			assert.Equal(t, blockHeaderVal0.Time, blockHeaderVal1.Time)

			author0, err := nodes[0].Engine().Author(blockHeaderVal0)
			if err != nil {
				t.Error("Error occurred while fetching author", "err", err)
			}
			author1, err := nodes[1].Engine().Author(blockHeaderVal1)
			if err != nil {
				t.Error("Error occurred while fetching author", "err", err)
			}
			assert.Equal(t, author0, author1)

			// After the end of sprint
			author2, err := nodes[0].Engine().Author(blockHeaders[0])
			if err != nil {
				t.Error("Error occurred while fetching author", "err", err)
			}
			author3, err := nodes[1].Engine().Author(blockHeaders[1])
			if err != nil {
				t.Error("Error occurred while fetching author", "err", err)
			}

			if test.forkExpected {
				assert.NotEqual(t, blockHeaders[0].Hash(), blockHeaders[1].Hash())
				assert.NotEqual(t, blockHeaders[0].Time, blockHeaders[1].Time)
				assert.NotEqual(t, author2, author3)
			} else {
				assert.Equal(t, blockHeaders[0].Hash(), blockHeaders[1].Hash())
				assert.Equal(t, blockHeaders[0].Time, blockHeaders[1].Time)
				assert.Equal(t, author2, author3)
			}
		})

	}

}

func TestInsertingSpanSizeBlocks(t *testing.T) {
	init := buildEthereumInstance(t, rawdb.NewMemoryDatabase())
	chain := init.ethereum.BlockChain()
	engine := init.ethereum.Engine()
	_bor := engine.(*bor.Bor)

	_, currentSpan := loadSpanFromFile(t)

	h, ctrl := getMockedHeimdallClient(t, currentSpan)
	defer func() {
		_bor.Close()
		ctrl.Finish()
	}()

	h.EXPECT().Close().AnyTimes()

	h.EXPECT().FetchCheckpoint(gomock.Any(), int64(-1)).Return(&checkpoint.Checkpoint{
		Proposer:   currentSpan.SelectedProducers[0].Address,
		StartBlock: big.NewInt(0),
		EndBlock:   big.NewInt(int64(spanSize)),
	}, nil).AnyTimes()

	h.EXPECT().FetchMilestone(gomock.Any()).Return(&milestone.Milestone{
		Proposer:   currentSpan.SelectedProducers[0].Address,
		StartBlock: big.NewInt(0),
		EndBlock:   big.NewInt(int64(spanSize)),
	}, nil).AnyTimes()

	h.EXPECT().FetchLastNoAckMilestone(gomock.Any()).Return("", nil).AnyTimes()

	h.EXPECT().FetchNoAckMilestone(gomock.Any(), string("test")).Return(nil).AnyTimes()

	_bor.SetHeimdallClient(h)

	block := init.genesis.ToBlock()
	// to := int64(block.Header().Time)

	currentValidators := []*valset.Validator{valset.NewValidator(addr, 10)}

	spanner := getMockedSpanner(t, currentValidators)
	_bor.SetSpanner(spanner)

	// Insert sprintSize # of blocks so that span is fetched at the start of a new sprint
	for i := uint64(1); i <= spanSize; i++ {
		block = buildNextBlock(t, _bor, chain, block, nil, init.genesis.Config.Bor, nil, currentValidators)
		insertNewBlock(t, chain, block)
	}

	spanner = getMockedSpanner(t, currentSpan.ValidatorSet.Validators)
	_bor.SetSpanner(spanner)

	validators, err := _bor.GetCurrentValidators(context.Background(), block.Hash(), spanSize) // check validator set at the first block of new span
	if err != nil {
		t.Fatalf("%s", err)
	}

	require.Equal(t, 3, len(validators))
	for i, validator := range validators {
		require.Equal(t, validator.Address.Bytes(), currentSpan.SelectedProducers[i].Address.Bytes())
		require.Equal(t, validator.VotingPower, currentSpan.SelectedProducers[i].VotingPower)
	}
}

func TestFetchStateSyncEvents(t *testing.T) {
	init := buildEthereumInstance(t, rawdb.NewMemoryDatabase())
	chain := init.ethereum.BlockChain()
	engine := init.ethereum.Engine()
	_bor := engine.(*bor.Bor)

	defer _bor.Close()

	// A. Insert blocks for 0th sprint
	block := init.genesis.ToBlock()

	// B.1 Mock /bor/span/1
	res, _ := loadSpanFromFile(t)

	currentValidators := []*valset.Validator{valset.NewValidator(addr, 10)}

	spanner := getMockedSpanner(t, currentValidators)
	_bor.SetSpanner(spanner)

	// Insert sprintSize # of blocks so that span is fetched at the start of a new sprint
	for i := uint64(1); i < sprintSize; i++ {
		if IsSpanEnd(i) {
			currentValidators = res.Result.ValidatorSet.Validators
		}

		block = buildNextBlock(t, _bor, chain, block, nil, init.genesis.Config.Bor, nil, currentValidators)
		insertNewBlock(t, chain, block)
	}

	// B. Before inserting 1st block of the next sprint, mock heimdall deps
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	h := mocks.NewMockIHeimdallClient(ctrl)
	h.EXPECT().Close().AnyTimes()

	h.EXPECT().Span(gomock.Any(), uint64(1)).Return(&res.Result, nil).AnyTimes()

	h.EXPECT().FetchCheckpoint(gomock.Any(), int64(-1)).Return(&checkpoint.Checkpoint{}, nil).AnyTimes()

	h.EXPECT().FetchMilestone(gomock.Any()).Return(&milestone.Milestone{}, nil).AnyTimes()

	h.EXPECT().FetchLastNoAckMilestone(gomock.Any()).Return("", nil).AnyTimes()

	h.EXPECT().FetchNoAckMilestone(gomock.Any(), string("test")).Return(nil).AnyTimes()

	// B.2 Mock State Sync events
	fromID := uint64(1)
	// at # sprintSize, events are fetched for [fromID, (block-sprint).Time)
	to := int64(chain.GetHeaderByNumber(0).Time)
	eventCount := 50

	sample := getSampleEventRecord(t)
	sample.Time = time.Unix(to-int64(eventCount+1), 0) // last event.Time will be just < to
	eventRecords := generateFakeStateSyncEvents(sample, eventCount)

	h.EXPECT().StateSyncEvents(gomock.Any(), fromID, to).Return(eventRecords, nil).AnyTimes()
	_bor.SetHeimdallClient(h)

	block = buildNextBlock(t, _bor, chain, block, nil, init.genesis.Config.Bor, nil, res.Result.ValidatorSet.Validators)

	// Validate the state sync transactions set by consensus
	validateStateSyncEvents(t, eventRecords, chain.GetStateSync())

	insertNewBlock(t, chain, block)
}

func validateStateSyncEvents(t *testing.T, expected []*clerk.EventRecordWithTime, got []*types.StateSyncData) {
	require.Equal(t, len(expected), len(got), "number of state sync events should be equal")

	for i := 0; i < len(expected); i++ {
		require.Equal(t, expected[i].ID, got[i].ID, fmt.Sprintf("state sync ids should be equal - index: %d, expected: %d, got: %d", i, expected[i].ID, got[i].ID))
	}
}

func TestFetchStateSyncEvents_2(t *testing.T) {
	init := buildEthereumInstance(t, rawdb.NewMemoryDatabase())
	chain := init.ethereum.BlockChain()
	engine := init.ethereum.Engine()
	_bor := engine.(*bor.Bor)

	defer _bor.Close()

	// Mock /bor/span/1
	res, _ := loadSpanFromFile(t)

	// add the block producer
	res.Result.ValidatorSet.Validators = append(res.Result.ValidatorSet.Validators, valset.NewValidator(addr, 4500))

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	h := mocks.NewMockIHeimdallClient(ctrl)
	h.EXPECT().Close().AnyTimes()

	h.EXPECT().Span(gomock.Any(), uint64(1)).Return(&res.Result, nil).AnyTimes()

	h.EXPECT().FetchCheckpoint(gomock.Any(), int64(-1)).Return(&checkpoint.Checkpoint{}, nil).AnyTimes()

	h.EXPECT().FetchMilestone(gomock.Any()).Return(&milestone.Milestone{}, nil).AnyTimes()

	h.EXPECT().FetchLastNoAckMilestone(gomock.Any()).Return("", nil).AnyTimes()

	h.EXPECT().FetchNoAckMilestone(gomock.Any(), string("test")).Return(nil).AnyTimes()

	// Mock State Sync events
	// at # sprintSize, events are fetched for [fromID, (block-sprint).Time)
	fromID := uint64(1)
	to := int64(chain.GetHeaderByNumber(0).Time)
	sample := getSampleEventRecord(t)

	// First query will be from [id=1, (block-sprint).Time]
	// Insert 5 events in this time range
	eventRecords := []*clerk.EventRecordWithTime{
		buildStateEvent(sample, 1, 3), // id = 1, time = 1
		buildStateEvent(sample, 2, 1), // id = 2, time = 3
		buildStateEvent(sample, 3, 2), // id = 3, time = 2
		// event with id 5 is missing
		buildStateEvent(sample, 4, 5), // id = 4, time = 5
		buildStateEvent(sample, 6, 4), // id = 6, time = 4
	}

	h.EXPECT().StateSyncEvents(gomock.Any(), fromID, to).Return(eventRecords, nil).AnyTimes()
	_bor.SetHeimdallClient(h)

	// Insert blocks for 0th sprint
	block := init.genesis.ToBlock()

	var currentValidators []*valset.Validator

	for i := uint64(1); i <= sprintSize; i++ {
		if IsSpanEnd(i) {
			currentValidators = res.Result.ValidatorSet.Validators
		} else {
			currentValidators = []*valset.Validator{valset.NewValidator(addr, 10)}
		}

		spanner := getMockedSpanner(t, currentValidators)
		_bor.SetSpanner(spanner)

		block = buildNextBlock(t, _bor, chain, block, nil, init.genesis.Config.Bor, nil, currentValidators)
		insertNewBlock(t, chain, block)
	}

	lastStateID, _ := _bor.GenesisContractsClient.LastStateId(nil, sprintSize, block.Hash())

	// state 6 was not written
	require.Equal(t, uint64(4), lastStateID.Uint64())

	//
	fromID = uint64(5)
	to = int64(chain.GetHeaderByNumber(sprintSize).Time)

	eventRecords = []*clerk.EventRecordWithTime{
		buildStateEvent(sample, 5, 7),
		buildStateEvent(sample, 6, 4),
	}
	h.EXPECT().StateSyncEvents(gomock.Any(), fromID, to).Return(eventRecords, nil).AnyTimes()

	for i := sprintSize + 1; i <= spanSize; i++ {
		if IsSpanEnd(i) {
			currentValidators = res.Result.ValidatorSet.Validators
		} else {
			currentValidators = []*valset.Validator{valset.NewValidator(addr, 10)}
		}

		spanner := getMockedSpanner(t, currentValidators)
		_bor.SetSpanner(spanner)

		block = buildNextBlock(t, _bor, chain, block, nil, init.genesis.Config.Bor, nil, res.Result.ValidatorSet.Validators)
		insertNewBlock(t, chain, block)
	}

	lastStateID, _ = _bor.GenesisContractsClient.LastStateId(nil, spanSize, block.Hash())
	require.Equal(t, uint64(6), lastStateID.Uint64())
}

func TestOutOfTurnSigning(t *testing.T) {
	init := buildEthereumInstance(t, rawdb.NewMemoryDatabase())
	chain := init.ethereum.BlockChain()
	engine := init.ethereum.Engine()
	_bor := engine.(*bor.Bor)

	defer _bor.Close()

	_, heimdallSpan := loadSpanFromFile(t)
	proposer := valset.NewValidator(addr, 10)
	heimdallSpan.ValidatorSet.Validators = append(heimdallSpan.ValidatorSet.Validators, proposer)

	// add the block producer
	h, ctrl := getMockedHeimdallClient(t, heimdallSpan)
	defer ctrl.Finish()

	h.EXPECT().Close().AnyTimes()

	h.EXPECT().FetchCheckpoint(gomock.Any(), int64(-1)).Return(&checkpoint.Checkpoint{}, nil).AnyTimes()

	h.EXPECT().FetchMilestone(gomock.Any()).Return(&milestone.Milestone{}, nil).AnyTimes()

	h.EXPECT().FetchLastNoAckMilestone(gomock.Any()).Return("", nil).AnyTimes()

	h.EXPECT().FetchNoAckMilestone(gomock.Any(), string("test")).Return(nil).AnyTimes()

	spanner := getMockedSpanner(t, heimdallSpan.ValidatorSet.Validators)
	_bor.SetSpanner(spanner)

	_bor.SetHeimdallClient(h)

	block := init.genesis.ToBlock()

	setDifficulty := func(header *types.Header) {
		if IsSprintStart(header.Number.Uint64()) {
			header.Difficulty = big.NewInt(int64(len(heimdallSpan.ValidatorSet.Validators)))
		}
	}

	for i := uint64(1); i < spanSize; i++ {
		block = buildNextBlock(t, _bor, chain, block, nil, init.genesis.Config.Bor, nil, heimdallSpan.ValidatorSet.Validators, setDifficulty)
		insertNewBlock(t, chain, block)
	}

	// insert spanSize-th block
	// This account is one the out-of-turn validators for 1st (0-indexed) span
	signer := "c8deb0bea5c41afe8e37b4d1bd84e31adff11b09c8c96ff4b605003cce067cd9"
	signerKey, _ := hex.DecodeString(signer)
	newKey, _ := crypto.HexToECDSA(signer)
	newAddr := crypto.PubkeyToAddress(newKey.PublicKey)
	expectedSuccessionNumber := 2

	parentTime := block.Time()

	setParentTime := func(header *types.Header) {
		header.Time = parentTime + 1
	}

	const turn = 1

	setDifficulty = func(header *types.Header) {
		header.Difficulty = big.NewInt(int64(len(heimdallSpan.ValidatorSet.Validators)) - turn)
	}

	block = buildNextBlock(t, _bor, chain, block, signerKey, init.genesis.Config.Bor, nil, heimdallSpan.ValidatorSet.Validators, setParentTime, setDifficulty)
	_, err := chain.InsertChain([]*types.Block{block})
	require.Equal(t,
		bor.BlockTooSoonError{Number: spanSize, Succession: expectedSuccessionNumber},
		*err.(*bor.BlockTooSoonError))

	expectedDifficulty := uint64(len(heimdallSpan.ValidatorSet.Validators) - expectedSuccessionNumber - turn) // len(validators) - succession
	header := block.Header()

	diff := bor.CalcProducerDelay(header.Number.Uint64(), expectedSuccessionNumber, init.genesis.Config.Bor)
	header.Time += diff

	sign(t, header, signerKey, init.genesis.Config.Bor)

	block = types.NewBlockWithHeader(header)

	_, err = chain.InsertChain([]*types.Block{block})
	require.NotNil(t, err)
	require.Equal(t,
		bor.WrongDifficultyError{Number: spanSize, Expected: expectedDifficulty, Actual: 3, Signer: newAddr.Bytes()},
		*err.(*bor.WrongDifficultyError))

	header.Difficulty = new(big.Int).SetUint64(expectedDifficulty)
	sign(t, header, signerKey, init.genesis.Config.Bor)
	block = types.NewBlockWithHeader(header)

	_, err = chain.InsertChain([]*types.Block{block})
	require.Nil(t, err)
}

func TestSignerNotFound(t *testing.T) {
	init := buildEthereumInstance(t, rawdb.NewMemoryDatabase())
	chain := init.ethereum.BlockChain()
	engine := init.ethereum.Engine()
	_bor := engine.(*bor.Bor)

	defer _bor.Close()

	_, heimdallSpan := loadSpanFromFile(t)

	h, ctrl := getMockedHeimdallClient(t, heimdallSpan)
	defer ctrl.Finish()

	h.EXPECT().Close().AnyTimes()
	h.EXPECT().FetchCheckpoint(gomock.Any(), int64(-1)).Return(&checkpoint.Checkpoint{}, nil).AnyTimes()

	h.EXPECT().FetchMilestone(gomock.Any()).Return(&milestone.Milestone{}, nil).AnyTimes()

	h.EXPECT().FetchLastNoAckMilestone(gomock.Any()).Return("", nil).AnyTimes()

	h.EXPECT().FetchNoAckMilestone(gomock.Any(), string("test")).Return(nil).AnyTimes()

	_bor.SetHeimdallClient(h)

	block := init.genesis.ToBlock()

	// random signer account that is not a part of the validator set
	const signer = "3714d99058cd64541433d59c6b391555b2fd9b54629c2b717a6c9c00d1127b6b"
	signerKey, _ := hex.DecodeString(signer)
	newKey, _ := crypto.HexToECDSA(signer)
	newAddr := crypto.PubkeyToAddress(newKey.PublicKey)

	_bor.Authorize(newAddr, func(account accounts.Account, s string, data []byte) ([]byte, error) {
		return crypto.Sign(crypto.Keccak256(data), newKey)
	})

	block = buildNextBlock(t, _bor, chain, block, signerKey, init.genesis.Config.Bor, nil, heimdallSpan.ValidatorSet.Validators)

	_, err := chain.InsertChain([]*types.Block{block})
	require.Equal(t,
		*err.(*bor.UnauthorizedSignerError),
		bor.UnauthorizedSignerError{Number: 0, Signer: newAddr.Bytes()})
}

// TestEIP1559Transition tests the following:
//
//  1. A transaction whose gasFeeCap is greater than the baseFee is valid.
//  2. Gas accounting for access lists on EIP-1559 transactions is correct.
//  3. Only the transaction's tip will be received by the coinbase.
//  4. The transaction sender pays for both the tip and baseFee.
//  5. The coinbase receives only the partially realized tip when
//     gasFeeCap - gasTipCap < baseFee.
//  6. Legacy transaction behave as expected (e.g. gasPrice = gasFeeCap = gasTipCap).
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
			Config: params.BorUnittestChainConfig,
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
	genesis := gspec.MustCommit(db, trie.NewDatabase(db, trie.HashDefaults))
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
	gspec.MustCommit(diskdb, trie.NewDatabase(diskdb, trie.HashDefaults))

	chain, err := core.NewBlockChain(diskdb, nil, gspec, nil, engine, vm.Config{}, nil, nil, nil)
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
	actual = state.GetBalance(common.HexToAddress(params.BorUnittestChainConfig.Bor.CalculateBurntContract(block.NumberU64())))
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
	actual = state.GetBalance(common.HexToAddress(params.BorUnittestChainConfig.Bor.CalculateBurntContract(block.NumberU64())))
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
	actual = state.GetBalance(common.HexToAddress(params.BorUnittestChainConfig.Bor.CalculateBurntContract(block.NumberU64())))
	burntAmount := new(big.Int).Mul(
		block.BaseFee(),
		big.NewInt(int64(block.GasUsed())),
	)
	expected = new(big.Int).Add(burntContractBalance, burntAmount)
	if actual.Cmp(expected) != 0 {
		t.Fatalf("burnt contract balance incorrect: expected %d, got %d", expected, actual)
	}
}

func TestBurnContract(t *testing.T) {
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
			Config: params.BorUnittestChainConfig,
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
	gspec.Config.Bor.BurntContract = map[string]string{
		"0": "0x000000000000000000000000000000000000aaab",
		"1": "0x000000000000000000000000000000000000aaac",
		"2": "0x000000000000000000000000000000000000aaad",
		"3": "0x000000000000000000000000000000000000aaae",
	}
	genesis := gspec.MustCommit(db, trie.NewDatabase(db, trie.HashDefaults))
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
	gspec.MustCommit(diskdb, trie.NewDatabase(diskdb, trie.HashDefaults))

	chain, err := core.NewBlockChain(diskdb, nil, gspec, nil, engine, vm.Config{}, nil, nil, nil)
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
	actual = state.GetBalance(common.HexToAddress(gspec.Config.Bor.CalculateBurntContract(block.NumberU64())))
	expected = new(big.Int).Mul(new(big.Int).SetUint64(block.GasUsed()), block.BaseFee())
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
	actual = state.GetBalance(common.HexToAddress(gspec.Config.Bor.CalculateBurntContract(block.NumberU64())))
	expected = new(big.Int).Mul(new(big.Int).SetUint64(block.GasUsed()), block.BaseFee())
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
	})

	if n, err := chain.InsertChain(blocks); err != nil {
		t.Fatalf("block %d: failed to insert into chain: %v", n, err)
	}

	block = chain.GetBlockByNumber(3)
	state, _ = chain.State()
	effectiveTip = block.Transactions()[0].GasTipCap().Uint64() - block.BaseFee().Uint64()

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
	actual = state.GetBalance(common.HexToAddress(gspec.Config.Bor.CalculateBurntContract(block.NumberU64())))
	expected = new(big.Int).Mul(new(big.Int).SetUint64(block.GasUsed()), block.BaseFee())
	if actual.Cmp(expected) != 0 {
		t.Fatalf("burnt contract balance incorrect: expected %d, got %d", expected, actual)
	}

	// 4: Ensure the tx sender paid for the gasUsed * (effectiveTip + block baseFee).
	actual = new(big.Int).Sub(funds, state.GetBalance(addr3))
	expected = new(big.Int).SetUint64(block.GasUsed() * (effectiveTip + block.BaseFee().Uint64()))
	if actual.Cmp(expected) != 0 {
		t.Fatalf("sender balance incorrect: expected %d, got %d", expected, actual)
	}
}

func TestBurnContractContractFetch(t *testing.T) {
	config := params.BorUnittestChainConfig
	config.Bor.BurntContract = map[string]string{
		"10":  "0x000000000000000000000000000000000000aaab",
		"100": "0x000000000000000000000000000000000000aaad",
	}

	burnContractAddr10 := config.Bor.CalculateBurntContract(10)
	burnContractAddr11 := config.Bor.CalculateBurntContract(11)
	burnContractAddr99 := config.Bor.CalculateBurntContract(99)
	burnContractAddr100 := config.Bor.CalculateBurntContract(100)
	burnContractAddr101 := config.Bor.CalculateBurntContract(101)

	if burnContractAddr10 != "0x000000000000000000000000000000000000aaab" {
		t.Fatalf("incorrect burnt contract address: expected %s, got %s", "0x000000000000000000000000000000000000aaab", burnContractAddr10)
	}
	if burnContractAddr11 != "0x000000000000000000000000000000000000aaab" {
		t.Fatalf("incorrect burnt contract address: expected %s, got %s", "0x000000000000000000000000000000000000aaab", burnContractAddr11)
	}
	if burnContractAddr99 != "0x000000000000000000000000000000000000aaab" {
		t.Fatalf("incorrect burnt contract address: expected %s, got %s", "0x000000000000000000000000000000000000aaab", burnContractAddr99)
	}
	if burnContractAddr100 != "0x000000000000000000000000000000000000aaad" {
		t.Fatalf("incorrect burnt contract address: expected %s, got %s", "0x000000000000000000000000000000000000aaad", burnContractAddr100)
	}
	if burnContractAddr101 != "0x000000000000000000000000000000000000aaad" {
		t.Fatalf("incorrect burnt contract address: expected %s, got %s", "0x000000000000000000000000000000000000aaad", burnContractAddr101)
	}

	config.Bor.BurntContract = map[string]string{
		"10":   "0x000000000000000000000000000000000000aaab",
		"100":  "0x000000000000000000000000000000000000aaad",
		"1000": "0x000000000000000000000000000000000000aaae",
	}

	burnContractAddr10 = config.Bor.CalculateBurntContract(10)
	burnContractAddr11 = config.Bor.CalculateBurntContract(11)
	burnContractAddr99 = config.Bor.CalculateBurntContract(99)
	burnContractAddr100 = config.Bor.CalculateBurntContract(100)
	burnContractAddr101 = config.Bor.CalculateBurntContract(101)
	burnContractAddr999 := config.Bor.CalculateBurntContract(999)
	burnContractAddr1000 := config.Bor.CalculateBurntContract(1000)
	burnContractAddr1001 := config.Bor.CalculateBurntContract(1001)

	if burnContractAddr10 != "0x000000000000000000000000000000000000aaab" {
		t.Fatalf("incorrect burnt contract address: expected %s, got %s", "0x000000000000000000000000000000000000aaab", burnContractAddr10)
	}
	if burnContractAddr11 != "0x000000000000000000000000000000000000aaab" {
		t.Fatalf("incorrect burnt contract address: expected %s, got %s", "0x000000000000000000000000000000000000aaab", burnContractAddr11)
	}
	if burnContractAddr99 != "0x000000000000000000000000000000000000aaab" {
		t.Fatalf("incorrect burnt contract address: expected %s, got %s", "0x000000000000000000000000000000000000aaab", burnContractAddr99)
	}
	if burnContractAddr100 != "0x000000000000000000000000000000000000aaad" {
		t.Fatalf("incorrect burnt contract address: expected %s, got %s", "0x000000000000000000000000000000000000aaad", burnContractAddr100)
	}
	if burnContractAddr101 != "0x000000000000000000000000000000000000aaad" {
		t.Fatalf("incorrect burnt contract address: expected %s, got %s", "0x000000000000000000000000000000000000aaad", burnContractAddr101)
	}
	if burnContractAddr999 != "0x000000000000000000000000000000000000aaad" {
		t.Fatalf("incorrect burnt contract address: expected %s, got %s", "0x000000000000000000000000000000000000aaad", burnContractAddr999)
	}
	if burnContractAddr1000 != "0x000000000000000000000000000000000000aaae" {
		t.Fatalf("incorrect burnt contract address: expected %s, got %s", "0x000000000000000000000000000000000000aaae", burnContractAddr1000)
	}
	if burnContractAddr1001 != "0x000000000000000000000000000000000000aaae" {
		t.Fatalf("incorrect burnt contract address: expected %s, got %s", "0x000000000000000000000000000000000000aaae", burnContractAddr1001)
	}
}

// EIP1559 is not supported without EIP155. An error is expected
func TestEIP1559TransitionWithEIP155(t *testing.T) {
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
			Config: params.BorUnittestChainConfig,
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

	genesis := gspec.MustCommit(db, trie.NewDatabase(db, trie.HashDefaults))

	// Use signer without chain ID
	signer := types.HomesteadSigner{}

	_, _ = core.GenerateChain(gspec.Config, genesis, engine, db, 1, func(i int, b *core.BlockGen) {
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

		var err error

		tx := types.NewTx(txdata)
		tx, err = types.SignTx(tx, signer, key1)

		require.ErrorIs(t, err, types.ErrTxTypeNotSupported)
	})
}

// it is up to a user to use protected transactions. so if a transaction is unprotected no errors related to chainID are expected.
// transactions are checked in 2 places: transaction pool and blockchain processor.
func TestTransitionWithoutEIP155(t *testing.T) {
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
			Config: params.BorUnittestChainConfig,
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

	genesis := gspec.MustCommit(db, trie.NewDatabase(db, trie.HashDefaults))

	// Use signer without chain ID
	signer := types.HomesteadSigner{}
	//signer := types.FrontierSigner{}

	blocks, _ := core.GenerateChain(gspec.Config, genesis, engine, db, 1, func(i int, b *core.BlockGen) {
		b.SetCoinbase(common.Address{1})

		txdata := &types.LegacyTx{
			Nonce:    0,
			To:       &aa,
			Gas:      30000,
			GasPrice: newGwei(5),
		}

		var err error

		tx := types.NewTx(txdata)
		tx, err = types.SignTx(tx, signer, key1)

		require.Nil(t, err)
		require.False(t, tx.Protected())

		from, err := types.Sender(types.EIP155Signer{}, tx)
		require.Equal(t, addr1, from)
		require.Nil(t, err)

		b.AddTx(tx)
	})

	diskdb := rawdb.NewMemoryDatabase()
	gspec.MustCommit(diskdb, trie.NewDatabase(diskdb, trie.HashDefaults))

	chain, err := core.NewBlockChain(diskdb, nil, gspec, nil, engine, vm.Config{}, nil, nil, nil)
	if err != nil {
		t.Fatalf("failed to create tester chain: %v", err)
	}
	if n, err := chain.InsertChain(blocks); err != nil {
		t.Fatalf("block %d: failed to insert into chain: %v", n, err)
	}

	block := chain.GetBlockByNumber(1)

	require.Len(t, block.Transactions(), 1)
}

func TestJaipurFork(t *testing.T) {
	init := buildEthereumInstance(t, rawdb.NewMemoryDatabase())
	chain := init.ethereum.BlockChain()
	engine := init.ethereum.Engine()

	_bor := engine.(*bor.Bor)
	defer _bor.Close()

	block := init.genesis.ToBlock()

	res, _ := loadSpanFromFile(t)

	spanner := getMockedSpanner(t, res.Result.ValidatorSet.Validators)
	_bor.SetSpanner(spanner)

	for i := uint64(1); i < sprintSize; i++ {
		block = buildNextBlock(t, _bor, chain, block, nil, init.genesis.Config.Bor, nil, res.Result.ValidatorSet.Validators)
		insertNewBlock(t, chain, block)

		if block.Number().Uint64() == init.genesis.Config.Bor.JaipurBlock.Uint64()-1 {
			require.Equal(t, testSealHash(block.Header(), init.genesis.Config.Bor), bor.SealHash(block.Header(), init.genesis.Config.Bor))
		}

		if block.Number().Uint64() == init.genesis.Config.Bor.JaipurBlock.Uint64() {
			require.Equal(t, testSealHash(block.Header(), init.genesis.Config.Bor), bor.SealHash(block.Header(), init.genesis.Config.Bor))
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
	if c.IsJaipur(header.Number) {
		if header.BaseFee != nil {
			enc = append(enc, header.BaseFee)
		}
	}
	if err := rlp.Encode(w, enc); err != nil {
		panic("can't encode: " + err.Error())
	}
}
