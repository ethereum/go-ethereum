package blockvalidation

import (
	"encoding/json"
	"io/ioutil"
	"math/big"
	"os"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/consensus/ethash"
	"github.com/ethereum/go-ethereum/consensus/misc"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/beacon"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/eth"
	"github.com/ethereum/go-ethereum/eth/downloader"
	"github.com/ethereum/go-ethereum/eth/ethconfig"
	"github.com/ethereum/go-ethereum/eth/tracers/logger"
	"github.com/ethereum/go-ethereum/node"
	"github.com/ethereum/go-ethereum/p2p"
	"github.com/ethereum/go-ethereum/params"
	"github.com/stretchr/testify/require"

	boostTypes "github.com/flashbots/go-boost-utils/types"
)

/* Based on catalyst API tests */

var (
	// testKey is a private key to use for funding a tester account.
	testKey, _ = crypto.HexToECDSA("b71c71a67e1177ad4e901695e1b4b9ee17ae16c6668d313eac2f96dbcda3f291")

	// testAddr is the Ethereum address of the tester account.
	testAddr = crypto.PubkeyToAddress(testKey.PublicKey)

	testBalance = big.NewInt(2e18)
)

func TestValidateBuilderSubmissionV1(t *testing.T) {
	genesis, preMergeBlocks := generatePreMergeChain(20)
	n, ethservice := startEthService(t, genesis, preMergeBlocks)
	ethservice.Merger().ReachTTD()
	defer n.Close()

	api := NewBlockValidationAPI(ethservice, nil)
	parent := preMergeBlocks[len(preMergeBlocks)-1]

	// This EVM code generates a log when the contract is created.
	logCode := common.Hex2Bytes("60606040525b7f24ec1d3ff24c2f6ff210738839dbc339cd45a5294d85c79361016243157aae7b60405180905060405180910390a15b600a8060416000396000f360606040526008565b00")

	statedb, _ := ethservice.BlockChain().StateAt(parent.Root())
	nonce := statedb.GetNonce(testAddr)

	tx1, _ := types.SignTx(types.NewTransaction(nonce, common.Address{0x16}, big.NewInt(10), 21000, big.NewInt(2*params.InitialBaseFee), nil), types.LatestSigner(ethservice.BlockChain().Config()), testKey)
	ethservice.TxPool().AddLocal(tx1)

	cc, _ := types.SignTx(types.NewContractCreation(nonce+1, new(big.Int), 1000000, big.NewInt(2*params.InitialBaseFee), logCode), types.LatestSigner(ethservice.BlockChain().Config()), testKey)
	ethservice.TxPool().AddLocal(cc)

	baseFee := misc.CalcBaseFee(params.AllEthashProtocolChanges, preMergeBlocks[len(preMergeBlocks)-1].Header())
	tx2, _ := types.SignTx(types.NewTransaction(nonce+2, testAddr, big.NewInt(10), 21000, baseFee, nil), types.LatestSigner(ethservice.BlockChain().Config()), testKey)
	ethservice.TxPool().AddLocal(tx2)

	execData, err := assembleBlock(api, parent.Hash(), &beacon.PayloadAttributesV1{
		Timestamp: parent.Time() + 5,
	})
	require.EqualValues(t, len(execData.Transactions), 3)
	require.NoError(t, err)

	payload, err := ExecutableDataToExecutionPayload(execData)
	require.NoError(t, err)

	proposerAddr := boostTypes.Address{}
	proposerAddr.FromSlice(testAddr[:])

	blockRequest := &BuilderBlockValidationRequest{
		BuilderSubmitBlockRequest: boostTypes.BuilderSubmitBlockRequest{
			Signature: boostTypes.Signature{},
			Message: &boostTypes.BidTrace{
				ParentHash:           boostTypes.Hash(execData.ParentHash),
				BlockHash:            boostTypes.Hash(execData.BlockHash),
				ProposerFeeRecipient: proposerAddr,
				GasLimit:             execData.GasLimit,
				GasUsed:              execData.GasUsed,
			},
			ExecutionPayload: payload,
		},
		RegisteredGasLimit: execData.GasLimit,
	}
	require.ErrorContains(t, api.ValidateBuilderSubmissionV1(blockRequest), "inaccurate payment")
	blockRequest.Message.Value = boostTypes.IntToU256(10)
	require.NoError(t, api.ValidateBuilderSubmissionV1(blockRequest))

	blockRequest.Message.GasLimit += 1
	blockRequest.ExecutionPayload.GasLimit += 1
	updatePayloadHash(t, blockRequest)

	require.ErrorContains(t, api.ValidateBuilderSubmissionV1(blockRequest), "incorrect gas limit set")

	blockRequest.Message.GasLimit -= 1
	blockRequest.ExecutionPayload.GasLimit -= 1
	updatePayloadHash(t, blockRequest)

	// TODO: test with contract calling blacklisted address
	// Test tx from blacklisted address
	api.accessVerifier = &AccessVerifier{
		blacklistedAddresses: map[common.Address]struct{}{
			testAddr: struct{}{},
		},
	}
	require.ErrorContains(t, api.ValidateBuilderSubmissionV1(blockRequest), "transaction from blacklisted address 0x71562b71999873DB5b286dF957af199Ec94617F7")

	// Test tx to blacklisted address
	api.accessVerifier = &AccessVerifier{
		blacklistedAddresses: map[common.Address]struct{}{
			common.Address{0x16}: struct{}{},
		},
	}
	require.ErrorContains(t, api.ValidateBuilderSubmissionV1(blockRequest), "transaction to blacklisted address 0x1600000000000000000000000000000000000000")

	api.accessVerifier = nil

	blockRequest.Message.GasUsed = 10
	require.ErrorContains(t, api.ValidateBuilderSubmissionV1(blockRequest), "incorrect GasUsed 10, expected 98990")
	blockRequest.Message.GasUsed = execData.GasUsed

	newTestKey, _ := crypto.HexToECDSA("b71c71a67e1177ad4e901695e1b4b9ee17ae16c6668d313eac2f96dbcda3f290")
	invalidTx, err := types.SignTx(types.NewTransaction(0, common.Address{}, new(big.Int).Mul(big.NewInt(2e18), big.NewInt(10)), 19000, big.NewInt(2*params.InitialBaseFee), nil), types.LatestSigner(ethservice.BlockChain().Config()), newTestKey)
	require.NoError(t, err)

	txData, err := invalidTx.MarshalBinary()
	require.NoError(t, err)
	execData.Transactions = append(execData.Transactions, txData)

	invalidPayload, err := ExecutableDataToExecutionPayload(execData)
	require.NoError(t, err)
	invalidPayload.GasUsed = execData.GasUsed
	invalidPayload.LogsBloom = boostTypes.Bloom{}
	copy(invalidPayload.ReceiptsRoot[:], hexutil.MustDecode("0x56e81f171bcc55a6ff8345e692c0f86e5b48e01b996cadc001622fb5e363b421")[:32])
	blockRequest.ExecutionPayload = invalidPayload
	updatePayloadHash(t, blockRequest)
	require.ErrorContains(t, api.ValidateBuilderSubmissionV1(blockRequest), "could not apply tx 3", "insufficient funds for gas * price + value")
}

func updatePayloadHash(t *testing.T, blockRequest *BuilderBlockValidationRequest) {
	updatedBlock, err := beacon.ExecutionPayloadToBlock(blockRequest.ExecutionPayload)
	require.NoError(t, err)
	copy(blockRequest.Message.BlockHash[:], updatedBlock.Hash().Bytes()[:32])
}

func generatePreMergeChain(n int) (*core.Genesis, []*types.Block) {
	db := rawdb.NewMemoryDatabase()
	config := params.AllEthashProtocolChanges
	genesis := &core.Genesis{
		Config:     config,
		Alloc:      core.GenesisAlloc{testAddr: {Balance: testBalance}},
		ExtraData:  []byte("test genesis"),
		Timestamp:  9000,
		BaseFee:    big.NewInt(params.InitialBaseFee),
		Difficulty: big.NewInt(0),
	}
	testNonce := uint64(0)
	generate := func(i int, g *core.BlockGen) {
		g.OffsetTime(5)
		g.SetExtra([]byte("test"))
		tx, _ := types.SignTx(types.NewTransaction(testNonce, common.HexToAddress("0x9a9070028361F7AAbeB3f2F2Dc07F82C4a98A02a"), big.NewInt(1), params.TxGas, big.NewInt(params.InitialBaseFee*2), nil), types.LatestSigner(config), testKey)
		g.AddTx(tx)
		testNonce++
	}
	gblock := genesis.MustCommit(db)
	engine := ethash.NewFaker()
	blocks, _ := core.GenerateChain(config, gblock, engine, db, n, generate)
	totalDifficulty := big.NewInt(0)
	for _, b := range blocks {
		totalDifficulty.Add(totalDifficulty, b.Difficulty())
	}
	config.TerminalTotalDifficulty = totalDifficulty
	return genesis, blocks
}

// startEthService creates a full node instance for testing.
func startEthService(t *testing.T, genesis *core.Genesis, blocks []*types.Block) (*node.Node, *eth.Ethereum) {
	t.Helper()

	n, err := node.New(&node.Config{
		P2P: p2p.Config{
			ListenAddr:  "0.0.0.0:0",
			NoDiscovery: true,
			MaxPeers:    25,
		}})
	if err != nil {
		t.Fatal("can't create node:", err)
	}

	ethcfg := &ethconfig.Config{Genesis: genesis, Ethash: ethash.Config{PowMode: ethash.ModeFake}, SyncMode: downloader.SnapSync, TrieTimeout: time.Minute, TrieDirtyCache: 256, TrieCleanCache: 256}
	ethservice, err := eth.New(n, ethcfg)
	if err != nil {
		t.Fatal("can't create eth service:", err)
	}
	if err := n.Start(); err != nil {
		t.Fatal("can't start node:", err)
	}
	if _, err := ethservice.BlockChain().InsertChain(blocks); err != nil {
		n.Close()
		t.Fatal("can't import test blocks:", err)
	}
	time.Sleep(500 * time.Millisecond) // give txpool enough time to consume head event

	ethservice.SetEtherbase(testAddr)
	ethservice.SetSynced()
	return n, ethservice
}

func assembleBlock(api *BlockValidationAPI, parentHash common.Hash, params *beacon.PayloadAttributesV1) (*beacon.ExecutableDataV1, error) {
	block, err := api.eth.Miner().GetSealingBlockSync(parentHash, params.Timestamp, params.SuggestedFeeRecipient, params.Random, false)
	if err != nil {
		return nil, err
	}
	return beacon.BlockToExecutableData(block), nil
}

func TestBlacklistLoad(t *testing.T) {
	file, err := os.CreateTemp(".", "blacklist")
	require.NoError(t, err)
	defer os.Remove(file.Name())

	av, err := NewAccessVerifierFromFile(file.Name())
	require.Error(t, err)
	require.Nil(t, av)

	ba := BlacklistedAddresses{common.Address{0x13}, common.Address{0x14}}
	bytes, err := json.MarshalIndent(ba, "", " ")
	require.NoError(t, err)
	err = ioutil.WriteFile(file.Name(), bytes, 0644)
	require.NoError(t, err)

	av, err = NewAccessVerifierFromFile(file.Name())
	require.NoError(t, err)
	require.NotNil(t, av)
	require.EqualValues(t, av.blacklistedAddresses, map[common.Address]struct{}{
		common.Address{0x13}: struct{}{},
		common.Address{0x14}: struct{}{},
	})

	require.NoError(t, av.verifyTraces(logger.NewAccessListTracer(nil, common.Address{}, common.Address{}, nil)))

	acl := types.AccessList{
		types.AccessTuple{
			Address: common.Address{0x14},
		},
	}
	tracer := logger.NewAccessListTracer(acl, common.Address{}, common.Address{}, nil)
	require.ErrorContains(t, av.verifyTraces(tracer), "blacklisted address 0x1400000000000000000000000000000000000000 in execution trace")

	acl = types.AccessList{
		types.AccessTuple{
			Address: common.Address{0x15},
		},
	}
	tracer = logger.NewAccessListTracer(acl, common.Address{}, common.Address{}, nil)
	require.NoError(t, av.verifyTraces(tracer))
}

func ExecutableDataToExecutionPayload(data *beacon.ExecutableDataV1) (*boostTypes.ExecutionPayload, error) {
	transactionData := make([]hexutil.Bytes, len(data.Transactions))
	for i, tx := range data.Transactions {
		transactionData[i] = hexutil.Bytes(tx)
	}

	baseFeePerGas := new(boostTypes.U256Str)
	err := baseFeePerGas.FromBig(data.BaseFeePerGas)
	if err != nil {
		return nil, err
	}

	return &boostTypes.ExecutionPayload{
		ParentHash:    [32]byte(data.ParentHash),
		FeeRecipient:  [20]byte(data.FeeRecipient),
		StateRoot:     [32]byte(data.StateRoot),
		ReceiptsRoot:  [32]byte(data.ReceiptsRoot),
		LogsBloom:     boostTypes.Bloom(types.BytesToBloom(data.LogsBloom)),
		Random:        [32]byte(data.Random),
		BlockNumber:   data.Number,
		GasLimit:      data.GasLimit,
		GasUsed:       data.GasUsed,
		Timestamp:     data.Timestamp,
		ExtraData:     data.ExtraData,
		BaseFeePerGas: *baseFeePerGas,
		BlockHash:     [32]byte(data.BlockHash),
		Transactions:  transactionData,
	}, nil
}
