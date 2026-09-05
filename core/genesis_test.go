// Copyright 2017 The go-ethereum Authors
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
	"encoding/json"
	"fmt"
	"math/big"
	"reflect"
	"strings"
	"testing"

	"github.com/davecgh/go-spew/spew"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/consensus/ethash"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/params"
	"github.com/ethereum/go-ethereum/triedb"
	"github.com/ethereum/go-ethereum/triedb/pathdb"
)

func TestSetupGenesis(t *testing.T) {
	testSetupGenesis(t, rawdb.HashScheme)
	testSetupGenesis(t, rawdb.PathScheme)
}

func testSetupGenesis(t *testing.T, scheme string) {
	var (
		customghash = common.HexToHash("0x89c99d90b79719238d2645c7642f2c9295246e80775b38cfd162b696817fbd50")
		customg     = Genesis{
			Config: &params.ChainConfig{HomesteadBlock: big.NewInt(3), Ethash: &params.EthashConfig{}},
			Alloc: types.GenesisAlloc{
				{1}: {Balance: big.NewInt(1), Storage: map[common.Hash]common.Hash{{1}: {1}}},
			},
		}
		oldcustomg = customg
	)
	oldcustomg.Config = &params.ChainConfig{HomesteadBlock: big.NewInt(2), Ethash: &params.EthashConfig{}}

	tests := []struct {
		name           string
		fn             func(ethdb.Database) (*params.ChainConfig, common.Hash, *params.ConfigCompatError, error)
		wantConfig     *params.ChainConfig
		wantHash       common.Hash
		wantErr        error
		wantCompactErr *params.ConfigCompatError
	}{
		{
			name: "genesis without ChainConfig",
			fn: func(db ethdb.Database) (*params.ChainConfig, common.Hash, *params.ConfigCompatError, error) {
				return SetupGenesisBlock(db, triedb.NewDatabase(db, newDbConfig(scheme)), new(Genesis))
			},
			wantErr: errGenesisNoConfig,
		},
		{
			name: "no block in DB, genesis == nil",
			fn: func(db ethdb.Database) (*params.ChainConfig, common.Hash, *params.ConfigCompatError, error) {
				return SetupGenesisBlock(db, triedb.NewDatabase(db, newDbConfig(scheme)), nil)
			},
			wantHash:   params.MainnetGenesisHash,
			wantConfig: params.MainnetChainConfig,
		},
		{
			name: "mainnet block in DB, genesis == nil",
			fn: func(db ethdb.Database) (*params.ChainConfig, common.Hash, *params.ConfigCompatError, error) {
				DefaultGenesisBlock().MustCommit(db, triedb.NewDatabase(db, newDbConfig(scheme)))
				return SetupGenesisBlock(db, triedb.NewDatabase(db, newDbConfig(scheme)), nil)
			},
			wantHash:   params.MainnetGenesisHash,
			wantConfig: params.MainnetChainConfig,
		},
		{
			name: "custom block in DB, genesis == nil",
			fn: func(db ethdb.Database) (*params.ChainConfig, common.Hash, *params.ConfigCompatError, error) {
				tdb := triedb.NewDatabase(db, newDbConfig(scheme))
				customg.Commit(db, tdb, nil)
				return SetupGenesisBlock(db, tdb, nil)
			},
			wantHash:   customghash,
			wantConfig: customg.Config,
		},
		{
			name: "custom block in DB, genesis == sepolia",
			fn: func(db ethdb.Database) (*params.ChainConfig, common.Hash, *params.ConfigCompatError, error) {
				tdb := triedb.NewDatabase(db, newDbConfig(scheme))
				customg.Commit(db, tdb, nil)
				return SetupGenesisBlock(db, tdb, DefaultSepoliaGenesisBlock())
			},
			wantErr: &GenesisMismatchError{Stored: customghash, New: params.SepoliaGenesisHash},
		},
		{
			name: "custom block in DB, genesis == hoodi",
			fn: func(db ethdb.Database) (*params.ChainConfig, common.Hash, *params.ConfigCompatError, error) {
				tdb := triedb.NewDatabase(db, newDbConfig(scheme))
				customg.Commit(db, tdb, nil)
				return SetupGenesisBlock(db, tdb, DefaultHoodiGenesisBlock())
			},
			wantErr: &GenesisMismatchError{Stored: customghash, New: params.HoodiGenesisHash},
		},
		{
			name: "compatible config in DB",
			fn: func(db ethdb.Database) (*params.ChainConfig, common.Hash, *params.ConfigCompatError, error) {
				tdb := triedb.NewDatabase(db, newDbConfig(scheme))
				oldcustomg.Commit(db, tdb, nil)
				return SetupGenesisBlock(db, tdb, &customg)
			},
			wantHash:   customghash,
			wantConfig: customg.Config,
		},
		{
			name: "incompatible config in DB",
			fn: func(db ethdb.Database) (*params.ChainConfig, common.Hash, *params.ConfigCompatError, error) {
				// Commit the 'old' genesis block with Homestead transition at #2.
				// Advance to block #4, past the homestead transition block of customg.
				tdb := triedb.NewDatabase(db, newDbConfig(scheme))
				oldcustomg.Commit(db, tdb, nil)

				bc, _ := NewBlockChain(db, &oldcustomg, ethash.NewFullFaker(), DefaultConfig().WithStateScheme(scheme))
				defer bc.Stop()

				_, blocks, _ := GenerateChainWithGenesis(&oldcustomg, ethash.NewFaker(), 4, nil)
				bc.InsertChain(blocks)

				// This should return a compatibility error.
				return SetupGenesisBlock(db, tdb, &customg)
			},
			wantHash:   customghash,
			wantConfig: customg.Config,
			wantCompactErr: &params.ConfigCompatError{
				What:          "Homestead fork block",
				StoredBlock:   big.NewInt(2),
				NewBlock:      big.NewInt(3),
				RewindToBlock: 1,
			},
		},
	}

	for _, test := range tests {
		db := rawdb.NewMemoryDatabase()
		config, hash, compatErr, err := test.fn(db)
		// Check the return values.
		if !reflect.DeepEqual(err, test.wantErr) {
			spew := spew.ConfigState{DisablePointerAddresses: true, DisableCapacities: true}
			t.Errorf("%s: returned error %#v, want %#v", test.name, spew.NewFormatter(err), spew.NewFormatter(test.wantErr))
		}
		if !reflect.DeepEqual(compatErr, test.wantCompactErr) {
			spew := spew.ConfigState{DisablePointerAddresses: true, DisableCapacities: true}
			t.Errorf("%s: returned error %#v, want %#v", test.name, spew.NewFormatter(compatErr), spew.NewFormatter(test.wantCompactErr))
		}
		if !reflect.DeepEqual(config, test.wantConfig) {
			t.Errorf("%s:\nreturned %v\nwant     %v", test.name, config, test.wantConfig)
		}
		if hash != test.wantHash {
			t.Errorf("%s: returned hash %s, want %s", test.name, hash.Hex(), test.wantHash.Hex())
		} else if err == nil {
			// Check database content.
			stored := rawdb.ReadBlock(db, test.wantHash, 0)
			if stored.Hash() != test.wantHash {
				t.Errorf("%s: block in DB has hash %s, want %s", test.name, stored.Hash(), test.wantHash)
			}
		}
	}
}

// TestGenesisHashes checks the congruity of default genesis data to
// corresponding hardcoded genesis hash values.
func TestGenesisHashes(t *testing.T) {
	for i, c := range []struct {
		genesis *Genesis
		want    common.Hash
	}{
		{DefaultGenesisBlock(), params.MainnetGenesisHash},
		{DefaultSepoliaGenesisBlock(), params.SepoliaGenesisHash},
		{DefaultHoodiGenesisBlock(), params.HoodiGenesisHash},
	} {
		// Test via MustCommit
		db := rawdb.NewMemoryDatabase()
		if have := c.genesis.MustCommit(db, triedb.NewDatabase(db, triedb.HashDefaults)).Hash(); have != c.want {
			t.Errorf("case: %d a), want: %s, got: %s", i, c.want.Hex(), have.Hex())
		}
		// Test via ToBlock
		if have := c.genesis.ToBlock().Hash(); have != c.want {
			t.Errorf("case: %d b), want: %s, got: %s", i, c.want.Hex(), have.Hex())
		}
	}
}

func TestGenesisCommit(t *testing.T) {
	genesis := &Genesis{
		BaseFee: big.NewInt(params.InitialBaseFee),
		Config:  params.TestChainConfig,
		// difficulty is nil
	}

	db := rawdb.NewMemoryDatabase()
	genesisBlock := genesis.MustCommit(db, triedb.NewDatabase(db, triedb.HashDefaults))

	if genesis.Difficulty != nil {
		t.Fatalf("assumption wrong")
	}

	// This value should have been set as default in the ToBlock method.
	if genesisBlock.Difficulty().Cmp(params.GenesisDifficulty) != 0 {
		t.Errorf("assumption wrong: want: %d, got: %v", params.GenesisDifficulty, genesisBlock.Difficulty())
	}
}

func TestReadWriteGenesisAlloc(t *testing.T) {
	var (
		db    = rawdb.NewMemoryDatabase()
		alloc = &types.GenesisAlloc{
			{1}: {Balance: big.NewInt(1), Storage: map[common.Hash]common.Hash{{1}: {1}}},
			{2}: {Balance: big.NewInt(2), Storage: map[common.Hash]common.Hash{{2}: {2}}},
		}
		hash, _ = hashAlloc(&Genesis{Alloc: *alloc}, false)
	)
	blob, _ := json.Marshal(alloc)
	rawdb.WriteGenesisStateSpec(db, hash, blob)

	var reload types.GenesisAlloc
	err := reload.UnmarshalJSON(rawdb.ReadGenesisStateSpec(db, hash))
	if err != nil {
		t.Fatalf("Failed to load genesis state %v", err)
	}
	if len(reload) != len(*alloc) {
		t.Fatal("Unexpected genesis allocation")
	}
	for addr, account := range reload {
		want, ok := (*alloc)[addr]
		if !ok {
			t.Fatal("Account is not found")
		}
		if !reflect.DeepEqual(want, account) {
			t.Fatal("Unexpected account")
		}
	}
}

func newDbConfig(scheme string) *triedb.Config {
	if scheme == rawdb.HashScheme {
		return triedb.HashDefaults
	}
	config := *pathdb.Defaults
	config.NoAsyncFlush = true
	return &triedb.Config{PathDB: &config}
}

func TestBinaryGenesisCommit(t *testing.T) {
	var ubtTime uint64 = 0
	ubtConfig := &params.ChainConfig{
		ChainID:                 big.NewInt(1),
		HomesteadBlock:          big.NewInt(0),
		DAOForkBlock:            nil,
		DAOForkSupport:          false,
		EIP150Block:             big.NewInt(0),
		EIP155Block:             big.NewInt(0),
		EIP158Block:             big.NewInt(0),
		ByzantiumBlock:          big.NewInt(0),
		ConstantinopleBlock:     big.NewInt(0),
		PetersburgBlock:         big.NewInt(0),
		IstanbulBlock:           big.NewInt(0),
		MuirGlacierBlock:        big.NewInt(0),
		BerlinBlock:             big.NewInt(0),
		LondonBlock:             big.NewInt(0),
		ArrowGlacierBlock:       big.NewInt(0),
		GrayGlacierBlock:        big.NewInt(0),
		MergeNetsplitBlock:      nil,
		ShanghaiTime:            &ubtTime,
		CancunTime:              &ubtTime,
		PragueTime:              &ubtTime,
		OsakaTime:               &ubtTime,
		UBTTime:                 &ubtTime,
		TerminalTotalDifficulty: big.NewInt(0),
		EnableUBTAtGenesis:      true,
		Ethash:                  nil,
		Clique:                  nil,
		BlobScheduleConfig: &params.BlobScheduleConfig{
			Cancun: params.DefaultCancunBlobConfig,
			Prague: params.DefaultPragueBlobConfig,
		},
	}

	genesis := &Genesis{
		BaseFee:    big.NewInt(params.InitialBaseFee),
		Config:     ubtConfig,
		Timestamp:  ubtTime,
		Difficulty: big.NewInt(0),
		Alloc: types.GenesisAlloc{
			{1}: {Balance: big.NewInt(1), Storage: map[common.Hash]common.Hash{{1}: {1}}},
		},
	}

	expected := common.FromHex("0870fd587c41dc778019de8c5cb3193fe4ef1f417976461952d3712ba39163f5")
	got := genesis.ToBlock().Root().Bytes()
	if !bytes.Equal(got, expected) {
		t.Fatalf("invalid genesis state root, expected %x, got %x", expected, got)
	}

	db := rawdb.NewMemoryDatabase()

	config := *pathdb.Defaults
	config.NoAsyncFlush = true

	triedb := triedb.NewDatabase(db, &triedb.Config{
		IsUBT:             true,
		PathDB:            &config,
		BinTrieGroupDepth: triedb.DefaultBinTrieGroupDepth,
	})
	block := genesis.MustCommit(db, triedb)
	if !bytes.Equal(block.Root().Bytes(), expected) {
		t.Fatalf("invalid genesis state root, expected %x, got %x", expected, block.Root())
	}

	// Test that the trie is a unified binary trie
	if !triedb.IsUBT() {
		t.Fatalf("expected trie to be a unified binary trie")
	}
	vdb := rawdb.NewTable(db, string(rawdb.VerklePrefix))
	if !rawdb.HasAccountTrieNode(vdb, nil) {
		t.Fatal("could not find node")
	}
}

var (
	testConstructorRuntime = []byte{0xfe}
	testConstructorInit    = []byte{
		0x60, 0x42, 0x60, 0x01, 0x55, // SSTORE(1, 0x42)
		0x60, 0xfe, 0x60, 0x00, 0x53, // MSTORE8(0, 0xfe)
		0x60, 0x01, 0x60, 0x00, 0xf3, // RETURN(0, 1)
	}
)

func TestGenesisAllocConstructor(t *testing.T) {
	var (
		addr  = common.HexToAddress("0xc0de")
		db    = rawdb.NewMemoryDatabase()
		tdb   = triedb.NewDatabase(db, triedb.HashDefaults)
		gspec = &Genesis{
			Config:     params.TestChainConfig,
			Difficulty: big.NewInt(0),
			Alloc: types.GenesisAlloc{
				addr: {Balance: big.NewInt(1), Constructor: testConstructorInit},
			},
		}
	)
	block, err := gspec.Commit(db, tdb, nil)
	if err != nil {
		t.Fatalf("failed to commit genesis: %v", err)
	}
	if want := gspec.ToBlock().Root(); block.Root() != want {
		t.Errorf("state root mismatch: have %x, want %x", block.Root(), want)
	}
	statedb, err := state.New(block.Root(), state.NewDatabase(tdb, nil))
	if err != nil {
		t.Fatalf("failed to open genesis state: %v", err)
	}
	if code := statedb.GetCode(addr); !bytes.Equal(code, testConstructorRuntime) {
		t.Errorf("deployed code mismatch: have %#x, want %#x", code, testConstructorRuntime)
	}
	// Storage written by the constructor must land on the allocated account,
	// not on some address derived from it.
	if got, want := statedb.GetState(addr, common.Hash{31: 0x01}), (common.Hash{31: 0x42}); got != want {
		t.Errorf("constructor storage mismatch: have %x, want %x", got, want)
	}
	if got := statedb.GetNonce(addr); got != 1 {
		t.Errorf("nonce mismatch: have %d, want 1", got)
	}
	if got := statedb.GetBalance(addr); got.Uint64() != 1 {
		t.Errorf("balance mismatch: have %v, want 1", got)
	}
	for _, stray := range []common.Address{crypto.CreateAddress(addr, 0), crypto.CreateAddress(addr, 1), {}} {
		if statedb.Exist(stray) {
			t.Errorf("unexpected account %x in genesis state", stray)
		}
	}
}

// TestGenesisAllocConstructorOverrides checks that the explicit fields of an
// allocation take precedence over the constructor's effects.
func TestGenesisAllocConstructorOverrides(t *testing.T) {
	var (
		addr  = common.HexToAddress("0xc0de")
		db    = rawdb.NewMemoryDatabase()
		tdb   = triedb.NewDatabase(db, triedb.HashDefaults)
		gspec = &Genesis{
			Config:     params.TestChainConfig,
			Difficulty: big.NewInt(0),
			Alloc: types.GenesisAlloc{
				addr: {
					Balance:     big.NewInt(1),
					Nonce:       7,
					Storage:     map[common.Hash]common.Hash{{31: 0x01}: {31: 0x99}},
					Constructor: testConstructorInit,
				},
			},
		}
	)
	block, err := gspec.Commit(db, tdb, nil)
	if err != nil {
		t.Fatalf("failed to commit genesis: %v", err)
	}
	statedb, err := state.New(block.Root(), state.NewDatabase(tdb, nil))
	if err != nil {
		t.Fatalf("failed to open genesis state: %v", err)
	}
	if got, want := statedb.GetState(addr, common.Hash{31: 0x01}), (common.Hash{31: 0x99}); got != want {
		t.Errorf("storage mismatch: have %x, want %x", got, want)
	}
	if got := statedb.GetNonce(addr); got != 7 {
		t.Errorf("nonce mismatch: have %d, want 7", got)
	}
	if code := statedb.GetCode(addr); !bytes.Equal(code, testConstructorRuntime) {
		t.Errorf("deployed code mismatch: have %#x, want %#x", code, testConstructorRuntime)
	}
}

// TestGenesisAllocConstructorFailure checks that a constructor which cannot be
// executed surfaces an error instead of taking the process down, on both the
// hashing and the flushing path.
func TestGenesisAllocConstructorFailure(t *testing.T) {
	addr := common.HexToAddress("0xc0de")
	tests := []struct {
		name     string
		account  types.Account
		noConfig bool
		wantErr  string
	}{
		{
			name:    "revert",
			account: types.Account{Balance: big.NewInt(1), Constructor: []byte{0x60, 0x00, 0x60, 0x00, 0xfd}},
			wantErr: "execution reverted",
		},
		{
			name:    "invalid opcode",
			account: types.Account{Balance: big.NewInt(1), Constructor: []byte{0xfe}},
			wantErr: "invalid opcode",
		},
		{
			name:    "ef prefixed initcode",
			account: types.Account{Balance: big.NewInt(1), Constructor: []byte{0xef, 0x00}},
			wantErr: "initcode starts with 0xEF",
		},
		{
			name:    "code and constructor",
			account: types.Account{Balance: big.NewInt(1), Code: []byte{0x00}, Constructor: testConstructorInit},
			wantErr: "both code and constructor",
		},
		{
			name:     "no chain config",
			account:  types.Account{Balance: big.NewInt(1), Constructor: testConstructorInit},
			noConfig: true,
			wantErr:  errGenesisNoConfig.Error(),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gspec := &Genesis{
				Config:     params.TestChainConfig,
				Difficulty: big.NewInt(0),
				Alloc:      types.GenesisAlloc{addr: tt.account},
			}
			if tt.noConfig {
				gspec.Config = nil
			}
			err, panicked := hashAllocResult(gspec)
			if panicked {
				t.Errorf("hashAlloc panicked instead of returning an error: %v", err)
			}
			if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
				t.Errorf("hashAlloc error = %v, want it to contain %q", err, tt.wantErr)
			}
			tdb := triedb.NewDatabase(rawdb.NewMemoryDatabase(), triedb.HashDefaults)
			if _, err := flushAlloc(gspec, tdb, nil); err == nil || !strings.Contains(err.Error(), tt.wantErr) {
				t.Errorf("flushAlloc error = %v, want it to contain %q", err, tt.wantErr)
			}
		})
	}
}

// hashAllocResult runs hashAlloc, reporting whether it aborted through a panic
// rather than through an error.
func hashAllocResult(g *Genesis) (err error, panicked bool) {
	defer func() {
		if r := recover(); r != nil {
			err, panicked = fmt.Errorf("%v", r), true
		}
	}()
	_, err = hashAlloc(g, false)
	return err, false
}

// TestGenesisAllocConstructorJSON checks that constructors survive the JSON
// encoding used to persist and load genesis specifications.
func TestGenesisAllocConstructorJSON(t *testing.T) {
	alloc := types.GenesisAlloc{
		{1}: {Balance: big.NewInt(1), Constructor: testConstructorInit},
		{2}: {Balance: big.NewInt(2), Code: []byte{0x00}},
	}
	blob, err := json.Marshal(alloc)
	if err != nil {
		t.Fatalf("failed to marshal alloc: %v", err)
	}
	if !strings.Contains(string(blob), `"constructor":"`+hexutil.Encode(testConstructorInit)+`"`) {
		t.Fatalf("constructor missing from encoded alloc: %s", blob)
	}
	var reload types.GenesisAlloc
	if err := reload.UnmarshalJSON(blob); err != nil {
		t.Fatalf("failed to load genesis state: %v", err)
	}
	if !reflect.DeepEqual(reload, alloc) {
		t.Fatalf("alloc mismatch:\n%s", spew.Sdump(reload))
	}
}
