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

package downloader

import (
	"bytes"
	"math/big"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/consensus"
	"github.com/ethereum/go-ethereum/consensus/beacon"
	"github.com/ethereum/go-ethereum/consensus/ethash"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/eth/ethconfig"
	"github.com/ethereum/go-ethereum/eth/protocols/eth"
	"github.com/ethereum/go-ethereum/eth/protocols/snap"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/params"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/ethereum/go-ethereum/trie"
)

// downloadTester is a test simulator for mocking out local block chain.
type downloadTester struct {
	db         ethdb.Database
	chain      *core.BlockChain
	downloader *Downloader

	peers map[string]*downloadTesterPeer
	lock  sync.RWMutex
}

// newTester creates a new downloader test mocker.
func newTester(t *testing.T, mode ethconfig.SyncMode) *downloadTester {
	return newTesterWithNotification(t, mode, nil)
}

// newTesterWithNotification creates a new downloader test mocker (snap/1).
func newTesterWithNotification(t *testing.T, mode ethconfig.SyncMode, success func()) *downloadTester {
	return newTesterWithSnap(t, mode, success, false)
}

// newTesterWithSnap is like newTesterWithNotification but selects the snap/2
// state syncer when snapV2 is set.
func newTesterWithSnap(t *testing.T, mode ethconfig.SyncMode, success func(), snapV2 bool) *downloadTester {
	gspec := &core.Genesis{
		Config:  params.TestChainConfig,
		Alloc:   types.GenesisAlloc{testAddress: {Balance: big.NewInt(1000000000000000)}},
		BaseFee: big.NewInt(params.InitialBaseFee),
	}
	return newTesterWithGenesis(t, mode, success, snapV2, gspec, ethash.NewFaker())
}

// newTesterWithGenesis is like newTesterWithSnap, but the local chain is built
// from an arbitrary genesis specification and consensus engine.
func newTesterWithGenesis(t *testing.T, mode ethconfig.SyncMode, success func(), snapV2 bool, gspec *core.Genesis, engine consensus.Engine) *downloadTester {
	db, err := rawdb.Open(rawdb.NewMemoryDatabase(), rawdb.OpenOptions{})
	if err != nil {
		panic(err)
	}
	t.Cleanup(func() {
		db.Close()
	})
	chain, err := core.NewBlockChain(db, gspec, engine, nil)
	if err != nil {
		panic(err)
	}
	tester := &downloadTester{
		db:    db,
		chain: chain,
		peers: make(map[string]*downloadTesterPeer),
	}
	tester.downloader = New(db, mode, tester.chain, tester.dropPeer, success, snapV2)
	return tester
}

// terminate aborts any operations on the embedded downloader and releases all
// held resources.
func (dl *downloadTester) terminate() {
	dl.downloader.Terminate()
	dl.chain.Stop()
}

// newPeer registers a new block download source into the downloader.
func (dl *downloadTester) newPeer(id string, version uint, blocks []*types.Block) *downloadTesterPeer {
	return dl.newPeerWithChain(id, version, newTestBlockchain(blocks), nil)
}

// newPeerWithChain registers a new block download source into the downloader,
// serving content from the given pre-assembled chain. An optional gate can be
// specified to delay body deliveries until the respective access lists arrive.
func (dl *downloadTester) newPeerWithChain(id string, version uint, chain *core.BlockChain, gate *balGate) *downloadTesterPeer {
	dl.lock.Lock()
	defer dl.lock.Unlock()

	peer := &downloadTesterPeer{
		dl:             dl,
		id:             id,
		chain:          chain,
		withholdBodies: make(map[common.Hash]struct{}),
		balGate:        gate,
		dropped:        make(chan error, 1),
	}
	dl.peers[id] = peer

	if err := dl.downloader.RegisterPeer(id, version, peer); err != nil {
		panic(err)
	}
	if err := dl.downloader.snapSyncer.Register(peer); err != nil {
		panic(err)
	}
	return peer
}

// dropPeer simulates a hard peer removal from the connection pool.
func (dl *downloadTester) dropPeer(id string) {
	dl.lock.Lock()
	defer dl.lock.Unlock()

	delete(dl.peers, id)
	dl.downloader.snapSyncer.Unregister(id)
	dl.downloader.UnregisterPeer(id)
}

type downloadTesterPeer struct {
	dl             *downloadTester
	withholdBodies map[common.Hash]struct{}
	corruptBodies  bool     // if set, the peer serves incorrect blocks
	balGate        *balGate // if set, body deliveries wait for the access lists
	id             string
	chain          *core.BlockChain

	dropped chan error // signaled when res.Done receives an error
}

// balGate delays the body delivery of a set of blocks until their access lists
// were delivered into the downloader's queue. Access lists are a best-effort
// component that the queue never waits on, so without external serialization a
// test cannot assert their attachment deterministically.
type balGate struct {
	lock    sync.Mutex
	cond    *sync.Cond
	pending map[common.Hash]struct{} // blocks whose access list was not yet delivered
}

func newBALGate(hashes []common.Hash) *balGate {
	g := &balGate{
		pending: make(map[common.Hash]struct{}),
	}
	g.cond = sync.NewCond(&g.lock)
	for _, hash := range hashes {
		g.pending[hash] = struct{}{}
	}
	return g
}

// served flags the access lists of the given blocks as delivered.
func (g *balGate) served(hashes []common.Hash) {
	g.lock.Lock()
	defer g.lock.Unlock()

	for _, hash := range hashes {
		delete(g.pending, hash)
	}
	g.cond.Broadcast()
}

// wait blocks until the access lists of all the given blocks were delivered.
func (g *balGate) wait(hashes []common.Hash) {
	g.lock.Lock()
	defer g.lock.Unlock()

	for {
		blocked := false
		for _, hash := range hashes {
			if _, ok := g.pending[hash]; ok {
				blocked = true
				break
			}
		}
		if !blocked {
			return
		}
		g.cond.Wait()
	}
}

func unmarshalRlpHeaders(rlpdata []rlp.RawValue) []*types.Header {
	var headers = make([]*types.Header, len(rlpdata))
	for i, data := range rlpdata {
		var h types.Header
		if err := rlp.DecodeBytes(data, &h); err != nil {
			panic(err)
		}
		headers[i] = &h
	}
	return headers
}

// RequestHeadersByHash constructs a GetBlockHeaders function based on a hashed
// origin; associated with a particular peer in the download tester. The returned
// function can be used to retrieve batches of headers from the particular peer.
func (dlp *downloadTesterPeer) RequestHeadersByHash(origin common.Hash, amount int, skip int, reverse bool, sink chan *eth.Response) (*eth.Request, error) {
	// Service the header query via the live handler code
	rlpHeaders := eth.ServiceGetBlockHeadersQuery(dlp.chain, &eth.GetBlockHeadersRequest{
		Origin: eth.HashOrNumber{
			Hash: origin,
		},
		Amount:  uint64(amount),
		Skip:    uint64(skip),
		Reverse: reverse,
	}, nil)
	headers := unmarshalRlpHeaders(rlpHeaders)
	hashes := make([]common.Hash, len(headers))
	for i, header := range headers {
		hashes[i] = header.Hash()
	}
	// Deliver the headers to the downloader
	req := &eth.Request{
		Peer: dlp.id,
	}
	res := &eth.Response{
		Req:  req,
		Res:  (*eth.BlockHeadersRequest)(&headers),
		Meta: hashes,
		Time: 1,
		Done: make(chan error, 1), // Ignore the returned status
	}
	go func() {
		sink <- res
	}()
	return req, nil
}

// RequestHeadersByNumber constructs a GetBlockHeaders function based on a numbered
// origin; associated with a particular peer in the download tester. The returned
// function can be used to retrieve batches of headers from the particular peer.
func (dlp *downloadTesterPeer) RequestHeadersByNumber(origin uint64, amount int, skip int, reverse bool, sink chan *eth.Response) (*eth.Request, error) {
	// Service the header query via the live handler code
	rlpHeaders := eth.ServiceGetBlockHeadersQuery(dlp.chain, &eth.GetBlockHeadersRequest{
		Origin: eth.HashOrNumber{
			Number: origin,
		},
		Amount:  uint64(amount),
		Skip:    uint64(skip),
		Reverse: reverse,
	}, nil)
	headers := unmarshalRlpHeaders(rlpHeaders)
	hashes := make([]common.Hash, len(headers))
	for i, header := range headers {
		hashes[i] = header.Hash()
	}
	// Deliver the headers to the downloader
	req := &eth.Request{
		Peer: dlp.id,
	}
	res := &eth.Response{
		Req:  req,
		Res:  (*eth.BlockHeadersRequest)(&headers),
		Meta: hashes,
		Time: 1,
		Done: make(chan error, 1), // Ignore the returned status
	}
	go func() {
		sink <- res
	}()
	return req, nil
}

// RequestBodies constructs a getBlockBodies method associated with a particular
// peer in the download tester. The returned function can be used to retrieve
// batches of block bodies from the particularly requested peer.
func (dlp *downloadTesterPeer) RequestBodies(hashes []common.Hash, sink chan *eth.Response) (*eth.Request, error) {
	blobs := eth.ServiceGetBlockBodiesQuery(dlp.chain, hashes)

	bodies := make([]*types.Body, len(blobs))
	ethbodies := make([]eth.BlockBody, len(blobs))
	for i, blob := range blobs {
		bodies[i] = new(types.Body)
		rlp.DecodeBytes(blob, bodies[i])
		rlp.DecodeBytes(blob, &ethbodies[i])
	}
	var (
		txsHashes        = make([]common.Hash, len(bodies))
		uncleHashes      = make([]common.Hash, len(bodies))
		withdrawalHashes = make([]common.Hash, len(bodies))
	)
	hasher := trie.NewStackTrie(nil)
	for i, body := range bodies {
		hash := types.DeriveSha(types.Transactions(body.Transactions), hasher)
		if _, ok := dlp.withholdBodies[hash]; ok {
			txsHashes = append(txsHashes[:i], txsHashes[i+1:]...)
			uncleHashes = append(uncleHashes[:i], uncleHashes[i+1:]...)
			continue
		}
		txsHashes[i] = hash
		uncleHashes[i] = types.CalcUncleHash(body.Uncles)
		if body.Withdrawals != nil {
			withdrawalHashes[i] = types.DeriveSha(types.Withdrawals(body.Withdrawals), hasher)
		}
	}
	if dlp.corruptBodies {
		for i := range txsHashes {
			txsHashes[i] = common.Hash{0xff}
		}
	}
	req := &eth.Request{
		Peer: dlp.id,
	}
	res := &eth.Response{
		Req: req,
		Res: (*eth.BlockBodiesResponse)(&ethbodies),
		Meta: eth.BlockBodyHashes{
			TransactionRoots: txsHashes,
			UncleHashes:      uncleHashes,
			WithdrawalRoots:  withdrawalHashes,
		},
		Time: 1,
		Done: make(chan error),
	}
	go func() {
		// If gated, hold back the bodies until the access lists of the
		// requested blocks were delivered
		if dlp.balGate != nil {
			dlp.balGate.wait(hashes)
		}
		sink <- res
		if err := <-res.Done; err != nil {
			select {
			case dlp.dropped <- err:
			default:
			}
		}
	}()
	return req, nil
}

// RequestReceipts constructs a getReceipts method associated with a particular
// peer in the download tester. The returned function can be used to retrieve
// batches of block receipts from the particularly requested peer.
func (dlp *downloadTesterPeer) RequestReceipts(hashes []common.Hash, gasUsed []uint64, timestamps []uint64, sink chan *eth.Response) (*eth.Request, error) {
	blobs := eth.ServiceGetReceiptsQuery69(dlp.chain, hashes)
	receipts := make([]types.Receipts, blobs.Len())

	// compute hashes
	hashes = make([]common.Hash, blobs.Len())
	hasher := trie.NewStackTrie(nil)
	receiptLists, err := blobs.Items()
	if err != nil {
		panic(err)
	}
	for i, rl := range receiptLists {
		hashes[i] = types.DeriveSha(rl.Derivable(), hasher)
	}

	// deliver the response right away
	resp := eth.ReceiptsRLPResponse(types.EncodeBlockReceiptLists(receipts))
	res := &eth.Response{
		Req:  &eth.Request{Peer: dlp.id},
		Res:  &resp,
		Meta: hashes,
		Time: 1,
		Done: make(chan error, 1), // Ignore the returned status
	}
	go func() {
		sink <- res
	}()
	return res.Req, nil
}

// RequestBALs constructs a getBlockAccessLists method associated with a
// particular peer in the download tester. The returned function can be used to
// retrieve batches of block access lists from the particularly requested peer.
func (dlp *downloadTesterPeer) RequestBALs(hashes []common.Hash, sink chan *eth.Response) (*eth.Request, error) {
	var (
		bals   = make([]rlp.RawValue, 0, len(hashes))
		served = make([]common.Hash, 0, len(hashes))
	)
	for _, hash := range hashes {
		data := dlp.chain.GetAccessListRLP(hash)
		if len(data) == 0 {
			// The signal for a missing access list is the empty string
			bals = append(bals, rlp.EmptyString)
			continue
		}
		bals = append(bals, data)
		served = append(served, hash)
	}
	// compute the content hashes, zero hash for unavailable entries
	meta := make([]common.Hash, len(bals))
	for i, data := range bals {
		if bytes.Equal(data, rlp.EmptyString) {
			continue
		}
		meta[i] = crypto.Keccak256Hash(data)
	}
	// deliver the response right away
	resp := eth.BlockAccessListResponse(bals)
	res := &eth.Response{
		Req:  &eth.Request{Peer: dlp.id},
		Res:  &resp,
		Meta: meta,
		Time: 1,
		Done: make(chan error, 1),
	}
	go func() {
		sink <- res

		// If gated, unblock the body deliveries of the served blocks, but only
		// after the access lists were fully processed by the queue
		if dlp.balGate != nil {
			<-res.Done
			dlp.balGate.served(served)
		}
	}()
	return res.Req, nil
}

// ID retrieves the peer's unique identifier.
func (dlp *downloadTesterPeer) ID() string {
	return dlp.id
}

// RequestAccountRange fetches a batch of accounts rooted in a specific account
// trie, starting with the origin.
func (dlp *downloadTesterPeer) RequestAccountRange(id uint64, root, origin, limit common.Hash, bytes int) error {
	// Create the request and service it
	req := &snap.GetAccountRangePacket{
		ID:     id,
		Root:   root,
		Origin: origin,
		Limit:  limit,
		Bytes:  uint64(bytes),
	}
	slimaccs, proofs := snap.ServiceGetAccountRangeQuery(dlp.chain, req)

	// We need to convert to non-slim format, delegate to the packet code
	res := &snap.AccountRangePacket{
		ID:       id,
		Accounts: slimaccs,
		Proof:    proofs,
	}
	hashes, accounts, _ := res.Unpack()

	go dlp.dl.downloader.snapSyncer.OnAccounts(dlp, id, hashes, accounts, proofs)
	return nil
}

// RequestStorageRanges fetches a batch of storage slots belonging to one or
// more accounts. If slots from only one account is requested, an origin marker
// may also be used to retrieve from there.
func (dlp *downloadTesterPeer) RequestStorageRanges(id uint64, root common.Hash, accounts []common.Hash, origin, limit []byte, bytes int) error {
	// Create the request and service it
	req := &snap.GetStorageRangesPacket{
		ID:       id,
		Accounts: accounts,
		Root:     root,
		Origin:   origin,
		Limit:    limit,
		Bytes:    uint64(bytes),
	}
	storage, proofs := snap.ServiceGetStorageRangesQuery(dlp.chain, req)

	// We need to convert to demultiplex, delegate to the packet code
	res := &snap.StorageRangesPacket{
		ID:    id,
		Slots: storage,
		Proof: proofs,
	}
	hashes, slots := res.Unpack()

	go dlp.dl.downloader.snapSyncer.OnStorage(dlp, id, hashes, slots, proofs)
	return nil
}

// RequestByteCodes fetches a batch of bytecodes by hash.
func (dlp *downloadTesterPeer) RequestByteCodes(id uint64, hashes []common.Hash, bytes int) error {
	req := &snap.GetByteCodesPacket{
		ID:     id,
		Hashes: hashes,
		Bytes:  uint64(bytes),
	}
	codes := snap.ServiceGetByteCodesQuery(dlp.chain, req)
	go dlp.dl.downloader.snapSyncer.OnByteCodes(dlp, id, codes)
	return nil
}

// RequestTrieNodes fetches a batch of trie nodes (snap/1 healing). snap/2 never
// issues these, but the method is required to satisfy snap.SyncPeerV2.
func (dlp *downloadTesterPeer) RequestTrieNodes(id uint64, root common.Hash, count int, paths []snap.TrieNodePathSet, bytes int) error {
	encPaths, err := rlp.EncodeToRawList(paths)
	if err != nil {
		panic(err)
	}
	req := &snap.GetTrieNodesPacket{
		ID:    id,
		Root:  root,
		Paths: encPaths,
		Bytes: uint64(bytes),
	}
	nodes, _ := snap.ServiceGetTrieNodesQuery(dlp.chain, req)
	go dlp.dl.downloader.snapSyncer.OnTrieNodes(dlp, id, nodes)
	return nil
}

// RequestAccessLists fetches a batch of BALs by block hash.
func (dlp *downloadTesterPeer) RequestAccessLists(id uint64, hashes []common.Hash, bytes int) error {
	req := &snap.GetAccessListsPacket{
		ID:     id,
		Hashes: hashes,
		Bytes:  uint64(bytes),
	}
	als := snap.ServiceGetAccessListsQuery(dlp.chain, req)
	go dlp.dl.downloader.snapSyncer.OnAccessLists(dlp, id, als)
	return nil
}

// Log retrieves the peer's own contextual logger.
func (dlp *downloadTesterPeer) Log() log.Logger {
	return log.New("peer", dlp.id)
}

// assertOwnChain checks if the local chain contains the correct number of items
// of the various chain components.
func assertOwnChain(t *testing.T, tester *downloadTester, length int) {
	// Mark this method as a helper to report errors at callsite, not in here
	t.Helper()

	headers, blocks, receipts := length, length, length
	if hs := int(tester.chain.CurrentHeader().Number.Uint64()) + 1; hs != headers {
		t.Fatalf("synchronised headers mismatch: have %v, want %v", hs, headers)
	}
	if bs := int(tester.chain.CurrentBlock().Number.Uint64()) + 1; bs != blocks {
		t.Fatalf("synchronised blocks mismatch: have %v, want %v", bs, blocks)
	}
	if rs := int(tester.chain.CurrentSnapBlock().Number.Uint64()) + 1; rs != receipts {
		t.Fatalf("synchronised receipts mismatch: have %v, want %v", rs, receipts)
	}
}

func TestCanonicalSynchronisationFull(t *testing.T)   { testCanonSync(t, eth.ETH69, FullSync, false) }
func TestCanonicalSynchronisationSnap(t *testing.T)   { testCanonSync(t, eth.ETH69, SnapSync, false) }
func TestCanonicalSynchronisationSnapV2(t *testing.T) { testCanonSync(t, eth.ETH69, SnapSync, true) }

func testCanonSync(t *testing.T, protocol uint, mode SyncMode, snapV2 bool) {
	success := make(chan struct{})
	tester := newTesterWithSnap(t, mode, func() {
		close(success)
	}, snapV2)
	defer tester.terminate()

	// Create a small enough block chain to download
	chain := testChainBase.shorten(blockCacheMaxItems - 15)
	tester.newPeer("peer", protocol, chain.blocks[1:])

	// Synchronise with the peer and make sure all relevant data was retrieved
	if err := tester.downloader.BeaconSync(chain.blocks[len(chain.blocks)-1].Header(), nil); err != nil {
		t.Fatalf("failed to beacon-sync chain: %v", err)
	}
	select {
	case <-success:
		assertOwnChain(t, tester, len(chain.blocks))
	case <-time.NewTimer(time.Second * 3).C:
		t.Fatalf("Failed to sync chain in three seconds")
	}
}

// makeBALChain constructs a post-merge, Amsterdam-enabled chain whose blocks
// all carry a block access list commitment, along with the genesis needed to
// sync it. Every block contains a transaction so that no block has an empty
// body (empty-body blocks complete without a network retrieval, voiding any
// delivery ordering imposed by the tests).
func makeBALChain(n int) (*core.Genesis, []*types.Block) {
	config := *params.MergedTestChainConfig
	config.AmsterdamTime = new(uint64)

	gspec := &core.Genesis{
		Config: &config,
		Alloc: types.GenesisAlloc{
			testAddress:                      {Balance: new(big.Int).Mul(big.NewInt(1000), big.NewInt(params.Ether))},
			params.BeaconRootsAddress:        {Nonce: 1, Code: params.BeaconRootsCode, Balance: common.Big0},
			params.HistoryStorageAddress:     {Nonce: 1, Code: params.HistoryStorageCode, Balance: common.Big0},
			params.WithdrawalQueueAddress:    {Nonce: 1, Code: params.WithdrawalQueueCode, Balance: common.Big0},
			params.ConsolidationQueueAddress: {Nonce: 1, Code: params.ConsolidationQueueCode, Balance: common.Big0},
			params.BuilderDepositAddress:     {Nonce: 1, Code: params.BuilderDepositCode, Balance: common.Big0},
			params.BuilderExitAddress:        {Nonce: 1, Code: params.BuilderExitCode, Balance: common.Big0},
		},
		BaseFee:    big.NewInt(params.InitialBaseFee),
		Difficulty: common.Big0,
	}
	signer := types.LatestSigner(&config)
	_, blocks, _ := core.GenerateChainWithGenesis(gspec, beacon.New(ethash.NewFaker()), n, func(i int, block *core.BlockGen) {
		// The chain maker only executes the EIP-4788 system call when the
		// beacon root is set explicitly; without it, the generated access
		// lists would not match the ones computed at import.
		block.SetParentBeaconRoot(common.Hash{})

		tx, err := types.SignTx(types.NewTransaction(block.TxNonce(testAddress), common.Address{0x01}, big.NewInt(1000), params.TxGas, block.BaseFee(), nil), signer, testKey)
		if err != nil {
			panic(err)
		}
		block.AddTx(tx)
	})
	return gspec, blocks
}

// Tests that block access lists are retrieved from eth/71+ peers during sync
// and end up attached to the imported blocks; and that syncing against peers
// which cannot (or do not) serve access lists still completes, importing the
// blocks without them.
func TestBALSynchronisationFull(t *testing.T)       { testBALSync(t, FullSync, eth.ETH71) }
func TestBALSynchronisationSnap(t *testing.T)       { testBALSync(t, SnapSync, eth.ETH71) }
func TestBALSynchronisationLegacyPeer(t *testing.T) { testBALSync(t, FullSync, eth.ETH69) }

func testBALSync(t *testing.T, mode SyncMode, protocol uint) {
	gspec, blocks := makeBALChain(96) // long enough for a snap sync pivot below the head

	success := make(chan struct{})
	tester := newTesterWithGenesis(t, mode, func() { close(success) }, false, gspec, beacon.New(ethash.NewFaker()))
	defer tester.terminate()

	// Assemble the serving chain, executing the blocks to persist their access
	// lists.
	peerChain, err := core.NewBlockChain(rawdb.NewMemoryDatabase(), gspec, beacon.New(ethash.NewFaker()), nil)
	if err != nil {
		t.Fatalf("failed to create peer chain: %v", err)
	}
	defer peerChain.Stop()

	if _, err := peerChain.InsertChain(blocks); err != nil {
		t.Fatalf("failed to assemble peer chain: %v", err)
	}
	// Collect the blocks whose access lists the downloader should retrieve
	var eligible []common.Hash
	for _, block := range blocks {
		if hash := block.Header().BlockAccessListHash; hash != nil && *hash != types.EmptyBlockAccessListHash {
			eligible = append(eligible, block.Hash())
		}
	}
	if len(eligible) != len(blocks) {
		t.Fatalf("expected all %d blocks to commit to an access list, got %d", len(blocks), len(eligible))
	}
	// Access lists are best effort and never block the delivery of an otherwise
	// completed block. To assert their attachment deterministically, gate the
	// body deliveries of modern peers on the access lists arriving first.
	var gate *balGate
	if protocol >= eth.ETH71 {
		gate = newBALGate(eligible)
	}
	tester.newPeerWithChain("peer", protocol, peerChain, gate)

	// Track which imported blocks had an access list attached
	var (
		attachLock sync.Mutex
		attached   = make(map[uint64]bool)
	)
	tester.downloader.chainInsertHook = func(results []*fetchResult) {
		attachLock.Lock()
		defer attachLock.Unlock()
		for _, result := range results {
			if result.AccessList.Load() != nil {
				attached[result.Header.Number.Uint64()] = true
			}
		}
	}
	// Synchronise with the peer and make sure all relevant data was retrieved.
	// In snap mode, announce a finalized block too, directing the chain segment
	// below it straight into the ancient store: downloaded access lists must
	// end up retrievable through that path as well.
	var final *types.Header
	if mode == SnapSync {
		final = blocks[len(blocks)/3].Header()
	}
	if err := tester.downloader.BeaconSync(blocks[len(blocks)-1].Header(), final); err != nil {
		t.Fatalf("failed to beacon-sync chain: %v", err)
	}
	select {
	case <-success:
		assertOwnChain(t, tester, len(blocks)+1)
	case <-time.NewTimer(15 * time.Second).C:
		t.Fatalf("failed to sync chain in fifteen seconds")
	}
	attachLock.Lock()
	defer attachLock.Unlock()

	for _, block := range blocks {
		if protocol >= eth.ETH71 {
			// Modern peer: the access list must have been downloaded, attached
			// to the imported block and persisted locally
			if !attached[block.NumberU64()] {
				t.Errorf("block %d: no access list attached at import", block.NumberU64())
			}
			if have, want := tester.chain.GetAccessListRLP(block.Hash()), peerChain.GetAccessListRLP(block.Hash()); !bytes.Equal(have, want) {
				t.Errorf("block %d: persisted access list mismatch: have %x, want %x", block.NumberU64(), have, want)
			}
		} else {
			// Legacy peer: blocks must have been imported without access lists
			if attached[block.NumberU64()] {
				t.Errorf("block %d: unexpected access list attached at import", block.NumberU64())
			}
		}
	}
}

// Tests that if a large batch of blocks are being downloaded, it is throttled
// until the cached blocks are retrieved.
func TestThrottlingFull(t *testing.T) { testThrottling(t, eth.ETH69, FullSync) }
func TestThrottlingSnap(t *testing.T) { testThrottling(t, eth.ETH69, SnapSync) }

func testThrottling(t *testing.T, protocol uint, mode SyncMode) {
	tester := newTester(t, mode)
	defer tester.terminate()

	// Create a long block chain to download and the tester
	targetBlocks := len(testChainBase.blocks) - 1
	tester.newPeer("peer", protocol, testChainBase.blocks[1:])

	// Wrap the importer to allow stepping
	var blocked atomic.Uint32
	proceed := make(chan struct{})
	tester.downloader.chainInsertHook = func(results []*fetchResult) {
		blocked.Store(uint32(len(results)))
		<-proceed
	}
	// Start a synchronisation concurrently
	errc := make(chan error, 1)
	go func() {
		errc <- tester.downloader.BeaconSync(testChainBase.blocks[len(testChainBase.blocks)-1].Header(), nil)
	}()
	// Iteratively take some blocks, always checking the retrieval count
	for {
		// Check the retrieval count synchronously (! reason for this ugly block)
		tester.lock.RLock()
		retrieved := int(tester.chain.CurrentSnapBlock().Number.Uint64()) + 1
		tester.lock.RUnlock()
		if retrieved >= targetBlocks+1 {
			break
		}
		// Wait a bit for sync to throttle itself
		var cached, frozen int
		for start := time.Now(); time.Since(start) < 3*time.Second; {
			time.Sleep(25 * time.Millisecond)

			tester.lock.Lock()
			tester.downloader.queue.lock.Lock()
			tester.downloader.queue.resultCache.lock.Lock()
			{
				cached = tester.downloader.queue.resultCache.countCompleted()
				frozen = int(blocked.Load())
				retrieved = int(tester.chain.CurrentSnapBlock().Number.Uint64()) + 1
			}
			tester.downloader.queue.resultCache.lock.Unlock()
			tester.downloader.queue.lock.Unlock()
			tester.lock.Unlock()

			if cached == blockCacheMaxItems ||
				cached == blockCacheMaxItems-reorgProtHeaderDelay ||
				retrieved+cached+frozen == targetBlocks+1 ||
				retrieved+cached+frozen == targetBlocks+1-reorgProtHeaderDelay {
				break
			}
		}
		// Make sure we filled up the cache, then exhaust it
		time.Sleep(25 * time.Millisecond) // give it a chance to screw up
		tester.lock.RLock()
		retrieved = int(tester.chain.CurrentSnapBlock().Number.Uint64()) + 1
		tester.lock.RUnlock()
		if cached != blockCacheMaxItems && cached != blockCacheMaxItems-reorgProtHeaderDelay && retrieved+cached+frozen != targetBlocks+1 && retrieved+cached+frozen != targetBlocks+1-reorgProtHeaderDelay {
			t.Fatalf("block count mismatch: have %v, want %v (owned %v, blocked %v, target %v)", cached, blockCacheMaxItems, retrieved, frozen, targetBlocks+1)
		}
		// Permit the blocked blocks to import
		if blocked.Load() > 0 {
			blocked.Store(uint32(0))
			proceed <- struct{}{}
		}
	}
	// Check that we haven't pulled more blocks than available
	assertOwnChain(t, tester, targetBlocks+1)
	if err := <-errc; err != nil {
		t.Fatalf("block synchronization failed: %v", err)
	}
}

// Tests that a canceled download wipes all previously accumulated state.
func TestCancelFull(t *testing.T) { testCancel(t, eth.ETH69, FullSync) }
func TestCancelSnap(t *testing.T) { testCancel(t, eth.ETH69, SnapSync) }

func testCancel(t *testing.T, protocol uint, mode SyncMode) {
	complete := make(chan struct{})
	success := func() {
		close(complete)
	}
	tester := newTesterWithNotification(t, mode, success)
	defer tester.terminate()

	chain := testChainBase.shorten(MaxHeaderFetch)
	tester.newPeer("peer", protocol, chain.blocks[1:])

	// Make sure canceling works with a pristine downloader
	tester.downloader.Cancel()
	if !tester.downloader.queue.Idle() {
		t.Errorf("download queue not idle")
	}
	// Synchronise with the peer, but cancel afterwards
	if err := tester.downloader.BeaconSync(chain.blocks[len(chain.blocks)-1].Header(), nil); err != nil {
		t.Fatalf("failed to synchronise blocks: %v", err)
	}
	<-complete
	tester.downloader.Cancel()
	if !tester.downloader.queue.Idle() {
		t.Errorf("download queue not idle")
	}
}

// Tests that if a block is empty (e.g. header only), no body request should be
// made, and instead the header should be assembled into a whole block in itself.
func TestEmptyShortCircuitFull(t *testing.T) { testEmptyShortCircuit(t, eth.ETH69, FullSync) }
func TestEmptyShortCircuitSnap(t *testing.T) { testEmptyShortCircuit(t, eth.ETH69, SnapSync) }

func testEmptyShortCircuit(t *testing.T, protocol uint, mode SyncMode) {
	success := make(chan struct{})
	tester := newTesterWithNotification(t, mode, func() {
		close(success)
	})
	defer tester.terminate()

	// Create a block chain to download
	chain := testChainBase
	tester.newPeer("peer", protocol, chain.blocks[1:])

	// Instrument the downloader to signal body requests
	var bodiesHave, receiptsHave atomic.Int32
	tester.downloader.bodyFetchHook = func(headers []*types.Header) {
		bodiesHave.Add(int32(len(headers)))
	}
	tester.downloader.receiptFetchHook = func(headers []*types.Header) {
		receiptsHave.Add(int32(len(headers)))
	}

	if err := tester.downloader.BeaconSync(chain.blocks[len(chain.blocks)-1].Header(), nil); err != nil {
		t.Fatalf("failed to synchronise blocks: %v", err)
	}
	select {
	case <-success:
		checkProgress(t, tester.downloader, "initial", ethereum.SyncProgress{
			HighestBlock: uint64(len(chain.blocks) - 1),
			CurrentBlock: uint64(len(chain.blocks) - 1),
		})
	case <-time.NewTimer(time.Second * 3).C:
		t.Fatalf("Failed to sync chain in three seconds")
	}
	assertOwnChain(t, tester, len(chain.blocks))

	// Validate the number of block bodies that should have been requested
	bodiesNeeded, receiptsNeeded := 0, 0
	for _, block := range chain.blocks[1:] {
		if len(block.Transactions()) > 0 || len(block.Uncles()) > 0 {
			bodiesNeeded++
		}
	}
	for _, block := range chain.blocks[1:] {
		if mode == SnapSync && len(block.Transactions()) > 0 {
			receiptsNeeded++
		}
	}
	if int(bodiesHave.Load()) != bodiesNeeded {
		t.Errorf("body retrieval count mismatch: have %v, want %v", bodiesHave.Load(), bodiesNeeded)
	}
	if int(receiptsHave.Load()) != receiptsNeeded {
		t.Errorf("receipt retrieval count mismatch: have %v, want %v", receiptsHave.Load(), receiptsNeeded)
	}
}

func checkProgress(t *testing.T, d *Downloader, stage string, want ethereum.SyncProgress) {
	// Mark this method as a helper to report errors at callsite, not in here
	t.Helper()

	p := d.Progress()
	if p.StartingBlock != want.StartingBlock || p.CurrentBlock != want.CurrentBlock || p.HighestBlock != want.HighestBlock {
		t.Fatalf("%s progress mismatch:\nhave %+v\nwant %+v", stage, p, want)
	}
}

// Tests that peers below a pre-configured checkpoint block are prevented from
// being fast-synced from, avoiding potential cheap eclipse attacks.
func TestBeaconSyncFull(t *testing.T) { testBeaconSync(t, eth.ETH69, FullSync) }
func TestBeaconSyncSnap(t *testing.T) { testBeaconSync(t, eth.ETH69, SnapSync) }

func testBeaconSync(t *testing.T, protocol uint, mode SyncMode) {
	var cases = []struct {
		name  string // The name of testing scenario
		local int    // The length of local chain(canonical chain assumed), 0 means genesis is the head
	}{
		{name: "Beacon sync since genesis", local: 0},
		{name: "Beacon sync with short local chain", local: 1},
		{name: "Beacon sync with long local chain", local: blockCacheMaxItems - 15 - fsMinFullBlocks/2},
		{name: "Beacon sync with full local chain", local: blockCacheMaxItems - 15 - 1},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			success := make(chan struct{})
			tester := newTesterWithNotification(t, mode, func() {
				close(success)
			})
			defer tester.terminate()

			chain := testChainBase.shorten(blockCacheMaxItems - 15)
			tester.newPeer("peer", protocol, chain.blocks[1:])

			// Build the local chain segment if it's required
			if c.local > 0 {
				tester.chain.InsertChain(chain.blocks[1 : c.local+1])
			}
			if err := tester.downloader.BeaconSync(chain.blocks[len(chain.blocks)-1].Header(), nil); err != nil {
				t.Fatalf("Failed to beacon sync chain %v %v", c.name, err)
			}
			select {
			case <-success:
				// Ok, downloader fully cancelled after sync cycle
				if bs := int(tester.chain.CurrentBlock().Number.Uint64()) + 1; bs != len(chain.blocks) {
					t.Fatalf("synchronised blocks mismatch: have %v, want %v", bs, len(chain.blocks))
				}
			case <-time.NewTimer(time.Second * 3).C:
				t.Fatalf("Failed to sync chain in three seconds")
			}
		})
	}
}

// TestBeaconSyncRepairFork verifies the end-to-end repair of non-canonical block
// data. The local node sits on fork A, but fork B's blocks below the local head
// are also present by hash (no canonical mapping), as if imported optimistically
// via the engine API. When the beacon chain switches to fork B, sync must not
// anchor on the non-canonical fork-B data; it has to descend to the real common
// ancestor and re-deliver everything, ending with the full fork-B chain present
// and canonical at every height - for both snap and full sync.
func TestBeaconSyncRepairForkFull(t *testing.T) { testBeaconSyncRepairFork(t, eth.ETH69, FullSync) }
func TestBeaconSyncRepairForkSnap(t *testing.T) { testBeaconSyncRepairFork(t, eth.ETH69, SnapSync) }

func testBeaconSyncRepairFork(t *testing.T, protocol uint, mode SyncMode) {
	// Reuse the pre-generated fork chains (new chains can't be generated after the
	// package init). Fork A and fork B share the whole testChainBase prefix and
	// diverge at height len(testChainBase.blocks); fork B (the beacon target) is
	// longer, so it wins the reorg. The exact shortenings used here are the ones
	// registered as peer chains during init.
	localChain := testChainForkLightA.shorten(len(testChainBase.blocks) + 80)
	targetChain := testChainForkLightB.shorten(len(testChainBase.blocks) + MaxHeaderFetch)

	forkPoint := uint64(len(testChainBase.blocks)) // first height the forks differ
	localHead := uint64(len(localChain.blocks) - 1)
	targetHead := uint64(len(targetChain.blocks) - 1)

	success := make(chan struct{})
	tester := newTesterWithNotification(t, mode, func() {
		close(success)
	})
	defer tester.terminate()

	tester.newPeer("peer", protocol, targetChain.blocks[1:])

	// Make fork A the local canonical chain.
	if _, err := tester.chain.InsertChain(localChain.blocks[1 : localHead+1]); err != nil {
		t.Fatalf("failed to build local chain: %v", err)
	}
	// Seed fork B's divergent blocks that sit below the local head as scattered,
	// non-canonical data: full block data present by hash, but the canonical
	// mapping at those heights still points at fork A.
	for n := forkPoint; n <= localHead; n++ {
		b := targetChain.blocks[n]
		rawdb.WriteBlock(tester.db, b)
		rawdb.WriteReceipts(tester.db, b.Hash(), b.NumberU64(), types.Receipts{})
	}

	if err := tester.downloader.BeaconSync(targetChain.blocks[targetHead].Header(), nil); err != nil {
		t.Fatalf("failed to beacon-sync chain: %v", err)
	}
	select {
	case <-success:
	case <-time.NewTimer(10 * time.Second).C:
		t.Fatalf("failed to sync chain in ten seconds")
	}
	// The head must reach fork B's tip.
	if got := tester.chain.CurrentBlock().Number.Uint64(); got != targetHead {
		t.Fatalf("synced head mismatch: have %d, want %d", got, targetHead)
	}
	// Every height must be canonical to fork B and carry complete block data,
	// proving the non-canonical fork-A / seed data was fully reorged out.
	for n := uint64(1); n <= targetHead; n++ {
		want := targetChain.blocks[n].Hash()
		if got := rawdb.ReadCanonicalHash(tester.db, n); got != want {
			t.Fatalf("canonical hash at %d: have %x, want %x", n, got, want)
		}
		if !rawdb.HasHeader(tester.db, want, n) || !rawdb.HasBody(tester.db, want, n) {
			t.Fatalf("incomplete block data at %d after sync", n)
		}
		if !rawdb.HasReceipts(tester.db, want, n) {
			t.Fatalf("missing receipts at %d after sync", n)
		}
	}
}

// Tests that synchronisation progress (origin block number, current block number
// and highest block number) is tracked and updated correctly.
func TestSyncProgressFull(t *testing.T) { testSyncProgress(t, eth.ETH69, FullSync) }
func TestSyncProgressSnap(t *testing.T) { testSyncProgress(t, eth.ETH69, SnapSync) }

func testSyncProgress(t *testing.T, protocol uint, mode SyncMode) {
	success := make(chan struct{})
	tester := newTesterWithNotification(t, mode, func() {
		success <- struct{}{}
	})
	defer tester.terminate()
	checkProgress(t, tester.downloader, "pristine", ethereum.SyncProgress{})

	chain := testChainBase.shorten(blockCacheMaxItems - 15)
	shortChain := chain.shorten(len(chain.blocks) / 2).blocks[1:]

	// Connect to peer that provides all headers and part of the bodies
	faultyPeer := tester.newPeer("peer-half", protocol, shortChain)
	for _, header := range shortChain {
		faultyPeer.withholdBodies[header.Hash()] = struct{}{}
	}

	if err := tester.downloader.BeaconSync(chain.blocks[len(chain.blocks)/2-1].Header(), nil); err != nil {
		t.Fatalf("failed to beacon-sync chain: %v", err)
	}
	select {
	case <-success:
		// Ok, downloader fully cancelled after sync cycle
		checkProgress(t, tester.downloader, "peer-half", ethereum.SyncProgress{
			CurrentBlock: uint64(len(chain.blocks)/2 - 1),
			HighestBlock: uint64(len(chain.blocks)/2 - 1),
		})
	case <-time.NewTimer(time.Second * 3).C:
		t.Fatalf("Failed to sync chain in three seconds")
	}

	// Synchronise all the blocks and check continuation progress
	tester.newPeer("peer-full", protocol, chain.blocks[1:])
	if err := tester.downloader.BeaconSync(chain.blocks[len(chain.blocks)-1].Header(), nil); err != nil {
		t.Fatalf("failed to beacon-sync chain: %v", err)
	}
	startingBlock := uint64(len(chain.blocks)/2 - 1)

	select {
	case <-success:
		// Ok, downloader fully cancelled after sync cycle
		checkProgress(t, tester.downloader, "peer-full", ethereum.SyncProgress{
			StartingBlock: startingBlock,
			CurrentBlock:  uint64(len(chain.blocks) - 1),
			HighestBlock:  uint64(len(chain.blocks) - 1),
		})
	case <-time.NewTimer(time.Second * 3).C:
		t.Fatalf("Failed to sync chain in three seconds")
	}
}

func TestInvalidBodyPeerDrop(t *testing.T) {
	tester := newTester(t, FullSync)
	defer tester.terminate()

	chain := testChainBase.shorten(blockCacheMaxItems - 15)
	peer := tester.newPeer("corrupt", eth.ETH69, chain.blocks[1:])
	peer.corruptBodies = true

	if err := tester.downloader.BeaconSync(chain.blocks[len(chain.blocks)-1].Header(), nil); err != nil {
		t.Fatalf("failed to beacon-sync chain: %v", err)
	}
	select {
	case <-peer.dropped:
	case <-time.After(1 * time.Minute):
		t.Fatal("peer was not dropped")
	}
}
