// Copyright 2026-2027 The quarkchain Authors
// This file is part of the quarkchain library.
//
// The quarkchain library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The quarkchain library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the quarkchain library. If not, see <http://www.gnu.org/licenses/>.

package params

import (
	"errors"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
)

var (
	// RootStakingContractAddress hosts the staking contract only on shard 0.
	// Root-chain PoSW reads miners' locked stakes and mining signers from it.
	RootStakingContractAddress = common.HexToAddress("0x514b430000000000000000000000000000000001")

	// ShardStakingContractAddress hosts a staking contract on every shard.
	// Each shard's PoSW reads its miners' locked stakes and mining signers locally.
	ShardStakingContractAddress = common.HexToAddress("0x514b430000000000000000000000000000000002")
)

// PoSWStakingRuntimeCode is the unchanged Solidity 0.5.11 runtime from pyquarkchain's
// ROOT_CHAIN_POSW_CONTRACT_BYTECODE, without its 32-byte creation prefix.
// Reuse the original runtime because the required staking behavior is unchanged.
// No functional change or applicable compiler bug has been identified that
// requires modifying and recompiling the contract.
// Contract source: https://github.com/QuarkChain/QCEPs/blob/master/QCEP/qcep-1.md
var PoSWStakingRuntimeCode = common.FromHex("0x60806040526004361061007b5760003560e01c8063853828b61161004e578063853828b6146101b5578063a69df4b5146101ca578063f83d08ba146101df578063fd8c4646146101e75761007b565b806316934fc4146100d85780632e1a7d4d1461013c578063485d3834146101685780636c19e7831461018f575b336000908152602081905260409020805460ff16156100cb5760405162461bcd60e51b815260040180806020018281038252602681526020018061062e6026913960400191505060405180910390fd5b6100d5813461023b565b50005b3480156100e457600080fd5b5061010b600480360360208110156100fb57600080fd5b50356001600160a01b031661029b565b6040805194151585526020850193909352838301919091526001600160a01b03166060830152519081900360800190f35b34801561014857600080fd5b506101666004803603602081101561015f57600080fd5b50356102cf565b005b34801561017457600080fd5b5061017d61034a565b60408051918252519081900360200190f35b610166600480360360208110156101a557600080fd5b50356001600160a01b0316610351565b3480156101c157600080fd5b506101666103c8565b3480156101d657600080fd5b50610166610436565b6101666104f7565b3480156101f357600080fd5b5061021a6004803603602081101561020a57600080fd5b50356001600160a01b0316610558565b604080519283526001600160a01b0390911660208301528051918290030190f35b8015610297576002820154808201908111610291576040805162461bcd60e51b81526020600482015260116024820152706164646974696f6e206f766572666c6f7760781b604482015290519081900360640190fd5b60028301555b5050565b600060208190529081526040902080546001820154600283015460039093015460ff9092169290916001600160a01b031684565b336000908152602081905260409020805460ff1680156102f3575080600101544210155b6102fc57600080fd5b806002015482111561030d57600080fd5b6002810180548390039055604051339083156108fc029084906000818181858888f19350505050158015610345573d6000803e3d6000fd5b505050565b6203f48081565b336000908152602081905260409020805460ff16156103a15760405162461bcd60e51b81526004018080602001828103825260268152602001806106546026913960400191505060405180910390fd5b6003810180546001600160a01b0319166001600160a01b038416179055610297813461023b565b6103d06105fa565b5033600090815260208181526040918290208251608081018452815460ff16151581526001820154928101929092526002810154928201839052600301546001600160a01b031660608201529061042657600080fd5b61043381604001516102cf565b50565b336000908152602081905260409020805460ff16156104865760405162461bcd60e51b815260040180806020018281038252602b8152602001806106a1602b913960400191505060405180910390fd5b60008160020154116104df576040805162461bcd60e51b815260206004820152601b60248201527f73686f756c642068617665206578697374696e67207374616b65730000000000604482015290519081900360640190fd5b805460ff191660019081178255426203f48001910155565b336000908152602081905260409020805460ff166105465760405162461bcd60e51b815260040180806020018281038252602781526020018061067a6027913960400191505060405180910390fd5b805460ff19168155610433813461023b565b6000806105636105fa565b506001600160a01b03808416600090815260208181526040918290208251608081018452815460ff161580158252600183015493820193909352600282015493810193909352600301549092166060820152906105c75750600091508190506105f5565b60608101516000906001600160a01b03166105e35750836105ea565b5060608101515b604090910151925090505b915091565b6040518060800160405280600015158152602001600081526020016000815260200160006001600160a01b03168152509056fe73686f756c64206f6e6c7920616464207374616b657320696e206c6f636b656420737461746573686f756c64206f6e6c7920736574207369676e657220696e206c6f636b656420737461746573686f756c64206e6f74206c6f636b20616c72656164792d6c6f636b6564206163636f756e747373686f756c64206e6f7420756e6c6f636b20616c72656164792d756e6c6f636b6564206163636f756e7473a265627a7a72315820f2c044ad50ee08e7e49c575b49e8de27cac8322afdb97780b779aa1af44e40d364736f6c634300050b0032")

// PoSWStakingRuntimeCodeHash pins the runtime used by pyquarkchain and goquarkchain.
var PoSWStakingRuntimeCodeHash = common.HexToHash("0xee90e568da573f251d63256e843add8bd7a27cec1f4c2a06ef20380be68df0a3")

// PoSWConfig defines the difficulty, staking and subsidy rules for a root or shard chain.
// Both use the same calculations with their own configuration values.
// Amounts are in the smallest native currency unit. See posw_json.go for JSON formats.
// Call Check before using the configuration for consensus calculations.
type PoSWConfig struct {
	// AdjustmentCutoff divides the child-parent timestamp delta, in seconds,
	// when computing the base PoW difficulty adjustment.
	AdjustmentCutoff uint64 `json:"adjustmentCutoff"`
	// AdjustmentFactor divides the parent difficulty to obtain one adjustment step.
	AdjustmentFactor uint64 `json:"adjustmentFactor"`
	// MinimumDifficulty is the positive uint256 floor for base PoW difficulty,
	// before staking discounts.
	MinimumDifficulty *big.Int `json:"minimumDifficulty"`

	// CoinbaseAmount is the original block subsidy, before epoch decay and reward
	// tax. Zero disables the subsidy; transaction fees are accounted for separately.
	CoinbaseAmount *big.Int `json:"coinbaseAmount"`
	// EpochInterval is the number of blocks per reward and staking decay epoch.
	EpochInterval uint64 `json:"epochInterval"`
	// DecayFactor is the fraction retained each epoch, e.g. 0.88 retains 88%.
	// Check requires a finite decimal in [0, 1] with a reduced denominator no greater than 100.
	DecayFactor *big.Rat `json:"decayFactor"`
	// OriginalBlockNumber is the old chain's stop height represented by genesis.
	// It continues reward and staking epochs while new block numbers start at zero.
	OriginalBlockNumber uint64 `json:"originalBlockNumber,omitempty"`

	// StakingEnabled enables the stake-based difficulty discount.
	StakingEnabled bool `json:"stakingEnabled"`
	// WindowSize includes the candidate block. Eligibility counts this coinbase's
	// blocks among the preceding WindowSize-1 ancestors, including ordinary PoW blocks.
	WindowSize uint64 `json:"windowSize"`
	// DiffDivider divides the base PoW difficulty when the candidate qualifies for staking.
	DiffDivider uint64 `json:"diffDivider"`
	// StakePerBlock is the original stake required per block of quota, before decay.
	// The quota is capped at WindowSize and uses the value returned by StakePerBlockAt.
	StakePerBlock *big.Int `json:"stakePerBlock"`
	// PoSWStakingDecayEnabled applies the shared epoch decay to StakePerBlock.
	// Enable explicitly for shard staking to continue mainnet's already-active
	// decay; leave false for root. Reward decay is independent of this flag.
	PoSWStakingDecayEnabled bool `json:"poswStakingDecayEnabled"`
}

// Equal compares all consensus parameters by value. Two nil configs are equal.
func (c *PoSWConfig) Equal(other *PoSWConfig) bool {
	if c == nil || other == nil {
		return c == other
	}
	return c.AdjustmentCutoff == other.AdjustmentCutoff &&
		c.AdjustmentFactor == other.AdjustmentFactor &&
		configBlockEqual(c.MinimumDifficulty, other.MinimumDifficulty) &&
		configBlockEqual(c.CoinbaseAmount, other.CoinbaseAmount) &&
		c.EpochInterval == other.EpochInterval &&
		configRatEqual(c.DecayFactor, other.DecayFactor) &&
		c.OriginalBlockNumber == other.OriginalBlockNumber &&
		c.StakingEnabled == other.StakingEnabled &&
		c.WindowSize == other.WindowSize &&
		c.DiffDivider == other.DiffDivider &&
		configBlockEqual(c.StakePerBlock, other.StakePerBlock) &&
		c.PoSWStakingDecayEnabled == other.PoSWStakingDecayEnabled
}

// Check validates the required PoW and reward rules, and staking rules when enabled.
// A nil config is invalid; PoW and reward parameters are required even with staking disabled.
func (c *PoSWConfig) Check() error {
	if c == nil || c.AdjustmentCutoff == 0 || c.AdjustmentFactor == 0 || c.MinimumDifficulty == nil || c.MinimumDifficulty.Sign() <= 0 {
		return errors.New("posw: positive cutoff, factor and minimum difficulty are required")
	}
	if c.MinimumDifficulty.BitLen() > 256 {
		return errors.New("posw: minimum difficulty must fit uint256")
	}
	if c.CoinbaseAmount == nil || c.CoinbaseAmount.Sign() < 0 || c.CoinbaseAmount.BitLen() > 256 || c.EpochInterval == 0 {
		return errors.New("posw: a uint256 block subsidy and positive epoch interval are required")
	}
	if c.DecayFactor == nil || c.DecayFactor.Sign() < 0 || c.DecayFactor.Cmp(big.NewRat(1, 1)) > 0 {
		return errors.New("posw: decay factor must be between zero and one")
	}
	// Match pyquarkchain's block_reward_decay_factor denominator limit.
	if c.DecayFactor.Denom().Cmp(big.NewInt(100)) > 0 {
		return errors.New("posw: reward decay factor denominator must not exceed 100")
	}
	if _, exact := c.DecayFactor.FloatPrec(); !exact {
		return errors.New("posw: decay factor must have a finite decimal representation")
	}
	if c.StakePerBlock != nil && (c.StakePerBlock.Sign() < 0 || c.StakePerBlock.BitLen() > 256) {
		return errors.New("posw: stakePerBlock must fit uint256")
	}
	if c.StakingEnabled && (c.WindowSize == 0 || c.DiffDivider == 0 || c.StakePerBlock == nil || c.StakePerBlock.Sign() == 0) {
		return errors.New("posw: staking requires positive windowSize, stakePerBlock and diffDivider")
	}
	return nil
}

// BlockRewardAt returns the subsidy after epoch decay, before reward tax.
// The old height offset is applied internally. Genesis has no subsidy; positive
// heights return a new integer rounded down after decay. Callers apply the
// chain's AfterTax separately, preserving the decay-before-tax rounding order.
func (c *PoSWConfig) BlockRewardAt(number *big.Int) *big.Int {
	if number.Sign() == 0 {
		return new(big.Int)
	}
	return c.decayByEpoch(c.CoinbaseAmount, number)
}

// StakePerBlockAt returns the effective stake requirement at a new chain block
// number. Root uses a fixed value; shards enable decay. The result is a new integer.
// Pass the height without adding OriginalBlockNumber. An unset StakePerBlock
// returns zero, which Check permits only when staking is disabled.
func (c *PoSWConfig) StakePerBlockAt(number *big.Int) *big.Int {
	if c.StakePerBlock == nil {
		return new(big.Int)
	}
	if !c.PoSWStakingDecayEnabled {
		// Return a copy of the fixed stake requirement when decay is disabled.
		return new(big.Int).Set(c.StakePerBlock)
	}
	// Apply epoch decay to the configured stake requirement, rounded down.
	stake := c.decayByEpoch(c.StakePerBlock, number)
	// Match pyquarkchain: use the original requirement if decay rounds to zero.
	if stake.Sign() == 0 {
		stake.Set(c.StakePerBlock)
	}
	return stake
}

// decayByEpoch computes floor(value * DecayFactor^epoch), where
// epoch = floor((number + OriginalBlockNumber) / EpochInterval).
// It rounds only after exponentiation and does not modify its inputs.
func (c *PoSWConfig) decayByEpoch(value, number *big.Int) *big.Int {
	height := new(big.Int).Add(number, new(big.Int).SetUint64(c.OriginalBlockNumber))
	epoch := height.Div(height, new(big.Int).SetUint64(c.EpochInterval))
	numerator := new(big.Int).Exp(c.DecayFactor.Num(), epoch, nil)
	denominator := new(big.Int).Exp(c.DecayFactor.Denom(), epoch, nil)
	result := new(big.Int).Mul(value, numerator)
	return result.Div(result, denominator)
}
