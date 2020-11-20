package bor

import (
	"encoding/hex"
	"encoding/json"
	"io/ioutil"
	"math/big"
	"sort"
	"testing"

	"github.com/maticnetwork/bor/common"
	"github.com/maticnetwork/bor/consensus/bor"
	"github.com/maticnetwork/bor/core"
	"github.com/maticnetwork/bor/core/types"
	"github.com/maticnetwork/bor/crypto"
	"github.com/maticnetwork/bor/crypto/secp256k1"
	"github.com/maticnetwork/bor/eth"
	"github.com/maticnetwork/bor/ethdb"
	"github.com/maticnetwork/bor/node"
	"github.com/maticnetwork/bor/params"
)

var (
	// The genesis for tests was generated with following parameters
	extraSeal = 65 // Fixed number of extra-data suffix bytes reserved for signer seal

	// Only this account is a validator for the 0th span
	privKey = "b71c71a67e1177ad4e901695e1b4b9ee17ae16c6668d313eac2f96dbcda3f291"
	key, _  = crypto.HexToECDSA(privKey)
	addr    = crypto.PubkeyToAddress(key.PublicKey) // 0x71562b71999873DB5b286dF957af199Ec94617F7

	// This account is one the validators for 1st span (0-indexed)
	privKey2 = "9b28f36fbd67381120752d6172ecdcf10e06ab2d9a1367aac00cdcd6ac7855d3"
	key2, _  = crypto.HexToECDSA(privKey2)
	addr2    = crypto.PubkeyToAddress(key2.PublicKey) // 0x9fB29AAc15b9A4B7F17c3385939b007540f4d791

	validatorHeaderBytesLength        = common.AddressLength + 20 // address + power
	sprintSize                 uint64 = 4
	spanSize                   uint64 = 8
)

type initializeData struct {
	genesis  *core.Genesis
	ethereum *eth.Ethereum
}

func buildEthereumInstance(t *testing.T, db ethdb.Database) *initializeData {
	genesisData, err := ioutil.ReadFile("./testdata/genesis.json")
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

func insertNewBlock(t *testing.T, chain *core.BlockChain, block *types.Block) {
	if _, err := chain.InsertChain([]*types.Block{block}); err != nil {
		t.Fatalf("%s", err)
	}
}

func buildNextBlock(t *testing.T, _bor *bor.Bor, chain *core.BlockChain, block *types.Block, signer []byte, borConfig *params.BorConfig) *types.Block {
	header := block.Header()
	header.Number.Add(header.Number, big.NewInt(1))
	number := header.Number.Uint64()

	if signer == nil {
		signer = getSignerKey(header.Number.Uint64())
	}

	header.ParentHash = block.Hash()
	header.Time += bor.CalcProducerDelay(header.Number.Uint64(), 0, borConfig)
	header.Extra = make([]byte, 32+65) // vanity + extraSeal

	currentValidators := []*bor.Validator{bor.NewValidator(addr, 10)}

	isSpanEnd := (number+1)%spanSize == 0
	isSpanStart := number%spanSize == 0
	isSprintEnd := (header.Number.Uint64()+1)%sprintSize == 0
	if isSpanEnd {
		_, heimdallSpan := loadSpanFromFile(t)
		// this is to stash the validator bytes in the header
		currentValidators = heimdallSpan.ValidatorSet.Validators
	} else if isSpanStart {
		header.Difficulty = new(big.Int).SetInt64(3)
	}
	if isSprintEnd {
		sort.Sort(bor.ValidatorsByAddress(currentValidators))
		validatorBytes := make([]byte, len(currentValidators)*validatorHeaderBytesLength)
		header.Extra = make([]byte, 32+len(validatorBytes)+65) // vanity + validatorBytes + extraSeal
		for i, val := range currentValidators {
			copy(validatorBytes[i*validatorHeaderBytesLength:], val.HeaderBytes())
		}
		copy(header.Extra[32:], validatorBytes)
	}

	state, err := chain.State()
	if err != nil {
		t.Fatalf("%s", err)
	}
	_, err = _bor.FinalizeAndAssemble(chain, header, state, nil, nil, nil)
	if err != nil {
		t.Fatalf("%s", err)
	}
	sign(t, header, signer)
	return types.NewBlockWithHeader(header)
}

func sign(t *testing.T, header *types.Header, signer []byte) {
	sig, err := secp256k1.Sign(crypto.Keccak256(bor.BorRLP(header)), signer)
	if err != nil {
		t.Fatalf("%s", err)
	}
	copy(header.Extra[len(header.Extra)-extraSeal:], sig)
}

func stateSyncEventsPayload(t *testing.T) *bor.ResponseWithHeight {
	stateData, err := ioutil.ReadFile("./testdata/states.json")
	if err != nil {
		t.Fatalf("%s", err)
	}
	res := &bor.ResponseWithHeight{}
	if err := json.Unmarshal(stateData, res); err != nil {
		t.Fatalf("%s", err)
	}
	return res
}

func loadSpanFromFile(t *testing.T) (*bor.ResponseWithHeight, *bor.HeimdallSpan) {
	spanData, err := ioutil.ReadFile("./testdata/span.json")
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

func getSignerKey(number uint64) []byte {
	signerKey := privKey
	isSpanStart := number%spanSize == 0
	if isSpanStart {
		// validator set in the new span has changed
		signerKey = privKey2
	}
	_key, _ := hex.DecodeString(signerKey)
	return _key
}
