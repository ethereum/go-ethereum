package bortest

import (
	"encoding/hex"
	"encoding/json"
	"io/ioutil"
	"math/big"
	"testing"

	"github.com/maticnetwork/bor/consensus/bor"
	"github.com/maticnetwork/bor/core"
	"github.com/maticnetwork/bor/core/state"
	"github.com/maticnetwork/bor/core/types"
	"github.com/maticnetwork/bor/crypto"
	"github.com/maticnetwork/bor/crypto/secp256k1"
	"github.com/maticnetwork/bor/eth"
	"github.com/maticnetwork/bor/ethdb"
	"github.com/maticnetwork/bor/node"
	"github.com/maticnetwork/bor/params"
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

func buildMinimalNextHeader(t *testing.T, block *types.Block, borConfig *params.BorConfig) *types.Header {
	header := block.Header()
	header.Number.Add(header.Number, big.NewInt(1))
	header.ParentHash = block.Hash()
	header.Time += bor.CalcProducerDelay(header.Number.Uint64(), borConfig.Period, borConfig.Sprint, borConfig.ProducerDelay)
	isSprintEnd := (header.Number.Uint64()+1)%borConfig.Sprint == 0
	if isSprintEnd {
		header.Extra = make([]byte, 32+40+65) // vanity + validatorBytes + extraSeal
		// the genesis file was initialized with a validator 0x71562b71999873db5b286df957af199ec94617f7 with power 10
		// So, if you change ./genesis.json, do change the following as well
		validatorBytes, _ := hex.DecodeString("71562b71999873db5b286df957af199ec94617f7000000000000000000000000000000000000000a")
		copy(header.Extra[32:72], validatorBytes)
	} else {
		header.Extra = make([]byte, 32+65) // vanity + extraSeal
	}
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
