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
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/params"
	"github.com/stretchr/testify/require"
)

func TestBuilder_AddTxn_Simple(t *testing.T) {
	t.Parallel()
	config, backend := newMockBuilderConfig(t)

	builder, err := NewBuilder(config, &BuilderArgs{})
	require.NoError(t, err)

	tx1 := backend.newRandomTx(true)

	res, err := builder.AddTransaction(tx1)
	require.NoError(t, err)
	require.True(t, res.Success)
	require.Len(t, builder.env.receipts, 1)

	// we cannot add the same transaction again. Note that by design the
	// function does not error but returns the SimulateTransactionResult.success = false
	res, err = builder.AddTransaction(tx1)
	require.NoError(t, err)
	require.False(t, res.Success)
	require.Len(t, builder.env.receipts, 1)
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

	req, err := builder.Bid([48]byte{})
	require.NoError(t, err)

	fmt.Println("-- req --", req)
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
