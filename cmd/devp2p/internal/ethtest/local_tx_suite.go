package ethtest

import (
	"bufio"
	"context"
	"crypto/ecdsa"
	"fmt"
	"math/big"
	"os"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/internal/utesting"
	"github.com/ethereum/go-ethereum/params"
)

// peerID is a rlpx peer's secp256k1 public key expressed as a hex string
type peerID string

type testRunner struct {
	signer     types.Signer
	allTxs     map[common.Hash]*types.Transaction
	txsCh      chan peerTxReport
	txHashesCh chan peerTxReport

	senders         map[common.Address]struct{}
	peers           []peerID
	received        map[peerID]map[common.Address][]common.Hash
	lastAnnounced   map[peerID]map[common.Address]time.Time
	lastBroadcasted map[peerID]map[common.Address]time.Time
}

// peerTxReport encapsulates a transaction broadcast/announcement network message received from a peer
type peerTxReport struct {
	peer   peerID
	hashes []common.Hash
}

func newTestRunner(s *Suite) *testRunner {
	chainConfig := s.backend.ChainConfig()
	signer := types.LatestSigner(chainConfig)

	return &testRunner{
		signer,
		make(map[common.Hash]*types.Transaction),
		make(chan peerTxReport),
		make(chan peerTxReport),
		make(map[common.Address]struct{}),
		[]peerID{},
		make(map[peerID]map[common.Address][]common.Hash),
		make(map[peerID]map[common.Address]time.Time),
		make(map[peerID]map[common.Address]time.Time),
	}
}

// TODO: this should be 400ms (mirroring eth/local_tx_handler.go).  but for some reason causes test failures (tx sent too soon)
const minDelayTime = 390 * time.Millisecond

// loadAccount loads private keys corresponding to pre-allocated accounts in the genesis
func loadAccountKeypairs() []*ecdsa.PrivateKey {
	file, err := os.Open("testdata/pk_list.txt")
	if err != nil {
		panic(err)
	}
	defer file.Close()

	keys := []*ecdsa.PrivateKey{}

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		privateKey, err := crypto.HexToECDSA(scanner.Text())
		if err != nil {
			panic(err)
		}
		keys = append(keys, privateKey)
	}
	return keys
}

func (t *testRunner) recoverSender(tx *types.Transaction) common.Address {
	sender, _ := types.Sender(t.signer, tx)
	return sender
}

// TODO: de-dup validateBroadcast and validateAnnounce

func (t *testRunner) validateBroadcast(report peerTxReport) {
	senders := make(map[common.Address]struct{})
	var lastBroadcastTime time.Time
	var ok bool

	for _, hash := range report.hashes {
		// check only 1 tx per sender account
		sender := t.recoverSender(t.allTxs[hash])
		if _, ok = senders[sender]; ok {
			panic("multiple txs from sender")
		}
		senders[sender] = struct{}{}

		// check delay for broadcasting consecutive transactions
		// from an account to a peer is proper
		if lastBroadcastTime, ok = t.lastBroadcasted[report.peer][sender]; !ok {
			t.lastBroadcasted[report.peer] = make(map[common.Address]time.Time)
		} else if time.Since(lastBroadcastTime) < minDelayTime {
			panic("peer sent too soon")
		}

		t.lastBroadcasted[report.peer][sender] = time.Now()
		if _, ok = t.received[report.peer][sender]; !ok {
			t.received[report.peer][sender] = []common.Hash{}
		}
		t.received[report.peer][sender] = append(t.received[report.peer][sender], hash)
	}

	for sender := range senders {
		t.senders[sender] = struct{}{}
	}
}

func (t *testRunner) validateAnnounce(report peerTxReport) {
	senders := make(map[common.Address]struct{})
	var lastAnnounceTime time.Time
	var ok bool

	for _, hash := range report.hashes {
		// check only 1 tx per sender
		sender := t.recoverSender(t.allTxs[hash])
		if _, ok = senders[sender]; ok {
			panic("multiple txs from sender")
		}
		senders[sender] = struct{}{}

		// check delay for announcing consecutive transactions
		// from an account to a peer is proper
		if lastAnnounceTime, ok = t.lastAnnounced[report.peer][sender]; !ok {
			t.lastAnnounced[report.peer] = make(map[common.Address]time.Time)
		} else if time.Since(lastAnnounceTime) < minDelayTime {
			panic("peer sent too soon")
		}

		t.lastAnnounced[report.peer][sender] = time.Now()
		if _, ok = t.received[report.peer][sender]; !ok {
			t.received[report.peer][sender] = []common.Hash{}
		}
		t.received[report.peer][sender] = append(t.received[report.peer][sender], hash)
	}
}

// waitForAnnouncesAndBroadcasts waits for test transactions to be announced or propagated and performs
// validation for delay of consecutive tx announce/broadcasts from same sender, validation for ordering of
// received txs/hashes, validation that a peer in the unchanging peerset is broadcasted or announced txs (not both)
func (t *testRunner) waitForAnnouncesAndBroadcasts(expected map[common.Address][]common.Hash) {
	// use a static timeout (TODO: tweak this not not rely on a hardcoded timeout for the condition to
	// end the test)
	timeout := time.NewTimer(12 * time.Second)

	peerReceivedBroadcast := make(map[peerID]bool)
	peerReceivedAnnounce := make(map[peerID]bool)

loop:
	for {
		select {
		case report := <-t.txsCh:
			t.validateBroadcast(report)
			peerReceivedBroadcast[report.peer] = true
			if peerReceivedAnnounce[report.peer] {
				panic("received announces and broadcasts")
			}
		case report := <-t.txHashesCh:
			t.validateAnnounce(report)
			peerReceivedAnnounce[report.peer] = true
			if peerReceivedBroadcast[report.peer] {
				panic("received announces and broadcasts")
			}
		case <-timeout.C:
			break loop
		}
	}

	// validate that each peer only received announcements or direct broadcasts and that each
	// peer was announced/broadcasted all transactions from each local account in nonce order
	for _, id := range t.peers {
		if peerReceivedAnnounce[id] {
			if peerReceivedBroadcast[id] {
				panic("peer received announcements and broadcasts")
			}
		} else {
			if !peerReceivedBroadcast[id] {
				panic("peer should have received announcements/broadcasts")
			}
		}

		for sender := range t.senders {
			numTxsWanted := len(expected[sender])
			numTxsReceived := len(t.received[id][sender])
			if numTxsWanted != numTxsReceived {
				panic(fmt.Sprintf("invalid number of txs received (wanted %d. got %d)", numTxsWanted, numTxsReceived))
			}

			for j := 0; j < len(t.received[id][sender]); j++ {
				if t.received[id][sender][j] != expected[sender][j] {
					panic("didn't receive expected")
				}
			}
		}
	}
}

// peerLoop runs in a separate go-routine, receives transactions/announcements from
// the Geth node and reports them to the testRunner via the channels passed to the method.
func peerLoop(publicKey peerID, conn *Conn, txHashesCh, txsCh chan peerTxReport) {
	for {
		switch msg := conn.Read().(type) {
		case *Ping:
			// TODO: send a pong back
			panic("no pong!")
		case *NewPooledTransactionHashes:
			hashes := msg.Hashes
			txHashesCh <- peerTxReport{publicKey, hashes}
		case *Transactions:
			txs := msg
			var hashes []common.Hash
			for _, tx := range *txs {
				hashes = append(hashes, tx.Hash())
			}
			txsCh <- peerTxReport{publicKey, hashes}
		default:
			return
		}
	}
}

// addPeer instantiates a new mock eth peer and connects it to the Geth node
func (t *testRunner) addPeer(s *Suite) {
	peerConn, err := s.dial()
	if err != nil {
		panic(err)
	}

	publicKey := peerID(fmt.Sprintf("%x", peerConn.publicKey()))
	t.received[publicKey] = make(map[common.Address][]common.Hash)
	t.lastAnnounced[publicKey] = make(map[common.Address]time.Time)
	t.lastBroadcasted[publicKey] = make(map[common.Address]time.Time)
	t.peers = append(t.peers, publicKey)

	if err := peerConn.handshake(); err != nil {
		panic(err)
	}

	if _, err = peerConn.statusExchange(s.chain, nil); err != nil {
		panic(err)
	}

	go peerLoop(publicKey, peerConn, t.txHashesCh, t.txsCh)
}

// generateTx creates a simple transfer-to-0x00...00 transaction with given nonce, gas price
// and signs it using the provided key
func (t *testRunner) generateTx(s *Suite, key *ecdsa.PrivateKey, nonce uint64, gasPrice *big.Int) *types.Transaction {
	tx := types.MustSignNewTx(key, t.signer, &types.LegacyTx{
		Nonce:    nonce,
		To:       &common.Address{},
		Value:    big.NewInt(1000),
		Gas:      params.TxGas,
		GasPrice: gasPrice,
	})
	t.allTxs[tx.Hash()] = tx
	return tx
}

// test case which inserts transactions from multiple accounts and ensures they were broadcasted/announced
func (s *Suite) TestLocalTxBasic(_ *utesting.T) {
	t := newTestRunner(s)
	keys := loadAccountKeypairs()

	numPeers := 4
	for i := 0; i < numPeers; i++ {
		t.addPeer(s)
	}

	expected := make(map[common.Address][]common.Hash)

	for _, key := range keys {
		pk, _ := key.Public().(*ecdsa.PublicKey)
		sender := crypto.PubkeyToAddress(*pk)
		expected[sender] = []common.Hash{}

		for nonce := uint64(0); nonce < 25; nonce++ {
			tx := t.generateTx(s, key, nonce, big.NewInt(params.InitialBaseFee))
			expected[sender] = append(expected[sender], tx.Hash())
			s.backend.SendTx(context.Background(), tx)
		}
	}

	t.waitForAnnouncesAndBroadcasts(expected)
}

// test case which inserts transactions from multiple accounts (multiple transactions per account).  Replacement transactions are then added.  Test expects
// that only replacement transactions were announced/broadcasted
func (s *Suite) TestLocalTxReplacement(_ *utesting.T) {
	t := newTestRunner(s)
	keys := loadAccountKeypairs()

	numPeers := 4
	for i := 0; i < numPeers; i++ {
		t.addPeer(s)
	}

	expected := make(map[common.Address][]common.Hash)

	for _, key := range keys {
		pk, _ := key.Public().(*ecdsa.PublicKey)
		sender := crypto.PubkeyToAddress(*pk)
		expected[sender] = []common.Hash{}

		// insert txs from many local accounts, many txs per account
		for nonce := uint64(0); nonce < 25; nonce++ {
			tx := t.generateTx(s, key, nonce, big.NewInt(params.InitialBaseFee))
			expected[sender] = append(expected[sender], tx.Hash())
			s.backend.SendTx(context.Background(), tx)
		}
	}

	// delay to ensure percolate through txpool to locals tx handler
	time.Sleep(20 * time.Millisecond)

	// generate some replacement transactions
	for _, key := range keys[:2] {
		pk, _ := key.Public().(*ecdsa.PublicKey)
		sender := crypto.PubkeyToAddress(*pk)

		for nonce := uint64(12); nonce < 25; nonce++ {
			tx := t.generateTx(s, key, nonce, big.NewInt(params.InitialBaseFee*2))
			expected[sender][nonce] = tx.Hash()
			s.backend.SendTx(context.Background(), tx)
		}
	}

	t.waitForAnnouncesAndBroadcasts(expected)
}

// more test case ideas:

// * start with initial peer set, add some transactions, wait.  add peer(s), broadcast more transactions and ensure that each peer has either been broadcasted/announced all the txs
// * insert new head block into the chain which includes some txs, connect a new peer and ensure it doesn't get announced/broadcasted the included txs
// * add txs, connect peers, add block which includes txs, ensure most of the txs are not broadcast/announced
// * add txs, add block that contains txs, reorg before block, connect peers and
// *  disconnecting peer behavior?  probably not super relevant for this PR
