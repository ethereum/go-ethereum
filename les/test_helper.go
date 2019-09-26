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
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/accounts/abi/bind/backends"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/mclock"
	"github.com/ethereum/go-ethereum/consensus/ethash"
	"github.com/ethereum/go-ethereum/contracts/checkpointoracle/contract"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/eth"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/event"
	"github.com/ethereum/go-ethereum/les/flowcontrol"
	"github.com/ethereum/go-ethereum/light"
	"github.com/ethereum/go-ethereum/p2p"
	"github.com/ethereum/go-ethereum/p2p/enode"
	"github.com/ethereum/go-ethereum/params"
)

var (
	bankKey, _ = crypto.GenerateKey()
	bankAddr   = crypto.PubkeyToAddress(bankKey.PublicKey)
	bankFunds  = big.NewInt(1000000000000000000)

	userKey1, _ = crypto.GenerateKey()
	userKey2, _ = crypto.GenerateKey()
	userAddr1   = crypto.PubkeyToAddress(userKey1.PublicKey)
	userAddr2   = crypto.PubkeyToAddress(userKey2.PublicKey)

	testContractAddr         common.Address
	testContractCode         = common.Hex2Bytes("606060405260cc8060106000396000f360606040526000357c01000000000000000000000000000000000000000000000000000000009004806360cd2685146041578063c16431b914606b57603f565b005b6055600480803590602001909190505060a9565b6040518082815260200191505060405180910390f35b60886004808035906020019091908035906020019091905050608a565b005b80600060005083606481101560025790900160005b50819055505b5050565b6000600060005082606481101560025790900160005b5054905060c7565b91905056")
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

	// The token bucket buffer limit for testing purpose.
	testBufLimit = uint64(1000000)

	// The buffer recharging speed for testing purpose.
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

// prepare pre-commits specified number customized blocks into chain.
func prepare(n int, backend *backends.SimulatedBackend) {
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

func newTestClientHandler(backend *backends.SimulatedBackend, odr *LesOdr, indexers []*core.ChainIndexer, db ethdb.Database, peers *peerSet, ulcServers []string, ulcFraction int) *clientHandler {
	var (
		evmux  = new(event.TypeMux)
		engine = ethash.NewFaker()
		gspec  = core.Genesis{
			Config:   params.AllEthashProtocolChanges,
			Alloc:    core.GenesisAlloc{bankAddr: {Balance: bankFunds}},
			GasLimit: 100000000,
		}
		oracle *checkpointOracle
	)
	genesis := gspec.MustCommit(db)
	chain, _ := light.NewLightChain(odr, gspec.Config, engine, nil)
	if indexers != nil {
		checkpointConfig := &params.CheckpointOracleConfig{
			Address:   crypto.CreateAddress(bankAddr, 0),
			Signers:   []common.Address{signerAddr},
			Threshold: 1,
		}
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
		oracle = newCheckpointOracle(checkpointConfig, getLocal)
	}
	client := &LightEthereum{
		lesCommons: lesCommons{
			genesis:     genesis.Hash(),
			config:      &eth.Config{LightPeers: 100, NetworkId: NetworkId},
			chainConfig: params.AllEthashProtocolChanges,
			iConfig:     light.TestClientIndexerConfig,
			chainDb:     db,
			oracle:      oracle,
			chainReader: chain,
			peers:       peers,
			closeCh:     make(chan struct{}),
		},
		reqDist:    odr.retriever.dist,
		retriever:  odr.retriever,
		odr:        odr,
		engine:     engine,
		blockchain: chain,
		eventMux:   evmux,
	}
	client.handler = newClientHandler(ulcServers, ulcFraction, nil, client)

	if client.oracle != nil {
		client.oracle.start(backend)
	}
	return client.handler
}

func newTestServerHandler(blocks int, indexers []*core.ChainIndexer, db ethdb.Database, peers *peerSet, clock mclock.Clock) (*serverHandler, *backends.SimulatedBackend) {
	var (
		gspec = core.Genesis{
			Config:   params.AllEthashProtocolChanges,
			Alloc:    core.GenesisAlloc{bankAddr: {Balance: bankFunds}},
			GasLimit: 100000000,
		}
		oracle *checkpointOracle
	)
	genesis := gspec.MustCommit(db)

	// create a simulation backend and pre-commit several customized block to the database.
	simulation := backends.NewSimulatedBackendWithDatabase(db, gspec.Alloc, 100000000)
	prepare(blocks, simulation)

	txpoolConfig := core.DefaultTxPoolConfig
	txpoolConfig.Journal = ""
	txpool := core.NewTxPool(txpoolConfig, gspec.Config, simulation.Blockchain())
	if indexers != nil {
		checkpointConfig := &params.CheckpointOracleConfig{
			Address:   crypto.CreateAddress(bankAddr, 0),
			Signers:   []common.Address{signerAddr},
			Threshold: 1,
		}
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
		oracle = newCheckpointOracle(checkpointConfig, getLocal)
	}
	server := &LesServer{
		lesCommons: lesCommons{
			genesis:     genesis.Hash(),
			config:      &eth.Config{LightPeers: 100, NetworkId: NetworkId},
			chainConfig: params.AllEthashProtocolChanges,
			iConfig:     light.TestServerIndexerConfig,
			chainDb:     db,
			chainReader: simulation.Blockchain(),
			oracle:      oracle,
			peers:       peers,
			closeCh:     make(chan struct{}),
		},
		servingQueue: newServingQueue(int64(time.Millisecond*10), 1),
		defParams: flowcontrol.ServerParams{
			BufLimit:    testBufLimit,
			MinRecharge: testBufRecharge,
		},
		fcManager: flowcontrol.NewClientManager(nil, clock),
	}
	server.costTracker, server.freeCapacity = newCostTracker(db, server.config)
	server.costTracker.testCostList = testCostList(0) // Disable flow control mechanism.
	server.clientPool = newClientPool(db, 1, 10000, clock, nil)
	server.clientPool.setLimits(10000, 10000) // Assign enough capacity for clientpool
	server.handler = newServerHandler(server, simulation.Blockchain(), db, txpool, func() bool { return true })
	if server.oracle != nil {
		server.oracle.start(simulation)
	}
	server.servingQueue.setThreads(4)
	server.handler.start()
	return server.handler, simulation
}

// testPeer is a simulated peer to allow testing direct network calls.
type testPeer struct {
	peer *peer

	net p2p.MsgReadWriter // Network layer reader/writer to simulate remote messaging
	app *p2p.MsgPipeRW    // Application layer reader/writer to simulate the local side
}

// newTestPeer creates a new peer registered at the given protocol manager.
func newTestPeer(t *testing.T, name string, version int, handler *serverHandler, shake bool, testCost uint64) (*testPeer, <-chan error) {
	// Create a message pipe to communicate through
	app, net := p2p.MsgPipe()

	// Generate a random id and create the peer
	var id enode.ID
	rand.Read(id[:])
	peer := newPeer(version, NetworkId, false, p2p.NewPeer(id, name, nil), net)

	// Start the peer on a new thread
	errCh := make(chan error, 1)
	go func() {
		select {
		case <-handler.closeCh:
			errCh <- p2p.DiscQuitting
		case errCh <- handler.handle(peer):
		}
	}()
	tp := &testPeer{
		app:  app,
		net:  net,
		peer: peer,
	}
	// Execute any implicitly requested handshakes and return
	if shake {
		// Customize the cost table if required.
		if testCost != 0 {
			handler.server.costTracker.testCostList = testCostList(testCost)
		}
		var (
			genesis = handler.blockchain.Genesis()
			head    = handler.blockchain.CurrentHeader()
			td      = handler.blockchain.GetTd(head.Hash(), head.Number.Uint64())
		)
		tp.handshake(t, td, head.Hash(), head.Number.Uint64(), genesis.Hash(), testCostList(testCost))
	}
	return tp, errCh
}

// close terminates the local side of the peer, notifying the remote protocol
// manager of termination.
func (p *testPeer) close() {
	p.app.Close()
}

func newTestPeerPair(name string, version int, server *serverHandler, client *clientHandler) (*testPeer, <-chan error, *testPeer, <-chan error) {
	// Create a message pipe to communicate through
	app, net := p2p.MsgPipe()

	// Generate a random id and create the peer
	var id enode.ID
	rand.Read(id[:])

	peer1 := newPeer(version, NetworkId, false, p2p.NewPeer(id, name, nil), net)
	peer2 := newPeer(version, NetworkId, false, p2p.NewPeer(id, name, nil), app)

	// Start the peer on a new thread
	errc1 := make(chan error, 1)
	errc2 := make(chan error, 1)
	go func() {
		select {
		case <-server.closeCh:
			errc1 <- p2p.DiscQuitting
		case errc1 <- server.handle(peer1):
		}
	}()
	go func() {
		select {
		case <-client.closeCh:
			errc1 <- p2p.DiscQuitting
		case errc1 <- client.handle(peer2):
		}
	}()
	return &testPeer{peer: peer1, net: net, app: app}, errc1, &testPeer{peer: peer2, net: app, app: net}, errc2
}

// handshake simulates a trivial handshake that expects the same state from the
// remote side as we are simulating locally.
func (p *testPeer) handshake(t *testing.T, td *big.Int, head common.Hash, headNum uint64, genesis common.Hash, costList RequestCostList) {
	var expList keyValueList
	expList = expList.add("protocolVersion", uint64(p.peer.version))
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
	expList = expList.add("flowControl/MRC", costList)

	if err := p2p.ExpectMsg(p.app, StatusMsg, expList); err != nil {
		t.Fatalf("status recv: %v", err)
	}
	if err := p2p.Send(p.app, StatusMsg, sendList); err != nil {
		t.Fatalf("status send: %v", err)
	}
	p.peer.fcParams = flowcontrol.ServerParams{
		BufLimit:    testBufLimit,
		MinRecharge: testBufRecharge,
	}
}

type indexerCallback func(*core.ChainIndexer, *core.ChainIndexer, *core.ChainIndexer)

// testClient represents a client for testing with necessary auxiliary fields.
type testClient struct {
	clock   mclock.Clock
	db      ethdb.Database
	peer    *testPeer
	handler *clientHandler

	chtIndexer       *core.ChainIndexer
	bloomIndexer     *core.ChainIndexer
	bloomTrieIndexer *core.ChainIndexer
}

// testServer represents a server for testing with necessary auxiliary fields.
type testServer struct {
	clock   mclock.Clock
	backend *backends.SimulatedBackend
	db      ethdb.Database
	peer    *testPeer
	handler *serverHandler

	chtIndexer       *core.ChainIndexer
	bloomIndexer     *core.ChainIndexer
	bloomTrieIndexer *core.ChainIndexer
}

func newServerEnv(t *testing.T, blocks int, protocol int, callback indexerCallback, simClock bool, newPeer bool, testCost uint64) (*testServer, func()) {
	db := rawdb.NewMemoryDatabase()
	indexers := testIndexers(db, nil, light.TestServerIndexerConfig)

	var clock mclock.Clock = &mclock.System{}
	if simClock {
		clock = &mclock.Simulated{}
	}
	handler, b := newTestServerHandler(blocks, indexers, db, newPeerSet(), clock)

	var peer *testPeer
	if newPeer {
		peer, _ = newTestPeer(t, "peer", protocol, handler, true, testCost)
	}

	cIndexer, bIndexer, btIndexer := indexers[0], indexers[1], indexers[2]
	cIndexer.Start(handler.blockchain)
	bIndexer.Start(handler.blockchain)

	// Wait until indexers generate enough index data.
	if callback != nil {
		callback(cIndexer, bIndexer, btIndexer)
	}
	server := &testServer{
		clock:            clock,
		backend:          b,
		db:               db,
		peer:             peer,
		handler:          handler,
		chtIndexer:       cIndexer,
		bloomIndexer:     bIndexer,
		bloomTrieIndexer: btIndexer,
	}
	teardown := func() {
		if newPeer {
			peer.close()
			b.Close()
		}
		cIndexer.Close()
		bIndexer.Close()
	}
	return server, teardown
}

func newClientServerEnv(t *testing.T, blocks int, protocol int, callback indexerCallback, ulcServers []string, ulcFraction int, simClock bool, connect bool) (*testServer, *testClient, func()) {
	sdb, cdb := rawdb.NewMemoryDatabase(), rawdb.NewMemoryDatabase()
	speers, cPeers := newPeerSet(), newPeerSet()

	var clock mclock.Clock = &mclock.System{}
	if simClock {
		clock = &mclock.Simulated{}
	}
	dist := newRequestDistributor(cPeers, clock)
	rm := newRetrieveManager(cPeers, dist, nil)
	odr := NewLesOdr(cdb, light.TestClientIndexerConfig, rm)

	sindexers := testIndexers(sdb, nil, light.TestServerIndexerConfig)
	cIndexers := testIndexers(cdb, odr, light.TestClientIndexerConfig)

	scIndexer, sbIndexer, sbtIndexer := sindexers[0], sindexers[1], sindexers[2]
	ccIndexer, cbIndexer, cbtIndexer := cIndexers[0], cIndexers[1], cIndexers[2]
	odr.SetIndexers(ccIndexer, cbIndexer, cbtIndexer)

	server, b := newTestServerHandler(blocks, sindexers, sdb, speers, clock)
	client := newTestClientHandler(b, odr, cIndexers, cdb, cPeers, ulcServers, ulcFraction)

	scIndexer.Start(server.blockchain)
	sbIndexer.Start(server.blockchain)
	ccIndexer.Start(client.backend.blockchain)
	cbIndexer.Start(client.backend.blockchain)

	if callback != nil {
		callback(scIndexer, sbIndexer, sbtIndexer)
	}
	var (
		speer, cpeer *testPeer
		err1, err2   <-chan error
	)
	if connect {
		cpeer, err1, speer, err2 = newTestPeerPair("peer", protocol, server, client)
		select {
		case <-time.After(time.Millisecond * 100):
		case err := <-err1:
			t.Fatalf("peer 1 handshake error: %v", err)
		case err := <-err2:
			t.Fatalf("peer 2 handshake error: %v", err)
		}
	}
	s := &testServer{
		clock:            clock,
		backend:          b,
		db:               sdb,
		peer:             cpeer,
		handler:          server,
		chtIndexer:       scIndexer,
		bloomIndexer:     sbIndexer,
		bloomTrieIndexer: sbtIndexer,
	}
	c := &testClient{
		clock:            clock,
		db:               cdb,
		peer:             speer,
		handler:          client,
		chtIndexer:       ccIndexer,
		bloomIndexer:     cbIndexer,
		bloomTrieIndexer: cbtIndexer,
	}
	teardown := func() {
		if connect {
			speer.close()
			cpeer.close()
		}
		ccIndexer.Close()
		cbIndexer.Close()
		scIndexer.Close()
		sbIndexer.Close()
		b.Close()
	}
	return s, c, teardown
}
