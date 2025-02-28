package system_contract

import (
	"context"
	"math/big"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/scroll-tech/go-ethereum"
	"github.com/scroll-tech/go-ethereum/accounts"
	"github.com/scroll-tech/go-ethereum/common"
	"github.com/scroll-tech/go-ethereum/core/types"
	"github.com/scroll-tech/go-ethereum/log"
	"github.com/scroll-tech/go-ethereum/params"
	"github.com/scroll-tech/go-ethereum/rollup/sync_service"
	"github.com/scroll-tech/go-ethereum/trie"
)

var _ sync_service.EthClient = &fakeEthClient{}

func TestSystemContract_FetchSigner(t *testing.T) {
	log.Root().SetHandler(log.DiscardHandler())

	expectedSigner := common.HexToAddress("0x1234567890abcdef1234567890abcdef12345678")

	fakeClient := &fakeEthClient{value: expectedSigner}

	config := &params.SystemContractConfig{
		SystemContractAddress: common.HexToAddress("0xFAKE"),
		// The slot number can be arbitrary – fake client doesn't use it.
		Period:        10,
		RelaxedPeriod: false,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	sys := New(ctx, config, fakeClient)
	defer sys.Close()

	require.NoError(t, sys.fetchAddressFromL1())

	actualSigner := sys.currentSignerAddressL1()

	// Verify that the fetched signer equals the expectedSigner from our fake client.
	require.Equal(t, expectedSigner, actualSigner, "The SystemContract should update signerAddressL1 to the value provided by the client")
}

func TestSystemContract_AuthorizeCheck(t *testing.T) {
	// This test verifies that if the local signer does not match the authorized signer,
	// then the Seal() function returns an error.

	expectedSigner := common.HexToAddress("0x1234567890abcdef1234567890abcdef12345678")

	fakeClient := &fakeEthClient{value: expectedSigner}
	config := &params.SystemContractConfig{
		SystemContractAddress: common.HexToAddress("0xFAKE"),
		Period:                10,
		RelaxedPeriod:         false,
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	sys := New(ctx, config, fakeClient)
	defer sys.Close()

	require.NoError(t, sys.fetchAddressFromL1())

	// Authorize with a different signer than expected.
	differentSigner := common.HexToAddress("0xABCDEFabcdefABCDEFabcdefabcdefABCDEFABCD")
	sys.Authorize(differentSigner, func(acc accounts.Account, mimeType string, message []byte) ([]byte, error) {
		// For testing, return a dummy signature
		return []byte("dummy_sig"), nil
	})

	// Create a dummy block header.
	// We only need the block number and blocksignature data length for this test.
	header := &types.Header{
		Number: big.NewInt(100),
		// We use an extra slice with length equal to extraSeal
		BlockSignature: make([]byte, extraSeal),
	}

	// Call Seal() and expect an error since local signer != authorized signer.
	results := make(chan *types.Block)
	stop := make(chan struct{})
	err := sys.Seal(nil, (&types.Block{}).WithSeal(header), results, stop)

	require.Error(t, err, "Seal should return an error when the local signer is not authorized")
}

// TestSystemContract_SignsAfterUpdate simulates:
//  1. Initially, the SystemContract authorized signer (from StorageAt) is not the signer of the Block.
//  2. Later, after updating the fake client to the correct signer, the background
//     poll updates the SystemContract.
//  3. Once updated, if the local signing key is set to match, Seal() should succeed.
func TestSystemContract_SignsAfterUpdate(t *testing.T) {
	// Silence logging during tests.
	log.Root().SetHandler(log.DiscardHandler())

	// Define two addresses: one "wrong" and one "correct".
	oldSigner := common.HexToAddress("0x1111111111111111111111111111111111111111")
	updatedSigner := common.HexToAddress("0x2222222222222222222222222222222222222222")

	// Create a fake client that starts by returning the wrong signer.
	fakeClient := &fakeEthClient{
		value: oldSigner,
	}

	config := &params.SystemContractConfig{
		SystemContractAddress: common.HexToAddress("0xFAKE"), // Dummy value
		Period:                10,                            // arbitrary non-zero value
		RelaxedPeriod:         false,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	sys := New(ctx, config, fakeClient)
	defer sys.Close()

	require.NoError(t, sys.fetchAddressFromL1())

	// Verify that initially the fetched signer equals oldSigner.
	initialSigner := sys.currentSignerAddressL1()
	require.Equal(t, oldSigner, initialSigner, "Initial signerAddressL1 should be oldSigner")

	// Now, simulate an update: change the fake client's returned value to updatedSigner.
	fakeClient.mu.Lock()
	fakeClient.value = updatedSigner
	fakeClient.mu.Unlock()

	// fetch new value from L1 (simulating a background poll)
	require.NoError(t, sys.fetchAddressFromL1())

	// Verify that system contract's signerAddressL1 is now updated to updatedSigner.
	newSigner := sys.currentSignerAddressL1()
	require.Equal(t, newSigner, updatedSigner, "SignerAddressL1 should update to updatedSigner after polling")

	// Now simulate authorizing with the correct local signer.
	sys.Authorize(updatedSigner, func(acc accounts.Account, mimeType string, message []byte) ([]byte, error) {
		// For testing, return a dummy signature.
		return []byte("dummy_signature"), nil
	})

	// Create a dummy header for sealing.
	header := &types.Header{
		Number:         big.NewInt(100),
		BlockSignature: make([]byte, extraSeal),
	}

	// Construct a new block from the header using NewBlock constructor.
	block := types.NewBlock(header, nil, nil, nil, trie.NewStackTrie(nil))

	results := make(chan *types.Block)
	stop := make(chan struct{})

	// Call Seal. It should succeed (i.e. return no error) because local signer now equals authorized signer.
	err := sys.Seal(nil, block, results, stop)
	require.NoError(t, err, "Seal should succeed when the local signer is authorized after update")

	// Wait for the result from Seal's goroutine.
	select {
	case sealedBlock := <-results:
		require.NotNil(t, sealedBlock, "Seal should eventually return a sealed block")
		// Optionally, you may log or further inspect sealedBlock here.
	case <-time.After(15 * time.Second):
		t.Fatal("Timed out waiting for Seal to return a sealed block")
	}
}

// fakeEthClient implements a minimal version of sync_service.EthClient for testing purposes.
type fakeEthClient struct {
	mu sync.Mutex
	// value is the fixed value to return from StorageAt.
	// We'll assume StorageAt returns a 32-byte value representing an Ethereum address.
	value common.Address
}

// BlockNumber returns 0.
func (f *fakeEthClient) BlockNumber(ctx context.Context) (uint64, error) {
	return 0, nil
}

// ChainID returns a zero-value chain ID.
func (f *fakeEthClient) ChainID(ctx context.Context) (*big.Int, error) {
	return big.NewInt(0), nil
}

// FilterLogs returns an empty slice of logs.
func (f *fakeEthClient) FilterLogs(ctx context.Context, q ethereum.FilterQuery) ([]types.Log, error) {
	return []types.Log{}, nil
}

// HeaderByNumber returns nil.
func (f *fakeEthClient) HeaderByNumber(ctx context.Context, number *big.Int) (*types.Header, error) {
	return nil, nil
}

// SubscribeFilterLogs returns a nil subscription.
func (f *fakeEthClient) SubscribeFilterLogs(ctx context.Context, query ethereum.FilterQuery, ch chan<- types.Log) (ethereum.Subscription, error) {
	return nil, nil
}

// TransactionByHash returns (nil, false, nil).
func (f *fakeEthClient) TransactionByHash(ctx context.Context, txHash common.Hash) (*types.Transaction, bool, error) {
	return nil, false, nil
}

// BlockByHash returns nil.
func (f *fakeEthClient) BlockByHash(ctx context.Context, hash common.Hash) (*types.Block, error) {
	return nil, nil
}

// StorageAt returns the byte representation of f.value.
func (f *fakeEthClient) StorageAt(ctx context.Context, account common.Address, key common.Hash, blockNumber *big.Int) ([]byte, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.value.Bytes(), nil
}
