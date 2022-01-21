package core

import (
	"math/big"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/consensus/ethash"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/params"
)

func TestChain2HeadEvent(t *testing.T) {
	var (
		db      = rawdb.NewMemoryDatabase()
		key1, _ = crypto.HexToECDSA("b71c71a67e1177ad4e901695e1b4b9ee17ae16c6668d313eac2f96dbcda3f291")
		addr1   = crypto.PubkeyToAddress(key1.PublicKey)
		gspec   = &Genesis{
			Config: params.TestChainConfig,
			Alloc:  GenesisAlloc{addr1: {Balance: big.NewInt(10000000000000000)}},
		}
		genesis = gspec.MustCommit(db)
		signer  = types.LatestSigner(gspec.Config)
	)

	blockchain, _ := NewBlockChain(db, nil, gspec.Config, ethash.NewFaker(), vm.Config{}, nil, nil)
	defer blockchain.Stop()

	chain2HeadCh := make(chan Chain2HeadEvent, 64)
	blockchain.SubscribeChain2HeadEvent(chain2HeadCh)

	chain, _ := GenerateChain(gspec.Config, genesis, ethash.NewFaker(), db, 3, func(i int, gen *BlockGen) {})
	if _, err := blockchain.InsertChain(chain); err != nil {
		t.Fatalf("failed to insert chain: %v", err)
	}

	replacementBlocks, _ := GenerateChain(gspec.Config, genesis, ethash.NewFaker(), db, 4, func(i int, gen *BlockGen) {
		tx, err := types.SignTx(types.NewContractCreation(gen.TxNonce(addr1), new(big.Int), 1000000, gen.header.BaseFee, nil), signer, key1)
		if i == 2 {
			gen.OffsetTime(-9)
		}
		if err != nil {
			t.Fatalf("failed to create tx: %v", err)
		}
		gen.AddTx(tx)
	})

	if _, err := blockchain.InsertChain(replacementBlocks); err != nil {
		t.Fatalf("failed to insert chain: %v", err)
	}

	i := 0
	readEvent := func() *Chain2HeadEvent {
		select {
		case ev := <-chain2HeadCh:
			i++
			return &ev
		case <-time.After(2 * time.Second):
			t.Fatal("timeout")
		}
		return nil
	}

	// head event
	event1 := readEvent()
	if event1.Type != Chain2HeadCanonicalEvent {
		t.Fatal("it should be type head")
	}
	if event1.NewChain[0].Hash() != chain[2].Hash() {
		t.Fatalf("%d : Hash Does Not Match", i)
	}

	// fork event
	event2 := readEvent()
	if event2.Type != Chain2HeadForkEvent {
		t.Fatal("it should be type fork")
	}
	if event2.NewChain[0].Hash() != replacementBlocks[0].Hash() {
		t.Fatalf("%d : Hash Does Not Match", i)
	}

	// fork event
	event3 := readEvent()
	if event3.Type != Chain2HeadForkEvent {
		t.Fatal("it should be type fork")
	}
	if event3.NewChain[0].Hash() != replacementBlocks[1].Hash() {
		t.Fatalf("%d : Hash Does Not Match", i)
	}

	// reorg event
	//In this event the channel recieves an array of Blocks in NewChain and OldChain
	expectedOldChainHashes := [3]common.Hash{0: chain[2].Hash(), 1: chain[1].Hash(), 2: chain[0].Hash()}
	expectedNewChainHashes := [3]common.Hash{0: replacementBlocks[2].Hash(), 1: replacementBlocks[1].Hash(), 2: replacementBlocks[0].Hash()}

	event4 := readEvent()
	if event4.Type != Chain2HeadReorgEvent {
		t.Fatal("it should be type reorg")
	}
	for j := 0; j < len(event4.OldChain); j++ {
		if event4.OldChain[j].Hash() != expectedOldChainHashes[j] {
			t.Fatalf("%d : Oldchain hashes Does Not Match", i)
		}
	}
	for j := 0; j < len(event4.NewChain); j++ {
		if event4.NewChain[j].Hash() != expectedNewChainHashes[j] {
			t.Fatalf("%d : Newchain hashes Does Not Match", i)
		}
	}

	// head event
	event5 := readEvent()
	if event5.Type != Chain2HeadCanonicalEvent {
		t.Fatal("it should be type head")
	}
	if event5.NewChain[0].Hash() != replacementBlocks[3].Hash() {
		t.Fatalf("%d : Hash Does Not Match", i)
	}
}
