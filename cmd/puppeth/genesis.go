// Copyright 2017 The go-ethereum Authors
// This file is part of go-ethereum.
//
// go-ethereum is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// go-ethereum is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU General Public License for more details.
//
// You should have received a copy of the GNU General Public License
// along with go-ethereum. If not, see <http://www.gnu.org/licenses/>.

package main

import (
	"encoding/binary"
	"errors"
	"math"
	"math/big"
	"strings"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	math2 "github.com/ethereum/go-ethereum/common/math"
	"github.com/ethereum/go-ethereum/consensus/ethash"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/params"
)

// alethGenesisSpec represents the genesis specification format used by the
// C++ Ethereum implementation.
type alethGenesisSpec struct {
	SealEngine string `json:"sealEngine"`
	Params     struct {
		AccountStartNonce       math2.HexOrDecimal64   `json:"accountStartNonce"`
		MaximumExtraDataSize    hexutil.Uint64         `json:"maximumExtraDataSize"`
		HomesteadForkBlock      hexutil.Uint64         `json:"homesteadForkBlock"`
		DaoHardforkBlock        math2.HexOrDecimal64   `json:"daoHardforkBlock"`
		EIP150ForkBlock         hexutil.Uint64         `json:"EIP150ForkBlock"`
		EIP158ForkBlock         hexutil.Uint64         `json:"EIP158ForkBlock"`
		ByzantiumForkBlock      hexutil.Uint64         `json:"byzantiumForkBlock"`
		ConstantinopleForkBlock hexutil.Uint64         `json:"constantinopleForkBlock"`
		MinGasLimit             hexutil.Uint64         `json:"minGasLimit"`
		MaxGasLimit             hexutil.Uint64         `json:"maxGasLimit"`
		TieBreakingGas          bool                   `json:"tieBreakingGas"`
		GasLimitBoundDivisor    math2.HexOrDecimal64   `json:"gasLimitBoundDivisor"`
		MinimumDifficulty       *hexutil.Big           `json:"minimumDifficulty"`
		DifficultyBoundDivisor  *math2.HexOrDecimal256 `json:"difficultyBoundDivisor"`
		DurationLimit           *math2.HexOrDecimal256 `json:"durationLimit"`
		BlockReward             *hexutil.Big           `json:"blockReward"`
		NetworkID               hexutil.Uint64         `json:"networkID"`
		ChainID                 hexutil.Uint64         `json:"chainID"`
		AllowFutureBlocks       bool                   `json:"allowFutureBlocks"`
	} `json:"params"`

	Genesis struct {
		Nonce      hexutil.Bytes  `json:"nonce"`
		Difficulty *hexutil.Big   `json:"difficulty"`
		MixHash    common.Hash    `json:"mixHash"`
		Author     common.Address `json:"author"`
		Timestamp  hexutil.Uint64 `json:"timestamp"`
		ParentHash common.Hash    `json:"parentHash"`
		ExtraData  hexutil.Bytes  `json:"extraData"`
		GasLimit   hexutil.Uint64 `json:"gasLimit"`
	} `json:"genesis"`

	Accounts map[common.UnprefixedAddress]*alethGenesisSpecAccount `json:"accounts"`
}

// alethGenesisSpecAccount is the prefunded genesis account and/or precompiled
// contract definition.
type alethGenesisSpecAccount struct {
	Balance     *math2.HexOrDecimal256   `json:"balance"`
	Nonce       uint64                   `json:"nonce,omitempty"`
	Precompiled *alethGenesisSpecBuiltin `json:"precompiled,omitempty"`
}

// alethGenesisSpecBuiltin is the precompiled contract definition.
type alethGenesisSpecBuiltin struct {
	Name          string                         `json:"name,omitempty"`
	StartingBlock hexutil.Uint64                 `json:"startingBlock,omitempty"`
	Linear        *alethGenesisSpecLinearPricing `json:"linear,omitempty"`
}

type alethGenesisSpecLinearPricing struct {
	Base uint64 `json:"base"`
	Word uint64 `json:"word"`
}

// newAlethGenesisSpec converts a go-ethereum genesis block into a Aleth-specific
// chain specification format.
func newAlethGenesisSpec(network string, genesis *core.Genesis) (*alethGenesisSpec, error) {
	// Only ethash is currently supported between go-ethereum and aleth
	if genesis.Config.Ethash == nil {
		return nil, errors.New("unsupported consensus engine")
	}
	// Reconstruct the chain spec in Aleth format
	spec := &alethGenesisSpec{
		SealEngine: "Ethash",
	}
	// Some defaults
	spec.Params.AccountStartNonce = 0
	spec.Params.TieBreakingGas = false
	spec.Params.AllowFutureBlocks = false
	spec.Params.DaoHardforkBlock = 0

	spec.Params.HomesteadForkBlock = (hexutil.Uint64)(genesis.Config.HomesteadBlock.Uint64())
	spec.Params.EIP150ForkBlock = (hexutil.Uint64)(genesis.Config.EIP150Block.Uint64())
	spec.Params.EIP158ForkBlock = (hexutil.Uint64)(genesis.Config.EIP158Block.Uint64())

	// Byzantium
	if num := genesis.Config.ByzantiumBlock; num != nil {
		spec.setByzantium(num)
	}
	// Constantinople
	if num := genesis.Config.ConstantinopleBlock; num != nil {
		spec.setConstantinople(num)
	}

	spec.Params.NetworkID = (hexutil.Uint64)(genesis.Config.ChainID.Uint64())
	spec.Params.ChainID = (hexutil.Uint64)(genesis.Config.ChainID.Uint64())
	spec.Params.MaximumExtraDataSize = (hexutil.Uint64)(params.MaximumExtraDataSize)
	spec.Params.MinGasLimit = (hexutil.Uint64)(params.MinGasLimit)
	spec.Params.MaxGasLimit = (hexutil.Uint64)(math.MaxInt64)
	spec.Params.MinimumDifficulty = (*hexutil.Big)(params.MinimumDifficulty)
	spec.Params.DifficultyBoundDivisor = (*math2.HexOrDecimal256)(params.DifficultyBoundDivisor)
	spec.Params.GasLimitBoundDivisor = (math2.HexOrDecimal64)(params.GasLimitBoundDivisor)
	spec.Params.DurationLimit = (*math2.HexOrDecimal256)(params.DurationLimit)
	spec.Params.BlockReward = (*hexutil.Big)(ethash.FrontierBlockReward)

	spec.Genesis.Nonce = (hexutil.Bytes)(make([]byte, 8))
	binary.LittleEndian.PutUint64(spec.Genesis.Nonce[:], genesis.Nonce)

	spec.Genesis.MixHash = genesis.Mixhash
	spec.Genesis.Difficulty = (*hexutil.Big)(genesis.Difficulty)
	spec.Genesis.Author = genesis.Coinbase
	spec.Genesis.Timestamp = (hexutil.Uint64)(genesis.Timestamp)
	spec.Genesis.ParentHash = genesis.ParentHash
	spec.Genesis.ExtraData = (hexutil.Bytes)(genesis.ExtraData)
	spec.Genesis.GasLimit = (hexutil.Uint64)(genesis.GasLimit)

	for address, account := range genesis.Alloc {
		spec.setAccount(address, account)
	}

	spec.setPrecompile(1, &alethGenesisSpecBuiltin{Name: "ecrecover",
		Linear: &alethGenesisSpecLinearPricing{Base: 3000}})
	spec.setPrecompile(2, &alethGenesisSpecBuiltin{Name: "sha256",
		Linear: &alethGenesisSpecLinearPricing{Base: 60, Word: 12}})
	spec.setPrecompile(3, &alethGenesisSpecBuiltin{Name: "ripemd160",
		Linear: &alethGenesisSpecLinearPricing{Base: 600, Word: 120}})
	spec.setPrecompile(4, &alethGenesisSpecBuiltin{Name: "identity",
		Linear: &alethGenesisSpecLinearPricing{Base: 15, Word: 3}})
	if genesis.Config.ByzantiumBlock != nil {
		spec.setPrecompile(5, &alethGenesisSpecBuiltin{Name: "modexp",
			StartingBlock: (hexutil.Uint64)(genesis.Config.ByzantiumBlock.Uint64())})
		spec.setPrecompile(6, &alethGenesisSpecBuiltin{Name: "alt_bn128_G1_add",
			StartingBlock: (hexutil.Uint64)(genesis.Config.ByzantiumBlock.Uint64()),
			Linear:        &alethGenesisSpecLinearPricing{Base: 500}})
		spec.setPrecompile(7, &alethGenesisSpecBuiltin{Name: "alt_bn128_G1_mul",
			StartingBlock: (hexutil.Uint64)(genesis.Config.ByzantiumBlock.Uint64()),
			Linear:        &alethGenesisSpecLinearPricing{Base: 40000}})
		spec.setPrecompile(8, &alethGenesisSpecBuiltin{Name: "alt_bn128_pairing_product",
			StartingBlock: (hexutil.Uint64)(genesis.Config.ByzantiumBlock.Uint64())})
	}
	return spec, nil
}

func (spec *alethGenesisSpec) setPrecompile(address byte, data *alethGenesisSpecBuiltin) {
	if spec.Accounts == nil {
		spec.Accounts = make(map[common.UnprefixedAddress]*alethGenesisSpecAccount)
	}
	addr := common.UnprefixedAddress(common.BytesToAddress([]byte{address}))
	if _, exist := spec.Accounts[addr]; !exist {
		spec.Accounts[addr] = &alethGenesisSpecAccount{}
	}
	spec.Accounts[addr].Precompiled = data
}

func (spec *alethGenesisSpec) setAccount(address common.Address, account core.GenesisAccount) {
	if spec.Accounts == nil {
		spec.Accounts = make(map[common.UnprefixedAddress]*alethGenesisSpecAccount)
	}

	a, exist := spec.Accounts[common.UnprefixedAddress(address)]
	if !exist {
		a = &alethGenesisSpecAccount{}
		spec.Accounts[common.UnprefixedAddress(address)] = a
	}
	a.Balance = (*math2.HexOrDecimal256)(account.Balance)
	a.Nonce = account.Nonce

}

func (spec *alethGenesisSpec) setByzantium(num *big.Int) {
	spec.Params.ByzantiumForkBlock = hexutil.Uint64(num.Uint64())
}

func (spec *alethGenesisSpec) setConstantinople(num *big.Int) {
	spec.Params.ConstantinopleForkBlock = hexutil.Uint64(num.Uint64())
}

// parityChainSpec is the chain specification format used by Parity.
type parityChainSpec struct {
	Name    string `json:"name"`
	Datadir string `json:"dataDir"`
	Engine  struct {
		Ethash struct {
			Params struct {
				MinimumDifficulty      *hexutil.Big      `json:"minimumDifficulty"`
				DifficultyBoundDivisor *hexutil.Big      `json:"difficultyBoundDivisor"`
				DurationLimit          *hexutil.Big      `json:"durationLimit"`
				BlockReward            map[string]string `json:"blockReward"`
				DifficultyBombDelays   map[string]string `json:"difficultyBombDelays"`
				HomesteadTransition    hexutil.Uint64    `json:"homesteadTransition"`
				EIP100bTransition      hexutil.Uint64    `json:"eip100bTransition"`
			} `json:"params"`
		} `json:"Ethash"`
	} `json:"engine"`

	Params struct {
		AccountStartNonce     hexutil.Uint64       `json:"accountStartNonce"`
		MaximumExtraDataSize  hexutil.Uint64       `json:"maximumExtraDataSize"`
		MinGasLimit           hexutil.Uint64       `json:"minGasLimit"`
		GasLimitBoundDivisor  math2.HexOrDecimal64 `json:"gasLimitBoundDivisor"`
		NetworkID             hexutil.Uint64       `json:"networkID"`
		ChainID               hexutil.Uint64       `json:"chainID"`
		MaxCodeSize           hexutil.Uint64       `json:"maxCodeSize"`
		MaxCodeSizeTransition hexutil.Uint64       `json:"maxCodeSizeTransition"`
		EIP98Transition       hexutil.Uint64       `json:"eip98Transition"`
		EIP150Transition      hexutil.Uint64       `json:"eip150Transition"`
		EIP160Transition      hexutil.Uint64       `json:"eip160Transition"`
		EIP161abcTransition   hexutil.Uint64       `json:"eip161abcTransition"`
		EIP161dTransition     hexutil.Uint64       `json:"eip161dTransition"`
		EIP155Transition      hexutil.Uint64       `json:"eip155Transition"`
		EIP140Transition      hexutil.Uint64       `json:"eip140Transition"`
		EIP211Transition      hexutil.Uint64       `json:"eip211Transition"`
		EIP214Transition      hexutil.Uint64       `json:"eip214Transition"`
		EIP658Transition      hexutil.Uint64       `json:"eip658Transition"`
		EIP145Transition      hexutil.Uint64       `json:"eip145Transition"`
		EIP1014Transition     hexutil.Uint64       `json:"eip1014Transition"`
		EIP1052Transition     hexutil.Uint64       `json:"eip1052Transition"`
		EIP1283Transition     hexutil.Uint64       `json:"eip1283Transition"`
	} `json:"params"`

	Genesis struct {
		Seal struct {
			Ethereum struct {
				Nonce   hexutil.Bytes `json:"nonce"`
				MixHash hexutil.Bytes `json:"mixHash"`
			} `json:"ethereum"`
		} `json:"seal"`

		Difficulty *hexutil.Big   `json:"difficulty"`
		Author     common.Address `json:"author"`
		Timestamp  hexutil.Uint64 `json:"timestamp"`
		ParentHash common.Hash    `json:"parentHash"`
		ExtraData  hexutil.Bytes  `json:"extraData"`
		GasLimit   hexutil.Uint64 `json:"gasLimit"`
	} `json:"genesis"`

	Nodes    []string                                             `json:"nodes"`
	Accounts map[common.UnprefixedAddress]*parityChainSpecAccount `json:"accounts"`
}

// parityChainSpecAccount is the prefunded genesis account and/or precompiled
// contract definition.
type parityChainSpecAccount struct {
	Balance math2.HexOrDecimal256   `json:"balance"`
	Nonce   math2.HexOrDecimal64    `json:"nonce,omitempty"`
	Builtin *parityChainSpecBuiltin `json:"builtin,omitempty"`
}

// parityChainSpecBuiltin is the precompiled contract definition.
type parityChainSpecBuiltin struct {
	Name       string                  `json:"name,omitempty"`
	ActivateAt math2.HexOrDecimal64    `json:"activate_at,omitempty"`
	Pricing    *parityChainSpecPricing `json:"pricing,omitempty"`
}

// parityChainSpecPricing represents the different pricing models that builtin
// contracts might advertise using.
type parityChainSpecPricing struct {
	Linear       *parityChainSpecLinearPricing       `json:"linear,omitempty"`
	ModExp       *parityChainSpecModExpPricing       `json:"modexp,omitempty"`
	AltBnPairing *parityChainSpecAltBnPairingPricing `json:"alt_bn128_pairing,omitempty"`
}

type parityChainSpecLinearPricing struct {
	Base uint64 `json:"base"`
	Word uint64 `json:"word"`
}

type parityChainSpecModExpPricing struct {
	Divisor uint64 `json:"divisor"`
}

type parityChainSpecAltBnPairingPricing struct {
	Base uint64 `json:"base"`
	Pair uint64 `json:"pair"`
}

// newParityChainSpec converts a go-ethereum genesis block into a Parity specific
// chain specification format.
func newParityChainSpec(network string, genesis *core.Genesis, bootnodes []string) (*parityChainSpec, error) {
	// Only ethash is currently supported between go-ethereum and Parity
	if genesis.Config.Ethash == nil {
		return nil, errors.New("unsupported consensus engine")
	}
	// Reconstruct the chain spec in Parity's format
	spec := &parityChainSpec{
		Name:    network,
		Nodes:   bootnodes,
		Datadir: strings.ToLower(network),
	}
	spec.Engine.Ethash.Params.BlockReward = make(map[string]string)
	spec.Engine.Ethash.Params.DifficultyBombDelays = make(map[string]string)
	// Frontier
	spec.Engine.Ethash.Params.MinimumDifficulty = (*hexutil.Big)(params.MinimumDifficulty)
	spec.Engine.Ethash.Params.DifficultyBoundDivisor = (*hexutil.Big)(params.DifficultyBoundDivisor)
	spec.Engine.Ethash.Params.DurationLimit = (*hexutil.Big)(params.DurationLimit)
	spec.Engine.Ethash.Params.BlockReward["0x0"] = hexutil.EncodeBig(ethash.FrontierBlockReward)

	// Homestead
	spec.Engine.Ethash.Params.HomesteadTransition = hexutil.Uint64(genesis.Config.HomesteadBlock.Uint64())

	// Tangerine Whistle : 150
	// https://github.com/ethereum/EIPs/blob/master/EIPS/eip-608.md
	spec.Params.EIP150Transition = hexutil.Uint64(genesis.Config.EIP150Block.Uint64())

	// Spurious Dragon: 155, 160, 161, 170
	// https://github.com/ethereum/EIPs/blob/master/EIPS/eip-607.md
	spec.Params.EIP155Transition = hexutil.Uint64(genesis.Config.EIP155Block.Uint64())
	spec.Params.EIP160Transition = hexutil.Uint64(genesis.Config.EIP155Block.Uint64())
	spec.Params.EIP161abcTransition = hexutil.Uint64(genesis.Config.EIP158Block.Uint64())
	spec.Params.EIP161dTransition = hexutil.Uint64(genesis.Config.EIP158Block.Uint64())

	// Byzantium
	if num := genesis.Config.ByzantiumBlock; num != nil {
		spec.setByzantium(num)
	}
	// Constantinople
	if num := genesis.Config.ConstantinopleBlock; num != nil {
		spec.setConstantinople(num)
	}
	spec.Params.MaximumExtraDataSize = (hexutil.Uint64)(params.MaximumExtraDataSize)
	spec.Params.MinGasLimit = (hexutil.Uint64)(params.MinGasLimit)
	spec.Params.GasLimitBoundDivisor = (math2.HexOrDecimal64)(params.GasLimitBoundDivisor)
	spec.Params.NetworkID = (hexutil.Uint64)(genesis.Config.ChainID.Uint64())
	spec.Params.ChainID = (hexutil.Uint64)(genesis.Config.ChainID.Uint64())
	spec.Params.MaxCodeSize = params.MaxCodeSize
	// geth has it set from zero
	spec.Params.MaxCodeSizeTransition = 0

	// Disable this one
	spec.Params.EIP98Transition = math.MaxInt64

	spec.Genesis.Seal.Ethereum.Nonce = (hexutil.Bytes)(make([]byte, 8))
	binary.LittleEndian.PutUint64(spec.Genesis.Seal.Ethereum.Nonce[:], genesis.Nonce)

	spec.Genesis.Seal.Ethereum.MixHash = (hexutil.Bytes)(genesis.Mixhash[:])
	spec.Genesis.Difficulty = (*hexutil.Big)(genesis.Difficulty)
	spec.Genesis.Author = genesis.Coinbase
	spec.Genesis.Timestamp = (hexutil.Uint64)(genesis.Timestamp)
	spec.Genesis.ParentHash = genesis.ParentHash
	spec.Genesis.ExtraData = (hexutil.Bytes)(genesis.ExtraData)
	spec.Genesis.GasLimit = (hexutil.Uint64)(genesis.GasLimit)

	spec.Accounts = make(map[common.UnprefixedAddress]*parityChainSpecAccount)
	for address, account := range genesis.Alloc {
		bal := math2.HexOrDecimal256(*account.Balance)

		spec.Accounts[common.UnprefixedAddress(address)] = &parityChainSpecAccount{
			Balance: bal,
			Nonce:   math2.HexOrDecimal64(account.Nonce),
		}
	}
	spec.setPrecompile(1, &parityChainSpecBuiltin{Name: "ecrecover",
		Pricing: &parityChainSpecPricing{Linear: &parityChainSpecLinearPricing{Base: 3000}}})

	spec.setPrecompile(2, &parityChainSpecBuiltin{
		Name: "sha256", Pricing: &parityChainSpecPricing{Linear: &parityChainSpecLinearPricing{Base: 60, Word: 12}},
	})
	spec.setPrecompile(3, &parityChainSpecBuiltin{
		Name: "ripemd160", Pricing: &parityChainSpecPricing{Linear: &parityChainSpecLinearPricing{Base: 600, Word: 120}},
	})
	spec.setPrecompile(4, &parityChainSpecBuiltin{
		Name: "identity", Pricing: &parityChainSpecPricing{Linear: &parityChainSpecLinearPricing{Base: 15, Word: 3}},
	})
	if genesis.Config.ByzantiumBlock != nil {
		blnum := math2.HexOrDecimal64(genesis.Config.ByzantiumBlock.Uint64())
		spec.setPrecompile(5, &parityChainSpecBuiltin{
			Name: "modexp", ActivateAt: blnum, Pricing: &parityChainSpecPricing{ModExp: &parityChainSpecModExpPricing{Divisor: 20}},
		})
		spec.setPrecompile(6, &parityChainSpecBuiltin{
			Name: "alt_bn128_add", ActivateAt: blnum, Pricing: &parityChainSpecPricing{Linear: &parityChainSpecLinearPricing{Base: 500}},
		})
		spec.setPrecompile(7, &parityChainSpecBuiltin{
			Name: "alt_bn128_mul", ActivateAt: blnum, Pricing: &parityChainSpecPricing{Linear: &parityChainSpecLinearPricing{Base: 40000}},
		})
		spec.setPrecompile(8, &parityChainSpecBuiltin{
			Name: "alt_bn128_pairing", ActivateAt: blnum, Pricing: &parityChainSpecPricing{AltBnPairing: &parityChainSpecAltBnPairingPricing{Base: 100000, Pair: 80000}},
		})
	}
	return spec, nil
}

func (spec *parityChainSpec) setPrecompile(address byte, data *parityChainSpecBuiltin) {
	if spec.Accounts == nil {
		spec.Accounts = make(map[common.UnprefixedAddress]*parityChainSpecAccount)
	}
	a := common.UnprefixedAddress(common.BytesToAddress([]byte{address}))
	if _, exist := spec.Accounts[a]; !exist {
		spec.Accounts[a] = &parityChainSpecAccount{}
	}
	spec.Accounts[a].Builtin = data
}

func (spec *parityChainSpec) setByzantium(num *big.Int) {
	spec.Engine.Ethash.Params.BlockReward[hexutil.EncodeBig(num)] = hexutil.EncodeBig(ethash.ByzantiumBlockReward)
	spec.Engine.Ethash.Params.DifficultyBombDelays[hexutil.EncodeBig(num)] = hexutil.EncodeUint64(3000000)
	n := hexutil.Uint64(num.Uint64())
	spec.Engine.Ethash.Params.EIP100bTransition = n
	spec.Params.EIP140Transition = n
	spec.Params.EIP211Transition = n
	spec.Params.EIP214Transition = n
	spec.Params.EIP658Transition = n
}

func (spec *parityChainSpec) setConstantinople(num *big.Int) {
	spec.Engine.Ethash.Params.BlockReward[hexutil.EncodeBig(num)] = hexutil.EncodeBig(ethash.ConstantinopleBlockReward)
	spec.Engine.Ethash.Params.DifficultyBombDelays[hexutil.EncodeBig(num)] = hexutil.EncodeUint64(2000000)
	n := hexutil.Uint64(num.Uint64())
	spec.Params.EIP145Transition = n
	spec.Params.EIP1014Transition = n
	spec.Params.EIP1052Transition = n
	spec.Params.EIP1283Transition = n
}

// pyEthereumGenesisSpec represents the genesis specification format used by the
// Python Ethereum implementation.
type pyEthereumGenesisSpec struct {
	Nonce      hexutil.Bytes     `json:"nonce"`
	Timestamp  hexutil.Uint64    `json:"timestamp"`
	ExtraData  hexutil.Bytes     `json:"extraData"`
	GasLimit   hexutil.Uint64    `json:"gasLimit"`
	Difficulty *hexutil.Big      `json:"difficulty"`
	Mixhash    common.Hash       `json:"mixhash"`
	Coinbase   common.Address    `json:"coinbase"`
	Alloc      core.GenesisAlloc `json:"alloc"`
	ParentHash common.Hash       `json:"parentHash"`
}

// newPyEthereumGenesisSpec converts a go-ethereum genesis block into a Parity specific
// chain specification format.
func newPyEthereumGenesisSpec(network string, genesis *core.Genesis) (*pyEthereumGenesisSpec, error) {
	// Only ethash is currently supported between go-ethereum and pyethereum
	if genesis.Config.Ethash == nil {
		return nil, errors.New("unsupported consensus engine")
	}
	spec := &pyEthereumGenesisSpec{
		Timestamp:  (hexutil.Uint64)(genesis.Timestamp),
		ExtraData:  genesis.ExtraData,
		GasLimit:   (hexutil.Uint64)(genesis.GasLimit),
		Difficulty: (*hexutil.Big)(genesis.Difficulty),
		Mixhash:    genesis.Mixhash,
		Coinbase:   genesis.Coinbase,
		Alloc:      genesis.Alloc,
		ParentHash: genesis.ParentHash,
	}
	spec.Nonce = (hexutil.Bytes)(make([]byte, 8))
	binary.LittleEndian.PutUint64(spec.Nonce[:], genesis.Nonce)

	return spec, nil
}
