package builder

import (
	"fmt"
	"math/big"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/math"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/params"
	"github.com/google/uuid"
)

// blockchain is the minimum interface to the blockchain
// required to build a block
type blockchain interface {
	core.ChainContext

	// Header returns the current tip of the chain
	CurrentHeader() *types.Header

	// StateAt returns the state at the given root
	StateAt(root common.Hash) (*state.StateDB, error)

	// Config returns the chain config
	Config() *params.ChainConfig
}

type Config struct {
	GasCeil            uint64
	SessionIdleTimeout time.Duration
}

type SessionManager struct {
	sessions      map[string]*builder
	sessionTimers map[string]*time.Timer
	sessionsLock  sync.RWMutex
	blockchain    blockchain
	config        *Config
}

func NewSessionManager(blockchain blockchain, config *Config) *SessionManager {
	if config.GasCeil == 0 {
		config.GasCeil = 1000000000000000000
	}
	if config.SessionIdleTimeout == 0 {
		config.SessionIdleTimeout = 5 * time.Second
	}

	s := &SessionManager{
		sessions:      make(map[string]*builder),
		sessionTimers: make(map[string]*time.Timer),
		blockchain:    blockchain,
		config:        config,
	}
	return s
}

// NewSession creates a new builder session and returns the session id
func (s *SessionManager) NewSession() (string, error) {
	s.sessionsLock.Lock()
	defer s.sessionsLock.Unlock()

	parent := s.blockchain.CurrentHeader()

	chainConfig := s.blockchain.Config()

	header := &types.Header{
		ParentHash: parent.Hash(),
		Number:     new(big.Int).Add(parent.Number, common.Big1),
		GasLimit:   core.CalcGasLimit(parent.GasLimit, s.config.GasCeil),
		Time:       1000,             // TODO: fix this
		Coinbase:   common.Address{}, // TODO: fix this
		Difficulty: big.NewInt(1),
	}

	// Set baseFee and GasLimit if we are on an EIP-1559 chain
	if chainConfig.IsLondon(header.Number) {
		header.BaseFee = CalcBaseFee(chainConfig, parent)
		if !chainConfig.IsLondon(parent.Number) {
			parentGasLimit := parent.GasLimit * chainConfig.ElasticityMultiplier()
			header.GasLimit = core.CalcGasLimit(parentGasLimit, s.config.GasCeil)
		}
	}

	stateRef, err := s.blockchain.StateAt(parent.Root)
	if err != nil {
		return "", err
	}

	cfg := &builderConfig{
		preState: stateRef,
		header:   header,
		config:   s.blockchain.Config(),
		context:  s.blockchain,
	}

	id := uuid.New().String()[:7]
	s.sessions[id] = newBuilder(cfg)

	// start session timer
	s.sessionTimers[id] = time.AfterFunc(s.config.SessionIdleTimeout, func() {
		s.sessionsLock.Lock()
		defer s.sessionsLock.Unlock()

		delete(s.sessions, id)
		delete(s.sessionTimers, id)
	})

	return id, nil
}

func (s *SessionManager) getSession(sessionId string) (*builder, error) {
	s.sessionsLock.RLock()
	defer s.sessionsLock.RUnlock()

	session, ok := s.sessions[sessionId]
	if !ok {
		return nil, fmt.Errorf("session %s not found", sessionId)
	}

	// reset session timer
	s.sessionTimers[sessionId].Reset(s.config.SessionIdleTimeout)

	return session, nil
}

func (s *SessionManager) AddTransaction(sessionId string, tx *types.Transaction) (*types.SimulateTransactionResult, error) {
	builder, err := s.getSession(sessionId)
	if err != nil {
		return nil, err
	}
	return builder.AddTransaction(tx)
}

// CalcBaseFee calculates the basefee of the header.
func CalcBaseFee(config *params.ChainConfig, parent *types.Header) *big.Int {
	// If the current block is the first EIP-1559 block, return the InitialBaseFee.
	if !config.IsLondon(parent.Number) {
		return new(big.Int).SetUint64(params.InitialBaseFee)
	}

	parentGasTarget := parent.GasLimit / config.ElasticityMultiplier()
	// If the parent gasUsed is the same as the target, the baseFee remains unchanged.
	if parent.GasUsed == parentGasTarget {
		return new(big.Int).Set(parent.BaseFee)
	}

	var (
		num   = new(big.Int)
		denom = new(big.Int)
	)

	if parent.GasUsed > parentGasTarget {
		// If the parent block used more gas than its target, the baseFee should increase.
		// max(1, parentBaseFee * gasUsedDelta / parentGasTarget / baseFeeChangeDenominator)
		num.SetUint64(parent.GasUsed - parentGasTarget)
		num.Mul(num, parent.BaseFee)
		num.Div(num, denom.SetUint64(parentGasTarget))
		num.Div(num, denom.SetUint64(config.BaseFeeChangeDenominator()))
		baseFeeDelta := math.BigMax(num, common.Big1)

		return num.Add(parent.BaseFee, baseFeeDelta)
	} else {
		// Otherwise if the parent block used less gas than its target, the baseFee should decrease.
		// max(0, parentBaseFee * gasUsedDelta / parentGasTarget / baseFeeChangeDenominator)
		num.SetUint64(parentGasTarget - parent.GasUsed)
		num.Mul(num, parent.BaseFee)
		num.Div(num, denom.SetUint64(parentGasTarget))
		num.Div(num, denom.SetUint64(config.BaseFeeChangeDenominator()))
		baseFee := num.Sub(parent.BaseFee, num)

		return math.BigMax(baseFee, common.Big0)
	}
}
