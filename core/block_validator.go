// Copyright 2015 The go-ethereum Authors
// This file is part of the go-ethereum library.
//
// The go-ethereum library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The go-ethereum library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the go-ethereum library. If not, see <http://www.gnu.org/licenses/>.

package core

import (
	"fmt"
	"math/big"
	"time"

	"github.com/ubiq/go-ubiq/common"
	"github.com/ubiq/go-ubiq/core/state"
	"github.com/ubiq/go-ubiq/core/types"
	"github.com/ubiq/go-ubiq/logger"
	"github.com/ubiq/go-ubiq/logger/glog"
	"github.com/ubiq/go-ubiq/params"
	"github.com/ubiq/go-ubiq/pow"
	"gopkg.in/fatih/set.v0"
)

var (
	big88               = big.NewInt(88)
	bigMinus99          = big.NewInt(-99)
	nPowAveragingWindow = big.NewInt(21)
	nPowMaxAdjustDown   = big.NewInt(16) // 16% adjustment down
	nPowMaxAdjustUp     = big.NewInt(8)  // 8% adjustment up

	diffChangeBlock       = big.NewInt(4088)
	nPowAveragingWindow88 = big.NewInt(88)
	nPowMaxAdjustDown2    = big.NewInt(3) // 3% adjustment down
	nPowMaxAdjustUp2      = big.NewInt(2) // 2% adjustment up

	// Flux
	fluxChangeBlock       = big.NewInt(8000)
	nPowMaxAdjustDownFlux = big.NewInt(5) // 0.5% adjustment down
	nPowMaxAdjustUpFlux   = big.NewInt(3) // 0.3% adjustment up
	nPowDampFlux          = big.NewInt(1) // 0.1%
)

func AveragingWindowTimespan() *big.Int {
	x := new(big.Int)
	return x.Mul(nPowAveragingWindow, big88)
}

func MinActualTimespan() *big.Int {
	// (AveragingWindowTimespan() * (100 - nPowMaxAdjustUp  )) / 100
	x := new(big.Int)
	y := new(big.Int)
	z := new(big.Int)
	x.Sub(big.NewInt(100), nPowMaxAdjustUp)
	y.Mul(AveragingWindowTimespan(), x)
	z.Div(y, big.NewInt(100))
	return z
}

func MaxActualTimespan() *big.Int {
	// (AveragingWindowTimespan() * (100 + nPowMaxAdjustDown)) / 100
	x := new(big.Int)
	y := new(big.Int)
	z := new(big.Int)
	x.Add(big.NewInt(100), nPowMaxAdjustDown)
	y.Mul(AveragingWindowTimespan(), x)
	z.Div(y, big.NewInt(100))
	return z
}

func AveragingWindowTimespan88() *big.Int {
	x := new(big.Int)
	return x.Mul(nPowAveragingWindow88, big88)
}

func MinActualTimespan2() *big.Int {
	x := new(big.Int)
	y := new(big.Int)
	z := new(big.Int)
	x.Sub(big.NewInt(100), nPowMaxAdjustUp2)
	y.Mul(AveragingWindowTimespan88(), x)
	z.Div(y, big.NewInt(100))
	return z
}

func MaxActualTimespan2() *big.Int {
	x := new(big.Int)
	y := new(big.Int)
	z := new(big.Int)
	x.Add(big.NewInt(100), nPowMaxAdjustDown2)
	y.Mul(AveragingWindowTimespan88(), x)
	z.Div(y, big.NewInt(100))
	return z
}

func MinActualTimespanFlux(dampen bool) *big.Int {
	x := new(big.Int)
	y := new(big.Int)
	z := new(big.Int)
	if dampen {
		x.Sub(big.NewInt(1000), nPowDampFlux)
		y.Mul(AveragingWindowTimespan88(), x)
		z.Div(y, big.NewInt(1000))
	} else {
		x.Sub(big.NewInt(1000), nPowMaxAdjustUpFlux)
		y.Mul(AveragingWindowTimespan88(), x)
		z.Div(y, big.NewInt(1000))
	}
	return z
}

func MaxActualTimespanFlux(dampen bool) *big.Int {
	x := new(big.Int)
	y := new(big.Int)
	z := new(big.Int)
	if dampen {
		x.Add(big.NewInt(1000), nPowDampFlux)
		y.Mul(AveragingWindowTimespan88(), x)
		z.Div(y, big.NewInt(1000))
	} else {
		x.Add(big.NewInt(1000), nPowMaxAdjustDownFlux)
		y.Mul(AveragingWindowTimespan88(), x)
		z.Div(y, big.NewInt(1000))
	}
	return z
}

// BlockValidator is responsible for validating block headers, uncles and
// processed state.
//
// BlockValidator implements Validator.
type BlockValidator struct {
	config *params.ChainConfig // Chain configuration options
	bc     *BlockChain         // Canonical block chain
	Pow    pow.PoW             // Proof of work used for validating
}

// NewBlockValidator returns a new block validator which is safe for re-use
func NewBlockValidator(config *params.ChainConfig, blockchain *BlockChain, pow pow.PoW) *BlockValidator {
	validator := &BlockValidator{
		config: config,
		Pow:    pow,
		bc:     blockchain,
	}
	return validator
}

// ValidateBlock validates the given block's header and uncles and verifies the
// the block header's transaction and uncle roots.
//
// ValidateBlock does not validate the header's pow. The pow work validated
// separately so we can process them in parallel.
//
// ValidateBlock also validates and makes sure that any previous state (or present)
// state that might or might not be present is checked to make sure that fast
// sync has done it's job proper. This prevents the block validator from accepting
// false positives where a header is present but the state is not.
func (v *BlockValidator) ValidateBlock(block *types.Block) error {
	if v.bc.HasBlock(block.Hash()) {
		if _, err := state.New(block.Root(), v.bc.chainDb); err == nil {
			return &KnownBlockError{block.Number(), block.Hash()}
		}
	}
	parent := v.bc.GetBlock(block.ParentHash(), block.NumberU64()-1)
	if parent == nil {
		return ParentError(block.ParentHash())
	}
	if _, err := state.New(parent.Root(), v.bc.chainDb); err != nil {
		return ParentError(block.ParentHash())
	}

	header := block.Header()
	// validate the block header
	if err := ValidateHeader(v.config, v.Pow, header, parent.Header(), false, false, v.bc); err != nil {
		return err
	}
	// verify the uncles are correctly rewarded
	if err := v.VerifyUncles(block, parent); err != nil {
		return err
	}

	// Verify UncleHash before running other uncle validations
	unclesSha := types.CalcUncleHash(block.Uncles())
	if unclesSha != header.UncleHash {
		return fmt.Errorf("invalid uncles root hash (remote: %x local: %x)", header.UncleHash, unclesSha)
	}

	// The transactions Trie's root (R = (Tr [[i, RLP(T1)], [i, RLP(T2)], ... [n, RLP(Tn)]]))
	// can be used by light clients to make sure they've received the correct Txs
	txSha := types.DeriveSha(block.Transactions())
	if txSha != header.TxHash {
		return fmt.Errorf("invalid transaction root hash (remote: %x local: %x)", header.TxHash, txSha)
	}

	return nil
}

// ValidateState validates the various changes that happen after a state
// transition, such as amount of used gas, the receipt roots and the state root
// itself. ValidateState returns a database batch if the validation was a success
// otherwise nil and an error is returned.
func (v *BlockValidator) ValidateState(block, parent *types.Block, statedb *state.StateDB, receipts types.Receipts, usedGas *big.Int) (err error) {
	header := block.Header()
	if block.GasUsed().Cmp(usedGas) != 0 {
		return ValidationError(fmt.Sprintf("invalid gas used (remote: %v local: %v)", block.GasUsed(), usedGas))
	}
	// Validate the received block's bloom with the one derived from the generated receipts.
	// For valid blocks this should always validate to true.
	rbloom := types.CreateBloom(receipts)
	if rbloom != header.Bloom {
		return fmt.Errorf("invalid bloom (remote: %x  local: %x)", header.Bloom, rbloom)
	}
	// Tre receipt Trie's root (R = (Tr [[H1, R1], ... [Hn, R1]]))
	receiptSha := types.DeriveSha(receipts)
	if receiptSha != header.ReceiptHash {
		return fmt.Errorf("invalid receipt root hash (remote: %x local: %x)", header.ReceiptHash, receiptSha)
	}
	// Validate the state root against the received state root and throw
	// an error if they don't match.
	if root := statedb.IntermediateRoot(v.config.IsEIP158(header.Number)); header.Root != root {
		return fmt.Errorf("invalid merkle root (remote: %x local: %x)", header.Root, root)
	}
	return nil
}

// VerifyUncles verifies the given block's uncles and applies the Ethereum
// consensus rules to the various block headers included; it will return an
// error if any of the included uncle headers were invalid. It returns an error
// if the validation failed.
func (v *BlockValidator) VerifyUncles(block, parent *types.Block) error {
	// validate that there are at most 2 uncles included in this block
	if len(block.Uncles()) > 2 {
		return ValidationError("Block can only contain maximum 2 uncles (contained %v)", len(block.Uncles()))
	}

	uncles := set.New()
	ancestors := make(map[common.Hash]*types.Block)
	for _, ancestor := range v.bc.GetBlocksFromHash(block.ParentHash(), 7) {
		ancestors[ancestor.Hash()] = ancestor
		// Include ancestors uncles in the uncle set. Uncles must be unique.
		for _, uncle := range ancestor.Uncles() {
			uncles.Add(uncle.Hash())
		}
	}
	ancestors[block.Hash()] = block
	uncles.Add(block.Hash())

	for i, uncle := range block.Uncles() {
		hash := uncle.Hash()
		if uncles.Has(hash) {
			// Error not unique
			return UncleError("uncle[%d](%x) not unique", i, hash[:4])
		}
		uncles.Add(hash)

		if ancestors[hash] != nil {
			branch := fmt.Sprintf("  O - %x\n  |\n", block.Hash())
			for h := range ancestors {
				branch += fmt.Sprintf("  O - %x\n  |\n", h)
			}
			glog.Infoln(branch)
			return UncleError("uncle[%d](%x) is ancestor", i, hash[:4])
		}

		if ancestors[uncle.ParentHash] == nil || uncle.ParentHash == parent.Hash() {
			return UncleError("uncle[%d](%x)'s parent is not ancestor (%x)", i, hash[:4], uncle.ParentHash[0:4])
		}

		if err := ValidateHeader(v.config, v.Pow, uncle, ancestors[uncle.ParentHash].Header(), true, true, v.bc); err != nil {
			return ValidationError(fmt.Sprintf("uncle[%d](%x) header invalid: %v", i, hash[:4], err))
		}
	}

	return nil
}

// ValidateHeader validates the given header and, depending on the pow arg,
// checks the proof of work of the given header. Returns an error if the
// validation failed.
func (v *BlockValidator) ValidateHeader(header, parent *types.Header, checkPow bool) error {
	// Short circuit if the parent is missing.
	if parent == nil {
		return ParentError(header.ParentHash)
	}
	// Short circuit if the header's already known or its parent is missing
	if v.bc.HasHeader(header.Hash()) {
		return nil
	}
	return ValidateHeader(v.config, v.Pow, header, parent, checkPow, false, v.bc)
}

// Validates a header. Returns an error if the header is invalid.
//
// See YP section 4.3.4. "Block Header Validity"
func ValidateHeader(config *params.ChainConfig, pow pow.PoW, header *types.Header, parent *types.Header, checkPow, uncle bool, bc *BlockChain) error {
	if big.NewInt(int64(len(header.Extra))).Cmp(params.MaximumExtraDataSize) == 1 {
		return fmt.Errorf("Header extra data too long (%d)", len(header.Extra))
	}

	if uncle {
		if header.Time.Cmp(common.MaxBig) == 1 {
			return BlockTSTooBigErr
		}
	} else {
		if header.Time.Cmp(big.NewInt(time.Now().Unix())) == 1 {
			return BlockFutureErr
		}
	}
	if header.Time.Cmp(parent.Time) != 1 {
		return BlockEqualTSErr
	}

	expd := CalcDifficulty(config, header.Time.Uint64(), parent.Time.Uint64(), parent.Number, parent.Difficulty, bc)
	if expd.Cmp(header.Difficulty) != 0 {
		return fmt.Errorf("Difficulty check failed for header (remote: %v local: %v)", header.Difficulty, expd)
	}

	a := new(big.Int).Set(parent.GasLimit)
	a = a.Sub(a, header.GasLimit)
	a.Abs(a)
	b := new(big.Int).Set(parent.GasLimit)
	b = b.Div(b, params.GasLimitBoundDivisor)
	if !(a.Cmp(b) < 0) || (header.GasLimit.Cmp(params.MinGasLimit) == -1) {
		return fmt.Errorf("GasLimit check failed for header (remote: %v local_max: %v)", header.GasLimit, b)
	}

	num := new(big.Int).Set(parent.Number)
	num.Sub(header.Number, num)
	if num.Cmp(big.NewInt(1)) != 0 {
		return BlockNumberErr
	}

	if checkPow {
		// Verify the nonce of the header. Return an error if it's not valid
		if !pow.Verify(types.NewBlockWithHeader(header)) {
			return &BlockNonceErr{header.Number, header.Hash(), header.Nonce.Uint64()}
		}
	}
	if !uncle && config.EIP150Block != nil && config.EIP150Block.Cmp(header.Number) == 0 {
		if config.EIP150Hash != (common.Hash{}) && config.EIP150Hash != header.Hash() {
			return ValidationError("Homestead gas reprice fork hash mismatch: have 0x%x, want 0x%x", header.Hash(), config.EIP150Hash)
		}
	}
	return nil
}

func ValidateHeaderHeaderChain(config *params.ChainConfig, pow pow.PoW, header *types.Header, parent *types.Header, checkPow, uncle bool, hc *HeaderChain) error {
	if big.NewInt(int64(len(header.Extra))).Cmp(params.MaximumExtraDataSize) == 1 {
		return fmt.Errorf("Header extra data too long (%d)", len(header.Extra))
	}

	if uncle {
		if header.Time.Cmp(common.MaxBig) == 1 {
			return BlockTSTooBigErr
		}
	} else {
		if header.Time.Cmp(big.NewInt(time.Now().Unix())) == 1 {
			return BlockFutureErr
		}
	}
	if header.Time.Cmp(parent.Time) != 1 {
		return BlockEqualTSErr
	}

	expd := CalcDifficultyHeaderChain(config, header.Time.Uint64(), parent.Time.Uint64(), parent.Number, parent.Difficulty, hc)
	if expd.Cmp(header.Difficulty) != 0 {
		return fmt.Errorf("Difficulty check failed for header (remote: %v local: %v)", header.Difficulty, expd)
	}

	a := new(big.Int).Set(parent.GasLimit)
	a = a.Sub(a, header.GasLimit)
	a.Abs(a)
	b := new(big.Int).Set(parent.GasLimit)
	b = b.Div(b, params.GasLimitBoundDivisor)
	if !(a.Cmp(b) < 0) || (header.GasLimit.Cmp(params.MinGasLimit) == -1) {
		return fmt.Errorf("GasLimit check failed for header (remote: %v local_max: %v)", header.GasLimit, b)
	}

	num := new(big.Int).Set(parent.Number)
	num.Sub(header.Number, num)
	if num.Cmp(big.NewInt(1)) != 0 {
		return BlockNumberErr
	}

	if checkPow {
		// Verify the nonce of the header. Return an error if it's not valid
		if !pow.Verify(types.NewBlockWithHeader(header)) {
			return &BlockNonceErr{header.Number, header.Hash(), header.Nonce.Uint64()}
		}
	}
	if !uncle && config.EIP150Block != nil && config.EIP150Block.Cmp(header.Number) == 0 {
		if config.EIP150Hash != (common.Hash{}) && config.EIP150Hash != header.Hash() {
			return ValidationError("Homestead gas reprice fork hash mismatch: have 0x%x, want 0x%x", header.Hash(), config.EIP150Hash)
		}
	}
	return nil
}

func CalcDifficulty(config *params.ChainConfig, time, parentTime uint64, parentNumber, parentDiff *big.Int, bc *BlockChain) *big.Int {
	if parentNumber.Cmp(diffChangeBlock) < 0 {
		return CalcDifficultyOrig(time, parentTime, parentNumber, parentDiff, bc)
	}
	if parentNumber.Cmp(fluxChangeBlock) < 0 {
		return CalcDifficulty2(time, parentTime, parentNumber, parentDiff, bc)
	} else {
		return FluxDifficulty(time, parentTime, parentNumber, parentDiff, bc)
	}
}

// CalcDifficulty is the difficulty adjustment algorithm. It returns
// the difficulty that a new block should have when created at time
// given the parent block's time and difficulty.
// Rewritten to be based on Digibyte's Digishield v3 retargeting
func CalcDifficultyOrig(time, parentTime uint64, parentNumber, parentDiff *big.Int, bc *BlockChain) *big.Int {
	// holds intermediate values to make the algo easier to read & audit
	x := new(big.Int)
	nFirstBlock := new(big.Int)
	nFirstBlock.Sub(parentNumber, nPowAveragingWindow)

	glog.V(logger.Debug).Infof("CalcDifficulty parentNumber: %v parentDiff: %v\n", parentNumber, parentDiff)

	// Check we have enough blocks
	if parentNumber.Cmp(nPowAveragingWindow) < 1 {
		glog.V(logger.Debug).Infof("CalcDifficulty: parentNumber(%+x) < nPowAveragingWindow(%+x)\n", parentNumber, nPowAveragingWindow)
		x.Set(parentDiff)
		return x
	}

	// Limit adjustment step
	// Use medians to prevent time-warp attacks
	// nActualTimespan := nLastBlockTime - nFirstBlockTime
	nLastBlockTime := bc.CalcPastMedianTime(parentNumber.Uint64())
	nFirstBlockTime := bc.CalcPastMedianTime(nFirstBlock.Uint64())
	nActualTimespan := new(big.Int)
	nActualTimespan.Sub(nLastBlockTime, nFirstBlockTime)
	glog.V(logger.Debug).Infof("CalcDifficulty nActualTimespan = %v before dampening\n", nActualTimespan)

	// nActualTimespan = AveragingWindowTimespan() + (nActualTimespan-AveragingWindowTimespan())/4
	y := new(big.Int)
	y.Sub(nActualTimespan, AveragingWindowTimespan())
	y.Div(y, big.NewInt(4))
	nActualTimespan.Add(y, AveragingWindowTimespan())
	glog.V(logger.Debug).Infof("CalcDifficulty nActualTimespan = %v before bounds\n", nActualTimespan)

	if nActualTimespan.Cmp(MinActualTimespan()) < 0 {
		nActualTimespan.Set(MinActualTimespan())
		glog.V(logger.Debug).Infoln("CalcDifficulty Minimum Timespan set")
	} else if nActualTimespan.Cmp(MaxActualTimespan()) > 0 {
		nActualTimespan.Set(MaxActualTimespan())
		glog.V(logger.Debug).Infoln("CalcDifficulty Maximum Timespan set")
	}

	glog.V(logger.Debug).Infof("CalcDifficulty nActualTimespan = %v final\n", nActualTimespan)

	// Retarget
	x.Mul(parentDiff, AveragingWindowTimespan())
	glog.V(logger.Debug).Infoln("CalcDifficulty parentDiff * AveragingWindowTimespan:", x)

	x.Div(x, nActualTimespan)
	glog.V(logger.Debug).Infoln("CalcDifficulty x / nActualTimespan:", x)

	// minimum difficulty can ever be (before exponential factor)
	if x.Cmp(params.MinimumDifficulty) < 0 {
		x.Set(params.MinimumDifficulty)
	}

	return x
}

func CalcDifficulty2(time, parentTime uint64, parentNumber, parentDiff *big.Int, bc *BlockChain) *big.Int {
	x := new(big.Int)
	nFirstBlock := new(big.Int)
	nFirstBlock.Sub(parentNumber, nPowAveragingWindow88)

	glog.V(logger.Debug).Infof("CalcDifficulty2 parentNumber: %v parentDiff: %v\n", parentNumber, parentDiff)

	nLastBlockTime := bc.CalcPastMedianTime(parentNumber.Uint64())
	nFirstBlockTime := bc.CalcPastMedianTime(nFirstBlock.Uint64())
	nActualTimespan := new(big.Int)
	nActualTimespan.Sub(nLastBlockTime, nFirstBlockTime)

	y := new(big.Int)
	y.Sub(nActualTimespan, AveragingWindowTimespan88())
	y.Div(y, big.NewInt(4))
	nActualTimespan.Add(y, AveragingWindowTimespan88())

	if nActualTimespan.Cmp(MinActualTimespan2()) < 0 {
		nActualTimespan.Set(MinActualTimespan2())
	} else if nActualTimespan.Cmp(MaxActualTimespan2()) > 0 {
		nActualTimespan.Set(MaxActualTimespan2())
	}

	x.Mul(parentDiff, AveragingWindowTimespan88())

	x.Div(x, nActualTimespan)

	if x.Cmp(params.MinimumDifficulty) < 0 {
		x.Set(params.MinimumDifficulty)
	}

	return x
}

func FluxDifficulty(time, parentTime uint64, parentNumber, parentDiff *big.Int, bc *BlockChain) *big.Int {
	x := new(big.Int)
	nFirstBlock := new(big.Int)
	nFirstBlock.Sub(parentNumber, nPowAveragingWindow88)

	diffTime := new(big.Int)
	diffTime.Sub(big.NewInt(int64(time)), big.NewInt(int64(parentTime)))

	nLastBlockTime := bc.CalcPastMedianTime(parentNumber.Uint64())
	nFirstBlockTime := bc.CalcPastMedianTime(nFirstBlock.Uint64())
	nActualTimespan := new(big.Int)
	nActualTimespan.Sub(nLastBlockTime, nFirstBlockTime)

	y := new(big.Int)
	y.Sub(nActualTimespan, AveragingWindowTimespan88())
	y.Div(y, big.NewInt(4))
	nActualTimespan.Add(y, AveragingWindowTimespan88())

	if nActualTimespan.Cmp(MinActualTimespanFlux(false)) < 0 {
		doubleBig88 := new(big.Int)
		doubleBig88.Mul(big88, big.NewInt(2))
		if diffTime.Cmp(doubleBig88) > 0 {
			nActualTimespan.Set(MinActualTimespanFlux(true))
		} else {
			nActualTimespan.Set(MinActualTimespanFlux(false))
		}
	} else if nActualTimespan.Cmp(MaxActualTimespanFlux(false)) > 0 {
		halfBig88 := new(big.Int)
		halfBig88.Div(big88, big.NewInt(2))
		if diffTime.Cmp(halfBig88) < 0 {
			nActualTimespan.Set(MaxActualTimespanFlux(true))
		} else {
			nActualTimespan.Set(MaxActualTimespanFlux(false))
		}
	}

	x.Mul(parentDiff, AveragingWindowTimespan88())

	x.Div(x, nActualTimespan)

	if x.Cmp(params.MinimumDifficulty) < 0 {
		x.Set(params.MinimumDifficulty)
	}

	return x
}

func CalcDifficultyHeaderChain(config *params.ChainConfig, time, parentTime uint64, parentNumber, parentDiff *big.Int, hc *HeaderChain) *big.Int {
	if parentNumber.Cmp(diffChangeBlock) < 0 {
		return CalcDifficultyHeaderChainOrig(time, parentTime, parentNumber, parentDiff, hc)
	}
	if parentNumber.Cmp(fluxChangeBlock) < 0 {
		return CalcDifficultyHeaderChain2(time, parentTime, parentNumber, parentDiff, hc)
	} else {
		return FluxDifficultyHeaderChain(time, parentTime, parentNumber, parentDiff, hc)
	}
}

func CalcDifficultyHeaderChainOrig(time, parentTime uint64, parentNumber, parentDiff *big.Int, hc *HeaderChain) *big.Int {
	// holds intermediate values to make the algo easier to read & audit
	x := new(big.Int)
	nFirstBlock := new(big.Int)
	nFirstBlock.Sub(parentNumber, nPowAveragingWindow)

	// Check we have enough blocks
	if parentNumber.Cmp(nPowAveragingWindow) < 1 {
		x.Set(parentDiff)
		return x
	}

	nLastBlockTime := hc.CalcPastMedianTime(parentNumber.Uint64())
	nFirstBlockTime := hc.CalcPastMedianTime(nFirstBlock.Uint64())
	nActualTimespan := new(big.Int)
	nActualTimespan.Sub(nLastBlockTime, nFirstBlockTime)

	y := new(big.Int)
	y.Sub(nActualTimespan, AveragingWindowTimespan())
	y.Div(y, big.NewInt(4))
	nActualTimespan.Add(y, AveragingWindowTimespan())

	if nActualTimespan.Cmp(MinActualTimespan()) < 0 {
		nActualTimespan.Set(MinActualTimespan())
	} else if nActualTimespan.Cmp(MaxActualTimespan()) > 0 {
		nActualTimespan.Set(MaxActualTimespan())
	}

	// Retarget
	x.Mul(parentDiff, AveragingWindowTimespan())
	x.Div(x, nActualTimespan)

	// minimum difficulty can ever be (before exponential factor)
	if x.Cmp(params.MinimumDifficulty) < 0 {
		x.Set(params.MinimumDifficulty)
	}

	return x
}

func CalcDifficultyHeaderChain2(time, parentTime uint64, parentNumber, parentDiff *big.Int, hc *HeaderChain) *big.Int {
	x := new(big.Int)
	nFirstBlock := new(big.Int)
	nFirstBlock.Sub(parentNumber, nPowAveragingWindow88)

	nLastBlockTime := hc.CalcPastMedianTime(parentNumber.Uint64())
	nFirstBlockTime := hc.CalcPastMedianTime(nFirstBlock.Uint64())
	nActualTimespan := new(big.Int)
	nActualTimespan.Sub(nLastBlockTime, nFirstBlockTime)

	y := new(big.Int)
	y.Sub(nActualTimespan, AveragingWindowTimespan88())
	y.Div(y, big.NewInt(4))
	nActualTimespan.Add(y, AveragingWindowTimespan88())

	if nActualTimespan.Cmp(MinActualTimespan2()) < 0 {
		nActualTimespan.Set(MinActualTimespan2())
	} else if nActualTimespan.Cmp(MaxActualTimespan2()) > 0 {
		nActualTimespan.Set(MaxActualTimespan2())
	}

	x.Mul(parentDiff, AveragingWindowTimespan88())
	x.Div(x, nActualTimespan)

	if x.Cmp(params.MinimumDifficulty) < 0 {
		x.Set(params.MinimumDifficulty)
	}

	return x
}

func FluxDifficultyHeaderChain(time, parentTime uint64, parentNumber, parentDiff *big.Int, hc *HeaderChain) *big.Int {
	x := new(big.Int)
	nFirstBlock := new(big.Int)
	nFirstBlock.Sub(parentNumber, nPowAveragingWindow88)

	diffTime := new(big.Int)
	diffTime.Sub(big.NewInt(int64(time)), big.NewInt(int64(parentTime)))

	nLastBlockTime := hc.CalcPastMedianTime(parentNumber.Uint64())
	nFirstBlockTime := hc.CalcPastMedianTime(nFirstBlock.Uint64())
	nActualTimespan := new(big.Int)
	nActualTimespan.Sub(nLastBlockTime, nFirstBlockTime)

	y := new(big.Int)
	y.Sub(nActualTimespan, AveragingWindowTimespan88())
	y.Div(y, big.NewInt(4))
	nActualTimespan.Add(y, AveragingWindowTimespan88())

	if nActualTimespan.Cmp(MinActualTimespanFlux(false)) < 0 {
		doubleBig88 := new(big.Int)
		doubleBig88.Mul(big88, big.NewInt(2))
		if diffTime.Cmp(doubleBig88) > 0 {
			nActualTimespan.Set(MinActualTimespanFlux(true))
		} else {
			nActualTimespan.Set(MinActualTimespanFlux(false))
		}
	} else if nActualTimespan.Cmp(MaxActualTimespanFlux(false)) > 0 {
		halfBig88 := new(big.Int)
		halfBig88.Div(big88, big.NewInt(2))
		if diffTime.Cmp(halfBig88) < 0 {
			nActualTimespan.Set(MaxActualTimespanFlux(true))
		} else {
			nActualTimespan.Set(MaxActualTimespanFlux(false))
		}
	}

	x.Mul(parentDiff, AveragingWindowTimespan88())
	x.Div(x, nActualTimespan)

	if x.Cmp(params.MinimumDifficulty) < 0 {
		x.Set(params.MinimumDifficulty)
	}

	return x
}

func CalcDifficultyLegacy(config *params.ChainConfig, time, parentTime uint64, parentNumber, parentDiff *big.Int) *big.Int {
	bigTime := new(big.Int).SetUint64(time)
	bigParentTime := new(big.Int).SetUint64(parentTime)

	x := new(big.Int)
	y := new(big.Int)

	x.Sub(bigTime, bigParentTime)
	x.Div(x, big88)
	x.Sub(common.Big1, x)

	if x.Cmp(bigMinus99) < 0 {
		x.Set(bigMinus99)
	}

	y.Div(parentDiff, params.DifficultyBoundDivisor)
	x.Mul(y, x)
	x.Add(parentDiff, x)

	if x.Cmp(params.MinimumDifficulty) < 0 {
		x.Set(params.MinimumDifficulty)
	}

	return x
}

// CalcGasLimit computes the gas limit of the next block after parent.
// The result may be modified by the caller.
// This is miner strategy, not consensus protocol.
func CalcGasLimit(parent *types.Block) *big.Int {
	// contrib = (parentGasUsed * 3 / 2) / 1024
	contrib := new(big.Int).Mul(parent.GasUsed(), big.NewInt(3))
	contrib = contrib.Div(contrib, big.NewInt(2))
	contrib = contrib.Div(contrib, params.GasLimitBoundDivisor)

	// decay = parentGasLimit / 1024 -1
	decay := new(big.Int).Div(parent.GasLimit(), params.GasLimitBoundDivisor)
	decay.Sub(decay, big.NewInt(1))

	/*
		strategy: gasLimit of block-to-mine is set based on parent's
		gasUsed value.  if parentGasUsed > parentGasLimit * (2/3) then we
		increase it, otherwise lower it (or leave it unchanged if it's right
		at that usage) the amount increased/decreased depends on how far away
		from parentGasLimit * (2/3) parentGasUsed is.
	*/
	gl := new(big.Int).Sub(parent.GasLimit(), decay)
	gl = gl.Add(gl, contrib)
	gl.Set(common.BigMax(gl, params.MinGasLimit))

	// however, if we're now below the target (TargetGasLimit) we increase the
	// limit as much as we can (parentGasLimit / 1024 -1)
	if gl.Cmp(params.TargetGasLimit) < 0 {
		gl.Add(parent.GasLimit(), decay)
		gl.Set(common.BigMin(gl, params.TargetGasLimit))
	}
	return gl
}
