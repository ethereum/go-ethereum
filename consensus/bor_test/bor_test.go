package bor_test

import (
	"encoding/hex"
	"encoding/json"
	"fmt"

	// "fmt"
	"io/ioutil"
	"math/big"
	"testing"

	"github.com/stretchr/testify/assert"
	// "github.com/stretchr/testify/mock"

	"github.com/maticnetwork/bor/common"
	"github.com/maticnetwork/bor/consensus/bor"
	"github.com/maticnetwork/bor/core"
	"github.com/maticnetwork/bor/core/rawdb"
	// "github.com/maticnetwork/bor/core/state"
	"github.com/maticnetwork/bor/core/types"
	// "github.com/maticnetwork/bor/core/vm"
	"github.com/maticnetwork/bor/crypto"
	"github.com/maticnetwork/bor/eth"

	"github.com/maticnetwork/bor/ethdb"
	"github.com/maticnetwork/bor/mocks"
	"github.com/maticnetwork/bor/node"
)

type initializeData struct {
	genesis *core.Genesis
	ethereum *eth.Ethereum
}

func TestCommitSpan(t *testing.T) {
	var (
		// db     = rawdb.NewMemoryDatabase()
		// key, _ = crypto.HexToECDSA("b71c71a67e1177ad4e901695e1b4b9ee17ae16c6668d313eac2f96dbcda3f291")
		// addr   = crypto.PubkeyToAddress(key.PublicKey)
	)
	init := buildEthereumInstance(t, rawdb.NewMemoryDatabase())
	chain := init.ethereum.BlockChain()
	engine := init.ethereum.Engine()
	_bor := engine.(*bor.Bor)

	spanData, err := ioutil.ReadFile("span.json")
	if err != nil {
		t.Fatalf("%s", err)
	}
	res := &bor.ResponseWithHeight{}
	if err := json.Unmarshal(spanData, res); err != nil {
		t.Fatalf("%s", err)
	}

	h := &mocks.IHeimdallClient{}
	h.On("FetchWithRetry", "bor", "span", "1").Return(res, nil)
	_bor.SetHeimdallClient(h)
	db := init.ethereum.ChainDb()
	block := init.genesis.ToBlock(db)
	statedb, err := chain.StateAt(block.Root())
	// statedb, _ := state.New(block.Root(), state.NewDatabase(db))
	// _, _, _, err = chain.Processor().Process(block, statedb, vm.Config{})
	// fmt.Println(err)
	// _bor.Finalize(chain, block.Header(), statedb, nil, nil)

	api := bor.NewBorApi(chain, _bor)
	validators, _ := api.GetCurrentValidators()
	fmt.Println(1, validators)

	header := block.Header()
	header.Number = big.NewInt(1)
	header.ParentHash = block.Hash()
	fmt.Println(block.Hash())
	// header.Hash =
	block = types.NewBlockWithHeader(header)

	fmt.Println("statedb.IntermediateRoot", statedb.IntermediateRoot(true))
	_bor.Finalize(chain, block.Header(), statedb, nil, nil)
	// _, _, _, err = chain.Processor().Process(block, statedb, vm.Config{})
	fmt.Println("statedb.IntermediateRoot 2", statedb.IntermediateRoot(true))
	statedb, err = chain.StateAt(block.Root())
	fmt.Println("statedb.IntermediateRoot 3", statedb.IntermediateRoot(true))

	if err != nil {
		t.Fatalf("%s", err)
	}
	// fmt.Println(header.Number)
	// _bor.Finalize(chain, header, statedb, nil, nil)

	// status, err := chain.writeBlockWithState(block, receipts, statedb)

	// chain.chainmu.Lock()
	td := big.NewInt(0)
	td.Add(block.Difficulty(), chain.GetTdByHash(block.ParentHash()))
	fmt.Println("td", td)

	rawdb.WriteTd(db,
		block.Hash(),
		block.NumberU64(),
		td,
	)
	rawdb.WriteBlock(db, block)
	root, err := statedb.Commit(false)
	fmt.Println("root", root)
	if err != nil {
		t.Fatalf("%s", err)
	}
	// if err := statedb.Reset(root); err != nil {
	// 	t.Fatalf("state reset after block %d failed: %v", block.NumberU64(), err)
	// }
	// blockchain.chainmu.Unlock()

	assert.True(t, h.AssertNumberOfCalls(t, "FetchWithRetry", 1))
	validators, _ = api.GetCurrentValidators()
	fmt.Println(2, validators)

	validators, _ = _bor.GetCurrentValidators(0, 255)
	fmt.Println(3, validators)

	validators, _ = _bor.GetCurrentValidators(0, 256)
	fmt.Println(4, validators)

	validators, _ = _bor.GetCurrentValidators(1, 256)
	fmt.Println(4, validators)
	// engine
}

func TestIsValidatorAction(t *testing.T) {
	var (
		db     = rawdb.NewMemoryDatabase()
		key, _ = crypto.HexToECDSA("b71c71a67e1177ad4e901695e1b4b9ee17ae16c6668d313eac2f96dbcda3f291")
		addr   = crypto.PubkeyToAddress(key.PublicKey)
	)
	init := buildEthereumInstance(t, db)
	ethereum := init.ethereum
	chain := ethereum.BlockChain()
	engine := ethereum.Engine()

	// proposeState
	data, _ := hex.DecodeString("ede01f170000000000000000000000000000000000000000000000000000000000000000")
	tx := types.NewTransaction(
		0,
		common.HexToAddress(chain.Config().Bor.StateReceiverContract),
		big.NewInt(0), 0, big.NewInt(0),
		data,
	)
	assert.True(t, engine.(*bor.Bor).IsValidatorAction(chain, addr, tx))

	// proposeSpan
	data, _ = hex.DecodeString("4b0e4d17")
	tx = types.NewTransaction(
		0,
		common.HexToAddress(chain.Config().Bor.ValidatorContract),
		big.NewInt(0), 0, big.NewInt(0),
		data,
	)
	assert.True(t, engine.(*bor.Bor).IsValidatorAction(chain, addr, tx))
}

func buildEthereumInstance(t *testing.T, db ethdb.Database) (*initializeData) {
	genesisData, err := ioutil.ReadFile("genesis.json")
	if err != nil {
		t.Fatalf("%s", err)
	}
	gen := &core.Genesis{}
	if err := json.Unmarshal(genesisData, gen); err != nil {
		t.Fatalf("%s", err)
	}
	// chainConfig, _, err := core.SetupGenesisBlock(db, gen)
	ethConf := &eth.Config{
		Genesis: gen,
	}
	ethConf.Genesis.MustCommit(db)

	// Create a temporary storage for the node keys and initialize it
	workspace, err := ioutil.TempDir("", "console-tester-")
	if err != nil {
		t.Fatalf("failed to create temporary keystore: %v", err)
	}

	// Create a networkless protocol stack and start an Ethereum service within
	stack, err := node.New(&node.Config{DataDir: workspace, UseLightweightKDF: true, Name: "console-tester"})
	if err != nil {
		t.Fatalf("failed to create node: %v", err)
	}
	err = stack.Register(func(ctx *node.ServiceContext) (node.Service, error) {
		s, err := eth.New(ctx, ethConf)
		return s, err
	})
	if err != nil {
		t.Fatalf("failed to register Ethereum protocol: %v", err)
	}

	// Start the node and assemble the JavaScript console around it
	if err = stack.Start(); err != nil {
		t.Fatalf("failed to start test stack: %v", err)
	}
	_, err = stack.Attach()
	if err != nil {
		t.Fatalf("failed to attach to node: %v", err)
	}

	var ethereum *eth.Ethereum
	stack.Service(&ethereum)
	ethConf.Genesis.MustCommit(ethereum.ChainDb())
	return &initializeData{
		genesis: gen,
		ethereum: ethereum,
	}
}
