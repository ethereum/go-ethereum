// Copyright 2022 The go-ethereum Authors
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

	"github.com/ethereum/go-ethereum/beacon/engine"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/consensus/ethash"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/eth/downloader"
	"github.com/ethereum/go-ethereum/eth/ethconfig"
	"github.com/ethereum/go-ethereum/les"
	"github.com/ethereum/go-ethereum/node"
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

func generatePreMergeChain(pre, post int) (*core.Genesis, []*types.Header, []*types.Block, []*types.Header, []*types.Block) {
	config := *params.AllEthashProtocolChanges
	genesis := &core.Genesis{
		Config:    &config,
		Alloc:     core.GenesisAlloc{testAddr: {Balance: testBalance}},
		ExtraData: []byte("test genesis"),
		Timestamp: 9000,
		BaseFee:   big.NewInt(params.InitialBaseFee),
	}
	// Pre-merge blocks
	db, preBLocks, _ := core.GenerateChainWithGenesis(genesis, ethash.NewFaker(), pre, nil)
	totalDifficulty := new(big.Int).Set(params.GenesisDifficulty)

	var preHeaders []*types.Header
	for _, b := range preBLocks {
		totalDifficulty.Add(totalDifficulty, b.Difficulty())
		preHeaders = append(preHeaders, b.Header())
	}
	config.TerminalTotalDifficulty = totalDifficulty
	// Post-merge blocks
	postBlocks, _ := core.GenerateChain(genesis.Config,
		preBLocks[len(preBLocks)-1], ethash.NewFaker(), db, post,
		func(i int, b *core.BlockGen) {
			b.SetPoS()
		})

	var postHeaders []*types.Header
	for _, b := range postBlocks {
		postHeaders = append(postHeaders, b.Header())
	}

	return genesis, preHeaders, preBLocks, postHeaders, postBlocks
}

func TestSetHeadBeforeTotalDifficulty(t *testing.T) {
	genesis, headers, blocks, _, _ := generatePreMergeChain(10, 0)
	n, lesService := startLesService(t, genesis, headers)
	defer n.Close()

	api := NewConsensusAPI(lesService)
	fcState := engine.ForkchoiceStateV1{
		HeadBlockHash:      blocks[5].Hash(),
		SafeBlockHash:      common.Hash{},
		FinalizedBlockHash: common.Hash{},
	}
	if _, err := api.ForkchoiceUpdatedV1(fcState, nil); err == nil {
		t.Errorf("fork choice updated before total terminal difficulty should fail")
	}
}

func TestExecutePayloadV1(t *testing.T) {
	genesis, headers, _, _, postBlocks := generatePreMergeChain(10, 2)
	n, lesService := startLesService(t, genesis, headers)
	lesService.Merger().ReachTTD()
	defer n.Close()

	api := NewConsensusAPI(lesService)
	fcState := engine.ForkchoiceStateV1{
		HeadBlockHash:      postBlocks[0].Hash(),
		SafeBlockHash:      common.Hash{},
		FinalizedBlockHash: common.Hash{},
	}
	if _, err := api.ForkchoiceUpdatedV1(fcState, nil); err != nil {
		t.Errorf("Failed to update head %v", err)
	}
	block := postBlocks[0]

	fakeBlock := types.NewBlock(&types.Header{
		ParentHash:  block.ParentHash(),
		UncleHash:   crypto.Keccak256Hash(nil),
		Coinbase:    block.Coinbase(),
		Root:        block.Root(),
		TxHash:      crypto.Keccak256Hash(nil),
		ReceiptHash: crypto.Keccak256Hash(nil),
		Bloom:       block.Bloom(),
		Difficulty:  big.NewInt(0),
		Number:      block.Number(),
		GasLimit:    block.GasLimit(),
		GasUsed:     block.GasUsed(),
		Time:        block.Time(),
		Extra:       block.Extra(),
		MixDigest:   block.MixDigest(),
		Nonce:       types.BlockNonce{},
		BaseFee:     block.BaseFee(),
	}, nil, nil, nil, trie.NewStackTrie(nil))

	_, err := api.ExecutePayloadV1(engine.ExecutableData{
		ParentHash:    fakeBlock.ParentHash(),
		FeeRecipient:  fakeBlock.Coinbase(),
		StateRoot:     fakeBlock.Root(),
		ReceiptsRoot:  fakeBlock.ReceiptHash(),
		LogsBloom:     fakeBlock.Bloom().Bytes(),
		Random:        fakeBlock.MixDigest(),
		Number:        fakeBlock.NumberU64(),
		GasLimit:      fakeBlock.GasLimit(),
		GasUsed:       fakeBlock.GasUsed(),
		Timestamp:     fakeBlock.Time(),
		ExtraData:     fakeBlock.Extra(),
		BaseFeePerGas: fakeBlock.BaseFee(),
		BlockHash:     fakeBlock.Hash(),
		Transactions:  encodeTransactions(fakeBlock.Transactions()),
	})
	if err != nil {
		t.Errorf("Failed to execute payload %v", err)
	}
	headHeader := api.les.BlockChain().CurrentHeader()
	if headHeader.Number.Uint64() != fakeBlock.NumberU64()-1 {
		t.Fatal("Unexpected chain head update")
	}
	fcState = engine.ForkchoiceStateV1{
		HeadBlockHash:      fakeBlock.Hash(),
		SafeBlockHash:      common.Hash{},
		FinalizedBlockHash: common.Hash{},
	}
	if _, err := api.ForkchoiceUpdatedV1(fcState, nil); err != nil {
		t.Fatal("Failed to update head")
	}
	headHeader = api.les.BlockChain().CurrentHeader()
	if headHeader.Number.Uint64() != fakeBlock.NumberU64() {
		t.Fatal("Failed to update chain head")
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
			if err := api.setCanonical(block.Hash()); err != nil {
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
func startLesService(t *testing.T, genesis *core.Genesis, headers []*types.Header) (*node.Node, *les.LightEthereum) {
	t.Helper()

	n, err := node.New(&node.Config{})
	if err != nil {
		t.Fatal("can't create node:", err)
	}
	ethcfg := &ethconfig.Config{
		Genesis:        genesis,
		Ethash:         ethash.Config{PowMode: ethash.ModeFake},
		SyncMode:       downloader.LightSync,
		TrieDirtyCache: 256,
		TrieCleanCache: 256,
		LightPeers:     10,
	}
	lesService, err := les.New(n, ethcfg)
	if err != nil {
		t.Fatal("can't create eth service:", err)
	}
	if err := n.Start(); err != nil {
		t.Fatal("can't start node:", err)
	}
	if _, err := lesService.BlockChain().InsertHeaderChain(headers, 0); err != nil {
		n.Close()
		t.Fatal("can't import test headers:", err)
	}
	return n, lesService
}

func encodeTransactions(txs []*types.Transaction) [][]byte {
	var enc = make([][]byte, len(txs))
	for i, tx := range txs {
		enc[i], _ = tx.MarshalBinary()
	}
	return enc
}
