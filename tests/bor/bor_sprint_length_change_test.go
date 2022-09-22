//go:build integration

package bor

import (
	"crypto/ecdsa"
	"os"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common/fdlimit"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/eth"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/p2p/enode"
	"github.com/ethereum/go-ethereum/params"
	"gotest.tools/assert"
)

// Sprint length change tests
func TestValidatorsBlockProduction(t *testing.T) {

	log.Root().SetHandler(log.LvlFilterHandler(log.LvlInfo, log.StreamHandler(os.Stderr, log.TerminalFormat(true))))
	fdlimit.Raise(2048)

	// Generate a batch of accounts to seal and fund with
	faucets := make([]*ecdsa.PrivateKey, 128)
	for i := 0; i < len(faucets); i++ {
		faucets[i], _ = crypto.GenerateKey()
	}

	// Create an Ethash network based off of the Ropsten config
	genesis := InitGenesis(t, faucets, "./testdata/genesis_sprint_length_change.json")

	var (
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
		nodes = append(nodes, ethBackend)
		enodes = append(enodes, stack.Server().Self())
	}

	// Iterate over all the nodes and start mining
	time.Sleep(3 * time.Second)
	for _, node := range nodes {
		if err := node.StartMining(1); err != nil {
			panic(err)
		}
	}

	for {

		// for block 0 to 7, the primary validator is node0
		// for block 8 to 15, the primary validator is node1
		// for block 16 to 19, the primary validator is node0
		// for block 20 to 23, the primary validator is node1
		blockHeaderVal0 := nodes[0].BlockChain().CurrentHeader()

		if blockHeaderVal0.Number.Uint64() == 24 {
			break
		}

	}

	// check block 7 miner ; expected author is node0 signer
	blockHeaderVal0 := nodes[0].BlockChain().GetHeaderByNumber(7)
	blockHeaderVal1 := nodes[1].BlockChain().GetHeaderByNumber(7)
	authorVal0, err := nodes[0].Engine().Author(blockHeaderVal0)
	if err != nil {
		log.Error("Error in getting author", "err", err)
	}
	authorVal1, err := nodes[1].Engine().Author(blockHeaderVal1)
	if err != nil {
		log.Error("Error in getting author", "err", err)
	}

	// check both nodes have the same block 7
	assert.Equal(t, authorVal0, authorVal1)

	// check block mined by node0
	assert.Equal(t, authorVal0, nodes[0].AccountManager().Accounts()[0])

	// check block 15 miner ; expected author is node1 signer
	blockHeaderVal0 = nodes[0].BlockChain().GetHeaderByNumber(15)
	blockHeaderVal1 = nodes[1].BlockChain().GetHeaderByNumber(15)
	authorVal0, err = nodes[0].Engine().Author(blockHeaderVal0)
	if err != nil {
		log.Error("Error in getting author", "err", err)
	}
	authorVal1, err = nodes[1].Engine().Author(blockHeaderVal1)
	if err != nil {
		log.Error("Error in getting author", "err", err)
	}

	// check both nodes have the same block 15
	assert.Equal(t, authorVal0, authorVal1)

	// check block mined by node1
	assert.Equal(t, authorVal0, nodes[1].AccountManager().Accounts()[0])

	// check block 19 miner ; expected author is node0 signer
	blockHeaderVal0 = nodes[0].BlockChain().GetHeaderByNumber(19)
	blockHeaderVal1 = nodes[1].BlockChain().GetHeaderByNumber(19)
	authorVal0, err = nodes[0].Engine().Author(blockHeaderVal0)
	if err != nil {
		log.Error("Error in getting author", "err", err)
	}
	authorVal1, err = nodes[1].Engine().Author(blockHeaderVal1)
	if err != nil {
		log.Error("Error in getting author", "err", err)
	}

	// check both nodes have the same block 19
	assert.Equal(t, authorVal0, authorVal1)

	// check block mined by node0
	assert.Equal(t, authorVal0, nodes[0].AccountManager().Accounts()[0])

	// check block 23 miner ; expected author is node1 signer
	blockHeaderVal0 = nodes[0].BlockChain().GetHeaderByNumber(23)
	blockHeaderVal1 = nodes[1].BlockChain().GetHeaderByNumber(23)
	authorVal0, err = nodes[0].Engine().Author(blockHeaderVal0)
	if err != nil {
		log.Error("Error in getting author", "err", err)
	}
	authorVal1, err = nodes[1].Engine().Author(blockHeaderVal1)
	if err != nil {
		log.Error("Error in getting author", "err", err)
	}

	// check both nodes have the same block 23
	assert.Equal(t, authorVal0, authorVal1)

	// check block mined by node1
	assert.Equal(t, authorVal0, nodes[1].AccountManager().Accounts()[0])

}

func TestSprintLengths(t *testing.T) {
	testBorConfig := params.TestChainConfig.Bor
	testBorConfig.Sprint = map[string]uint64{
		"0": 16,
		"8": 4,
	}
	assert.Equal(t, testBorConfig.CalculateSprint(0), 16)
	assert.Equal(t, testBorConfig.CalculateSprint(9), 4)

}
