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
	assert.Equal(t, testBorConfig.CalculateSprint(0), uint64(16))
	assert.Equal(t, testBorConfig.CalculateSprint(8), uint64(4))
	assert.Equal(t, testBorConfig.CalculateSprint(9), uint64(4))

}

var keys_21val = []string{"f8e385ea69ddaf460d062ec2748d04e6e126a0c873a5fdf6fbbda3e39dfc3e62",
	"7362912aca5664bdbba8ba39ca98a91aee51c232c67f60be2d043d2d9c39fa32",
	"0fec87e66604e7224c49f3513d28db9606fa1d1f38d5321b0c929b5d42caca1e",
	"06dd8c1acd2279b65fe209731866bcbd716b91de6df1a4237f9fa367d07432e5",
	"996a1f17a05c40fa78434f7c556c84db1f8498c8e229f70ef91612116c8be38e",
	"e5a45caa09c247f4e8ccbca6d65fadf2ab30e089fb23675ec479a6500345b755",
	"e91619a3a7e0d019655c15d6e4e50cf4cbdf3082422c180cddfecbe2e662c55d",
	"afcd70258935c56b9f488ce8691e91467af30cbbf09b505d8b82fa0063eede50",
	"fe8073957e8452e1e4d4d2493c54944dc738aee6800d69ed87d9df6e1eee5edc",
	"3987d9f9183363debbc9556c01f00a3ee3cf57648495a242e57181a779179fdd",
	"5a9ccbba1821843726a558ef10623b0871503852b7c1285f21f6842ad828f14f",
	"a5c3c7579ca6a3dc0bd1bd2947f471273c443d776e3446bfb34c494bcaac2611",
	"bbab31856297e3bc4c583d850f8f5fddf77547ace374cdc50253e163da69a05d",
	"80aa9f29d1f7f99bf0b73a6efcae1b42f4b9064a21a9b6fa74efcfef7cee6ef8",
	"f433996a400752ee245599870fe23eb85231b0c4b6a2e33e4204b17dce0c481e",
	"625e57139db8afb3748f175c67747f2fc562244bf78ed63d0de449cff1deccad",
	"ce3ebb26a76728c8650bc0b0f611ede6bf8f6befeeb3757fa24bb69ff1bdaff8",
	"7050d73354b4995410e88db7cf2f5fea5f5db2affcf0fe2b84ca16da36cf15fc",
	"620964f9a20384e9fb1108df8f4f13a14ff88b093b2354e92a57ea21b9b4e6e7",
	"3d557344e389159c233da56a390c55457aac2298ca9249f7cfe5ef5ed7c5aaf3",
	"f643453ff10b2a547e906791a5a2962ff83998251ff65ce4771bcaea374e80b8"}

func TestSprintDependantReorgs(t *testing.T) {
	log.Root().SetHandler(log.LvlFilterHandler(log.LvlInfo, log.StreamHandler(os.Stderr, log.TerminalFormat(true))))
	fdlimit.Raise(2048)

	// Generate a batch of accounts to seal and fund with
	faucets := make([]*ecdsa.PrivateKey, 128)
	for i := 0; i < len(faucets); i++ {
		faucets[i], _ = crypto.GenerateKey()
	}

	// Create an Ethash network based off of the Ropsten config
	genesis := InitGenesis(t, faucets, "./testdata/genesis_21val.json")

	var (
		nodes  []*eth.Ethereum
		enodes []*enode.Node
	)

	pkeys_21val := make([]*ecdsa.PrivateKey, len(keys_21val))
	for i, key := range keys_21val {
		pkeys_21val[i], _ = crypto.HexToECDSA(key)
	}

	for i := 0; i < 2; i++ {
		// Start the node and wait until it's up
		stack, ethBackend, err := InitMiner(genesis, pkeys_21val[i], true)
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
}
