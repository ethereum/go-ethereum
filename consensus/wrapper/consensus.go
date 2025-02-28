package wrapper

import (
	"math/big"
	"sync"

	"github.com/scroll-tech/go-ethereum/common"
	"github.com/scroll-tech/go-ethereum/consensus"
	"github.com/scroll-tech/go-ethereum/consensus/clique"
	"github.com/scroll-tech/go-ethereum/consensus/system_contract"
	"github.com/scroll-tech/go-ethereum/core/state"
	"github.com/scroll-tech/go-ethereum/core/types"
	"github.com/scroll-tech/go-ethereum/rpc"
)

// UpgradableEngine implements consensus.Engine and acts as a middleware to dispatch
// calls to either Clique or SystemContract consensus.
type UpgradableEngine struct {
	// forkBlock is the block number at which the switchover to SystemContract occurs.
	isUpgraded func(uint64) bool

	// clique is the original Clique consensus engine.
	clique consensus.Engine

	// system is the new SystemContract consensus engine.
	system consensus.Engine
}

// NewUpgradableEngine constructs a new upgradable consensus middleware.
func NewUpgradableEngine(isUpgraded func(uint64) bool, clique consensus.Engine, system consensus.Engine) *UpgradableEngine {
	return &UpgradableEngine{
		isUpgraded: isUpgraded,
		clique:     clique,
		system:     system,
	}
}

// chooseEngine returns the appropriate consensus engine based on the header's number.
func (ue *UpgradableEngine) chooseEngine(header *types.Header) consensus.Engine {
	if ue.isUpgraded(header.Time) {
		return ue.system
	}
	return ue.clique
}

// --------------------
// Methods to implement consensus.Engine

// Author returns the author of the block based on the header.
func (ue *UpgradableEngine) Author(header *types.Header) (common.Address, error) {
	return ue.chooseEngine(header).Author(header)
}

// VerifyHeader checks whether a header conforms to the consensus rules of the engine.
func (ue *UpgradableEngine) VerifyHeader(chain consensus.ChainHeaderReader, header *types.Header, seal bool) error {
	return ue.chooseEngine(header).VerifyHeader(chain, header, seal)
}

// VerifyHeaders verifies a batch of headers concurrently. In our use-case,
// headers can only be all system, all clique, or start with clique and then switch once to system.
func (ue *UpgradableEngine) VerifyHeaders(chain consensus.ChainHeaderReader, headers []*types.Header, seals []bool) (chan<- struct{}, <-chan error) {
	abort := make(chan struct{})
	out := make(chan error)

	// If there are no headers, return a closed error channel.
	if len(headers) == 0 {
		close(out)
		return nil, out
	}

	// Choose engine for the first and last header.
	firstEngine := ue.chooseEngine(headers[0])
	lastEngine := ue.chooseEngine(headers[len(headers)-1])

	// If the first header is system, then all headers must be system.
	if firstEngine == ue.system {
		return firstEngine.VerifyHeaders(chain, headers, seals)
	}

	// If first and last headers are both clique, then all headers are clique.
	if firstEngine == lastEngine {
		return firstEngine.VerifyHeaders(chain, headers, seals)
	}

	// Otherwise, headers start as clique then switch to system.  Since we assume
	// a single switchover, find the first header that uses system.
	splitIndex := 0
	for i, header := range headers {
		if ue.chooseEngine(header) == ue.system {
			splitIndex = i
			break
		}
	}
	// It's expected that splitIndex is > 0.
	cliqueHeaders := headers[:splitIndex]
	cliqueSeals := seals[:splitIndex]
	systemHeaders := headers[splitIndex:]
	systemSeals := seals[splitIndex:]

	// Create a wait group to merge results.
	var wg sync.WaitGroup
	wg.Add(2)

	// Launch concurrent verifications.
	go func() {
		defer wg.Done()
		_, cliqueResults := ue.clique.VerifyHeaders(chain, cliqueHeaders, cliqueSeals)
		for err := range cliqueResults {
			select {
			case <-abort:
				return
			case out <- err:
			}
		}
	}()

	go func() {
		defer wg.Done()
		_, systemResults := ue.system.VerifyHeaders(chain, systemHeaders, systemSeals)
		for err := range systemResults {
			select {
			case <-abort:
				return
			case out <- err:
			}
		}
	}()

	// Close the out channel when both verifications are complete.
	go func() {
		wg.Wait()
		close(out)
	}()

	return abort, out
}

// Prepare prepares a block header for sealing.
func (ue *UpgradableEngine) Prepare(chain consensus.ChainHeaderReader, header *types.Header) error {
	return ue.chooseEngine(header).Prepare(chain, header)
}

// Seal instructs the engine to start sealing a block.
func (ue *UpgradableEngine) Seal(chain consensus.ChainHeaderReader, block *types.Block, results chan<- *types.Block, stop <-chan struct{}) error {
	return ue.chooseEngine(block.Header()).Seal(chain, block, results, stop)
}

// CalcDifficulty calculates the block difficulty if applicable.
func (ue *UpgradableEngine) CalcDifficulty(chain consensus.ChainHeaderReader, time uint64, parent *types.Header) *big.Int {
	return ue.chooseEngine(parent).CalcDifficulty(chain, time, parent)
}

// Finalize finalizes the block, applying any post-transaction rules.
func (ue *UpgradableEngine) Finalize(chain consensus.ChainHeaderReader, header *types.Header, state *state.StateDB, txs []*types.Transaction, uncles []*types.Header) {
	ue.chooseEngine(header).Finalize(chain, header, state, txs, uncles)
}

// FinalizeAndAssemble finalizes and assembles a new block.
func (ue *UpgradableEngine) FinalizeAndAssemble(chain consensus.ChainHeaderReader, header *types.Header, state *state.StateDB, txs []*types.Transaction, uncles []*types.Header, receipts []*types.Receipt) (*types.Block, error) {
	return ue.chooseEngine(header).FinalizeAndAssemble(chain, header, state, txs, uncles, receipts)
}

// VerifyUncles verifies that no uncles are attached to the block.
func (ue *UpgradableEngine) VerifyUncles(chain consensus.ChainReader, block *types.Block) error {
	return ue.chooseEngine(block.Header()).VerifyUncles(chain, block)
}

// APIs returns any RPC APIs exposed by the consensus engine.
func (ue *UpgradableEngine) APIs(chain consensus.ChainHeaderReader) []rpc.API {
	// Determine the current chain head.
	head := chain.CurrentHeader()
	if head == nil {
		// Fallback: return the clique APIs (or an empty slice) if we don't have a header.
		return ue.clique.APIs(chain)
	}

	// Choose engine based on whether the chain head is before or after the fork block.
	if ue.isUpgraded(head.Time) {
		return ue.system.APIs(chain)
	}
	return ue.clique.APIs(chain)
}

// Close terminates the consensus engine.
func (ue *UpgradableEngine) Close() error {
	// Always close both engines.
	if err := ue.clique.Close(); err != nil {
		return err
	}
	return ue.system.Close()
}

// SealHash returns the hash of a block prior to it being sealed.
func (ue *UpgradableEngine) SealHash(header *types.Header) common.Hash {
	return ue.chooseEngine(header).SealHash(header)
}

// Authorize injects a private key into the consensus engine to mint new blocks
// with.
func (ue *UpgradableEngine) Authorize(signer common.Address, signFn clique.SignerFn, signFn2 system_contract.SignerFn) {
	if cliqueEngine, ok := ue.clique.(*clique.Clique); ok {
		cliqueEngine.Authorize(signer, signFn)
	}
	if sysContractEngine, ok := ue.system.(*system_contract.SystemContract); ok {
		sysContractEngine.Authorize(signer, signFn2)
	}
}
