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
		log.Error("failed to fetch signer address from L1", "err", err)
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
		return fmt.Errorf("retrieved empty signer address from L1 System Contract: contract=%s, slot=%s", s.config.SystemContractAddress.Hex(), s.config.SystemContractSlot.Hex())
	}

	log.Debug("Read address from system contract", "address", bAddress.Hex())

	s.lock.Lock()
	defer s.lock.Unlock()

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
