//go:build integration

package bor

import (
	"bytes"
	"crypto/ecdsa"
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
	"github.com/ethereum/go-ethereum/accounts/keystore"
	"github.com/ethereum/go-ethereum/cmd/utils"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/consensus"
	"github.com/ethereum/go-ethereum/consensus/bor"
	"github.com/ethereum/go-ethereum/consensus/bor/clerk"
	"github.com/ethereum/go-ethereum/consensus/bor/heimdall" //nolint:typecheck
	"github.com/ethereum/go-ethereum/consensus/bor/heimdall/checkpoint"
	"github.com/ethereum/go-ethereum/consensus/bor/heimdall/milestone"
	"github.com/ethereum/go-ethereum/consensus/bor/heimdall/span"
	"github.com/ethereum/go-ethereum/consensus/bor/valset"
	"github.com/ethereum/go-ethereum/consensus/misc/eip1559"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/txpool/legacypool"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/crypto/secp256k1"
	"github.com/ethereum/go-ethereum/eth"
	"github.com/ethereum/go-ethereum/eth/downloader"
	"github.com/ethereum/go-ethereum/eth/ethconfig"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/miner"
	"github.com/ethereum/go-ethereum/node"
	"github.com/ethereum/go-ethereum/p2p"
	"github.com/ethereum/go-ethereum/p2p/enode"
	"github.com/ethereum/go-ethereum/params"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/ethereum/go-ethereum/tests/bor/mocks"
	"github.com/ethereum/go-ethereum/trie"
	"github.com/ethereum/go-ethereum/triedb"
)

var (

	// Only this account is a validator for the 0th span
	key, _ = crypto.HexToECDSA(privKey)
	addr   = crypto.PubkeyToAddress(key.PublicKey) // 0x71562b71999873DB5b286dF957af199Ec94617F7

	// This account is one the validators for 1st span (0-indexed)
	key2, _ = crypto.HexToECDSA(privKey2)
	addr2   = crypto.PubkeyToAddress(key2.PublicKey) // 0x9fB29AAc15b9A4B7F17c3385939b007540f4d791

	// This account is secondary validator for 1st span (0-indexed)
	key3, _ = crypto.HexToECDSA(privKey3)
	addr3   = crypto.PubkeyToAddress(key3.PublicKey) // 0x96C42C56fdb78294F96B0cFa33c92bed7D75F96a

	keys = []*ecdsa.PrivateKey{key, key2}
)

const (
	privKey  = "b71c71a67e1177ad4e901695e1b4b9ee17ae16c6668d313eac2f96dbcda3f291"
	privKey2 = "9b28f36fbd67381120752d6172ecdcf10e06ab2d9a1367aac00cdcd6ac7855d3"
	privKey3 = "c8deb0bea5c41afe8e37b4d1bd84e31adff11b09c8c96ff4b605003cce067cd9"

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

func setupMiner(t *testing.T, n int, genesis *core.Genesis) ([]*node.Node, []*eth.Ethereum, []*enode.Node) {
	t.Helper()

	// Create an Ethash network based off of the Ropsten config
	var (
		stacks []*node.Node
		nodes  []*eth.Ethereum
		enodes []*enode.Node
	)

	for i := 0; i < n; i++ {
		// Start the node and wait until it's up
		stack, ethBackend, err := InitMiner(genesis, keys[i], true)
		if err != nil {
			t.Fatal("Error occurred while initialising miner", "error", err)
		}

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

	return stacks, nodes, enodes
}

func buildEthereumInstance(t *testing.T, db ethdb.Database, updateGenesis ...func(gen *core.Genesis)) *initializeData {
	genesisData, err := ioutil.ReadFile("./testdata/genesis.json")
	if err != nil {
		t.Fatalf("%s", err)
	}

	gen := &core.Genesis{}
	if err := json.Unmarshal(genesisData, gen); err != nil {
		t.Fatalf("%s", err)
	}
	for _, update := range updateGenesis {
		update(gen)
	}

	ethConf := &eth.Config{
		Genesis:     gen,
		BorLogs:     true,
		StateScheme: "hash",
	}
	ethConf.Genesis.MustCommit(db, triedb.NewDatabase(db, triedb.HashDefaults))

	ethereum := utils.CreateBorEthereum(ethConf)
	if err != nil {
		t.Fatalf("failed to register Ethereum protocol: %v", err)
	}

	ethConf.Genesis.MustCommit(ethereum.ChainDb(), triedb.NewDatabase(ethereum.ChainDb(), triedb.HashDefaults))

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

func buildHeader(t *testing.T, chain *core.BlockChain, parentBlock *types.Block, signer []byte, borConfig *params.BorConfig, currentValidators []*valset.Validator, opts ...Option) *types.Header {
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

	// Similar to the logic in bor consensus
	header.Time = parentBlock.Time() + bor.CalcProducerDelay(header.Number.Uint64(), 0, borConfig)
	// Keeping this causes some e2e tests to fail because they work under certain time assumptions
	// if header.Time < uint64(time.Now().Unix()) {
	// 	header.Time = uint64(time.Now().Unix())
	// }

	// Similar to logic in bor consensus (prepare)
	header.Extra = make([]byte, 32+65) // vanity + extraSeal
	if len(header.Extra) < types.ExtraVanityLength {
		header.Extra = append(header.Extra, bytes.Repeat([]byte{0x00}, types.ExtraVanityLength-len(header.Extra))...)
	}
	header.Extra = header.Extra[:types.ExtraVanityLength]

	var isSprintEnd bool
	if (number+1)%chain.Config().Bor.CalculateSprint(number) == 0 {
		isSprintEnd = true
	}
	isSpanStart := IsSpanStart(number)

	if isSpanStart {
		header.Difficulty = new(big.Int).SetInt64(int64(len(currentValidators)))
	}

	if isSprintEnd {
		sort.Sort(valset.ValidatorsByAddress(currentValidators))

		// Extra data is encoded differently after cancun
		if chain.Config().IsCancun(header.Number) {
			var tempValidatorBytes []byte
			for _, validator := range currentValidators {
				tempValidatorBytes = append(tempValidatorBytes, validator.HeaderBytes()...)
			}

			blockExtraData := &types.BlockExtraData{
				ValidatorBytes: tempValidatorBytes,
				TxDependency:   nil,
			}
			blockExtraDataBytes, err := rlp.EncodeToBytes(blockExtraData)
			if err != nil {
				t.Fatalf("error while encoding block extra data: %v", err)
			}
			header.Extra = append(header.Extra, blockExtraDataBytes...)
		} else {
			validatorBytes := make([]byte, len(currentValidators)*validatorHeaderBytesLength)
			header.Extra = make([]byte, 32+len(validatorBytes)+65) // vanity + validatorBytes + extraSeal

			for i, val := range currentValidators {
				copy(validatorBytes[i*validatorHeaderBytesLength:], val.HeaderBytes())
			}

			copy(header.Extra[32:], validatorBytes)
		}
	} else if chain.Config().IsCancun(header.Number) {
		blockExtraData := &types.BlockExtraData{
			ValidatorBytes: nil,
			TxDependency:   nil,
		}

		blockExtraDataBytes, err := rlp.EncodeToBytes(blockExtraData)
		if err != nil {
			t.Fatalf("error while encoding block extra data: %v", err)
		}

		header.Extra = append(header.Extra, blockExtraDataBytes...)
	}

	header.Extra = append(header.Extra, make([]byte, types.ExtraSealLength)...)

	if chain.Config().IsLondon(header.Number) {
		header.BaseFee = eip1559.CalcBaseFee(chain.Config(), parentBlock.Header())

		if !chain.Config().IsLondon(parentBlock.Number()) {
			parentGasLimit := parentBlock.GasLimit() * params.ElasticityMultiplier
			header.GasLimit = core.CalcGasLimit(parentGasLimit, parentGasLimit)
		}
	}

	for _, opt := range opts {
		opt(header)
	}

	return header
}

func buildNextBlock(t *testing.T, _bor consensus.Engine, chain *core.BlockChain, parentBlock *types.Block, signer []byte, borConfig *params.BorConfig, txs []*types.Transaction, currentValidators []*valset.Validator, skipSealing bool, opts ...Option) *types.Block {
	t.Helper()

	// Build a new header based on parent block
	header := buildHeader(t, chain, parentBlock, signer, borConfig, currentValidators, opts...)

	state, err := chain.State()
	if err != nil {
		t.Fatalf("%s", err)
	}

	b := &blockGen{header: header}
	for _, tx := range txs {
		b.addTxWithChain(chain, state, tx, addr)
	}

	// Finalize and seal the block
	block, err := _bor.FinalizeAndAssemble(chain, b.header, state, &types.Body{
		Transactions: b.txs,
	}, b.receipts)
	if err != nil {
		panic(fmt.Sprintf("error finalizing block: %v", err))
	}

	// Write state changes to db
	root, err := state.Commit(block.NumberU64(), chain.Config().IsEIP158(b.header.Number), false)
	if err != nil {
		panic(fmt.Sprintf("state write error: %v", err))
	}

	if err := state.Database().TrieDB().Commit(root, false); err != nil {
		panic(fmt.Sprintf("trie write error: %v", err))
	}

	res := make(chan *types.Block, 1)

	if skipSealing {
		header := block.Header()
		sign(t, header, signer, borConfig)
		return types.NewBlock(header, block.Body(), b.receipts, trie.NewStackTrie(nil))
	}

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

	statedb.SetTxContext(tx.Hash(), len(b.txs))

	context := core.NewEVMBlockContext(b.header, bc, nil)
	evm := vm.NewEVM(context, statedb, bc.Config(), vm.Config{})
	receipt, err := core.ApplyTransaction(evm, b.gasPool, statedb, b.header, tx, &b.header.GasUsed, nil)
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

func createMockSpan(address common.Address, chainId string) span.HeimdallSpan {
	// Mock span 0 for heimdall calls
	validator := valset.Validator{
		ID:               0,
		Address:          address,
		VotingPower:      10,
		ProposerPriority: 0,
	}
	validatorSet := valset.ValidatorSet{
		Validators: []*valset.Validator{&validator},
		Proposer:   &validator,
	}
	span0 := span.HeimdallSpan{
		Span: span.Span{
			ID:         0,
			StartBlock: 0,
			EndBlock:   255,
		},
		ValidatorSet:      validatorSet,
		SelectedProducers: []valset.Validator{validator},
		ChainID:           chainId,
	}

	return span0
}

func createMockHeimdall(ctrl *gomock.Controller, span0, span1 *span.HeimdallSpan) *mocks.MockIHeimdallClient {
	h := mocks.NewMockIHeimdallClient(ctrl)

	h.EXPECT().Close().AnyTimes()
	h.EXPECT().Span(gomock.Any(), uint64(0)).Return(span0, nil).AnyTimes()
	h.EXPECT().Span(gomock.Any(), uint64(1)).Return(span1, nil).AnyTimes()
	h.EXPECT().FetchCheckpoint(gomock.Any(), int64(-1)).Return(&checkpoint.Checkpoint{}, nil).AnyTimes()
	h.EXPECT().FetchMilestone(gomock.Any()).Return(&milestone.Milestone{}, nil).AnyTimes()
	h.EXPECT().FetchLastNoAckMilestone(gomock.Any()).Return("", nil).AnyTimes()
	h.EXPECT().FetchNoAckMilestone(gomock.Any(), string("test")).Return(nil).AnyTimes()

	return h
}

func getMockedSpanner(t *testing.T, validators []*valset.Validator) *bor.MockSpanner {
	t.Helper()

	spanner := bor.NewMockSpanner(gomock.NewController(t))
	spanner.EXPECT().GetCurrentValidatorsByHash(gomock.Any(), gomock.Any(), gomock.Any()).Return(validators, nil).AnyTimes()
	spanner.EXPECT().GetCurrentValidatorsByBlockNrOrHash(gomock.Any(), gomock.Any(), gomock.Any()).Return(validators, nil).AnyTimes()
	spanner.EXPECT().GetCurrentSpan(gomock.Any(), gomock.Any()).Return(&span.Span{0, 0, 0}, nil).AnyTimes()
	spanner.EXPECT().CommitSpan(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(nil).AnyTimes()
	return spanner
}

func generateFakeStateSyncEvents(sample *clerk.EventRecordWithTime, count int) []*clerk.EventRecordWithTime {
	events := make([]*clerk.EventRecordWithTime, count)
	event := *sample
	event.ID = 1
	events[0] = &clerk.EventRecordWithTime{}
	*events[0] = event

	for i := 1; i < count; i++ {
		event.ID = uint64(i + 1)
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

func InitGenesis(t *testing.T, faucets []*ecdsa.PrivateKey, fileLocation string, sprintSize uint64) *core.Genesis {
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
	genesis.Config.EIP150Block = big.NewInt(0)

	genesis.Config.Bor.Sprint["0"] = sprintSize

	return genesis
}

func InitMiner(genesis *core.Genesis, privKey *ecdsa.PrivateKey, withoutHeimdall bool) (*node.Node, *eth.Ethereum, error) {
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
		TxPool:          legacypool.DefaultConfig,
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
