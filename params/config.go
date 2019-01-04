// Copyright 2016 The go-ethereum Authors
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

package params

import (
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
)

// Genesis hashes to enforce below configs on.
var (
	MainnetGenesisHash = common.HexToHash("0xd4e56740f876aef8c010b86a40d5f56745a118d0906a34e69aec8c0db1cb8fa3")
	TestnetGenesisHash = common.HexToHash("0x41941023680923e0fe4d74a34bdac8141f2540e3ae90623718e47d66d1ca4a2d")
	RinkebyGenesisHash = common.HexToHash("0x6341fd3daf94b748c72ced5a5b26028f2474f5f00d824504e4fa37a75767e177")
)

var (
	// MainnetChainConfig is the chain parameters to run a node on the main network.
	MainnetChainConfig = &ChainConfig{
		ChainID:             big.NewInt(1),
		HomesteadBlock:      big.NewInt(1150000),
		DAOForkBlock:        big.NewInt(1920000),
		DAOForkSupport:      true,
		EIP150Block:         big.NewInt(2463000),
		EIP150Hash:          common.HexToHash("0x2086799aeebeae135c246c65021c82b4e15a2c451340993aacfd2751886514f0"),
		EIP155Block:         big.NewInt(2675000),
		EIP158Block:         big.NewInt(2675000),
		ByzantiumBlock:      big.NewInt(4370000),
		ConstantinopleBlock: big.NewInt(7080000),
		Ethash:              new(EthashConfig),
	}

	// MainnetTrustedCheckpoint contains the light client trusted checkpoint for the main network.
	MainnetTrustedCheckpoint = &TrustedCheckpoint{
		Name:         "mainnet",
		SectionIndex: 208,
		SectionHead:  common.HexToHash("0x5e9f7696c397d9df8f3b1abda857753575c6f5cff894e1a3d9e1a2af1bd9d6ac"),
		CHTRoot:      common.HexToHash("0x954a63134f6897f015f026387c59c98c4dae7b336610ff5a143455aac9153e9d"),
		BloomRoot:    common.HexToHash("0x8006c5e44b14d90d7cc9cd5fa1cb48cf53697ee3bbbf4b76fdfa70b0242500a9"),
	}

	// TestnetChainConfig contains the chain parameters to run a node on the Ropsten test network.
	TestnetChainConfig = &ChainConfig{
		ChainID:             big.NewInt(3),
		HomesteadBlock:      big.NewInt(0),
		DAOForkBlock:        nil,
		DAOForkSupport:      true,
		EIP150Block:         big.NewInt(0),
		EIP150Hash:          common.HexToHash("0x41941023680923e0fe4d74a34bdac8141f2540e3ae90623718e47d66d1ca4a2d"),
		EIP155Block:         big.NewInt(10),
		EIP158Block:         big.NewInt(10),
		ByzantiumBlock:      big.NewInt(1700000),
		ConstantinopleBlock: big.NewInt(4230000),
		Ethash:              new(EthashConfig),
	}

	// TestnetTrustedCheckpoint contains the light client trusted checkpoint for the Ropsten test network.
	TestnetTrustedCheckpoint = &TrustedCheckpoint{
		Name:         "testnet",
		SectionIndex: 139,
		SectionHead:  common.HexToHash("0x9fad89a5e3b993c8339b9cf2cbbeb72cd08774ea6b71b105b3dd880420c618f4"),
		CHTRoot:      common.HexToHash("0xc815833881989c5d2035147e1a79a33d22cbc5313e104ff01e6ab405bd28b317"),
		BloomRoot:    common.HexToHash("0xd94ee9f3c480858f53ec5d059aebdbb2e8d904702f100875ee59ec5f366e841d"),
	}

	// RinkebyChainConfig contains the chain parameters to run a node on the Rinkeby test network.
	RinkebyChainConfig = &ChainConfig{
		ChainID:             big.NewInt(4),
		HomesteadBlock:      big.NewInt(1),
		DAOForkBlock:        nil,
		DAOForkSupport:      true,
		EIP150Block:         big.NewInt(2),
		EIP150Hash:          common.HexToHash("0x9b095b36c15eaf13044373aef8ee0bd3a382a5abb92e402afa44b8249c3a90e9"),
		EIP155Block:         big.NewInt(3),
		EIP158Block:         big.NewInt(3),
		ByzantiumBlock:      big.NewInt(1035301),
		ConstantinopleBlock: big.NewInt(3660663),
		Clique: &CliqueConfig{
			Period: 15,
			Epoch:  30000,
		},
	}

	// RinkebyTrustedCheckpoint contains the light client trusted checkpoint for the Rinkeby test network.
	RinkebyTrustedCheckpoint = &TrustedCheckpoint{
		Name:         "rinkeby",
		SectionIndex: 105,
		SectionHead:  common.HexToHash("0xec8147d43f936258aaf1b9b9ec91b0a853abf7109f436a23649be809ea43d507"),
		CHTRoot:      common.HexToHash("0xd92703b444846a3db928e87e450770e5d5cbe193131dc8f7c4cf18b4de925a75"),
		BloomRoot:    common.HexToHash("0xff45a6f807138a2cde0cea0c209d9ce5ad8e43ccaae5a7c41af801bb72a1ef96"),
	}

	// AllEthashProtocolChanges contains every protocol change (EIPs) introduced
	// and accepted by the Ethereum core developers into the Ethash consensus.
	//
	// This configuration is intentionally not using keyed fields to force anyone
	// adding flags to the config to also have to set these fields.
	AllEthashProtocolChanges = &ChainConfig{
		big.NewInt(1337), // ChainID

		big.NewInt(0), // HomesteadBlock
		nil,           // EIP7FBlock

		nil,   // DAOForkBlock
		false, // DAOForkSupport

		big.NewInt(0), // EIP150Block
		common.Hash{}, // EIP150Hash
		big.NewInt(0), // EIP155Block
		big.NewInt(0), // EIP158Block
		nil,           // EIP160FBlock
		nil,           // EIP161FBlock
		nil,           // EIP170FBlock

		big.NewInt(0), // ByzantiumBlock
		nil,           // EIP100FBlock
		nil,           // EIP140FBlock
		nil,           // EIP198FBlock
		nil,           // EIP211FBlock
		nil,           // EIP212FBlock
		nil,           // EIP213FBlock
		nil,           // EIP214FBlock
		nil,           // EIP649FBlock
		nil,           // EIP658FBlock

		big.NewInt(0), // ConstantinopleBlock
		nil,           // EIP145FBlock
		nil,           // EIP1014FBlock
		nil,           // EIP1052FBlock
		nil,           // EIP1234FBlock
		nil,           // EIP1283FBlock

		nil,               // EWASMBlock
		new(EthashConfig), // Ethash
		nil,               // Clique
	}

	// AllCliqueProtocolChanges contains every protocol change (EIPs) introduced
	// and accepted by the Ethereum core developers into the Clique consensus.
	//
	// This configuration is intentionally not using keyed fields to force anyone
	// adding flags to the config to also have to set these fields.
	AllCliqueProtocolChanges = &ChainConfig{
		big.NewInt(1337), // ChainID

		big.NewInt(0), // HomesteadBlock
		nil,           // EIP7FBlock

		nil,   // DAOForkBlock
		false, // DAOForkSupport

		big.NewInt(0), // EIP150Block
		common.Hash{}, // EIP150Hash
		big.NewInt(0), // EIP155Block
		big.NewInt(0), // EIP158Block
		nil,           // EIP160FBlock
		nil,           // EIP161FBlock
		nil,           // EIP170FBlock

		big.NewInt(0), // ByzantiumBlock
		nil,           // EIP100FBlock
		nil,           // EIP140FBlock
		nil,           // EIP198FBlock
		nil,           // EIP211FBlock
		nil,           // EIP212FBlock
		nil,           // EIP213FBlock
		nil,           // EIP214FBlock
		nil,           // EIP649FBlock
		nil,           // EIP658FBlock

		big.NewInt(0), // ConstantinopleBlock
		nil,           // EIP145FBlock
		nil,           // EIP1014FBlock
		nil,           // EIP1052FBlock
		nil,           // EIP1234FBlock
		nil,           // EIP1283FBlock

		nil, // EWASMBlock
		nil, // Ethash
		&CliqueConfig{
			Period: 0,
			Epoch:  30000,
		},
	}

	// TestChainConfig is used for tests.
	TestChainConfig = &ChainConfig{
		big.NewInt(1), // ChainID

		big.NewInt(0), // HomesteadBlock
		nil,           // EIP7FBlock

		nil,   // DAOForkBlock
		false, // DAOForkSupport

		big.NewInt(0), // EIP150Block
		common.Hash{}, // EIP150Hash
		big.NewInt(0), // EIP155Block
		big.NewInt(0), // EIP158Block
		nil,           // EIP160FBlock
		nil,           // EIP161FBlock
		nil,           // EIP170FBlock

		big.NewInt(0), // ByzantiumBlock
		nil,           // EIP100FBlock
		nil,           // EIP140FBlock
		nil,           // EIP198FBlock
		nil,           // EIP211FBlock
		nil,           // EIP212FBlock
		nil,           // EIP213FBlock
		nil,           // EIP214FBlock
		nil,           // EIP649FBlock
		nil,           // EIP658FBlock

		big.NewInt(0), // ConstantinopleBlock
		nil,           // EIP145FBlock
		nil,           // EIP1014FBlock
		nil,           // EIP1052FBlock
		nil,           // EIP1234FBlock
		nil,           // EIP1283FBlock

		nil,               // EWASMBlock
		new(EthashConfig), // Ethash
		nil,               // Clique
	}

	// TestRules are all rules from TestChainConfig initialized at 0.
	TestRules = TestChainConfig.Rules(new(big.Int))
)

// TrustedCheckpoint represents a set of post-processed trie roots (CHT and
// BloomTrie) associated with the appropriate section index and head hash. It is
// used to start light syncing from this checkpoint and avoid downloading the
// entire header chain while still being able to securely access old headers/logs.
type TrustedCheckpoint struct {
	Name         string      `json:"-"`
	SectionIndex uint64      `json:"sectionIndex"`
	SectionHead  common.Hash `json:"sectionHead"`
	CHTRoot      common.Hash `json:"chtRoot"`
	BloomRoot    common.Hash `json:"bloomRoot"`
}

// ChainConfig is the core config which determines the blockchain settings.
//
// ChainConfig is stored in the database on a per block basis. This means
// that any network, identified by its genesis block, can have its own
// set of configuration options.
type ChainConfig struct {
	ChainID *big.Int `json:"chainId"` // chainId identifies the current chain and is used for replay protection

	// HF: Homestead
	HomesteadBlock *big.Int `json:"homesteadBlock,omitempty"` // Homestead switch block (nil = no fork, 0 = already homestead)
	// Note: EIPs 2 and 8 were also included in this fork, but have not been distinguished individually in the code.
	//
	// DELEGATECALL
	// https://eips.ethereum.org/EIPS/eip-7
	EIP7FBlock *big.Int `json:"eip7FBlock,omitempy"`

	// HF: DAO
	DAOForkBlock   *big.Int `json:"daoForkBlock,omitempty"`   // TheDAO hard-fork switch block (nil = no fork)
	DAOForkSupport bool     `json:"daoForkSupport,omitempty"` // Whether the nodes supports or opposes the DAO hard-fork

	// HF: Tangerine Whistle
	// EIP150 implements the Gas price changes (https://github.com/ethereum/EIPs/issues/150)
	EIP150Block *big.Int    `json:"eip150Block,omitempty"` // EIP150 HF block (nil = no fork)
	EIP150Hash  common.Hash `json:"eip150Hash,omitempty"`  // EIP150 HF hash (needed for header only clients as only gas pricing changed)

	// HF: Spurious Dragon
	EIP155Block *big.Int `json:"eip155Block,omitempty"` // EIP155 HF block
	EIP158Block *big.Int `json:"eip158Block,omitempty"` // EIP158 HF block, includes implementations of 158/161, 160, and 170
	//
	// EXP cost increase
	// https://github.com/ethereum/EIPs/blob/master/EIPS/eip-160.md
	EIP160FBlock *big.Int `json:"eip160FBlock,omitempty"`
	// State trie clearing (== EIP158 proper)
	// https://github.com/ethereum/EIPs/blob/master/EIPS/eip-161.md
	EIP161FBlock *big.Int `json:"eip161FBlock,omitempty"`
	// Contract code size limit
	// https://github.com/ethereum/EIPs/blob/master/EIPS/eip-170.md
	EIP170FBlock *big.Int `json:"eip170FBlock,omitempty"`

	// HF: Byzantium
	ByzantiumBlock *big.Int `json:"byzantiumBlock,omitempty"` // Byzantium switch block (nil = no fork, 0 = already on byzantium)
	//
	// Difficulty adjustment to target mean block time including uncles
	// https://github.com/ethereum/EIPs/issues/100
	EIP100FBlock *big.Int `json:"eip100FBlock,omitempty"`
	// Opcode REVERT
	// https://eips.ethereum.org/EIPS/eip-140
	EIP140FBlock *big.Int `json:"eip140FBlock,omitempty"`
	// Precompiled contract for bigint_modexp
	// https://github.com/ethereum/EIPs/issues/198
	EIP198FBlock *big.Int `json:"eip198FBlock,omitempty"`
	// Opcodes RETURNDATACOPY, RETURNDATASIZE
	// https://github.com/ethereum/EIPs/issues/211
	EIP211FBlock *big.Int `json:"eip211FBlock,omitempty"`
	// Precompiled contract for pairing check
	// https://github.com/ethereum/EIPs/issues/212
	EIP212FBlock *big.Int `json:"eip212FBlock,omitempty"`
	// Precompiled contracts for addition and scalar multiplication on the elliptic curve alt_bn128
	// https://github.com/ethereum/EIPs/issues/213
	EIP213FBlock *big.Int `json:"eip213FBlock,omitempty"`
	// Opcode STATICCALL
	// https://github.com/ethereum/EIPs/issues/214
	EIP214FBlock *big.Int `json:"eip214FBlock,omitempty"`
	// Metropolis diff bomb delay and reducing block reward
	// https://github.com/ethereum/EIPs/issues/649
	// note that this is closely related to EIP100.
	// In fact, EIP100 is bundled in
	EIP649FBlock *big.Int `json:"eip649FBlock,omitempty"`
	// Transaction receipt status
	// https://github.com/ethereum/EIPs/issues/658
	EIP658FBlock *big.Int `json:"eip658FBlock,omitempty"`
	// NOT CONFIGURABLE: prevent overwriting contracts
	// https://github.com/ethereum/EIPs/issues/684
	// EIP684FBlock *big.Int `json:"eip684BFlock,omitempty"`

	// HF: Constantinople
	ConstantinopleBlock *big.Int `json:"constantinopleBlock,omitempty"` // Constantinople switch block (nil = no fork, 0 = already activated)
	//
	// Opcodes SHR, SHL, SAR
	// https://eips.ethereum.org/EIPS/eip-145
	EIP145FBlock *big.Int `json:"eip145FBlock,omitempty"`
	// Opcode CREATE2
	// https://eips.ethereum.org/EIPS/eip-1014
	EIP1014FBlock *big.Int `json:"eip1014FBlock,omitempty"`
	// Opcode EXTCODEHASH
	// https://eips.ethereum.org/EIPS/eip-1052
	EIP1052FBlock *big.Int `json:"eip1052FBlock,omitempty"`
	// Constantinople difficulty bomb delay and block reward adjustment
	// https://eips.ethereum.org/EIPS/eip-1234
	EIP1234FBlock *big.Int `json:"eip1234FBlock,omitempty"`
	// Net gas metering
	// https://eips.ethereum.org/EIPS/eip-1283
	EIP1283FBlock *big.Int `json:"eip1283FBlock,omitempty"`

	EWASMBlock *big.Int `json:"ewasmBlock,omitempty"` // EWASM switch block (nil = no fork, 0 = already activated)

	// Various consensus engines
	Ethash *EthashConfig `json:"ethash,omitempty"`
	Clique *CliqueConfig `json:"clique,omitempty"`
}

// EthashConfig is the consensus engine configs for proof-of-work based sealing.
type EthashConfig struct{}

// String implements the stringer interface, returning the consensus engine details.
func (c *EthashConfig) String() string {
	return "ethash"
}

// CliqueConfig is the consensus engine configs for proof-of-authority based sealing.
type CliqueConfig struct {
	Period uint64 `json:"period"` // Number of seconds between blocks to enforce
	Epoch  uint64 `json:"epoch"`  // Epoch length to reset votes and checkpoint
}

// String implements the stringer interface, returning the consensus engine details.
func (c *CliqueConfig) String() string {
	return "clique"
}

// String implements the fmt.Stringer interface.
func (c *ChainConfig) String() string {
	var engine interface{}
	switch {
	case c.Ethash != nil:
		engine = c.Ethash
	case c.Clique != nil:
		engine = c.Clique
	default:
		engine = "unknown"
	}
	return fmt.Sprintf("{ChainID: %v Homestead: %v DAO: %v DAOSupport: %v EIP150: %v EIP155: %v EIP158: %v Byzantium: %v Constantinople: %v Engine: %v}",
		c.ChainID,
		c.HomesteadBlock,
		c.DAOForkBlock,
		c.DAOForkSupport,
		c.EIP150Block,
		c.EIP155Block,
		c.EIP158Block,
		c.ByzantiumBlock,
		c.ConstantinopleBlock,
		engine,
	)
}

// IsHomestead returns whether num is either equal to the homestead block or greater.
func (c *ChainConfig) IsHomestead(num *big.Int) bool {
	return isForked(c.HomesteadBlock, num)
}

// IsEIP7F returns whether num is equal to or greater than the Homestead or EIP7 block.
func (c *ChainConfig) IsEIP7F(num *big.Int) bool {
	return c.IsHomestead(num) || isForked(c.EIP7FBlock, num)
}

// IsDAOFork returns whether num is either equal to the DAO fork block or greater.
func (c *ChainConfig) IsDAOFork(num *big.Int) bool {
	return isForked(c.DAOForkBlock, num)
}

// IsEIP150 returns whether num is either equal to the EIP150 fork block or greater.
func (c *ChainConfig) IsEIP150(num *big.Int) bool {
	return isForked(c.EIP150Block, num)
}

// IsEIP155 returns whether num is either equal to the EIP155 fork block or greater.
func (c *ChainConfig) IsEIP155(num *big.Int) bool {
	return isForked(c.EIP155Block, num)
}

// EIP158HFFBlocks returns the canonical EIP blocks configured for the implemented EIP158HF fork,
// a subset of features introduced at the Spurious Dragon fork.
func (c *ChainConfig) EIP158HFFBlocks() []*big.Int {
	return []*big.Int{
		c.EIP160FBlock,
		c.EIP161FBlock,
		c.EIP170FBlock,
	}
}

// IsEIP158HF returns whether num is either equal to the "EIP158 Hardfork"
// (an implemented-in-code subset of the Spurious Dragon hard-fork) block or greater.
func (c *ChainConfig) IsEIP158HF(num *big.Int) bool {
	return isForked(c.EIP158Block, num) || func(n *big.Int) bool {
		blocks := c.EIP158HFFBlocks()
		for i := range blocks {
			if !isForked(blocks[i], n) {
				return false
			}
		}
		return true
	}(num)
}

// IsEIP160F returns whether num is either equal to or greater than the "EIP158HF" Block or EIP160 block.
func (c *ChainConfig) IsEIP160F(num *big.Int) bool {
	return c.IsEIP158HF(num) || isForked(c.EIP160FBlock, num)
}

// IsEIP161F returns whether num is either equal to or greater than the "EIP158HF" Block or EIP161 block.
func (c *ChainConfig) IsEIP161F(num *big.Int) bool {
	return c.IsEIP158HF(num) || isForked(c.EIP161FBlock, num)
}

// IsEIP170F returns whether num is either equal to or greater than the "EIP158HF" Block or EIP170 block.
func (c *ChainConfig) IsEIP170F(num *big.Int) bool {
	return c.IsEIP158HF(num) || isForked(c.EIP170FBlock, num)
}

//ByzantiumEIPFBlocks returns the canonical EIP blocks configured for the Byzantium Fork.
func (c *ChainConfig) ByzantiumEIPFBlocks() []*big.Int {
	return []*big.Int{
		c.EIP100FBlock,
		c.EIP140FBlock,
		c.EIP198FBlock,
		c.EIP211FBlock,
		c.EIP212FBlock,
		c.EIP213FBlock,
		c.EIP214FBlock,
		c.EIP649FBlock,
		c.EIP658FBlock,
	}
}

// IsByzantium returns whether num is either equal to the Byzantium fork block or greater,
// or whether the configured params satisfy all requirements fulfilling the Byzantium fork.
func (c *ChainConfig) IsByzantium(num *big.Int) bool {
	return isForked(c.ByzantiumBlock, num) || func(n *big.Int) bool {
		blocks := c.ByzantiumEIPFBlocks()
		for i := range blocks {
			if !isForked(blocks[i], n) {
				return false
			}
		}
		return true
	}(num)
}

// IsEIP100F returns whether num is equal to or greater than the Byzantium or EIP100 block.
func (c *ChainConfig) IsEIP100F(num *big.Int) bool {
	return c.IsByzantium(num) || isForked(c.EIP100FBlock, num)
}

// IsEIP140F returns whether num is equal to or greater than the Byzantium or EIP140 block.
func (c *ChainConfig) IsEIP140F(num *big.Int) bool {
	return c.IsByzantium(num) || isForked(c.EIP140FBlock, num)
}

// IsEIP198F returns whether num is equal to or greater than the Byzantium or EIP198 block.
func (c *ChainConfig) IsEIP198F(num *big.Int) bool {
	return c.IsByzantium(num) || isForked(c.EIP198FBlock, num)
}

// IsEIP211F returns whether num is equal to or greater than the Byzantium or EIP211 block.
func (c *ChainConfig) IsEIP211F(num *big.Int) bool {
	return c.IsByzantium(num) || isForked(c.EIP211FBlock, num)
}

// IsEIP212F returns whether num is equal to or greater than the Byzantium or EIP212 block.
func (c *ChainConfig) IsEIP212F(num *big.Int) bool {
	return c.IsByzantium(num) || isForked(c.EIP212FBlock, num)
}

// IsEIP213F returns whether num is equal to or greater than the Byzantium or EIP213 block.
func (c *ChainConfig) IsEIP213F(num *big.Int) bool {
	return c.IsByzantium(num) || isForked(c.EIP213FBlock, num)
}

// IsEIP214F returns whether num is equal to or greater than the Byzantium or EIP214 block.
func (c *ChainConfig) IsEIP214F(num *big.Int) bool {
	return c.IsByzantium(num) || isForked(c.EIP214FBlock, num)
}

// IsEIP649F returns whether num is equal to or greater than the Byzantium or EIP649 block.
func (c *ChainConfig) IsEIP649F(num *big.Int) bool {
	return c.IsByzantium(num) || isForked(c.EIP649FBlock, num)
}

// IsEIP658F returns whether num is equal to or greater than the Byzantium or EIP658 block.
func (c *ChainConfig) IsEIP658F(num *big.Int) bool {
	return c.IsByzantium(num) || isForked(c.EIP658FBlock, num)
}

// ConstantinopleEIPFBlocks returns the canonical blocks configured for the Constantinople Fork.
func (c *ChainConfig) ConstantinopleEIPFBlocks() []*big.Int {
	return []*big.Int{
		c.EIP145FBlock,
		c.EIP1014FBlock,
		c.EIP1052FBlock,
		c.EIP1234FBlock,
		c.EIP1283FBlock,
	}
}

// IsConstantinople returns whether num is either equal to the Constantinople fork block or greater,
// or whether configured params satisfy all requirements fulfilling the Constantinople fork.
func (c *ChainConfig) IsConstantinople(num *big.Int) bool {
	return isForked(c.ConstantinopleBlock, num) || func(n *big.Int) bool {
		blocks := c.ConstantinopleEIPFBlocks()
		for i := range blocks {
			if !isForked(blocks[i], n) {
				return false
			}
		}
		return true
	}(num)
}

// IsEIP145F returns whether num is equal to or greater than the Constantinople or EIP145 block.
func (c *ChainConfig) IsEIP145F(num *big.Int) bool {
	return c.IsConstantinople(num) || isForked(c.EIP145FBlock, num)
}

// IsEIP1014F returns whether num is equal to or greater than the Constantinople or EIP1014 block.
func (c *ChainConfig) IsEIP1014F(num *big.Int) bool {
	return c.IsConstantinople(num) || isForked(c.EIP1014FBlock, num)
}

// IsEIP1052F returns whether num is equal to or greater than the Constantinople or EIP1052 block.
func (c *ChainConfig) IsEIP1052F(num *big.Int) bool {
	return c.IsConstantinople(num) || isForked(c.EIP1052FBlock, num)
}

// IsEIP1234F returns whether num is equal to or greater than the Constantinople or EIP1234 block.
func (c *ChainConfig) IsEIP1234F(num *big.Int) bool {
	return c.IsConstantinople(num) || isForked(c.EIP1234FBlock, num)
}

// IsEIP1283F returns whether num is equal to or greater than the Constantinople or EIP1283 block.
func (c *ChainConfig) IsEIP1283F(num *big.Int) bool {
	return c.IsConstantinople(num) || isForked(c.EIP1283FBlock, num)
}

// IsEWASM returns whether num represents a block number after the EWASM fork
func (c *ChainConfig) IsEWASM(num *big.Int) bool {
	return isForked(c.EWASMBlock, num)
}

// GasTable returns the gas table corresponding to the current phase.
//
// The returned GasTable's fields shouldn't, under any circumstances, be changed.
func (c *ChainConfig) GasTable(num *big.Int) GasTable {
	if num == nil {
		return GasTableHomestead
	}
	switch {
	case c.IsEIP1052F(num):
		return GasTableEIP1052
	case c.IsEIP160F(num):
		return GasTableEIP160
	case c.IsEIP150(num):
		return GasTableEIP150
	default:
		return GasTableHomestead
	}
}

// CheckCompatible checks whether scheduled fork transitions have been imported
// with a mismatching chain configuration.
func (c *ChainConfig) CheckCompatible(newcfg *ChainConfig, height uint64) *ConfigCompatError {
	bhead := new(big.Int).SetUint64(height)

	// Iterate checkCompatible to find the lowest conflict.
	var lasterr *ConfigCompatError
	for {
		err := c.checkCompatible(newcfg, bhead)
		if err == nil || (lasterr != nil && err.RewindTo == lasterr.RewindTo) {
			break
		}
		lasterr = err
		bhead.SetUint64(err.RewindTo)
	}
	return lasterr
}

func (c *ChainConfig) checkCompatible(newcfg *ChainConfig, head *big.Int) *ConfigCompatError {
	for _, ch := range []struct {
		name   string
		c1, c2 *big.Int
	}{
		{"Homestead", c.HomesteadBlock, newcfg.HomesteadBlock},
		{"EIP7F", c.EIP7FBlock, newcfg.EIP7FBlock},
		{"DAO", c.DAOForkBlock, newcfg.DAOForkBlock},
		{"EIP150", c.EIP150Block, newcfg.EIP150Block},
		{"EIP155", c.EIP155Block, newcfg.EIP155Block},
		{"EIP158", c.EIP158Block, newcfg.EIP158Block},
		{"EIP160F", c.EIP160FBlock, newcfg.EIP160FBlock},
		{"EIP161F", c.EIP161FBlock, newcfg.EIP161FBlock},
		{"EIP170F", c.EIP170FBlock, newcfg.EIP170FBlock},
		{"Byzantium", c.ByzantiumBlock, newcfg.ByzantiumBlock},
		{"EIP100F", c.EIP100FBlock, newcfg.EIP100FBlock},
		{"EIP140F", c.EIP140FBlock, newcfg.EIP140FBlock},
		{"EIP198F", c.EIP198FBlock, newcfg.EIP198FBlock},
		{"EIP211F", c.EIP211FBlock, newcfg.EIP211FBlock},
		{"EIP212F", c.EIP212FBlock, newcfg.EIP212FBlock},
		{"EIP213F", c.EIP213FBlock, newcfg.EIP213FBlock},
		{"EIP214F", c.EIP214FBlock, newcfg.EIP214FBlock},
		{"EIP649F", c.EIP649FBlock, newcfg.EIP649FBlock},
		{"EIP658F", c.EIP658FBlock, newcfg.EIP658FBlock},
		{"Constantinople", c.ConstantinopleBlock, newcfg.ConstantinopleBlock},
		{"EIP145F", c.EIP145FBlock, newcfg.EIP145FBlock},
		{"EIP1014F", c.EIP1014FBlock, newcfg.EIP1014FBlock},
		{"EIP1052F", c.EIP1052FBlock, newcfg.EIP1052FBlock},
		{"EIP1234F", c.EIP1234FBlock, newcfg.EIP1234FBlock},
		{"EIP1283F", c.EIP1283FBlock, newcfg.EIP1283FBlock},
		{"EWASM", c.EWASMBlock, newcfg.EWASMBlock},
	} {
		if err := func(c1, c2, head *big.Int) *ConfigCompatError {
			if isForkIncompatible(ch.c1, ch.c2, head) {
				return newCompatError(ch.name+" fork block", ch.c1, ch.c2)
			}
			return nil
		}(ch.c1, ch.c2, head); err != nil {
			return err
		}
	}

	if c.IsDAOFork(head) && c.DAOForkSupport != newcfg.DAOForkSupport {
		return newCompatError("DAO fork support flag", c.DAOForkBlock, newcfg.DAOForkBlock)
	}
	if c.IsEIP155(head) && !configNumEqual(c.ChainID, newcfg.ChainID) {
		return newCompatError("EIP155 chain ID", c.EIP155Block, newcfg.EIP155Block)
	}
	// Either Byzantium block must be set OR EIP100 and EIP649 must be equivalent
	if newcfg.ByzantiumBlock == nil {
		if !configNumEqual(newcfg.EIP100FBlock, newcfg.EIP649FBlock) {
			return newCompatError("EIP100F/EIP649F not equal", newcfg.EIP100FBlock, newcfg.EIP649FBlock)
		}
		if isForkIncompatible(c.EIP100FBlock, newcfg.EIP649FBlock, head) {
			return newCompatError("EIP100F/EIP649F fork block", c.EIP100FBlock, newcfg.EIP649FBlock)
		}
		if isForkIncompatible(c.EIP649FBlock, newcfg.EIP100FBlock, head) {
			return newCompatError("EIP649F/EIP100F fork block", c.EIP649FBlock, newcfg.EIP100FBlock)
		}
	}

	return nil
}

// isForkIncompatible returns true if a fork scheduled at s1 cannot be rescheduled to
// block s2 because head is already past the fork.
func isForkIncompatible(s1, s2, head *big.Int) bool {
	return (isForked(s1, head) || isForked(s2, head)) && !configNumEqual(s1, s2)
}

// isForked returns whether a fork scheduled at block s is active at the given head block.
func isForked(s, head *big.Int) bool {
	if s == nil || head == nil {
		return false
	}
	return s.Cmp(head) <= 0
}

func configNumEqual(x, y *big.Int) bool {
	if x == nil {
		return y == nil
	}
	if y == nil {
		return x == nil
	}
	return x.Cmp(y) == 0
}

// ConfigCompatError is raised if the locally-stored blockchain is initialised with a
// ChainConfig that would alter the past.
type ConfigCompatError struct {
	What string
	// block numbers of the stored and new configurations
	StoredConfig, NewConfig *big.Int
	// the block number to which the local chain must be rewound to correct the error
	RewindTo uint64
}

func newCompatError(what string, storedblock, newblock *big.Int) *ConfigCompatError {
	var rew *big.Int
	switch {
	case storedblock == nil:
		rew = newblock
	case newblock == nil || storedblock.Cmp(newblock) < 0:
		rew = storedblock
	default:
		rew = newblock
	}
	err := &ConfigCompatError{what, storedblock, newblock, 0}
	if rew != nil && rew.Sign() > 0 {
		err.RewindTo = rew.Uint64() - 1
	}
	return err
}

func (err *ConfigCompatError) Error() string {
	return fmt.Sprintf("mismatching %s in database (have %d, want %d, rewindto %d)", err.What, err.StoredConfig, err.NewConfig, err.RewindTo)
}

// Rules wraps ChainConfig and is merely syntactic sugar or can be used for functions
// that do not have or require information about the block.
//
// Rules is a one time interface meaning that it shouldn't be used in between transition
// phases.
type Rules struct {
	ChainID                                                                                                        *big.Int
	IsHomestead, IsEIP7F                                                                                           bool
	IsEIP150                                                                                                       bool
	IsEIP155                                                                                                       bool
	IsEIP158HF, IsEIP160F, IsEIP161F, IsEIP170F                                                                    bool
	IsByzantium, IsEIP100F, IsEIP140F, IsEIP198F, IsEIP211F, IsEIP212F, IsEIP213F, IsEIP214F, IsEIP649F, IsEIP658F bool
	IsConstantinople, IsEIP145F, IsEIP1014F, IsEIP1052F, IsEIP1283F, IsEIP1234F                                    bool
}

// Rules ensures c's ChainID is not nil.
func (c *ChainConfig) Rules(num *big.Int) Rules {
	chainID := c.ChainID
	if chainID == nil {
		chainID = new(big.Int)
	}
	return Rules{
		ChainID: new(big.Int).Set(chainID),

		IsHomestead: c.IsHomestead(num),
		IsEIP7F:     c.IsEIP7F(num),

		IsEIP150:   c.IsEIP150(num),
		IsEIP155:   c.IsEIP155(num),
		IsEIP158HF: c.IsEIP158HF(num),
		IsEIP160F:  c.IsEIP160F(num),
		IsEIP161F:  c.IsEIP161F(num),
		IsEIP170F:  c.IsEIP170F(num),

		IsByzantium: c.IsByzantium(num),
		IsEIP100F:   c.IsEIP100F(num),
		IsEIP140F:   c.IsEIP140F(num),
		IsEIP198F:   c.IsEIP198F(num),
		IsEIP211F:   c.IsEIP211F(num),
		IsEIP212F:   c.IsEIP212F(num),
		IsEIP213F:   c.IsEIP213F(num),
		IsEIP214F:   c.IsEIP214F(num),
		IsEIP649F:   c.IsEIP649F(num),
		IsEIP658F:   c.IsEIP658F(num),

		IsConstantinople: c.IsConstantinople(num),
		IsEIP145F:        c.IsEIP145F(num),
		IsEIP1014F:       c.IsEIP1014F(num),
		IsEIP1052F:       c.IsEIP1052F(num),
		IsEIP1234F:       c.IsEIP1234F(num),
		IsEIP1283F:       c.IsEIP1283F(num),
	}
}
