// Copyright 2021 The go-ethereum Authors
// This file is part of the go-ethereum library.
//
// The go-ethereum library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The go-ethereum library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the go-ethereum library. If not, see <http://www.gnu.org/licenses/>.

package catalyst

import (
	"bytes"
	"fmt"
	"math/big"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/consensus/ethash"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/beacon"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/eth"
	"github.com/ethereum/go-ethereum/eth/downloader"
	"github.com/ethereum/go-ethereum/eth/ethconfig"
	"github.com/ethereum/go-ethereum/node"
	"github.com/ethereum/go-ethereum/p2p"
	"github.com/ethereum/go-ethereum/params"
	"github.com/ethereum/go-ethereum/trie"
)

var (
	// testKey is a private key to use for funding a tester account.
	testKey, _ = crypto.HexToECDSA("b71c71a67e1177ad4e901695e1b4b9ee17ae16c6668d313eac2f96dbcda3f291")

	// testAddr is the Ethereum address of the tester account.
	testAddr = crypto.PubkeyToAddress(testKey.PublicKey)

	testBalance = big.NewInt(2e18)
)

func generatePreMergeChain(n int) (*core.Genesis, []*types.Block) {
	config := *params.AllEthashProtocolChanges
	genesis := &core.Genesis{
		Config:     &config,
		Alloc:      core.GenesisAlloc{testAddr: {Balance: testBalance}},
		ExtraData:  []byte("test genesis"),
		Timestamp:  9000,
		BaseFee:    big.NewInt(params.InitialBaseFee),
		Difficulty: big.NewInt(0),
	}
	testNonce := uint64(0)
	generate := func(i int, g *core.BlockGen) {
		g.OffsetTime(5)
		g.SetExtra([]byte("test"))
		tx, _ := types.SignTx(types.NewTransaction(testNonce, common.HexToAddress("0x9a9070028361F7AAbeB3f2F2Dc07F82C4a98A02a"), big.NewInt(1), params.TxGas, big.NewInt(params.InitialBaseFee*2), nil), types.LatestSigner(&config), testKey)
		g.AddTx(tx)
		testNonce++
	}
	_, blocks, _ := core.GenerateChainWithGenesis(genesis, ethash.NewFaker(), n, generate)
	totalDifficulty := big.NewInt(0)
	for _, b := range blocks {
		totalDifficulty.Add(totalDifficulty, b.Difficulty())
	}
	config.TerminalTotalDifficulty = totalDifficulty
	return genesis, blocks
}

func TestEth2AssembleBlock(t *testing.T) {
	genesis, blocks := generatePreMergeChain(10)
	n, ethservice := startEthService(t, genesis, blocks)
	defer n.Close()

	api := NewConsensusAPI(ethservice)
	signer := types.NewEIP155Signer(ethservice.BlockChain().Config().ChainID)
	tx, err := types.SignTx(types.NewTransaction(uint64(10), blocks[9].Coinbase(), big.NewInt(1000), params.TxGas, big.NewInt(params.InitialBaseFee), nil), signer, testKey)
	if err != nil {
		t.Fatalf("error signing transaction, err=%v", err)
	}
	ethservice.TxPool().AddLocal(tx)
	blockParams := beacon.PayloadAttributesV1{
		Timestamp: blocks[9].Time() + 5,
	}
	execData, err := assembleBlock(api, blocks[9].Hash(), &blockParams)
	if err != nil {
		t.Fatalf("error producing block, err=%v", err)
	}
	if len(execData.Transactions) != 1 {
		t.Fatalf("invalid number of transactions %d != 1", len(execData.Transactions))
	}
}

func TestEth2AssembleBlockWithAnotherBlocksTxs(t *testing.T) {
	genesis, blocks := generatePreMergeChain(10)
	n, ethservice := startEthService(t, genesis, blocks[:9])
	defer n.Close()

	api := NewConsensusAPI(ethservice)

	// Put the 10th block's tx in the pool and produce a new block
	api.eth.TxPool().AddRemotesSync(blocks[9].Transactions())
	blockParams := beacon.PayloadAttributesV1{
		Timestamp: blocks[8].Time() + 5,
	}
	execData, err := assembleBlock(api, blocks[8].Hash(), &blockParams)
	if err != nil {
		t.Fatalf("error producing block, err=%v", err)
	}
	if len(execData.Transactions) != blocks[9].Transactions().Len() {
		t.Fatalf("invalid number of transactions %d != 1", len(execData.Transactions))
	}
}

func TestSetHeadBeforeTotalDifficulty(t *testing.T) {
	genesis, blocks := generatePreMergeChain(10)
	n, ethservice := startEthService(t, genesis, blocks)
	defer n.Close()

	api := NewConsensusAPI(ethservice)
	fcState := beacon.ForkchoiceStateV1{
		HeadBlockHash:      blocks[5].Hash(),
		SafeBlockHash:      common.Hash{},
		FinalizedBlockHash: common.Hash{},
	}
	if resp, err := api.ForkchoiceUpdatedV1(fcState, nil); err != nil {
		t.Errorf("fork choice updated should not error: %v", err)
	} else if resp.PayloadStatus.Status != beacon.INVALID_TERMINAL_BLOCK.Status {
		t.Errorf("fork choice updated before total terminal difficulty should be INVALID")
	}
}

func TestEth2PrepareAndGetPayload(t *testing.T) {
	genesis, blocks := generatePreMergeChain(10)
	// We need to properly set the terminal total difficulty
	genesis.Config.TerminalTotalDifficulty.Sub(genesis.Config.TerminalTotalDifficulty, blocks[9].Difficulty())
	n, ethservice := startEthService(t, genesis, blocks[:9])
	defer n.Close()

	api := NewConsensusAPI(ethservice)

	// Put the 10th block's tx in the pool and produce a new block
	ethservice.TxPool().AddLocals(blocks[9].Transactions())
	blockParams := beacon.PayloadAttributesV1{
		Timestamp: blocks[8].Time() + 5,
	}
	fcState := beacon.ForkchoiceStateV1{
		HeadBlockHash:      blocks[8].Hash(),
		SafeBlockHash:      common.Hash{},
		FinalizedBlockHash: common.Hash{},
	}
	_, err := api.ForkchoiceUpdatedV1(fcState, &blockParams)
	if err != nil {
		t.Fatalf("error preparing payload, err=%v", err)
	}
	payloadID := computePayloadId(fcState.HeadBlockHash, &blockParams)
	execData, err := api.GetPayloadV1(payloadID)
	if err != nil {
		t.Fatalf("error getting payload, err=%v", err)
	}
	if len(execData.Transactions) != blocks[9].Transactions().Len() {
		t.Fatalf("invalid number of transactions %d != 1", len(execData.Transactions))
	}
	// Test invalid payloadID
	var invPayload beacon.PayloadID
	copy(invPayload[:], payloadID[:])
	invPayload[0] = ^invPayload[0]
	_, err = api.GetPayloadV1(invPayload)
	if err == nil {
		t.Fatal("expected error retrieving invalid payload")
	}
}

func checkLogEvents(t *testing.T, logsCh <-chan []*types.Log, rmLogsCh <-chan core.RemovedLogsEvent, wantNew, wantRemoved int) {
	t.Helper()

	if len(logsCh) != wantNew {
		t.Fatalf("wrong number of log events: got %d, want %d", len(logsCh), wantNew)
	}
	if len(rmLogsCh) != wantRemoved {
		t.Fatalf("wrong number of removed log events: got %d, want %d", len(rmLogsCh), wantRemoved)
	}
	// Drain events.
	for i := 0; i < len(logsCh); i++ {
		<-logsCh
	}
	for i := 0; i < len(rmLogsCh); i++ {
		<-rmLogsCh
	}
}

func TestInvalidPayloadTimestamp(t *testing.T) {
	genesis, preMergeBlocks := generatePreMergeChain(10)
	n, ethservice := startEthService(t, genesis, preMergeBlocks)
	defer n.Close()
	var (
		api    = NewConsensusAPI(ethservice)
		parent = ethservice.BlockChain().CurrentBlock()
	)
	tests := []struct {
		time      uint64
		shouldErr bool
	}{
		{0, true},
		{parent.Time(), true},
		{parent.Time() - 1, true},

		// TODO (MariusVanDerWijden) following tests are currently broken,
		// fixed in upcoming merge-kiln-v2 pr
		//{parent.Time() + 1, false},
		//{uint64(time.Now().Unix()) + uint64(time.Minute), false},
	}

	for i, test := range tests {
		t.Run(fmt.Sprintf("Timestamp test: %v", i), func(t *testing.T) {
			params := beacon.PayloadAttributesV1{
				Timestamp:             test.time,
				Random:                crypto.Keccak256Hash([]byte{byte(123)}),
				SuggestedFeeRecipient: parent.Coinbase(),
			}
			fcState := beacon.ForkchoiceStateV1{
				HeadBlockHash:      parent.Hash(),
				SafeBlockHash:      common.Hash{},
				FinalizedBlockHash: common.Hash{},
			}
			_, err := api.ForkchoiceUpdatedV1(fcState, &params)
			if test.shouldErr && err == nil {
				t.Fatalf("expected error preparing payload with invalid timestamp, err=%v", err)
			} else if !test.shouldErr && err != nil {
				t.Fatalf("error preparing payload with valid timestamp, err=%v", err)
			}
		})
	}
}

func TestEth2NewBlock(t *testing.T) {
	genesis, preMergeBlocks := generatePreMergeChain(10)
	n, ethservice := startEthService(t, genesis, preMergeBlocks)
	defer n.Close()

	var (
		api    = NewConsensusAPI(ethservice)
		parent = preMergeBlocks[len(preMergeBlocks)-1]

		// This EVM code generates a log when the contract is created.
		logCode = common.Hex2Bytes("60606040525b7f24ec1d3ff24c2f6ff210738839dbc339cd45a5294d85c79361016243157aae7b60405180905060405180910390a15b600a8060416000396000f360606040526008565b00")
	)
	// The event channels.
	newLogCh := make(chan []*types.Log, 10)
	rmLogsCh := make(chan core.RemovedLogsEvent, 10)
	ethservice.BlockChain().SubscribeLogsEvent(newLogCh)
	ethservice.BlockChain().SubscribeRemovedLogsEvent(rmLogsCh)

	for i := 0; i < 10; i++ {
		statedb, _ := ethservice.BlockChain().StateAt(parent.Root())
		nonce := statedb.GetNonce(testAddr)
		tx, _ := types.SignTx(types.NewContractCreation(nonce, new(big.Int), 1000000, big.NewInt(2*params.InitialBaseFee), logCode), types.LatestSigner(ethservice.BlockChain().Config()), testKey)
		ethservice.TxPool().AddLocal(tx)

		execData, err := assembleBlock(api, parent.Hash(), &beacon.PayloadAttributesV1{
			Timestamp: parent.Time() + 5,
		})
		if err != nil {
			t.Fatalf("Failed to create the executable data %v", err)
		}
		block, err := beacon.ExecutableDataToBlock(*execData)
		if err != nil {
			t.Fatalf("Failed to convert executable data to block %v", err)
		}
		newResp, err := api.NewPayloadV1(*execData)
		if err != nil || newResp.Status != "VALID" {
			t.Fatalf("Failed to insert block: %v", err)
		}
		if ethservice.BlockChain().CurrentBlock().NumberU64() != block.NumberU64()-1 {
			t.Fatalf("Chain head shouldn't be updated")
		}
		checkLogEvents(t, newLogCh, rmLogsCh, 0, 0)
		fcState := beacon.ForkchoiceStateV1{
			HeadBlockHash:      block.Hash(),
			SafeBlockHash:      block.Hash(),
			FinalizedBlockHash: block.Hash(),
		}
		if _, err := api.ForkchoiceUpdatedV1(fcState, nil); err != nil {
			t.Fatalf("Failed to insert block: %v", err)
		}
		if ethservice.BlockChain().CurrentBlock().NumberU64() != block.NumberU64() {
			t.Fatalf("Chain head should be updated")
		}
		checkLogEvents(t, newLogCh, rmLogsCh, 1, 0)

		parent = block
	}

	// Introduce fork chain
	var (
		head = ethservice.BlockChain().CurrentBlock().NumberU64()
	)
	parent = preMergeBlocks[len(preMergeBlocks)-1]
	for i := 0; i < 10; i++ {
		execData, err := assembleBlock(api, parent.Hash(), &beacon.PayloadAttributesV1{
			Timestamp: parent.Time() + 6,
		})
		if err != nil {
			t.Fatalf("Failed to create the executable data %v", err)
		}
		block, err := beacon.ExecutableDataToBlock(*execData)
		if err != nil {
			t.Fatalf("Failed to convert executable data to block %v", err)
		}
		newResp, err := api.NewPayloadV1(*execData)
		if err != nil || newResp.Status != "VALID" {
			t.Fatalf("Failed to insert block: %v", err)
		}
		if ethservice.BlockChain().CurrentBlock().NumberU64() != head {
			t.Fatalf("Chain head shouldn't be updated")
		}

		fcState := beacon.ForkchoiceStateV1{
			HeadBlockHash:      block.Hash(),
			SafeBlockHash:      block.Hash(),
			FinalizedBlockHash: block.Hash(),
		}
		if _, err := api.ForkchoiceUpdatedV1(fcState, nil); err != nil {
			t.Fatalf("Failed to insert block: %v", err)
		}
		if ethservice.BlockChain().CurrentBlock().NumberU64() != block.NumberU64() {
			t.Fatalf("Chain head should be updated")
		}
		parent, head = block, block.NumberU64()
	}
}

func TestEth2DeepReorg(t *testing.T) {
	// TODO (MariusVanDerWijden) TestEth2DeepReorg is currently broken, because it tries to reorg
	// before the totalTerminalDifficulty threshold
	/*
		genesis, preMergeBlocks := generatePreMergeChain(core.TriesInMemory * 2)
		n, ethservice := startEthService(t, genesis, preMergeBlocks)
		defer n.Close()

		var (
			api    = NewConsensusAPI(ethservice, nil)
			parent = preMergeBlocks[len(preMergeBlocks)-core.TriesInMemory-1]
			head   = ethservice.BlockChain().CurrentBlock().NumberU64()
		)
		if ethservice.BlockChain().HasBlockAndState(parent.Hash(), parent.NumberU64()) {
			t.Errorf("Block %d not pruned", parent.NumberU64())
		}
		for i := 0; i < 10; i++ {
			execData, err := api.assembleBlock(AssembleBlockParams{
				ParentHash: parent.Hash(),
				Timestamp:  parent.Time() + 5,
			})
			if err != nil {
				t.Fatalf("Failed to create the executable data %v", err)
			}
			block, err := ExecutableDataToBlock(ethservice.BlockChain().Config(), parent.Header(), *execData)
			if err != nil {
				t.Fatalf("Failed to convert executable data to block %v", err)
			}
			newResp, err := api.ExecutePayload(*execData)
			if err != nil || newResp.Status != "VALID" {
				t.Fatalf("Failed to insert block: %v", err)
			}
			if ethservice.BlockChain().CurrentBlock().NumberU64() != head {
				t.Fatalf("Chain head shouldn't be updated")
			}
			if err := api.setHead(block.Hash()); err != nil {
				t.Fatalf("Failed to set head: %v", err)
			}
			if ethservice.BlockChain().CurrentBlock().NumberU64() != block.NumberU64() {
				t.Fatalf("Chain head should be updated")
			}
			parent, head = block, block.NumberU64()
		}
	*/
}

// startEthService creates a full node instance for testing.
func startEthService(t *testing.T, genesis *core.Genesis, blocks []*types.Block) (*node.Node, *eth.Ethereum) {
	t.Helper()

	n, err := node.New(&node.Config{
		P2P: p2p.Config{
			ListenAddr:  "0.0.0.0:0",
			NoDiscovery: true,
			MaxPeers:    25,
		}})
	if err != nil {
		t.Fatal("can't create node:", err)
	}

	ethcfg := &ethconfig.Config{Genesis: genesis, Ethash: ethash.Config{PowMode: ethash.ModeFake}, SyncMode: downloader.FullSync, TrieTimeout: time.Minute, TrieDirtyCache: 256, TrieCleanCache: 256}
	ethservice, err := eth.New(n, ethcfg)
	if err != nil {
		t.Fatal("can't create eth service:", err)
	}
	if err := n.Start(); err != nil {
		t.Fatal("can't start node:", err)
	}
	if _, err := ethservice.BlockChain().InsertChain(blocks); err != nil {
		n.Close()
		t.Fatal("can't import test blocks:", err)
	}
	time.Sleep(500 * time.Millisecond) // give txpool enough time to consume head event

	ethservice.SetEtherbase(testAddr)
	ethservice.SetSynced()
	return n, ethservice
}

func TestFullAPI(t *testing.T) {
	genesis, preMergeBlocks := generatePreMergeChain(10)
	n, ethservice := startEthService(t, genesis, preMergeBlocks)
	defer n.Close()
	var (
		parent = ethservice.BlockChain().CurrentBlock()
		// This EVM code generates a log when the contract is created.
		logCode = common.Hex2Bytes("60606040525b7f24ec1d3ff24c2f6ff210738839dbc339cd45a5294d85c79361016243157aae7b60405180905060405180910390a15b600a8060416000396000f360606040526008565b00")
	)

	callback := func(parent *types.Block) {
		statedb, _ := ethservice.BlockChain().StateAt(parent.Root())
		nonce := statedb.GetNonce(testAddr)
		tx, _ := types.SignTx(types.NewContractCreation(nonce, new(big.Int), 1000000, big.NewInt(2*params.InitialBaseFee), logCode), types.LatestSigner(ethservice.BlockChain().Config()), testKey)
		ethservice.TxPool().AddLocal(tx)
	}

	setupBlocks(t, ethservice, 10, parent, callback)
}

func setupBlocks(t *testing.T, ethservice *eth.Ethereum, n int, parent *types.Block, callback func(parent *types.Block)) {
	api := NewConsensusAPI(ethservice)
	for i := 0; i < n; i++ {
		callback(parent)

		payload := getNewPayload(t, api, parent)

		execResp, err := api.NewPayloadV1(*payload)
		if err != nil {
			t.Fatalf("can't execute payload: %v", err)
		}
		if execResp.Status != beacon.VALID {
			t.Fatalf("invalid status: %v", execResp.Status)
		}
		fcState := beacon.ForkchoiceStateV1{
			HeadBlockHash:      payload.BlockHash,
			SafeBlockHash:      payload.ParentHash,
			FinalizedBlockHash: payload.ParentHash,
		}
		if _, err := api.ForkchoiceUpdatedV1(fcState, nil); err != nil {
			t.Fatalf("Failed to insert block: %v", err)
		}
		if ethservice.BlockChain().CurrentBlock().NumberU64() != payload.Number {
			t.Fatal("Chain head should be updated")
		}
		if ethservice.BlockChain().CurrentFinalizedBlock().NumberU64() != payload.Number-1 {
			t.Fatal("Finalized block should be updated")
		}
		parent = ethservice.BlockChain().CurrentBlock()
	}
}

func TestExchangeTransitionConfig(t *testing.T) {
	genesis, preMergeBlocks := generatePreMergeChain(10)
	n, ethservice := startEthService(t, genesis, preMergeBlocks)
	defer n.Close()
	var (
		api = NewConsensusAPI(ethservice)
	)
	// invalid ttd
	config := beacon.TransitionConfigurationV1{
		TerminalTotalDifficulty: (*hexutil.Big)(big.NewInt(0)),
		TerminalBlockHash:       common.Hash{},
		TerminalBlockNumber:     0,
	}
	if _, err := api.ExchangeTransitionConfigurationV1(config); err == nil {
		t.Fatal("expected error on invalid config, invalid ttd")
	}
	// invalid terminal block hash
	config = beacon.TransitionConfigurationV1{
		TerminalTotalDifficulty: (*hexutil.Big)(genesis.Config.TerminalTotalDifficulty),
		TerminalBlockHash:       common.Hash{1},
		TerminalBlockNumber:     0,
	}
	if _, err := api.ExchangeTransitionConfigurationV1(config); err == nil {
		t.Fatal("expected error on invalid config, invalid hash")
	}
	// valid config
	config = beacon.TransitionConfigurationV1{
		TerminalTotalDifficulty: (*hexutil.Big)(genesis.Config.TerminalTotalDifficulty),
		TerminalBlockHash:       common.Hash{},
		TerminalBlockNumber:     0,
	}
	if _, err := api.ExchangeTransitionConfigurationV1(config); err != nil {
		t.Fatalf("expected no error on valid config, got %v", err)
	}
	// valid config
	config = beacon.TransitionConfigurationV1{
		TerminalTotalDifficulty: (*hexutil.Big)(genesis.Config.TerminalTotalDifficulty),
		TerminalBlockHash:       preMergeBlocks[5].Hash(),
		TerminalBlockNumber:     6,
	}
	if _, err := api.ExchangeTransitionConfigurationV1(config); err != nil {
		t.Fatalf("expected no error on valid config, got %v", err)
	}
}

/*
TestNewPayloadOnInvalidChain sets up a valid chain and tries to feed blocks
from an invalid chain to test if latestValidHash (LVH) works correctly.

We set up the following chain where P1 ... Pn and P1” are valid while
P1' is invalid.
We expect
(1) The LVH to point to the current inserted payload if it was valid.
(2) The LVH to point to the valid parent on an invalid payload (if the parent is available).
(3) If the parent is unavailable, the LVH should not be set.

	CommonAncestor◄─▲── P1 ◄── P2  ◄─ P3  ◄─ ... ◄─ Pn
	                │
	                └── P1' ◄─ P2' ◄─ P3' ◄─ ... ◄─ Pn'
	                │
	                └── P1''
*/
func TestNewPayloadOnInvalidChain(t *testing.T) {
	genesis, preMergeBlocks := generatePreMergeChain(10)
	n, ethservice := startEthService(t, genesis, preMergeBlocks)
	defer n.Close()

	var (
		api    = NewConsensusAPI(ethservice)
		parent = ethservice.BlockChain().CurrentBlock()
		// This EVM code generates a log when the contract is created.
		logCode = common.Hex2Bytes("60606040525b7f24ec1d3ff24c2f6ff210738839dbc339cd45a5294d85c79361016243157aae7b60405180905060405180910390a15b600a8060416000396000f360606040526008565b00")
	)
	for i := 0; i < 10; i++ {
		statedb, _ := ethservice.BlockChain().StateAt(parent.Root())
		nonce := statedb.GetNonce(testAddr)
		tx, _ := types.SignTx(types.NewContractCreation(nonce, new(big.Int), 1000000, big.NewInt(2*params.InitialBaseFee), logCode), types.LatestSigner(ethservice.BlockChain().Config()), testKey)
		ethservice.TxPool().AddLocal(tx)

		params := beacon.PayloadAttributesV1{
			Timestamp:             parent.Time() + 1,
			Random:                crypto.Keccak256Hash([]byte{byte(i)}),
			SuggestedFeeRecipient: parent.Coinbase(),
		}

		fcState := beacon.ForkchoiceStateV1{
			HeadBlockHash:      parent.Hash(),
			SafeBlockHash:      common.Hash{},
			FinalizedBlockHash: common.Hash{},
		}
		resp, err := api.ForkchoiceUpdatedV1(fcState, &params)
		if err != nil {
			t.Fatalf("error preparing payload, err=%v", err)
		}
		if resp.PayloadStatus.Status != beacon.VALID {
			t.Fatalf("error preparing payload, invalid status: %v", resp.PayloadStatus.Status)
		}
		payload, err := api.GetPayloadV1(*resp.PayloadID)
		if err != nil {
			t.Fatalf("can't get payload: %v", err)
		}
		// TODO(493456442, marius) this test can be flaky since we rely on a 100ms
		// allowance for block generation internally.
		if len(payload.Transactions) == 0 {
			t.Fatalf("payload should not be empty")
		}
		execResp, err := api.NewPayloadV1(*payload)
		if err != nil {
			t.Fatalf("can't execute payload: %v", err)
		}
		if execResp.Status != beacon.VALID {
			t.Fatalf("invalid status: %v", execResp.Status)
		}
		fcState = beacon.ForkchoiceStateV1{
			HeadBlockHash:      payload.BlockHash,
			SafeBlockHash:      payload.ParentHash,
			FinalizedBlockHash: payload.ParentHash,
		}
		if _, err := api.ForkchoiceUpdatedV1(fcState, nil); err != nil {
			t.Fatalf("Failed to insert block: %v", err)
		}
		if ethservice.BlockChain().CurrentBlock().NumberU64() != payload.Number {
			t.Fatalf("Chain head should be updated")
		}
		parent = ethservice.BlockChain().CurrentBlock()
	}
}

func assembleBlock(api *ConsensusAPI, parentHash common.Hash, params *beacon.PayloadAttributesV1) (*beacon.ExecutableDataV1, error) {
	block, err := api.eth.Miner().GetSealingBlockSync(parentHash, params.Timestamp, params.SuggestedFeeRecipient, params.Random, false)
	if err != nil {
		return nil, err
	}
	return beacon.BlockToExecutableData(block), nil
}

func TestEmptyBlocks(t *testing.T) {
	genesis, preMergeBlocks := generatePreMergeChain(10)
	n, ethservice := startEthService(t, genesis, preMergeBlocks)
	defer n.Close()

	commonAncestor := ethservice.BlockChain().CurrentBlock()
	api := NewConsensusAPI(ethservice)

	// Setup 10 blocks on the canonical chain
	setupBlocks(t, ethservice, 10, commonAncestor, func(parent *types.Block) {})

	// (1) check LatestValidHash by sending a normal payload (P1'')
	payload := getNewPayload(t, api, commonAncestor)

	status, err := api.NewPayloadV1(*payload)
	if err != nil {
		t.Fatal(err)
	}
	if status.Status != beacon.VALID {
		t.Errorf("invalid status: expected VALID got: %v", status.Status)
	}
	if !bytes.Equal(status.LatestValidHash[:], payload.BlockHash[:]) {
		t.Fatalf("invalid LVH: got %v want %v", status.LatestValidHash, payload.BlockHash)
	}

	// (2) Now send P1' which is invalid
	payload = getNewPayload(t, api, commonAncestor)
	payload.GasUsed += 1
	payload = setBlockhash(payload)
	// Now latestValidHash should be the common ancestor
	status, err = api.NewPayloadV1(*payload)
	if err != nil {
		t.Fatal(err)
	}
	if status.Status != beacon.INVALID {
		t.Errorf("invalid status: expected INVALID got: %v", status.Status)
	}
	// Expect 0x0 on INVALID block on top of PoW block
	expected := common.Hash{}
	if !bytes.Equal(status.LatestValidHash[:], expected[:]) {
		t.Fatalf("invalid LVH: got %v want %v", status.LatestValidHash, expected)
	}

	// (3) Now send a payload with unknown parent
	payload = getNewPayload(t, api, commonAncestor)
	payload.ParentHash = common.Hash{1}
	payload = setBlockhash(payload)
	// Now latestValidHash should be the common ancestor
	status, err = api.NewPayloadV1(*payload)
	if err != nil {
		t.Fatal(err)
	}
	if status.Status != beacon.SYNCING {
		t.Errorf("invalid status: expected SYNCING got: %v", status.Status)
	}
	if status.LatestValidHash != nil {
		t.Fatalf("invalid LVH: got %v wanted nil", status.LatestValidHash)
	}
}

func getNewPayload(t *testing.T, api *ConsensusAPI, parent *types.Block) *beacon.ExecutableDataV1 {
	params := beacon.PayloadAttributesV1{
		Timestamp:             parent.Time() + 1,
		Random:                crypto.Keccak256Hash([]byte{byte(1)}),
		SuggestedFeeRecipient: parent.Coinbase(),
	}

	payload, err := assembleBlock(api, parent.Hash(), &params)
	if err != nil {
		t.Fatal(err)
	}
	return payload
}

// setBlockhash sets the blockhash of a modified ExecutableData.
// Can be used to make modified payloads look valid.
func setBlockhash(data *beacon.ExecutableDataV1) *beacon.ExecutableDataV1 {
	txs, _ := decodeTransactions(data.Transactions)
	number := big.NewInt(0)
	number.SetUint64(data.Number)
	header := &types.Header{
		ParentHash:  data.ParentHash,
		UncleHash:   types.EmptyUncleHash,
		Coinbase:    data.FeeRecipient,
		Root:        data.StateRoot,
		TxHash:      types.DeriveSha(types.Transactions(txs), trie.NewStackTrie(nil)),
		ReceiptHash: data.ReceiptsRoot,
		Bloom:       types.BytesToBloom(data.LogsBloom),
		Difficulty:  common.Big0,
		Number:      number,
		GasLimit:    data.GasLimit,
		GasUsed:     data.GasUsed,
		Time:        data.Timestamp,
		BaseFee:     data.BaseFeePerGas,
		Extra:       data.ExtraData,
		MixDigest:   data.Random,
	}
	block := types.NewBlockWithHeader(header).WithBody(txs, nil /* uncles */)
	data.BlockHash = block.Hash()
	return data
}

func decodeTransactions(enc [][]byte) ([]*types.Transaction, error) {
	var txs = make([]*types.Transaction, len(enc))
	for i, encTx := range enc {
		var tx types.Transaction
		if err := tx.UnmarshalBinary(encTx); err != nil {
			return nil, fmt.Errorf("invalid transaction %d: %v", i, err)
		}
		txs[i] = &tx
	}
	return txs, nil
}

func TestTrickRemoteBlockCache(t *testing.T) {
	// Setup two nodes
	genesis, preMergeBlocks := generatePreMergeChain(10)
	nodeA, ethserviceA := startEthService(t, genesis, preMergeBlocks)
	nodeB, ethserviceB := startEthService(t, genesis, preMergeBlocks)
	defer nodeA.Close()
	defer nodeB.Close()
	for nodeB.Server().NodeInfo().Ports.Listener == 0 {
		time.Sleep(250 * time.Millisecond)
	}
	nodeA.Server().AddPeer(nodeB.Server().Self())
	nodeB.Server().AddPeer(nodeA.Server().Self())
	apiA := NewConsensusAPI(ethserviceA)
	apiB := NewConsensusAPI(ethserviceB)

	commonAncestor := ethserviceA.BlockChain().CurrentBlock()

	// Setup 10 blocks on the canonical chain
	setupBlocks(t, ethserviceA, 10, commonAncestor, func(parent *types.Block) {})
	commonAncestor = ethserviceA.BlockChain().CurrentBlock()

	var invalidChain []*beacon.ExecutableDataV1
	// create a valid payload (P1)
	//payload1 := getNewPayload(t, apiA, commonAncestor)
	//invalidChain = append(invalidChain, payload1)

	// create an invalid payload2 (P2)
	payload2 := getNewPayload(t, apiA, commonAncestor)
	//payload2.ParentHash = payload1.BlockHash
	payload2.GasUsed += 1
	payload2 = setBlockhash(payload2)
	invalidChain = append(invalidChain, payload2)

	head := payload2
	// create some valid payloads on top
	for i := 0; i < 10; i++ {
		payload := getNewPayload(t, apiA, commonAncestor)
		payload.ParentHash = head.BlockHash
		payload = setBlockhash(payload)
		invalidChain = append(invalidChain, payload)
		head = payload
	}

	// feed the payloads to node B
	for _, payload := range invalidChain {
		status, err := apiB.NewPayloadV1(*payload)
		if err != nil {
			panic(err)
		}
		if status.Status == beacon.VALID {
			t.Error("invalid status: VALID on an invalid chain")
		}
		// Now reorg to the head of the invalid chain
		resp, err := apiB.ForkchoiceUpdatedV1(beacon.ForkchoiceStateV1{HeadBlockHash: payload.BlockHash, SafeBlockHash: payload.BlockHash, FinalizedBlockHash: payload.ParentHash}, nil)
		if err != nil {
			t.Fatal(err)
		}
		if resp.PayloadStatus.Status == beacon.VALID {
			t.Error("invalid status: VALID on an invalid chain")
		}
		time.Sleep(100 * time.Millisecond)
	}
}

func TestInvalidBloom(t *testing.T) {
	genesis, preMergeBlocks := generatePreMergeChain(10)
	n, ethservice := startEthService(t, genesis, preMergeBlocks)
	ethservice.Merger().ReachTTD()
	defer n.Close()

	commonAncestor := ethservice.BlockChain().CurrentBlock()
	api := NewConsensusAPI(ethservice)

	// Setup 10 blocks on the canonical chain
	setupBlocks(t, ethservice, 10, commonAncestor, func(parent *types.Block) {})

	// (1) check LatestValidHash by sending a normal payload (P1'')
	payload := getNewPayload(t, api, commonAncestor)
	payload.LogsBloom = append(payload.LogsBloom, byte(1))
	status, err := api.NewPayloadV1(*payload)
	if err != nil {
		t.Fatal(err)
	}
	if status.Status != beacon.INVALIDBLOCKHASH {
		t.Errorf("invalid status: expected VALID got: %v", status.Status)
	}
}

func TestNewPayloadOnInvalidTerminalBlock(t *testing.T) {
	genesis, preMergeBlocks := generatePreMergeChain(100)
	fmt.Println(genesis.Config.TerminalTotalDifficulty)
	genesis.Config.TerminalTotalDifficulty = preMergeBlocks[0].Difficulty() //.Sub(genesis.Config.TerminalTotalDifficulty, preMergeBlocks[len(preMergeBlocks)-1].Difficulty())

	fmt.Println(genesis.Config.TerminalTotalDifficulty)
	n, ethservice := startEthService(t, genesis, preMergeBlocks)
	defer n.Close()

	var (
		api    = NewConsensusAPI(ethservice)
		parent = preMergeBlocks[len(preMergeBlocks)-1]
	)

	// Test parent already post TTD in FCU
	fcState := beacon.ForkchoiceStateV1{
		HeadBlockHash:      parent.Hash(),
		SafeBlockHash:      common.Hash{},
		FinalizedBlockHash: common.Hash{},
	}
	resp, err := api.ForkchoiceUpdatedV1(fcState, nil)
	if err != nil {
		t.Fatalf("error sending forkchoice, err=%v", err)
	}
	if resp.PayloadStatus != beacon.INVALID_TERMINAL_BLOCK {
		t.Fatalf("error sending invalid forkchoice, invalid status: %v", resp.PayloadStatus.Status)
	}

	// Test parent already post TTD in NewPayload
	params := beacon.PayloadAttributesV1{
		Timestamp:             parent.Time() + 1,
		Random:                crypto.Keccak256Hash([]byte{byte(1)}),
		SuggestedFeeRecipient: parent.Coinbase(),
	}
	empty, err := api.eth.Miner().GetSealingBlockSync(parent.Hash(), params.Timestamp, params.SuggestedFeeRecipient, params.Random, true)
	if err != nil {
		t.Fatalf("error preparing payload, err=%v", err)
	}
	data := *beacon.BlockToExecutableData(empty)
	resp2, err := api.NewPayloadV1(data)
	if err != nil {
		t.Fatalf("error sending NewPayload, err=%v", err)
	}
	if resp2 != beacon.INVALID_TERMINAL_BLOCK {
		t.Fatalf("error sending invalid forkchoice, invalid status: %v", resp.PayloadStatus.Status)
	}
}
