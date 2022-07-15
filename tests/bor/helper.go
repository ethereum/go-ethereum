//go:build integration

package bor

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"math/big"
	"sort"
	"testing"
	"time"

	"github.com/golang/mock/gomock"

	"github.com/ethereum/go-ethereum/accounts"
	"github.com/ethereum/go-ethereum/cmd/utils"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/consensus"
	"github.com/ethereum/go-ethereum/consensus/bor"
	"github.com/ethereum/go-ethereum/consensus/bor/clerk"
	"github.com/ethereum/go-ethereum/consensus/bor/heimdall" //nolint:typecheck
	"github.com/ethereum/go-ethereum/consensus/bor/heimdall/span"
	"github.com/ethereum/go-ethereum/consensus/bor/valset"
	"github.com/ethereum/go-ethereum/consensus/misc"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/crypto/secp256k1"
	"github.com/ethereum/go-ethereum/eth"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/params"
	"github.com/ethereum/go-ethereum/tests/bor/mocks"
)

var (

	// Only this account is a validator for the 0th span
	key, _ = crypto.HexToECDSA(privKey)
	addr   = crypto.PubkeyToAddress(key.PublicKey) // 0x71562b71999873DB5b286dF957af199Ec94617F7

	// This account is one the validators for 1st span (0-indexed)
	key2, _ = crypto.HexToECDSA(privKey2)
	addr2   = crypto.PubkeyToAddress(key2.PublicKey) // 0x9fB29AAc15b9A4B7F17c3385939b007540f4d791
)

const (
	privKey  = "b71c71a67e1177ad4e901695e1b4b9ee17ae16c6668d313eac2f96dbcda3f291"
	privKey2 = "9b28f36fbd67381120752d6172ecdcf10e06ab2d9a1367aac00cdcd6ac7855d3"

	// The genesis for tests was generated with following parameters
	extraSeal = 65 // Fixed number of extra-data suffix bytes reserved for signer seal

	sprintSize uint64 = 4
	spanSize   uint64 = 8

	validatorHeaderBytesLength = common.AddressLength + 20 // address + power
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
		BorLogs: true,
	}

	ethConf.Genesis.MustCommit(db)

	ethereum := utils.CreateBorEthereum(ethConf)
	if err != nil {
		t.Fatalf("failed to register Ethereum protocol: %v", err)
	}

	ethConf.Genesis.MustCommit(ethereum.ChainDb())

	ethereum.Engine().(*bor.Bor).Authorize(addr, func(account accounts.Account, s string, data []byte) ([]byte, error) {
		return crypto.Sign(crypto.Keccak256(data), key)
	})

	return &initializeData{
		genesis:  gen,
		ethereum: ethereum,
	}
}

func insertNewBlock(t *testing.T, chain *core.BlockChain, block *types.Block) {
	t.Helper()

	if _, err := chain.InsertChain([]*types.Block{block}); err != nil {
		t.Fatalf("%s", err)
	}
}

type Option func(header *types.Header)

func buildNextBlock(t *testing.T, _bor consensus.Engine, chain *core.BlockChain, parentBlock *types.Block, signer []byte, borConfig *params.BorConfig, txs []*types.Transaction, currentValidators []*valset.Validator, opts ...Option) *types.Block {
	t.Helper()

	header := &types.Header{
		Number:     big.NewInt(int64(parentBlock.Number().Uint64() + 1)),
		Difficulty: big.NewInt(int64(parentBlock.Difficulty().Uint64())),
		GasLimit:   parentBlock.GasLimit(),
		ParentHash: parentBlock.Hash(),
	}
	number := header.Number.Uint64()

	if signer == nil {
		signer = getSignerKey(header.Number.Uint64())
	}

	header.Time = parentBlock.Time() + bor.CalcProducerDelay(header.Number.Uint64(), 0, borConfig)
	header.Extra = make([]byte, 32+65) // vanity + extraSeal

	isSpanStart := IsSpanStart(number)
	isSprintEnd := IsSprintEnd(number)

	if isSpanStart {
		header.Difficulty = new(big.Int).SetInt64(int64(len(currentValidators)))
	}

	if isSprintEnd {
		sort.Sort(valset.ValidatorsByAddress(currentValidators))

		validatorBytes := make([]byte, len(currentValidators)*validatorHeaderBytesLength)
		header.Extra = make([]byte, 32+len(validatorBytes)+65) // vanity + validatorBytes + extraSeal

		for i, val := range currentValidators {
			copy(validatorBytes[i*validatorHeaderBytesLength:], val.HeaderBytes())
		}

		copy(header.Extra[32:], validatorBytes)
	}

	if chain.Config().IsLondon(header.Number) {
		header.BaseFee = misc.CalcBaseFee(chain.Config(), parentBlock.Header())

		if !chain.Config().IsLondon(parentBlock.Number()) {
			parentGasLimit := parentBlock.GasLimit() * params.ElasticityMultiplier
			header.GasLimit = core.CalcGasLimit(parentGasLimit, parentGasLimit)
		}
	}

	for _, opt := range opts {
		opt(header)
	}

	state, err := chain.State()
	if err != nil {
		t.Fatalf("%s", err)
	}

	b := &blockGen{header: header}
	for _, tx := range txs {
		b.addTxWithChain(chain, state, tx, addr)
	}

	// Finalize and seal the block
	block, _ := _bor.FinalizeAndAssemble(chain, b.header, state, b.txs, nil, b.receipts)

	// Write state changes to db
	root, err := state.Commit(chain.Config().IsEIP158(b.header.Number))
	if err != nil {
		panic(fmt.Sprintf("state write error: %v", err))
	}

	if err := state.Database().TrieDB().Commit(root, false, nil); err != nil {
		panic(fmt.Sprintf("trie write error: %v", err))
	}

	res := make(chan *types.Block, 1)

	err = _bor.Seal(chain, block, res, nil)
	if err != nil {
		// an error case - sign manually
		sign(t, header, signer, borConfig)
		return types.NewBlockWithHeader(header)
	}

	return <-res
}

type blockGen struct {
	txs      []*types.Transaction
	receipts []*types.Receipt
	gasPool  *core.GasPool
	header   *types.Header
}

func (b *blockGen) addTxWithChain(bc *core.BlockChain, statedb *state.StateDB, tx *types.Transaction, coinbase common.Address) {
	if b.gasPool == nil {
		b.setCoinbase(coinbase)
	}

	statedb.Prepare(tx.Hash(), len(b.txs))

	receipt, err := core.ApplyTransaction(bc.Config(), bc, &b.header.Coinbase, b.gasPool, statedb, b.header, tx, &b.header.GasUsed, vm.Config{})
	if err != nil {
		panic(err)
	}

	b.txs = append(b.txs, tx)
	b.receipts = append(b.receipts, receipt)
}

func (b *blockGen) setCoinbase(addr common.Address) {
	if b.gasPool != nil {
		if len(b.txs) > 0 {
			panic("coinbase must be set before adding transactions")
		}

		panic("coinbase can only be set once")
	}

	b.header.Coinbase = addr
	b.gasPool = new(core.GasPool).AddGas(b.header.GasLimit)
}

func sign(t *testing.T, header *types.Header, signer []byte, c *params.BorConfig) {
	t.Helper()

	sig, err := secp256k1.Sign(crypto.Keccak256(bor.BorRLP(header, c)), signer)
	if err != nil {
		t.Fatalf("%s", err)
	}

	copy(header.Extra[len(header.Extra)-extraSeal:], sig)
}

//nolint:unused,deadcode
func stateSyncEventsPayload(t *testing.T) *heimdall.StateSyncEventsResponse {
	t.Helper()

	stateData, err := ioutil.ReadFile("./testdata/states.json")
	if err != nil {
		t.Fatalf("%s", err)
	}

	res := &heimdall.StateSyncEventsResponse{}
	if err := json.Unmarshal(stateData, res); err != nil {
		t.Fatalf("%s", err)
	}

	return res
}

//nolint:unused,deadcode
func loadSpanFromFile(t *testing.T) (*heimdall.SpanResponse, *span.HeimdallSpan) {
	t.Helper()

	spanData, err := ioutil.ReadFile("./testdata/span.json")
	if err != nil {
		t.Fatalf("%s", err)
	}

	res := &heimdall.SpanResponse{}

	if err := json.Unmarshal(spanData, res); err != nil {
		t.Fatalf("%s", err)
	}

	return res, &res.Result
}

func getSignerKey(number uint64) []byte {
	signerKey := privKey

	if IsSpanStart(number) {
		// validator set in the new span has changed
		signerKey = privKey2
	}

	newKey, _ := hex.DecodeString(signerKey)

	return newKey
}

func getMockedHeimdallClient(t *testing.T, heimdallSpan *span.HeimdallSpan) (*mocks.MockIHeimdallClient, *gomock.Controller) {
	t.Helper()

	ctrl := gomock.NewController(t)
	h := mocks.NewMockIHeimdallClient(ctrl)

	h.EXPECT().Span(gomock.Any(), uint64(1)).Return(heimdallSpan, nil).AnyTimes()

	h.EXPECT().StateSyncEvents(gomock.Any(), gomock.Any(), gomock.Any()).
		Return([]*clerk.EventRecordWithTime{getSampleEventRecord(t)}, nil).AnyTimes()

	return h, ctrl
}

func generateFakeStateSyncEvents(sample *clerk.EventRecordWithTime, count int) []*clerk.EventRecordWithTime {
	events := make([]*clerk.EventRecordWithTime, count)
	event := *sample
	event.ID = 1
	events[0] = &clerk.EventRecordWithTime{}
	*events[0] = event

	for i := 1; i < count; i++ {
		event.ID = uint64(i)
		event.Time = event.Time.Add(1 * time.Second)
		events[i] = &clerk.EventRecordWithTime{}
		*events[i] = event
	}

	return events
}

func buildStateEvent(sample *clerk.EventRecordWithTime, id uint64, timeStamp int64) *clerk.EventRecordWithTime {
	event := *sample
	event.ID = id
	event.Time = time.Unix(timeStamp, 0)

	return &event
}

func getSampleEventRecord(t *testing.T) *clerk.EventRecordWithTime {
	t.Helper()

	eventRecords := stateSyncEventsPayload(t)
	eventRecords.Result[0].Time = time.Unix(1, 0)

	return eventRecords.Result[0]
}

func newGwei(n int64) *big.Int {
	return new(big.Int).Mul(big.NewInt(n), big.NewInt(params.GWei))
}

func IsSpanEnd(number uint64) bool {
	return (number+1)%spanSize == 0
}

func IsSpanStart(number uint64) bool {
	return number%spanSize == 0
}

func IsSprintStart(number uint64) bool {
	return number%sprintSize == 0
}

func IsSprintEnd(number uint64) bool {
	return (number+1)%sprintSize == 0
}
