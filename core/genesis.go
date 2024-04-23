// Copyright 2014 The go-ethereum Authors
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
	"bytes"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"strings"

	"github.com/cryptoecc/ETH-ECC/common"
	"github.com/cryptoecc/ETH-ECC/common/hexutil"
	"github.com/cryptoecc/ETH-ECC/common/math"
	"github.com/cryptoecc/ETH-ECC/core/rawdb"
	"github.com/cryptoecc/ETH-ECC/core/state"
	"github.com/cryptoecc/ETH-ECC/core/types"
	"github.com/cryptoecc/ETH-ECC/crypto"
	"github.com/cryptoecc/ETH-ECC/ethdb"
	"github.com/cryptoecc/ETH-ECC/log"
	"github.com/cryptoecc/ETH-ECC/params"
	"github.com/cryptoecc/ETH-ECC/rlp"
	"github.com/cryptoecc/ETH-ECC/trie"
)

//go:generate go run github.com/fjl/gencodec -type Genesis -field-override genesisSpecMarshaling -out gen_genesis.go
//go:generate go run github.com/fjl/gencodec -type GenesisAccount -field-override genesisAccountMarshaling -out gen_genesis_account.go

var errGenesisNoConfig = errors.New("genesis has no chain configuration")

// Genesis specifies the header fields, state of a genesis block. It also defines hard
// fork switch-over blocks through the chain configuration.
type Genesis struct {
	Config     *params.ChainConfig `json:"config"`
	Nonce      uint64              `json:"nonce"`
	Timestamp  uint64              `json:"timestamp"`
	ExtraData  []byte              `json:"extraData"`
	GasLimit   uint64              `json:"gasLimit"   gencodec:"required"`
	Difficulty *big.Int            `json:"difficulty" gencodec:"required"`
	Mixhash    common.Hash         `json:"mixHash"`
	Coinbase   common.Address      `json:"coinbase"`
	Alloc      GenesisAlloc        `json:"alloc"      gencodec:"required"`

	// These fields are used for consensus tests. Please don't use them
	// in actual genesis blocks.
	Number     uint64      `json:"number"`
	GasUsed    uint64      `json:"gasUsed"`
	ParentHash common.Hash `json:"parentHash"`
	BaseFee    *big.Int    `json:"baseFeePerGas"`
}

// GenesisAlloc specifies the initial state that is part of the genesis block.
type GenesisAlloc map[common.Address]GenesisAccount

func (ga *GenesisAlloc) UnmarshalJSON(data []byte) error {
	m := make(map[common.UnprefixedAddress]GenesisAccount)
	if err := json.Unmarshal(data, &m); err != nil {
		return err
	}
	*ga = make(GenesisAlloc)
	for addr, a := range m {
		(*ga)[common.Address(addr)] = a
	}
	return nil
}

// deriveHash computes the state root according to the genesis specification.
func (ga *GenesisAlloc) deriveHash() (common.Hash, error) {
	// Create an ephemeral in-memory database for computing hash,
	// all the derived states will be discarded to not pollute disk.
	db := state.NewDatabase(rawdb.NewMemoryDatabase())
	statedb, err := state.New(common.Hash{}, db, nil)
	if err != nil {
		return common.Hash{}, err
	}
	for addr, account := range *ga {
		statedb.AddBalance(addr, account.Balance)
		statedb.SetCode(addr, account.Code)
		statedb.SetNonce(addr, account.Nonce)
		for key, value := range account.Storage {
			statedb.SetState(addr, key, value)
		}
	}
	return statedb.Commit(false)
}

// flush is very similar with deriveHash, but the main difference is
// all the generated states will be persisted into the given database.
// Also, the genesis state specification will be flushed as well.
func (ga *GenesisAlloc) flush(db ethdb.Database) error {
	statedb, err := state.New(common.Hash{}, state.NewDatabaseWithConfig(db, &trie.Config{Preimages: true}), nil)
	if err != nil {
		return err
	}
	for addr, account := range *ga {
		statedb.AddBalance(addr, account.Balance)
		statedb.SetCode(addr, account.Code)
		statedb.SetNonce(addr, account.Nonce)
		for key, value := range account.Storage {
			statedb.SetState(addr, key, value)
		}
	}
	root, err := statedb.Commit(false)
	if err != nil {
		return err
	}
	err = statedb.Database().TrieDB().Commit(root, true, nil)
	if err != nil {
		return err
	}
	// Marshal the genesis state specification and persist.
	blob, err := json.Marshal(ga)
	if err != nil {
		return err
	}
	rawdb.WriteGenesisStateSpec(db, root, blob)
	return nil
}

// CommitGenesisState loads the stored genesis state with the given block
// hash and commits them into the given database handler.
func CommitGenesisState(db ethdb.Database, hash common.Hash) error {
	var alloc GenesisAlloc
	blob := rawdb.ReadGenesisStateSpec(db, hash)
	if len(blob) != 0 {
		if err := alloc.UnmarshalJSON(blob); err != nil {
			return err
		}
	} else {
		// Genesis allocation is missing and there are several possibilities:
		// the node is legacy which doesn't persist the genesis allocation or
		// the persisted allocation is just lost.
		// - supported networks(mainnet, testnets), recover with defined allocations
		// - private network, can't recover
		var genesis *Genesis
		switch hash {
		case params.MainnetGenesisHash:
			genesis = DefaultGenesisBlock()
		case params.RopstenGenesisHash:
			genesis = DefaultRopstenGenesisBlock()
		case params.RinkebyGenesisHash:
			genesis = DefaultRinkebyGenesisBlock()
		case params.GoerliGenesisHash:
			genesis = DefaultGoerliGenesisBlock()
		case params.SepoliaGenesisHash:
			genesis = DefaultSepoliaGenesisBlock()
		case params.LveGenesisHash:
			genesis = DefaultLveGenesisBlock()
		case params.SeoulGenesisHash:
			genesis = DefaultSeoulGenesisBlock()
		case params.GwangjuGenesisHash:
			genesis = DefaultGwangjuGenesisBlock()
		}
		if genesis != nil {
			alloc = genesis.Alloc
		} else {
			return errors.New("not found")
		}
	}
	return alloc.flush(db)
}

// GenesisAccount is an account in the state of the genesis block.
type GenesisAccount struct {
	Code       []byte                      `json:"code,omitempty"`
	Storage    map[common.Hash]common.Hash `json:"storage,omitempty"`
	Balance    *big.Int                    `json:"balance" gencodec:"required"`
	Nonce      uint64                      `json:"nonce,omitempty"`
	PrivateKey []byte                      `json:"secretKey,omitempty"` // for tests
}

// field type overrides for gencodec
type genesisSpecMarshaling struct {
	Nonce      math.HexOrDecimal64
	Timestamp  math.HexOrDecimal64
	ExtraData  hexutil.Bytes
	GasLimit   math.HexOrDecimal64
	GasUsed    math.HexOrDecimal64
	Number     math.HexOrDecimal64
	Difficulty *math.HexOrDecimal256
	BaseFee    *math.HexOrDecimal256
	Alloc      map[common.UnprefixedAddress]GenesisAccount
}

type genesisAccountMarshaling struct {
	Code       hexutil.Bytes
	Balance    *math.HexOrDecimal256
	Nonce      math.HexOrDecimal64
	Storage    map[storageJSON]storageJSON
	PrivateKey hexutil.Bytes
}

// storageJSON represents a 256 bit byte array, but allows less than 256 bits when
// unmarshaling from hex.
type storageJSON common.Hash

func (h *storageJSON) UnmarshalText(text []byte) error {
	text = bytes.TrimPrefix(text, []byte("0x"))
	if len(text) > 64 {
		return fmt.Errorf("too many hex characters in storage key/value %q", text)
	}
	offset := len(h) - len(text)/2 // pad on the left
	if _, err := hex.Decode(h[offset:], text); err != nil {
		return fmt.Errorf("invalid hex storage key/value %q", text)
	}
	return nil
}

func (h storageJSON) MarshalText() ([]byte, error) {
	return hexutil.Bytes(h[:]).MarshalText()
}

// GenesisMismatchError is raised when trying to overwrite an existing
// genesis block with an incompatible one.
type GenesisMismatchError struct {
	Stored, New common.Hash
}

func (e *GenesisMismatchError) Error() string {
	return fmt.Sprintf("database contains incompatible genesis (have %x, new %x)", e.Stored, e.New)
}

// ChainOverrides contains the changes to chain config.
type ChainOverrides struct {
	OverrideTerminalTotalDifficulty       *big.Int
	OverrideTerminalTotalDifficultyPassed *bool
}

// SetupGenesisBlock writes or updates the genesis block in db.
// The block that will be used is:
//
//                          genesis == nil       genesis != nil
//                       +------------------------------------------
//     db has no genesis |  main-net default  |  genesis
//     db has genesis    |  from DB           |  genesis (if compatible)
//
// The stored chain configuration will be updated if it is compatible (i.e. does not
// specify a fork block below the local head block). In case of a conflict, the
// error is a *params.ConfigCompatError and the new, unwritten config is returned.
//
// The returned chain configuration is never nil.
func SetupGenesisBlock(db ethdb.Database, genesis *Genesis) (*params.ChainConfig, common.Hash, error) {
	return SetupGenesisBlockWithOverride(db, genesis, nil)
}

func SetupGenesisBlockWithOverride(db ethdb.Database, genesis *Genesis, overrides *ChainOverrides) (*params.ChainConfig, common.Hash, error) {
	if genesis != nil && genesis.Config == nil {
		return params.AllEthashProtocolChanges, common.Hash{}, errGenesisNoConfig
	}

	applyOverrides := func(config *params.ChainConfig) {
		if config != nil {
			if overrides != nil && overrides.OverrideTerminalTotalDifficulty != nil {
				config.TerminalTotalDifficulty = overrides.OverrideTerminalTotalDifficulty
			}
			if overrides != nil && overrides.OverrideTerminalTotalDifficultyPassed != nil {
				config.TerminalTotalDifficultyPassed = *overrides.OverrideTerminalTotalDifficultyPassed
			}
		}
	}

	// Just commit the new block if there is no stored genesis block.
	stored := rawdb.ReadCanonicalHash(db, 0)
	if (stored == common.Hash{}) {
		if genesis == nil {
			log.Info("Writing default main-net genesis block")
			genesis = DefaultGenesisBlock()
		} else {
			log.Info("Writing custom genesis block")
		}
		block, err := genesis.Commit(db)
		if err != nil {
			return genesis.Config, common.Hash{}, err
		}
		applyOverrides(genesis.Config)
		return genesis.Config, block.Hash(), nil
	}
	// We have the genesis block in database(perhaps in ancient database)
	// but the corresponding state is missing.
	header := rawdb.ReadHeader(db, stored, 0)
	if _, err := state.New(header.Root, state.NewDatabaseWithConfig(db, nil), nil); err != nil {
		if genesis == nil {
			genesis = DefaultGenesisBlock()
		}
		// Ensure the stored genesis matches with the given one.
		hash := genesis.ToBlock().Hash()
		if hash != stored {
			return genesis.Config, hash, &GenesisMismatchError{stored, hash}
		}
		block, err := genesis.Commit(db)
		if err != nil {
			return genesis.Config, hash, err
		}
		applyOverrides(genesis.Config)
		return genesis.Config, block.Hash(), nil
	}
	// Check whether the genesis block is already written.
	if genesis != nil {
		hash := genesis.ToBlock().Hash()
		if hash != stored {
			return genesis.Config, hash, &GenesisMismatchError{stored, hash}
		}
	}
	// Get the existing chain configuration.
	newcfg := genesis.configOrDefault(stored)
	applyOverrides(newcfg)
	if err := newcfg.CheckConfigForkOrder(); err != nil {
		return newcfg, common.Hash{}, err
	}
	storedcfg := rawdb.ReadChainConfig(db, stored)
	if storedcfg == nil {
		log.Warn("Found genesis block without chain config")
		rawdb.WriteChainConfig(db, stored, newcfg)
		return newcfg, stored, nil
	}
	// Special case: if a private network is being used (no genesis and also no
	// mainnet hash in the database), we must not apply the `configOrDefault`
	// chain config as that would be AllProtocolChanges (applying any new fork
	// on top of an existing private network genesis block). In that case, only
	// apply the overrides.
	if genesis == nil && stored != params.MainnetGenesisHash {
		newcfg = storedcfg
		applyOverrides(newcfg)
	}
	// Check config compatibility and write the config. Compatibility errors
	// are returned to the caller unless we're already at block zero.
	height := rawdb.ReadHeaderNumber(db, rawdb.ReadHeadHeaderHash(db))
	if height == nil {
		return newcfg, stored, fmt.Errorf("missing block number for head header hash")
	}
	compatErr := storedcfg.CheckCompatible(newcfg, *height)
	if compatErr != nil && *height != 0 && compatErr.RewindTo != 0 {
		return newcfg, stored, compatErr
	}
	rawdb.WriteChainConfig(db, stored, newcfg)
	return newcfg, stored, nil
}

// LoadCliqueConfig loads the stored clique config if the chain config
// is already present in database, otherwise, return the config in the
// provided genesis specification. Note the returned clique config can
// be nil if we are not in the clique network.
func LoadCliqueConfig(db ethdb.Database, genesis *Genesis) (*params.CliqueConfig, error) {
	// Load the stored chain config from the database. It can be nil
	// in case the database is empty. Notably, we only care about the
	// chain config corresponds to the canonical chain.
	stored := rawdb.ReadCanonicalHash(db, 0)
	if stored != (common.Hash{}) {
		storedcfg := rawdb.ReadChainConfig(db, stored)
		if storedcfg != nil {
			return storedcfg.Clique, nil
		}
	}
	// Load the clique config from the provided genesis specification.
	if genesis != nil {
		// Reject invalid genesis spec without valid chain config
		if genesis.Config == nil {
			return nil, errGenesisNoConfig
		}
		// If the canonical genesis header is present, but the chain
		// config is missing(initialize the empty leveldb with an
		// external ancient chain segment), ensure the provided genesis
		// is matched.
		if stored != (common.Hash{}) && genesis.ToBlock().Hash() != stored {
			return nil, &GenesisMismatchError{stored, genesis.ToBlock().Hash()}
		}
		return genesis.Config.Clique, nil
	}
	// There is no stored chain config and no new config provided,
	// In this case the default chain config(mainnet) will be used,
	// namely ethash is the specified consensus engine, return nil.
	return nil, nil
}

func LoadEccpowConfig(db ethdb.Database, genesis *Genesis) (*params.EccpowConfig, error) {
	// Load the stored chain config from the database. It can be nil
	// in case the database is empty. Notably, we only care about the
	// chain config corresponds to the canonical chain.
	stored := rawdb.ReadCanonicalHash(db, 0)
	if stored != (common.Hash{}) {
		storedcfg := rawdb.ReadChainConfig(db, stored)
		if storedcfg != nil {
			return storedcfg.Eccpow, nil
		}
	}
	// Load the clique config from the provided genesis specification.
	if genesis != nil {
		// Reject invalid genesis spec without valid chain config
		if genesis.Config == nil {
			return nil, errGenesisNoConfig
		}
		// If the canonical genesis header is present, but the chain
		// config is missing(initialize the empty leveldb with an
		// external ancient chain segment), ensure the provided genesis
		// is matched.
		if stored != (common.Hash{}) && genesis.ToBlock().Hash() != stored {
			return nil, &GenesisMismatchError{stored, genesis.ToBlock().Hash()}
		}
		return genesis.Config.Eccpow, nil
	}
	// There is no stored chain config and no new config provided,
	// In this case the default chain config(mainnet) will be used,
	// namely ethash is the specified consensus engine, return nil.
	return nil, nil
}



func (g *Genesis) configOrDefault(ghash common.Hash) *params.ChainConfig {
	switch {
	case g != nil:
		return g.Config
	case ghash == params.MainnetGenesisHash:
		return params.MainnetChainConfig
	case ghash == params.RopstenGenesisHash:
		return params.RopstenChainConfig
	case ghash == params.SepoliaGenesisHash:
		return params.SepoliaChainConfig
	case ghash == params.RinkebyGenesisHash:
		return params.RinkebyChainConfig
	case ghash == params.GoerliGenesisHash:
		return params.GoerliChainConfig
	case ghash == params.KilnGenesisHash:
		return DefaultKilnGenesisBlock().Config
	case ghash == params.LveGenesisHash:
		return params.LveChainConfig
	case ghash == params.SeoulGenesisHash:
		return params.SeoulChainConfig
	case ghash == params.GwangjuGenesisHash:
		return params.GwangjuChainConfig
	default:
		return params.AllEthashProtocolChanges
	}
}

// ToBlock returns the genesis block according to genesis specification.
func (g *Genesis) ToBlock() *types.Block {
	root, err := g.Alloc.deriveHash()
	if err != nil {
		panic(err)
	}
	head := &types.Header{
		Number:     new(big.Int).SetUint64(g.Number),
		Nonce:      types.EncodeNonce(g.Nonce),
		Time:       g.Timestamp,
		ParentHash: g.ParentHash,
		Extra:      g.ExtraData,
		GasLimit:   g.GasLimit,
		GasUsed:    g.GasUsed,
		BaseFee:    g.BaseFee,
		Difficulty: g.Difficulty,
		MixDigest:  g.Mixhash,
		Coinbase:   g.Coinbase,
		Root:       root,
	}
	if g.GasLimit == 0 {
		head.GasLimit = params.GenesisGasLimit
	}
	if g.Difficulty == nil && g.Mixhash == (common.Hash{}) {
		head.Difficulty = params.GenesisDifficulty
	}
	if g.Config != nil && g.Config.IsLondon(common.Big0) {
		if g.BaseFee != nil {
			head.BaseFee = g.BaseFee
		} else {
			head.BaseFee = new(big.Int).SetUint64(params.InitialBaseFee)
		}
	}
	return types.NewBlock(head, nil, nil, nil, trie.NewStackTrie(nil))
}

// Commit writes the block and state of a genesis specification to the database.
// The block is committed as the canonical head block.
func (g *Genesis) Commit(db ethdb.Database) (*types.Block, error) {
	block := g.ToBlock()
	if block.Number().Sign() != 0 {
		return nil, errors.New("can't commit genesis block with number > 0")
	}
	config := g.Config
	if config == nil {
		config = params.AllEthashProtocolChanges
	}
	if err := config.CheckConfigForkOrder(); err != nil {
		return nil, err
	}
	if config.Clique != nil && len(block.Extra()) < 32+crypto.SignatureLength {
		return nil, errors.New("can't start clique chain without signers")
	}
	// All the checks has passed, flush the states derived from the genesis
	// specification as well as the specification itself into the provided
	// database.
	if err := g.Alloc.flush(db); err != nil {
		return nil, err
	}
	rawdb.WriteTd(db, block.Hash(), block.NumberU64(), block.Difficulty())
	rawdb.WriteBlock(db, block)
	rawdb.WriteReceipts(db, block.Hash(), block.NumberU64(), nil)
	rawdb.WriteCanonicalHash(db, block.Hash(), block.NumberU64())
	rawdb.WriteHeadBlockHash(db, block.Hash())
	rawdb.WriteHeadFastBlockHash(db, block.Hash())
	rawdb.WriteHeadHeaderHash(db, block.Hash())
	rawdb.WriteChainConfig(db, block.Hash(), config)
	return block, nil
}

// MustCommit writes the genesis block and state to db, panicking on error.
// The block is committed as the canonical head block.
func (g *Genesis) MustCommit(db ethdb.Database) *types.Block {
	block, err := g.Commit(db)
	if err != nil {
		panic(err)
	}
	return block
}

// DefaultGenesisBlock returns the Ethereum main net genesis block.
func DefaultGenesisBlock() *Genesis {
	return &Genesis{
		Config:     params.MainnetChainConfig,
		Nonce:      66,
		ExtraData:  hexutil.MustDecode("0x11bbe8db4e347b4e8c937c1c8370e4b5ed33adb3db69cbdb7a38e1e50b1b82fa"),
		GasLimit:   5000,
		Difficulty: big.NewInt(17179869184),
		Alloc:      decodePrealloc(mainnetAllocData),
	}
}

// DefaultRopstenGenesisBlock returns the Ropsten network genesis block.
func DefaultRopstenGenesisBlock() *Genesis {
	return &Genesis{
		Config:     params.RopstenChainConfig,
		Nonce:      66,
		ExtraData:  hexutil.MustDecode("0x3535353535353535353535353535353535353535353535353535353535353535"),
		GasLimit:   16777216,
		Difficulty: big.NewInt(1048576),
		Alloc:      decodePrealloc(ropstenAllocData),
	}
}

// DefaultRinkebyGenesisBlock returns the Rinkeby network genesis block.
func DefaultRinkebyGenesisBlock() *Genesis {
	return &Genesis{
		Config:     params.RinkebyChainConfig,
		Timestamp:  1492009146,
		ExtraData:  hexutil.MustDecode("0x52657370656374206d7920617574686f7269746168207e452e436172746d616e42eb768f2244c8811c63729a21a3569731535f067ffc57839b00206d1ad20c69a1981b489f772031b279182d99e65703f0076e4812653aab85fca0f00000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000"),
		GasLimit:   4700000,
		Difficulty: big.NewInt(1),
		Alloc:      decodePrealloc(rinkebyAllocData),
	}
}

// DefaultGoerliGenesisBlock returns the Görli network genesis block.
func DefaultGoerliGenesisBlock() *Genesis {
	return &Genesis{
		Config:     params.GoerliChainConfig,
		Timestamp:  1548854791,
		ExtraData:  hexutil.MustDecode("0x22466c6578692069732061207468696e6722202d204166726900000000000000e0a2bd4258d2768837baa26a28fe71dc079f84c70000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000"),
		GasLimit:   10485760,
		Difficulty: big.NewInt(1),
		Alloc:      decodePrealloc(goerliAllocData),
	}
}

// DefaultSepoliaGenesisBlock returns the Sepolia network genesis block.
func DefaultSepoliaGenesisBlock() *Genesis {
	return &Genesis{
		Config:     params.SepoliaChainConfig,
		Nonce:      0,
		ExtraData:  []byte("Sepolia, Athens, Attica, Greece!"),
		GasLimit:   0x1c9c380,
		Difficulty: big.NewInt(0x20000),
		Timestamp:  1633267481,
		Alloc:      decodePrealloc(sepoliaAllocData),
	}
}

// DefaultKilnGenesisBlock returns the kiln network genesis block.
func DefaultKilnGenesisBlock() *Genesis {
	g := new(Genesis)
	reader := strings.NewReader(KilnAllocData)
	if err := json.NewDecoder(reader).Decode(g); err != nil {
		panic(err)
	}
	return g
}

// DefaultlveGenesisBlock returns the LVE network genesis block.
//change!!
func DefaultLveGenesisBlock() *Genesis {
	return &Genesis{
		Config:     params.LveChainConfig,
		Nonce:      0,
		Timestamp:  1651123670,
		ExtraData:  hexutil.MustDecode("0x0000000000000000000000000000000000000000000000000000000000000000"),
		GasLimit:   4700000,
		Difficulty: big.NewInt(524288),
		Mixhash:    common.HexToHash("0x0000000000000000000000000000000000000000000000000000000000000000"),
		Coinbase:   common.HexToAddress("0x0000000000000000000000000000000000000000"),
		Number:     0,
		GasUsed:    0,
		ParentHash: common.HexToHash("0x0000000000000000000000000000000000000000000000000000000000000000"),
		Alloc: map[common.Address]GenesisAccount{
			common.BytesToAddress([]byte{0}):   {Balance: big.NewInt(1)},
			common.BytesToAddress([]byte{1}):   {Balance: big.NewInt(1)}, // ECRecover
			common.BytesToAddress([]byte{2}):   {Balance: big.NewInt(1)}, // SHA256
			common.BytesToAddress([]byte{3}):   {Balance: big.NewInt(1)}, // RIPEMD
			common.BytesToAddress([]byte{4}):   {Balance: big.NewInt(1)}, // Identity
			common.BytesToAddress([]byte{5}):   {Balance: big.NewInt(1)}, // ModExp
			common.BytesToAddress([]byte{6}):   {Balance: big.NewInt(1)}, // ECAdd
			common.BytesToAddress([]byte{7}):   {Balance: big.NewInt(1)}, // ECScalarMul
			common.BytesToAddress([]byte{8}):   {Balance: big.NewInt(1)}, // ECPairing
			common.BytesToAddress([]byte{8}):   {Balance: big.NewInt(1)},
			common.BytesToAddress([]byte{9}):   {Balance: big.NewInt(1)},
			common.BytesToAddress([]byte{10}):  {Balance: big.NewInt(1)},
			common.BytesToAddress([]byte{11}):  {Balance: big.NewInt(1)},
			common.BytesToAddress([]byte{12}):  {Balance: big.NewInt(1)},
			common.BytesToAddress([]byte{13}):  {Balance: big.NewInt(1)},
			common.BytesToAddress([]byte{14}):  {Balance: big.NewInt(1)},
			common.BytesToAddress([]byte{15}):  {Balance: big.NewInt(1)},
			common.BytesToAddress([]byte{16}):  {Balance: big.NewInt(1)},
			common.BytesToAddress([]byte{17}):  {Balance: big.NewInt(1)},
			common.BytesToAddress([]byte{18}):  {Balance: big.NewInt(1)},
			common.BytesToAddress([]byte{19}):  {Balance: big.NewInt(1)},
			common.BytesToAddress([]byte{20}):  {Balance: big.NewInt(1)},
			common.BytesToAddress([]byte{21}):  {Balance: big.NewInt(1)},
			common.BytesToAddress([]byte{22}):  {Balance: big.NewInt(1)},
			common.BytesToAddress([]byte{23}):  {Balance: big.NewInt(1)},
			common.BytesToAddress([]byte{24}):  {Balance: big.NewInt(1)},
			common.BytesToAddress([]byte{25}):  {Balance: big.NewInt(1)},
			common.BytesToAddress([]byte{26}):  {Balance: big.NewInt(1)},
			common.BytesToAddress([]byte{27}):  {Balance: big.NewInt(1)},
			common.BytesToAddress([]byte{28}):  {Balance: big.NewInt(1)},
			common.BytesToAddress([]byte{29}):  {Balance: big.NewInt(1)},
			common.BytesToAddress([]byte{30}):  {Balance: big.NewInt(1)},
			common.BytesToAddress([]byte{31}):  {Balance: big.NewInt(1)},
			common.BytesToAddress([]byte{32}):  {Balance: big.NewInt(1)},
			common.BytesToAddress([]byte{33}):  {Balance: big.NewInt(1)},
			common.BytesToAddress([]byte{34}):  {Balance: big.NewInt(1)},
			common.BytesToAddress([]byte{35}):  {Balance: big.NewInt(1)},
			common.BytesToAddress([]byte{36}):  {Balance: big.NewInt(1)},
			common.BytesToAddress([]byte{37}):  {Balance: big.NewInt(1)},
			common.BytesToAddress([]byte{38}):  {Balance: big.NewInt(1)},
			common.BytesToAddress([]byte{39}):  {Balance: big.NewInt(1)},
			common.BytesToAddress([]byte{40}):  {Balance: big.NewInt(1)},
			common.BytesToAddress([]byte{41}):  {Balance: big.NewInt(1)},
			common.BytesToAddress([]byte{42}):  {Balance: big.NewInt(1)},
			common.BytesToAddress([]byte{43}):  {Balance: big.NewInt(1)},
			common.BytesToAddress([]byte{44}):  {Balance: big.NewInt(1)},
			common.BytesToAddress([]byte{45}):  {Balance: big.NewInt(1)},
			common.BytesToAddress([]byte{46}):  {Balance: big.NewInt(1)},
			common.BytesToAddress([]byte{47}):  {Balance: big.NewInt(1)},
			common.BytesToAddress([]byte{48}):  {Balance: big.NewInt(1)},
			common.BytesToAddress([]byte{49}):  {Balance: big.NewInt(1)},
			common.BytesToAddress([]byte{50}):  {Balance: big.NewInt(1)},
			common.BytesToAddress([]byte{51}):  {Balance: big.NewInt(1)},
			common.BytesToAddress([]byte{52}):  {Balance: big.NewInt(1)},
			common.BytesToAddress([]byte{53}):  {Balance: big.NewInt(1)},
			common.BytesToAddress([]byte{54}):  {Balance: big.NewInt(1)},
			common.BytesToAddress([]byte{55}):  {Balance: big.NewInt(1)},
			common.BytesToAddress([]byte{56}):  {Balance: big.NewInt(1)},
			common.BytesToAddress([]byte{57}):  {Balance: big.NewInt(1)},
			common.BytesToAddress([]byte{58}):  {Balance: big.NewInt(1)},
			common.BytesToAddress([]byte{59}):  {Balance: big.NewInt(1)},
			common.BytesToAddress([]byte{60}):  {Balance: big.NewInt(1)},
			common.BytesToAddress([]byte{61}):  {Balance: big.NewInt(1)},
			common.BytesToAddress([]byte{62}):  {Balance: big.NewInt(1)},
			common.BytesToAddress([]byte{63}):  {Balance: big.NewInt(1)},
			common.BytesToAddress([]byte{64}):  {Balance: big.NewInt(1)},
			common.BytesToAddress([]byte{65}):  {Balance: big.NewInt(1)},
			common.BytesToAddress([]byte{66}):  {Balance: big.NewInt(1)},
			common.BytesToAddress([]byte{67}):  {Balance: big.NewInt(1)},
			common.BytesToAddress([]byte{68}):  {Balance: big.NewInt(1)},
			common.BytesToAddress([]byte{69}):  {Balance: big.NewInt(1)},
			common.BytesToAddress([]byte{70}):  {Balance: big.NewInt(1)},
			common.BytesToAddress([]byte{71}):  {Balance: big.NewInt(1)},
			common.BytesToAddress([]byte{72}):  {Balance: big.NewInt(1)},
			common.BytesToAddress([]byte{73}):  {Balance: big.NewInt(1)},
			common.BytesToAddress([]byte{74}):  {Balance: big.NewInt(1)},
			common.BytesToAddress([]byte{75}):  {Balance: big.NewInt(1)},
			common.BytesToAddress([]byte{76}):  {Balance: big.NewInt(1)},
			common.BytesToAddress([]byte{77}):  {Balance: big.NewInt(1)},
			common.BytesToAddress([]byte{78}):  {Balance: big.NewInt(1)},
			common.BytesToAddress([]byte{79}):  {Balance: big.NewInt(1)},
			common.BytesToAddress([]byte{80}):  {Balance: big.NewInt(1)},
			common.BytesToAddress([]byte{81}):  {Balance: big.NewInt(1)},
			common.BytesToAddress([]byte{82}):  {Balance: big.NewInt(1)},
			common.BytesToAddress([]byte{83}):  {Balance: big.NewInt(1)},
			common.BytesToAddress([]byte{84}):  {Balance: big.NewInt(1)},
			common.BytesToAddress([]byte{85}):  {Balance: big.NewInt(1)},
			common.BytesToAddress([]byte{86}):  {Balance: big.NewInt(1)},
			common.BytesToAddress([]byte{87}):  {Balance: big.NewInt(1)},
			common.BytesToAddress([]byte{88}):  {Balance: big.NewInt(1)},
			common.BytesToAddress([]byte{89}):  {Balance: big.NewInt(1)},
			common.BytesToAddress([]byte{90}):  {Balance: big.NewInt(1)},
			common.BytesToAddress([]byte{91}):  {Balance: big.NewInt(1)},
			common.BytesToAddress([]byte{92}):  {Balance: big.NewInt(1)},
			common.BytesToAddress([]byte{93}):  {Balance: big.NewInt(1)},
			common.BytesToAddress([]byte{94}):  {Balance: big.NewInt(1)},
			common.BytesToAddress([]byte{95}):  {Balance: big.NewInt(1)},
			common.BytesToAddress([]byte{96}):  {Balance: big.NewInt(1)},
			common.BytesToAddress([]byte{97}):  {Balance: big.NewInt(1)},
			common.BytesToAddress([]byte{98}):  {Balance: big.NewInt(1)},
			common.BytesToAddress([]byte{99}):  {Balance: big.NewInt(1)},
			common.BytesToAddress([]byte{100}): {Balance: big.NewInt(1)},
			common.BytesToAddress([]byte{101}): {Balance: big.NewInt(1)},
			common.BytesToAddress([]byte{102}): {Balance: big.NewInt(1)},
			common.BytesToAddress([]byte{103}): {Balance: big.NewInt(1)},
			common.BytesToAddress([]byte{104}): {Balance: big.NewInt(1)},
			common.BytesToAddress([]byte{105}): {Balance: big.NewInt(1)},
			common.BytesToAddress([]byte{106}): {Balance: big.NewInt(1)},
			common.BytesToAddress([]byte{107}): {Balance: big.NewInt(1)},
			common.BytesToAddress([]byte{108}): {Balance: big.NewInt(1)},
			common.BytesToAddress([]byte{109}): {Balance: big.NewInt(1)},
			common.BytesToAddress([]byte{110}): {Balance: big.NewInt(1)},
			common.BytesToAddress([]byte{111}): {Balance: big.NewInt(1)},
			common.BytesToAddress([]byte{112}): {Balance: big.NewInt(1)},
			common.BytesToAddress([]byte{113}): {Balance: big.NewInt(1)},
			common.BytesToAddress([]byte{114}): {Balance: big.NewInt(1)},
			common.BytesToAddress([]byte{115}): {Balance: big.NewInt(1)},
			common.BytesToAddress([]byte{116}): {Balance: big.NewInt(1)},
			common.BytesToAddress([]byte{117}): {Balance: big.NewInt(1)},
			common.BytesToAddress([]byte{118}): {Balance: big.NewInt(1)},
			common.BytesToAddress([]byte{119}): {Balance: big.NewInt(1)},
			common.BytesToAddress([]byte{120}): {Balance: big.NewInt(1)},
			common.BytesToAddress([]byte{121}): {Balance: big.NewInt(1)},
			common.BytesToAddress([]byte{122}): {Balance: big.NewInt(1)},
			common.BytesToAddress([]byte{123}): {Balance: big.NewInt(1)},
			common.BytesToAddress([]byte{124}): {Balance: big.NewInt(1)},
			common.BytesToAddress([]byte{125}): {Balance: big.NewInt(1)},
			common.BytesToAddress([]byte{126}): {Balance: big.NewInt(1)},
			common.BytesToAddress([]byte{127}): {Balance: big.NewInt(1)},
			common.BytesToAddress([]byte{128}): {Balance: big.NewInt(1)},
			common.BytesToAddress([]byte{129}): {Balance: big.NewInt(1)},
			common.BytesToAddress([]byte{130}): {Balance: big.NewInt(1)},
			common.BytesToAddress([]byte{131}): {Balance: big.NewInt(1)},
			common.BytesToAddress([]byte{132}): {Balance: big.NewInt(1)},
			common.BytesToAddress([]byte{133}): {Balance: big.NewInt(1)},
			common.BytesToAddress([]byte{134}): {Balance: big.NewInt(1)},
			common.BytesToAddress([]byte{135}): {Balance: big.NewInt(1)},
			common.BytesToAddress([]byte{136}): {Balance: big.NewInt(1)},
			common.BytesToAddress([]byte{137}): {Balance: big.NewInt(1)},
			common.BytesToAddress([]byte{138}): {Balance: big.NewInt(1)},
			common.BytesToAddress([]byte{139}): {Balance: big.NewInt(1)},
			common.BytesToAddress([]byte{140}): {Balance: big.NewInt(1)},
			common.BytesToAddress([]byte{141}): {Balance: big.NewInt(1)},
			common.BytesToAddress([]byte{142}): {Balance: big.NewInt(1)},
			common.BytesToAddress([]byte{143}): {Balance: big.NewInt(1)},
			common.BytesToAddress([]byte{144}): {Balance: big.NewInt(1)},
			common.BytesToAddress([]byte{145}): {Balance: big.NewInt(1)},
			common.BytesToAddress([]byte{146}): {Balance: big.NewInt(1)},
			common.BytesToAddress([]byte{147}): {Balance: big.NewInt(1)},
			common.BytesToAddress([]byte{148}): {Balance: big.NewInt(1)},
			common.BytesToAddress([]byte{149}): {Balance: big.NewInt(1)},
			common.BytesToAddress([]byte{150}): {Balance: big.NewInt(1)},
			common.BytesToAddress([]byte{151}): {Balance: big.NewInt(1)},
			common.BytesToAddress([]byte{152}): {Balance: big.NewInt(1)},
			common.BytesToAddress([]byte{153}): {Balance: big.NewInt(1)},
			common.BytesToAddress([]byte{154}): {Balance: big.NewInt(1)},
			common.BytesToAddress([]byte{155}): {Balance: big.NewInt(1)},
			common.BytesToAddress([]byte{156}): {Balance: big.NewInt(1)},
			common.BytesToAddress([]byte{157}): {Balance: big.NewInt(1)},
			common.BytesToAddress([]byte{158}): {Balance: big.NewInt(1)},
			common.BytesToAddress([]byte{159}): {Balance: big.NewInt(1)},
			common.BytesToAddress([]byte{160}): {Balance: big.NewInt(1)},
			common.BytesToAddress([]byte{161}): {Balance: big.NewInt(1)},
			common.BytesToAddress([]byte{162}): {Balance: big.NewInt(1)},
			common.BytesToAddress([]byte{163}): {Balance: big.NewInt(1)},
			common.BytesToAddress([]byte{164}): {Balance: big.NewInt(1)},
			common.BytesToAddress([]byte{165}): {Balance: big.NewInt(1)},
			common.BytesToAddress([]byte{166}): {Balance: big.NewInt(1)},
			common.BytesToAddress([]byte{167}): {Balance: big.NewInt(1)},
			common.BytesToAddress([]byte{168}): {Balance: big.NewInt(1)},
			common.BytesToAddress([]byte{169}): {Balance: big.NewInt(1)},
			common.BytesToAddress([]byte{170}): {Balance: big.NewInt(1)},
			common.BytesToAddress([]byte{171}): {Balance: big.NewInt(1)},
			common.BytesToAddress([]byte{172}): {Balance: big.NewInt(1)},
			common.BytesToAddress([]byte{173}): {Balance: big.NewInt(1)},
			common.BytesToAddress([]byte{174}): {Balance: big.NewInt(1)},
			common.BytesToAddress([]byte{175}): {Balance: big.NewInt(1)},
			common.BytesToAddress([]byte{176}): {Balance: big.NewInt(1)},
			common.BytesToAddress([]byte{177}): {Balance: big.NewInt(1)},
			common.BytesToAddress([]byte{178}): {Balance: big.NewInt(1)},
			common.BytesToAddress([]byte{179}): {Balance: big.NewInt(1)},
			common.BytesToAddress([]byte{180}): {Balance: big.NewInt(1)},
			common.BytesToAddress([]byte{181}): {Balance: big.NewInt(1)},
			common.BytesToAddress([]byte{182}): {Balance: big.NewInt(1)},
			common.BytesToAddress([]byte{183}): {Balance: big.NewInt(1)},
			common.BytesToAddress([]byte{184}): {Balance: big.NewInt(1)},
			common.BytesToAddress([]byte{185}): {Balance: big.NewInt(1)},
			common.BytesToAddress([]byte{186}): {Balance: big.NewInt(1)},
			common.BytesToAddress([]byte{187}): {Balance: big.NewInt(1)},
			common.BytesToAddress([]byte{188}): {Balance: big.NewInt(1)},
			common.BytesToAddress([]byte{189}): {Balance: big.NewInt(1)},
			common.BytesToAddress([]byte{190}): {Balance: big.NewInt(1)},
			common.BytesToAddress([]byte{191}): {Balance: big.NewInt(1)},
			common.BytesToAddress([]byte{192}): {Balance: big.NewInt(1)},
			common.BytesToAddress([]byte{193}): {Balance: big.NewInt(1)},
			common.BytesToAddress([]byte{194}): {Balance: big.NewInt(1)},
			common.BytesToAddress([]byte{195}): {Balance: big.NewInt(1)},
			common.BytesToAddress([]byte{196}): {Balance: big.NewInt(1)},
			common.BytesToAddress([]byte{197}): {Balance: big.NewInt(1)},
			common.BytesToAddress([]byte{198}): {Balance: big.NewInt(1)},
			common.BytesToAddress([]byte{199}): {Balance: big.NewInt(1)},
			common.BytesToAddress([]byte{200}): {Balance: big.NewInt(1)},
			common.BytesToAddress([]byte{201}): {Balance: big.NewInt(1)},
			common.BytesToAddress([]byte{202}): {Balance: big.NewInt(1)},
			common.BytesToAddress([]byte{203}): {Balance: big.NewInt(1)},
			common.BytesToAddress([]byte{204}): {Balance: big.NewInt(1)},
			common.BytesToAddress([]byte{205}): {Balance: big.NewInt(1)},
			common.BytesToAddress([]byte{206}): {Balance: big.NewInt(1)},
			common.BytesToAddress([]byte{207}): {Balance: big.NewInt(1)},
			common.BytesToAddress([]byte{208}): {Balance: big.NewInt(1)},
			common.BytesToAddress([]byte{209}): {Balance: big.NewInt(1)},
			common.BytesToAddress([]byte{210}): {Balance: big.NewInt(1)},
			common.BytesToAddress([]byte{211}): {Balance: big.NewInt(1)},
			common.BytesToAddress([]byte{212}): {Balance: big.NewInt(1)},
			common.BytesToAddress([]byte{213}): {Balance: big.NewInt(1)},
			common.BytesToAddress([]byte{214}): {Balance: big.NewInt(1)},
			common.BytesToAddress([]byte{215}): {Balance: big.NewInt(1)},
			common.BytesToAddress([]byte{216}): {Balance: big.NewInt(1)},
			common.BytesToAddress([]byte{217}): {Balance: big.NewInt(1)},
			common.BytesToAddress([]byte{218}): {Balance: big.NewInt(1)},
			common.BytesToAddress([]byte{219}): {Balance: big.NewInt(1)},
			common.BytesToAddress([]byte{220}): {Balance: big.NewInt(1)},
			common.BytesToAddress([]byte{221}): {Balance: big.NewInt(1)},
			common.BytesToAddress([]byte{222}): {Balance: big.NewInt(1)},
			common.BytesToAddress([]byte{223}): {Balance: big.NewInt(1)},
			common.BytesToAddress([]byte{224}): {Balance: big.NewInt(1)},
			common.BytesToAddress([]byte{225}): {Balance: big.NewInt(1)},
			common.BytesToAddress([]byte{226}): {Balance: big.NewInt(1)},
			common.BytesToAddress([]byte{227}): {Balance: big.NewInt(1)},
			common.BytesToAddress([]byte{228}): {Balance: big.NewInt(1)},
			common.BytesToAddress([]byte{229}): {Balance: big.NewInt(1)},
			common.BytesToAddress([]byte{230}): {Balance: big.NewInt(1)},
			common.BytesToAddress([]byte{231}): {Balance: big.NewInt(1)},
			common.BytesToAddress([]byte{232}): {Balance: big.NewInt(1)},
			common.BytesToAddress([]byte{233}): {Balance: big.NewInt(1)},
			common.BytesToAddress([]byte{234}): {Balance: big.NewInt(1)},
			common.BytesToAddress([]byte{235}): {Balance: big.NewInt(1)},
			common.BytesToAddress([]byte{236}): {Balance: big.NewInt(1)},
			common.BytesToAddress([]byte{237}): {Balance: big.NewInt(1)},
			common.BytesToAddress([]byte{238}): {Balance: big.NewInt(1)},
			common.BytesToAddress([]byte{239}): {Balance: big.NewInt(1)},
			common.BytesToAddress([]byte{240}): {Balance: big.NewInt(1)},
			common.BytesToAddress([]byte{241}): {Balance: big.NewInt(1)},
			common.BytesToAddress([]byte{242}): {Balance: big.NewInt(1)},
			common.BytesToAddress([]byte{243}): {Balance: big.NewInt(1)},
			common.BytesToAddress([]byte{244}): {Balance: big.NewInt(1)},
			common.BytesToAddress([]byte{245}): {Balance: big.NewInt(1)},
			common.BytesToAddress([]byte{246}): {Balance: big.NewInt(1)},
			common.BytesToAddress([]byte{247}): {Balance: big.NewInt(1)},
			common.BytesToAddress([]byte{248}): {Balance: big.NewInt(1)},
			common.BytesToAddress([]byte{249}): {Balance: big.NewInt(1)},
			common.BytesToAddress([]byte{250}): {Balance: big.NewInt(1)},
			common.BytesToAddress([]byte{251}): {Balance: big.NewInt(1)},
			common.BytesToAddress([]byte{252}): {Balance: big.NewInt(1)},
			common.BytesToAddress([]byte{253}): {Balance: big.NewInt(1)},
			common.BytesToAddress([]byte{254}): {Balance: big.NewInt(1)},
			common.BytesToAddress([]byte{255}): {Balance: big.NewInt(1)},
		},
	}
}

func DefaultSeoulGenesisBlock() *Genesis {
	balanceStr := "40996800000000000000000000"
	balance, _ := new(big.Int).SetString(balanceStr, 10)
	return &Genesis{
		Config:     params.SeoulChainConfig,
		Nonce:      103,
		Timestamp:  1691449688,
		ExtraData:  []byte("Worldland Seoul"),
		GasLimit:   30000000,
		Difficulty: big.NewInt(1023),
		Alloc: map[common.Address]GenesisAccount{
			common.HexToAddress("0x8C98EAeA19F1B9B36af58e7d7E78e0F1df8138f0"): { Balance: balance },
		},
	}
}

func DefaultGwangjuGenesisBlock() *Genesis {
	balanceStr := "40996800000000000000000000"
	balance, _ := new(big.Int).SetString(balanceStr, 10)
	return &Genesis{
		Config:     params.GwangjuChainConfig,
		Nonce:      10395,
		Timestamp:  1689649200,
		ExtraData:  []byte("Worldland Gwnagju"),
		GasLimit:   30000000,
		Difficulty: big.NewInt(1023),
		Alloc:      map[common.Address]GenesisAccount{
			common.HexToAddress("0x8C98EAeA19F1B9B36af58e7d7E78e0F1df8138f0"): { Balance: balance },
		},
	}
}

// DeveloperGenesisBlock returns the 'geth --dev' genesis block.
func DeveloperGenesisBlock(period uint64, gasLimit uint64, faucet common.Address) *Genesis {
	// Override the default period to the user requested one
	config := *params.AllCliqueProtocolChanges
	config.Clique = &params.CliqueConfig{
		Period: period,
		Epoch:  config.Clique.Epoch,
	}

	// Assemble and return the genesis with the precompiles and faucet pre-funded
	return &Genesis{
		Config:     &config,
		ExtraData:  append(append(make([]byte, 32), faucet[:]...), make([]byte, crypto.SignatureLength)...),
		GasLimit:   gasLimit,
		BaseFee:    big.NewInt(params.InitialBaseFee),
		Difficulty: big.NewInt(1),
		Alloc: map[common.Address]GenesisAccount{
			common.BytesToAddress([]byte{1}): {Balance: big.NewInt(1)}, // ECRecover
			common.BytesToAddress([]byte{2}): {Balance: big.NewInt(1)}, // SHA256
			common.BytesToAddress([]byte{3}): {Balance: big.NewInt(1)}, // RIPEMD
			common.BytesToAddress([]byte{4}): {Balance: big.NewInt(1)}, // Identity
			common.BytesToAddress([]byte{5}): {Balance: big.NewInt(1)}, // ModExp
			common.BytesToAddress([]byte{6}): {Balance: big.NewInt(1)}, // ECAdd
			common.BytesToAddress([]byte{7}): {Balance: big.NewInt(1)}, // ECScalarMul
			common.BytesToAddress([]byte{8}): {Balance: big.NewInt(1)}, // ECPairing
			common.BytesToAddress([]byte{9}): {Balance: big.NewInt(1)}, // BLAKE2b
			faucet:                           {Balance: new(big.Int).Sub(new(big.Int).Lsh(big.NewInt(1), 256), big.NewInt(9))},
		},
	}
}

func decodePrealloc(data string) GenesisAlloc {
	var p []struct{ Addr, Balance *big.Int }
	if err := rlp.NewStream(strings.NewReader(data), 0).Decode(&p); err != nil {
		panic(err)
	}
	ga := make(GenesisAlloc, len(p))
	for _, account := range p {
		ga[common.BigToAddress(account.Addr)] = GenesisAccount{Balance: account.Balance}
	}
	return ga
}
