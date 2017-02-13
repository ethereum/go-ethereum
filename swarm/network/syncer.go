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

package network

import (
	"encoding/binary"
	"encoding/json"
	"fmt"
	"path/filepath"

	"github.com/ethereum/go-ethereum/logger"
	"github.com/ethereum/go-ethereum/logger/glog"
	"github.com/ethereum/go-ethereum/swarm/storage"
)

// syncer parameters (global, not peer specific) default values
const (
	requestDbBatchSize = 512  // size of batch before written to request db
	keyBufferSize      = 1024 // size of buffer  for unsynced keys
	syncBatchSize      = 128  // maximum batchsize for outgoing requests
	syncBufferSize     = 128  // size of buffer  for delivery requests
	syncCacheSize      = 1024 // cache capacity to store request queue in memory
)

// priorities
const (
	Low        = iota // 0
	Medium            // 1
	High              // 2
	priorities        // 3 number of priority levels
)

// request types
const (
	DeliverReq   = iota // 0
	PushReq             // 1
	PropagateReq        // 2
	HistoryReq          // 3
	BacklogReq          // 4
)

// json serialisable struct to record the syncronisation state between 2 peers
type syncState struct {
	*storage.DbSyncState // embeds the following 4 fields:
	// Start      Key    // lower limit of address space
	// Stop       Key    // upper limit of address space
	// First      uint64 // counter taken from last sync state
	// Last       uint64 // counter of remote peer dbStore at the time of last connection
	SessionAt  uint64      // set at the time of connection
	LastSeenAt uint64      // set at the time of connection
	Latest     storage.Key // cursor of dbstore when last (continuously set by syncer)
	Synced     bool        // true iff Sync is done up to the last disconnect
	synced     chan bool   // signal that sync stage finished
}

// wrapper of db-s to provide mockable custom local chunk store access to syncer
type DbAccess struct {
	db  *storage.DbStore
	loc *storage.LocalStore
}

func NewDbAccess(loc *storage.LocalStore) *DbAccess {
	return &DbAccess{loc.DbStore.(*storage.DbStore), loc}
}

// to obtain the chunks from key or request db entry only
func (self *DbAccess) get(key storage.Key) (*storage.Chunk, error) {
	return self.loc.Get(key)
}

// current storage counter of chunk db
func (self *DbAccess) counter() uint64 {
	return self.db.Counter()
}

// implemented by dbStoreSyncIterator
type keyIterator interface {
	Next() storage.Key
}

// generator function for iteration by address range and storage counter
func (self *DbAccess) iterator(s *syncState) keyIterator {
	it, err := self.db.NewSyncIterator(*(s.DbSyncState))
	if err != nil {
		return nil
	}
	return keyIterator(it)
}

func (self syncState) String() string {
	if self.Synced {
		return fmt.Sprintf(
			"session started at: %v, last seen at: %v, latest key: %v",
			self.SessionAt, self.LastSeenAt,
			self.Latest.Log(),
		)
	} else {
		return fmt.Sprintf(
			"address: %v-%v, index: %v-%v, session started at: %v, last seen at: %v, latest key: %v",
			self.Start.Log(), self.Stop.Log(),
			self.First, self.Last,
			self.SessionAt, self.LastSeenAt,
			self.Latest.Log(),
		)
	}
}

// syncer parameters (global, not peer specific)
type SyncParams struct {
	RequestDbPath      string // path for request db (leveldb)
	RequestDbBatchSize uint   // nuber of items before batch is saved to requestdb
	KeyBufferSize      uint   // size of key buffer
	SyncBatchSize      uint   // maximum batchsize for outgoing requests
	SyncBufferSize     uint   // size of buffer for
	SyncCacheSize      uint   // cache capacity to store request queue in memory
	SyncPriorities     []uint // list of priority levels for req types 0-3
	SyncModes          []bool // list of sync modes for  for req types 0-3
}

// constructor with default values
func NewSyncParams(bzzdir string) *SyncParams {
	return &SyncParams{
		RequestDbPath:      filepath.Join(bzzdir, "requests"),
		RequestDbBatchSize: requestDbBatchSize,
		KeyBufferSize:      keyBufferSize,
		SyncBufferSize:     syncBufferSize,
		SyncBatchSize:      syncBatchSize,
		SyncCacheSize:      syncCacheSize,
		SyncPriorities:     []uint{High, Medium, Medium, Low, Low},
		SyncModes:          []bool{true, true, true, true, false},
	}
}

// syncer is the agent that manages content distribution/storage replication/chunk storeRequest forwarding
type syncer struct {
	*SyncParams                     // sync parameters
	syncF           func() bool     // if syncing is needed
	key             storage.Key     // remote peers address key
	state           *syncState      // sync state for our dbStore
	syncStates      chan *syncState // different stages of sync
	deliveryRequest chan bool       // one of two triggers needed to send unsyncedKeys
	newUnsyncedKeys chan bool       // one of two triggers needed to send unsynced keys
	quit            chan bool       // signal to quit loops

	// DB related fields
	dbAccess *DbAccess            // access to dbStore
	db       *storage.LDBDatabase // delivery msg db

	// native fields
	queues     [priorities]*syncDb                   // in-memory cache / queues for sync reqs
	keys       [priorities]chan interface{}          // buffer for unsynced keys
	deliveries [priorities]chan *storeRequestMsgData // delivery

	// bzz protocol instance outgoing message callbacks (mockable for testing)
	unsyncedKeys func([]*syncRequest, *syncState) error // send unsyncedKeysMsg
	store        func(*storeRequestMsgData) error       // send storeRequestMsg
}

// a syncer instance is linked to each peer connection
// constructor is called from protocol after successful handshake
// the returned instance is attached to the peer and can be called
// by the forwarder
func newSyncer(
	db *storage.LDBDatabase, remotekey storage.Key,
	dbAccess *DbAccess,
	unsyncedKeys func([]*syncRequest, *syncState) error,
	store func(*storeRequestMsgData) error,
	params *SyncParams,
	state *syncState,
	syncF func() bool,
) (*syncer, error) {

	syncBufferSize := params.SyncBufferSize
	keyBufferSize := params.KeyBufferSize
	dbBatchSize := params.RequestDbBatchSize

	self := &syncer{
		syncF:           syncF,
		key:             remotekey,
		dbAccess:        dbAccess,
		syncStates:      make(chan *syncState, 20),
		deliveryRequest: make(chan bool, 1),
		newUnsyncedKeys: make(chan bool, 1),
		SyncParams:      params,
		state:           state,
		quit:            make(chan bool),
		unsyncedKeys:    unsyncedKeys,
		store:           store,
	}

	// initialising
	for i := 0; i < priorities; i++ {
		self.keys[i] = make(chan interface{}, keyBufferSize)
		self.deliveries[i] = make(chan *storeRequestMsgData)
		// initialise a syncdb instance for each priority queue
		self.queues[i] = newSyncDb(db, remotekey, uint(i), syncBufferSize, dbBatchSize, self.deliver(uint(i)))
	}
	glog.V(logger.Info).Infof("syncer started: %v", state)
	// launch chunk delivery service
	go self.syncDeliveries()
	// launch sync task manager
	if self.syncF() {
		go self.sync()
	}
	// process unsynced keys to broadcast
	go self.syncUnsyncedKeys()

	return self, nil
}

// metadata serialisation
func encodeSync(state *syncState) (*json.RawMessage, error) {
	data, err := json.MarshalIndent(state, "", " ")
	if err != nil {
		return nil, err
	}
	meta := json.RawMessage(data)
	return &meta, nil
}

func decodeSync(meta *json.RawMessage) (*syncState, error) {
	if meta == nil {
		return nil, fmt.Errorf("unable to deserialise sync state from <nil>")
	}
	data := []byte(*(meta))
	if len(data) == 0 {
		return nil, fmt.Errorf("unable to deserialise sync state from <nil>")
	}
	state := &syncState{DbSyncState: &storage.DbSyncState{}}
	err := json.Unmarshal(data, state)
	return state, err
}

/*
 sync implements the syncing script
 * first all items left in the request Db are replayed
   * type = StaleSync
   * Mode: by default once again via confirmation roundtrip
   * Priority: the items are replayed as the proirity specified for StaleSync
   * but within the order respects earlier priority level of request
 * after all items are consumed for a priority level, the the respective
  queue for delivery requests is open (this way new reqs not written to db)
  (TODO: this should be checked)
 * the sync state provided by the remote peer is used to sync history
   * all the backlog from earlier (aborted) syncing is completed starting from latest
   * if Last  < LastSeenAt then all items in between then process all
     backlog from upto last disconnect
   * if Last > 0 &&

 sync is called from the syncer constructor and is not supposed to be used externally
*/
func (self *syncer) sync() {
	state := self.state
	// sync finished
	defer close(self.syncStates)

	// 0. first replay stale requests from request db
	if state.SessionAt == 0 {
		glog.V(logger.Debug).Infof("syncer[%v]: nothing to sync", self.key.Log())
		return
	}
	glog.V(logger.Debug).Infof("syncer[%v]: start replaying stale requests from request db", self.key.Log())
	for p := priorities - 1; p >= 0; p-- {
		self.queues[p].dbRead(false, 0, self.replay())
	}
	glog.V(logger.Debug).Infof("syncer[%v]: done replaying stale requests from request db", self.key.Log())

	// unless peer is synced sync unfinished history beginning on
	if !state.Synced {
		start := state.Start

		if !storage.IsZeroKey(state.Latest) {
			// 1. there is unfinished earlier sync
			state.Start = state.Latest
			glog.V(logger.Debug).Infof("syncer[%v]: start syncronising backlog (unfinished sync: %v)", self.key.Log(), state)
			// blocks while the entire history upto state is synced
			self.syncState(state)
			if state.Last < state.SessionAt {
				state.First = state.Last + 1
			}
		}
		state.Latest = storage.ZeroKey
		state.Start = start
		// 2. sync up to last disconnect1
		if state.First < state.LastSeenAt {
			state.Last = state.LastSeenAt
			glog.V(logger.Debug).Infof("syncer[%v]: start syncronising history upto last disconnect at %v: %v", self.key.Log(), state.LastSeenAt, state)
			self.syncState(state)
			state.First = state.LastSeenAt
		}
		state.Latest = storage.ZeroKey

	} else {
		// synchronisation starts at end of last session
		state.First = state.LastSeenAt
	}

	// 3. sync up to current session start
	// if there have been new chunks since last session
	if state.LastSeenAt < state.SessionAt {
		state.Last = state.SessionAt
		glog.V(logger.Debug).Infof("syncer[%v]: start syncronising history since last disconnect at %v up until session start at %v: %v", self.key.Log(), state.LastSeenAt, state.SessionAt, state)
		// blocks until state syncing is finished
		self.syncState(state)
	}
	glog.V(logger.Info).Infof("syncer[%v]: syncing all history complete", self.key.Log())

}

// wait till syncronised block uptil state is synced
func (self *syncer) syncState(state *syncState) {
	self.syncStates <- state
	select {
	case <-state.synced:
	case <-self.quit:
	}
}

// stop quits both request processor and saves the request cache to disk
func (self *syncer) stop() {
	close(self.quit)
	glog.V(logger.Detail).Infof("syncer[%v]: stop and save sync request db backlog", self.key.Log())
	for _, db := range self.queues {
		db.stop()
	}
}

// rlp serialisable sync request
type syncRequest struct {
	Key      storage.Key
	Priority uint
}

func (self *syncRequest) String() string {
	return fmt.Sprintf("<Key: %v, Priority: %v>", self.Key.Log(), self.Priority)
}

func (self *syncer) newSyncRequest(req interface{}, p int) (*syncRequest, error) {
	key, _, _, _, err := parseRequest(req)
	// TODO: if req has chunk, it should be put in a cache
	// create
	if err != nil {
		return nil, err
	}
	return &syncRequest{key, uint(p)}, nil
}

// serves historical items from the DB
// * read is on demand, blocking unless history channel is read
// * accepts sync requests (syncStates) to create new db iterator
// * closes the channel one iteration finishes
func (self *syncer) syncHistory(state *syncState) chan interface{} {
	var n uint
	history := make(chan interface{})
	glog.V(logger.Debug).Infof("syncer[%v]: syncing history between %v - %v for chunk addresses %v - %v", self.key.Log(), state.First, state.Last, state.Start, state.Stop)
	it := self.dbAccess.iterator(state)
	if it != nil {
		go func() {
			// signal end of the iteration ended
			defer close(history)
		IT:
			for {
				key := it.Next()
				if key == nil {
					break IT
				}
				select {
				// blocking until history channel is read from
				case history <- storage.Key(key):
					n++
					glog.V(logger.Detail).Infof("syncer[%v]: history: %v (%v keys)", self.key.Log(), key.Log(), n)
					state.Latest = key
				case <-self.quit:
					return
				}
			}
			glog.V(logger.Debug).Infof("syncer[%v]: finished syncing history between %v - %v for chunk addresses %v - %v (at %v) (chunks = %v)", self.key.Log(), state.First, state.Last, state.Start, state.Stop, state.Latest, n)
		}()
	}
	return history
}

// triggers key syncronisation
func (self *syncer) sendUnsyncedKeys() {
	select {
	case self.deliveryRequest <- true:
	default:
	}
}

// assembles a new batch of unsynced keys
// * keys are drawn from the key buffers in order of priority queue
// * if the queues of priority for History (HistoryReq) or higher are depleted,
//   historical data is used so historical items are lower priority within
//   their priority group.
// * Order of historical data is unspecified
func (self *syncer) syncUnsyncedKeys() {
	// send out new
	var unsynced []*syncRequest
	var more, justSynced bool
	var keyCount, historyCnt int
	var history chan interface{}

	priority := High
	keys := self.keys[priority]
	var newUnsyncedKeys, deliveryRequest chan bool
	keyCounts := make([]int, priorities)
	histPrior := self.SyncPriorities[HistoryReq]
	syncStates := self.syncStates
	state := self.state

LOOP:
	for {

		var req interface{}
		// select the highest priority channel to read from
		// keys channels are buffered so the highest priority ones
		// are checked first - integrity can only be guaranteed if writing
		// is locked while selecting
		if priority != High || len(keys) == 0 {
			// selection is not needed if the High priority queue has items
			keys = nil
		PRIORITIES:
			for priority = High; priority >= 0; priority-- {
				// the first priority channel that is non-empty will be assigned to keys
				if len(self.keys[priority]) > 0 {
					glog.V(logger.Detail).Infof("syncer[%v]: reading request with	priority %v", self.key.Log(), priority)
					keys = self.keys[priority]
					break PRIORITIES
				}
				glog.V(logger.Detail).Infof("syncer[%v/%v]: queue: [%v, %v, %v]", self.key.Log(), priority, len(self.keys[High]), len(self.keys[Medium]), len(self.keys[Low]))
				// if the input queue is empty on this level, resort to history if there is any
				if uint(priority) == histPrior && history != nil {
					glog.V(logger.Detail).Infof("syncer[%v]: reading history for %v", self.key.Log(), self.key)
					keys = history
					break PRIORITIES
				}
			}
		}

		// if peer ready to receive but nothing to send
		if keys == nil && deliveryRequest == nil {
			// if no items left and switch to waiting mode
			glog.V(logger.Detail).Infof("syncer[%v]: buffers consumed. Waiting", self.key.Log())
			newUnsyncedKeys = self.newUnsyncedKeys
		}

		// send msg iff
		// * peer is ready to receive keys AND (
		// * all queues and history are depleted OR
		// * batch full OR
		// * all history have been consumed, synced)
		if deliveryRequest == nil &&
			(justSynced ||
				len(unsynced) > 0 && keys == nil ||
				len(unsynced) == int(self.SyncBatchSize)) {
			justSynced = false
			// listen to requests
			deliveryRequest = self.deliveryRequest
			newUnsyncedKeys = nil // not care about data until next req comes in
			// set sync to current counter
			// (all nonhistorical outgoing traffic sheduled and persisted
			state.LastSeenAt = self.dbAccess.counter()
			state.Latest = storage.ZeroKey
			glog.V(logger.Detail).Infof("syncer[%v]: sending %v", self.key.Log(), unsynced)
			//  send the unsynced keys
			stateCopy := *state
			err := self.unsyncedKeys(unsynced, &stateCopy)
			if err != nil {
				glog.V(logger.Warn).Infof("syncer[%v]: unable to send unsynced keys: %v", err)
			}
			self.state = state
			glog.V(logger.Debug).Infof("syncer[%v]: --> %v keys sent: (total: %v (%v), history: %v), sent sync state: %v", self.key.Log(), len(unsynced), keyCounts, keyCount, historyCnt, stateCopy)
			unsynced = nil
			keys = nil
		}

		// process item and add it to the batch
		select {
		case <-self.quit:
			break LOOP
		case req, more = <-keys:
			if keys == history && !more {
				glog.V(logger.Detail).Infof("syncer[%v]: syncing history segment complete", self.key.Log())
				// history channel is closed, waiting for new state (called from sync())
				syncStates = self.syncStates
				state.Synced = true // this signals that the  current segment is complete
				select {
				case state.synced <- false:
				case <-self.quit:
					break LOOP
				}
				justSynced = true
				history = nil
			}
		case <-deliveryRequest:
			glog.V(logger.Detail).Infof("syncer[%v]: peer ready to receive", self.key.Log())

			// this 1 cap channel can wake up the loop
			// signaling that peer is ready to receive unsynced Keys
			// the channel is set to nil any further writes will be ignored
			deliveryRequest = nil

		case <-newUnsyncedKeys:
			glog.V(logger.Detail).Infof("syncer[%v]: new unsynced keys available", self.key.Log())
			// this 1 cap channel can wake up the loop
			// signals that data is available to send if peer is ready to receive
			newUnsyncedKeys = nil
			keys = self.keys[High]

		case state, more = <-syncStates:
			// this resets the state
			if !more {
				state = self.state
				glog.V(logger.Detail).Infof("syncer[%v]: (priority %v) syncing complete upto %v)", self.key.Log(), priority, state)
				state.Synced = true
				syncStates = nil
			} else {
				glog.V(logger.Detail).Infof("syncer[%v]: (priority %v) syncing history upto %v priority %v)", self.key.Log(), priority, state, histPrior)
				state.Synced = false
				history = self.syncHistory(state)
				// only one history at a time, only allow another one once the
				// history channel is closed
				syncStates = nil
			}
		}
		if req == nil {
			continue LOOP
		}

		glog.V(logger.Detail).Infof("syncer[%v]: (priority %v) added to unsynced keys: %v", self.key.Log(), priority, req)
		keyCounts[priority]++
		keyCount++
		if keys == history {
			glog.V(logger.Detail).Infof("syncer[%v]: (priority %v) history item %v (synced = %v)", self.key.Log(), priority, req, state.Synced)
			historyCnt++
		}
		if sreq, err := self.newSyncRequest(req, priority); err == nil {
			// extract key from req
			glog.V(logger.Detail).Infof("syncer[%v]: (priority %v): request %v (synced = %v)", self.key.Log(), priority, req, state.Synced)
			unsynced = append(unsynced, sreq)
		} else {
			glog.V(logger.Warn).Infof("syncer[%v]: (priority %v): error creating request for %v: %v)", self.key.Log(), priority, req, state.Synced, err)
		}

	}
}

// delivery loop
// takes into account priority, send store Requests with chunk (delivery)
// idle blocking if no new deliveries in any of the queues
func (self *syncer) syncDeliveries() {
	var req *storeRequestMsgData
	p := High
	var deliveries chan *storeRequestMsgData
	var msg *storeRequestMsgData
	var err error
	var c = [priorities]int{}
	var n = [priorities]int{}
	var total, success uint

	for {
		deliveries = self.deliveries[p]
		select {
		case req = <-deliveries:
			n[p]++
			c[p]++
		default:
			if p == Low {
				// blocking, depletion on all channels, no preference for priority
				select {
				case req = <-self.deliveries[High]:
					n[High]++
				case req = <-self.deliveries[Medium]:
					n[Medium]++
				case req = <-self.deliveries[Low]:
					n[Low]++
				case <-self.quit:
					return
				}
				p = High
			} else {
				p--
				continue
			}
		}
		total++
		msg, err = self.newStoreRequestMsgData(req)
		if err != nil {
			glog.V(logger.Warn).Infof("syncer[%v]: failed to create store request for %v: %v", self.key.Log(), req, err)
		} else {
			err = self.store(msg)
			if err != nil {
				glog.V(logger.Warn).Infof("syncer[%v]: failed to deliver %v: %v", self.key.Log(), req, err)
			} else {
				success++
				glog.V(logger.Detail).Infof("syncer[%v]: %v successfully delivered", self.key.Log(), req)
			}
		}
		if total%self.SyncBatchSize == 0 {
			glog.V(logger.Debug).Infof("syncer[%v]: deliver Total: %v, Success: %v, High: %v/%v, Medium: %v/%v, Low %v/%v", self.key.Log(), total, success, c[High], n[High], c[Medium], n[Medium], c[Low], n[Low])
		}
	}
}

/*
 addRequest handles requests for delivery
 it accepts 4 types:

 * storeRequestMsgData: coming from netstore propagate response
 * chunk: coming from forwarding (questionable: id?)
 * key: from incoming syncRequest
 * syncDbEntry: key,id encoded in db

 If sync mode is on for the type of request, then
 it sends the request to the keys queue of the correct priority
 channel buffered with capacity (SyncBufferSize)

 If sync mode is off then, requests are directly sent to deliveries
*/
func (self *syncer) addRequest(req interface{}, ty int) {
	// retrieve priority for request type name int8

	priority := self.SyncPriorities[ty]
	// sync mode for this type ON
	if self.syncF() || ty == DeliverReq {
		if self.SyncModes[ty] {
			self.addKey(req, priority, self.quit)
		} else {
			self.addDelivery(req, priority, self.quit)
		}
	}
}

// addKey queues sync request for sync confirmation with given priority
// ie the key will go out in an unsyncedKeys message
func (self *syncer) addKey(req interface{}, priority uint, quit chan bool) bool {
	select {
	case self.keys[priority] <- req:
		// this wakes up the unsynced keys loop if idle
		select {
		case self.newUnsyncedKeys <- true:
		default:
		}
		return true
	case <-quit:
		return false
	}
}

// addDelivery queues delivery request for with given priority
// ie the chunk will be delivered ASAP mod priority queueing handled by syncdb
// requests are persisted across sessions for correct sync
func (self *syncer) addDelivery(req interface{}, priority uint, quit chan bool) bool {
	select {
	case self.queues[priority].buffer <- req:
		return true
	case <-quit:
		return false
	}
}

// doDelivery delivers the chunk for the request with given priority
// without queuing
func (self *syncer) doDelivery(req interface{}, priority uint, quit chan bool) bool {
	msgdata, err := self.newStoreRequestMsgData(req)
	if err != nil {
		glog.V(logger.Warn).Infof("unable to deliver request %v: %v", msgdata, err)
		return false
	}
	select {
	case self.deliveries[priority] <- msgdata:
		return true
	case <-quit:
		return false
	}
}

// returns the delivery function for given priority
// passed on to syncDb
func (self *syncer) deliver(priority uint) func(req interface{}, quit chan bool) bool {
	return func(req interface{}, quit chan bool) bool {
		return self.doDelivery(req, priority, quit)
	}
}

// returns the replay function passed on to syncDb
// depending on sync mode settings for BacklogReq,
// re	play of request db backlog sends items via confirmation
// or directly delivers
func (self *syncer) replay() func(req interface{}, quit chan bool) bool {
	sync := self.SyncModes[BacklogReq]
	priority := self.SyncPriorities[BacklogReq]
	// sync mode for this type ON
	if sync {
		return func(req interface{}, quit chan bool) bool {
			return self.addKey(req, priority, quit)
		}
	} else {
		return func(req interface{}, quit chan bool) bool {
			return self.doDelivery(req, priority, quit)
		}

	}
}

// given a request, extends it to a full storeRequestMsgData
// polimorphic: see addRequest for the types accepted
func (self *syncer) newStoreRequestMsgData(req interface{}) (*storeRequestMsgData, error) {

	key, id, chunk, sreq, err := parseRequest(req)
	if err != nil {
		return nil, err
	}

	if sreq == nil {
		if chunk == nil {
			var err error
			chunk, err = self.dbAccess.get(key)
			if err != nil {
				return nil, err
			}
		}

		sreq = &storeRequestMsgData{
			Id:    id,
			Key:   chunk.Key,
			SData: chunk.SData,
		}
	}

	return sreq, nil
}

// parse request types and extracts, key, id, chunk, request if available
// does not do chunk lookup !
func parseRequest(req interface{}) (storage.Key, uint64, *storage.Chunk, *storeRequestMsgData, error) {
	var key storage.Key
	var entry *syncDbEntry
	var chunk *storage.Chunk
	var id uint64
	var ok bool
	var sreq *storeRequestMsgData
	var err error

	if key, ok = req.(storage.Key); ok {
		id = generateId()

	} else if entry, ok = req.(*syncDbEntry); ok {
		id = binary.BigEndian.Uint64(entry.val[32:])
		key = storage.Key(entry.val[:32])

	} else if chunk, ok = req.(*storage.Chunk); ok {
		key = chunk.Key
		id = generateId()

	} else if sreq, ok = req.(*storeRequestMsgData); ok {
		key = sreq.Key
	} else {
		err = fmt.Errorf("type not allowed: %v (%T)", req, req)
	}

	return key, id, chunk, sreq, err
}
