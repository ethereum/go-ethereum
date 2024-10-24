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
	"context"
	crand "crypto/rand"
	"errors"
	"fmt"
	"math/big"
	"math/rand"
	"reflect"
	"sync"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/beacon/engine"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/consensus"
	beaconConsensus "github.com/ethereum/go-ethereum/consensus/beacon"
	"github.com/ethereum/go-ethereum/consensus/ethash"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/crypto/kzg4844"
	"github.com/ethereum/go-ethereum/eth"
	"github.com/ethereum/go-ethereum/eth/downloader"
	"github.com/ethereum/go-ethereum/eth/ethconfig"
	"github.com/ethereum/go-ethereum/internal/version"
	"github.com/ethereum/go-ethereum/miner"
	"github.com/ethereum/go-ethereum/node"
	"github.com/ethereum/go-ethereum/p2p"
	"github.com/ethereum/go-ethereum/params"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/ethereum/go-ethereum/trie"
)

var (
	// testKey is a private key to use for funding a tester account.
	testKey, _ = crypto.HexToECDSA("b71c71a67e1177ad4e901695e1b4b9ee17ae16c6668d313eac2f96dbcda3f291")

	// testAddr is the Ethereum address of the tester account.
	testAddr = crypto.PubkeyToAddress(testKey.PublicKey)

	testBalance = big.NewInt(2e18)
)

func generateMergeChain(n int, merged bool) (*core.Genesis, []*types.Block) {
	config := *params.AllEthashProtocolChanges
	engine := consensus.Engine(beaconConsensus.New(ethash.NewFaker()))
	if merged {
		config.TerminalTotalDifficulty = common.Big0
		engine = beaconConsensus.NewFaker()
	}
	genesis := &core.Genesis{
		Config: &config,
		Alloc: types.GenesisAlloc{
			testAddr:                  {Balance: testBalance},
			params.BeaconRootsAddress: {Balance: common.Big0, Code: common.Hex2Bytes("3373fffffffffffffffffffffffffffffffffffffffe14604457602036146024575f5ffd5b620180005f350680545f35146037575f5ffd5b6201800001545f5260205ff35b6201800042064281555f359062018000015500")},
			config.DepositContractAddress: {
				// Simple deposit generator, source: https://gist.github.com/lightclient/54abb2af2465d6969fa6d1920b9ad9d7
				Code:    common.Hex2Bytes("6080604052366103aa575f603067ffffffffffffffff811115610025576100246103ae565b5b6040519080825280601f01601f1916602001820160405280156100575781602001600182028036833780820191505090505b5090505f8054906101000a900460ff1660f81b815f8151811061007d5761007c6103db565b5b60200101907effffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff191690815f1a9053505f602067ffffffffffffffff8111156100c7576100c66103ae565b5b6040519080825280601f01601f1916602001820160405280156100f95781602001600182028036833780820191505090505b5090505f8054906101000a900460ff1660f81b815f8151811061011f5761011e6103db565b5b60200101907effffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff191690815f1a9053505f600867ffffffffffffffff811115610169576101686103ae565b5b6040519080825280601f01601f19166020018201604052801561019b5781602001600182028036833780820191505090505b5090505f8054906101000a900460ff1660f81b815f815181106101c1576101c06103db565b5b60200101907effffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff191690815f1a9053505f606067ffffffffffffffff81111561020b5761020a6103ae565b5b6040519080825280601f01601f19166020018201604052801561023d5781602001600182028036833780820191505090505b5090505f8054906101000a900460ff1660f81b815f81518110610263576102626103db565b5b60200101907effffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff191690815f1a9053505f600867ffffffffffffffff8111156102ad576102ac6103ae565b5b6040519080825280601f01601f1916602001820160405280156102df5781602001600182028036833780820191505090505b5090505f8054906101000a900460ff1660f81b815f81518110610305576103046103db565b5b60200101907effffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff191690815f1a9053505f8081819054906101000a900460ff168092919061035090610441565b91906101000a81548160ff021916908360ff160217905550507f649bbc62d0e31342afea4e5cd82d4049e7e1ee912fc0889aa790803be39038c585858585856040516103a09594939291906104d9565b60405180910390a1005b5f80fd5b7f4e487b71000000000000000000000000000000000000000000000000000000005f52604160045260245ffd5b7f4e487b71000000000000000000000000000000000000000000000000000000005f52603260045260245ffd5b7f4e487b71000000000000000000000000000000000000000000000000000000005f52601160045260245ffd5b5f60ff82169050919050565b5f61044b82610435565b915060ff820361045e5761045d610408565b5b600182019050919050565b5f81519050919050565b5f82825260208201905092915050565b8281835e5f83830152505050565b5f601f19601f8301169050919050565b5f6104ab82610469565b6104b58185610473565b93506104c5818560208601610483565b6104ce81610491565b840191505092915050565b5f60a0820190508181035f8301526104f181886104a1565b9050818103602083015261050581876104a1565b9050818103604083015261051981866104a1565b9050818103606083015261052d81856104a1565b9050818103608083015261054181846104a1565b9050969550505050505056fea26469706673582212208569967e58690162d7d6fe3513d07b393b4c15e70f41505cbbfd08f53eba739364736f6c63430008190033"),
				Nonce:   0,
				Balance: big.NewInt(0),
			},
		},
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
	_, blocks, _ := core.GenerateChainWithGenesis(genesis, engine, n, generate)

	if !merged {
		totalDifficulty := big.NewInt(0)
		for _, b := range blocks {
			totalDifficulty.Add(totalDifficulty, b.Difficulty())
		}
		config.TerminalTotalDifficulty = totalDifficulty
	}

	return genesis, blocks
}

func TestEth2AssembleBlock(t *testing.T) {
	genesis, blocks := generateMergeChain(10, false)
	n, ethservice := startEthService(t, genesis, blocks)
	defer n.Close()

	api := NewConsensusAPI(ethservice)
	signer := types.NewEIP155Signer(ethservice.BlockChain().Config().ChainID)
	tx, err := types.SignTx(types.NewTransaction(uint64(10), blocks[9].Coinbase(), big.NewInt(1000), params.TxGas, big.NewInt(params.InitialBaseFee), nil), signer, testKey)
	if err != nil {
		t.Fatalf("error signing transaction, err=%v", err)
	}
	ethservice.TxPool().Add([]*types.Transaction{tx}, true, true)
	blockParams := engine.PayloadAttributes{
		Timestamp: blocks[9].Time() + 5,
	}
	// The miner needs to pick up on the txs in the pool, so a few retries might be
	// needed.
	if _, testErr := assembleWithTransactions(api, blocks[9].Hash(), &blockParams, 1); testErr != nil {
		t.Fatal(testErr)
	}
}

// assembleWithTransactions tries to assemble a block, retrying until it has 'want',
// number of transactions in it, or it has retried three times.
func assembleWithTransactions(api *ConsensusAPI, parentHash common.Hash, params *engine.PayloadAttributes, want int) (execData *engine.ExecutableData, err error) {
	for retries := 3; retries > 0; retries-- {
		execData, err = assembleBlock(api, parentHash, params)
		if err != nil {
			return nil, err
		}
		if have, want := len(execData.Transactions), want; have != want {
			err = fmt.Errorf("invalid number of transactions, have %d want %d", have, want)
			continue
		}
		return execData, nil
	}
	return nil, err
}

func TestEth2AssembleBlockWithAnotherBlocksTxs(t *testing.T) {
	genesis, blocks := generateMergeChain(10, false)
	n, ethservice := startEthService(t, genesis, blocks[:9])
	defer n.Close()

	api := NewConsensusAPI(ethservice)

	// Put the 10th block's tx in the pool and produce a new block
	txs := blocks[9].Transactions()
	api.eth.TxPool().Add(txs, false, true)
	blockParams := engine.PayloadAttributes{
		Timestamp: blocks[8].Time() + 5,
	}
	// The miner needs to pick up on the txs in the pool, so a few retries might be
	// needed.
	if _, err := assembleWithTransactions(api, blocks[8].Hash(), &blockParams, blocks[9].Transactions().Len()); err != nil {
		t.Fatal(err)
	}
}

func TestSetHeadBeforeTotalDifficulty(t *testing.T) {
	genesis, blocks := generateMergeChain(10, false)
	n, ethservice := startEthService(t, genesis, blocks)
	defer n.Close()

	api := NewConsensusAPI(ethservice)
	fcState := engine.ForkchoiceStateV1{
		HeadBlockHash:      blocks[5].Hash(),
		SafeBlockHash:      common.Hash{},
		FinalizedBlockHash: common.Hash{},
	}
	if resp, err := api.ForkchoiceUpdatedV1(fcState, nil); err != nil {
		t.Errorf("fork choice updated should not error: %v", err)
	} else if resp.PayloadStatus.Status != engine.INVALID_TERMINAL_BLOCK.Status {
		t.Errorf("fork choice updated before total terminal difficulty should be INVALID")
	}
}

func TestEth2PrepareAndGetPayload(t *testing.T) {
	genesis, blocks := generateMergeChain(10, false)
	// We need to properly set the terminal total difficulty
	genesis.Config.TerminalTotalDifficulty.Sub(genesis.Config.TerminalTotalDifficulty, blocks[9].Difficulty())
	n, ethservice := startEthService(t, genesis, blocks[:9])
	defer n.Close()

	api := NewConsensusAPI(ethservice)

	// Put the 10th block's tx in the pool and produce a new block
	txs := blocks[9].Transactions()
	ethservice.TxPool().Add(txs, true, true)
	blockParams := engine.PayloadAttributes{
		Timestamp: blocks[8].Time() + 5,
	}
	fcState := engine.ForkchoiceStateV1{
		HeadBlockHash:      blocks[8].Hash(),
		SafeBlockHash:      common.Hash{},
		FinalizedBlockHash: common.Hash{},
	}
	_, err := api.ForkchoiceUpdatedV1(fcState, &blockParams)
	if err != nil {
		t.Fatalf("error preparing payload, err=%v", err)
	}
	// give the payload some time to be built
	time.Sleep(100 * time.Millisecond)
	payloadID := (&miner.BuildPayloadArgs{
		Parent:       fcState.HeadBlockHash,
		Timestamp:    blockParams.Timestamp,
		FeeRecipient: blockParams.SuggestedFeeRecipient,
		Random:       blockParams.Random,
		BeaconRoot:   blockParams.BeaconRoot,
		Version:      engine.PayloadV1,
	}).Id()
	execData, err := api.GetPayloadV1(payloadID)
	if err != nil {
		t.Fatalf("error getting payload, err=%v", err)
	}
	if len(execData.Transactions) != blocks[9].Transactions().Len() {
		t.Fatalf("invalid number of transactions %d != 1", len(execData.Transactions))
	}
	// Test invalid payloadID
	var invPayload engine.PayloadID
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
	genesis, preMergeBlocks := generateMergeChain(10, false)
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
		{parent.Time, true},
		{parent.Time - 1, true},
		{parent.Time + 1, false},
		{uint64(time.Now().Unix()) + uint64(time.Minute), false},
	}

	for i, test := range tests {
		t.Run(fmt.Sprintf("Timestamp test: %v", i), func(t *testing.T) {
			params := engine.PayloadAttributes{
				Timestamp:             test.time,
				Random:                crypto.Keccak256Hash([]byte{byte(123)}),
				SuggestedFeeRecipient: parent.Coinbase,
			}
			fcState := engine.ForkchoiceStateV1{
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
	genesis, preMergeBlocks := generateMergeChain(10, false)
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
		ethservice.TxPool().Add([]*types.Transaction{tx}, true, true)

		execData, err := assembleWithTransactions(api, parent.Hash(), &engine.PayloadAttributes{
			Timestamp: parent.Time() + 5,
		}, 1)
		if err != nil {
			t.Fatalf("Failed to create the executable data, block %d: %v", i, err)
		}
		block, err := engine.ExecutableDataToBlock(*execData, nil, nil, nil)
		if err != nil {
			t.Fatalf("Failed to convert executable data to block %v", err)
		}
		newResp, err := api.NewPayloadV1(*execData)
		switch {
		case err != nil:
			t.Fatalf("Failed to insert block: %v", err)
		case newResp.Status != "VALID":
			t.Fatalf("Failed to insert block: %v", newResp.Status)
		case ethservice.BlockChain().CurrentBlock().Number.Uint64() != block.NumberU64()-1:
			t.Fatalf("Chain head shouldn't be updated")
		}
		checkLogEvents(t, newLogCh, rmLogsCh, 0, 0)
		fcState := engine.ForkchoiceStateV1{
			HeadBlockHash:      block.Hash(),
			SafeBlockHash:      block.Hash(),
			FinalizedBlockHash: block.Hash(),
		}
		if _, err := api.ForkchoiceUpdatedV1(fcState, nil); err != nil {
			t.Fatalf("Failed to insert block: %v", err)
		}
		if have, want := ethservice.BlockChain().CurrentBlock().Number.Uint64(), block.NumberU64(); have != want {
			t.Fatalf("Chain head should be updated, have %d want %d", have, want)
		}
		checkLogEvents(t, newLogCh, rmLogsCh, 1, 0)

		parent = block
	}

	// Introduce fork chain
	var (
		head = ethservice.BlockChain().CurrentBlock().Number.Uint64()
	)
	parent = preMergeBlocks[len(preMergeBlocks)-1]
	for i := 0; i < 10; i++ {
		execData, err := assembleBlock(api, parent.Hash(), &engine.PayloadAttributes{
			Timestamp: parent.Time() + 6,
		})
		if err != nil {
			t.Fatalf("Failed to create the executable data %v", err)
		}
		block, err := engine.ExecutableDataToBlock(*execData, nil, nil, nil)
		if err != nil {
			t.Fatalf("Failed to convert executable data to block %v", err)
		}
		newResp, err := api.NewPayloadV1(*execData)
		if err != nil || newResp.Status != "VALID" {
			t.Fatalf("Failed to insert block: %v", err)
		}
		if ethservice.BlockChain().CurrentBlock().Number.Uint64() != head {
			t.Fatalf("Chain head shouldn't be updated")
		}

		fcState := engine.ForkchoiceStateV1{
			HeadBlockHash:      block.Hash(),
			SafeBlockHash:      block.Hash(),
			FinalizedBlockHash: block.Hash(),
		}
		if _, err := api.ForkchoiceUpdatedV1(fcState, nil); err != nil {
			t.Fatalf("Failed to insert block: %v", err)
		}
		if ethservice.BlockChain().CurrentBlock().Number.Uint64() != block.NumberU64() {
			t.Fatalf("Chain head should be updated")
		}
		parent, head = block, block.NumberU64()
	}
}

func TestEth2DeepReorg(t *testing.T) {
	// TODO (MariusVanDerWijden) TestEth2DeepReorg is currently broken, because it tries to reorg
	// before the totalTerminalDifficulty threshold
	/*
		genesis, preMergeBlocks := generateMergeChain(core.TriesInMemory * 2, false)
		n, ethservice := startEthService(t, genesis, preMergeBlocks)
		defer n.Close()

		var (
			api    = NewConsensusAPI(ethservice, nil)
			parent = preMergeBlocks[len(preMergeBlocks)-core.TriesInMemory-1]
			head   = ethservice.BlockChain().CurrentBlock().Number.Uint64()()
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
			if ethservice.BlockChain().CurrentBlock().Number.Uint64()() != head {
				t.Fatalf("Chain head shouldn't be updated")
			}
			if err := api.setHead(block.Hash()); err != nil {
				t.Fatalf("Failed to set head: %v", err)
			}
			if ethservice.BlockChain().CurrentBlock().Number.Uint64()() != block.NumberU64() {
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

	mcfg := miner.DefaultConfig
	mcfg.PendingFeeRecipient = testAddr
	ethcfg := &ethconfig.Config{Genesis: genesis, SyncMode: downloader.FullSync, TrieTimeout: time.Minute, TrieDirtyCache: 256, TrieCleanCache: 256, Miner: mcfg}
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

	ethservice.SetSynced()
	return n, ethservice
}

func TestFullAPI(t *testing.T) {
	genesis, preMergeBlocks := generateMergeChain(10, false)
	n, ethservice := startEthService(t, genesis, preMergeBlocks)
	defer n.Close()
	var (
		parent = ethservice.BlockChain().CurrentBlock()
		// This EVM code generates a log when the contract is created.
		logCode = common.Hex2Bytes("60606040525b7f24ec1d3ff24c2f6ff210738839dbc339cd45a5294d85c79361016243157aae7b60405180905060405180910390a15b600a8060416000396000f360606040526008565b00")
	)

	callback := func(parent *types.Header) {
		statedb, _ := ethservice.BlockChain().StateAt(parent.Root)
		nonce := statedb.GetNonce(testAddr)
		tx, _ := types.SignTx(types.NewContractCreation(nonce, new(big.Int), 1000000, big.NewInt(2*params.InitialBaseFee), logCode), types.LatestSigner(ethservice.BlockChain().Config()), testKey)
		ethservice.TxPool().Add([]*types.Transaction{tx}, true, false)
	}

	setupBlocks(t, ethservice, 10, parent, callback, nil, nil)
}

func setupBlocks(t *testing.T, ethservice *eth.Ethereum, n int, parent *types.Header, callback func(parent *types.Header), withdrawals [][]*types.Withdrawal, beaconRoots []common.Hash) []*types.Header {
	api := NewConsensusAPI(ethservice)
	var blocks []*types.Header
	for i := 0; i < n; i++ {
		callback(parent)
		var w []*types.Withdrawal
		if withdrawals != nil {
			w = withdrawals[i]
		}
		var h *common.Hash
		if beaconRoots != nil {
			h = &beaconRoots[i]
		}

		envelope := getNewEnvelope(t, api, parent, w, h)
		execResp, err := api.newPayload(*envelope.ExecutionPayload, []common.Hash{}, h, envelope.Requests, false)
		if err != nil {
			t.Fatalf("can't execute payload: %v", err)
		}
		if execResp.Status != engine.VALID {
			t.Fatalf("invalid status: %v %s", execResp.Status, *execResp.ValidationError)
		}
		payload := envelope.ExecutionPayload
		fcState := engine.ForkchoiceStateV1{
			HeadBlockHash:      payload.BlockHash,
			SafeBlockHash:      payload.ParentHash,
			FinalizedBlockHash: payload.ParentHash,
		}
		if _, err := api.ForkchoiceUpdatedV1(fcState, nil); err != nil {
			t.Fatalf("Failed to insert block: %v", err)
		}
		if ethservice.BlockChain().CurrentBlock().Number.Uint64() != payload.Number {
			t.Fatal("Chain head should be updated")
		}
		if ethservice.BlockChain().CurrentFinalBlock().Number.Uint64() != payload.Number-1 {
			t.Fatal("Finalized block should be updated")
		}
		parent = ethservice.BlockChain().CurrentBlock()
		blocks = append(blocks, parent)
	}
	return blocks
}

func TestExchangeTransitionConfig(t *testing.T) {
	genesis, preMergeBlocks := generateMergeChain(10, false)
	n, ethservice := startEthService(t, genesis, preMergeBlocks)
	defer n.Close()

	// invalid ttd
	api := NewConsensusAPI(ethservice)
	config := engine.TransitionConfigurationV1{
		TerminalTotalDifficulty: (*hexutil.Big)(big.NewInt(0)),
		TerminalBlockHash:       common.Hash{},
		TerminalBlockNumber:     0,
	}
	if _, err := api.ExchangeTransitionConfigurationV1(config); err == nil {
		t.Fatal("expected error on invalid config, invalid ttd")
	}
	// invalid terminal block hash
	config = engine.TransitionConfigurationV1{
		TerminalTotalDifficulty: (*hexutil.Big)(genesis.Config.TerminalTotalDifficulty),
		TerminalBlockHash:       common.Hash{1},
		TerminalBlockNumber:     0,
	}
	if _, err := api.ExchangeTransitionConfigurationV1(config); err == nil {
		t.Fatal("expected error on invalid config, invalid hash")
	}
	// valid config
	config = engine.TransitionConfigurationV1{
		TerminalTotalDifficulty: (*hexutil.Big)(genesis.Config.TerminalTotalDifficulty),
		TerminalBlockHash:       common.Hash{},
		TerminalBlockNumber:     0,
	}
	if _, err := api.ExchangeTransitionConfigurationV1(config); err != nil {
		t.Fatalf("expected no error on valid config, got %v", err)
	}
	// valid config
	config = engine.TransitionConfigurationV1{
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
	genesis, preMergeBlocks := generateMergeChain(10, false)
	n, ethservice := startEthService(t, genesis, preMergeBlocks)
	defer n.Close()

	var (
		api    = NewConsensusAPI(ethservice)
		parent = ethservice.BlockChain().CurrentBlock()
		signer = types.LatestSigner(ethservice.BlockChain().Config())
		// This EVM code generates a log when the contract is created.
		logCode = common.Hex2Bytes("60606040525b7f24ec1d3ff24c2f6ff210738839dbc339cd45a5294d85c79361016243157aae7b60405180905060405180910390a15b600a8060416000396000f360606040526008565b00")
	)
	for i := 0; i < 10; i++ {
		statedb, _ := ethservice.BlockChain().StateAt(parent.Root)
		tx := types.MustSignNewTx(testKey, signer, &types.LegacyTx{
			Nonce:    statedb.GetNonce(testAddr),
			Value:    new(big.Int),
			Gas:      1000000,
			GasPrice: big.NewInt(2 * params.InitialBaseFee),
			Data:     logCode,
		})
		ethservice.TxPool().Add([]*types.Transaction{tx}, false, true)
		var (
			params = engine.PayloadAttributes{
				Timestamp:             parent.Time + 1,
				Random:                crypto.Keccak256Hash([]byte{byte(i)}),
				SuggestedFeeRecipient: parent.Coinbase,
			}
			fcState = engine.ForkchoiceStateV1{
				HeadBlockHash:      parent.Hash(),
				SafeBlockHash:      common.Hash{},
				FinalizedBlockHash: common.Hash{},
			}
			payload *engine.ExecutableData
			resp    engine.ForkChoiceResponse
			err     error
		)
		for i := 0; ; i++ {
			if resp, err = api.ForkchoiceUpdatedV1(fcState, &params); err != nil {
				t.Fatalf("error preparing payload, err=%v", err)
			}
			if resp.PayloadStatus.Status != engine.VALID {
				t.Fatalf("error preparing payload, invalid status: %v", resp.PayloadStatus.Status)
			}
			// give the payload some time to be built
			time.Sleep(50 * time.Millisecond)
			if payload, err = api.GetPayloadV1(*resp.PayloadID); err != nil {
				t.Fatalf("can't get payload: %v", err)
			}
			if len(payload.Transactions) > 0 {
				break
			}
			// No luck this time we need to update the params and try again.
			params.Timestamp = params.Timestamp + 1
			if i > 10 {
				t.Fatalf("payload should not be empty")
			}
		}
		execResp, err := api.NewPayloadV1(*payload)
		if err != nil {
			t.Fatalf("can't execute payload: %v", err)
		}
		if execResp.Status != engine.VALID {
			t.Fatalf("invalid status: %v", execResp.Status)
		}
		fcState = engine.ForkchoiceStateV1{
			HeadBlockHash:      payload.BlockHash,
			SafeBlockHash:      payload.ParentHash,
			FinalizedBlockHash: payload.ParentHash,
		}
		if _, err := api.ForkchoiceUpdatedV1(fcState, nil); err != nil {
			t.Fatalf("Failed to insert block: %v", err)
		}
		if ethservice.BlockChain().CurrentBlock().Number.Uint64() != payload.Number {
			t.Fatalf("Chain head should be updated")
		}
		parent = ethservice.BlockChain().CurrentBlock()
	}
}

func assembleEnvelope(api *ConsensusAPI, parentHash common.Hash, params *engine.PayloadAttributes) (*engine.ExecutionPayloadEnvelope, error) {
	args := &miner.BuildPayloadArgs{
		Parent:       parentHash,
		Timestamp:    params.Timestamp,
		FeeRecipient: params.SuggestedFeeRecipient,
		Random:       params.Random,
		Withdrawals:  params.Withdrawals,
		BeaconRoot:   params.BeaconRoot,
	}
	payload, err := api.eth.Miner().BuildPayload(args, false)
	if err != nil {
		return nil, err
	}
	return payload.ResolveFull(), nil
}

func assembleBlock(api *ConsensusAPI, parentHash common.Hash, params *engine.PayloadAttributes) (*engine.ExecutableData, error) {
	envelope, err := assembleEnvelope(api, parentHash, params)
	if err != nil {
		return nil, err
	}
	return envelope.ExecutionPayload, nil
}

func TestEmptyBlocks(t *testing.T) {
	genesis, preMergeBlocks := generateMergeChain(10, false)
	n, ethservice := startEthService(t, genesis, preMergeBlocks)
	defer n.Close()

	commonAncestor := ethservice.BlockChain().CurrentBlock()
	api := NewConsensusAPI(ethservice)

	// Setup 10 blocks on the canonical chain
	setupBlocks(t, ethservice, 10, commonAncestor, func(parent *types.Header) {}, nil, nil)

	// (1) check LatestValidHash by sending a normal payload (P1'')
	payload := getNewPayload(t, api, commonAncestor, nil, nil)

	status, err := api.NewPayloadV1(*payload)
	if err != nil {
		t.Fatal(err)
	}
	if status.Status != engine.VALID {
		t.Errorf("invalid status: expected VALID got: %v", status.Status)
	}
	if !bytes.Equal(status.LatestValidHash[:], payload.BlockHash[:]) {
		t.Fatalf("invalid LVH: got %v want %v", status.LatestValidHash, payload.BlockHash)
	}

	// (2) Now send P1' which is invalid
	payload = getNewPayload(t, api, commonAncestor, nil, nil)
	payload.GasUsed += 1
	payload = setBlockhash(payload)
	// Now latestValidHash should be the common ancestor
	status, err = api.NewPayloadV1(*payload)
	if err != nil {
		t.Fatal(err)
	}
	if status.Status != engine.INVALID {
		t.Errorf("invalid status: expected INVALID got: %v", status.Status)
	}
	// Expect 0x0 on INVALID block on top of PoW block
	expected := common.Hash{}
	if !bytes.Equal(status.LatestValidHash[:], expected[:]) {
		t.Fatalf("invalid LVH: got %v want %v", status.LatestValidHash, expected)
	}

	// (3) Now send a payload with unknown parent
	payload = getNewPayload(t, api, commonAncestor, nil, nil)
	payload.ParentHash = common.Hash{1}
	payload = setBlockhash(payload)
	// Now latestValidHash should be the common ancestor
	status, err = api.NewPayloadV1(*payload)
	if err != nil {
		t.Fatal(err)
	}
	if status.Status != engine.SYNCING {
		t.Errorf("invalid status: expected SYNCING got: %v", status.Status)
	}
	if status.LatestValidHash != nil {
		t.Fatalf("invalid LVH: got %v wanted nil", status.LatestValidHash)
	}
}

func getNewEnvelope(t *testing.T, api *ConsensusAPI, parent *types.Header, withdrawals []*types.Withdrawal, beaconRoot *common.Hash) *engine.ExecutionPayloadEnvelope {
	params := engine.PayloadAttributes{
		Timestamp:             parent.Time + 1,
		Random:                crypto.Keccak256Hash([]byte{byte(1)}),
		SuggestedFeeRecipient: parent.Coinbase,
		Withdrawals:           withdrawals,
		BeaconRoot:            beaconRoot,
	}

	envelope, err := assembleEnvelope(api, parent.Hash(), &params)
	if err != nil {
		t.Fatal(err)
	}
	return envelope
}

func getNewPayload(t *testing.T, api *ConsensusAPI, parent *types.Header, withdrawals []*types.Withdrawal, beaconRoot *common.Hash) *engine.ExecutableData {
	return getNewEnvelope(t, api, parent, withdrawals, beaconRoot).ExecutionPayload
}

// setBlockhash sets the blockhash of a modified ExecutableData.
// Can be used to make modified payloads look valid.
func setBlockhash(data *engine.ExecutableData) *engine.ExecutableData {
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
	block := types.NewBlockWithHeader(header).WithBody(types.Body{Transactions: txs})
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
	genesis, preMergeBlocks := generateMergeChain(10, false)
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
	setupBlocks(t, ethserviceA, 10, commonAncestor, func(parent *types.Header) {}, nil, nil)
	commonAncestor = ethserviceA.BlockChain().CurrentBlock()

	var invalidChain []*engine.ExecutableData
	// create a valid payload (P1)
	//payload1 := getNewPayload(t, apiA, commonAncestor)
	//invalidChain = append(invalidChain, payload1)

	// create an invalid payload2 (P2)
	payload2 := getNewPayload(t, apiA, commonAncestor, nil, nil)
	//payload2.ParentHash = payload1.BlockHash
	payload2.GasUsed += 1
	payload2 = setBlockhash(payload2)
	invalidChain = append(invalidChain, payload2)

	head := payload2
	// create some valid payloads on top
	for i := 0; i < 10; i++ {
		payload := getNewPayload(t, apiA, commonAncestor, nil, nil)
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
		if status.Status == engine.VALID {
			t.Error("invalid status: VALID on an invalid chain")
		}
		// Now reorg to the head of the invalid chain
		resp, err := apiB.ForkchoiceUpdatedV1(engine.ForkchoiceStateV1{HeadBlockHash: payload.BlockHash, SafeBlockHash: payload.BlockHash, FinalizedBlockHash: payload.ParentHash}, nil)
		if err != nil {
			t.Fatal(err)
		}
		if resp.PayloadStatus.Status == engine.VALID {
			t.Error("invalid status: VALID on an invalid chain")
		}
		time.Sleep(100 * time.Millisecond)
	}
}

func TestInvalidBloom(t *testing.T) {
	genesis, preMergeBlocks := generateMergeChain(10, false)
	n, ethservice := startEthService(t, genesis, preMergeBlocks)
	defer n.Close()

	commonAncestor := ethservice.BlockChain().CurrentBlock()
	api := NewConsensusAPI(ethservice)

	// Setup 10 blocks on the canonical chain
	setupBlocks(t, ethservice, 10, commonAncestor, func(parent *types.Header) {}, nil, nil)

	// (1) check LatestValidHash by sending a normal payload (P1'')
	payload := getNewPayload(t, api, commonAncestor, nil, nil)
	payload.LogsBloom = append(payload.LogsBloom, byte(1))
	status, err := api.NewPayloadV1(*payload)
	if err != nil {
		t.Fatal(err)
	}
	if status.Status != engine.INVALID {
		t.Errorf("invalid status: expected INVALID got: %v", status.Status)
	}
}

func TestNewPayloadOnInvalidTerminalBlock(t *testing.T) {
	genesis, preMergeBlocks := generateMergeChain(100, false)
	n, ethservice := startEthService(t, genesis, preMergeBlocks)
	defer n.Close()
	api := NewConsensusAPI(ethservice)

	// Test parent already post TTD in FCU
	parent := preMergeBlocks[len(preMergeBlocks)-2]
	fcState := engine.ForkchoiceStateV1{
		HeadBlockHash:      parent.Hash(),
		SafeBlockHash:      common.Hash{},
		FinalizedBlockHash: common.Hash{},
	}
	resp, err := api.ForkchoiceUpdatedV1(fcState, nil)
	if err != nil {
		t.Fatalf("error sending forkchoice, err=%v", err)
	}
	if resp.PayloadStatus != engine.INVALID_TERMINAL_BLOCK {
		t.Fatalf("error sending invalid forkchoice, invalid status: %v", resp.PayloadStatus.Status)
	}

	// Test parent already post TTD in NewPayload
	args := &miner.BuildPayloadArgs{
		Parent:       parent.Hash(),
		Timestamp:    parent.Time() + 1,
		Random:       crypto.Keccak256Hash([]byte{byte(1)}),
		FeeRecipient: parent.Coinbase(),
	}
	payload, err := api.eth.Miner().BuildPayload(args, false)
	if err != nil {
		t.Fatalf("error preparing payload, err=%v", err)
	}
	data := *payload.Resolve().ExecutionPayload
	// We need to recompute the blockhash, since the miner computes a wrong (correct) blockhash
	txs, _ := decodeTransactions(data.Transactions)
	header := &types.Header{
		ParentHash:  data.ParentHash,
		UncleHash:   types.EmptyUncleHash,
		Coinbase:    data.FeeRecipient,
		Root:        data.StateRoot,
		TxHash:      types.DeriveSha(types.Transactions(txs), trie.NewStackTrie(nil)),
		ReceiptHash: data.ReceiptsRoot,
		Bloom:       types.BytesToBloom(data.LogsBloom),
		Difficulty:  common.Big0,
		Number:      new(big.Int).SetUint64(data.Number),
		GasLimit:    data.GasLimit,
		GasUsed:     data.GasUsed,
		Time:        data.Timestamp,
		BaseFee:     data.BaseFeePerGas,
		Extra:       data.ExtraData,
		MixDigest:   data.Random,
	}
	block := types.NewBlockWithHeader(header).WithBody(types.Body{Transactions: txs})
	data.BlockHash = block.Hash()
	// Send the new payload
	resp2, err := api.NewPayloadV1(data)
	if err != nil {
		t.Fatalf("error sending NewPayload, err=%v", err)
	}
	if resp2 != engine.INVALID_TERMINAL_BLOCK {
		t.Fatalf("error sending invalid forkchoice, invalid status: %v", resp.PayloadStatus.Status)
	}
}

// TestSimultaneousNewBlock does several parallel inserts, both as
// newPayLoad and forkchoiceUpdate. This is to test that the api behaves
// well even of the caller is not being 'serial'.
func TestSimultaneousNewBlock(t *testing.T) {
	genesis, preMergeBlocks := generateMergeChain(10, false)
	n, ethservice := startEthService(t, genesis, preMergeBlocks)
	defer n.Close()

	var (
		api    = NewConsensusAPI(ethservice)
		parent = preMergeBlocks[len(preMergeBlocks)-1]
	)
	for i := 0; i < 10; i++ {
		execData, err := assembleBlock(api, parent.Hash(), &engine.PayloadAttributes{
			Timestamp: parent.Time() + 5,
		})
		if err != nil {
			t.Fatalf("Failed to create the executable data %v", err)
		}
		// Insert it 10 times in parallel. Should be ignored.
		{
			var (
				wg      sync.WaitGroup
				testErr error
				errMu   sync.Mutex
			)
			wg.Add(10)
			for ii := 0; ii < 10; ii++ {
				go func() {
					defer wg.Done()
					if newResp, err := api.NewPayloadV1(*execData); err != nil {
						errMu.Lock()
						testErr = fmt.Errorf("failed to insert block: %w", err)
						errMu.Unlock()
					} else if newResp.Status != "VALID" {
						errMu.Lock()
						testErr = fmt.Errorf("failed to insert block: %v", newResp.Status)
						errMu.Unlock()
					}
				}()
			}
			wg.Wait()
			if testErr != nil {
				t.Fatal(testErr)
			}
		}
		block, err := engine.ExecutableDataToBlock(*execData, nil, nil, nil)
		if err != nil {
			t.Fatalf("Failed to convert executable data to block %v", err)
		}
		if ethservice.BlockChain().CurrentBlock().Number.Uint64() != block.NumberU64()-1 {
			t.Fatalf("Chain head shouldn't be updated")
		}
		fcState := engine.ForkchoiceStateV1{
			HeadBlockHash:      block.Hash(),
			SafeBlockHash:      block.Hash(),
			FinalizedBlockHash: block.Hash(),
		}
		{
			var (
				wg      sync.WaitGroup
				testErr error
				errMu   sync.Mutex
			)
			wg.Add(10)
			// Do each FCU 10 times
			for ii := 0; ii < 10; ii++ {
				go func() {
					defer wg.Done()
					if _, err := api.ForkchoiceUpdatedV1(fcState, nil); err != nil {
						errMu.Lock()
						testErr = fmt.Errorf("failed to insert block: %w", err)
						errMu.Unlock()
					}
				}()
			}
			wg.Wait()
			if testErr != nil {
				t.Fatal(testErr)
			}
		}
		if have, want := ethservice.BlockChain().CurrentBlock().Number.Uint64(), block.NumberU64(); have != want {
			t.Fatalf("Chain head should be updated, have %d want %d", have, want)
		}
		parent = block
	}
}

// TestWithdrawals creates and verifies two post-Shanghai blocks. The first
// includes zero withdrawals and the second includes two.
func TestWithdrawals(t *testing.T) {
	genesis, blocks := generateMergeChain(10, true)
	// Set shanghai time to last block + 5 seconds (first post-merge block)
	time := blocks[len(blocks)-1].Time() + 5
	genesis.Config.ShanghaiTime = &time

	n, ethservice := startEthService(t, genesis, blocks)
	defer n.Close()

	api := NewConsensusAPI(ethservice)

	// 10: Build Shanghai block with no withdrawals.
	parent := ethservice.BlockChain().CurrentHeader()
	blockParams := engine.PayloadAttributes{
		Timestamp:   parent.Time + 5,
		Withdrawals: make([]*types.Withdrawal, 0),
	}
	fcState := engine.ForkchoiceStateV1{
		HeadBlockHash: parent.Hash(),
	}
	resp, err := api.ForkchoiceUpdatedV2(fcState, &blockParams)
	if err != nil {
		t.Fatalf("error preparing payload, err=%v", err)
	}
	if resp.PayloadStatus.Status != engine.VALID {
		t.Fatalf("unexpected status (got: %s, want: %s)", resp.PayloadStatus.Status, engine.VALID)
	}

	// 10: verify state root is the same as parent
	payloadID := (&miner.BuildPayloadArgs{
		Parent:       fcState.HeadBlockHash,
		Timestamp:    blockParams.Timestamp,
		FeeRecipient: blockParams.SuggestedFeeRecipient,
		Random:       blockParams.Random,
		Withdrawals:  blockParams.Withdrawals,
		BeaconRoot:   blockParams.BeaconRoot,
		Version:      engine.PayloadV2,
	}).Id()
	execData, err := api.GetPayloadV2(payloadID)
	if err != nil {
		t.Fatalf("error getting payload, err=%v", err)
	}
	if execData.ExecutionPayload.StateRoot != parent.Root {
		t.Fatalf("mismatch state roots (got: %s, want: %s)", execData.ExecutionPayload.StateRoot, blocks[8].Root())
	}

	// 10: verify locally built block
	if status, err := api.NewPayloadV2(*execData.ExecutionPayload); err != nil {
		t.Fatalf("error validating payload: %v", err)
	} else if status.Status != engine.VALID {
		t.Fatalf("invalid payload")
	}

	// 11: build shanghai block with withdrawal
	aa := common.Address{0xaa}
	bb := common.Address{0xbb}
	blockParams = engine.PayloadAttributes{
		Timestamp: execData.ExecutionPayload.Timestamp + 5,
		Withdrawals: []*types.Withdrawal{
			{
				Index:   0,
				Address: aa,
				Amount:  32,
			},
			{
				Index:   1,
				Address: bb,
				Amount:  33,
			},
		},
	}
	fcState.HeadBlockHash = execData.ExecutionPayload.BlockHash
	_, err = api.ForkchoiceUpdatedV2(fcState, &blockParams)
	if err != nil {
		t.Fatalf("error preparing payload, err=%v", err)
	}

	// 11: verify locally build block.
	payloadID = (&miner.BuildPayloadArgs{
		Parent:       fcState.HeadBlockHash,
		Timestamp:    blockParams.Timestamp,
		FeeRecipient: blockParams.SuggestedFeeRecipient,
		Random:       blockParams.Random,
		Withdrawals:  blockParams.Withdrawals,
		BeaconRoot:   blockParams.BeaconRoot,
		Version:      engine.PayloadV2,
	}).Id()
	execData, err = api.GetPayloadV2(payloadID)
	if err != nil {
		t.Fatalf("error getting payload, err=%v", err)
	}
	if status, err := api.NewPayloadV2(*execData.ExecutionPayload); err != nil {
		t.Fatalf("error validating payload: %v", err)
	} else if status.Status != engine.VALID {
		t.Fatalf("invalid payload")
	}

	// 11: set block as head.
	fcState.HeadBlockHash = execData.ExecutionPayload.BlockHash
	_, err = api.ForkchoiceUpdatedV2(fcState, nil)
	if err != nil {
		t.Fatalf("error preparing payload, err=%v", err)
	}

	// 11: verify withdrawals were processed.
	db, _, err := ethservice.APIBackend.StateAndHeaderByNumber(context.Background(), rpc.BlockNumber(execData.ExecutionPayload.Number))
	if err != nil {
		t.Fatalf("unable to load db: %v", err)
	}
	for i, w := range blockParams.Withdrawals {
		// w.Amount is in gwei, balance in wei
		if db.GetBalance(w.Address).Uint64() != w.Amount*params.GWei {
			t.Fatalf("failed to process withdrawal %d", i)
		}
	}
}

func TestNilWithdrawals(t *testing.T) {
	genesis, blocks := generateMergeChain(10, true)
	// Set shanghai time to last block + 4 seconds (first post-merge block)
	time := blocks[len(blocks)-1].Time() + 4
	genesis.Config.ShanghaiTime = &time

	n, ethservice := startEthService(t, genesis, blocks)
	defer n.Close()

	api := NewConsensusAPI(ethservice)
	parent := ethservice.BlockChain().CurrentHeader()
	aa := common.Address{0xaa}

	type test struct {
		blockParams engine.PayloadAttributes
		wantErr     bool
	}
	tests := []test{
		// Before Shanghai
		{
			blockParams: engine.PayloadAttributes{
				Timestamp:   parent.Time + 2,
				Withdrawals: nil,
			},
			wantErr: false,
		},
		{
			blockParams: engine.PayloadAttributes{
				Timestamp:   parent.Time + 2,
				Withdrawals: make([]*types.Withdrawal, 0),
			},
			wantErr: true,
		},
		{
			blockParams: engine.PayloadAttributes{
				Timestamp: parent.Time + 2,
				Withdrawals: []*types.Withdrawal{
					{
						Index:   0,
						Address: aa,
						Amount:  32,
					},
				},
			},
			wantErr: true,
		},
		// After Shanghai
		{
			blockParams: engine.PayloadAttributes{
				Timestamp:   parent.Time + 5,
				Withdrawals: nil,
			},
			wantErr: true,
		},
		{
			blockParams: engine.PayloadAttributes{
				Timestamp:   parent.Time + 5,
				Withdrawals: make([]*types.Withdrawal, 0),
			},
			wantErr: false,
		},
		{
			blockParams: engine.PayloadAttributes{
				Timestamp: parent.Time + 5,
				Withdrawals: []*types.Withdrawal{
					{
						Index:   0,
						Address: aa,
						Amount:  32,
					},
				},
			},
			wantErr: false,
		},
	}

	fcState := engine.ForkchoiceStateV1{
		HeadBlockHash: parent.Hash(),
	}

	for _, test := range tests {
		var (
			err            error
			payloadVersion engine.PayloadVersion
			shanghai       = genesis.Config.IsShanghai(genesis.Config.LondonBlock, test.blockParams.Timestamp)
		)
		if !shanghai {
			payloadVersion = engine.PayloadV1
			_, err = api.ForkchoiceUpdatedV1(fcState, &test.blockParams)
		} else {
			payloadVersion = engine.PayloadV2
			_, err = api.ForkchoiceUpdatedV2(fcState, &test.blockParams)
		}
		if test.wantErr {
			if err == nil {
				t.Fatal("wanted error on fcuv2 with invalid withdrawals")
			}
			continue
		}
		if err != nil {
			t.Fatalf("error preparing payload, err=%v", err)
		}

		// 11: verify locally build block.
		payloadID := (&miner.BuildPayloadArgs{
			Parent:       fcState.HeadBlockHash,
			Timestamp:    test.blockParams.Timestamp,
			FeeRecipient: test.blockParams.SuggestedFeeRecipient,
			Random:       test.blockParams.Random,
			Version:      payloadVersion,
		}).Id()
		execData, err := api.GetPayloadV2(payloadID)
		if err != nil {
			t.Fatalf("error getting payload, err=%v", err)
		}
		var status engine.PayloadStatusV1
		if !shanghai {
			status, err = api.NewPayloadV1(*execData.ExecutionPayload)
		} else {
			status, err = api.NewPayloadV2(*execData.ExecutionPayload)
		}
		if err != nil {
			t.Fatalf("error validating payload: %v", err.(*engine.EngineAPIError).ErrorData())
		} else if status.Status != engine.VALID {
			t.Fatalf("invalid payload")
		}
	}
}

func setupBodies(t *testing.T) (*node.Node, *eth.Ethereum, []*types.Block) {
	genesis, blocks := generateMergeChain(10, true)

	// Enable next forks on the last block.
	time := blocks[len(blocks)-1].Header().Time + 1
	genesis.Config.ShanghaiTime = &time
	genesis.Config.CancunTime = &time
	genesis.Config.PragueTime = &time

	n, ethservice := startEthService(t, genesis, blocks)

	var (
		// This EVM code generates a log when the contract is created.
		logCode = common.Hex2Bytes("60606040525b7f24ec1d3ff24c2f6ff210738839dbc339cd45a5294d85c79361016243157aae7b60405180905060405180910390a15b600a8060416000396000f360606040526008565b00")
		parent  = ethservice.BlockChain().CurrentBlock()
	)

	// Each block, this callback will include two txs that generate body values like logs and requests.
	callback := func(parent *types.Header) {
		var (
			statedb, _ = ethservice.BlockChain().StateAt(parent.Root)
			// Create tx to trigger log generator.
			tx1, _ = types.SignTx(types.NewContractCreation(statedb.GetNonce(testAddr), new(big.Int), 1000000, big.NewInt(2*params.InitialBaseFee), logCode), types.LatestSigner(ethservice.BlockChain().Config()), testKey)
			// Create tx to trigger deposit generator.
			tx2, _ = types.SignTx(types.NewTransaction(statedb.GetNonce(testAddr)+1, ethservice.APIBackend.ChainConfig().DepositContractAddress, new(big.Int), 500000, big.NewInt(2*params.InitialBaseFee), nil), types.LatestSigner(ethservice.BlockChain().Config()), testKey)
		)
		ethservice.TxPool().Add([]*types.Transaction{tx1}, false, false)
		ethservice.TxPool().Add([]*types.Transaction{tx2}, false, false)
	}

	// Make some withdrawals to include.
	withdrawals := make([][]*types.Withdrawal, 10)
	withdrawals[0] = nil // should be filtered out by miner
	withdrawals[1] = make([]*types.Withdrawal, 0)
	for i := 2; i < len(withdrawals); i++ {
		addr := make([]byte, 20)
		crand.Read(addr)
		withdrawals[i] = []*types.Withdrawal{
			{Index: rand.Uint64(), Validator: rand.Uint64(), Amount: rand.Uint64(), Address: common.BytesToAddress(addr)},
		}
	}

	// Make beacon root update for each block.
	beaconRoots := make([]common.Hash, 10)
	for i := 0; i < 10; i++ {
		beaconRoots[i] = common.Hash{byte(i)}
	}

	// Create the blocks.
	newHeaders := setupBlocks(t, ethservice, 10, parent, callback, withdrawals, beaconRoots)
	newBlocks := make([]*types.Block, len(newHeaders))
	for i, header := range newHeaders {
		newBlocks[i] = ethservice.BlockChain().GetBlock(header.Hash(), header.Number.Uint64())
	}

	return n, ethservice, append(blocks, newBlocks...)
}

func allHashes(blocks []*types.Block) []common.Hash {
	var hashes []common.Hash
	for _, b := range blocks {
		hashes = append(hashes, b.Hash())
	}
	return hashes
}
func allBodies(blocks []*types.Block) []*types.Body {
	var bodies []*types.Body
	for _, b := range blocks {
		bodies = append(bodies, b.Body())
	}
	return bodies
}

func TestGetBlockBodiesByHash(t *testing.T) {
	node, eth, blocks := setupBodies(t)
	api := NewConsensusAPI(eth)
	defer node.Close()

	tests := []struct {
		results []*types.Body
		hashes  []common.Hash
	}{
		// First pow block
		{
			results: []*types.Body{eth.BlockChain().GetBlockByNumber(0).Body()},
			hashes:  []common.Hash{eth.BlockChain().GetBlockByNumber(0).Hash()},
		},
		// Last pow block
		{
			results: []*types.Body{blocks[9].Body()},
			hashes:  []common.Hash{blocks[9].Hash()},
		},
		// First post-merge block
		{
			results: []*types.Body{blocks[10].Body()},
			hashes:  []common.Hash{blocks[10].Hash()},
		},
		// Pre & post merge blocks
		{
			results: []*types.Body{blocks[0].Body(), blocks[9].Body(), blocks[14].Body()},
			hashes:  []common.Hash{blocks[0].Hash(), blocks[9].Hash(), blocks[14].Hash()},
		},
		// unavailable block
		{
			results: []*types.Body{blocks[0].Body(), nil, blocks[14].Body()},
			hashes:  []common.Hash{blocks[0].Hash(), {1, 2}, blocks[14].Hash()},
		},
		// same block multiple times
		{
			results: []*types.Body{blocks[0].Body(), nil, blocks[0].Body(), blocks[0].Body()},
			hashes:  []common.Hash{blocks[0].Hash(), {1, 2}, blocks[0].Hash(), blocks[0].Hash()},
		},
		// all blocks
		{
			results: allBodies(blocks),
			hashes:  allHashes(blocks),
		},
	}

	for k, test := range tests {
		result := api.GetPayloadBodiesByHashV2(test.hashes)
		for i, r := range result {
			if err := checkEqualBody(test.results[i], r); err != nil {
				t.Fatalf("test %v: invalid response: %v\nexpected %+v\ngot %+v", k, err, test.results[i], r)
			}
		}
	}
}

func TestGetBlockBodiesByRange(t *testing.T) {
	node, eth, blocks := setupBodies(t)
	api := NewConsensusAPI(eth)
	defer node.Close()

	tests := []struct {
		results []*types.Body
		start   hexutil.Uint64
		count   hexutil.Uint64
	}{
		{
			results: []*types.Body{blocks[9].Body()},
			start:   10,
			count:   1,
		},
		// Genesis
		{
			results: []*types.Body{blocks[0].Body()},
			start:   1,
			count:   1,
		},
		// First post-merge block
		{
			results: []*types.Body{blocks[9].Body()},
			start:   10,
			count:   1,
		},
		// Pre & post merge blocks
		{
			results: []*types.Body{blocks[7].Body(), blocks[8].Body(), blocks[9].Body(), blocks[10].Body()},
			start:   8,
			count:   4,
		},
		// unavailable block
		{
			results: []*types.Body{blocks[18].Body(), blocks[19].Body()},
			start:   19,
			count:   3,
		},
		// unavailable block
		{
			results: []*types.Body{blocks[19].Body()},
			start:   20,
			count:   2,
		},
		{
			results: []*types.Body{blocks[19].Body()},
			start:   20,
			count:   1,
		},
		// whole range unavailable
		{
			results: make([]*types.Body, 0),
			start:   22,
			count:   2,
		},
		// allBlocks
		{
			results: allBodies(blocks),
			start:   1,
			count:   hexutil.Uint64(len(blocks)),
		},
	}

	for k, test := range tests {
		result, err := api.GetPayloadBodiesByRangeV2(test.start, test.count)
		if err != nil {
			t.Fatal(err)
		}
		if len(result) == len(test.results) {
			for i, r := range result {
				if err := checkEqualBody(test.results[i], r); err != nil {
					t.Fatalf("test %d: invalid response: %v\nexpected %+v\ngot %+v", k, err, test.results[i], r)
				}
			}
		} else {
			t.Fatalf("test %d: invalid length want %v got %v", k, len(test.results), len(result))
		}
	}
}

func TestGetBlockBodiesByRangeInvalidParams(t *testing.T) {
	node, eth, _ := setupBodies(t)
	api := NewConsensusAPI(eth)
	defer node.Close()
	tests := []struct {
		start hexutil.Uint64
		count hexutil.Uint64
		want  *engine.EngineAPIError
	}{
		// Genesis
		{
			start: 0,
			count: 1,
			want:  engine.InvalidParams,
		},
		// No block requested
		{
			start: 1,
			count: 0,
			want:  engine.InvalidParams,
		},
		// Genesis & no block
		{
			start: 0,
			count: 0,
			want:  engine.InvalidParams,
		},
		// More than 1024 blocks
		{
			start: 1,
			count: 1025,
			want:  engine.TooLargeRequest,
		},
	}
	for i, tc := range tests {
		result, err := api.GetPayloadBodiesByRangeV2(tc.start, tc.count)
		if err == nil {
			t.Fatalf("test %d: expected error, got %v", i, result)
		}
		if have, want := err.Error(), tc.want.Error(); have != want {
			t.Fatalf("test %d: have %s, want %s", i, have, want)
		}
	}
}

func checkEqualBody(a *types.Body, b *engine.ExecutionPayloadBody) error {
	if a == nil && b == nil {
		return nil
	} else if a == nil || b == nil {
		return errors.New("nil vs. non-nil")
	}
	if len(a.Transactions) != len(b.TransactionData) {
		return errors.New("transactions length mismatch")
	}
	for i, tx := range a.Transactions {
		data, _ := tx.MarshalBinary()
		if !bytes.Equal(data, b.TransactionData[i]) {
			return fmt.Errorf("transaction %d mismatch", i)
		}
	}
	if !reflect.DeepEqual(a.Withdrawals, b.Withdrawals) {
		return fmt.Errorf("withdrawals mismatch")
	}
	return nil
}

func TestBlockToPayloadWithBlobs(t *testing.T) {
	header := types.Header{}
	var txs []*types.Transaction

	inner := types.BlobTx{
		BlobHashes: make([]common.Hash, 1),
	}

	txs = append(txs, types.NewTx(&inner))
	sidecars := []*types.BlobTxSidecar{
		{
			Blobs:       make([]kzg4844.Blob, 1),
			Commitments: make([]kzg4844.Commitment, 1),
			Proofs:      make([]kzg4844.Proof, 1),
		},
	}

	block := types.NewBlock(&header, &types.Body{Transactions: txs}, nil, trie.NewStackTrie(nil))
	envelope := engine.BlockToExecutableData(block, nil, sidecars, nil)
	var want int
	for _, tx := range txs {
		want += len(tx.BlobHashes())
	}
	if got := len(envelope.BlobsBundle.Commitments); got != want {
		t.Fatalf("invalid number of commitments: got %v, want %v", got, want)
	}
	if got := len(envelope.BlobsBundle.Proofs); got != want {
		t.Fatalf("invalid number of proofs: got %v, want %v", got, want)
	}
	if got := len(envelope.BlobsBundle.Blobs); got != want {
		t.Fatalf("invalid number of blobs: got %v, want %v", got, want)
	}
	_, err := engine.ExecutableDataToBlock(*envelope.ExecutionPayload, make([]common.Hash, 1), nil, nil)
	if err != nil {
		t.Error(err)
	}
}

// This checks that beaconRoot is applied to the state from the engine API.
func TestParentBeaconBlockRoot(t *testing.T) {
	//log.SetDefault(log.NewLogger(log.NewTerminalHandlerWithLevel(colorable.NewColorableStderr(), log.LevelTrace, true)))

	genesis, blocks := generateMergeChain(10, true)

	// Set cancun time to last block + 5 seconds
	time := blocks[len(blocks)-1].Time() + 5
	genesis.Config.ShanghaiTime = &time
	genesis.Config.CancunTime = &time

	n, ethservice := startEthService(t, genesis, blocks)
	defer n.Close()

	api := NewConsensusAPI(ethservice)

	// 11: Build Shanghai block with no withdrawals.
	parent := ethservice.BlockChain().CurrentHeader()
	blockParams := engine.PayloadAttributes{
		Timestamp:   parent.Time + 5,
		Withdrawals: make([]*types.Withdrawal, 0),
		BeaconRoot:  &common.Hash{42},
	}
	fcState := engine.ForkchoiceStateV1{
		HeadBlockHash: parent.Hash(),
	}
	resp, err := api.ForkchoiceUpdatedV3(fcState, &blockParams)
	if err != nil {
		t.Fatalf("error preparing payload, err=%v", err.(*engine.EngineAPIError).ErrorData())
	}
	if resp.PayloadStatus.Status != engine.VALID {
		t.Fatalf("unexpected status (got: %s, want: %s)", resp.PayloadStatus.Status, engine.VALID)
	}

	// 11: verify state root is the same as parent
	payloadID := (&miner.BuildPayloadArgs{
		Parent:       fcState.HeadBlockHash,
		Timestamp:    blockParams.Timestamp,
		FeeRecipient: blockParams.SuggestedFeeRecipient,
		Random:       blockParams.Random,
		Withdrawals:  blockParams.Withdrawals,
		BeaconRoot:   blockParams.BeaconRoot,
		Version:      engine.PayloadV3,
	}).Id()
	execData, err := api.GetPayloadV3(payloadID)
	if err != nil {
		t.Fatalf("error getting payload, err=%v", err)
	}

	// 11: verify locally built block
	if status, err := api.NewPayloadV3(*execData.ExecutionPayload, []common.Hash{}, &common.Hash{42}); err != nil {
		t.Fatalf("error validating payload: %v", err)
	} else if status.Status != engine.VALID {
		t.Fatalf("invalid payload")
	}

	fcState.HeadBlockHash = execData.ExecutionPayload.BlockHash
	resp, err = api.ForkchoiceUpdatedV3(fcState, nil)
	if err != nil {
		t.Fatalf("error preparing payload, err=%v", err.(*engine.EngineAPIError).ErrorData())
	}
	if resp.PayloadStatus.Status != engine.VALID {
		t.Fatalf("unexpected status (got: %s, want: %s)", resp.PayloadStatus.Status, engine.VALID)
	}

	// 11: verify beacon root was processed.
	db, _, err := ethservice.APIBackend.StateAndHeaderByNumber(context.Background(), rpc.BlockNumber(execData.ExecutionPayload.Number))
	if err != nil {
		t.Fatalf("unable to load db: %v", err)
	}
	var (
		timeIdx = common.BigToHash(big.NewInt(int64(execData.ExecutionPayload.Timestamp % 98304)))
		rootIdx = common.BigToHash(big.NewInt(int64((execData.ExecutionPayload.Timestamp % 98304) + 98304)))
	)

	if num := db.GetState(params.BeaconRootsAddress, timeIdx); num != timeIdx {
		t.Fatalf("incorrect number stored: want %s, got %s", timeIdx, num)
	}
	if root := db.GetState(params.BeaconRootsAddress, rootIdx); root != *blockParams.BeaconRoot {
		t.Fatalf("incorrect root stored: want %s, got %s", *blockParams.BeaconRoot, root)
	}
}

func TestWitnessCreationAndConsumption(t *testing.T) {
	//log.SetDefault(log.NewLogger(log.NewTerminalHandlerWithLevel(colorable.NewColorableStderr(), log.LevelTrace, true)))

	genesis, blocks := generateMergeChain(10, true)

	// Set cancun time to semi-last block + 5 seconds
	timestamp := blocks[len(blocks)-2].Time() + 5
	genesis.Config.ShanghaiTime = &timestamp
	genesis.Config.CancunTime = &timestamp

	n, ethservice := startEthService(t, genesis, blocks[:9])
	defer n.Close()

	api := NewConsensusAPI(ethservice)

	// Put the 10th block's tx in the pool and produce a new block
	txs := blocks[9].Transactions()

	ethservice.TxPool().Add(txs, true, true)
	blockParams := engine.PayloadAttributes{
		Timestamp:   blocks[8].Time() + 5,
		Withdrawals: make([]*types.Withdrawal, 0),
		BeaconRoot:  &common.Hash{42},
	}
	fcState := engine.ForkchoiceStateV1{
		HeadBlockHash:      blocks[8].Hash(),
		SafeBlockHash:      common.Hash{},
		FinalizedBlockHash: common.Hash{},
	}
	_, err := api.ForkchoiceUpdatedWithWitnessV3(fcState, &blockParams)
	if err != nil {
		t.Fatalf("error preparing payload, err=%v", err)
	}
	// Give the payload some time to be built
	time.Sleep(100 * time.Millisecond)

	payloadID := (&miner.BuildPayloadArgs{
		Parent:       fcState.HeadBlockHash,
		Timestamp:    blockParams.Timestamp,
		FeeRecipient: blockParams.SuggestedFeeRecipient,
		Random:       blockParams.Random,
		Withdrawals:  blockParams.Withdrawals,
		BeaconRoot:   blockParams.BeaconRoot,
		Version:      engine.PayloadV3,
	}).Id()
	envelope, err := api.GetPayloadV3(payloadID)
	if err != nil {
		t.Fatalf("error getting payload, err=%v", err)
	}
	if len(envelope.ExecutionPayload.Transactions) != blocks[9].Transactions().Len() {
		t.Fatalf("invalid number of transactions %d != %d", len(envelope.ExecutionPayload.Transactions), blocks[9].Transactions().Len())
	}
	if envelope.Witness == nil {
		t.Fatalf("witness missing from payload")
	}
	// Test stateless execution of the created witness
	wantStateRoot := envelope.ExecutionPayload.StateRoot
	wantReceiptRoot := envelope.ExecutionPayload.ReceiptsRoot

	envelope.ExecutionPayload.StateRoot = common.Hash{}
	envelope.ExecutionPayload.ReceiptsRoot = common.Hash{}

	res, err := api.ExecuteStatelessPayloadV3(*envelope.ExecutionPayload, []common.Hash{}, &common.Hash{42}, *envelope.Witness)
	if err != nil {
		t.Fatalf("error executing stateless payload witness: %v", err)
	}
	if res.StateRoot != wantStateRoot {
		t.Fatalf("stateless state root mismatch: have %v, want %v", res.StateRoot, wantStateRoot)
	}
	if res.ReceiptsRoot != wantReceiptRoot {
		t.Fatalf("stateless receipt root mismatch: have %v, want %v", res.ReceiptsRoot, wantReceiptRoot)
	}
	// Test block insertion with witness creation
	envelope.ExecutionPayload.StateRoot = wantStateRoot
	envelope.ExecutionPayload.ReceiptsRoot = wantReceiptRoot

	res2, err := api.NewPayloadWithWitnessV3(*envelope.ExecutionPayload, []common.Hash{}, &common.Hash{42})
	if err != nil {
		t.Fatalf("error executing stateless payload witness: %v", err)
	}
	if res2.Witness == nil {
		t.Fatalf("witness missing from payload")
	}
	// Test stateless execution of the created witness
	wantStateRoot = envelope.ExecutionPayload.StateRoot
	wantReceiptRoot = envelope.ExecutionPayload.ReceiptsRoot

	envelope.ExecutionPayload.StateRoot = common.Hash{}
	envelope.ExecutionPayload.ReceiptsRoot = common.Hash{}

	res, err = api.ExecuteStatelessPayloadV3(*envelope.ExecutionPayload, []common.Hash{}, &common.Hash{42}, *res2.Witness)
	if err != nil {
		t.Fatalf("error executing stateless payload witness: %v", err)
	}
	if res.StateRoot != wantStateRoot {
		t.Fatalf("stateless state root mismatch: have %v, want %v", res.StateRoot, wantStateRoot)
	}
	if res.ReceiptsRoot != wantReceiptRoot {
		t.Fatalf("stateless receipt root mismatch: have %v, want %v", res.ReceiptsRoot, wantReceiptRoot)
	}
}

// TestGetClientVersion verifies the expected version info is returned.
func TestGetClientVersion(t *testing.T) {
	genesis, preMergeBlocks := generateMergeChain(10, false)
	n, ethservice := startEthService(t, genesis, preMergeBlocks)
	defer n.Close()

	api := NewConsensusAPI(ethservice)
	info := engine.ClientVersionV1{
		Code:    "TT",
		Name:    "test",
		Version: "1.1.1",
		Commit:  "0x12345678",
	}
	infos := api.GetClientVersionV1(info)
	if len(infos) != 1 {
		t.Fatalf("expected only one returned client version, got %d", len(infos))
	}
	info = infos[0]
	if info.Code != engine.ClientCode || info.Name != engine.ClientName || info.Version != version.WithMeta {
		t.Fatalf("client info does match expected, got %s", info.String())
	}
}
