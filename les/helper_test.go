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

// This file contains some shares testing functionality, common to  multiple
// different files and modules being tested.

package les

import (
	"context"
	"crypto/rand"
	"math/big"
	"sync"
	"testing"
	"time"

	"github.com/maticnetwork/bor/accounts/abi/bind"
	"github.com/maticnetwork/bor/accounts/abi/bind/backends"
	"github.com/maticnetwork/bor/common"
	"github.com/maticnetwork/bor/common/mclock"
	"github.com/maticnetwork/bor/consensus/ethash"
	"github.com/maticnetwork/bor/contracts/checkpointoracle/contract"
	"github.com/maticnetwork/bor/core"
	"github.com/maticnetwork/bor/core/rawdb"
	"github.com/maticnetwork/bor/core/types"
	"github.com/maticnetwork/bor/crypto"
	"github.com/maticnetwork/bor/eth"
	"github.com/maticnetwork/bor/ethdb"
	"github.com/maticnetwork/bor/event"
	"github.com/maticnetwork/bor/les/flowcontrol"
	"github.com/maticnetwork/bor/light"
	"github.com/maticnetwork/bor/p2p"
	"github.com/maticnetwork/bor/p2p/enode"
	"github.com/maticnetwork/bor/params"
)

var (
	bankKey, _ = crypto.GenerateKey()
	bankAddr   = crypto.PubkeyToAddress(bankKey.PublicKey)
	bankFunds  = big.NewInt(1000000000000000000)

	userKey1, _ = crypto.GenerateKey()
	userKey2, _ = crypto.GenerateKey()
	userAddr1   = crypto.PubkeyToAddress(userKey1.PublicKey)
	userAddr2   = crypto.PubkeyToAddress(userKey2.PublicKey)

	testContractCode         = common.Hex2Bytes("606060405260cc8060106000396000f360606040526000357c01000000000000000000000000000000000000000000000000000000009004806360cd2685146041578063c16431b914606b57603f565b005b6055600480803590602001909190505060a9565b6040518082815260200191505060405180910390f35b60886004808035906020019091908035906020019091905050608a565b005b80600060005083606481101560025790900160005b50819055505b5050565b6000600060005082606481101560025790900160005b5054905060c7565b91905056")
	testContractAddr         common.Address
	testContractCodeDeployed = testContractCode[16:]
	testContractDeployed     = uint64(2)

	testEventEmitterCode = common.Hex2Bytes("60606040523415600e57600080fd5b7f57050ab73f6b9ebdd9f76b8d4997793f48cf956e965ee070551b9ca0bb71584e60405160405180910390a160358060476000396000f3006060604052600080fd00a165627a7a723058203f727efcad8b5811f8cb1fc2620ce5e8c63570d697aef968172de296ea3994140029")

	// Checkpoint registrar relative
	registrarAddr common.Address
	signerKey, _  = crypto.GenerateKey()
	signerAddr    = crypto.PubkeyToAddress(signerKey.PublicKey)
)

var (
	// The block frequency for creating checkpoint(only used in test)
	sectionSize = big.NewInt(512)

	// The number of confirmations needed to generate a checkpoint(only used in test).
	processConfirms = big.NewInt(4)

	//
	testBufLimit    = uint64(1000000)
	testBufRecharge = uint64(1000)
)

/*
contract test {

    uint256[100] data;

    function Put(uint256 addr, uint256 value) {
        data[addr] = value;
    }

    function Get(uint256 addr) constant returns (uint256 value) {
        return data[addr];
    }
}
*/

// prepareTestchain pre-commits specified number customized blocks into chain.
func prepareTestchain(n int, backend *backends.SimulatedBackend) {
	var (
		ctx    = context.Background()
		signer = types.HomesteadSigner{}
	)
	for i := 0; i < n; i++ {
		switch i {
		case 0:
			// deploy checkpoint contract
			registrarAddr, _, _, _ = contract.DeployCheckpointOracle(bind.NewKeyedTransactor(bankKey), backend, []common.Address{signerAddr}, sectionSize, processConfirms, big.NewInt(1))
			// bankUser transfers some ether to user1
			nonce, _ := backend.PendingNonceAt(ctx, bankAddr)
			tx, _ := types.SignTx(types.NewTransaction(nonce, userAddr1, big.NewInt(10000), params.TxGas, nil, nil), signer, bankKey)
			backend.SendTransaction(ctx, tx)
		case 1:
			bankNonce, _ := backend.PendingNonceAt(ctx, bankAddr)
			userNonce1, _ := backend.PendingNonceAt(ctx, userAddr1)

			// bankUser transfers more ether to user1
			tx1, _ := types.SignTx(types.NewTransaction(bankNonce, userAddr1, big.NewInt(1000), params.TxGas, nil, nil), signer, bankKey)
			backend.SendTransaction(ctx, tx1)

			// user1 relays ether to user2
			tx2, _ := types.SignTx(types.NewTransaction(userNonce1, userAddr2, big.NewInt(1000), params.TxGas, nil, nil), signer, userKey1)
			backend.SendTransaction(ctx, tx2)

			// user1 deploys a test contract
			tx3, _ := types.SignTx(types.NewContractCreation(userNonce1+1, big.NewInt(0), 200000, big.NewInt(0), testContractCode), signer, userKey1)
			backend.SendTransaction(ctx, tx3)
			testContractAddr = crypto.CreateAddress(userAddr1, userNonce1+1)

			// user1 deploys a event contract
			tx4, _ := types.SignTx(types.NewContractCreation(userNonce1+2, big.NewInt(0), 200000, big.NewInt(0), testEventEmitterCode), signer, userKey1)
			backend.SendTransaction(ctx, tx4)
		case 2:
			// bankUser transfer some ether to signer
			bankNonce, _ := backend.PendingNonceAt(ctx, bankAddr)
			tx1, _ := types.SignTx(types.NewTransaction(bankNonce, signerAddr, big.NewInt(1000000000), params.TxGas, nil, nil), signer, bankKey)
			backend.SendTransaction(ctx, tx1)

			// invoke test contract
			data := common.Hex2Bytes("C16431B900000000000000000000000000000000000000000000000000000000000000010000000000000000000000000000000000000000000000000000000000000001")
			tx2, _ := types.SignTx(types.NewTransaction(bankNonce+1, testContractAddr, big.NewInt(0), 100000, nil, data), signer, bankKey)
			backend.SendTransaction(ctx, tx2)
		case 3:
			// invoke test contract
			bankNonce, _ := backend.PendingNonceAt(ctx, bankAddr)
			data := common.Hex2Bytes("C16431B900000000000000000000000000000000000000000000000000000000000000020000000000000000000000000000000000000000000000000000000000000002")
			tx, _ := types.SignTx(types.NewTransaction(bankNonce, testContractAddr, big.NewInt(0), 100000, nil, data), signer, bankKey)
			backend.SendTransaction(ctx, tx)
		}
		backend.Commit()
	}
}

// testIndexers creates a set of indexers with specified params for testing purpose.
func testIndexers(db ethdb.Database, odr light.OdrBackend, config *light.IndexerConfig) []*core.ChainIndexer {
	var indexers [3]*core.ChainIndexer
	indexers[0] = light.NewChtIndexer(db, odr, config.ChtSize, config.ChtConfirms)
	indexers[1] = eth.NewBloomIndexer(db, config.BloomSize, config.BloomConfirms)
	indexers[2] = light.NewBloomTrieIndexer(db, odr, config.BloomSize, config.BloomTrieSize)
	// make bloomTrieIndexer as a child indexer of bloom indexer.
	indexers[1].AddChildIndexer(indexers[2])
	return indexers[:]
}

// newTestProtocolManager creates a new protocol manager for testing purposes,
// with the given number of blocks already known, potential notification
// channels for different events and relative chain indexers array.
func newTestProtocolManager(lightSync bool, blocks int, odr *LesOdr, indexers []*core.ChainIndexer, peers *peerSet, db ethdb.Database, ulcServers []string, ulcFraction int, testCost uint64, clock mclock.Clock) (*ProtocolManager, *backends.SimulatedBackend, error) {
	var (
		evmux  = new(event.TypeMux)
		engine = ethash.NewFaker()
		gspec  = core.Genesis{
			Config: params.AllEthashProtocolChanges,
			Alloc:  core.GenesisAlloc{bankAddr: {Balance: bankFunds}},
		}
		pool   txPool
		chain  BlockChain
		exitCh = make(chan struct{})
	)
	gspec.MustCommit(db)
	if peers == nil {
		peers = newPeerSet()
	}
	// create a simulation backend and pre-commit several customized block to the database.
	simulation := backends.NewSimulatedBackendWithDatabase(db, gspec.Alloc, 100000000)
	prepareTestchain(blocks, simulation)

	// initialize empty chain for light client or pre-committed chain for server.
	if lightSync {
		chain, _ = light.NewLightChain(odr, gspec.Config, engine, nil)
	} else {
		chain = simulation.Blockchain()
		pool = core.NewTxPool(core.DefaultTxPoolConfig, gspec.Config, simulation.Blockchain())
	}

	// Create contract registrar
	indexConfig := light.TestServerIndexerConfig
	if lightSync {
		indexConfig = light.TestClientIndexerConfig
	}
	config := &params.CheckpointOracleConfig{
		Address:   crypto.CreateAddress(bankAddr, 0),
		Signers:   []common.Address{signerAddr},
		Threshold: 1,
	}
	var reg *checkpointOracle
	if indexers != nil {
		getLocal := func(index uint64) params.TrustedCheckpoint {
			chtIndexer := indexers[0]
			sectionHead := chtIndexer.SectionHead(index)
			return params.TrustedCheckpoint{
				SectionIndex: index,
				SectionHead:  sectionHead,
				CHTRoot:      light.GetChtRoot(db, index, sectionHead),
				BloomRoot:    light.GetBloomTrieRoot(db, index, sectionHead),
			}
		}
		reg = newCheckpointOracle(config, getLocal)
	}
	pm, err := NewProtocolManager(gspec.Config, nil, indexConfig, ulcServers, ulcFraction, lightSync, NetworkId, evmux, peers, chain, pool, db, odr, nil, reg, exitCh, new(sync.WaitGroup), func() bool { return true })
	if err != nil {
		return nil, nil, err
	}
	// Registrar initialization could failed if checkpoint contract is not specified.
	if pm.reg != nil {
		pm.reg.start(simulation)
	}
	// Set up les server stuff.
	if !lightSync {
		srv := &LesServer{lesCommons: lesCommons{protocolManager: pm, chainDb: db}}
		pm.server = srv
		pm.servingQueue = newServingQueue(int64(time.Millisecond*10), 1)
		pm.servingQueue.setThreads(4)

		srv.defParams = flowcontrol.ServerParams{
			BufLimit:    testBufLimit,
			MinRecharge: testBufRecharge,
		}
		srv.testCost = testCost
		srv.fcManager = flowcontrol.NewClientManager(nil, clock)
	}
	pm.Start(1000)
	return pm, simulation, nil
}

// newTestProtocolManagerMust creates a new protocol manager for testing purposes,
// with the given number of blocks already known, potential notification channels
// for different events and relative chain indexers array. In case of an error, the
// constructor force-fails the test.
func newTestProtocolManagerMust(t *testing.T, lightSync bool, blocks int, odr *LesOdr, indexers []*core.ChainIndexer, peers *peerSet, db ethdb.Database, ulcServers []string, ulcFraction int) (*ProtocolManager, *backends.SimulatedBackend) {
	pm, backend, err := newTestProtocolManager(lightSync, blocks, odr, indexers, peers, db, ulcServers, ulcFraction, 0, &mclock.System{})
	if err != nil {
		t.Fatalf("Failed to create protocol manager: %v", err)
	}
	return pm, backend
}

// testPeer is a simulated peer to allow testing direct network calls.
type testPeer struct {
	net p2p.MsgReadWriter // Network layer reader/writer to simulate remote messaging
	app *p2p.MsgPipeRW    // Application layer reader/writer to simulate the local side
	*peer
}

// newTestPeer creates a new peer registered at the given protocol manager.
func newTestPeer(t *testing.T, name string, version int, pm *ProtocolManager, shake bool, testCost uint64) (*testPeer, <-chan error) {
	// Create a message pipe to communicate through
	app, net := p2p.MsgPipe()

	// Generate a random id and create the peer
	var id enode.ID
	rand.Read(id[:])

	peer := pm.newPeer(version, NetworkId, p2p.NewPeer(id, name, nil), net)

	// Start the peer on a new thread
	errc := make(chan error, 1)
	go func() {
		select {
		case pm.newPeerCh <- peer:
			errc <- pm.handle(peer)
		case <-pm.quitSync:
			errc <- p2p.DiscQuitting
		}
	}()
	tp := &testPeer{
		app:  app,
		net:  net,
		peer: peer,
	}
	// Execute any implicitly requested handshakes and return
	if shake {
		var (
			genesis = pm.blockchain.Genesis()
			head    = pm.blockchain.CurrentHeader()
			td      = pm.blockchain.GetTd(head.Hash(), head.Number.Uint64())
		)
		tp.handshake(t, td, head.Hash(), head.Number.Uint64(), genesis.Hash(), testCost)
	}
	return tp, errc
}

func newTestPeerPair(name string, version int, pm, pm2 *ProtocolManager) (*peer, <-chan error, *peer, <-chan error) {
	// Create a message pipe to communicate through
	app, net := p2p.MsgPipe()

	// Generate a random id and create the peer
	var id enode.ID
	rand.Read(id[:])

	peer := pm.newPeer(version, NetworkId, p2p.NewPeer(id, name, nil), net)
	peer2 := pm2.newPeer(version, NetworkId, p2p.NewPeer(id, name, nil), app)

	// Start the peer on a new thread
	errc := make(chan error, 1)
	errc2 := make(chan error, 1)
	go func() {
		select {
		case pm.newPeerCh <- peer:
			errc <- pm.handle(peer)
		case <-pm.quitSync:
			errc <- p2p.DiscQuitting
		}
	}()
	go func() {
		select {
		case pm2.newPeerCh <- peer2:
			errc2 <- pm2.handle(peer2)
		case <-pm2.quitSync:
			errc2 <- p2p.DiscQuitting
		}
	}()
	return peer, errc, peer2, errc2
}

// handshake simulates a trivial handshake that expects the same state from the
// remote side as we are simulating locally.
func (p *testPeer) handshake(t *testing.T, td *big.Int, head common.Hash, headNum uint64, genesis common.Hash, testCost uint64) {
	var expList keyValueList
	expList = expList.add("protocolVersion", uint64(p.version))
	expList = expList.add("networkId", uint64(NetworkId))
	expList = expList.add("headTd", td)
	expList = expList.add("headHash", head)
	expList = expList.add("headNum", headNum)
	expList = expList.add("genesisHash", genesis)
	sendList := make(keyValueList, len(expList))
	copy(sendList, expList)
	expList = expList.add("serveHeaders", nil)
	expList = expList.add("serveChainSince", uint64(0))
	expList = expList.add("serveStateSince", uint64(0))
	expList = expList.add("serveRecentState", uint64(core.TriesInMemory-4))
	expList = expList.add("txRelay", nil)
	expList = expList.add("flowControl/BL", testBufLimit)
	expList = expList.add("flowControl/MRR", testBufRecharge)
	expList = expList.add("flowControl/MRC", testCostList(testCost))

	if err := p2p.ExpectMsg(p.app, StatusMsg, expList); err != nil {
		t.Fatalf("status recv: %v", err)
	}
	if err := p2p.Send(p.app, StatusMsg, sendList); err != nil {
		t.Fatalf("status send: %v", err)
	}

	p.fcParams = flowcontrol.ServerParams{
		BufLimit:    testBufLimit,
		MinRecharge: testBufRecharge,
	}
}

// close terminates the local side of the peer, notifying the remote protocol
// manager of termination.
func (p *testPeer) close() {
	p.app.Close()
}

// TestEntity represents a network entity for testing with necessary auxiliary fields.
type TestEntity struct {
	db      ethdb.Database
	rPeer   *peer
	tPeer   *testPeer
	peers   *peerSet
	pm      *ProtocolManager
	backend *backends.SimulatedBackend

	// Indexers
	chtIndexer       *core.ChainIndexer
	bloomIndexer     *core.ChainIndexer
	bloomTrieIndexer *core.ChainIndexer
}

// newServerEnv creates a server testing environment with a connected test peer for testing purpose.
func newServerEnv(t *testing.T, blocks int, protocol int, waitIndexers func(*core.ChainIndexer, *core.ChainIndexer, *core.ChainIndexer)) (*TestEntity, func()) {
	db := rawdb.NewMemoryDatabase()
	indexers := testIndexers(db, nil, light.TestServerIndexerConfig)

	pm, b := newTestProtocolManagerMust(t, false, blocks, nil, indexers, nil, db, nil, 0)
	peer, _ := newTestPeer(t, "peer", protocol, pm, true, 0)

	cIndexer, bIndexer, btIndexer := indexers[0], indexers[1], indexers[2]
	cIndexer.Start(pm.blockchain.(*core.BlockChain))
	bIndexer.Start(pm.blockchain.(*core.BlockChain))

	// Wait until indexers generate enough index data.
	if waitIndexers != nil {
		waitIndexers(cIndexer, bIndexer, btIndexer)
	}

	return &TestEntity{
			db:               db,
			tPeer:            peer,
			pm:               pm,
			backend:          b,
			chtIndexer:       cIndexer,
			bloomIndexer:     bIndexer,
			bloomTrieIndexer: btIndexer,
		}, func() {
			peer.close()
			// Note bloom trie indexer will be closed by it parent recursively.
			cIndexer.Close()
			bIndexer.Close()
		}
}

// newClientServerEnv creates a client/server arch environment with a connected les server and light client pair
// for testing purpose.
func newClientServerEnv(t *testing.T, blocks int, protocol int, waitIndexers func(*core.ChainIndexer, *core.ChainIndexer, *core.ChainIndexer), newPeer bool) (*TestEntity, *TestEntity, func()) {
	db, ldb := rawdb.NewMemoryDatabase(), rawdb.NewMemoryDatabase()
	peers, lPeers := newPeerSet(), newPeerSet()

	dist := newRequestDistributor(lPeers, make(chan struct{}), &mclock.System{})
	rm := newRetrieveManager(lPeers, dist, nil)
	odr := NewLesOdr(ldb, light.TestClientIndexerConfig, rm)

	indexers := testIndexers(db, nil, light.TestServerIndexerConfig)
	lIndexers := testIndexers(ldb, odr, light.TestClientIndexerConfig)

	cIndexer, bIndexer, btIndexer := indexers[0], indexers[1], indexers[2]
	lcIndexer, lbIndexer, lbtIndexer := lIndexers[0], lIndexers[1], lIndexers[2]

	odr.SetIndexers(lcIndexer, lbtIndexer, lbIndexer)

	pm, b := newTestProtocolManagerMust(t, false, blocks, nil, indexers, peers, db, nil, 0)
	lpm, lb := newTestProtocolManagerMust(t, true, 0, odr, lIndexers, lPeers, ldb, nil, 0)

	startIndexers := func(clientMode bool, pm *ProtocolManager) {
		if clientMode {
			lcIndexer.Start(pm.blockchain.(*light.LightChain))
			lbIndexer.Start(pm.blockchain.(*light.LightChain))
		} else {
			cIndexer.Start(pm.blockchain.(*core.BlockChain))
			bIndexer.Start(pm.blockchain.(*core.BlockChain))
		}
	}

	startIndexers(false, pm)
	startIndexers(true, lpm)

	// Execute wait until function if it is specified.
	if waitIndexers != nil {
		waitIndexers(cIndexer, bIndexer, btIndexer)
	}

	var (
		peer, lPeer *peer
		err1, err2  <-chan error
	)
	if newPeer {
		peer, err1, lPeer, err2 = newTestPeerPair("peer", protocol, pm, lpm)
		select {
		case <-time.After(time.Millisecond * 100):
		case err := <-err1:
			t.Fatalf("peer 1 handshake error: %v", err)
		case err := <-err2:
			t.Fatalf("peer 2 handshake error: %v", err)
		}
	}

	return &TestEntity{
			db:               db,
			pm:               pm,
			rPeer:            peer,
			peers:            peers,
			backend:          b,
			chtIndexer:       cIndexer,
			bloomIndexer:     bIndexer,
			bloomTrieIndexer: btIndexer,
		}, &TestEntity{
			db:               ldb,
			pm:               lpm,
			rPeer:            lPeer,
			peers:            lPeers,
			backend:          lb,
			chtIndexer:       lcIndexer,
			bloomIndexer:     lbIndexer,
			bloomTrieIndexer: lbtIndexer,
		}, func() {
			// Note bloom trie indexers will be closed by their parents recursively.
			cIndexer.Close()
			bIndexer.Close()
			lcIndexer.Close()
			lbIndexer.Close()
		}
}
