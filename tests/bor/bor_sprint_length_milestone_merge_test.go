//go:build integration

// nolint
package bor

import (
	"crypto/ecdsa"
	"encoding/csv"
	"fmt" // nolint: staticcheck
	_log "log"
	"math"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/fdlimit"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/eth"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/node"
	"github.com/ethereum/go-ethereum/p2p/enode"
)

var (

	// Only this account is a validator for the 0th span
	keySprintLength_Milestone, _ = crypto.HexToECDSA(privKeySprintLength_Milestone)

	// This account is one the validators for 1st span (0-indexed)
	keySprintLength_Milestone2, _ = crypto.HexToECDSA(privKeySprintLength_Milestone2)

	keysSprintLength_Milestone = []*ecdsa.PrivateKey{keySprintLength_Milestone, keySprintLength_Milestone2}
)

const (
	privKeySprintLength_Milestone  = "b71c71a67e1177ad4e901695e1b4b9ee17ae16c6668d313eac2f96dbcda3f291"
	privKeySprintLength_Milestone2 = "9b28f36fbd67381120752d6172ecdcf10e06ab2d9a1367aac00cdcd6ac7855d3"
)

var keys_21validator = []map[string]string{
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

func getTestSprintLengthMilestoneReorgCases() []map[string]uint64 {
	faultyNodes := []int64{2, 4}
	milestoneMark := []int64{48, 72}
	sprintLen := int64(16)
	milestoneLen := int64(24)
	reorgsLengthTests := make([]map[string]uint64, 0)

	for i := int64(0); i < int64(len(faultyNodes)); i++ {
		for j := int64(1); j < int64(14); j = j + 2 {
			startBlock := faultyNodes[i]*sprintLen + j
			diff := milestoneMark[i] - startBlock
			if diff <= 0 {
				break
			}
			a := diff*2 - 2
			for k := diff + 1; k < a; k = k + 4 {
				for l := 0; l < 2; l++ {

					reorgsLengthTest := map[string]uint64{
						"reorgLength":     uint64(k),
						"startBlock":      uint64(startBlock),
						"sprintSize":      uint64(sprintLen),
						"faultyNode":      uint64(faultyNodes[i]), // node 1(index) is primary validator of the first sprint
						"milestoneLength": uint64(milestoneLen),
						"milestoneFlag":   uint64(l),
					}

					reorgsLengthTests = append(reorgsLengthTests, reorgsLengthTest)
				}
			}
		}
	}
	return reorgsLengthTests
}

func getTestSprintLengthMilestoneReorgCases2Nodes() []map[string]interface{} {
	sprintSizes := []uint64{16, 32, 64}
	faultyNodes := [][]uint64{{0, 1}, {1, 2}, {0, 2}}
	milestoneLength := []uint64{16, 32, 64}
	reorgsLengthTests := make([]map[string]interface{}, 0)

	for i := uint64(0); i < uint64(len(sprintSizes)); i++ {
		for m := uint64(0); m < uint64(len(milestoneLength)); m++ {
			maxReorgLength := sprintSizes[i] * 4
			for j := uint64(20); j <= maxReorgLength; j = j + 8 {
				maxStartBlock := sprintSizes[i] - 1
				for k := sprintSizes[i] / 2; k <= maxStartBlock; k = k + 8 {
					for l := uint64(0); l < uint64(len(faultyNodes)); l++ {
						if j+k < sprintSizes[i] {
							continue
						}

						reorgsLengthTest := map[string]interface{}{
							"reorgLength":     j,
							"startBlock":      k,
							"sprintSize":      sprintSizes[i],
							"faultyNodes":     faultyNodes[l], // node 1(index) is primary validator of the first sprint
							"milestoneLength": milestoneLength[m],
						}
						reorgsLengthTests = append(reorgsLengthTests, reorgsLengthTest)
					}
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
	//      "milestoneLength": 32
	// 	},
	// }
	return reorgsLengthTests
}

func SprintLengthMilestoneReorgIndividual(t *testing.T, index int, tt map[string]uint64) (uint64, uint64, uint64, uint64, uint64, uint64, uint64, uint64) {
	t.Helper()

	log.Warn("Case ----- ", "Index", index, "InducedReorgLength", tt["reorgLength"], "BlockStart", tt["startBlock"], "SprintSize", tt["startBlock"], "MilestoneFlag", tt["milestoneFlag"], "MilestoneLength", tt["milestoneLength"], "DisconnectedNode", tt["faultyNode"])
	observerOldChainLength, faultyOldChainLength := SetupValidatorsAndTestSprintLengthMilestone(t, tt)

	if observerOldChainLength > 0 {
		log.Warn("Observer", "Old Chain length", observerOldChainLength)
	}

	if faultyOldChainLength > 0 {
		log.Warn("Faulty", "Old Chain length", faultyOldChainLength)
	}

	return tt["reorgLength"], tt["startBlock"], tt["sprintSize"], tt["milestoneFlag"], tt["milestoneLength"], tt["faultyNode"], faultyOldChainLength, observerOldChainLength
}

func SprintLengthMilestoneReorgIndividual2Nodes(t *testing.T, index int, tt map[string]interface{}) (uint64, uint64, uint64, uint64, []uint64, uint64, uint64) {
	t.Helper()

	log.Warn("Case ----- ", "Index", index, "InducedReorgLength", tt["reorgLength"], "BlockStart", tt["startBlock"], "SprintSize", tt["sprintSize"], "DisconnectedNode", tt["faultyNodes"])
	observerOldChainLength, faultyOldChainLength := SetupValidatorsAndTest2NodesSprintLengthMilestone(t, tt)

	if observerOldChainLength > 0 {
		log.Warn("Observer", "Old Chain length", observerOldChainLength)
	}

	if faultyOldChainLength > 0 {
		log.Warn("Faulty", "Old Chain length", faultyOldChainLength)
	}

	fNodes, _ := tt["faultyNodes"].([]uint64)

	return tt["reorgLength"].(uint64), tt["startBlock"].(uint64), tt["sprintSize"].(uint64), tt["milestoneLength"].(uint64), fNodes, faultyOldChainLength, observerOldChainLength
}

func TestSprintLengthMilestoneReorg2Nodes(t *testing.T) {
	t.Skip()
	// t.Parallel()

	log.Root().SetHandler(log.LvlFilterHandler(3, log.StreamHandler(os.Stderr, log.TerminalFormat(true))))

	_, err := fdlimit.Raise(2048)

	if err != nil {
		panic(err)
	}

	reorgsLengthTests := getTestSprintLengthMilestoneReorgCases2Nodes()
	f, err := os.Create("sprintReorgMilestone2Nodes.csv")

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
	err = w.Write([]string{"Induced Reorg Length", "Start Block", "Sprint Size", "MilestoneLength", "Disconnected Node Ids", "Disconnected Node Id's Rewind Length", "Observer Node Id's Reorg Length"})
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

		go SprintLengthMilestoneReorgIndividual2NodesHelper(t, index, tt, w, &wg)
	}
}

func TestSprintLengthMilestoneReorg(t *testing.T) {
	t.Skip()
	// t.Parallel()

	log.Root().SetHandler(log.LvlFilterHandler(3, log.StreamHandler(os.Stderr, log.TerminalFormat(true))))

	_, err := fdlimit.Raise(2048)

	if err != nil {
		panic(err)
	}

	reorgsLengthTests := getTestSprintLengthMilestoneReorgCases()
	f, err := os.Create("sprintMilestoneReorg.csv")

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
	err = w.Write([]string{"Induced Reorg Length", "Start Block", "Sprint Size", "MilestoneFlag", "Milestone Length", "Disconnected Node Id", "Disconnected Node Id's Rewind Length", "Observer Node Id's Reorg Length"})

	w.Flush()
	if err != nil {
		panic(err)
	}

	err = w.Write([]string{fmt.Sprint(len(reorgsLengthTests))})

	var wg sync.WaitGroup
	for index, tt := range reorgsLengthTests {
		if index%4 == 0 {
			wg.Wait()
		}

		wg.Add(1)

		go SprintLengthMilestoneReorgIndividualHelper(t, index, tt, w, &wg)
	}
}

func SprintLengthMilestoneReorgIndividualHelper(t *testing.T, index int, tt map[string]uint64, w *csv.Writer, wg *sync.WaitGroup) {
	t.Helper()

	r1, r2, r3, r4, r5, r6, r7, r8 := SprintLengthMilestoneReorgIndividual(t, index, tt)
	err := w.Write([]string{fmt.Sprint(r1), fmt.Sprint(r2), fmt.Sprint(r3), fmt.Sprint(r4), fmt.Sprint(r5), fmt.Sprint(r6), fmt.Sprint(r7), fmt.Sprint(r8)})

	if err != nil {
		panic(err)
	}

	w.Flush()
	(*wg).Done()
}

func SprintLengthMilestoneReorgIndividual2NodesHelper(t *testing.T, index int, tt map[string]interface{}, w *csv.Writer, wg *sync.WaitGroup) {
	t.Helper()

	r1, r2, r3, r4, r5, r6, r7 := SprintLengthMilestoneReorgIndividual2Nodes(t, index, tt)
	err := w.Write([]string{fmt.Sprint(r1), fmt.Sprint(r2), fmt.Sprint(r3), fmt.Sprint(r4), fmt.Sprint(r5), fmt.Sprint(r6), fmt.Sprint(r7)})

	if err != nil {
		panic(err)
	}

	w.Flush()
	(*wg).Done()
}

// nolint: gocognit
func SetupValidatorsAndTest2NodesSprintLengthMilestone(t *testing.T, tt map[string]interface{}) (uint64, uint64) {
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
	genesis := InitGenesis(t, faucets, "./testdata/genesis_7val.json", tt["sprintSize"].(uint64))

	nodes := make([]*eth.Ethereum, len(keys_21val))
	enodes := make([]*enode.Node, len(keys_21val))
	stacks := make([]*node.Node, len(keys_21val))

	pkeys_21val := make([]*ecdsa.PrivateKey, len(keys_21val))

	for index, signerdata := range keys_21val {
		pkeys_21val[index], _ = crypto.HexToECDSA(signerdata["priv_key"])
	}

	for i := 0; i < len(keys_21val); i++ {
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

	milestoneLength := tt["milestoneLength"].(uint64)

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

		if math.Mod(float64(blockHeaderObserver.Number.Uint64()), float64(milestoneLength)) == 0 {
			blockHash := blockHeaderObserver.Hash()
			nodes[subscribedNodeIndex].Downloader().ChainValidator.ProcessMilestone(blockHeaderObserver.Number.Uint64(), blockHash)
		}

		if blockHeaderObserver.Number.Uint64() == tt["startBlock"].(uint64)+tt["reorgLength"].(uint64) {
			stacks[faultyProducerIndex].Server().NoDiscovery = false
			stacks[faultyProducerIndex].Server().MaxPeers = 100

			for _, enode := range enodes {
				stacks[faultyProducerIndex].Server().AddPeer(enode)
			}
		}

		if blockHeaderObserver.Number.Uint64() >= 255 {
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

func SetupValidatorsAndTestSprintLengthMilestone(t *testing.T, tt map[string]uint64) (uint64, uint64) {
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
	genesis := InitGenesis(t, faucets, "./testdata/genesis_7val.json", tt["sprintSize"])

	nodes := make([]*eth.Ethereum, len(keys_21val))
	enodes := make([]*enode.Node, len(keys_21val))
	stacks := make([]*node.Node, len(keys_21val))

	pkeys_21val := make([]*ecdsa.PrivateKey, len(keys_21val))

	for index, signerdata := range keys_21val {
		pkeys_21val[index], _ = crypto.HexToECDSA(signerdata["priv_key"])
	}

	for i := 0; i < len(keys_21val); i++ {
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

	milestoneLength := tt["milestoneLength"]
	milestoneFlag := tt["milestoneFlag"]

	nodes[subscribedNodeIndex].BlockChain().SubscribeChain2HeadEvent(chain2HeadChObserver)
	nodes[faultyProducerIndex].BlockChain().SubscribeChain2HeadEvent(chain2HeadChFaulty)

	stacks[faultyProducerIndex].Server().NoDiscovery = true

	var milestoneNum uint64 = 0
	var milestoneHash common.Hash
	var lastRun uint64 = 0

	for {
		blockHeaderObserver := nodes[subscribedNodeIndex].BlockChain().CurrentHeader()
		blockHeaderFaulty := nodes[faultyProducerIndex].BlockChain().CurrentHeader()

		log.Warn("Current Observer block", "number", blockHeaderObserver.Number, "hash", blockHeaderObserver.Hash())
		if blockHeaderFaulty != nil {
			log.Warn("Current Faulty block", "number", blockHeaderFaulty.Number, "hash", blockHeaderFaulty.Hash())
		}

		if blockHeaderFaulty.Number.Uint64() == tt["startBlock"] {
			stacks[faultyProducerIndex].Server().MaxPeers = 0

			for _, enode := range enodes {
				stacks[faultyProducerIndex].Server().RemovePeer(enode)
			}
		}

		if blockHeaderFaulty.Number.Uint64() >= tt["startBlock"] && blockHeaderFaulty.Number.Uint64() < tt["startBlock"]+tt["reorgLength"] {
			for _, enode := range enodes {
				stacks[faultyProducerIndex].Server().RemovePeer(enode)
			}
		}

		if milestoneFlag == 1 {
			if blockHeaderObserver.Number.Uint64() >= milestoneLength && math.Mod(float64(blockHeaderObserver.Number.Uint64()), float64(milestoneLength)) == 0 && blockHeaderObserver.Number.Uint64() > milestoneNum {
				milestoneNum = blockHeaderObserver.Number.Uint64()
				milestoneHash = blockHeaderObserver.Hash()
			}

			if blockHeaderObserver.Number.Uint64() > lastRun {
				for _, nodeTemp := range nodes {
					_, _, err := borVerifyTemP(nodeTemp, milestoneNum-milestoneLength+1, milestoneNum, milestoneHash.String())
					if err == nil {
						nodeTemp.Downloader().ChainValidator.ProcessMilestone(milestoneNum, milestoneHash)
					} else {
						nodeTemp.Downloader().ChainValidator.ProcessFutureMilestone(milestoneNum, milestoneHash)
					}
				}
			}
		}

		if blockHeaderFaulty.Number.Uint64() == tt["startBlock"]+tt["reorgLength"] {
			stacks[faultyProducerIndex].Server().NoDiscovery = false
			stacks[faultyProducerIndex].Server().MaxPeers = 100

			for _, enode := range enodes {
				stacks[faultyProducerIndex].Server().AddPeer(enode)
			}
		}

		if blockHeaderFaulty.Number.Uint64() >= 150 {
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

func borVerifyTemP(eth *eth.Ethereum, start uint64, end uint64, hash string) (string, uint64, error) {
	// check if we have the given blocks
	currentBlock := eth.BlockChain().CurrentBlock()

	if currentBlock == nil {
		log.Debug("Failed to fetch current block from blockchain while verifying incoming milestone")
		return hash, 0, errMissingBlocks
	}

	head := currentBlock.Number.Uint64()
	if head < end {
		log.Debug("Current head block behind incoming milestone block", "head", head, "end block", end)
		return hash, 0, errMissingBlocks
	}

	var localHash string

	block := eth.BlockChain().GetBlockByNumber(end)
	if block == nil {
		log.Debug("Failed to get end block hash while whitelisting milestone", "number", end)
		return hash, 0, errEndBlock
	}

	localHash = block.Hash().String()

	//nolint
	if localHash != hash {

		log.Warn("End block hash mismatch while whitelisting milestone", "expected", localHash, "got", hash)
		var (
			rewindTo uint64
			doExist  bool
		)

		if doExist, rewindTo, _ = eth.Downloader().GetWhitelistedMilestone(); doExist {

		} else if doExist, rewindTo, _ = eth.Downloader().GetWhitelistedCheckpoint(); doExist {

		} else {
			if start <= 0 {
				rewindTo = 0
			} else {
				rewindTo = start - 1
			}
		}

		if head-rewindTo > 255 {
			rewindTo = head - 255
		}

		rewindBackTemp(eth, rewindTo)

		return hash, rewindTo, errHashMismatch
	}

	return block.Hash().String(), 0, nil
}

// Stop the miner if the mining process is running and rewind back the chain
func rewindBackTemp(eth *eth.Ethereum, rewindTo uint64) {
	if eth.Miner().Mining() {
		ch := make(chan struct{})
		eth.Miner().Stop(ch)
		<-ch
		rewindTemp(eth, rewindTo)
		eth.StartMining()
	} else {

		rewindTemp(eth, rewindTo)

	}
}

func rewindTemp(eth *eth.Ethereum, rewindTo uint64) {
	log.Warn("Rewinding chain to :", rewindTo, "block number")
	err := eth.BlockChain().SetHead(rewindTo)

	if err != nil {
		log.Error("Error while rewinding the chain to", "Block Number", rewindTo, "Error", err)
	}
}
