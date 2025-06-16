package system_contract

import (
	"context"
	"fmt"
	"math/big"
	"sync"
	"time"

	"github.com/scroll-tech/go-ethereum"
	"github.com/scroll-tech/go-ethereum/common"
	"github.com/scroll-tech/go-ethereum/core/types"
	"github.com/scroll-tech/go-ethereum/log"
	"github.com/scroll-tech/go-ethereum/params"
	"github.com/scroll-tech/go-ethereum/rollup/sync_service"
)

const (
	defaultSyncInterval = 10 * time.Second
)

// SystemContract is the proof-of-authority consensus engine that fetches
// the authorized signer from the SystemConfig contract, starting from EuclidV2.
type SystemContract struct {
	config *params.SystemContractConfig // Consensus engine configuration parameters
	client sync_service.EthClient       // RPC client to fetch info from L1

	signerAddressL1 common.Address // Address of the signer stored in L1 System Contract

	signer common.Address // Ethereum address of the signing key
	signFn SignerFn       // Signer function to authorize hashes with
	lock   sync.RWMutex   // Protects the signer and proposals fields

	ctx    context.Context
	cancel context.CancelFunc
}

// New creates a SystemContract consensus engine with the initial
// authorized signer fetched from L1 (if available).
func New(ctx context.Context, config *params.SystemContractConfig, client sync_service.EthClient) *SystemContract {
	log.Info("Initializing system_contract consensus engine", "config", config)

	ctx, cancel := context.WithCancel(ctx)

	s := &SystemContract{
		config: config,
		client: client,

		ctx:    ctx,
		cancel: cancel,
	}

	if err := s.fetchAddressFromL1(); err != nil {
		log.Error("Failed to fetch signer address from L1", "err", err)
	}

	return s
}

// Authorize injects a private key into the consensus engine to mint new blocks
// with.
func (s *SystemContract) Authorize(signer common.Address, signFn SignerFn) {
	log.Info("Authorizing system contract consensus", "signer", signer.Hex())
	s.lock.Lock()
	defer s.lock.Unlock()

	s.signer = signer
	s.signFn = signFn
}

func (s *SystemContract) Start() {
	go func() {
		log.Info("starting SystemContract")
		syncTicker := time.NewTicker(defaultSyncInterval)
		defer syncTicker.Stop()
		for {
			select {
			case <-s.ctx.Done():
				return
			default:
			}
			select {
			case <-s.ctx.Done():
				return
			case <-syncTicker.C:
				if err := s.fetchAddressFromL1(); err != nil {
					log.Error("Failed to fetch signer address from L1", "err", err)
				}
			}
		}
	}()
}

func (s *SystemContract) fetchAddressFromL1() error {
	address, err := s.client.StorageAt(s.ctx, s.config.SystemContractAddress, s.config.SystemContractSlot, nil)
	if err != nil {
		return fmt.Errorf("failed to get signer address from L1 System Contract: %w", err)
	}
	bAddress := common.BytesToAddress(address)

	s.lock.Lock()
	defer s.lock.Unlock()

	// Validate the address is not empty
	if bAddress == (common.Address{}) {
		log.Debug("Retrieved empty signer address from L1 System Contract", "contract", s.config.SystemContractAddress.Hex(), "slot", s.config.SystemContractSlot.Hex())

		// Not initialized yet -- we don't consider this an error
		if s.signerAddressL1 == (common.Address{}) {
			log.Warn("System Contract signer address not initialized")
			return nil
		}

		return fmt.Errorf("retrieved empty signer address from L1 System Contract")
	}

	log.Debug("Read address from system contract", "address", bAddress.Hex())

	if s.signerAddressL1 != bAddress {
		s.signerAddressL1 = bAddress
		log.Info("Updated new signer from L1 system contract", "signer", bAddress.Hex())
	}

	return nil
}

// Close implements consensus.Engine.
func (s *SystemContract) Close() error {
	s.cancel()
	return nil
}

func (s *SystemContract) currentSignerAddressL1() common.Address {
	s.lock.RLock()
	defer s.lock.RUnlock()

	return s.signerAddressL1
}

func (s *SystemContract) localSignerAddress() common.Address {
	s.lock.RLock()
	defer s.lock.RUnlock()

	return s.signer
}

// FakeEthClient implements a minimal version of sync_service.EthClient for testing purposes.
type FakeEthClient struct {
	mu sync.Mutex
	// Value is the fixed Value to return from StorageAt.
	// We'll assume StorageAt returns a 32-byte Value representing an Ethereum address.
	Value common.Address
}

// BlockNumber returns 0.
func (f *FakeEthClient) BlockNumber(ctx context.Context) (uint64, error) {
	return 0, nil
}

// ChainID returns a zero-value chain ID.
func (f *FakeEthClient) ChainID(ctx context.Context) (*big.Int, error) {
	return big.NewInt(0), nil
}

// FilterLogs returns an empty slice of logs.
func (f *FakeEthClient) FilterLogs(ctx context.Context, q ethereum.FilterQuery) ([]types.Log, error) {
	return []types.Log{}, nil
}

// HeaderByNumber returns nil.
func (f *FakeEthClient) HeaderByNumber(ctx context.Context, number *big.Int) (*types.Header, error) {
	return nil, nil
}

// SubscribeFilterLogs returns a nil subscription.
func (f *FakeEthClient) SubscribeFilterLogs(ctx context.Context, query ethereum.FilterQuery, ch chan<- types.Log) (ethereum.Subscription, error) {
	return nil, nil
}

// TransactionByHash returns (nil, false, nil).
func (f *FakeEthClient) TransactionByHash(ctx context.Context, txHash common.Hash) (*types.Transaction, bool, error) {
	return nil, false, nil
}

// BlockByHash returns nil.
func (f *FakeEthClient) BlockByHash(ctx context.Context, hash common.Hash) (*types.Block, error) {
	return nil, nil
}

// StorageAt returns the byte representation of f.value.
func (f *FakeEthClient) StorageAt(ctx context.Context, account common.Address, key common.Hash, blockNumber *big.Int) ([]byte, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.Value.Bytes(), nil
}
