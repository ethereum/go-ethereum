package builder

import (
	"context"
	"math/big"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/accounts"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/consensus/clique"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/core/txpool"
	"github.com/ethereum/go-ethereum/core/txpool/legacypool"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/params"
	"github.com/ethereum/go-ethereum/suave/builder/api"
	"github.com/stretchr/testify/require"
)

func TestSessionManager_SessionTimeout(t *testing.T) {
	mngr, _ := newSessionManager(t, &Config{
		SessionIdleTimeout: 500 * time.Millisecond,
	})

	args := &api.BuildBlockArgs{}

	id, err := mngr.NewSession(context.TODO(), args)
	require.NoError(t, err)

	time.Sleep(1 * time.Second)

	_, err = mngr.getSession(id, false)
	require.Error(t, err)
}

func TestSessionManager_MaxConcurrentSessions(t *testing.T) {
	t.Parallel()

	const d = time.Millisecond * 100
	args := &api.BuildBlockArgs{}

	mngr, _ := newSessionManager(t, &Config{
		MaxConcurrentSessions: 1,
		SessionIdleTimeout:    d,
	})

	t.Run("SessionAvailable", func(t *testing.T) {
		sess, err := mngr.NewSession(context.TODO(), args)
		require.NoError(t, err)
		require.NotZero(t, sess)
	})

	t.Run("ContextExpired", func(t *testing.T) {
		t.Skip("Skipping flaky test")

		ctx, cancel := context.WithCancel(context.Background())
		cancel()

		sess, err := mngr.NewSession(ctx, args)
		require.Zero(t, sess)
		require.ErrorIs(t, err, context.Canceled)
	})

	t.Run("SessionExpired", func(t *testing.T) {
		time.Sleep(d) // Wait for the session to expire.

		// We should be able to open a session again.
		sess, err := mngr.NewSession(context.TODO(), args)
		require.NoError(t, err)
		require.NotZero(t, sess)
	})
}

func TestSessionManager_SessionRefresh(t *testing.T) {
	mngr, _ := newSessionManager(t, &Config{
		SessionIdleTimeout: 500 * time.Millisecond,
	})

	args := &api.BuildBlockArgs{}
	id, err := mngr.NewSession(context.TODO(), args)
	require.NoError(t, err)

	// if we query the session under the idle timeout,
	// we should be able to refresh it
	for i := 0; i < 5; i++ {
		time.Sleep(250 * time.Millisecond)

		_, err = mngr.getSession(id, false)
		require.NoError(t, err)
	}

	// if we query the session after the idle timeout,
	// we should get an error

	time.Sleep(1 * time.Second)

	_, err = mngr.getSession(id, false)
	require.Error(t, err)
}

func TestSessionManager_StartSession(t *testing.T) {
	// test that the session starts and it can simulate transactions
	mngr, bMock := newSessionManager(t, &Config{})

	args := &api.BuildBlockArgs{}
	id, err := mngr.NewSession(context.TODO(), args)
	require.NoError(t, err)

	txn := bMock.newTransfer(t, common.Address{}, big.NewInt(1))
	receipt, err := mngr.AddTransaction(id, txn)
	require.NoError(t, err)
	require.NotNil(t, receipt)

	// test that you can simulate the transaction on the fly
	receipt2, err := mngr.AddTransaction("", txn)
	require.NoError(t, err)
	require.Equal(t, receipt, receipt2)
}

func newSessionManager(t *testing.T, cfg *Config) (*SessionManager, *testBackend) {
	backend := newTestBackend(t)

	if cfg == nil {
		cfg = &Config{}
	}
	return NewSessionManager(backend.chain, backend.pool, cfg), backend
}

var (
	testBankKey, _  = crypto.GenerateKey()
	testBankAddress = crypto.PubkeyToAddress(testBankKey.PublicKey)
)

type testBackend struct {
	chain *core.BlockChain
	pool  *txpool.TxPool
}

func (tb *testBackend) newTransfer(t *testing.T, to common.Address, amount *big.Int) *types.Transaction {
	gasPrice := big.NewInt(10 * params.InitialBaseFee)
	tx, _ := types.SignTx(types.NewTransaction(tb.pool.Nonce(testBankAddress), to, amount, params.TxGas, gasPrice, nil), types.HomesteadSigner{}, testBankKey)
	return tx
}

func newTestBackend(t *testing.T) *testBackend {
	// code based on miner 'newTestWorker'
	testTxPoolConfig := legacypool.DefaultConfig
	testTxPoolConfig.Journal = ""

	var (
		db     = rawdb.NewMemoryDatabase()
		config = *params.AllCliqueProtocolChanges
	)
	config.Clique = &params.CliqueConfig{Period: 1, Epoch: 30000}
	engine := clique.New(config.Clique, db)

	var gspec = &core.Genesis{
		Config: &config,
		Alloc:  core.GenesisAlloc{testBankAddress: {Balance: big.NewInt(1000000000000000000)}},
	}

	gspec.ExtraData = make([]byte, 32+common.AddressLength+crypto.SignatureLength)
	copy(gspec.ExtraData[32:32+common.AddressLength], testBankAddress.Bytes())
	engine.Authorize(testBankAddress, func(account accounts.Account, s string, data []byte) ([]byte, error) {
		return crypto.Sign(crypto.Keccak256(data), testBankKey)
	})

	chain, err := core.NewBlockChain(db, &core.CacheConfig{TrieDirtyDisabled: true}, gspec, nil, engine, vm.Config{}, nil, nil)
	if err != nil {
		t.Fatalf("core.NewBlockChain failed: %v", err)
	}
	pool := legacypool.New(testTxPoolConfig, chain)
	txpool, _ := txpool.New(testTxPoolConfig.PriceLimit, chain, []txpool.SubPool{pool})

	return &testBackend{chain: chain, pool: txpool}
}
