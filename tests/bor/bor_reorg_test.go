//go:build integration

package bor

import (
	"sync"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/core/types"
	"github.com/stretchr/testify/assert"
)

func TestValidatorWentOffline(t *testing.T) {

	// Create an Ethash network based off of the Ropsten config
	genesis := initGenesis(t)
	stacks, nodes, enodes := setupMiner(t, 2, genesis)

	defer func() {
		for _, stack := range stacks {
			stack.Close()
		}
	}()

	// Iterate over all the nodes and start mining
	for _, node := range nodes {
		if err := node.StartMining(1); err != nil {
			t.Fatal("Error occured while starting miner", "node", node, "error", err)
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
		t.Error("Error in getting author", "err", err)
	}
	authorVal1, err := nodes[1].Engine().Author(blockHeaderVal1)
	if err != nil {
		t.Error("Error in getting author", "err", err)
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
		t.Error("Error in getting author", "err", err)
	}
	authorVal1, err = nodes[1].Engine().Author(blockHeaderVal1)
	if err != nil {
		t.Error("Error in getting author", "err", err)
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
		t.Error("Error in getting author", "err", err)
	}
	authorVal1, err = nodes[1].Engine().Author(blockHeaderVal1)
	if err != nil {
		t.Error("Error in getting author", "err", err)
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
		t.Error("Error in getting author", "err", err)
	}
	authorVal1, err = nodes[1].Engine().Author(blockHeaderVal1)
	if err != nil {
		t.Error("Error in getting author", "err", err)
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
		sprint        uint64
		blockTime     map[string]uint64
		change        uint64
		producerDelay uint64
		forkExpected  bool
	}{
		{
			name:   "No fork after 2 sprints with producer delay = max block time",
			sprint: 128,
			blockTime: map[string]uint64{
				"0":   5,
				"128": 2,
				"256": 8,
			},
			change:        2,
			producerDelay: 8,
			forkExpected:  false,
		},
		{
			name:   "No Fork after 1 sprint producer delay = max block time",
			sprint: 64,
			blockTime: map[string]uint64{
				"0":  5,
				"64": 2,
			},
			change:        1,
			producerDelay: 5,
			forkExpected:  false,
		},
		{
			name:   "Fork after 4 sprints with producer delay < max block time",
			sprint: 16,
			blockTime: map[string]uint64{
				"0":  2,
				"64": 5,
			},
			change:        4,
			producerDelay: 4,
			forkExpected:  true,
		},
	}

	// Create an Ethash network based off of the Ropsten config
	genesis := initGenesis(t)

	for _, test := range cases {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

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
				if err := node.StartMining(1); err != nil {
					t.Fatal("Error occured while starting miner", "node", node, "error", err)
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

					for {
						select {
						case <-ticker.C:
							blockHeaders[i] = nodes[i].BlockChain().GetHeaderByNumber(test.sprint*test.change + 10)
							if blockHeaders[i] != nil {
								return
							}
						default:

						}
					}

				}(i)
			}

			wg.Wait()
			ticker.Stop()

			// Before the end of sprint
			blockHeaderVal0 := nodes[0].BlockChain().GetHeaderByNumber(test.sprint - 1)
			blockHeaderVal1 := nodes[1].BlockChain().GetHeaderByNumber(test.sprint - 1)
			assert.Equal(t, blockHeaderVal0.Hash(), blockHeaderVal1.Hash())
			assert.Equal(t, blockHeaderVal0.Time, blockHeaderVal1.Time)

			author0, err := nodes[0].Engine().Author(blockHeaderVal0)
			if err != nil {
				t.Error("Error occured while fetching author", "err", err)
			}
			author1, err := nodes[1].Engine().Author(blockHeaderVal1)
			if err != nil {
				t.Error("Error occured while fetching author", "err", err)
			}
			assert.Equal(t, author0, author1)

			// After the end of sprint
			author2, err := nodes[0].Engine().Author(blockHeaders[0])
			if err != nil {
				t.Error("Error occured while fetching author", "err", err)
			}
			author3, err := nodes[1].Engine().Author(blockHeaders[1])
			if err != nil {
				t.Error("Error occured while fetching author", "err", err)
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
