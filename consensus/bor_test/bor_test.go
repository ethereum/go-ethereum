package bor_test

import (
	"encoding/hex"
	"encoding/json"
	"io/ioutil"
	"math/big"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/maticnetwork/bor/common"
	"github.com/maticnetwork/bor/consensus/bor"
	"github.com/maticnetwork/bor/core"
	"github.com/maticnetwork/bor/core/rawdb"
	"github.com/maticnetwork/bor/core/types"
	"github.com/maticnetwork/bor/crypto"
	"github.com/maticnetwork/bor/eth"

	"github.com/maticnetwork/bor/ethdb"
	"github.com/maticnetwork/bor/node"
)

func TestIsValidatorAction(t *testing.T) {
	var (
		db     = rawdb.NewMemoryDatabase()
		key, _ = crypto.HexToECDSA("b71c71a67e1177ad4e901695e1b4b9ee17ae16c6668d313eac2f96dbcda3f291")
		addr   = crypto.PubkeyToAddress(key.PublicKey)
	)
	ethereum := buildEthereumInstance(t, db)
	chain := ethereum.BlockChain()
	engine := ethereum.Engine()
	bor := engine.(*bor.Bor)

	// proposeState
	data, _ := hex.DecodeString("ede01f170000000000000000000000000000000000000000000000000000000000000000")
	tx := types.NewTransaction(
		0,
		common.HexToAddress(chain.Config().Bor.StateReceiverContract),
		big.NewInt(0), 0, big.NewInt(0),
		data,
	)
	assert.True(t, bor.IsValidatorAction(chain, addr, tx))

	// proposeSpan
	data, _ = hex.DecodeString("4b0e4d17")
	tx = types.NewTransaction(
		0,
		common.HexToAddress(chain.Config().Bor.ValidatorContract),
		big.NewInt(0), 0, big.NewInt(0),
		data,
	)
	assert.True(t, bor.IsValidatorAction(chain, addr, tx))
}

func buildEthereumInstance(t *testing.T, db ethdb.Database) *eth.Ethereum {
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
	return ethereum
}
