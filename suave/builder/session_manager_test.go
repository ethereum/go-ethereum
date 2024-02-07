package builder

import (
	"context"
	"crypto/ecdsa"
	"math/big"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/consensus"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/params"
	"github.com/stretchr/testify/require"
)

func TestSessionManager_SessionTimeout(t *testing.T) {
	mngr, _ := newSessionManager(t, &Config{
		SessionIdleTimeout: 500 * time.Millisecond,
	})

	id, err := mngr.NewSession(context.TODO(), nil)
	require.NoError(t, err)

	time.Sleep(1 * time.Second)

	_, err = mngr.getSession(id)
	require.Error(t, err)
}

func TestSessionManager_MaxConcurrentSessions(t *testing.T) {
	t.Parallel()

	const d = time.Millisecond * 100

	mngr, _ := newSessionManager(t, &Config{
		MaxConcurrentSessions: 1,
		SessionIdleTimeout:    d,
	})

	t.Run("SessionAvailable", func(t *testing.T) {
		sess, err := mngr.NewSession(context.TODO(), nil)
		require.NoError(t, err)
		require.NotZero(t, sess)
	})

	t.Run("ContextExpired", func(t *testing.T) {
		t.Skip("Skipping flaky test")

		ctx, cancel := context.WithCancel(context.Background())
		cancel()

		sess, err := mngr.NewSession(ctx, nil)
		require.Zero(t, sess)
		require.ErrorIs(t, err, context.Canceled)
	})

	t.Run("SessionExpired", func(t *testing.T) {
		time.Sleep(d) // Wait for the session to expire.

		// We should be able to open a session again.
		sess, err := mngr.NewSession(context.TODO(), nil)
		require.NoError(t, err)
		require.NotZero(t, sess)
	})
}

func TestSessionManager_SessionRefresh(t *testing.T) {
	mngr, _ := newSessionManager(t, &Config{
		SessionIdleTimeout: 500 * time.Millisecond,
	})

	id, err := mngr.NewSession(context.TODO(), nil)
	require.NoError(t, err)

	// if we query the session under the idle timeout,
	// we should be able to refresh it
	for i := 0; i < 5; i++ {
		time.Sleep(250 * time.Millisecond)

		_, err = mngr.getSession(id)
		require.NoError(t, err)
	}

	// if we query the session after the idle timeout,
	// we should get an error

	time.Sleep(1 * time.Second)

	_, err = mngr.getSession(id)
	require.Error(t, err)
}

func TestSessionManager_StartSession(t *testing.T) {
	// test that the session starts and it can simulate transactions
	mngr, bMock := newSessionManager(t, &Config{})

	id, err := mngr.NewSession(context.TODO(), nil)
	require.NoError(t, err)

	txn := bMock.state.newTransfer(t, common.Address{}, big.NewInt(1))
	receipt, err := mngr.AddTransaction(id, txn)
	require.NoError(t, err)
	require.NotNil(t, receipt)
}

func newSessionManager(t *testing.T, cfg *Config) (*SessionManager, *blockchainMock) {
	if cfg == nil {
		cfg = &Config{}
	}

	state := newMockState(t)

	bMock := &blockchainMock{
		state: state,
	}
	return NewSessionManager(bMock, cfg), bMock
}

type blockchainMock struct {
	state *mockState
}

func (b *blockchainMock) Engine() consensus.Engine {
	panic("TODO")
}

func (b *blockchainMock) GetHeader(common.Hash, uint64) *types.Header {
	panic("TODO")
}

func (b *blockchainMock) Config() *params.ChainConfig {
	return b.state.chainConfig
}

func (b *blockchainMock) CurrentHeader() *types.Header {
	return &types.Header{
		Number:     big.NewInt(1),
		Difficulty: big.NewInt(1),
		Root:       b.state.stateRoot,
	}
}

func (b *blockchainMock) StateAt(root common.Hash) (*state.StateDB, error) {
	return b.state.stateAt(root)
}

type mockState struct {
	stateRoot common.Hash
	statedb   state.Database

	premineKey    *ecdsa.PrivateKey
	premineKeyAdd common.Address

	nextNonce uint64 // figure out a better way
	signer    types.Signer

	chainConfig *params.ChainConfig
}

func newMockState(t *testing.T) *mockState {
	premineKey, _ := crypto.GenerateKey() // TODO: it would be nice to have it deterministic
	premineKeyAddr := crypto.PubkeyToAddress(premineKey.PublicKey)

	// create a state reference with at least one premined account
	// In order to test the statedb in isolation, we are going
	// to commit this pre-state to a memory database
	db := state.NewDatabase(rawdb.NewMemoryDatabase())
	preState, err := state.New(types.EmptyRootHash, db, nil)
	require.NoError(t, err)

	preState.AddBalance(premineKeyAddr, big.NewInt(1000000000000000000))

	root, err := preState.Commit(1, true)
	require.NoError(t, err)

	// for the sake of this test, we only need all the forks enabled
	chainConfig := params.TestChainConfig

	// Disable london so that we do not check gasFeeCap (TODO: Fix)
	chainConfig.LondonBlock = big.NewInt(100)

	return &mockState{
		statedb:       db,
		stateRoot:     root,
		premineKey:    premineKey,
		premineKeyAdd: premineKeyAddr,
		signer:        types.NewEIP155Signer(chainConfig.ChainID),
		chainConfig:   chainConfig,
	}
}

func (m *mockState) stateAt(root common.Hash) (*state.StateDB, error) {
	return state.New(root, m.statedb, nil)
}

func (m *mockState) getNonce() uint64 {
	next := m.nextNonce
	m.nextNonce++
	return next
}

func (m *mockState) newTransfer(t *testing.T, to common.Address, amount *big.Int) *types.Transaction {
	tx := types.NewTransaction(m.getNonce(), to, amount, 1000000, big.NewInt(1), nil)
	return m.newTxn(t, tx)
}

func (m *mockState) newTxn(t *testing.T, tx *types.Transaction) *types.Transaction {
	// sign the transaction
	signature, err := crypto.Sign(m.signer.Hash(tx).Bytes(), m.premineKey)
	require.NoError(t, err)

	// include the signature in the transaction
	tx, err = tx.WithSignature(m.signer, signature)
	require.NoError(t, err)

	return tx
}
