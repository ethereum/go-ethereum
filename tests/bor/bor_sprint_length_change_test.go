package bor

import (
	"crypto/ecdsa"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io/ioutil" // nolint: staticcheck
	_log "log"
	"math/big"
	"os"
	"sync"
	"testing"
	"time"

	"gotest.tools/assert"

	"github.com/ethereum/go-ethereum/accounts/keystore"
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
)

var (

	// Only this account is a validator for the 0th span
	keySprintLength, _ = crypto.HexToECDSA(privKeySprintLength)

	// This account is one of the validators for 1st span (0-indexed)
	keySprintLength2, _ = crypto.HexToECDSA(privKeySprintLength2)

	keysSprintLength = []*ecdsa.PrivateKey{keySprintLength, keySprintLength2}
)

const (
	privKeySprintLength  = "b71c71a67e1177ad4e901695e1b4b9ee17ae16c6668d313eac2f96dbcda3f291"
	privKeySprintLength2 = "9b28f36fbd67381120752d6172ecdcf10e06ab2d9a1367aac00cdcd6ac7855d3"
)

// Sprint length change tests
func TestValidatorsBlockProduction(t *testing.T) {
	// t.Parallel()

	log.Root().SetHandler(log.LvlFilterHandler(3, log.StreamHandler(os.Stderr, log.TerminalFormat(true))))

	_, err := fdlimit.Raise(2048)

	if err != nil {
		panic(err)
	}

	// Generate a batch of accounts to seal and fund with
	faucets := make([]*ecdsa.PrivateKey, 128)
	for i := 0; i < len(faucets); i++ {
		faucets[i], _ = crypto.GenerateKey()
	}

	// Create an Ethash network based off of the Ropsten config
	// Generate a batch of accounts to seal and fund with
	genesis := InitGenesisSprintLength(t, faucets, "./testdata/genesis_sprint_length_change.json", 8)

	nodes := make([]*eth.Ethereum, 2)
	enodes := make([]*enode.Node, 2)

	for i := 0; i < 2; i++ {
		// Start the node and wait until it's up
		stack, ethBackend, err := InitMinerSprintLength(genesis, keysSprintLength[i], true)
		if err != nil {
			panic(err)
		}
		defer stack.Close()

		for stack.Server().NodeInfo().Ports.Listener == 0 {
			time.Sleep(250 * time.Millisecond)
		}
		// Connect the node to all the previous ones
		for j, n := range enodes {
			if j < i {
				stack.Server().AddPeer(n)
			}
		}
		// Start tracking the node and its enode
		nodes[i] = ethBackend
		enodes[i] = stack.Server().Self()
	}

	// Iterate over all the nodes and start mining
	time.Sleep(3 * time.Second)

	for _, node := range nodes {
		if err := node.StartMining(); err != nil {
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
	// t.Parallel()

	testBorConfig := params.TestChainConfig.Bor
	testBorConfig.Sprint = map[string]uint64{
		"0": 16,
		"8": 4,
	}
	assert.Equal(t, testBorConfig.CalculateSprint(0), uint64(16))
	assert.Equal(t, testBorConfig.CalculateSprint(8), uint64(4))
	assert.Equal(t, testBorConfig.CalculateSprint(9), uint64(4))
}

func TestProducerDelay(t *testing.T) {
	// t.Parallel()

	testBorConfig := params.TestChainConfig.Bor
	testBorConfig.ProducerDelay = map[string]uint64{
		"0": 16,
		"8": 4,
	}
	assert.Equal(t, testBorConfig.CalculateProducerDelay(0), uint64(16))
	assert.Equal(t, testBorConfig.CalculateProducerDelay(8), uint64(4))
	assert.Equal(t, testBorConfig.CalculateProducerDelay(9), uint64(4))
}

var keys_21val = []map[string]string{
	{
		"address":  "0x5C3E1B893B9315a968fcC6bce9EB9F7d8E22edB3",
		"priv_key": "c19fac8e538447124ad2408d9fbaeda2bb686fee763dca7a6bab58ea12442413",
		"pub_key":  "0x0495421933eda03dcc37f9186c24e255b569513aefae71e96d55d0db3df17502e24e86297b01a167fab9ce1174f06ee3110510ac242e39218bd964de5b345edbd6",
	},
	{
		"address":  "0x73E033779C9030D4528d86FbceF5B02e97488921",
		"priv_key": "61eb51cf8936309151ab7b931841ea033b6a09931f6a100b464fbbd74f3e0bd7",
		"pub_key":  "0x04f9a5e9bf76b45ac58f1b018ccba4b83b3531010cdadf42174c18a9db9879ef1dcb5d1254ce834bc108b110cd8d0186ed69a0387528a142bdb5936faf58bf98c9",
	},
	{
		"address":  "0x751eC4877450B8a4D652d0D70197337FC38a42e6",
		"priv_key": "6e7f48d012c9c0baadbdc88af32521e2e477fd6898a9b65e6abe19fd6652cb2e",
		"pub_key":  "0x0479db4c0b757bf0e5d9b8954b078ab7c0e91d6c19697904d23d07ea4853c8584382de91174929ba5c598214b8a991471ae051458ea787cdc15a4e435a55ef8059",
	},
	{
		"address":  "0xA464DC4810Bc79B956810759e314d85BcE35cD1c",
		"priv_key": "3efcf3f7014a6257f4a443119851414111820c681b27525dab3f35e72e28e51e",
		"pub_key":  "0x040180920306bf598ea050e258f2c7e50804a77a64f5a11705e08d18ee71eb0a80fafc95d0a42b92371ded042edda16c1f0b5f2fef7c4113ad66c59a71c29d977e",
	},
	{
		"address":  "0xb005bc07015170266Bd430f3EC1322938603be20",
		"priv_key": "17cd9b38c2b3a639c7d97ccbf2bb6c7140ab8f625aec4c249bc8e4cfd3bf9a96",
		"pub_key":  "0x04435a70d343aa569e6f3386c73e39a1aa6f88c30e5943baedda9618b55cc944a2de1d114aff6d0e9fa002bebc780b04ef6c1b8a06bbf0d41c10d1efa55390f198",
	},
	{
		"address":  "0xE8d02Da3dFeeB3e755472D95D666BD6821D92129",
		"priv_key": "45c9ef66361a2283cef14184f128c41949103b791aa622ead3c0bc844648b835",
		"pub_key":  "0x04a14651ddc80467eb589d72d95153fa695e4cb2e4bb99edeb912e840d309d61313b6f4676081b099f29e6598ecf98cb7b44bb862d019920718b558f27ba94ca51",
	},
	{
		"address":  "0xF93B54Cf36E917f625B48e1e3C9F93BC2344Fb06",
		"priv_key": "93788a1305605808df1f9a96b5e1157da191680cf08bc15e077138f517563cd5",
		"pub_key":  "0x045eee11dceccd9cccc371ca3d96d74c848e785223f1e5df4d1a7f08efdfeb90bd8f0035342a9c26068cf6c7ab395ca3ceea555541325067fc187c375390efa57d",
	},
}

func getTestSprintLengthReorgCases2Nodes() []map[string]interface{} {
	sprintSizes := []uint64{64}
	faultyNodes := [][]uint64{{0, 1}, {1, 2}, {0, 2}}
	reorgsLengthTests := make([]map[string]interface{}, 0)

	for i := uint64(0); i < uint64(len(sprintSizes)); i++ {
		maxReorgLength := sprintSizes[i] * 4
		for j := uint64(20); j <= maxReorgLength; j = j + 8 {
			maxStartBlock := sprintSizes[i] - 1
			for k := sprintSizes[i] / 2; k <= maxStartBlock; k = k + 4 {
				for l := uint64(0); l < uint64(len(faultyNodes)); l++ {
					if j+k < sprintSizes[i] {
						continue
					}

					reorgsLengthTest := map[string]interface{}{
						"reorgLength": j,
						"startBlock":  k,
						"sprintSize":  sprintSizes[i],
						"faultyNodes": faultyNodes[l], // node 1(index) is primary validator of the first sprint
					}
					reorgsLengthTests = append(reorgsLengthTests, reorgsLengthTest)
				}
			}
		}
	}
	// reorgsLengthTests := []map[string]uint64{
	// 	{
	// 		"reorgLength": 3,
	// 		"startBlock":  7,
	// 		"sprintSize":  8,
	// 		"faultyNode":  1,
	// 	},
	// }
	return reorgsLengthTests
}

func getTestSprintLengthReorgCases() []map[string]uint64 {
	sprintSizes := []uint64{64, 32, 16, 8}
	faultyNodes := []uint64{0, 1}
	reorgsLengthTests := make([]map[string]uint64, 0)

	for i := uint64(0); i < uint64(len(sprintSizes)); i++ {
		maxReorgLength := sprintSizes[i] * 4
		for j := uint64(3); j <= maxReorgLength; j = j + 4 {
			maxStartBlock := sprintSizes[i] - 1
			for k := sprintSizes[i] / 2; k <= maxStartBlock; k = k + 4 {
				for l := uint64(0); l < uint64(len(faultyNodes)); l++ {
					if j+k < sprintSizes[i] {
						continue
					}

					reorgsLengthTest := map[string]uint64{
						"reorgLength": j,
						"startBlock":  k,
						"sprintSize":  sprintSizes[i],
						"faultyNode":  faultyNodes[l], // node 1(index) is primary validator of the first sprint
					}
					reorgsLengthTests = append(reorgsLengthTests, reorgsLengthTest)
				}
			}
		}
	}
	// reorgsLengthTests := []map[string]uint64{
	// 	{
	// 		"reorgLength": 3,
	// 		"startBlock":  7,
	// 		"sprintSize":  8,
	// 		"faultyNode":  1,
	// 	},
	// }
	return reorgsLengthTests
}

func SprintLengthReorgIndividual(t *testing.T, index int, tt map[string]uint64) (uint64, uint64, uint64, uint64, uint64, uint64) {
	t.Helper()

	log.Warn("Case ----- ", "Index", index, "InducedReorgLength", tt["reorgLength"], "BlockStart", tt["startBlock"], "SprintSize", tt["sprintSize"], "DisconnectedNode", tt["faultyNode"])
	observerOldChainLength, faultyOldChainLength := SetupValidatorsAndTest(t, tt)

	if observerOldChainLength > 0 {
		log.Warn("Observer", "Old Chain length", observerOldChainLength)
	}

	if faultyOldChainLength > 0 {
		log.Warn("Faulty", "Old Chain length", faultyOldChainLength)
	}

	return tt["reorgLength"], tt["startBlock"], tt["sprintSize"], tt["faultyNode"], faultyOldChainLength, observerOldChainLength
}

func SprintLengthReorgIndividual2Nodes(t *testing.T, index int, tt map[string]interface{}) (uint64, uint64, uint64, []uint64, uint64, uint64) {
	t.Helper()

	log.Warn("Case ----- ", "Index", index, "InducedReorgLength", tt["reorgLength"], "BlockStart", tt["startBlock"], "SprintSize", tt["sprintSize"], "DisconnectedNode", tt["faultyNodes"])
	observerOldChainLength, faultyOldChainLength := SetupValidatorsAndTest2Nodes(t, tt)

	if observerOldChainLength > 0 {
		log.Warn("Observer", "Old Chain length", observerOldChainLength)
	}

	if faultyOldChainLength > 0 {
		log.Warn("Faulty", "Old Chain length", faultyOldChainLength)
	}

	fNodes, _ := tt["faultyNodes"].([]uint64)

	return tt["reorgLength"].(uint64), tt["startBlock"].(uint64), tt["sprintSize"].(uint64), fNodes, faultyOldChainLength, observerOldChainLength
}

func TestSprintLengthReorg2Nodes(t *testing.T) {
	t.Skip()
	// t.Parallel()

	log.Root().SetHandler(log.LvlFilterHandler(3, log.StreamHandler(os.Stderr, log.TerminalFormat(true))))

	_, err := fdlimit.Raise(2048)

	if err != nil {
		panic(err)
	}

	reorgsLengthTests := getTestSprintLengthReorgCases2Nodes()
	f, err := os.Create("sprintReorg2Nodes.csv")

	defer func() {
		err = f.Close()

		if err != nil {
			panic(err)
		}
	}()

	if err != nil {
		_log.Fatalln("failed to open file", err)
	}

	w := csv.NewWriter(f)
	err = w.Write([]string{"Induced Reorg Length", "Start Block", "Sprint Size", "Disconnected Node Ids", "Disconnected Node Id's Reorg Length", "Observer Node Id's Reorg Length"})
	w.Flush()

	if err != nil {
		panic(err)
	}

	var wg sync.WaitGroup

	for index, tt := range reorgsLengthTests {
		if index%4 == 0 {
			wg.Wait()
		}

		wg.Add(1)

		go SprintLengthReorgIndividual2NodesHelper(t, index, tt, w, &wg)
	}
}

func TestSprintLengthReorg(t *testing.T) {
	t.Skip()
	// t.Parallel()

	reorgsLengthTests := getTestSprintLengthReorgCases()
	f, err := os.Create("sprintReorg.csv")

	defer func() {
		err = f.Close()

		if err != nil {
			panic(err)
		}
	}()

	if err != nil {
		_log.Fatalln("failed to open file", err)
	}

	w := csv.NewWriter(f)
	err = w.Write([]string{"Induced Reorg Length", "Start Block", "Sprint Size", "Disconnected Node Id", "Disconnected Node Id's Reorg Length", "Observer Node Id's Reorg Length"})
	w.Flush()

	if err != nil {
		panic(err)
	}

	var wg sync.WaitGroup

	for index, tt := range reorgsLengthTests {
		if index%4 == 0 {
			wg.Wait()
		}

		wg.Add(1)

		go SprintLengthReorgIndividualHelper(t, index, tt, w, &wg)
	}
}

func SprintLengthReorgIndividualHelper(t *testing.T, index int, tt map[string]uint64, w *csv.Writer, wg *sync.WaitGroup) {
	t.Helper()

	r1, r2, r3, r4, r5, r6 := SprintLengthReorgIndividual(t, index, tt)
	err := w.Write([]string{fmt.Sprint(r1), fmt.Sprint(r2), fmt.Sprint(r3), fmt.Sprint(r4), fmt.Sprint(r5), fmt.Sprint(r6)})

	if err != nil {
		panic(err)
	}

	w.Flush()
	(*wg).Done()
}

func SprintLengthReorgIndividual2NodesHelper(t *testing.T, index int, tt map[string]interface{}, w *csv.Writer, wg *sync.WaitGroup) {
	t.Helper()

	r1, r2, r3, r4, r5, r6 := SprintLengthReorgIndividual2Nodes(t, index, tt)
	err := w.Write([]string{fmt.Sprint(r1), fmt.Sprint(r2), fmt.Sprint(r3), fmt.Sprint(r4), fmt.Sprint(r5), fmt.Sprint(r6)})

	if err != nil {
		panic(err)
	}

	w.Flush()
	(*wg).Done()
}

// nolint: gocognit
func SetupValidatorsAndTest(t *testing.T, tt map[string]uint64) (uint64, uint64) {
	t.Helper()

	log.Root().SetHandler(log.LvlFilterHandler(3, log.StreamHandler(os.Stderr, log.TerminalFormat(true))))

	_, err := fdlimit.Raise(2048)

	if err != nil {
		panic(err)
	}

	// Generate a batch of accounts to seal and fund with
	faucets := make([]*ecdsa.PrivateKey, 128)
	for i := 0; i < len(faucets); i++ {
		faucets[i], _ = crypto.GenerateKey()
	}

	// Create an Ethash network based off of the Ropsten config
	// Generate a batch of accounts to seal and fund with
	genesis := InitGenesisSprintLength(t, faucets, "./testdata/genesis_7val.json", tt["sprintSize"])

	nodes := make([]*eth.Ethereum, len(keys_21val))
	enodes := make([]*enode.Node, len(keys_21val))
	stacks := make([]*node.Node, len(keys_21val))

	pkeys_21val := make([]*ecdsa.PrivateKey, len(keys_21val))

	for index, signerdata := range keys_21val {
		pkeys_21val[index], _ = crypto.HexToECDSA(signerdata["priv_key"])
	}

	for i := 0; i < len(keys_21val); i++ {
		// Start the node and wait until it's up
		stack, ethBackend, err := InitMinerSprintLength(genesis, pkeys_21val[i], true)
		if err != nil {
			panic(err)
		}
		defer stack.Close()

		for stack.Server().NodeInfo().Ports.Listener == 0 {
			time.Sleep(250 * time.Millisecond)
		}
		// Connect the node to all the previous ones
		for j, n := range enodes {
			if j < i {
				stack.Server().AddPeer(n)
			}
		}
		// Start tracking the node and its enode
		stacks[i] = stack
		nodes[i] = ethBackend
		enodes[i] = stack.Server().Self()
	}

	// Iterate over all the nodes and start mining
	time.Sleep(3 * time.Second)

	for _, node := range nodes {
		if err := node.StartMining(); err != nil {
			panic(err)
		}
	}

	chain2HeadChObserver := make(chan core.Chain2HeadEvent, 64)
	chain2HeadChFaulty := make(chan core.Chain2HeadEvent, 64)

	var observerOldChainLength, faultyOldChainLength uint64

	faultyProducerIndex := tt["faultyNode"] // node causing reorg :: faulty ::
	subscribedNodeIndex := 6                // node on different partition, produces 7th sprint but our testcase does not run till 7th sprint. :: observer ::

	nodes[subscribedNodeIndex].BlockChain().SubscribeChain2HeadEvent(chain2HeadChObserver)
	nodes[faultyProducerIndex].BlockChain().SubscribeChain2HeadEvent(chain2HeadChFaulty)

	stacks[faultyProducerIndex].Server().NoDiscovery = true

	for {
		blockHeaderObserver := nodes[subscribedNodeIndex].BlockChain().CurrentHeader()
		blockHeaderFaulty := nodes[faultyProducerIndex].BlockChain().CurrentHeader()

		log.Warn("Current Observer block", "number", blockHeaderObserver.Number, "hash", blockHeaderObserver.Hash())
		log.Warn("Current Faulty block", "number", blockHeaderFaulty.Number, "hash", blockHeaderFaulty.Hash())

		if blockHeaderFaulty.Number.Uint64() == tt["startBlock"] {
			stacks[faultyProducerIndex].Server().MaxPeers = 0

			for _, enode := range enodes {
				stacks[faultyProducerIndex].Server().RemovePeer(enode)
			}
		}

		if blockHeaderObserver.Number.Uint64() >= tt["startBlock"] && blockHeaderObserver.Number.Uint64() < tt["startBlock"]+tt["reorgLength"] {
			for _, enode := range enodes {
				stacks[faultyProducerIndex].Server().RemovePeer(enode)
			}
		}

		if blockHeaderObserver.Number.Uint64() == tt["startBlock"]+tt["reorgLength"] {
			stacks[faultyProducerIndex].Server().NoDiscovery = false
			stacks[faultyProducerIndex].Server().MaxPeers = 100

			for _, enode := range enodes {
				stacks[faultyProducerIndex].Server().AddPeer(enode)
			}
		}

		if blockHeaderFaulty.Number.Uint64() >= 255 {
			break
		}

		select {
		case ev := <-chain2HeadChObserver:
			if ev.Type == core.Chain2HeadReorgEvent {
				if len(ev.OldChain) > 1 {
					observerOldChainLength = uint64(len(ev.OldChain))
					return observerOldChainLength, 0
				}
			}

		case ev := <-chain2HeadChFaulty:
			if ev.Type == core.Chain2HeadReorgEvent {
				if len(ev.OldChain) > 1 {
					faultyOldChainLength = uint64(len(ev.OldChain))
					return 0, faultyOldChainLength
				}
			}

		default:
			time.Sleep(500 * time.Millisecond)
		}
	}

	return 0, 0
}

// nolint: gocognit
func SetupValidatorsAndTest2Nodes(t *testing.T, tt map[string]interface{}) (uint64, uint64) {
	t.Helper()

	log.Root().SetHandler(log.LvlFilterHandler(3, log.StreamHandler(os.Stderr, log.TerminalFormat(true))))

	_, err := fdlimit.Raise(2048)

	if err != nil {
		panic(err)
	}

	// Generate a batch of accounts to seal and fund with
	faucets := make([]*ecdsa.PrivateKey, 128)
	for i := 0; i < len(faucets); i++ {
		faucets[i], _ = crypto.GenerateKey()
	}

	// Create an Ethash network based off of the Ropsten config
	// Generate a batch of accounts to seal and fund with
	genesis := InitGenesisSprintLength(t, faucets, "./testdata/genesis_7val.json", tt["sprintSize"].(uint64))

	nodes := make([]*eth.Ethereum, len(keys_21val))
	enodes := make([]*enode.Node, len(keys_21val))
	stacks := make([]*node.Node, len(keys_21val))

	pkeys_21val := make([]*ecdsa.PrivateKey, len(keys_21val))

	for index, signerdata := range keys_21val {
		pkeys_21val[index], _ = crypto.HexToECDSA(signerdata["priv_key"])
	}

	for i := 0; i < len(keys_21val); i++ {
		// Start the node and wait until it's up
		stack, ethBackend, err := InitMinerSprintLength(genesis, pkeys_21val[i], true)
		if err != nil {
			panic(err)
		}
		defer stack.Close()

		for stack.Server().NodeInfo().Ports.Listener == 0 {
			time.Sleep(250 * time.Millisecond)
		}
		// Connect the node to all the previous ones
		for j, n := range enodes {
			if j < i {
				stack.Server().AddPeer(n)
			}
		}
		// Start tracking the node and its enode
		stacks[i] = stack
		nodes[i] = ethBackend
		enodes[i] = stack.Server().Self()
	}

	// Iterate over all the nodes and start mining
	time.Sleep(3 * time.Second)

	for _, node := range nodes {
		if err := node.StartMining(); err != nil {
			panic(err)
		}
	}

	chain2HeadChObserver := make(chan core.Chain2HeadEvent, 64)
	chain2HeadChFaulty := make(chan core.Chain2HeadEvent, 64)

	var observerOldChainLength, faultyOldChainLength uint64

	faultyProducerIndex := tt["faultyNodes"].([]uint64)[0] // node causing reorg :: faulty ::
	subscribedNodeIndex := 6                               // node on different partition, produces 7th sprint but our testcase does not run till 7th sprint. :: observer ::

	nodes[subscribedNodeIndex].BlockChain().SubscribeChain2HeadEvent(chain2HeadChObserver)
	nodes[faultyProducerIndex].BlockChain().SubscribeChain2HeadEvent(chain2HeadChFaulty)

	stacks[faultyProducerIndex].Server().NoDiscovery = true

	for {
		blockHeaderObserver := nodes[subscribedNodeIndex].BlockChain().CurrentHeader()
		blockHeaderFaulty := nodes[faultyProducerIndex].BlockChain().CurrentHeader()

		log.Warn("Current Observer block", "number", blockHeaderObserver.Number, "hash", blockHeaderObserver.Hash())
		log.Warn("Current Faulty block", "number", blockHeaderFaulty.Number, "hash", blockHeaderFaulty.Hash())

		if blockHeaderObserver.Number.Uint64() >= tt["startBlock"].(uint64) && blockHeaderObserver.Number.Uint64() < tt["startBlock"].(uint64)+tt["reorgLength"].(uint64) {
			for _, n := range tt["faultyNodes"].([]uint64) {
				stacks[n].Server().MaxPeers = 1

				for _, enode := range enodes {
					stacks[n].Server().RemovePeer(enode)
				}

				for _, m := range tt["faultyNodes"].([]uint64) {
					stacks[m].Server().AddPeer(enodes[n])
				}
			}
		}

		if blockHeaderObserver.Number.Uint64() == tt["startBlock"].(uint64)+tt["reorgLength"].(uint64) {
			stacks[faultyProducerIndex].Server().NoDiscovery = false
			stacks[faultyProducerIndex].Server().MaxPeers = 100

			for _, enode := range enodes {
				stacks[faultyProducerIndex].Server().AddPeer(enode)
			}
		}

		if blockHeaderFaulty.Number.Uint64() >= 255 {
			break
		}

		select {
		case ev := <-chain2HeadChObserver:
			if ev.Type == core.Chain2HeadReorgEvent {
				if len(ev.OldChain) > 1 {
					observerOldChainLength = uint64(len(ev.OldChain))
					return observerOldChainLength, 0
				}
			}

		case ev := <-chain2HeadChFaulty:
			if ev.Type == core.Chain2HeadReorgEvent {
				if len(ev.OldChain) > 1 {
					faultyOldChainLength = uint64(len(ev.OldChain))
					return 0, faultyOldChainLength
				}
			}

		default:
			time.Sleep(500 * time.Millisecond)
		}
	}

	return 0, 0
}

func InitGenesisSprintLength(t *testing.T, faucets []*ecdsa.PrivateKey, fileLocation string, sprintSize uint64) *core.Genesis {
	t.Helper()

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
	genesis.Config.Bor.Sprint["0"] = sprintSize

	return genesis
}

func InitMinerSprintLength(genesis *core.Genesis, privKey *ecdsa.PrivateKey, withoutHeimdall bool) (*node.Node, *eth.Ethereum, error) {
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
		GPO:             ethconfig.Defaults.GPO,
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

	_, err = kStore.ImportECDSA(privKey, "")

	if err != nil {
		return nil, nil, err
	}

	acc := kStore.Accounts()[0]
	err = kStore.Unlock(acc, "")

	if err != nil {
		return nil, nil, err
	}

	// proceed to authorize the local account manager in any case
	ethBackend.AccountManager().AddBackend(kStore)

	err = stack.Start()

	return stack, ethBackend, err
}
