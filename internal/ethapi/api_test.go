package ethapi

import (
	"context"
	"crypto/ecdsa"
	"encoding/json"
	"fmt"
	"math/big"
	"math/rand"
	"testing"

	"github.com/aws/aws-sdk-go/awstesting"
	"github.com/ethereum/go-ethereum/accounts"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/bloombits"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/eth/downloader"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/event"
	"github.com/ethereum/go-ethereum/params"
	"github.com/ethereum/go-ethereum/rpc"
)

var (
	internalTxNonce    = hexutil.Uint64(uint64(rand.Int()))
	internalTxCalldata = hexutil.Bytes{0, 1, 2, 3, 4, 5, 6, 7}
	internalTxSender   = common.Address{0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 0, 1, 2, 3, 4, 5, 6, 7, 8, 9}
	internalTxTarget   = common.Address{9, 8, 7, 6, 5, 4, 3, 2, 1, 0, 9, 8, 7, 6, 5, 4, 3, 2, 1, 0}
	backendTimestamp   = int64(0)
)

type testCase struct {
	backendContext     backendContext
	inputCtx           context.Context
	inputMessageAndSig []hexutil.Bytes
	hasErrors          bool
	resultingTimestamp int64
	multipleBatches    bool
}

func getTestCases(pk *ecdsa.PrivateKey) []testCase {
	return []testCase{
		// Bad input -- message and sig not of length 2
		{inputCtx: getFakeContext(), inputMessageAndSig: []hexutil.Bytes{}, hasErrors: true},
		{inputCtx: getFakeContext(), inputMessageAndSig: []hexutil.Bytes{[]byte{1, 2, 3}}, hasErrors: true},
		{inputCtx: getFakeContext(), inputMessageAndSig: []hexutil.Bytes{[]byte{1}, []byte{2}, []byte{3}}, hasErrors: true},

		// Bad input -- message not signed
		{inputCtx: getFakeContext(), inputMessageAndSig: []hexutil.Bytes{[]byte{1}, []byte{2}}, hasErrors: true},

		// Bad input -- message is signed but incorrect format
		{inputCtx: getFakeContext(), inputMessageAndSig: getInputMessageAndSignature([]byte{1}, pk), hasErrors: true},

		// Returns 0 errors if no transactions but timestamp updated
		{inputCtx: getFakeContext(), inputMessageAndSig: getBlockBatchesInputMessageAndSignature(pk, 0, 1, []int{})},
		{inputCtx: getFakeContext(), inputMessageAndSig: getBlockBatchesInputMessageAndSignature(pk, 1, 1, []int{}), resultingTimestamp: 1},

		// Handles one transaction and updates timestamp
		{inputCtx: getFakeContext(), inputMessageAndSig: getBlockBatchesInputMessageAndSignature(pk, 1, 1, []int{1}), resultingTimestamp: 1},
		{backendContext: backendContext{sendTxsErrors: getDummyErrors([]int{0}, 1)}, inputCtx: getFakeContext(), inputMessageAndSig: getBlockBatchesInputMessageAndSignature(pk, 1, 1, []int{1}), hasErrors: true, resultingTimestamp: 1},

		// Handles one batch of multiple transaction and updates timestamp
		{inputCtx: getFakeContext(), inputMessageAndSig: getBlockBatchesInputMessageAndSignature(pk, 1, 1, []int{2}), resultingTimestamp: 1},
		{backendContext: backendContext{sendTxsErrors: getDummyErrors([]int{1}, 2)}, inputCtx: getFakeContext(), inputMessageAndSig: getBlockBatchesInputMessageAndSignature(pk, 1, 2, []int{2}), hasErrors: true, resultingTimestamp: 1},

		// Handles multiple transactions and updates timestamp
		{inputCtx: getFakeContext(), inputMessageAndSig: getBlockBatchesInputMessageAndSignature(pk, 2, 1, []int{1, 2, 3}), resultingTimestamp: 2},
		{backendContext: backendContext{sendTxsErrors: getDummyErrors([]int{0, 2}, 3)}, inputCtx: getFakeContext(), inputMessageAndSig: getBlockBatchesInputMessageAndSignature(pk, 1, 1, []int{1, 2, 3}), hasErrors: true, resultingTimestamp: 1, multipleBatches: true},
	}
}

func TestSendBlockBatches(t *testing.T) {
	blockBatchSenderPrivKey, _ := crypto.GenerateKey()
	txSignerPrivKey, _ := crypto.GenerateKey()

	for testNum, testCase := range getTestCases(blockBatchSenderPrivKey) {
		backendTimestamp = 0
		api := getTestPublicTransactionPoolAPI(txSignerPrivKey, blockBatchSenderPrivKey, testCase.backendContext)
		res := api.SendBlockBatches(testCase.inputCtx, testCase.inputMessageAndSig)
		h := func(r []error) bool {
			for _, e := range r {
				if e != nil {
					return true
				}
			}
			return false
		}
		hasErrors := h(res)

		// For debugging and verification:
		fmt.Printf("test case %d had output errors: %v\n", testNum, res)
		if testCase.hasErrors && !hasErrors {
			t.Fatalf("test case %d expected output errors but did not result in any. Errors: %v", testNum, res)
		}
		if !testCase.hasErrors && hasErrors {
			t.Fatalf("test case %d did not expect output errors but resulted in %d. Errors: %v", testNum, len(res), res)
		}
		if hasErrors && len(testCase.backendContext.sendTxsErrors) > 0 {
			// Note: Cannot handle test cases with multiple batches the same way because errors are aggregated from the endpoint and not from sendTxsErrors
			if testCase.multipleBatches {
				errorCount := func(r []error) int {
					c := 0
					for _, e := range r {
						if e != nil {
							c++
						}
					}
					return c
				}
				if errorCount(res) != errorCount(testCase.backendContext.sendTxsErrors) {
					t.Fatalf("test case %d expected %d errors but resulted in %d", testNum, errorCount(res), errorCount(testCase.backendContext.sendTxsErrors))
				}

			} else {
				if len(res) != len(testCase.backendContext.sendTxsErrors) {
					t.Fatalf("test case %d expected %d output errors but received %d. Errors: %v", testNum, len(testCase.backendContext.sendTxsErrors), len(res), res)
				}
				for i, err := range res {
					if err != nil && testCase.backendContext.sendTxsErrors[i] == nil {
						t.Fatalf("test case %d had an error output mismatch. Received error at index %d when one wasn't expected. Expected output: %v, output: %v", testNum, i, testCase.backendContext.sendTxsErrors, res)
					}
					if err == nil && testCase.backendContext.sendTxsErrors[i] != nil {
						t.Fatalf("test case %d had an error output mismatch. Did not receive an error at index %d when one was expected. Expected output: %v, output: %v", testNum, i, testCase.backendContext.sendTxsErrors, res)
					}
				}
			}
		}
		if backendTimestamp != testCase.resultingTimestamp {
			t.Fatalf("test case %d should have updated timestamp to %d but it was %d after execution.", testNum, testCase.resultingTimestamp, backendTimestamp)
		}
	}
}

func getDummyErrors(errorIndicies []int, outputSize int) []error {
	errs := make([]error, outputSize)
	for _, i := range errorIndicies {
		errs[i] = fmt.Errorf("error %d", i)
	}
	return errs
}

func getRandomRollupTransaction() *RollupTransaction {
	gasLimit := hexutil.Uint64(uint64(0))
	return &RollupTransaction{
		Nonce:    &internalTxNonce,
		GasLimit: &gasLimit,
		Sender:   &internalTxSender,
		Target:   &internalTxTarget,
		Calldata: &internalTxCalldata,
	}
}

func getBlockBatchesInputMessageAndSignature(privKey *ecdsa.PrivateKey, timestamp int64, blockNumber int, batchSizes []int) []hexutil.Bytes {
	ts := hexutil.Uint64(uint64(timestamp))
	blockNum := hexutil.Uint64(uint64(blockNumber))

	batches := make([][]*RollupTransaction, len(batchSizes))
	for i, s := range batchSizes {
		batches[i] = make([]*RollupTransaction, s)
		for index := 0; index < s; index++ {
			batches[i][index] = getRandomRollupTransaction()
		}
	}
	bb := &BlockBatches{
		Timestamp:   &ts,
		BlockNumber: &blockNum,
		Batches:     batches,
	}

	message, _ := json.Marshal(bb)
	return getInputMessageAndSignature(message, privKey)
}

func getInputMessageAndSignature(message []byte, privKey *ecdsa.PrivateKey) []hexutil.Bytes {
	sig, _ := crypto.Sign(crypto.Keccak256(message), privKey)
	return []hexutil.Bytes{message, sig}
}

func getFakeContext() context.Context {
	return &awstesting.FakeContext{
		Error:  fmt.Errorf("fake error%s", "!"),
		DoneCh: make(chan struct{}, 1),
	}
}

func getTestPublicTransactionPoolAPI(txSignerPrivKey *ecdsa.PrivateKey, blockBatchSenderPrivKey *ecdsa.PrivateKey, backendContext backendContext) *PublicTransactionPoolAPI {
	backend := newMockBackend(&blockBatchSenderPrivKey.PublicKey, backendContext)
	return NewPublicTransactionPoolAPI(backend, nil, txSignerPrivKey)
}

type backendContext struct {
	currentBlockNumber int64
	signerNonce        uint64
	sendTxsErrors      []error
}

type mockBackend struct {
	blockBatchSender *ecdsa.PublicKey
	testContext      backendContext
	timestamp        int64
}

func newMockBackend(blockBatchSender *ecdsa.PublicKey, backendContext backendContext) mockBackend {
	return mockBackend{
		blockBatchSender: blockBatchSender,
		testContext:      backendContext,
	}
}

func (m mockBackend) Downloader() *downloader.Downloader {
	panic("not implemented")
}

func (m mockBackend) ProtocolVersion() int {
	panic("not implemented")
}

func (m mockBackend) SuggestPrice(ctx context.Context) (*big.Int, error) {
	panic("not implemented")
}

func (m mockBackend) ChainDb() ethdb.Database {
	panic("not implemented")
}

func (m mockBackend) AccountManager() *accounts.Manager {
	panic("not implemented")
}

func (m mockBackend) ExtRPCEnabled() bool {
	panic("not implemented")
}

func (m mockBackend) RPCGasCap() *big.Int {
	panic("not implemented")
}

func (m mockBackend) SetHead(number uint64) {
	panic("not implemented")
}

func (m mockBackend) HeaderByNumber(ctx context.Context, number rpc.BlockNumber) (*types.Header, error) {
	panic("not implemented")
}

func (m mockBackend) HeaderByHash(ctx context.Context, hash common.Hash) (*types.Header, error) {
	panic("not implemented")
}

func (m mockBackend) HeaderByNumberOrHash(ctx context.Context, blockNrOrHash rpc.BlockNumberOrHash) (*types.Header, error) {
	panic("not implemented")
}

func (m mockBackend) BlockByNumber(ctx context.Context, number rpc.BlockNumber) (*types.Block, error) {
	panic("not implemented")
}

func (m mockBackend) BlockByHash(ctx context.Context, hash common.Hash) (*types.Block, error) {
	panic("not implemented")
}

func (m mockBackend) BlockByNumberOrHash(ctx context.Context, blockNrOrHash rpc.BlockNumberOrHash) (*types.Block, error) {
	panic("not implemented")
}

func (m mockBackend) StateAndHeaderByNumber(ctx context.Context, number rpc.BlockNumber) (*state.StateDB, *types.Header, error) {
	panic("not implemented")
}

func (m mockBackend) StateAndHeaderByNumberOrHash(ctx context.Context, blockNrOrHash rpc.BlockNumberOrHash) (*state.StateDB, *types.Header, error) {
	panic("not implemented")
}

func (m mockBackend) GetReceipts(ctx context.Context, hash common.Hash) (types.Receipts, error) {
	panic("not implemented")
}

func (m mockBackend) GetTd(hash common.Hash) *big.Int {
	panic("not implemented")
}

func (m mockBackend) GetEVM(ctx context.Context, msg core.Message, state *state.StateDB, header *types.Header) (*vm.EVM, func() error, error) {
	panic("not implemented")
}

func (m mockBackend) SubscribeChainEvent(ch chan<- core.ChainEvent) event.Subscription {
	panic("not implemented")
}

func (m mockBackend) SubscribeChainHeadEvent(ch chan<- core.ChainHeadEvent) event.Subscription {
	panic("not implemented")
}

func (m mockBackend) SubscribeChainSideEvent(ch chan<- core.ChainSideEvent) event.Subscription {
	panic("not implemented")
}

func (m mockBackend) SendTx(ctx context.Context, signedTx *types.Transaction) error {
	panic("not implemented")
}

func (m mockBackend) GetTransaction(ctx context.Context, txHash common.Hash) (*types.Transaction, common.Hash, uint64, uint64, error) {
	panic("not implemented")
}

func (m mockBackend) GetPoolTransactions() (types.Transactions, error) {
	panic("not implemented")
}

func (m mockBackend) GetPoolTransaction(txHash common.Hash) *types.Transaction {
	panic("not implemented")
}

func (m mockBackend) GetPoolNonce(ctx context.Context, addr common.Address) (uint64, error) {
	return m.testContext.signerNonce, nil
}

func (m mockBackend) Stats() (pending int, queued int) {
	panic("not implemented")
}

func (m mockBackend) TxPoolContent() (map[common.Address]types.Transactions, map[common.Address]types.Transactions) {
	panic("not implemented")
}

func (m mockBackend) SubscribeNewTxsEvent(chan<- core.NewTxsEvent) event.Subscription {
	panic("not implemented")
}

func (m mockBackend) BloomStatus() (uint64, uint64) {
	panic("not implemented")
}

func (m mockBackend) GetLogs(ctx context.Context, blockHash common.Hash) ([][]*types.Log, error) {
	panic("not implemented")
}

func (m mockBackend) ServiceFilter(ctx context.Context, session *bloombits.MatcherSession) {
	panic("not implemented")
}

func (m mockBackend) SubscribeLogsEvent(ch chan<- []*types.Log) event.Subscription {
	panic("not implemented")
}

func (m mockBackend) SubscribePendingLogsEvent(ch chan<- []*types.Log) event.Subscription {
	panic("not implemented")
}

func (m mockBackend) SubscribeRemovedLogsEvent(ch chan<- core.RemovedLogsEvent) event.Subscription {
	panic("not implemented")
}

func (m mockBackend) SendTxs(ctx context.Context, signedTxs []*types.Transaction) []error {
	if len(m.testContext.sendTxsErrors) == 0 || len(m.testContext.sendTxsErrors) != len(signedTxs) {
		return make([]error, len(signedTxs))
	}
	return m.testContext.sendTxsErrors
}

func (m mockBackend) SetTimestamp(timestamp int64) {
	backendTimestamp = timestamp
}

func (m mockBackend) ChainConfig() *params.ChainConfig {
	return &params.ChainConfig{
		BlockBatchesSender: m.blockBatchSender,
	}
}

func (m mockBackend) CurrentBlock() *types.Block {
	header := &types.Header{
		ParentHash:  common.Hash{},
		UncleHash:   common.Hash{},
		Coinbase:    common.Address{},
		Root:        common.Hash{},
		TxHash:      common.Hash{},
		ReceiptHash: common.Hash{},
		Bloom:       types.Bloom{},
		Difficulty:  nil,
		Number:      big.NewInt(m.testContext.currentBlockNumber),
		GasLimit:    0,
		GasUsed:     0,
		Time:        0,
		Extra:       nil,
		MixDigest:   common.Hash{},
		Nonce:       types.BlockNonce{},
	}

	return types.NewBlock(header, []*types.Transaction{}, []*types.Header{}, []*types.Receipt{})
}
