package core

import (
	"fmt"
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

	readEvent := func() *Chain2HeadEvent {
		select {
		case evnt := <-chain2HeadCh:
			return &evnt
		case <-time.After(2 * time.Second):
			t.Fatal("timeout")
		}
		return nil
	}

	// head event
	evnt := readEvent()
	fmt.Println(evnt.Type)

	// fork event
	evnt = readEvent()
	fmt.Println(evnt.Type)

	// fork event
	evnt = readEvent()
	fmt.Println(evnt.Type)

	// reorg event
	evnt = readEvent()
	fmt.Println(evnt.Type)

	// head event
	evnt = readEvent()
	fmt.Println(evnt.Type)

	return

	// first two block of the secondary chain are for a brief moment considered
	// side chains because up to that point the first one is considered the
	// heavier chain.
	expectedReorgHashes := map[common.Hash]bool{
		replacementBlocks[0].Hash(): true,
		replacementBlocks[1].Hash(): true,
		replacementBlocks[2].Hash(): true,
	}

	expectedReplacedHashes := map[common.Hash]bool{
		chain[0].Hash(): true,
		chain[1].Hash(): true,
		chain[2].Hash(): true,
	}

	expectedForkHashes := map[common.Hash]bool{
		replacementBlocks[0].Hash(): true,
		replacementBlocks[1].Hash(): true,
	}

	expectedHeadHashes := map[common.Hash]bool{
		replacementBlocks[3].Hash(): true,
	}
	i := 0

	//number of totalEvents are 4 : when the second chain is generated, there are 2 fork events,
	//then reorg happens and triggers 1 event, then last head block of Replacement chain triggers 1 event
	totalEvents := 4

	const timeoutDura = 10 * time.Second
	timeout := time.NewTimer(timeoutDura)
done:
	for {
		select {
		case ev := <-chain2HeadCh:
			if ev.Type == Chain2HeadReorgEvent {
				//Reorg Event Sends Chain of Added Blocks in NewChain. So need to check all of them in reorgHashes
				for j := 0; j < len(ev.OldChain); j++ {
					block := ev.OldChain[j]
					if _, ok := expectedReplacedHashes[block.Hash()]; !ok {
						t.Errorf("%d: didn't expect %x to be in side chain", i, block.Hash())
					}

				}
				//Reorg Event also Sends Chain of Removed Blocks in NewChain. So need to check all of them in replacedHashes
				for j := 0; j < len(ev.OldChain); j++ {
					block := ev.NewChain[j]
					if _, ok := expectedReorgHashes[block.Hash()]; !ok {
						t.Errorf("%d: didn't expect %x to be in side chain", i, block.Hash())
					}

				}

			}

			if ev.Type == Chain2HeadForkEvent {
				block := ev.NewChain[0]
				if _, ok := expectedForkHashes[block.Hash()]; !ok {
					t.Errorf("%d: didn't expect %x to be in fork chain", i, block.Hash())
				}
			}

			if ev.Type == Chain2HeadCanonicalEvent {
				block := ev.NewChain[0]
				if _, ok := expectedHeadHashes[block.Hash()]; !ok {
					t.Errorf("%d: didn't expect %x to be in head chain", i, block.Hash())
				}
			}

			i++

			if i == (totalEvents) {
				timeout.Stop()

				break done
			}
			timeout.Reset(timeoutDura)

		case <-timeout.C:
			t.Fatal("Timeout. Possibly not all blocks were triggered for sideevent")
		}
	}

	// make sure no more events are fired
	select {
	case e := <-chain2HeadCh:
		t.Errorf("unexpected event fired: %v", e)
	case <-time.After(250 * time.Millisecond):
	}

}
