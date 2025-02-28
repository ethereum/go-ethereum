package system_contract

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/scroll-tech/go-ethereum/common"
	"github.com/scroll-tech/go-ethereum/log"
	"github.com/scroll-tech/go-ethereum/params"
	"github.com/scroll-tech/go-ethereum/rollup/sync_service"
)

const (
	defaultSyncInterval = 10 * time.Second
)

// SystemContract
type SystemContract struct {
	config *params.SystemContractConfig // Consensus engine configuration parameters
	client sync_service.EthClient

	signerAddressL1 common.Address // Address of the signer stored in L1 System Contract

	signer common.Address // Ethereum address of the signing key
	signFn SignerFn       // Signer function to authorize hashes with
	lock   sync.RWMutex   // Protects the signer and proposals fields

	ctx    context.Context
	cancel context.CancelFunc
}

// New creates a SystemContract consensus engine with the initial
// signers set to the ones provided by the user.
func New(ctx context.Context, config *params.SystemContractConfig, client sync_service.EthClient) *SystemContract {
	ctx, cancel := context.WithCancel(ctx)

	s := &SystemContract{
		config: config,
		client: client,

		ctx:    ctx,
		cancel: cancel,
	}

	if err := s.fetchAddressFromL1(); err != nil {
		log.Error("failed to fetch signer address from L1", "err", err)
	}
	return s
}

// Authorize injects a private key into the consensus engine to mint new blocks
// with.
func (s *SystemContract) Authorize(signer common.Address, signFn SignerFn) {
	log.Info("Authorizing system contract", "signer", signer.Hex())
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
					log.Error("failed to fetch signer address from L1", "err", err)
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

	// Validate the address is not empty
	if bAddress == (common.Address{}) {
		return fmt.Errorf("retrieved empty signer address from L1 System Contract")
	}

	log.Debug("Read address from system contract", "address", bAddress.Hex())

	s.lock.RLock()
	addressChanged := s.signerAddressL1 != bAddress
	s.lock.RUnlock()

	if addressChanged {
		s.lock.Lock()
		s.signerAddressL1 = bAddress
		s.lock.Unlock()
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
