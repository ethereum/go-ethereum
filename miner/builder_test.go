package miner

import (
	"encoding/json"
	"fmt"
	"math/big"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/consensus/clique"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/params"
	suavextypes "github.com/ethereum/go-ethereum/suave/builder/api"
	"github.com/stretchr/testify/require"
)

func TestBuilder_AddTxn_Simple(t *testing.T) {
	t.Parallel()
	config, backend := newMockBuilderConfig(t)

	builder, err := NewBuilder(config, &BuilderArgs{})
	require.NoError(t, err)

	tx1 := backend.newRandomTx(false)

	res, err := builder.AddTransaction(tx1)
	require.NoError(t, err)
	require.True(t, res.Success)
	require.Len(t, builder.env.receipts, 1)
	require.Equal(t, big.NewInt(1000), builder.env.state.GetBalance(testUserAddress))

	// we cannot add the same transaction again. Note that by design the
	// function does not error but returns the SimulateTransactionResult.success = false
	res, err = builder.AddTransaction(tx1)
	require.NoError(t, err)
	require.False(t, res.Success)
	require.Len(t, builder.env.receipts, 1)
	require.Equal(t, big.NewInt(1000), builder.env.state.GetBalance(testUserAddress))
}

func TestBuilder_AddTxns_Simple(t *testing.T) {
	t.Parallel()
	config, backend := newMockBuilderConfig(t)
	builder, err := NewBuilder(config, &BuilderArgs{})
	require.NoError(t, err)

	tx1 := backend.newRandomTx(false)
	tx2 := backend.newRandomTxWithNonce(1)

	res, err := builder.AddTransactions([]*types.Transaction{tx1, tx2})
	require.NoError(t, err)
	require.Len(t, res, 2)
	for _, r := range res {
		require.True(t, r.Success)
	}
	require.Equal(t, big.NewInt(2000), builder.env.state.GetBalance(testUserAddress))

	tx3 := backend.newRandomTxWithNonce(2)
	tx4 := backend.newRandomTxWithNonce(1000) // fails with nonce too high

	res, err = builder.AddTransactions([]*types.Transaction{tx3, tx4})
	require.NoError(t, err)
	require.Len(t, res, 2)
	require.True(t, res[0].Success)
	require.False(t, res[1].Success)
	require.Len(t, builder.env.txs, 2)
	require.Equal(t, big.NewInt(2000), builder.env.state.GetBalance(testUserAddress))
}

func TestBuilder_AddBundles_Simple(t *testing.T) {
	t.Parallel()
	config, backend := newMockBuilderConfig(t)
	builder, err := NewBuilder(config, &BuilderArgs{})
	require.NoError(t, err)

	tx1 := backend.newRandomTx(false)
	tx2 := backend.newRandomTxWithNonce(1)

	bundle1 := &suavextypes.Bundle{
		Txs: []*types.Transaction{tx1, tx2},
	}

	tx3 := backend.newRandomTxWithNonce(2)
	tx4 := backend.newRandomTxWithNonce(3)

	bundle2 := &suavextypes.Bundle{
		Txs: []*types.Transaction{tx3, tx4},
	}

	res, err := builder.AddBundles([]*suavextypes.Bundle{bundle1, bundle2})
	require.NoError(t, err)
	require.Len(t, res, 2)
	require.True(t, res[0].Success)
	require.True(t, res[1].Success)
	require.Equal(t, big.NewInt(4000), builder.env.state.GetBalance(testUserAddress))
}

func TestBuilder_AddBundles_RevertHashes(t *testing.T) {
	t.Parallel()
	config, backend := newMockBuilderConfig(t)
	builder, err := NewBuilder(config, &BuilderArgs{})
	require.NoError(t, err)

	tx1 := backend.newRandomTx(false)
	tx2 := backend.newRandomTxWithNonce(3) // fails with nonce too high

	bundle := &suavextypes.Bundle{
		Txs: []*types.Transaction{tx1, tx2},
	}

	res, err := builder.AddBundles([]*suavextypes.Bundle{bundle})
	require.NoError(t, err)
	require.Len(t, res, 1)
	require.False(t, res[0].Success)
	require.Len(t, res[0].SimulateTransactionResults, 2)
	require.True(t, res[0].SimulateTransactionResults[0].Success)
	require.False(t, res[0].SimulateTransactionResults[1].Success)
	require.Equal(t, big.NewInt(0), builder.env.state.GetBalance(testUserAddress))

	bundle.RevertingHashes = []common.Hash{tx2.Hash()}

	res, err = builder.AddBundles([]*suavextypes.Bundle{bundle})
	require.NoError(t, err)
	require.Len(t, res, 1)
	require.True(t, res[0].Success)
	require.Len(t, res[0].SimulateTransactionResults, 2)
	require.True(t, res[0].SimulateTransactionResults[0].Success)
	require.False(t, res[0].SimulateTransactionResults[1].Success)
	require.Equal(t, big.NewInt(1000), builder.env.state.GetBalance(testUserAddress))
}

func TestBuilder_AddBundles_InvalidParams(t *testing.T) {
	t.Parallel()
	config, backend := newMockBuilderConfig(t)
	// set builder target block number to 10
	backend.insertRandomBlocks(9)

	builder, err := NewBuilder(config, &BuilderArgs{})
	require.Equal(t, uint64(10), builder.env.header.Number.Uint64())
	require.NoError(t, err)

	tx1 := backend.newRandomTx(false)
	tx2 := backend.newRandomTx(false)

	bundle := &suavextypes.Bundle{
		Txs:         []*types.Transaction{tx1, tx2},
		BlockNumber: big.NewInt(20),
	}

	res, err := builder.AddBundles([]*suavextypes.Bundle{bundle})
	require.NoError(t, err)
	require.Len(t, res, 1)
	require.False(t, res[0].Success)
	require.Equal(t, ErrInvalidBlockNumber.Error(), res[0].Error)
	require.Len(t, res[0].SimulateTransactionResults, 0)
	require.Equal(t, big.NewInt(0), builder.env.state.GetBalance(testUserAddress))

	bundle = &suavextypes.Bundle{
		Txs:         []*types.Transaction{tx1, tx2},
		BlockNumber: big.NewInt(5),
		MaxBlock:    big.NewInt(6),
	}

	res, err = builder.AddBundles([]*suavextypes.Bundle{bundle})
	require.NoError(t, err)
	require.Len(t, res, 1)
	require.False(t, res[0].Success)
	require.Equal(t, ErrExceedsMaxBlock.Error(), res[0].Error)
	require.Len(t, res[0].SimulateTransactionResults, 0)
	require.Equal(t, big.NewInt(0), builder.env.state.GetBalance(testUserAddress))

	bundle = &suavextypes.Bundle{
		Txs: []*types.Transaction{},
	}

	res, err = builder.AddBundles([]*suavextypes.Bundle{bundle})
	require.NoError(t, err)
	require.False(t, res[0].Success)
	require.Equal(t, ErrEmptyTxs.Error(), res[0].Error)
	require.Len(t, res[0].SimulateTransactionResults, 0)
	require.Equal(t, big.NewInt(0), builder.env.state.GetBalance(testUserAddress))
}

func TestBuilder_FillTransactions(t *testing.T) {
	t.Parallel()
	config, backend := newMockBuilderConfig(t)

	builder, err := NewBuilder(config, &BuilderArgs{})
	require.NoError(t, err)

	tx1 := backend.newRandomTx(true)
	errArr := backend.TxPool().Add(types.Transactions{tx1}, false, true)
	require.NoError(t, errArr[0])

	tx2 := backend.newRandomTx(true)
	errArr = backend.TxPool().Add(types.Transactions{tx2}, false, true)
	require.NoError(t, errArr[0])

	require.NoError(t, builder.FillPending())
	require.Len(t, builder.env.receipts, 2)

	require.Equal(t, tx1.Hash(), builder.env.receipts[0].TxHash)
	require.Equal(t, tx2.Hash(), builder.env.receipts[1].TxHash)
}

func TestBuilder_BuildBlock(t *testing.T) {
	t.Parallel()

	config, backend := newMockBuilderConfig(t)

	builder, err := NewBuilder(config, &BuilderArgs{})
	require.NoError(t, err)

	tx1 := backend.newRandomTx(true)

	_, err = builder.AddTransaction(tx1)
	require.NoError(t, err)

	block, err := builder.BuildBlock()
	require.NoError(t, err)
	require.NotNil(t, block)
	require.Len(t, block.Transactions(), 1)
}

func TestBuilder_ContractWithLogs(t *testing.T) {
	// test that we can simulate a txn with a contract that emits events
	t.Parallel()

	config, backend := newMockBuilderConfig(t)

	input, err := suaveExample1Artifact.Abi.Pack("get", big.NewInt(1))
	require.NoError(t, err)

	tx := backend.newCall(suaveExample1Addr, input)

	builder, err := NewBuilder(config, &BuilderArgs{})
	require.NoError(t, err)

	simResult, err := builder.AddTransaction(tx)
	require.NoError(t, err)
	require.True(t, simResult.Success)
	require.Len(t, simResult.Logs, 1)

	require.Equal(t, simResult.Logs[0].Addr, suaveExample1Addr)
	require.Equal(t, simResult.Logs[0].Topics[0], suaveExample1Artifact.Abi.Events["SomeEvent"].ID)
}

func TestBuilder_Bid(t *testing.T) {
	t.Parallel()

	config, _ := newMockBuilderConfig(t)

	builder, err := NewBuilder(config, &BuilderArgs{})
	require.NoError(t, err)

	_, err = builder.Bid([48]byte{})
	require.Error(t, err, "cannot create bid without block")

	_, err = builder.BuildBlock()
	require.NoError(t, err)

	_, err = builder.Bid([48]byte{})
	require.NoError(t, err)
}

func TestBuilder_Balance(t *testing.T) {
	t.Parallel()

	config, backend := newMockBuilderConfig(t)

	builder, err := NewBuilder(config, &BuilderArgs{})
	require.NoError(t, err)

	balance := builder.GetBalance(testBankAddress)
	require.Equal(t, balance, testBankFunds)

	// make a random txn that consumes gas
	tx1 := backend.newRandomTx(true)
	_, err = builder.AddTransaction(tx1)
	require.NoError(t, err)

	balance2 := builder.GetBalance(testBankAddress)
	require.NotEqual(t, balance2, testBankFunds)
}

func newMockBuilderConfig(t *testing.T) (*BuilderConfig, *testWorkerBackend) {
	var (
		db     = rawdb.NewMemoryDatabase()
		config = *params.AllCliqueProtocolChanges
	)
	config.Clique = &params.CliqueConfig{Period: 1, Epoch: 30000}
	engine := clique.New(config.Clique, db)

	w, backend := newTestWorker(t, &config, engine, db, 0)
	w.close()

	bConfig := &BuilderConfig{
		ChainConfig: w.chainConfig,
		Engine:      w.engine,
		EthBackend:  w.eth,
		Chain:       w.chain,
		GasCeil:     10000000,
	}
	return bConfig, backend
}

func (b *testWorkerBackend) newRandomTxWithNonce(nonce uint64) *types.Transaction {
	gasPrice := big.NewInt(10 * params.InitialBaseFee)
	tx, _ := types.SignTx(types.NewTransaction(nonce, testUserAddress, big.NewInt(1000), params.TxGas, gasPrice, nil), types.HomesteadSigner{}, testBankKey)
	return tx
}

func (b *testWorkerBackend) insertRandomBlocks(n int) []*types.Block {
	extraVanity := 32
	extraSeal := crypto.SignatureLength
	diffInTurn := big.NewInt(2)
	signer := new(types.HomesteadSigner)
	_, blocks, _ := core.GenerateChainWithGenesis(b.genesis, b.chain.Engine(), n, func(i int, block *core.BlockGen) {
		block.SetDifficulty(big.NewInt(2)) // diffInTurn

		if i != 1 {
			tx, err := types.SignTx(types.NewTransaction(block.TxNonce(testBankAddress), common.Address{0x00}, new(big.Int), params.TxGas, block.BaseFee(), nil), signer, testBankKey)
			if err != nil {
				panic(err)
			}
			block.AddTxWithChain(b.chain, tx)
		}
	})

	for i, block := range blocks {
		header := block.Header()
		if i > 0 {
			header.ParentHash = blocks[i-1].Hash()
		}
		header.Extra = make([]byte, extraVanity+extraSeal)
		header.Difficulty = diffInTurn

		sig, _ := crypto.Sign(clique.SealHash(header).Bytes(), testBankKey)
		copy(header.Extra[len(header.Extra)-extraSeal:], sig)
		blocks[i] = block.WithSeal(header)
	}

	if _, err := b.chain.InsertChain(blocks); err != nil {
		panic(fmt.Sprintf("failed to insert initial blocks: %v", err))
	}

	return blocks
}

func (b *testWorkerBackend) newCall(to common.Address, data []byte) *types.Transaction {
	gasPrice := big.NewInt(10 * params.InitialBaseFee)
	tx, _ := types.SignTx(types.NewTransaction(b.txPool.Nonce(testBankAddress), to, big.NewInt(0), 1000000, gasPrice, data), types.HomesteadSigner{}, testBankKey)
	return tx
}

var (
	suaveExample1Addr     = common.Address{0x1}
	suaveExample1Artifact = readContractArtifact("Example.sol/Example.json")
)

type artifactObj struct {
	Abi              *abi.ABI `json:"abi"`
	DeployedBytecode struct {
		Object string
	} `json:"deployedBytecode"`
}

func readContractArtifact(name string) *artifactObj {
	// Get the caller's file path.
	_, filename, _, _ := runtime.Caller(1)

	// Resolve the directory of the caller's file.
	callerDir := filepath.Dir(filename)

	// Construct the absolute path to the target file.
	targetFilePath := filepath.Join(callerDir, "./contracts/out/", name)

	data, err := os.ReadFile(targetFilePath)
	if err != nil {
		panic(fmt.Sprintf("failed to read artifact %s: %v. Maybe you forgot to generate the artifacts? `cd suave && forge build`", name, err))
	}

	var obj artifactObj
	if err := json.Unmarshal(data, &obj); err != nil {
		panic(fmt.Sprintf("failed to unmarshal artifact %s: %v", name, err))
	}
	return &obj
}
