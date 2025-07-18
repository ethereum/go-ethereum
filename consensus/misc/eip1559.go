// Copyright 2021 The go-ethereum Authors
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

package misc

import (
	"fmt"
	"math/big"
	"sync"

	"github.com/scroll-tech/go-ethereum/common"
	"github.com/scroll-tech/go-ethereum/core/types"
	"github.com/scroll-tech/go-ethereum/log"
	"github.com/scroll-tech/go-ethereum/params"
	"github.com/scroll-tech/go-ethereum/rollup/rcfg"
	"github.com/scroll-tech/go-ethereum/rpc"
)

const (
	// Protocol-enforced maximum L2 base fee.
	// We would only go above this if L1 base fee hits 2931 Gwei.
	MaximumL2BaseFee = 10000000000

	// L2 base fee fallback values, in case the L2 system contract
	// is not deployed on not configured yet.
	DefaultBaseFeeOverhead = 15680000
	DefaultBaseFeeScalar   = 34000000000000
)

// L2 base fee formula constants and defaults.
// l2BaseFee = (l1BaseFee * scalar) / PRECISION + overhead.
// `scalar` accounts for finalization costs. `overhead` accounts for sequencing and proving costs.
var (
	// We use 1e18 for precision to match the contract implementation.
	BaseFeePrecision = new(big.Int).SetUint64(1e18)

	// scalar and overhead are updated automatically in `Blockchain.writeBlockWithState`.
	baseFeeScalar   = big.NewInt(0)
	baseFeeOverhead = big.NewInt(0)

	lock sync.RWMutex
)

func ReadL2BaseFeeCoefficients() (scalar *big.Int, overhead *big.Int) {
	lock.RLock()
	defer lock.RUnlock()
	return new(big.Int).Set(baseFeeScalar), new(big.Int).Set(baseFeeOverhead)
}

func UpdateL2BaseFeeOverhead(newOverhead *big.Int) {
	if newOverhead == nil {
		log.Error("Failed to set L2 base fee overhead, new value is <nil>")
		return
	}
	lock.Lock()
	defer lock.Unlock()
	baseFeeOverhead.Set(newOverhead)
}

func UpdateL2BaseFeeScalar(newScalar *big.Int) {
	if newScalar == nil {
		log.Error("Failed to set L2 base fee scalar, new value is <nil>")
		return
	}
	lock.Lock()
	defer lock.Unlock()
	baseFeeScalar.Set(newScalar)
}

// VerifyEip1559Header verifies some header attributes which were changed in EIP-1559,
// - gas limit check
// - basefee check
func VerifyEip1559Header(config *params.ChainConfig, parent, header *types.Header) error {
	// Verify that the gas limit remains within allowed bounds
	if err := VerifyGaslimit(parent.GasLimit, header.GasLimit); err != nil {
		return err
	}
	// Verify the header is not malformed
	if header.BaseFee == nil {
		return fmt.Errorf("header is missing baseFee")
	}
	// note: we do not verify L2 base fee, the sequencer has the
	// right to set any base fee below the maximum. L2 base fee
	// is not subject to L2 consensus or zk verification.
	if header.BaseFee.Cmp(big.NewInt(MaximumL2BaseFee)) > 0 {
		return fmt.Errorf("invalid baseFee: have %s, maximum %d", header.BaseFee, MaximumL2BaseFee)
	}
	return nil
}

// CalcBaseFee calculates the basefee of the header.
func CalcBaseFee(config *params.ChainConfig, parent *types.Header, parentL1BaseFee *big.Int, currentHeaderTime uint64) *big.Int {
	if config.Clique != nil && config.Clique.ShadowForkHeight != 0 && parent.Number.Uint64() >= config.Clique.ShadowForkHeight {
		return big.NewInt(10000000) // 0.01 Gwei
	}

	scalar, overhead := ReadL2BaseFeeCoefficients()

	if parent == nil || parent.Number == nil || !config.IsFeynman(currentHeaderTime) {
		return calcBaseFee(scalar, overhead, parentL1BaseFee)
	}
	// In Feynman base fee calculation, we reuse the contract's baseFeeOverhead slot as the proving base fee.
	return calcBaseFeeFeynman(config, parent, overhead)
}

// calcBaseFeeFeynman calculates the basefee of the header for Feynman fork.
func calcBaseFeeFeynman(config *params.ChainConfig, parent *types.Header, overhead *big.Int) *big.Int {
	baseFeeEIP1559 := calcBaseFeeEIP1559(config, parent)
	baseFee := new(big.Int).Set(baseFeeEIP1559)
	baseFee.Add(baseFee, overhead)

	// Apply maximum base fee bound to the final result (including overhead)
	if baseFee.Cmp(big.NewInt(MaximumL2BaseFee)) > 0 {
		baseFee = big.NewInt(MaximumL2BaseFee)
	}

	return baseFee
}

// CalcBaseFee calculates the basefee of the header.
func calcBaseFeeEIP1559(config *params.ChainConfig, parent *types.Header) *big.Int {
	// If the current block is the first EIP-1559 block, return the InitialBaseFee.
	if !config.IsFeynman(parent.Time) {
		// If the parent block is not nil, return its base fee to make fee transition smooth.
		if parent.BaseFee != nil {
			return new(big.Int).Set(parent.BaseFee)
		}
		return new(big.Int).SetUint64(params.InitialBaseFee)
	}

	parentBaseFeeEIP1559 := extractBaseFeeEIP1559(config, parent.BaseFee)
	parentGasTarget := parent.GasLimit / config.ElasticityMultiplier()
	// If the parent gasUsed is the same as the target, the baseFee remains unchanged.
	if parent.GasUsed == parentGasTarget {
		return new(big.Int).Set(parentBaseFeeEIP1559)
	}

	var (
		num   = new(big.Int)
		denom = new(big.Int)
	)

	if parent.GasUsed > parentGasTarget {
		// If the parent block used more gas than its target, the baseFee should increase.
		// max(1, parentBaseFee * gasUsedDelta / parentGasTarget / baseFeeChangeDenominator)
		num.SetUint64(parent.GasUsed - parentGasTarget)
		num.Mul(num, parentBaseFeeEIP1559)
		num.Div(num, denom.SetUint64(parentGasTarget))
		num.Div(num, denom.SetUint64(config.BaseFeeChangeDenominator()))
		if num.Cmp(common.Big1) < 0 {
			return num.Add(parentBaseFeeEIP1559, common.Big1)
		}
		baseFee := num.Add(parentBaseFeeEIP1559, num)
		return baseFee
	} else {
		// Otherwise if the parent block used less gas than its target, the baseFee should decrease.
		// max(0, parentBaseFee * gasUsedDelta / parentGasTarget / baseFeeChangeDenominator)
		num.SetUint64(parentGasTarget - parent.GasUsed)
		num.Mul(num, parentBaseFeeEIP1559)
		num.Div(num, denom.SetUint64(parentGasTarget))
		num.Div(num, denom.SetUint64(config.BaseFeeChangeDenominator()))

		baseFee := num.Sub(parentBaseFeeEIP1559, num)
		if baseFee.Cmp(common.Big0) < 0 {
			baseFee = common.Big0
		}
		return baseFee
	}
}

func extractBaseFeeEIP1559(_ *params.ChainConfig, baseFee *big.Int) *big.Int {
	_, overhead := ReadL2BaseFeeCoefficients()

	// In Feynman base fee calculation, we reuse the contract's baseFeeOverhead slot as the proving base fee.
	result := new(big.Int).Sub(baseFee, overhead)

	// Add underflow protection: return max(0, baseFee - overhead)
	//
	// Potential underflow scenarios:
	// - Contract overhead updates: when overhead is updated via contract,
	//   it might become larger than current base fee
	if result.Sign() < 0 {
		return big.NewInt(0)
	}
	return result
}

// MinBaseFee calculates the minimum L2 base fee based on the current coefficients.
func MinBaseFee() *big.Int {
	scalar, overhead := ReadL2BaseFeeCoefficients()
	return calcBaseFee(scalar, overhead, big.NewInt(0))
}

func calcBaseFee(scalar, overhead, parentL1BaseFee *big.Int) *big.Int {
	baseFee := new(big.Int).Set(parentL1BaseFee)
	baseFee.Mul(baseFee, scalar)
	baseFee.Div(baseFee, BaseFeePrecision)
	baseFee.Add(baseFee, overhead)

	if baseFee.Cmp(big.NewInt(MaximumL2BaseFee)) > 0 {
		baseFee = big.NewInt(MaximumL2BaseFee)
	}

	return baseFee
}

type State interface {
	GetState(addr common.Address, hash common.Hash) common.Hash
}

func InitializeL2BaseFeeCoefficients(chainConfig *params.ChainConfig, state State) error {
	overhead := common.Big0
	scalar := common.Big0

	if l2SystemConfig := chainConfig.Scroll.L2SystemConfigAddress(); l2SystemConfig != (common.Address{}) {
		overhead = state.GetState(l2SystemConfig, rcfg.L2BaseFeeOverheadSlot).Big()
		scalar = state.GetState(l2SystemConfig, rcfg.L2BaseFeeScalarSlot).Big()
	} else {
		log.Warn("L2SystemConfig address is not configured")
	}

	// fallback to default if contract is not deployed or configured yet
	if overhead.Cmp(common.Big0) == 0 {
		overhead = big.NewInt(DefaultBaseFeeOverhead)
	}
	if scalar.Cmp(common.Big0) == 0 {
		scalar = big.NewInt(DefaultBaseFeeScalar)
	}

	// update local view of coefficients
	lock.Lock()
	defer lock.Unlock()
	baseFeeOverhead.Set(overhead)
	baseFeeScalar.Set(scalar)
	log.Info("Initialized L2 base fee coefficients", "overhead", overhead, "scalar", scalar)
	return nil
}

type API struct{}

type L2BaseFeeConfig struct {
	Scalar   *big.Int `json:"scalar,omitempty"`
	Overhead *big.Int `json:"overhead,omitempty"`
}

func (api *API) GetL2BaseFeeConfig() *L2BaseFeeConfig {
	scalar, overhead := ReadL2BaseFeeCoefficients()

	return &L2BaseFeeConfig{
		Scalar:   scalar,
		Overhead: overhead,
	}
}

func APIs() []rpc.API {
	return []rpc.API{{
		Namespace: "scroll",
		Version:   "1.0",
		Service:   &API{},
		Public:    false,
	}}
}
