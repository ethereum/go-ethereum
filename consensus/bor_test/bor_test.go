package bor_test

import (
	"encoding/hex"
	"encoding/json"
	"io/ioutil"
	"math/big"
	"testing"

	"github.com/maticnetwork/bor/common"
	"github.com/maticnetwork/bor/consensus/bor"
	"github.com/maticnetwork/bor/core"
	"github.com/maticnetwork/bor/core/rawdb"
	"github.com/maticnetwork/bor/core/state"
	"github.com/maticnetwork/bor/crypto/secp256k1"
	"github.com/stretchr/testify/assert"

	"github.com/maticnetwork/bor/core/types"
	"github.com/maticnetwork/bor/crypto"
	"github.com/maticnetwork/bor/eth"

	"github.com/maticnetwork/bor/ethdb"
	"github.com/maticnetwork/bor/mocks"
	"github.com/maticnetwork/bor/node"
)

var (
	extraSeal = 65 // Fixed number of extra-data suffix bytes reserved for signer seal
	privKey   = "b71c71a67e1177ad4e901695e1b4b9ee17ae16c6668d313eac2f96dbcda3f291"
	key, _    = crypto.HexToECDSA("b71c71a67e1177ad4e901695e1b4b9ee17ae16c6668d313eac2f96dbcda3f291")
	addr      = crypto.PubkeyToAddress(key.PublicKey) // 0x71562b71999873DB5b286dF957af199Ec94617F7
)

type initializeData struct {
	genesis  *core.Genesis
	ethereum *eth.Ethereum
}

func TestCommitSpan(t *testing.T) {
	init := buildEthereumInstance(t, rawdb.NewMemoryDatabase())
	chain := init.ethereum.BlockChain()
	engine := init.ethereum.Engine()
	_bor := engine.(*bor.Bor)

	// Mock HeimdallClient.FetchWithRetry to return span data from span.json
	res, heimdallSpan := loadSpanFromFile(t)
	h := &mocks.IHeimdallClient{}
	h.On("FetchWithRetry", "bor", "span", "1").
		Return(res, nil).
		Times(2) // both FinalizeAndAssemble and chain.InsertChain call HeimdallClient.FetchWithRetry. @todo Investigate this in depth
	_bor.SetHeimdallClient(h)

	db := init.ethereum.ChainDb()
	block := init.genesis.ToBlock(db)
	// Build 1st block's header
	header := buildMinimalNextHeader(t, block, init.genesis.Config.Bor.Period)

	statedb, err := chain.State()
	if err != nil {
		t.Fatalf("%s", err)
	}

	_key, _ := hex.DecodeString(privKey)
	insertNewBlock(t, _bor, chain, header, statedb, _key)

	assert.True(t, h.AssertNumberOfCalls(t, "FetchWithRetry", 2))
	validators, err := _bor.GetCurrentValidators(1, 256) // new span starts at 256
	if err != nil {
		t.Fatalf("%s", err)
	}

	assert.Equal(t, len(validators), 3)
	for i, validator := range validators {
		assert.Equal(t, validator.Address.Bytes(), heimdallSpan.SelectedProducers[i].Address.Bytes())
		assert.Equal(t, validator.VotingPower, heimdallSpan.SelectedProducers[i].VotingPower)
	}
}

func TestIsValidatorAction(t *testing.T) {
	init := buildEthereumInstance(t, rawdb.NewMemoryDatabase())
	chain := init.ethereum.BlockChain()
	engine := init.ethereum.Engine()
	_bor := engine.(*bor.Bor)

	proposeStateData, _ := hex.DecodeString("ede01f170000000000000000000000000000000000000000000000000000000000000000")
	proposeSpanData, _ := hex.DecodeString("4b0e4d17")
	var tx *types.Transaction
	tx = types.NewTransaction(
		0,
		common.HexToAddress(chain.Config().Bor.StateReceiverContract),
		big.NewInt(0), 0, big.NewInt(0),
		proposeStateData,
	)
	assert.True(t, _bor.IsValidatorAction(chain, addr, tx))

	tx = types.NewTransaction(
		0,
		common.HexToAddress(chain.Config().Bor.ValidatorContract),
		big.NewInt(0), 0, big.NewInt(0),
		proposeSpanData,
	)
	assert.True(t, _bor.IsValidatorAction(chain, addr, tx))

	res, heimdallSpan := loadSpanFromFile(t)
	h := &mocks.IHeimdallClient{}
	h.On("FetchWithRetry", "bor", "span", "1").Return(res, nil)
	_bor.SetHeimdallClient(h)

	// Build 1st block's header
	db := init.ethereum.ChainDb()
	block := init.genesis.ToBlock(db)

	header := buildMinimalNextHeader(t, block, init.genesis.Config.Bor.Period)
	statedb, err := chain.State()
	if err != nil {
		t.Fatalf("%s", err)
	}

	_key, _ := hex.DecodeString(privKey)
	insertNewBlock(t, _bor, chain, header, statedb, _key)
	block = types.NewBlockWithHeader(header)

	var headers []*types.Header
	for i := int64(2); i <= 255; i++ {
		header := buildMinimalNextHeader(t, block, init.genesis.Config.Bor.Period)
		headers = append(headers, header)
		block = types.NewBlockWithHeader(header)
	}
	t.Logf("inserting %v headers", len(headers))
	if _, err := chain.InsertHeaderChain(headers, 0); err != nil {
		t.Fatalf("%s", err)
	}

	for _, validator := range heimdallSpan.SelectedProducers {
		_addr := validator.Address
		tx = types.NewTransaction(
			0,
			common.HexToAddress(chain.Config().Bor.StateReceiverContract),
			big.NewInt(0), 0, big.NewInt(0),
			proposeStateData,
		)
		assert.True(t, _bor.IsValidatorAction(chain, _addr, tx))

		tx = types.NewTransaction(
			0,
			common.HexToAddress(chain.Config().Bor.ValidatorContract),
			big.NewInt(0), 0, big.NewInt(0),
			proposeSpanData,
		)
		assert.True(t, _bor.IsValidatorAction(chain, _addr, tx))
	}
}

func buildEthereumInstance(t *testing.T, db ethdb.Database) *initializeData {
	genesisData, err := ioutil.ReadFile("genesis.json")
	if err != nil {
		t.Fatalf("%s", err)
	}
	gen := &core.Genesis{}
	if err := json.Unmarshal(genesisData, gen); err != nil {
		t.Fatalf("%s", err)
	}
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
		genesis:  gen,
		ethereum: ethereum,
	}
}

func insertNewBlock(t *testing.T, _bor *bor.Bor, chain *core.BlockChain, header *types.Header, statedb *state.StateDB, privKey []byte) {
	_, err := _bor.FinalizeAndAssemble(chain, header, statedb, nil, nil, nil)
	if err != nil {
		t.Fatalf("%s", err)
	}

	sig, err := secp256k1.Sign(crypto.Keccak256(bor.BorRLP(header)), privKey)
	if err != nil {
		t.Fatalf("%s", err)
	}
	copy(header.Extra[len(header.Extra)-extraSeal:], sig)

	block := types.NewBlockWithHeader(header)
	if _, err := chain.InsertChain([]*types.Block{block}); err != nil {
		t.Fatalf("%s", err)
	}
}

func buildMinimalNextHeader(t *testing.T, block *types.Block, period uint64) *types.Header {
	header := block.Header()
	header.Number.Add(header.Number, big.NewInt(1))
	header.ParentHash = block.Hash()
	header.Time += (period + 1)
	header.Extra = make([]byte, 97) // vanity (32) + extraSeal (65)
	_key, _ := hex.DecodeString(privKey)
	sig, err := secp256k1.Sign(crypto.Keccak256(bor.BorRLP(header)), _key)
	if err != nil {
		t.Fatalf("%s", err)
	}
	copy(header.Extra[len(header.Extra)-extraSeal:], sig)
	return header
}

func loadSpanFromFile(t *testing.T) (*bor.ResponseWithHeight, *bor.HeimdallSpan) {
	spanData, err := ioutil.ReadFile("span.json")
	if err != nil {
		t.Fatalf("%s", err)
	}
	res := &bor.ResponseWithHeight{}
	if err := json.Unmarshal(spanData, res); err != nil {
		t.Fatalf("%s", err)
	}

	heimdallSpan := &bor.HeimdallSpan{}
	if err := json.Unmarshal(res.Result, heimdallSpan); err != nil {
		t.Fatalf("%s", err)
	}
	return res, heimdallSpan
}
