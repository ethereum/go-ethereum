//go:build integration

package bor

import (
	"crypto/ecdsa"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"math/big"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/accounts/keystore"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/fdlimit"
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
	"gotest.tools/assert"
)

var (
	// addr1 = 0x71562b71999873DB5b286dF957af199Ec94617F7
	pkey1, _ = crypto.HexToECDSA("b71c71a67e1177ad4e901695e1b4b9ee17ae16c6668d313eac2f96dbcda3f291")
	// addr2 = 0x9fB29AAc15b9A4B7F17c3385939b007540f4d791
	pkey2, _ = crypto.HexToECDSA("9b28f36fbd67381120752d6172ecdcf10e06ab2d9a1367aac00cdcd6ac7855d3")
	keys     = []*ecdsa.PrivateKey{pkey1, pkey2}
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
	genesis := InitGenesis1(t, faucets, "./testdata/genesis_sprint_length_change.json")

	var (
		nodes  []*eth.Ethereum
		enodes []*enode.Node
	)
	for i := 0; i < 2; i++ {
		// Start the node and wait until it's up
		stack, ethBackend, err := InitMiner1(genesis, keys[i], true)
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

var keys_21val = []map[string]string{
	{
		"address":  "0x387d24252f81Ef0d2F33c344986644a5acC794A2",
		"priv_key": "f8e385ea69ddaf460d062ec2748d04e6e126a0c873a5fdf6fbbda3e39dfc3e62",
		"pub_key":  "0x04b0c1c59b85bf89ee1f24a034feb7f25937996d0d2c36dfde188d643138d79a50a6c1f30dff9b5b74334afb387f287842dfb17f95263ffec4eac38ebba939d513",
	},
	{
		"address":  "0xA73335dA992875cF74359D966bBa2f4471CE1Cb7",
		"priv_key": "7362912aca5664bdbba8ba39ca98a91aee51c232c67f60be2d043d2d9c39fa32",
		"pub_key":  "0x044647e004cace245444a575065d56709f3bdaafba2aba0bbfb545fc8857f4259ec27f815a79b4edf1d421729a1a19d26d0b3efc2b6a32d1ef02f29bea0f55ca19",
	},
	{
		"address":  "0x54FA823e70Dcd10a735F4202602A895D5978c27C",
		"priv_key": "0fec87e66604e7224c49f3513d28db9606fa1d1f38d5321b0c929b5d42caca1e",
		"pub_key":  "0x04a86d9c85f0f42f5e612d0c261e5fcc3d6195a8e372eaa52368ad4ac55d30ac0ec6457c2fd27e41898bff823489c7c2951a0769c34d477813319f824c5fa4b4c4",
	},
	{
		"address":  "0xf630f2C51e17bECf5190Bc95B5211CdDB6848559",
		"priv_key": "06dd8c1acd2279b65fe209731866bcbd716b91de6df1a4237f9fa367d07432e5",
		"pub_key":  "0x0450ed3533599f0cc9f06843c6e512911e96c5238ef9b9cf7e1d1c8f00923774e8573f84c93068cc667f60d4c2c0f253b7ebd2ed6552abfe8dd05c792fe90c6c21",
	},
	{
		"address":  "0x0b02C2957AfA5Bc02CE7ADC2d973D4d0A5d67Aa8",
		"priv_key": "996a1f17a05c40fa78434f7c556c84db1f8498c8e229f70ef91612116c8be38e",
		"pub_key":  "0x0432dcee2dbea82250e850652986327d5d260c611de4897704384d13957232839f1fbbba4dc0533462e25db72e023382bfd947c0a09698fa9d3ceae51091e0a581",
	},
	{
		"address":  "0x1BE1047566F230C21Edae22446713e7087a9e81C",
		"priv_key": "e5a45caa09c247f4e8ccbca6d65fadf2ab30e089fb23675ec479a6500345b755",
		"pub_key":  "0x04585a0a04b62449351c4aef5c7e78c331d1123d5e4e2621346f1257c0c98347b30dd64d9973ae4a12fd006599da1400520324adeceafbe4c67c9a87d08b2d232b",
	},
	{
		"address":  "0x316679E22D8acf5955e2562a4ABf54feC109D1ed",
		"priv_key": "e91619a3a7e0d019655c15d6e4e50cf4cbdf3082422c180cddfecbe2e662c55d",
		"pub_key":  "0x04f69cf0b77fc139453060bc7a047697dd0efd3eeb1a408f529cd524d349fd0e9cfcc01b53a250477640de4f1287729ea947b3ad474cc44223669379a12e5caf66",
	},
	{
		"address":  "0x51ecFbE68aa337c720E6f17041CA6044d0958493",
		"priv_key": "afcd70258935c56b9f488ce8691e91467af30cbbf09b505d8b82fa0063eede50",
		"pub_key":  "0x04d2bc21f71f869dd7243fda7465a021918ede8fd6e25740eec4cacd703bfb480aad9a684bacf8079d9f4c11e79921312b2b11ee32c87c862a71fc9ac97be5be2f",
	},
	{
		"address":  "0x40E910D1bcBD0DACEF856594E02b176Be3EE736b",
		"priv_key": "fe8073957e8452e1e4d4d2493c54944dc738aee6800d69ed87d9df6e1eee5edc",
		"pub_key":  "0x0407d9666bdf36432e2edeba744e7115d3618a88d9c304d52b1121b9eceb9a3d13f653b97240ea688eb80d688c84c3995c831db27a0ffba9303216fc742190869f",
	},
	{
		"address":  "0x3594Cbcdd629A59ce16B581F7Bf6eA1E49F8F634",
		"priv_key": "3987d9f9183363debbc9556c01f00a3ee3cf57648495a242e57181a779179fdd",
		"pub_key":  "0x04de350843bd0e41f3671e99b16b05a867d598242cd92020d82eeb3bbc1143243e3ae81755d12fef0f5e5c24278e9ebd3c675d9c17dbff928dc916284de58bd44a",
	},
	{
		"address":  "0x5F2CbdEE214AFeF14608A724559C870Df5636463",
		"priv_key": "5a9ccbba1821843726a558ef10623b0871503852b7c1285f21f6842ad828f14f",
		"pub_key":  "0x045c6813c956abc2dd006cb5492693878380ca26f9af6eba415112a09f387cd1dcc383cab12dfc811d1f6b6be603e89b066d6d43c95622b4857ba7717a9c11eb6d",
	},
	{
		"address":  "0x39539B6E7dEfa23482bEB77f3559a9413d861676",
		"priv_key": "a5c3c7579ca6a3dc0bd1bd2947f471273c443d776e3446bfb34c494bcaac2611",
		"pub_key":  "0x0458b3d8fb3166c39b0f4d6c9239763b2ceaf1e2be63c23787978508e7747193adbfbe7333bd082b8f36260c47c49129f406bf35f31051a7906c95f7149780d23f",
	},
	{
		"address":  "0x433929e706F85F7a5b7e95c21043b221B18351b9",
		"priv_key": "bbab31856297e3bc4c583d850f8f5fddf77547ace374cdc50253e163da69a05d",
		"pub_key":  "0x04a45bfa6d2854a0276b62c7f4da4ae619110ce88b273ce7d365013df6a79c3441435a4eddd5eb07c7e2fb26779d46114fdabd5536e99aeadb39552c6750691273",
	},
	{
		"address":  "0xd0aeeB8CB9a457F6b6d2279eb48057291E65b442",
		"priv_key": "80aa9f29d1f7f99bf0b73a6efcae1b42f4b9064a21a9b6fa74efcfef7cee6ef8",
		"pub_key":  "0x0453f38c0d286abfc4f38bf44775be92c4adfd33397c5aa9790467584f5591d8eae9cc7569b8e40b0e15105b00ce28a680935c6e91f57a42924fe18dff0b785b8c",
	},
	{
		"address":  "0x909721f84066AB0C8FBf6E1309818A461D1C2881",
		"priv_key": "f433996a400752ee245599870fe23eb85231b0c4b6a2e33e4204b17dce0c481e",
		"pub_key":  "0x0410e5bfd3e42d228552c049c2b0af15bbfe73fbb5c7909280aa841d3eae24bfe5d17bc538def2c04c5e8c23c2ad9417578d96664b688a70de0e59b9aa9d3d9118",
	},
	{
		"address":  "0xf96A0EC13f0E3FEa25270925b0Ffb0896857F366",
		"priv_key": "625e57139db8afb3748f175c67747f2fc562244bf78ed63d0de449cff1deccad",
		"pub_key":  "0x046c6dbfb92754f2a1ad40ef3b9845d93376a96fcb0e94ffb86675faafbd175e6f159a3a0f940ea3ba7393cdc48e78642a7e2ed76a7bd6e33e106449d1f6f8da7f",
	},
	{
		"address":  "0xCd297859cAC13Fb4CCa55Fd2C591EBF035EBfbD9",
		"priv_key": "ce3ebb26a76728c8650bc0b0f611ede6bf8f6befeeb3757fa24bb69ff1bdaff8",
		"pub_key":  "0x0437147a7da1db2995b003da1cec8d894914b6df03cbc433492d1bf2b8065e5828807b50731d02c47d666f5225368bf58509dedae9ac472cf7e7baf111b1723568",
	},
	{
		"address":  "0xf6A1A2D64B3835AC821aCC07275cd5F4FfF00a1F",
		"priv_key": "7050d73354b4995410e88db7cf2f5fea5f5db2affcf0fe2b84ca16da36cf15fc",
		"pub_key":  "0x04d78f6716c606b384a951d56cbd90097e58170dc4e73d0cccb6b61ee60a9728128041c013acd10e41b2158a18462b1471bd5d1506e777969163e37c39a0e78d10",
	},
	{
		"address":  "0xc6F92b3C655DE61c9B0178600617e6F5E5a0fa7E",
		"priv_key": "620964f9a20384e9fb1108df8f4f13a14ff88b093b2354e92a57ea21b9b4e6e7",
		"pub_key":  "0x0438a76bd6ba6303768eac466b946807946a70afc29230181b7fada1629e149b4302d9208b08324cff6e232d8a92f6b74d3af941300c15dea3f30ee0d40f8d4fd8",
	},
	{
		"address":  "0x63ccD7e06399e4360993Fe5aD27dFd3DeE54b1cD",
		"priv_key": "3d557344e389159c233da56a390c55457aac2298ca9249f7cfe5ef5ed7c5aaf3",
		"pub_key":  "0x043bcf3f87fb96e8e3bc1174e9f79abd9f64fc7ddb413f031cd32abe75c6d79e0a4c099e2e8648a47a83cb1c43ca58a60538d7051fa972c2e2967541f7dbfed1fa",
	},
	{
		"address":  "0x09fa9bc378B41029C2858f9Cc2Ef8a1707DC38c9",
		"priv_key": "f643453ff10b2a547e906791a5a2962ff83998251ff65ce4771bcaea374e80b8",
		"pub_key":  "0x040bbd3813f194b106c385ee03b7d2b9512d582ec8d25613612976ed6b4e5bfa98d2b8b075f18d49deae4dcef55c5698e631c0ea390c6bdbe983d6510e3aef83c5",
	},
}

func TestSprintLengthReorg(t *testing.T) {
	reorgsLengthTests := []map[string]uint64{
		{
			"reorgLength": 10,
			"startBlock":  16,
		},
		// {
		// 	"reorgLength": 20,
		// 	"validator":   0,
		// 	"startBlock":  16,
		// },
		// {
		// 	"reorgLength": 30,
		// 	"validator":   0,
		// 	"startBlock":  16,
		// },
		// {
		// 	"reorgLength": 10,
		// 	"validator":   0,
		// 	"startBlock":  196,
		// },
	}
	for _, tt := range reorgsLengthTests {
		SetupValidatorsAndTest(t, tt)
	}
}

func SetupValidatorsAndTest(t *testing.T, tt map[string]uint64) {
	log.Root().SetHandler(log.LvlFilterHandler(log.LvlInfo, log.StreamHandler(os.Stderr, log.TerminalFormat(true))))
	fdlimit.Raise(2048)

	// Create an Ethash network based off of the Ropsten config
	genesis := InitGenesis1(t, nil, "./testdata/genesis_21val.json")

	var (
		nodes  []*eth.Ethereum
		enodes []*enode.Node
		stacks []*node.Node
	)

	pkeys_21val := make([]*ecdsa.PrivateKey, len(keys_21val))
	for index, signerdata := range keys_21val {
		pkeys_21val[index], _ = crypto.HexToECDSA(signerdata["priv_key"])
	}

	for i := 0; i < len(keys_21val); i++ {
		// Start the node and wait until it's up
		stack, ethBackend, err := InitMiner1(genesis, pkeys_21val[i], true)
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
	chain2HeadCh := make(chan core.Chain2HeadEvent, 64)
	primaryProducerIndex := 0
	subscribedNodeIndex := 0

	for {

		blockHeaderVal0 := nodes[0].BlockChain().CurrentHeader()
		author, _ := nodes[0].Engine().Author(blockHeaderVal0)

		if blockHeaderVal0.Number.Uint64() == tt["startBlock"] {
			for index, signerdata := range keys_21val {
				if strings.EqualFold(signerdata["address"], author.String()) {
					primaryProducerIndex = index
				}
			}
			for _, enode := range enodes {
				stacks[primaryProducerIndex].Server().RemovePeer(enode)
			}
			if primaryProducerIndex == 0 {
				subscribedNodeIndex = 1
			}
			nodes[subscribedNodeIndex].BlockChain().SubscribeChain2HeadEvent(chain2HeadCh)
			fmt.Println("----------------- startBlock", tt["startBlock"])
		}
		if blockHeaderVal0.Number.Uint64() == tt["startBlock"]+tt["reorgLength"]+1 {
			for _, enode := range enodes {
				stacks[primaryProducerIndex].Server().AddPeer(enode)
			}
			fmt.Println("----------------- endblock", tt["startBlock"]+tt["reorgLength"]+1)
		}
		if blockHeaderVal0.Number.Uint64() == tt["startBlock"]+tt["reorgLength"]+2 {
			fmt.Println("----------------- endblock", tt["startBlock"]+tt["reorgLength"]+2)
			break
		}

		select {
		case ev := <-chain2HeadCh:
			fmt.Printf("\n---------------\n%+v\n---------------\n", ev)

			// if len(ev.NewChain) != len(expect.Added) {
			// 	t.Fatal("Newchain and Added Array Size don't match")
			// }
			// if len(ev.OldChain) != len(expect.Removed) {
			// 	t.Fatal("Oldchain and Removed Array Size don't match")
			// }

			// for j := 0; j < len(ev.OldChain); j++ {
			// 	if ev.OldChain[j].Hash() != expect.Removed[j] {
			// 		t.Fatal("Oldchain hashes Do Not Match")
			// 	}
			// }
			// for j := 0; j < len(ev.NewChain); j++ {
			// 	if ev.NewChain[j].Hash() != expect.Added[j] {
			// 		t.Fatalf("Newchain hashes Do Not Match %s %s", ev.NewChain[j].Hash(), expect.Added[j])
			// 	}
			// }
		}
	}

}

func InitMiner1(genesis *core.Genesis, privKey *ecdsa.PrivateKey, withoutHeimdall bool) (*node.Node, *eth.Ethereum, error) {
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
		WithoutHeimdall: withoutHeimdall,
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

func InitGenesis1(t *testing.T, faucets []*ecdsa.PrivateKey, fileLocation string) *core.Genesis {

	// sprint size = 8 in genesis
	genesisData, err := ioutil.ReadFile(fileLocation)
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
