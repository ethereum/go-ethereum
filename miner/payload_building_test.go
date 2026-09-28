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
	"bytes"
	"context"
	"math/big"
	"reflect"
	"testing"
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
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/crypto/kzg4844"
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
	} {
		id := tt.Id().String()
		if prev, exists := ids[id]; exists {
			t.Errorf("ID collision, case %d and case %d: id %v", prev, i, id)
		}
		ids[id] = i
	}
}

func TestBuildPayloadWithBlobCarryingFrameTx(t *testing.T) {
	var (
		config        = *params.MergedTestChainConfig
		zeroTime      = uint64(0)
		consensusCore = beacon.New(ethash.NewFaker())
		senderKey, _  = crypto.GenerateKey()
		sender        = crypto.PubkeyToAddress(senderKey.PublicKey)
		blobs         = []kzg4844.Blob{{0x01}, {0x02}}
		commitments   []kzg4844.Commitment
		cellProofs    []kzg4844.Proof
		slotNumber    = uint64(1)
		beaconRoot    = common.Hash{}
	)
	config.AmsterdamTime = &zeroTime
	config.BogotaTime = &zeroTime
	alloc := core.SystemContractAllocs()
	alloc[sender] = types.Account{Balance: big.NewInt(params.Ether)}
	alloc[params.FrameTxExpiryVerifier] = types.Account{Code: params.FrameTxExpiryVerifierCode, Balance: big.NewInt(0)}
	genesis := &core.Genesis{Config: &config, Alloc: alloc, Difficulty: common.Big0}
	chain, err := core.NewBlockChain(rawdb.NewMemoryDatabase(), genesis, consensusCore, nil)
	if err != nil {
		t.Fatalf("failed to create chain: %v", err)
	}
	defer chain.Stop()
	legacyPool := legacypool.New(testTxPoolConfig, chain)
	pool, err := txpool.New(testTxPoolConfig.PriceLimit, chain, []txpool.SubPool{legacyPool})
	if err != nil {
		t.Fatalf("failed to create txpool: %v", err)
	}
	defer pool.Close()

	for i := range blobs {
		commitment, err := kzg4844.BlobToCommitment(&blobs[i])
		if err != nil {
			t.Fatalf("failed to commit to blob: %v", err)
		}
		proofs, err := kzg4844.ComputeCellProofs(&blobs[i])
		if err != nil {
			t.Fatalf("failed to compute cell proofs: %v", err)
		}
		commitments = append(commitments, commitment)
		cellProofs = append(cellProofs, proofs...)
	}
	sidecar := types.NewBlobTxSidecar(types.BlobSidecarVersion1, blobs, commitments, cellProofs)
	blobHashes := sidecar.BlobHashes()
	frameTx := &types.FrameTx{
		ChainID: uint256.MustFromBig(config.ChainID),
		Sender:  sender,
		Frames: []types.Frame{{
			Mode:      types.ModeVerify,
			Flags:     types.ApproveExecutionAndPayment,
			GasLimits: types.Limits{Execution: 100_000},
			Value:     uint256.NewInt(0),
		}},
		Signatures: types.SignatureList{{
			Scheme: types.FrameTxSchemeSecp256k1,
			Signer: sender.Bytes(),
		}},
		Fees: types.Fees{
			MaxPriorityFeePerGas: uint256.NewInt(params.GWei),
			MaxFeePerGas:         uint256.NewInt(10 * params.GWei),
			MaxFeePerBlobGas:     uint256.NewInt(params.GWei),
		},
		BlobVersionedHashes: blobHashes,
	}
	sigHash := types.LatestSigner(&config).Hash(types.NewTx(frameTx))
	sig, err := crypto.Sign(sigHash[:], senderKey)
	if err != nil {
		t.Fatalf("failed to sign frame transaction: %v", err)
	}
	frameTx.Signatures[0].Signature = append([]byte{sig[64]}, sig[:64]...)
	if errs := pool.Add([]*types.Transaction{types.NewTx(frameTx).WithBlobTxSidecar(sidecar)}, true); errs[0] != nil {
		t.Fatalf("failed to add frame transaction: %v", errs[0])
	}

	miner := New(&testWorkerBackend{chain: chain, txPool: pool, genesis: genesis}, testConfig, consensusCore)
	result := miner.generateWork(context.Background(), &generateParams{
		timestamp:   chain.CurrentBlock().Time + 12,
		parentHash:  chain.CurrentBlock().Hash(),
		withdrawals: types.Withdrawals{},
		beaconRoot:  &beaconRoot,
		slotNum:     &slotNumber,
	}, false)
	if result.err != nil {
		t.Fatalf("failed to build block: %v", result.err)
	}
	block := result.block
	if len(block.Transactions()) != 1 {
		t.Fatalf("expected the frame transaction in the block, got %d transactions", len(block.Transactions()))
	}
	if want := uint64(len(blobHashes) * params.BlobTxBlobGasPerBlob); *block.BlobGasUsed() != want {
		t.Fatalf("header blob gas used mismatch: have %d, want %d", *block.BlobGasUsed(), want)
	}
	envelope := engine.BlockToExecutableData(block, result.fees, result.sidecars, result.requests)
	bundle := envelope.BlobsBundle
	if len(bundle.Blobs) != len(blobs) || len(bundle.Commitments) != len(commitments) || len(bundle.Proofs) != len(cellProofs) {
		t.Fatalf("blobs bundle size mismatch: have %d blobs, %d commitments, %d proofs", len(bundle.Blobs), len(bundle.Commitments), len(bundle.Proofs))
	}
	for i := range blobs {
		if !bytes.Equal(bundle.Blobs[i], blobs[i][:]) || !bytes.Equal(bundle.Commitments[i], commitments[i][:]) {
			t.Fatalf("blobs bundle entry %d does not match the frame transaction sidecar", i)
		}
	}
	for i := range cellProofs {
		if !bytes.Equal(bundle.Proofs[i], cellProofs[i][:]) {
			t.Fatalf("blobs bundle proof %d does not match the frame transaction sidecar", i)
		}
	}
	payloadBlock, err := engine.ExecutableDataToBlock(*envelope.ExecutionPayload, blobHashes, &beaconRoot, envelope.Requests)
	if err != nil {
		t.Fatalf("payload with the frame transaction versioned hashes rejected: %v", err)
	}
	if _, err := chain.InsertChain(types.Blocks{payloadBlock}); err != nil {
		t.Fatalf("built block failed validation: %v", err)
	}
}
