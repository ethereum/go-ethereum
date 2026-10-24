// Copyright 2022 The go-ethereum Authors
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

package miner

import (
	"context"
	"errors"
	"math/big"
	"reflect"
	"strings"
	"sync"
	"testing"
	"testing/synctest"
	"time"

	"github.com/ethereum/go-ethereum/beacon/engine"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/consensus"
	"github.com/ethereum/go-ethereum/consensus/beacon"
	"github.com/ethereum/go-ethereum/consensus/clique"
	"github.com/ethereum/go-ethereum/consensus/ethash"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/core/txpool"
	"github.com/ethereum/go-ethereum/core/txpool/legacypool"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/params"
	"github.com/holiman/uint256"
)

var (
	// Test chain configurations
	testTxPoolConfig  legacypool.Config
	ethashChainConfig *params.ChainConfig
	cliqueChainConfig *params.ChainConfig

	// Test accounts
	testBankKey, _  = crypto.GenerateKey()
	testBankAddress = crypto.PubkeyToAddress(testBankKey.PublicKey)
	testBankFunds   = big.NewInt(1000000000000000000)

	testUserKey, _  = crypto.GenerateKey()
	testUserAddress = crypto.PubkeyToAddress(testUserKey.PublicKey)

	// Test transactions
	pendingTxs []*types.Transaction
	newTxs     []*types.Transaction

	testConfig = Config{
		PendingFeeRecipient: testBankAddress,
		Recommit:            time.Second,
		GasCeil:             params.GenesisGasLimit,
	}
)

func init() {
	testTxPoolConfig = legacypool.DefaultConfig
	testTxPoolConfig.Journal = ""
	ethashChainConfig = new(params.ChainConfig)
	*ethashChainConfig = *params.TestChainConfig
	cliqueChainConfig = new(params.ChainConfig)
	*cliqueChainConfig = *params.TestChainConfig
	cliqueChainConfig.Clique = &params.CliqueConfig{
		Period: 10,
		Epoch:  30000,
	}

	signer := types.LatestSigner(params.TestChainConfig)
	tx1 := types.MustSignNewTx(testBankKey, signer, &types.AccessListTx{
		ChainID:  params.TestChainConfig.ChainID,
		Nonce:    0,
		To:       &testUserAddress,
		Value:    big.NewInt(1000),
		Gas:      params.TxGas,
		GasPrice: big.NewInt(params.InitialBaseFee),
	})
	pendingTxs = append(pendingTxs, tx1)

	tx2 := types.MustSignNewTx(testBankKey, signer, &types.LegacyTx{
		Nonce:    1,
		To:       &testUserAddress,
		Value:    big.NewInt(1000),
		Gas:      params.TxGas,
		GasPrice: big.NewInt(params.InitialBaseFee),
	})
	newTxs = append(newTxs, tx2)
}

// testWorkerBackend implements worker.Backend interfaces and wraps all information needed during the testing.
type testWorkerBackend struct {
	db      ethdb.Database
	txPool  *txpool.TxPool
	chain   *core.BlockChain
	genesis *core.Genesis
}

func newTestWorkerBackend(t *testing.T, chainConfig *params.ChainConfig, engine consensus.Engine, db ethdb.Database, n int) *testWorkerBackend {
	var gspec = &core.Genesis{
		Config: chainConfig,
		Alloc:  types.GenesisAlloc{testBankAddress: {Balance: testBankFunds}},
	}
	switch e := engine.(type) {
	case *clique.Clique:
		gspec.ExtraData = make([]byte, 32+common.AddressLength+crypto.SignatureLength)
		copy(gspec.ExtraData[32:32+common.AddressLength], testBankAddress.Bytes())
		e.Authorize(testBankAddress)
	case *ethash.Ethash:
	case *beacon.Beacon:
	default:
		t.Fatalf("unexpected consensus engine type: %T", engine)
	}
	chain, err := core.NewBlockChain(db, gspec, engine, &core.BlockChainConfig{ArchiveMode: true})
	if err != nil {
		t.Fatalf("core.NewBlockChain failed: %v", err)
	}
	pool := legacypool.New(testTxPoolConfig, chain)
	txpool, _ := txpool.New(testTxPoolConfig.PriceLimit, chain, []txpool.SubPool{pool})

	return &testWorkerBackend{
		db:      db,
		chain:   chain,
		txPool:  txpool,
		genesis: gspec,
	}
}

func (b *testWorkerBackend) BlockChain() *core.BlockChain { return b.chain }
func (b *testWorkerBackend) TxPool() *txpool.TxPool       { return b.txPool }

func newTestWorker(t *testing.T, chainConfig *params.ChainConfig, engine consensus.Engine, db ethdb.Database, blocks int) (*Miner, *testWorkerBackend) {
	backend := newTestWorkerBackend(t, chainConfig, engine, db, blocks)
	backend.txPool.Add(pendingTxs, true)
	w := New(backend, testConfig, engine)
	return w, backend
}

func TestBuildPayload(t *testing.T) {
	var (
		db        = rawdb.NewMemoryDatabase()
		recipient = common.HexToAddress("0xdeadbeef")
	)
	w, b := newTestWorker(t, params.TestChainConfig, ethash.NewFaker(), db, 0)

	timestamp := uint64(time.Now().Unix())
	args := &BuildPayloadArgs{
		Parent:       b.chain.CurrentBlock().Hash(),
		Timestamp:    timestamp,
		Random:       common.Hash{},
		FeeRecipient: recipient,
	}
	payload, err := w.buildPayload(context.Background(), args, false)
	if err != nil {
		t.Fatalf("Failed to build payload %v", err)
	}
	verify := func(outer *engine.ExecutionPayloadEnvelope, txs int) {
		payload := outer.ExecutionPayload
		if payload.ParentHash != b.chain.CurrentBlock().Hash() {
			t.Fatal("Unexpected parent hash")
		}
		if payload.Random != (common.Hash{}) {
			t.Fatal("Unexpected random value")
		}
		if payload.Timestamp != timestamp {
			t.Fatal("Unexpected timestamp")
		}
		if payload.FeeRecipient != recipient {
			t.Fatal("Unexpected fee recipient")
		}
		if len(payload.Transactions) != txs {
			t.Fatal("Unexpected transaction set")
		}
	}
	empty := payload.ResolveEmpty()
	verify(empty, 0)

	full := payload.ResolveFull()
	verify(full, len(pendingTxs))

	// Ensure resolve can be called multiple times and the
	// result should be unchanged
	dataOne := payload.Resolve()
	dataTwo := payload.Resolve()
	if !reflect.DeepEqual(dataOne, dataTwo) {
		t.Fatal("Unexpected payload data")
	}
}

// TestBuildPayloadInclusionList checks that even the empty payload carries the
// inclusion list transactions that fit, including one that only becomes valid
// once another one is in, and leaves blob transactions out.
func TestBuildPayloadInclusionList(t *testing.T) {
	w, b := newTestWorker(t, params.TestChainConfig, ethash.NewFaker(), rawdb.NewMemoryDatabase(), 0)

	var (
		signer   = types.LatestSigner(params.TestChainConfig)
		transfer = func(nonce uint64) *types.Transaction {
			return types.MustSignNewTx(testBankKey, signer, &types.LegacyTx{
				Nonce:    nonce,
				To:       &testUserAddress,
				Value:    big.NewInt(1),
				Gas:      params.TxGas,
				GasPrice: big.NewInt(params.InitialBaseFee),
			})
		}
		first, second = transfer(0), transfer(1)
		blob          = types.MustSignNewTx(testBankKey, types.LatestSignerForChainID(params.TestChainConfig.ChainID), &types.BlobTx{
			ChainID:    uint256.MustFromBig(params.TestChainConfig.ChainID),
			GasFeeCap:  uint256.NewInt(params.InitialBaseFee),
			Gas:        params.TxGas,
			BlobFeeCap: uint256.NewInt(params.InitialBaseFee),
			BlobHashes: []common.Hash{{0x01}},
		})
		list [][]byte
	)
	for _, tx := range []*types.Transaction{second, blob, first} {
		enc, err := tx.MarshalBinary()
		if err != nil {
			t.Fatal(err)
		}
		list = append(list, enc)
	}
	payload, err := w.buildPayload(context.Background(), &BuildPayloadArgs{
		Parent:        b.chain.CurrentBlock().Hash(),
		Timestamp:     uint64(time.Now().Unix()),
		InclusionList: list,
	}, false)
	if err != nil {
		t.Fatalf("Failed to build payload %v", err)
	}
	txs := payload.ResolveEmpty().ExecutionPayload.Transactions
	if len(txs) != 2 {
		t.Fatalf("empty payload has %d transactions, want 2", len(txs))
	}
	for i, want := range []*types.Transaction{first, second} {
		var tx types.Transaction
		if err := tx.UnmarshalBinary(txs[i]); err != nil {
			t.Fatal(err)
		}
		if tx.Hash() != want.Hash() {
			t.Errorf("transaction %d is %x, want %x", i, tx.Hash(), want.Hash())
		}
	}
	// The full payload starts with the pool's transaction from the same
	// account, and still appends the second one.
	txs = payload.ResolveFull().ExecutionPayload.Transactions
	var last types.Transaction
	if err := last.UnmarshalBinary(txs[len(txs)-1]); err != nil {
		t.Fatal(err)
	}
	if len(txs) != 2 || last.Hash() != second.Hash() {
		t.Errorf("full payload has %d transactions ending in %x, want 2 ending in %x", len(txs), last.Hash(), second.Hash())
	}
}

// payloadGateEngine pauses the first block containing transactions, after the
// inclusion list has been applied. Empty attribute validation is unaffected.
type payloadGateEngine struct {
	consensus.Engine
	calls       int
	entered     chan struct{}
	release     chan struct{}
	fullEntered chan struct{}
	fullRelease chan struct{}
}

func (e *payloadGateEngine) Finalize(chain consensus.ChainHeaderReader, header *types.Header, state vm.StateDB, body *types.Body) {
	if len(body.Transactions) > 0 {
		e.calls++
		switch e.calls {
		case 1:
			close(e.entered)
			<-e.release
		case 2:
			close(e.fullEntered)
			<-e.fullRelease
		}
	}
	e.Engine.Finalize(chain, header, state, body)
}

func TestBuildPayloadAsyncInclusionList(t *testing.T) {
	for _, resolve := range []string{"Resolve", "ResolveEmpty", "ResolveFull"} {
		t.Run(resolve, func(t *testing.T) {
			fake := ethash.NewFaker()
			b := newTestWorkerBackend(t, params.TestChainConfig, fake, rawdb.NewMemoryDatabase(), 0)
			defer b.chain.Stop()
			defer b.txPool.Close()
			tx := pendingTxs[0] // Only in the inclusion list, never added to the pool.
			enc, err := tx.MarshalBinary()
			if err != nil {
				t.Fatal(err)
			}
			synctest.Test(t, func(t *testing.T) {
				gate := &payloadGateEngine{
					Engine: fake, entered: make(chan struct{}), release: make(chan struct{}),
					fullEntered: make(chan struct{}), fullRelease: make(chan struct{}),
				}
				w := New(b, testConfig, gate)
				type buildResult struct {
					payload *Payload
					err     error
				}
				built := make(chan buildResult, 1)
				var payload *Payload
				var release sync.Once
				var fullRelease sync.Once
				defer func() {
					release.Do(func() { close(gate.release) })
					fullRelease.Do(func() { close(gate.fullRelease) })
					if payload == nil {
						payload = (<-built).payload
					}
					if payload != nil {
						payload.Resolve()
					}
				}()
				go func() {
					p, err := w.buildPayload(context.Background(), &BuildPayloadArgs{
						Parent:        b.chain.CurrentBlock().Hash(),
						Timestamp:     1,
						InclusionList: [][]byte{enc},
					}, true)
					built <- buildResult{p, err}
				}()
				<-gate.entered
				synctest.Wait()
				select {
				case result := <-built:
					payload = result.payload
					if result.err != nil {
						t.Fatal(result.err)
					}
				default:
					t.Fatal("buildPayload waits for inclusion list execution before returning the payload ID")
				}
				resolved := make(chan *engine.ExecutionPayloadEnvelope, 1)
				go func() {
					switch resolve {
					case "Resolve":
						resolved <- payload.Resolve()
					case "ResolveEmpty":
						resolved <- payload.ResolveEmpty()
					case "ResolveFull":
						resolved <- payload.ResolveFull()
					}
				}()
				synctest.Wait()
				select {
				case <-resolved:
					t.Fatal("resolver returned before the inclusion list payload was ready")
				default:
				}
				release.Do(func() { close(gate.release) })
				if resolve == "ResolveFull" {
					<-gate.fullEntered
					synctest.Wait()
					if len(resolved) != 0 {
						t.Fatal("ResolveFull returned after only the initial payload became ready")
					}
				}
				fullRelease.Do(func() { close(gate.fullRelease) })
				envelope := <-resolved
				if envelope == nil || len(envelope.ExecutionPayload.Transactions) != 1 {
					t.Fatalf("missing inclusion list transaction: %v", envelope)
				}
				if envelope.Witness == nil {
					t.Fatal("inclusion list payload lost its witness")
				}
				var got types.Transaction
				if err := got.UnmarshalBinary(envelope.ExecutionPayload.Transactions[0]); err != nil {
					t.Fatal(err)
				}
				if got.Hash() != tx.Hash() {
					t.Fatalf("transaction %s, want %s", got.Hash(), tx.Hash())
				}
				if first, second := payload.Resolve(), payload.Resolve(); !reflect.DeepEqual(first, second) {
					t.Fatal("repeated resolution changed the payload")
				}
			})
		})
	}
}

type payloadFailEngine struct {
	consensus.Engine
	calls    int
	failFrom int
	entered  chan struct{}
	release  chan struct{}
}

func (e *payloadFailEngine) Prepare(chain consensus.ChainHeaderReader, header *types.Header) error {
	e.calls++
	if e.calls >= e.failFrom {
		if e.calls == e.failFrom && e.entered != nil {
			close(e.entered)
			<-e.release
		}
		return errors.New("payload build failed")
	}
	return e.Engine.Prepare(chain, header)
}

func TestBuildPayloadInclusionListFailure(t *testing.T) {
	fake := ethash.NewFaker()
	b := newTestWorkerBackend(t, params.TestChainConfig, fake, rawdb.NewMemoryDatabase(), 0)
	defer b.chain.Stop()
	defer b.txPool.Close()
	synctest.Test(t, func(t *testing.T) {
		gate := &payloadFailEngine{Engine: fake, failFrom: 2, entered: make(chan struct{}), release: make(chan struct{})}
		w := New(b, testConfig, gate)
		payload, err := w.buildPayload(context.Background(), &BuildPayloadArgs{
			Parent:        b.chain.CurrentBlock().Hash(),
			Timestamp:     1,
			InclusionList: [][]byte{{1}},
		}, false)
		if err != nil {
			t.Fatal(err)
		}
		var release sync.Once
		defer func() {
			release.Do(func() { close(gate.release) })
			payload.Resolve()
		}()
		<-gate.entered
		results := make(chan *engine.ExecutionPayloadEnvelope, 3)
		go func() { results <- payload.Resolve() }()
		go func() { results <- payload.ResolveEmpty() }()
		go func() { results <- payload.ResolveFull() }()
		synctest.Wait()
		if len(results) != 0 {
			t.Fatal("resolver returned before the initial build completed")
		}
		release.Do(func() { close(gate.release) })
		for range 3 {
			if result := <-results; result != nil {
				t.Fatal("failed initial build returned a provisional payload")
			}
		}
		if payload.Resolve() != nil || payload.ResolveFull() != nil || payload.ResolveEmpty() != nil {
			t.Fatal("failed payload became available on repeated resolution")
		}
	})
}

func TestBuildPayloadFullFailureTimeout(t *testing.T) {
	fake := ethash.NewFaker()
	b := newTestWorkerBackend(t, params.TestChainConfig, fake, rawdb.NewMemoryDatabase(), 0)
	defer b.chain.Stop()
	defer b.txPool.Close()
	synctest.Test(t, func(t *testing.T) {
		w := New(b, testConfig, &payloadFailEngine{Engine: fake, failFrom: 2})
		payload, err := w.buildPayload(context.Background(), &BuildPayloadArgs{
			Parent: b.chain.CurrentBlock().Hash(), Timestamp: 1,
		}, false)
		if err != nil {
			t.Fatal(err)
		}
		defer payload.Resolve()
		// All full builds fail. Virtual time advances to the build deadline;
		// no later update can satisfy this waiter.
		if payload.ResolveFull() != nil {
			t.Fatal("failed full builds returned a payload")
		}
		if payload.Resolve() == nil {
			t.Fatal("full build timeout discarded the valid initial payload")
		}
	})
}

func TestPayloadResolveStopsFullWaiter(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		payload := newPayload(types.NewBlockWithHeader(&types.Header{Number: big.NewInt(1)}), nil, nil, engine.PayloadID{})
		resolved := make(chan *engine.ExecutionPayloadEnvelope, 1)
		go func() { resolved <- payload.ResolveFull() }()
		synctest.Wait()
		if payload.Resolve() == nil {
			t.Fatal("initial payload was not delivered")
		}
		synctest.Wait()
		select {
		case envelope := <-resolved:
			if envelope != nil {
				t.Fatal("full waiter returned an unbuilt payload")
			}
		default:
			t.Fatal("delivery did not wake the full payload waiter")
		}
	})
}

func TestBuildPayloadInvalidAttributes(t *testing.T) {
	w, b := newTestWorker(t, params.TestChainConfig, ethash.NewFaker(), rawdb.NewMemoryDatabase(), 0)
	defer b.chain.Stop()
	defer b.txPool.Close()
	for _, test := range []struct {
		name   string
		modify func(*BuildPayloadArgs)
		want   string
	}{
		{"parent", func(a *BuildPayloadArgs) { a.Parent = common.Hash{1} }, "missing parent"},
		{"timestamp", func(a *BuildPayloadArgs) { a.Timestamp = 0 }, "invalid timestamp"},
		{"withdrawals", func(a *BuildPayloadArgs) { a.Withdrawals = types.Withdrawals{} }, "unexpected withdrawals before shanghai"},
	} {
		t.Run(test.name, func(t *testing.T) {
			args := &BuildPayloadArgs{Parent: b.chain.CurrentBlock().Hash(), Timestamp: 1, InclusionList: [][]byte{{1}}}
			test.modify(args)
			payload, err := w.buildPayload(context.Background(), args, false)
			if payload != nil || err == nil || !strings.Contains(err.Error(), test.want) {
				t.Fatalf("payload %v, error %v; want synchronous %q", payload, err, test.want)
			}
		})
	}
	t.Run("slot", func(t *testing.T) {
		config := *w.chainConfig
		config.AmsterdamTime = new(uint64)
		w.chainConfig = &config
		payload, err := w.buildPayload(context.Background(), &BuildPayloadArgs{
			Parent: b.chain.CurrentBlock().Hash(), Timestamp: 1, InclusionList: [][]byte{{1}},
		}, false)
		if payload != nil || err == nil || !strings.Contains(err.Error(), "no slot number set post-amsterdam") {
			t.Fatalf("payload %v, error %v; want synchronous missing slot error", payload, err)
		}
	})
}

func TestPayloadId(t *testing.T) {
	t.Parallel()
	ids := make(map[string]int)
	for i, tt := range []*BuildPayloadArgs{
		{
			Parent:       common.Hash{1},
			Timestamp:    1,
			Random:       common.Hash{0x1},
			FeeRecipient: common.Address{0x1},
		},
		// Different parent
		{
			Parent:       common.Hash{2},
			Timestamp:    1,
			Random:       common.Hash{0x1},
			FeeRecipient: common.Address{0x1},
		},
		// Different timestamp
		{
			Parent:       common.Hash{2},
			Timestamp:    2,
			Random:       common.Hash{0x1},
			FeeRecipient: common.Address{0x1},
		},
		// Different Random
		{
			Parent:       common.Hash{2},
			Timestamp:    2,
			Random:       common.Hash{0x2},
			FeeRecipient: common.Address{0x1},
		},
		// Different fee-recipient
		{
			Parent:       common.Hash{2},
			Timestamp:    2,
			Random:       common.Hash{0x2},
			FeeRecipient: common.Address{0x2},
		},
		// Different withdrawals (non-empty)
		{
			Parent:       common.Hash{2},
			Timestamp:    2,
			Random:       common.Hash{0x2},
			FeeRecipient: common.Address{0x2},
			Withdrawals: []*types.Withdrawal{
				{
					Index:     0,
					Validator: 0,
					Address:   common.Address{},
					Amount:    0,
				},
			},
		},
		// Different withdrawals (non-empty)
		{
			Parent:       common.Hash{2},
			Timestamp:    2,
			Random:       common.Hash{0x2},
			FeeRecipient: common.Address{0x2},
			Withdrawals: []*types.Withdrawal{
				{
					Index:     2,
					Validator: 0,
					Address:   common.Address{},
					Amount:    0,
				},
			},
		},
		// Different inclusion list (non-empty)
		{
			Parent:        common.Hash{2},
			Timestamp:     2,
			Random:        common.Hash{0x2},
			FeeRecipient:  common.Address{0x2},
			InclusionList: [][]byte{{0x1}},
		},
		// Different inclusion list (non-empty)
		{
			Parent:        common.Hash{2},
			Timestamp:     2,
			Random:        common.Hash{0x2},
			FeeRecipient:  common.Address{0x2},
			InclusionList: [][]byte{{0x2}},
		},
	} {
		id := tt.Id().String()
		if prev, exists := ids[id]; exists {
			t.Errorf("ID collision, case %d and case %d: id %v", prev, i, id)
		}
		ids[id] = i
	}
}
