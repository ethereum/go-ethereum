//go:build integration

package bor

import (
	"crypto/ecdsa"
	"encoding/json"
	"io/ioutil"
	"math/big"
	"os"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/accounts/keystore"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/fdlimit"
	"github.com/ethereum/go-ethereum/consensus/ethash"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/eth"
	"github.com/ethereum/go-ethereum/eth/downloader"
	"github.com/ethereum/go-ethereum/eth/ethconfig"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/miner"
	"github.com/ethereum/go-ethereum/node"
	"github.com/ethereum/go-ethereum/p2p"
	"github.com/ethereum/go-ethereum/p2p/enode"
	"github.com/ethereum/go-ethereum/params"
	"github.com/stretchr/testify/assert"
)

var (
	// addr1 = 0x71562b71999873DB5b286dF957af199Ec94617F7
	pkey1, _ = crypto.HexToECDSA("b71c71a67e1177ad4e901695e1b4b9ee17ae16c6668d313eac2f96dbcda3f291")
	// addr2 = 0x9fB29AAc15b9A4B7F17c3385939b007540f4d791
	pkey2, _ = crypto.HexToECDSA("9b28f36fbd67381120752d6172ecdcf10e06ab2d9a1367aac00cdcd6ac7855d3")
	keys     = []*ecdsa.PrivateKey{pkey1, pkey2}
)

func initMiner(genesis *core.Genesis, privKey *ecdsa.PrivateKey) (*node.Node, *eth.Ethereum, error) {
	// Define the basic configurations for the Ethereum node
	datadir, _ := ioutil.TempDir("", "")

	config := &node.Config{
		Name:    "geth",
		Version: params.Version,
		DataDir: datadir,
		P2P: p2p.Config{
			ListenAddr:  "0.0.0.0:0",
			NoDiscovery: true,
			MaxPeers:    25,
		},
		UseLightweightKDF: true,
	}
	// Create the node and configure a full Ethereum node on it
	stack, err := node.New(config)
	if err != nil {
		return nil, nil, err
	}
	ethBackend, err := eth.New(stack, &ethconfig.Config{
		Genesis:         genesis,
		NetworkId:       genesis.Config.ChainID.Uint64(),
		SyncMode:        downloader.FullSync,
		DatabaseCache:   256,
		DatabaseHandles: 256,
		TxPool:          core.DefaultTxPoolConfig,
		GPO:             ethconfig.Defaults.GPO,
		Ethash:          ethconfig.Defaults.Ethash,
		Miner: miner.Config{
			Etherbase: crypto.PubkeyToAddress(privKey.PublicKey),
			GasCeil:   genesis.GasLimit * 11 / 10,
			GasPrice:  big.NewInt(1),
			Recommit:  time.Second,
		},
		WithoutHeimdall: true,
	})
	if err != nil {
		return nil, nil, err
	}

	// register backend to account manager with keystore for signing
	keydir := stack.KeyStoreDir()

	n, p := keystore.StandardScryptN, keystore.StandardScryptP
	kStore := keystore.NewKeyStore(keydir, n, p)

	kStore.ImportECDSA(privKey, "")
	acc := kStore.Accounts()[0]
	kStore.Unlock(acc, "")
	// proceed to authorize the local account manager in any case
	ethBackend.AccountManager().AddBackend(kStore)

	// ethBackend.AccountManager().AddBackend()
	err = stack.Start()
	return stack, ethBackend, err
}

func initGenesis(t *testing.T, faucets []*ecdsa.PrivateKey) *core.Genesis {

	// sprint size = 8 in genesis
	genesisData, err := ioutil.ReadFile("./testdata/genesis_2val.json")
	if err != nil {
		t.Fatalf("%s", err)
	}

	genesis := &core.Genesis{}

	if err := json.Unmarshal(genesisData, genesis); err != nil {
		t.Fatalf("%s", err)
	}

	genesis.Config.ChainID = big.NewInt(15001)
	genesis.Config.EIP150Hash = common.Hash{}

	return genesis
}

func TestValidatorWentOffline(t *testing.T) {

	log.Root().SetHandler(log.LvlFilterHandler(log.LvlInfo, log.StreamHandler(os.Stderr, log.TerminalFormat(true))))
	fdlimit.Raise(2048)

	// Generate a batch of accounts to seal and fund with
	faucets := make([]*ecdsa.PrivateKey, 128)
	for i := 0; i < len(faucets); i++ {
		faucets[i], _ = crypto.GenerateKey()
	}

	// Pre-generate the ethash mining DAG so we don't race
	ethash.MakeDataset(1, ethconfig.Defaults.Ethash.DatasetDir)

	// Create an Ethash network based off of the Ropsten config
	genesis := initGenesis(t, faucets)

	var (
		stacks []*node.Node
		nodes  []*eth.Ethereum
		enodes []*enode.Node
	)
	for i := 0; i < 2; i++ {
		// Start the node and wait until it's up
		stack, ethBackend, err := initMiner(genesis, keys[i])
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
		if err := node.StartMining(1); err != nil {
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
