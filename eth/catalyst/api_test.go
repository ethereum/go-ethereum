// Copyright 2020 The go-ethereum Authors
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
	"math/big"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/consensus/ethash"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/eth"
	"github.com/ethereum/go-ethereum/eth/ethconfig"
	"github.com/ethereum/go-ethereum/node"
	"github.com/ethereum/go-ethereum/params"
)

var (
	// testKey is a private key to use for funding a tester account.
	testKey, _ = crypto.HexToECDSA("b71c71a67e1177ad4e901695e1b4b9ee17ae16c6668d313eac2f96dbcda3f291")

	// testAddr is the Ethereum address of the tester account.
	testAddr = crypto.PubkeyToAddress(testKey.PublicKey)

	testBalance = big.NewInt(2e18)
)

func generatePreMergeChain(n int) (*core.Genesis, []*types.Block) {
	db := rawdb.NewMemoryDatabase()
	config := params.AllEthashProtocolChanges
	genesis := &core.Genesis{
		Config:    config,
		Alloc:     core.GenesisAlloc{testAddr: {Balance: testBalance}},
		ExtraData: []byte("test genesis"),
		Timestamp: 9000,
		BaseFee:   big.NewInt(params.InitialBaseFee),
	}
	testNonce := uint64(0)
	generate := func(i int, g *core.BlockGen) {
		g.OffsetTime(5)
		g.SetExtra([]byte("test"))
		tx, _ := types.SignTx(types.NewTransaction(testNonce, common.HexToAddress("0x9a9070028361F7AAbeB3f2F2Dc07F82C4a98A02a"), big.NewInt(1), params.TxGas, big.NewInt(params.InitialBaseFee*2), nil), types.LatestSigner(config), testKey)
		g.AddTx(tx)
		testNonce++
	}
	gblock := genesis.ToBlock(db)
	engine := ethash.NewFaker()
	blocks, _ := core.GenerateChain(config, gblock, engine, db, n, generate)
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

	api := NewConsensusAPI(ethservice, nil)
	signer := types.NewEIP155Signer(ethservice.BlockChain().Config().ChainID)
	tx, err := types.SignTx(types.NewTransaction(uint64(10), blocks[9].Coinbase(), big.NewInt(1000), params.TxGas, big.NewInt(params.InitialBaseFee), nil), signer, testKey)
	if err != nil {
		t.Fatalf("error signing transaction, err=%v", err)
	}
	ethservice.TxPool().AddLocal(tx)
	blockParams := AssembleBlockParams{
		ParentHash: blocks[9].Hash(),
		Timestamp:  blocks[9].Time() + 5,
	}
	execData, err := api.assembleBlock(blockParams)
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

	api := NewConsensusAPI(ethservice, nil)

	// Put the 10th block's tx in the pool and produce a new block
	api.insertTransactions(blocks[9].Transactions())
	blockParams := AssembleBlockParams{
		ParentHash: blocks[8].Hash(),
		Timestamp:  blocks[8].Time() + 5,
	}
	execData, err := api.assembleBlock(blockParams)
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

	api := NewConsensusAPI(ethservice, nil)

	if err := api.ForkchoiceUpdated(ForkChoiceParams{HeadBlockHash: blocks[5].Hash()}); err == nil {
		t.Errorf("fork choice updated before total terminal difficulty should fail")
	}
}

func TestEth2PrepareAndGetPayload(t *testing.T) {
	genesis, blocks := generatePreMergeChain(10)
	n, ethservice := startEthService(t, genesis, blocks[:9])
	defer n.Close()

	api := NewConsensusAPI(ethservice, nil)

	// Put the 10th block's tx in the pool and produce a new block
	api.insertTransactions(blocks[9].Transactions())
	blockParams := AssembleBlockParams{
		ParentHash: blocks[8].Hash(),
		Timestamp:  blocks[8].Time() + 5,
	}
	respID, err := api.PreparePayload(blockParams)
	if err != nil {
		t.Fatalf("error preparing payload, err=%v", err)
	}
	execData, err := api.GetPayload(hexutil.Uint64(respID.PayloadID))
	if err != nil {
		t.Fatalf("error getting payload, err=%v", err)
	}
	if len(execData.Transactions) != blocks[9].Transactions().Len() {
		t.Fatalf("invalid number of transactions %d != 1", len(execData.Transactions))
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

func TestEth2NewBlock(t *testing.T) {
	genesis, preMergeBlocks := generatePreMergeChain(10)
	n, ethservice := startEthService(t, genesis, preMergeBlocks)
	ethservice.Merger().ReachTTD()
	defer n.Close()

	var (
		api    = NewConsensusAPI(ethservice, nil)
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

		execData, err := api.assembleBlock(AssembleBlockParams{
			ParentHash: parent.Hash(),
			Timestamp:  parent.Time() + 5,
		})
		if err != nil {
			t.Fatalf("Failed to create the executable data %v", err)
		}
		block, err := ExecutableDataToBlock(*execData)
		if err != nil {
			t.Fatalf("Failed to convert executable data to block %v", err)
		}
		newResp, err := api.ExecutePayload(*execData)
		if err != nil || newResp.Status != "VALID" {
			t.Fatalf("Failed to insert block: %v", err)
		}
		if ethservice.BlockChain().CurrentBlock().NumberU64() != block.NumberU64()-1 {
			t.Fatalf("Chain head shouldn't be updated")
		}
		checkLogEvents(t, newLogCh, rmLogsCh, 0, 0)

		if err := api.ForkchoiceUpdated(ForkChoiceParams{HeadBlockHash: block.Hash(), FinalizedBlockHash: block.Hash()}); err != nil {
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
		execData, err := api.assembleBlock(AssembleBlockParams{
			ParentHash: parent.Hash(),
			Timestamp:  parent.Time() + 6,
		})
		if err != nil {
			t.Fatalf("Failed to create the executable data %v", err)
		}
		block, err := ExecutableDataToBlock(*execData)
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

		if err := api.ConsensusValidated(ConsensusValidatedParams{BlockHash: block.Hash(), Status: "VALID"}); err != nil {
			t.Fatalf("Failed to insert block: %v", err)
		}
		if err := api.ForkchoiceUpdated(ForkChoiceParams{FinalizedBlockHash: block.Hash(), HeadBlockHash: block.Hash()}); err != nil {
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

	n, err := node.New(&node.Config{})
	if err != nil {
		t.Fatal("can't create node:", err)
	}

	ethcfg := &ethconfig.Config{Genesis: genesis, Ethash: ethash.Config{PowMode: ethash.ModeFake}, TrieTimeout: time.Minute, TrieDirtyCache: 256, TrieCleanCache: 256}
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
	ethservice.SetEtherbase(testAddr)
	ethservice.SetSynced()

	return n, ethservice
}

func TestFullAPI(t *testing.T) {
	genesis, preMergeBlocks := generatePreMergeChain(10)
	n, ethservice := startEthService(t, genesis, preMergeBlocks)
	ethservice.Merger().ReachTTD()
	defer n.Close()
	var (
		api    = NewConsensusAPI(ethservice, nil)
		parent = ethservice.BlockChain().CurrentBlock()
		// This EVM code generates a log when the contract is created.
		logCode = common.Hex2Bytes("60606040525b7f24ec1d3ff24c2f6ff210738839dbc339cd45a5294d85c79361016243157aae7b60405180905060405180910390a15b600a8060416000396000f360606040526008565b00")
	)
	for i := 0; i < 10; i++ {
		statedb, _ := ethservice.BlockChain().StateAt(parent.Root())
		nonce := statedb.GetNonce(testAddr)
		tx, _ := types.SignTx(types.NewContractCreation(nonce, new(big.Int), 1000000, big.NewInt(2*params.InitialBaseFee), logCode), types.LatestSigner(ethservice.BlockChain().Config()), testKey)
		ethservice.TxPool().AddLocal(tx)

		params := AssembleBlockParams{
			ParentHash:   parent.Hash(),
			Timestamp:    parent.Time() + 1,
			Random:       crypto.Keccak256Hash([]byte{byte(i)}),
			FeeRecipient: parent.Coinbase(),
		}
		resp, err := api.PreparePayload(params)
		if err != nil {
			t.Fatalf("can't prepare payload: %v", err)
		}
		payload, err := api.GetPayload(hexutil.Uint64(resp.PayloadID))
		if err != nil {
			t.Fatalf("can't get payload: %v", err)
		}
		execResp, err := api.ExecutePayload(*payload)
		if err != nil {
			t.Fatalf("can't execute payload: %v", err)
		}
		if execResp.Status != VALID.Status {
			t.Fatalf("invalid status: %v", execResp.Status)
		}

		if err := api.ConsensusValidated(ConsensusValidatedParams{BlockHash: payload.BlockHash, Status: VALID.Status}); err != nil {
			t.Fatalf("failed to validate consensus: %v", err)
		}

		if err := api.ForkchoiceUpdated(ForkChoiceParams{HeadBlockHash: payload.BlockHash, FinalizedBlockHash: payload.BlockHash}); err != nil {
			t.Fatalf("Failed to insert block: %v", err)
		}
		if ethservice.BlockChain().CurrentBlock().NumberU64() != payload.Number {
			t.Fatalf("Chain head should be updated")
		}
		parent = ethservice.BlockChain().CurrentBlock()

	}
}
