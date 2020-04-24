// Copyright 2020 The go-ethereum Authors
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

package utils

import (
	"errors"
	"fmt"
	"reflect"
	"sync"
	"time"
	"unsafe"

	"github.com/ethereum/go-ethereum/common/mclock"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/p2p/enode"
	"github.com/ethereum/go-ethereum/p2p/enr"
	"github.com/ethereum/go-ethereum/rlp"
)

type (
	// NodeStateMachine connects different system components operating on subsets of
	// network nodes. Node states are represented by 64 bit vectors with each bit assigned
	// to a state flag. Each state flag has a descriptor structure and the mapping is
	// created automatically. It is possible to subscribe to subsets of state flags and
	// receive a callback if one of the nodes has a relevant state flag changed.
	// Callbacks can also modify further flags of the same node or other nodes. State
	// updates only return after all immediate effects throughout the system have happened
	// (deadlocks should be avoided by design of the implemented state logic). The caller
	// can also add timeouts assigned to a certain node and a subset of state flags.
	// If the timeout elapses, the flags are reset. If all relevant flags are reset then
	// the timer is dropped. The state flags and the associated timers are persisted in
	// the database if the flag descriptor enables saving.
	//
	// Extra node fields can also be registered so system components can also store more
	// complex state for each node that is relevant to them, without creating a custom
	// peer set. Fields can be shared across multiple components if they all know the
	// field ID. Fields are also assigned to a set of state flags and the contents of
	// each field is only retained for a certain node as long as at least one of these
	// flags is set (meaning that the node is still interesting to the component(s) which
	// are interested in the given field). Persistent fields should have an encoder and
	// a decoder function.
	NodeStateMachine struct {
		started, stopped        bool
		lock                    sync.Mutex
		clock                   mclock.Clock
		db                      ethdb.KeyValueStore
		dbMappingKey, dbNodeKey []byte
		mappings                []nsMapping
		currentMapping          int
		nodes                   map[enode.ID]*nodeInfo
		offlineCallbackList     []offlineCallback

		// Registered state flags or fields. Modifications are allowed
		// only when the node state machine has not been initialized.
		nodeStates       map[*NodeStateFlag]int
		nodeStateNameMap map[string]int
		nodeFields       []*nodeFieldInfo
		nodeFieldMap     map[*NodeField]int
		nodeFieldNameMap map[string]int
		saveFlags        NodeStateBitMask

		// Installed callbacks. Modifications are allowed only when the
		// node state machine has not been initialized.
		stateSubs []nodeStateSub

		// Testing hooks, only for testing purposes.
		saveNodeHook func(*nodeInfo)
	}

	// NodeStateFlag describes a node state flag. Each registered instance is automatically
	// mapped to a bit of the 64 bit node states.
	// If saveImmediately is true then the node is saved each time the flag is switched on
	// or off. If saveAtShutdown is true then the node is saved when state machine is shutdown.
	NodeStateFlag struct {
		name       string
		persistent bool
	}

	// NodeField describes an optional node field of the given type. The contents
	// of the field are only retained for each node as long as at least one of the
	// specified flags is set. If all relevant flags are reset then the field is removed
	// after all callbacks of the state change are processed.
	NodeField struct {
		name   string
		ftype  reflect.Type
		encode func(interface{}) ([]byte, error)
		decode func([]byte) (interface{}, error)
	}

	NodeStateSetup struct {
		Flags  []*NodeStateFlag
		Fields []*NodeField
	}

	// nsMapping describes an index mapping of node state flags and fields. Mapping is
	// determined during startup, before loading node data from the database. The used
	// mapping is saved for each node and is converted upon loading if it has changed
	// since the last time it was saved.
	nsMapping struct {
		States, Fields []string
	}

	// NodeStateBitMask describes a node state or state mask. It represents a subset
	// of node flags with each bit assigned to a flag index (LSB represents flag 0).
	NodeStateBitMask uint64

	// NodeStateCallback is a subscription callback which is called when one of the
	// state flags that is included in the subscription state mask is changed.
	// Note: oldState and newState are also masked with the subscription mask so only
	// the relevant bits are included.
	NodeStateCallback func(n *enode.Node, oldState, newState NodeStateBitMask)

	NodeFieldCallback func(n *enode.Node, state NodeStateBitMask, oldValue, newValue interface{})

	// nodeInfo contains node state, fields and state timeouts
	nodeInfo struct {
		node      *enode.Node
		state     NodeStateBitMask
		timeouts  []*nodeStateTimeout
		fields    []interface{}
		db, dirty bool
	}

	nodeInfoEnc struct {
		Enr     enr.Record
		Mapping uint
		State   NodeStateBitMask
		Fields  [][]byte
	}

	nodeStateSub struct {
		mask     NodeStateBitMask
		callback NodeStateCallback
	}

	nodeStateTimeout struct {
		mask  NodeStateBitMask
		timer mclock.Timer
	}

	nodeFieldInfo struct {
		*NodeField
		subs []NodeFieldCallback
	}

	offlineCallback struct {
		node   *enode.Node
		state  NodeStateBitMask
		fields []interface{}
	}
)

var (
	errAlreadyStarted = errors.New("state machine already started")
	errNotStarted     = errors.New("state machine not started yet")
	errStateOverflow  = errors.New("registered state flag exceeds the limit")
	errNameCollision  = errors.New("state flag or node field name collision")
	errOutOfBound     = errors.New("out of bound")
	errInvalidField   = errors.New("invalid field type")

	OfflineFlag = NewFlag("offline")
)

// OfflineState is a special state that is assumed to be set before a node is loaded from
// the database and after it is shut down.
const OfflineState = NodeStateBitMask(1)

// NewNodeStateMachine creates a new node state machine.
// If db is not nil then the node states, fields and active timeouts are persisted.
// Persistence can be enabled or disabled for each state flag and field.
func NewNodeStateMachine(db ethdb.KeyValueStore, dbKey []byte, clock mclock.Clock, setup NodeStateSetup) *NodeStateMachine {
	// init flag is always mapped to index 0 (OfflineState bit mask)
	flags := append([]*NodeStateFlag{OfflineFlag}, setup.Flags...)
	if len(flags) > 8*int(unsafe.Sizeof(NodeStateBitMask(0))) {
		panic("Too many node state flags")
	}
	ns := &NodeStateMachine{
		db:               db,
		dbMappingKey:     append(dbKey, []byte("mapping:")...),
		dbNodeKey:        append(dbKey, []byte("node:")...),
		clock:            clock,
		nodes:            make(map[enode.ID]*nodeInfo),
		nodeStates:       make(map[*NodeStateFlag]int),
		nodeStateNameMap: make(map[string]int),
		nodeFields:       make([]*nodeFieldInfo, len(setup.Fields)),
		nodeFieldMap:     make(map[*NodeField]int),
		nodeFieldNameMap: make(map[string]int),
	}
	for index, flag := range flags {
		if _, ok := ns.nodeStateNameMap[flag.name]; ok {
			panic("Node state flag name collision")
		}
		ns.nodeStates[flag] = index
		ns.nodeStateNameMap[flag.name] = index
		if flag.persistent {
			ns.saveFlags |= NodeStateBitMask(1) << index
		}
	}
	for index, field := range setup.Fields {
		if _, ok := ns.nodeFieldNameMap[field.name]; ok {
			panic("Node field name collision")
		}
		ns.nodeFields[index] = &nodeFieldInfo{NodeField: field}
		ns.nodeFieldMap[field] = index
		ns.nodeFieldNameMap[field.name] = index
	}
	return ns
}

// NewFlag creates a new node state flag
func NewFlag(name string) *NodeStateFlag {
	return &NodeStateFlag{
		name: name,
	}
}

// NewPersistentFlag creates a new persistent node state flag
func NewPersistentFlag(name string) *NodeStateFlag {
	return &NodeStateFlag{
		name:       name,
		persistent: true,
	}
}

// NewField creates a new node state field
func NewField(name string, ftype reflect.Type) *NodeField {
	return &NodeField{
		name:  name,
		ftype: ftype,
	}
}

// NewPersistentField creates a new persistent node field
func NewPersistentField(name string, ftype reflect.Type, encode func(interface{}) ([]byte, error), decode func([]byte) (interface{}, error)) *NodeField {
	return &NodeField{
		name:   name,
		ftype:  ftype,
		encode: encode,
		decode: decode,
	}
}

// SubscribeState adds a node state subscription. The callback is called while the state
// machine mutex is not held and it is allowed to make further state updates. All immediate
// changes throughout the system are processed in the same thread/goroutine. It is the
// responsibility of the implemented state logic to avoid deadlocks caused by the callbacks,
// infinite toggling of flags or hazardous/non-deterministic state changes.
// State subscriptions should be installed before loading the node database or making the
// first state update.
func (ns *NodeStateMachine) SubscribeState(mask NodeStateBitMask, callback NodeStateCallback) error {
	ns.lock.Lock()
	defer ns.lock.Unlock()

	if ns.started {
		return errAlreadyStarted
	}
	ns.stateSubs = append(ns.stateSubs, nodeStateSub{mask, callback})
	return nil
}

func (ns *NodeStateMachine) SubscribeField(field int, callback NodeFieldCallback) error {
	ns.lock.Lock()
	defer ns.lock.Unlock()

	if ns.started {
		return errAlreadyStarted
	}
	if field >= len(ns.nodeFields) {
		log.Error("Field index out of bounds", "index", field, "field count", len(ns.nodeFields))
		return errOutOfBound
	}
	f := ns.nodeFields[field]
	f.subs = append(f.subs, callback)
	return nil
}

// newNode creates a new nodeInfo
func (ns *NodeStateMachine) newNode(n *enode.Node) *nodeInfo {
	return &nodeInfo{node: n, fields: make([]interface{}, len(ns.nodeFields))}
}

// checkStarted checks whether the state machine has already been started and panics otherwise.
func (ns *NodeStateMachine) checkStarted() {
	if !ns.started {
		panic(errNotStarted)
	}
}

// Start starts the state machine, enabling state and field operations and disabling
// further setup operations (flag and field registrations).
func (ns *NodeStateMachine) Start() {
	ns.lock.Lock()
	if ns.started {
		panic(errAlreadyStarted)
	}
	ns.started = true
	if ns.db != nil {
		ns.loadFromDb()
	}
	ns.lock.Unlock()
	ns.offlineCallbacks(true)
}

// Stop stops the state machine and saves its state if a database was supplied
func (ns *NodeStateMachine) Stop() {
	ns.lock.Lock()
	for _, node := range ns.nodes {
		fields := make([]interface{}, len(node.fields))
		copy(fields, node.fields)
		ns.offlineCallbackList = append(ns.offlineCallbackList, offlineCallback{node.node, node.state, fields})
	}
	ns.stopped = true
	if ns.db != nil {
		ns.saveToDb()
		ns.lock.Unlock()
	} else {
		ns.lock.Unlock()
	}
	ns.offlineCallbacks(false)
}

// loadFromDb loads persisted node states from the database
func (ns *NodeStateMachine) loadFromDb() {
	if enc, err := ns.db.Get(ns.dbMappingKey); err == nil {
		if err := rlp.DecodeBytes(enc, &ns.mappings); err != nil {
			log.Error("Failed to decode scheme", "error", err)
			return
		}
	}
	mapping := nsMapping{
		States: make([]string, len(ns.nodeStates)),
		Fields: make([]string, len(ns.nodeFields)),
	}
	for flag, index := range ns.nodeStates {
		mapping.States[index] = flag.name
	}
	for index, field := range ns.nodeFields {
		mapping.Fields[index] = field.name
	}
	ns.currentMapping = -1
loop:
	for i, m := range ns.mappings {
		if len(m.States) != len(mapping.States) {
			continue loop
		}
		if len(m.Fields) != len(mapping.Fields) {
			continue loop
		}
		for j, s := range mapping.States {
			if m.States[j] != s {
				continue loop
			}
		}
		for j, s := range mapping.Fields {
			if m.Fields[j] != s {
				continue loop
			}
		}
		ns.currentMapping = i
		break
	}
	if ns.currentMapping == -1 {
		ns.currentMapping = len(ns.mappings)
		ns.mappings = append(ns.mappings, mapping)

		// Scheme should be persisted properly, otherwise
		// all persisted data can't be resolved next time.
		enc, err := rlp.EncodeToBytes(ns.mappings)
		if err != nil {
			panic("Failed to encode scheme")
		}
		if err := ns.db.Put(ns.dbMappingKey, enc); err != nil {
			panic("Failed to save scheme")
		}
	}

	it := ns.db.NewIteratorWithPrefix(ns.dbNodeKey)
	for it.Next() {
		var id enode.ID
		if len(it.Key()) != len(ns.dbNodeKey)+len(id) {
			log.Error("Node state db entry with invalid length", "found", len(it.Key()), "expected", len(ns.dbNodeKey)+len(id))
			continue
		}
		copy(id[:], it.Key()[len(ns.dbNodeKey):])
		ns.decodeNode(id, it.Value())
	}
}

type dummyIdentity enode.ID

func (id dummyIdentity) Verify(r *enr.Record, sig []byte) error { return nil }
func (id dummyIdentity) NodeAddr(r *enr.Record) []byte          { return id[:] }

// decodeNode decodes a node database entry and adds it to the node set if successful
func (ns *NodeStateMachine) decodeNode(id enode.ID, data []byte) {
	var enc nodeInfoEnc
	if err := rlp.DecodeBytes(data, &enc); err != nil {
		log.Error("Failed to decode node info", "id", id, "error", err)
		return
	}
	n, _ := enode.New(dummyIdentity(id), &enc.Enr)
	node := ns.newNode(n)
	node.db = true

	if int(enc.Mapping) >= len(ns.mappings) {
		log.Error("Unknown scheme", "id", id, "index", enc.Mapping, "len", len(ns.mappings))
		return
	}
	encMapping := ns.mappings[int(enc.Mapping)]
	if len(enc.Fields) != len(encMapping.Fields) {
		log.Error("Invalid node field count", "id", id, "stored", len(enc.Fields), "mapping", len(encMapping.Fields))
		return
	}
	// convertMask converts a old format state/mask to the latest version.
	convertMask := func(schemeID int, scheme []string, state NodeStateBitMask) (NodeStateBitMask, bool) {
		if schemeID == ns.currentMapping {
			return state, true // Nothing need to be changed
		}
		var converted NodeStateBitMask
		for i, name := range scheme {
			if (state & (NodeStateBitMask(1) << i)) != 0 {
				if index, ok := ns.nodeStateNameMap[name]; ok {
					converted |= NodeStateBitMask(1) << index
				} else {
					log.Error("Unknown state flag", "name", name)
					return NodeStateBitMask(0), false // unknown flag
				}
			}
		}
		return converted, true
	}
	// Resolve persisted node fields
	for i, encField := range enc.Fields {
		if len(encField) == 0 {
			continue
		}
		index := i
		if int(enc.Mapping) != ns.currentMapping {
			name := encMapping.Fields[i]
			var ok bool
			if index, ok = ns.nodeFieldNameMap[name]; !ok {
				log.Error("Unknown node field", "id", id, "field name", name)
				return
			}
		}
		if decode := ns.nodeFields[index].decode; decode != nil {
			if field, err := decode(encField); err == nil {
				node.fields[index] = field
			} else {
				log.Error("Failed to decode node field", "id", id, "field name", ns.nodeFields[index].name, "error", err)
				return
			}
		} else {
			log.Error("Cannot decode node field", "id", id, "field name", ns.nodeFields[index].name)
			return
		}
	}
	// Resolve node state
	state, success := convertMask(int(enc.Mapping), encMapping.States, enc.State)
	if !success {
		return
	}
	// It's a compatible node record, add it to set.
	ns.nodes[id] = node
	node.state = state
	fields := make([]interface{}, len(node.fields))
	copy(fields, node.fields)
	ns.offlineCallbackList = append(ns.offlineCallbackList, offlineCallback{node.node, state, fields})
	log.Debug("Loaded node state", "id", id, "state", ns.stateToString(enc.State))
}

// saveNode saves the given node info to the database
func (ns *NodeStateMachine) saveNode(id enode.ID, node *nodeInfo) error {
	if ns.db == nil {
		return nil
	}

	storedState := node.state & ns.saveFlags
	for _, t := range node.timeouts {
		storedState &= ^t.mask
	}
	if storedState == 0 {
		if node.db {
			node.db = false
			ns.deleteNode(id)
		}
		node.dirty = false
		return nil
	}

	enc := nodeInfoEnc{
		Enr:     *node.node.Record(),
		Mapping: uint(ns.currentMapping),
		State:   storedState,
		Fields:  make([][]byte, len(ns.nodeFields)),
	}
	log.Debug("Saved node state", "id", id, "state", ns.stateToString(enc.State))
	for i, f := range node.fields {
		if f == nil {
			continue
		}
		encode := ns.nodeFields[i].encode
		if encode == nil {
			continue
		}
		blob, err := encode(f)
		if err != nil {
			return err
		}
		enc.Fields[i] = blob
	}
	data, err := rlp.EncodeToBytes(&enc)
	if err != nil {
		return err
	}
	if err := ns.db.Put(append(ns.dbNodeKey, id[:]...), data); err != nil {
		return err
	}
	node.dirty, node.db = false, true

	if ns.saveNodeHook != nil {
		ns.saveNodeHook(node)
	}
	return nil
}

// deleteNode removes a node info from the database
func (ns *NodeStateMachine) deleteNode(id enode.ID) {
	ns.db.Delete(append(ns.dbNodeKey, id[:]...))
}

// saveToDb saves all nodes that have been changed but not saved immediately
func (ns *NodeStateMachine) saveToDb() {
	for id, node := range ns.nodes {
		if node.dirty {
			err := ns.saveNode(id, node)
			if err != nil {
				log.Error("Failed to save node", "id", id, "error", err)
			}
		}
	}
}

func (ns *NodeStateMachine) storeNode(n *enode.Node) (enode.ID, *nodeInfo) {
	id := n.ID()
	node := ns.nodes[id]
	if node != nil {
		node.node = n //TODO check whether the node has newer ENR than the stored one
	}
	return id, node
}

// Persist saves the state of the given node immediately
func (ns *NodeStateMachine) Persist(n *enode.Node) error {
	ns.lock.Lock()
	defer ns.lock.Unlock()

	if id, node := ns.storeNode(n); node != nil && node.dirty {
		err := ns.saveNode(id, node)
		if err != nil {
			log.Error("Failed to save node", "id", id, "error", err)
		}
		return err
	}
	return nil
}

// SetState updates the given node state flags and processes all resulting callbacks.
// It only returns after all subsequent immediate changes (including those changed by the
// callbacks) have been processed.
func (ns *NodeStateMachine) SetState(n *enode.Node, set, reset NodeStateBitMask, timeout time.Duration) {
	ns.lock.Lock()
	ns.checkStarted()
	if ns.stopped {
		ns.lock.Unlock()
		return
	}

	id, node := ns.storeNode(n)
	if node == nil {
		if set == 0 {
			return
		}
		node = ns.newNode(n)
		ns.nodes[id] = node
	}
	oldState := node.state
	newState := (node.state & (^reset)) | set
	changed := oldState ^ newState
	node.state = newState

	ns.removeTimeouts(node, reset|set)
	if newState == oldState {
		ns.lock.Unlock()
		return
	}
	if timeout != 0 && newState != 0 {
		ns.addTimeout(n, set, timeout)
	}
	if newState == 0 {
		delete(ns.nodes, id)
		if node.db {
			ns.deleteNode(id)
		}
	} else {
		if changed&ns.saveFlags != 0 {
			node.dirty = true
		}
	}
	ns.lock.Unlock()
	// call state update subscription callbacks without holding the mutex
	for _, sub := range ns.stateSubs {
		if changed&sub.mask != 0 {
			sub.callback(n, oldState&sub.mask, newState&sub.mask)
		}
	}
	if newState == 0 {
		// call field subscriptions for discarded fields
		for i, v := range node.fields {
			if v != nil {
				f := ns.nodeFields[i]
				if len(f.subs) > 0 {
					for _, cb := range f.subs {
						cb(n, 0, v, nil)
					}
				}
			}
		}
	}
}

// offlineCallbacks calls state update callbacks at startup or shutdown
func (ns *NodeStateMachine) offlineCallbacks(start bool) {
	for _, cb := range ns.offlineCallbackList {
		for _, sub := range ns.stateSubs {
			offState := OfflineState & sub.mask
			onState := cb.state & sub.mask
			if offState != onState {
				if start {
					sub.callback(cb.node, offState, onState)
				} else {
					sub.callback(cb.node, onState, offState)
				}
			}
		}
		for i, f := range cb.fields {
			if f != nil && ns.nodeFields[i].subs != nil {
				for _, fsub := range ns.nodeFields[i].subs {
					if start {
						fsub(cb.node, OfflineState, nil, f)
					} else {
						fsub(cb.node, OfflineState, f, nil)
					}
				}
			}
		}
	}
	ns.offlineCallbackList = nil
}

// AddTimeout adds a node state timeout associated to the given state flag(s).
// After the specified time interval, the relevant states will be reset.
func (ns *NodeStateMachine) AddTimeout(n *enode.Node, mask NodeStateBitMask, timeout time.Duration) {
	ns.lock.Lock()
	defer ns.lock.Unlock()

	ns.checkStarted()
	if ns.stopped {
		return
	}
	ns.addTimeout(n, mask, timeout)
}

// addTimeout adds a node state timeout associated to the given state flag(s).
func (ns *NodeStateMachine) addTimeout(n *enode.Node, mask NodeStateBitMask, timeout time.Duration) {
	_, node := ns.storeNode(n)
	if node == nil {
		return
	}
	mask &= node.state
	if mask == 0 {
		return
	}
	ns.removeTimeouts(node, mask)
	t := &nodeStateTimeout{mask: mask}
	t.timer = ns.clock.AfterFunc(timeout, func() {
		ns.SetState(n, 0, t.mask, 0)
	})
	node.timeouts = append(node.timeouts, t)
	if mask&ns.saveFlags != 0 {
		node.dirty = true
	}
}

// removeTimeout removes node state timeouts associated to the given state flag(s).
// If a timeout was associated to multiple flags which are not all included in the
// specified remove mask then only the included flags are de-associated and the timer
// stays active.
func (ns *NodeStateMachine) removeTimeouts(node *nodeInfo, mask NodeStateBitMask) {
	for i := 0; i < len(node.timeouts); i++ {
		t := node.timeouts[i]
		match := t.mask & mask
		if match == 0 {
			continue
		}
		t.mask -= match
		if t.mask != 0 {
			continue
		}
		t.timer.Stop()
		node.timeouts[i] = node.timeouts[len(node.timeouts)-1]
		node.timeouts = node.timeouts[:len(node.timeouts)-1]
		i--
		if match&ns.saveFlags != 0 {
			node.dirty = true
		}
	}
}

// GetField retrieves the given field of the given node
func (ns *NodeStateMachine) GetField(n *enode.Node, fieldId int) interface{} {
	ns.lock.Lock()
	defer ns.lock.Unlock()

	ns.checkStarted()
	if ns.stopped {
		return nil
	}
	if _, node := ns.storeNode(n); node != nil && fieldId < len(node.fields) {
		return node.fields[fieldId]
	}
	return nil
}

// SetField sets the given field of the given node
func (ns *NodeStateMachine) SetField(n *enode.Node, fieldId int, value interface{}) error {
	ns.lock.Lock()
	ns.checkStarted()
	if ns.stopped {
		ns.lock.Unlock()
		return nil
	}
	_, node := ns.storeNode(n)
	if node == nil {
		ns.lock.Unlock()
		return nil
	}
	// Refuse to set field if it's unknown or the relevant state is unset.
	if fieldId < 0 || fieldId >= len(ns.nodeFields) {
		log.Error("Field index out of bounds", "index", fieldId, "field count", len(ns.nodeFields))
		ns.lock.Unlock()
		return errOutOfBound
	}
	f := ns.nodeFields[fieldId]
	if value != nil && reflect.TypeOf(value) != f.ftype {
		log.Error("Invalid field type", "type", reflect.TypeOf(value), "required", f.ftype)
		ns.lock.Unlock()
		return errInvalidField
	}
	oldValue := node.fields[fieldId]
	if value == oldValue {
		ns.lock.Unlock()
		return nil
	}
	node.fields[fieldId] = value
	if f.encode != nil {
		node.dirty = true
	}

	state := node.state
	ns.lock.Unlock()
	if len(f.subs) > 0 {
		for _, cb := range f.subs {
			cb(n, state, oldValue, value)
		}
	}
	return nil
}

// StateMask returns the state mask associated with given flag.
func (ns *NodeStateMachine) StateMask(flag *NodeStateFlag) NodeStateBitMask {
	if state, ok := ns.nodeStates[flag]; ok {
		return NodeStateBitMask(1) << state
	}
	log.Error("Unknown state flag", "name", flag.name)
	return NodeStateBitMask(0)
}

// StatesMask assigns a bit index to the given flags if they have not been mapped before and
// returns the node state bit mask.
// State flags should be mapped before loading the node database or making the first state update.
func (ns *NodeStateMachine) StatesMask(flags []*NodeStateFlag) NodeStateBitMask {
	var mask NodeStateBitMask
	for _, flag := range flags {
		mask |= ns.StateMask(flag)
	}
	return mask
}

// FieldIndex returns the index of the given field
func (ns *NodeStateMachine) FieldIndex(field *NodeField) int {
	if index, ok := ns.nodeFieldMap[field]; ok {
		return index
	}
	log.Error("Unknown field", "name", field.name)
	return -1
}

// stateToString returns a list of the names of the flags specified in the bit mask
func (ns *NodeStateMachine) stateToString(states NodeStateBitMask) string {
	s := "["
	comma := false
	for field, index := range ns.nodeStates {
		if states&(NodeStateBitMask(1)<<index) != 0 {
			if comma {
				s = s + ", "
			}
			s = s + field.name
			comma = true
		}
	}
	s = s + "]"
	return s
}

// String returns the 2-based format to better represent "bits"
func (mask NodeStateBitMask) String() string {
	return fmt.Sprintf("%b", mask)
}

func (ns *NodeStateMachine) ForEach(require, disable NodeStateBitMask, cb func(n *enode.Node, state NodeStateBitMask)) {
	ns.lock.Lock()
	type callback struct {
		node  *enode.Node
		state NodeStateBitMask
	}
	var callbacks []callback
	for _, node := range ns.nodes {
		if node.state&require == require && node.state&disable == 0 {
			callbacks = append(callbacks, callback{node.node, node.state & (require | disable)})
		}
	}
	ns.lock.Unlock()
	for _, c := range callbacks {
		cb(c.node, c.state)
	}
}

func (ns *NodeStateMachine) GetNode(id enode.ID) *enode.Node {
	ns.lock.Lock()
	defer ns.lock.Unlock()

	if node := ns.nodes[id]; node != nil {
		return node.node
	}
	return nil
}
