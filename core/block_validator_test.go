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
	"math/big"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/consensus"
	"github.com/ethereum/go-ethereum/consensus/beacon"
	"github.com/ethereum/go-ethereum/consensus/clique"
	"github.com/ethereum/go-ethereum/consensus/ethash"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/params"
	"github.com/ethereum/go-ethereum/trie"
)

// -----------------------------------------------------------------------------
// Berachain:Prague-1 PoL validation tests
// -----------------------------------------------------------------------------

// newPrague1Config returns a ChainConfig where Prague (and Prague-1) are active
// from timestamp 0. The caller can specify the PoL distributor address.
func newPrague1Config(distributor common.Address) *params.ChainConfig {
	zero := uint64(0)
	cfg := *params.AllDevChainProtocolChanges // copy
	cfg.Berachain.Prague1 = params.Prague1Config{
		Time:                     &zero,
		BaseFeeChangeDenominator: 48,
		MinimumBaseFeeWei:        10000000000,
		PoLDistributorAddress:    distributor,
	}
	return &cfg
}

// buildTestChain initialises an in-memory blockchain with the provided config
// and returns the chain and its validator.
func buildTestChain(t *testing.T, cfg *params.ChainConfig) (*BlockChain, Validator) {
	t.Helper()
	db := rawdb.NewMemoryDatabase()
	genesis := &Genesis{Config: cfg}
	chain, err := NewBlockChain(db, genesis, ethash.NewFaker(), nil)
	if err != nil {
		t.Fatalf("failed to create blockchain: %v", err)
	}
	return chain, chain.Validator()
}

// samplePubkey returns a deterministic 48-byte pubkey for tests.
func samplePubkey() *common.Pubkey {
	var pk common.Pubkey
	for i := 0; i < common.PubkeyLength; i++ {
		pk[i] = byte(i)
	}
	return &pk
}

// makeBlock builds a child block on top of parent with the supplied txs and timestamp.
func makeBlock(parent *types.Header, txs types.Transactions, timestamp uint64) *types.Block {
	header := &types.Header{
		ParentHash:           parent.Hash(),
		Number:               new(big.Int).Add(parent.Number, big.NewInt(1)),
		Time:                 timestamp,
		GasLimit:             params.GenesisGasLimit * 10,
		Difficulty:           big.NewInt(1),
		ParentProposerPubkey: samplePubkey(),
		BaseFee:              big.NewInt(1000000000),
	}
	return types.NewBlock(header, &types.Body{Transactions: txs}, nil, trie.NewStackTrie(nil))
}

func TestValidateBody_Prague1_Valid(t *testing.T) {
	distributor := common.HexToAddress("0x1111111111111111111111111111111111111111")
	cfg := newPrague1Config(distributor)
	chain, validator := buildTestChain(t, cfg)

	// Build PoL tx + dummy tx.
	polTx, err := types.NewPoLTx(cfg.ChainID, distributor, big.NewInt(0), params.PoLTxGasLimit, big.NewInt(1000000000), samplePubkey())
	if err != nil {
		t.Fatalf("failed to create PoL tx: %v", err)
	}
	dummyTx := types.NewTx(&types.LegacyTx{Nonce: 1})

	block := makeBlock(chain.CurrentHeader(), types.Transactions{polTx, dummyTx}, 1)

	if err := validator.ValidateBody(block); err != nil {
		t.Fatalf("ValidateBody returned error for valid Prague1 block: %v", err)
	}
}

func TestValidateBody_Prague1_InvalidHash(t *testing.T) {
	distributor := common.HexToAddress("0x2222222222222222222222222222222222222222")
	cfg := newPrague1Config(distributor)
	chain, validator := buildTestChain(t, cfg)

	// PoL tx with WRONG pubkey (different from header.ParentProposerPubkey).
	wrongPk := &common.Pubkey{}
	polTx, _ := types.NewPoLTx(cfg.ChainID, distributor, big.NewInt(0), params.PoLTxGasLimit, big.NewInt(1000000000), wrongPk)
	block := makeBlock(chain.CurrentHeader(), types.Transactions{polTx}, 1)

	if err := validator.ValidateBody(block); err == nil {
		t.Fatalf("expected error due to invalid PoL hash, got nil")
	}
}

func TestValidateBody_Prague1_MisplacedPoL(t *testing.T) {
	distributor := common.HexToAddress("0x3333333333333333333333333333333333333333")
	cfg := newPrague1Config(distributor)
	chain, validator := buildTestChain(t, cfg)

	polTx, _ := types.NewPoLTx(cfg.ChainID, distributor, big.NewInt(0), params.PoLTxGasLimit, big.NewInt(1000000000), samplePubkey())
	dummyTx := types.NewTx(&types.LegacyTx{Nonce: 1})
	// PoL tx placed second.
	block := makeBlock(chain.CurrentHeader(), types.Transactions{dummyTx, polTx}, 1)

	if err := validator.ValidateBody(block); err == nil {
		t.Fatalf("expected error for PoL tx at index >0, got nil")
	}
}

func TestValidateBody_PrePrague1_PoLProhibited(t *testing.T) {
	distributor := common.HexToAddress("0x4444444444444444444444444444444444444444")
	// Prague time active, but Prague1 *future* at timestamp 1000.
	future := uint64(1000)
	cfg := *params.AllDevChainProtocolChanges
	cfg.Berachain.Prague1.Time = &future
	cfg.Berachain.Prague1.BaseFeeChangeDenominator = 48
	cfg.Berachain.Prague1.MinimumBaseFeeWei = 10000000000
	cfg.Berachain.Prague1.PoLDistributorAddress = distributor
	chain, validator := buildTestChain(t, &cfg)

	polTx, _ := types.NewPoLTx(cfg.ChainID, distributor, big.NewInt(0), params.PoLTxGasLimit, big.NewInt(1000000000), samplePubkey())
	block := makeBlock(chain.CurrentHeader(), types.Transactions{polTx}, 1) // timestamp 1 < future

	if err := validator.ValidateBody(block); err == nil {
		t.Fatalf("expected error: PoL tx before Prague1 fork should be invalid")
	}
}

// Tests that simple header verification works, for both good and bad blocks.
func TestHeaderVerification(t *testing.T) {
	testHeaderVerification(t, rawdb.HashScheme)
	testHeaderVerification(t, rawdb.PathScheme)
}

func testHeaderVerification(t *testing.T, scheme string) {
	// Create a simple chain to verify
	var (
		gspec        = &Genesis{Config: params.TestChainConfig}
		_, blocks, _ = GenerateChainWithGenesis(gspec, ethash.NewFaker(), 8, nil)
	)
	headers := make([]*types.Header, len(blocks))
	for i, block := range blocks {
		headers[i] = block.Header()
	}
	// Run the header checker for blocks one-by-one, checking for both valid and invalid nonces
	options := DefaultConfig().WithStateScheme(scheme)
	chain, err := NewBlockChain(rawdb.NewMemoryDatabase(), gspec, ethash.NewFaker(), options)
	defer chain.Stop()
	if err != nil {
		t.Fatal(err)
	}

	for i := 0; i < len(blocks); i++ {
		for j, valid := range []bool{true, false} {
			var results <-chan error

			if valid {
				engine := ethash.NewFaker()
				_, results = engine.VerifyHeaders(chain, []*types.Header{headers[i]})
			} else {
				engine := ethash.NewFakeFailer(headers[i].Number.Uint64())
				_, results = engine.VerifyHeaders(chain, []*types.Header{headers[i]})
			}
			// Wait for the verification result
			select {
			case result := <-results:
				if (result == nil) != valid {
					t.Errorf("test %d.%d: validity mismatch: have %v, want %v", i, j, result, valid)
				}
			case <-time.After(time.Second):
				t.Fatalf("test %d.%d: verification timeout", i, j)
			}
			// Make sure no more data is returned
			select {
			case result := <-results:
				t.Fatalf("test %d.%d: unexpected result returned: %v", i, j, result)
			case <-time.After(25 * time.Millisecond):
			}
		}
		chain.InsertChain(blocks[i : i+1])
	}
}

func TestHeaderVerificationForMergingClique(t *testing.T) { testHeaderVerificationForMerging(t, true) }
func TestHeaderVerificationForMergingEthash(t *testing.T) { testHeaderVerificationForMerging(t, false) }

// Tests the verification for eth1/2 merging, including pre-merge and post-merge
func testHeaderVerificationForMerging(t *testing.T, isClique bool) {
	var (
		gspec      *Genesis
		preBlocks  []*types.Block
		postBlocks []*types.Block
		engine     consensus.Engine
	)
	if isClique {
		var (
			key, _ = crypto.HexToECDSA("b71c71a67e1177ad4e901695e1b4b9ee17ae16c6668d313eac2f96dbcda3f291")
			addr   = crypto.PubkeyToAddress(key.PublicKey)
			config = *params.AllCliqueProtocolChanges
		)
		engine = beacon.New(clique.New(params.AllCliqueProtocolChanges.Clique, rawdb.NewMemoryDatabase()))
		gspec = &Genesis{
			Config:    &config,
			ExtraData: make([]byte, 32+common.AddressLength+crypto.SignatureLength),
			Alloc: map[common.Address]types.Account{
				addr: {Balance: big.NewInt(1)},
			},
			BaseFee:    big.NewInt(params.InitialBaseFee),
			Difficulty: new(big.Int),
		}
		copy(gspec.ExtraData[32:], addr[:])

		// chain_maker has no blockchain to retrieve the TTD from, setting to nil
		// is a hack to signal it to generate pre-merge blocks
		gspec.Config.TerminalTotalDifficulty = nil
		td := 0
		genDb, blocks, _ := GenerateChainWithGenesis(gspec, engine, 8, nil)

		for i, block := range blocks {
			header := block.Header()
			if i > 0 {
				header.ParentHash = blocks[i-1].Hash()
			}
			header.Extra = make([]byte, 32+crypto.SignatureLength)
			header.Difficulty = big.NewInt(2)

			sig, _ := crypto.Sign(engine.SealHash(header).Bytes(), key)
			copy(header.Extra[len(header.Extra)-crypto.SignatureLength:], sig)
			blocks[i] = block.WithSeal(header)

			// calculate td
			td += int(block.Difficulty().Uint64())
		}
		preBlocks = blocks
		gspec.Config.TerminalTotalDifficulty = big.NewInt(int64(td))
		postBlocks, _ = GenerateChain(gspec.Config, preBlocks[len(preBlocks)-1], engine, genDb, 8, nil)
	} else {
		config := *params.TestChainConfig
		gspec = &Genesis{Config: &config}
		engine = beacon.New(ethash.NewFaker())
		td := int(params.GenesisDifficulty.Uint64())
		genDb, blocks, _ := GenerateChainWithGenesis(gspec, engine, 8, nil)
		for _, block := range blocks {
			// calculate td
			td += int(block.Difficulty().Uint64())
		}
		preBlocks = blocks
		gspec.Config.TerminalTotalDifficulty = big.NewInt(int64(td))
		postBlocks, _ = GenerateChain(gspec.Config, preBlocks[len(preBlocks)-1], engine, genDb, 8, func(i int, gen *BlockGen) {
			gen.SetPoS()
		})
	}
	// Assemble header batch
	preHeaders := make([]*types.Header, len(preBlocks))
	for i, block := range preBlocks {
		preHeaders[i] = block.Header()
	}
	postHeaders := make([]*types.Header, len(postBlocks))
	for i, block := range postBlocks {
		postHeaders[i] = block.Header()
	}
	// Run the header checker for blocks one-by-one, checking for both valid and invalid nonces
	chain, err := NewBlockChain(rawdb.NewMemoryDatabase(), gspec, engine, nil)
	defer chain.Stop()
	if err != nil {
		t.Fatal(err)
	}

	// Verify the blocks before the merging
	for i := 0; i < len(preBlocks); i++ {
		_, results := engine.VerifyHeaders(chain, []*types.Header{preHeaders[i]})
		// Wait for the verification result
		select {
		case result := <-results:
			if result != nil {
				t.Errorf("pre-block %d: verification failed %v", i, result)
			}
		case <-time.After(time.Second):
			t.Fatalf("pre-block %d: verification timeout", i)
		}
		// Make sure no more data is returned
		select {
		case result := <-results:
			t.Fatalf("pre-block %d: unexpected result returned: %v", i, result)
		case <-time.After(25 * time.Millisecond):
		}
		chain.InsertChain(preBlocks[i : i+1])
	}
	// Verify the blocks after the merging
	for i := 0; i < len(postBlocks); i++ {
		_, results := engine.VerifyHeaders(chain, []*types.Header{postHeaders[i]})
		// Wait for the verification result
		select {
		case result := <-results:
			if result != nil {
				t.Errorf("post-block %d: verification failed %v", i, result)
			}
		case <-time.After(time.Second):
			t.Fatalf("test %d: verification timeout", i)
		}
		// Make sure no more data is returned
		select {
		case result := <-results:
			t.Fatalf("post-block %d: unexpected result returned: %v", i, result)
		case <-time.After(25 * time.Millisecond):
		}
		chain.InsertBlockWithoutSetHead(postBlocks[i], false)
	}

	// Verify the blocks with pre-merge blocks and post-merge blocks
	var headers []*types.Header
	for _, block := range preBlocks {
		headers = append(headers, block.Header())
	}
	for _, block := range postBlocks {
		headers = append(headers, block.Header())
	}
	_, results := engine.VerifyHeaders(chain, headers)
	for i := 0; i < len(headers); i++ {
		select {
		case result := <-results:
			if result != nil {
				t.Errorf("test %d: verification failed %v", i, result)
			}
		case <-time.After(time.Second):
			t.Fatalf("test %d: verification timeout", i)
		}
	}
	// Make sure no more data is returned
	select {
	case result := <-results:
		t.Fatalf("unexpected result returned: %v", result)
	case <-time.After(25 * time.Millisecond):
	}
}

func TestCalcGasLimit(t *testing.T) {
	for i, tc := range []struct {
		pGasLimit uint64
		max       uint64
		min       uint64
	}{
		{20000000, 20019530, 19980470},
		{40000000, 40039061, 39960939},
	} {
		// Increase
		if have, want := CalcGasLimit(tc.pGasLimit, 2*tc.pGasLimit), tc.max; have != want {
			t.Errorf("test %d: have %d want <%d", i, have, want)
		}
		// Decrease
		if have, want := CalcGasLimit(tc.pGasLimit, 0), tc.min; have != want {
			t.Errorf("test %d: have %d want >%d", i, have, want)
		}
		// Small decrease
		if have, want := CalcGasLimit(tc.pGasLimit, tc.pGasLimit-1), tc.pGasLimit-1; have != want {
			t.Errorf("test %d: have %d want %d", i, have, want)
		}
		// Small increase
		if have, want := CalcGasLimit(tc.pGasLimit, tc.pGasLimit+1), tc.pGasLimit+1; have != want {
			t.Errorf("test %d: have %d want %d", i, have, want)
		}
		// No change
		if have, want := CalcGasLimit(tc.pGasLimit, tc.pGasLimit), tc.pGasLimit; have != want {
			t.Errorf("test %d: have %d want %d", i, have, want)
		}
	}
}
