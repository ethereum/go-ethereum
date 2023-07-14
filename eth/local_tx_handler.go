package eth

// TODO: port relevant parts of debug logging previously in eth/handler.go here

import (
	"math"
	"sort"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/lru"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/event"
)

const (
	// broadcastWaitTime is the minimum time that we will delay between broadcasting transactions
	// from the same local address to a peer.
	broadcastWaitTime = 400 * time.Millisecond
	// announceWaitTime is the minimum time that we will delay between announcing transactions
	// from the same local address to a peer.
	announceWaitTime = 400 * time.Millisecond
)

// peerID is a rlpx peer's secp256k1 public key expressed as a hex string
type peerID string

// localAccountStatus represents the state of local transaction announcement/broadcast for a given peer
// and local sender Ethereum address
type localAccountStatus struct {
	// TODO: maybe change the below (and associated logic) to
	// "highest" to avoid confusion
	// the lowest nonce from the account that has not had a transaction sent to the peer from us
	lowestUnsentNonce uint64
	// the lowest nonce from the account that hasn't had a transaction announced to the peer from us
	lowestUnannouncedNonce uint64
	// the last time we sent a transaction from this account to the associated peer
	lastBroadcastTime time.Time
	// the last time we announced a transaction from this account to the associated peer
	lastAnnounceTime time.Time
}

// peerStateCache implements a 2-level lookup keyed by peerID, sender address and mapping to a
// localAccountStatus
type peerStateCache struct {
	internal lru.BasicLRU[peerID, map[common.Address]localAccountStatus]
}

// getOrNew returns the existing localAccountStatus for a given peer and local sender address
// or creates and returns a new one if it didn't previously exist.
func (p *peerStateCache) getOrNew(peer peerID, addr common.Address) localAccountStatus {
	var peerEntry map[common.Address]localAccountStatus

	peerEntry, ok := p.internal.Get(peer)
	if !ok {
		peerEntry = make(map[common.Address]localAccountStatus)
	}

	accountEntry, ok := peerEntry[addr]
	if !ok {
		accountEntry = localAccountStatus{
			0,
			0,
			// initialize these so that we broadcast/announce to the peer immediately
			// for this account
			time.Now().Add(-broadcastWaitTime),
			time.Now().Add(-announceWaitTime),
		}
		peerEntry[addr] = accountEntry
		p.internal.Add(peer, peerEntry)
	}
	return accountEntry
}

// set sets the localAccountStatus for a given peer and local address
func (p *peerStateCache) set(peer peerID, addr common.Address, l *localAccountStatus) {
	as, _ := p.internal.Get(peer)
	as[addr] = *l
	p.internal.Add(peer, as)
}

// getLowestUnsentNonce returns the lowest unsent nonce for a peer/addr, creating an entry
// if none previously existed and returning 0.
func (p *peerStateCache) getLowestUnsentNonce(peer peerID, addr common.Address) uint64 {
	accountEntry := p.getOrNew(peer, addr)
	return accountEntry.lowestUnsentNonce
}

// getLowestUnannouncedNonce returns the lowest unannounced nonce for a peer/addr, creating an entry
// if none previously existed and returning 0.
func (p *peerStateCache) getLowestUnannouncedNonce(peer peerID, addr common.Address) uint64 {
	accountEntry := p.getOrNew(peer, addr)
	return accountEntry.lowestUnannouncedNonce
}

// setBroadcastTime sets the latest broadcast time for a peer/addr, creating an entry
// if none previously existed.
func (p *peerStateCache) setLastBroadcastTime(peer peerID, addr common.Address, time time.Time) {
	accountEntry := p.getOrNew(peer, addr)
	accountEntry.lastBroadcastTime = time
	p.set(peer, addr, &accountEntry)
}

// setBroadcastTime sets the latest broadcast time for a peer/addr, creating an entry
// if none previously existed.
func (p *peerStateCache) setLastAnnounceTime(peer peerID, addr common.Address, time time.Time) {
	accountEntry := p.getOrNew(peer, addr)
	accountEntry.lastAnnounceTime = time
	p.set(peer, addr, &accountEntry)
}

// setLowestUnsentNonce sets the last unsent nonce for a peer/addr, creating an entry
// if none previously existed.
func (p *peerStateCache) setLowestUnsentNonce(peer peerID, addr common.Address, nonce uint64) {
	accountEntry := p.getOrNew(peer, addr)
	accountEntry.lowestUnsentNonce = nonce
	p.set(peer, addr, &accountEntry)
}

// setLowestUnsentNonce sets the last unsent nonce for a peer/addr, creating an entry
// if none previously existed.
func (p *peerStateCache) setLowestUnannouncedNonce(peer peerID, addr common.Address, nonce uint64) {
	accountEntry := p.getOrNew(peer, addr)
	accountEntry.lowestUnannouncedNonce = nonce
	p.set(peer, addr, &accountEntry)
}

// getBroadcastTime returns the latest broadcast time for a given peer/sender, creating
// an entry if none previously existed and returning nil.
func (p *peerStateCache) getBroadcastTime(peer peerID, addr common.Address) time.Time {
	accountEntry := p.getOrNew(peer, addr)
	return accountEntry.lastBroadcastTime
}

// getAnnounceTime returns the latest announce time for a given peer/sender, creating
// an entry if none previously existed and returning nil.
func (p *peerStateCache) getAnnounceTime(peer peerID, addr common.Address) time.Time {
	accountEntry := p.getOrNew(peer, addr)
	return accountEntry.lastAnnounceTime
}

// localTxHandler implements new logic for broadcast/announcement of local transactions:
// transactions are broadcast nonce-ordered to a square root of the peerset,
// a delay of 1 second is added between sending consecutive transactions from the
// same account.
type localTxHandler struct {
	// map of local sender address to a nonce-ordered array of transactions sent from that account
	txQueues         map[common.Address][]*types.Transaction
	peersStatus      peerStateCache
	chain            *core.BlockChain
	peers            *peerSet
	chainHeadCh      <-chan core.ChainHeadEvent
	chainHeadSub     event.Subscription
	localTxsCh       <-chan core.NewTxsEvent
	localTxsSub      event.Subscription
	broadcastTrigger *time.Ticker
	announceTrigger  *time.Ticker
	signer           types.Signer
}

func newLocalsTxBroadcaster(txpool txPool, chain *core.BlockChain, peers *peerSet) *localTxHandler {
	chainHeadCh := make(chan core.ChainHeadEvent, 10)
	chainHeadSub := chain.SubscribeChainHeadEvent(chainHeadCh)
	localTxsCh := make(chan core.NewTxsEvent, 10)
	localTxsSub := txpool.SubscribeNewLocalTxsEvent(localTxsCh)

	// TODO: this maxPeers is arbitrarily chosen (and generous).
	// choose a proper value based on node configuration
	maxPeers := 64

	l := localTxHandler{
		make(map[common.Address][]*types.Transaction),
		peerStateCache{lru.NewBasicLRU[peerID, map[common.Address]localAccountStatus](maxPeers)},
		chain,
		peers,
		chainHeadCh,
		chainHeadSub,
		localTxsCh,
		localTxsSub,
		// use low default times to exhaust the ticker initially
		time.NewTicker(1 * time.Nanosecond),
		time.NewTicker(1 * time.Nanosecond),
		types.LatestSigner(chain.Config()),
	}

	<-l.broadcastTrigger.C
	<-l.announceTrigger.C
	return &l
}

// get the nonce of an account at the head block
func (l *localTxHandler) GetNonce(addr common.Address) uint64 {
	curState, err := l.chain.State()
	if err != nil {
		// TODO figure out what this could be
		panic(err)
	}
	return curState.GetNonce(addr)
}

// announcedRecently returns whether or not we sent a given peer a local transaction from sender
func (l *localTxHandler) announcedRecently(peer peerID, sender common.Address) bool {
	at := l.peersStatus.getAnnounceTime(peer, sender)
	return time.Since(at) <= announceWaitTime
}

// sentRecently returns whether or not we sent a given peer a local transaction from sender
func (l *localTxHandler) sentRecently(peer peerID, sender common.Address) bool {
	bt := l.peersStatus.getBroadcastTime(peer, sender)
	return time.Since(bt) <= announceWaitTime
}

// nextTxToBroadcast retrieves the next unsent lowest-nonce transaction from an account returns it
// (or nil if there is no unsent transaction from the account).  The internal state
// of localTxHandler is modified to reflect the tx as being sent to the peer.
func (l *localTxHandler) nextTxToBroadcast(peer *ethPeer, sender common.Address) *common.Hash {
	lowestUnsentNonce := l.peersStatus.getLowestUnsentNonce(peerID(peer.ID()), sender)
	txs := l.txQueues[sender]
	for i, tx := range txs {
		if tx.Nonce() >= lowestUnsentNonce {
			lowestUnsentNonce = tx.Nonce() + 1
			l.peersStatus.setLowestUnsentNonce(peerID(peer.ID()), sender, lowestUnsentNonce)

			if !peer.KnownTransaction(tx.Hash()) {
				l.peersStatus.setLastBroadcastTime(peerID(peer.ID()), sender, time.Now())

				if i != len(txs)-1 {
					// there is a higher nonce transaction on the queue
					// after this one.  reset the timer to ensure it will be sent
					l.broadcastTrigger.Reset(broadcastWaitTime)
				}
				res := tx.Hash()
				return &res
			}
		}
	}
	return nil
}

// nextTxToBroadcast retrieves the next unannounced lowest-nonce transaction from an account and returns it
// (or nil if there is no unannounced transaction from the account).
// The internal state of localTxHandler is modified to reflect the tx as being sent to the peer.
func (l *localTxHandler) nextTxToAnnounce(peer *ethPeer, sender common.Address) *common.Hash {
	lowestUnannouncedNonce := l.peersStatus.getLowestUnannouncedNonce(peerID(peer.ID()), sender)
	txs := l.txQueues[sender]
	for i, tx := range txs {
		if tx.Nonce() >= lowestUnannouncedNonce {
			lowestUnannouncedNonce = tx.Nonce() + 1
			l.peersStatus.setLowestUnannouncedNonce(peerID(peer.ID()), sender, lowestUnannouncedNonce)
			l.peersStatus.setLastAnnounceTime(peerID(peer.ID()), sender, time.Now())

			if i != len(txs)-1 {
				// there is a higher nonce transaction on the queue
				// after this one.  reset the timer to ensure it will be announced
				l.announceTrigger.Reset(announceWaitTime)
			}
			res := tx.Hash()
			return &res
		}
	}
	return nil
}

// maybeBroadcast broadcasts the next unsent and unknown transaction to each peer if:
//  1. the peer is in the square-root subset of the peerset
//  2. another transaction from the same sender has not been sent to the peer recently
//  3. the peer does not already have the transaction
func (l *localTxHandler) maybeBroadcast() {
	allPeers := l.peers.allEthPeers()
	allPeers = sortedPeers(allPeers)
	directBroadcastPeers := allPeers[:int(math.Sqrt(float64(len(allPeers))))]
	for _, peer := range directBroadcastPeers {
		var txsToBroadcast []common.Hash

		for addr := range l.txQueues {
			if l.sentRecently(peerID(peer.ID()), addr) {
				continue
			}
			if tx := l.nextTxToBroadcast(peer, addr); tx != nil {
				txsToBroadcast = append(txsToBroadcast, *tx)
			}
		}
		if len(txsToBroadcast) > 0 {
			peer.AsyncSendTransactions(txsToBroadcast)
		}
	}
}

// maybeBroadcast announces the next unknown transaction to each peer if:
//  1. the peer is in the square-root subset of the peerset
//  2. another transaction from the same sender has not been announced to the peer recently
func (l *localTxHandler) maybeAnnounce() {
	allPeers := l.peers.allEthPeers()
	allPeers = sortedPeers(allPeers)
	announcePeers := allPeers[int(math.Sqrt(float64(len(allPeers)))):]
	for _, peer := range announcePeers {
		var txsToAnnounce []common.Hash

		for addr := range l.txQueues {
			if l.announcedRecently(peerID(peer.ID()), addr) {
				continue
			}
			if tx := l.nextTxToAnnounce(peer, addr); tx != nil {
				txsToAnnounce = append(txsToAnnounce, *tx)
			}
		}
		if len(txsToAnnounce) > 0 {
			peer.AsyncSendPooledTransactionHashes(txsToAnnounce)
		}
	}
}

// trimLocals removes transactions from monitoring queues of their senders
// if their nonce is lte the sender account's nonce in the head state of the chain.
func (l *localTxHandler) trimLocals() {
	for addr, txs := range l.txQueues {
		var cutPoint int
		currentNonce := l.GetNonce(addr)
		for i, tx := range txs {
			if tx.Nonce() < currentNonce {
				cutPoint = i
			} else {
				break
			}
		}

		l.txQueues[addr] = l.txQueues[addr][cutPoint:]

		if len(l.txQueues[addr]) == 0 {
			delete(l.txQueues, addr)
		}
	}
}

// sortedPeers sorts a set of peers by the lexographic order of their rlpx secp256k1 public keys.
// the provided array is not modified.
func sortedPeers(unsorted []*ethPeer) []*ethPeer {
	peerMap := make(map[string]*ethPeer)
	peerMapKeys := []string{}

	for _, peer := range unsorted {
		pid := peer.ID()
		peerMap[pid] = peer
		peerMapKeys = append(peerMapKeys, pid)
	}

	sort.Strings(peerMapKeys)

	var res []*ethPeer
	for _, pid := range peerMapKeys {
		res = append(res, peerMap[pid])
	}

	return res
}

// TODO: noting an edge-case/idea/question here.  If we have already broadcasted/announced a set of transactions to a peer and we receive one or more replacement transactions with the same nonce
// should we broadcast/announce them all immediately?  seems like a no-brainer.

// mergeNonceOrderedTxs combines curTxs and newTxs together.  each list is nonce-contiguous, nonce-unique ordered ascending by nonce.  If the nonce of a transaction in newTxs matches a transaction in curTxs, only the transaction from newTxs is included in the result.
// The resulting list is nonce-contiguous, nonce-unique and ordered ascending by nonce.
func mergeNonceOrderedTxs(curTxs []*types.Transaction, newTxs []*types.Transaction) []*types.Transaction {
	// insert new txs into the sender's queue, keeping the resulting array nonce-ordered and
	// replacing pre-existing txs if there is a tx with same nonce
	curTxsIdx, newTxsIdx := 0, 0
	var res []*types.Transaction

	for curTxsIdx < len(curTxs) || newTxsIdx < len(newTxs) {
		if curTxsIdx < len(curTxs) && newTxsIdx < len(newTxs) {
			if curTxs[curTxsIdx].Nonce() < newTxs[newTxsIdx].Nonce() {
				res = append(res, curTxs[curTxsIdx])
				curTxsIdx++
			} else if curTxs[curTxsIdx].Nonce() > newTxs[newTxsIdx].Nonce() {
				res = append(res, newTxs[newTxsIdx])
				newTxsIdx++
			} else {
				res = append(res, newTxs[newTxsIdx])
				curTxsIdx++
				newTxsIdx++
			}
		} else if curTxsIdx < len(curTxs) {
			res = append(res, curTxs[curTxsIdx])
			curTxsIdx++
		} else if newTxsIdx < len(newTxs) {
			res = append(res, newTxs[newTxsIdx])
			newTxsIdx++
		}
	}
	return res
}

// updateLowestUnsentNonce sets the lowest unsent nonce for the account in each peer's tracker if
// the new lowest unsent nonce is lte the current one for the peer/sender.
func (l *localTxHandler) updateLowestUnsentNonce(sender common.Address, nonce uint64) {
	for _, peer := range l.peers.allEthPeers() {
		lastUnsentNonce := l.peersStatus.getLowestUnsentNonce(peerID(peer.ID()), sender)
		if nonce <= lastUnsentNonce {
			l.peersStatus.setLowestUnsentNonce(peerID(peer.ID()), sender, nonce)
		}
	}
}

// updateLowestUnannouncedNonce sets the lowest unannounced nonce for the account in each peer's tracker if
// the new lowest unsent nonce is lte the current one for the peer/sender.
func (l *localTxHandler) updateLowestUnannouncedNonce(sender common.Address, nonce uint64) {
	for _, peer := range l.peers.allEthPeers() {
		lastUnsentNonce := l.peersStatus.getLowestUnannouncedNonce(peerID(peer.ID()), sender)
		if nonce <= lastUnsentNonce {
			l.peersStatus.setLowestUnannouncedNonce(peerID(peer.ID()), sender, nonce)
		}
	}
}

// addLocalsFromSender inserts a list of nonce-ordered transactions into the tracking
// queue for the associated account.
func (l *localTxHandler) addLocalsFromSender(sender common.Address, newTxs []*types.Transaction) {
	var curTxs []*types.Transaction

	curTxs, ok := l.txQueues[sender]
	if !ok {
		l.txQueues[sender] = newTxs
		l.updateLowestUnsentNonce(sender, newTxs[0].Nonce())
		l.updateLowestUnannouncedNonce(sender, newTxs[0].Nonce())
		return
	}

	newTxsProceedCurrent := newTxs[0].Nonce() > l.txQueues[sender][len(l.txQueues[sender])-1].Nonce()
	if newTxsProceedCurrent {
		l.txQueues[sender] = append(curTxs, newTxs...)
	} else {
		txsOverlap := newTxs[len(newTxs)-1].Nonce() >= l.txQueues[sender][0].Nonce()
		if txsOverlap {
			l.txQueues[sender] = mergeNonceOrderedTxs(l.txQueues[sender], newTxs)
		} else {
			l.txQueues[sender] = append(newTxs, curTxs...)
		}
		l.updateLowestUnsentNonce(sender, newTxs[0].Nonce())
		l.updateLowestUnannouncedNonce(sender, newTxs[0].Nonce())
	}
}

// add a set of locals into the tracking queues
// assumes:
//  1. txs is a nonce-ordered, account-grouped list of transactions
//  2. there are no nonce gaps, multiple calls to addLocals pass transactions
//     with nonces that are contiguous/overlapping with values from previous calls.
func (l *localTxHandler) addLocals(txs []*types.Transaction) {
	acctTxs := make(map[common.Address][]*types.Transaction)
	lastSender := common.Address{}
	for _, tx := range txs {
		sender, _ := types.Sender(l.signer, tx)
		if sender != lastSender {
			acctTxs[sender] = types.Transactions{tx}
			lastSender = sender
		} else {
			acctTxs[sender] = append(acctTxs[sender], tx)
		}
	}

	for acct, txs := range acctTxs {
		l.addLocalsFromSender(acct, txs)
	}
}

// Stop stops the long-running go-routine event loop for the localTxHandler
func (l *localTxHandler) Stop() {
	l.chainHeadSub.Unsubscribe()
	l.localTxsSub.Unsubscribe()
}

// Run starts the long-running go-routine event loop for the localTxHandler
func (l *localTxHandler) Run(wg *sync.WaitGroup) {
	defer wg.Done()
	l.loop()
}

// loop is a long-running method which manages the life-cycle for the localTxHandler
func (l *localTxHandler) loop() {
	for {
		select {
		case <-l.chainHeadCh:
			l.trimLocals()
		case evt := <-l.localTxsCh:
			l.addLocals(evt.Txs)
			l.maybeBroadcast()
			l.maybeAnnounce()
		case <-l.broadcastTrigger.C:
			l.maybeBroadcast()
		case <-l.announceTrigger.C:
			l.maybeAnnounce()
		case <-l.localTxsSub.Err():
			return
		}
	}
}
