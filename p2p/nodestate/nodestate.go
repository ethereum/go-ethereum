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

package nodestate

import (
	"errors"
	"reflect"
	"sync"
	"time"
	"unsafe"

	"github.com/ethereum/go-ethereum/common/mclock"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/metrics"
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
	// the timer is dropped. State flags with no timeout are persisted in the database
	// if the flag descriptor enables saving. If a node has no state flags set at any
	// moment then it is discarded.
	//
	// Extra node fields can also be registered so system components can also store more
	// complex state for each node that is relevant to them, without creating a custom
	// peer set. Fields can be shared across multiple components if they all know the
	// field ID. Subscription to fields is also possible. Persistent fields should have
	// an encoder and a decoder function.
	NodeStateMachine struct {
		started, stopped    bool
		lock                sync.Mutex
		clock               mclock.Clock
		db                  ethdb.KeyValueStore
		dbNodeKey           []byte
		nodes               map[enode.ID]*nodeInfo
		offlineCallbackList []offlineCallback

		// Registered state flags or fields. Modifications are allowed
		// only when the node state machine has not been started.
		setup     *Setup
		fields    []*fieldInfo
		saveFlags bitMask

		// Installed callbacks. Modifications are allowed only when the
		// node state machine has not been started.
		stateSubs []stateSub

		// Testing hooks, only for testing purposes.
		saveNodeHook func(*nodeInfo)
	}

	// Flags represents a set of flags from a certain setup
	Flags struct {
		mask  bitMask
		setup *Setup
	}

	// Field represents a field from a certain setup
	Field struct {
		index int
		setup *Setup
	}

	// flagDefinition describes a node state flag. Each registered instance is automatically
	// mapped to a bit of the 64 bit node states.
	// If persistent is true then the node is saved when state machine is shutdown.
	flagDefinition struct {
		name       string
		persistent bool
	}

	// fieldDefinition describes an optional node field of the given type. The contents
	// of the field are only retained for each node as long as at least one of the
	// state flags is set.
	fieldDefinition struct {
		name   string
		ftype  reflect.Type
		encode func(interface{}) ([]byte, error)
		decode func([]byte) (interface{}, error)
	}

	// stateSetup contains the list of flags and fields used by the application
	Setup struct {
		Version uint
		flags   []flagDefinition
		fields  []fieldDefinition
	}

	// bitMask describes a node state or state mask. It represents a subset
	// of node flags with each bit assigned to a flag index (LSB represents flag 0).
	bitMask uint64

	// StateCallback is a subscription callback which is called when one of the
	// state flags that is included in the subscription state mask is changed.
	// Note: oldState and newState are also masked with the subscription mask so only
	// the relevant bits are included.
	StateCallback func(n *enode.Node, oldState, newState Flags)

	// FieldCallback is a subscription callback which is called when the value of
	// a specific field is changed.
	FieldCallback func(n *enode.Node, state Flags, oldValue, newValue interface{})

	// nodeInfo contains node state, fields and state timeouts
	nodeInfo struct {
		node      *enode.Node
		state     bitMask
		timeouts  []*nodeStateTimeout
		fields    []interface{}
		db, dirty bool
	}

	nodeInfoEnc struct {
		Enr     enr.Record
		Version uint
		State   bitMask
		Fields  [][]byte
	}

	stateSub struct {
		mask     bitMask
		callback StateCallback
	}

	nodeStateTimeout struct {
		mask  bitMask
		timer mclock.Timer
	}

	fieldInfo struct {
		fieldDefinition
		subs []FieldCallback
	}

	offlineCallback struct {
		node   *enode.Node
		state  bitMask
		fields []interface{}
	}
)

// offlineState is a special state that is assumed to be set before a node is loaded from
// the database and after it is shut down.
const offlineState = bitMask(1)

// NewFlag creates a new node state flag
func (s *Setup) NewFlag(name string) Flags {
	if s.flags == nil {
		s.flags = []flagDefinition{{name: "offline"}}
	}
	f := Flags{mask: bitMask(1) << uint(len(s.flags)), setup: s}
	s.flags = append(s.flags, flagDefinition{name: name})
	return f
}

// NewPersistentFlag creates a new persistent node state flag
func (s *Setup) NewPersistentFlag(name string) Flags {
	if s.flags == nil {
		s.flags = []flagDefinition{{name: "offline"}}
	}
	f := Flags{mask: bitMask(1) << uint(len(s.flags)), setup: s}
	s.flags = append(s.flags, flagDefinition{name: name, persistent: true})
	return f
}

// OfflineFlag returns the system-defined offline flag belonging to the given setup
func (s *Setup) OfflineFlag() Flags {
	return Flags{mask: offlineState, setup: s}
}

// NewField creates a new node state field
func (s *Setup) NewField(name string, ftype reflect.Type) Field {
	f := Field{index: len(s.fields), setup: s}
	s.fields = append(s.fields, fieldDefinition{
		name:  name,
		ftype: ftype,
	})
	return f
}

// NewPersistentField creates a new persistent node field
func (s *Setup) NewPersistentField(name string, ftype reflect.Type, encode func(interface{}) ([]byte, error), decode func([]byte) (interface{}, error)) Field {
	f := Field{index: len(s.fields), setup: s}
	s.fields = append(s.fields, fieldDefinition{
		name:   name,
		ftype:  ftype,
		encode: encode,
		decode: decode,
	})
	return f
}

// flagOp implements binary flag operations and also checks whether the operands belong to the same setup
func flagOp(a, b Flags, trueIfA, trueIfB, trueIfBoth bool) Flags {
	if a.setup == nil {
		if a.mask != 0 {
			panic("Node state flags have no setup reference")
		}
		a.setup = b.setup
	}
	if b.setup == nil {
		if b.mask != 0 {
			panic("Node state flags have no setup reference")
		}
		b.setup = a.setup
	}
	if a.setup != b.setup {
		panic("Node state flags belong to a different setup")
	}
	res := Flags{setup: a.setup}
	if trueIfA {
		res.mask |= a.mask & ^b.mask
	}
	if trueIfB {
		res.mask |= b.mask & ^a.mask
	}
	if trueIfBoth {
		res.mask |= a.mask & b.mask
	}
	return res
}

// And returns the set of flags present in both a and b
func (a Flags) And(b Flags) Flags { return flagOp(a, b, false, false, true) }

// AndNot returns the set of flags present in a but not in b
func (a Flags) AndNot(b Flags) Flags { return flagOp(a, b, true, false, false) }

// Or returns the set of flags present in either a or b
func (a Flags) Or(b Flags) Flags { return flagOp(a, b, true, true, true) }

// Xor returns the set of flags present in either a or b but not both
func (a Flags) Xor(b Flags) Flags { return flagOp(a, b, true, true, false) }

// HasAll returns true if b is a subset of a
func (a Flags) HasAll(b Flags) bool { return flagOp(a, b, false, true, false).mask == 0 }

// HasNone returns true if a and b have no shared flags
func (a Flags) HasNone(b Flags) bool { return flagOp(a, b, false, false, true).mask == 0 }

// Equals returns true if a and b have the same flags set
func (a Flags) Equals(b Flags) bool { return flagOp(a, b, true, true, false).mask == 0 }

// IsEmpty returns true if a has no flags set
func (a Flags) IsEmpty() bool { return a.mask == 0 }

// MergeFlags merges multiple sets of state flags
func MergeFlags(list ...Flags) Flags {
	if len(list) == 0 {
		return Flags{}
	}
	res := list[0]
	for i := 1; i < len(list); i++ {
		res = res.Or(list[i])
	}
	return res
}

// String returns a list of the names of the flags specified in the bit mask
func (f Flags) String() string {
	if f.mask == 0 {
		return "[]"
	}
	s := "["
	comma := false
	for index, flag := range f.setup.flags {
		if f.mask&(bitMask(1)<<uint(index)) != 0 {
			if comma {
				s = s + ", "
			}
			s = s + flag.name
			comma = true
		}
	}
	s = s + "]"
	return s
}

// NewNodeStateMachine creates a new node state machine.
// If db is not nil then the node states, fields and active timeouts are persisted.
// Persistence can be enabled or disabled for each state flag and field.
func NewNodeStateMachine(db ethdb.KeyValueStore, dbKey []byte, clock mclock.Clock, setup *Setup) *NodeStateMachine {
	if setup.flags == nil {
		panic("No state flags defined")
	}
	if len(setup.flags) > 8*int(unsafe.Sizeof(bitMask(0))) {
		panic("Too many node state flags")
	}
	ns := &NodeStateMachine{
		db:        db,
		dbNodeKey: dbKey,
		clock:     clock,
		setup:     setup,
		nodes:     make(map[enode.ID]*nodeInfo),
		fields:    make([]*fieldInfo, len(setup.fields)),
	}
	stateNameMap := make(map[string]int)
	for index, flag := range setup.flags {
		if _, ok := stateNameMap[flag.name]; ok {
			panic("Node state flag name collision")
		}
		stateNameMap[flag.name] = index
		if flag.persistent {
			ns.saveFlags |= bitMask(1) << uint(index)
		}
	}
	fieldNameMap := make(map[string]int)
	for index, field := range setup.fields {
		if _, ok := fieldNameMap[field.name]; ok {
			panic("Node field name collision")
		}
		ns.fields[index] = &fieldInfo{fieldDefinition: field}
		fieldNameMap[field.name] = index
	}
	return ns
}

// stateMask checks whether the set of flags belongs to the same setup and returns its internal bit mask
func (ns *NodeStateMachine) stateMask(flags Flags) bitMask {
	if flags.setup != ns.setup && flags.mask != 0 {
		panic("Node state flags belong to a different setup")
	}
	return flags.mask
}

// fieldIndex checks whether the field belongs to the same setup and returns its internal index
func (ns *NodeStateMachine) fieldIndex(field Field) int {
	if field.setup != ns.setup {
		panic("Node field belongs to a different setup")
	}
	return field.index
}

// SubscribeState adds a node state subscription. The callback is called while the state
// machine mutex is not held and it is allowed to make further state updates. All immediate
// changes throughout the system are processed in the same thread/goroutine. It is the
// responsibility of the implemented state logic to avoid deadlocks caused by the callbacks,
// infinite toggling of flags or hazardous/non-deterministic state changes.
// State subscriptions should be installed before loading the node database or making the
// first state update.
func (ns *NodeStateMachine) SubscribeState(flags Flags, callback StateCallback) {
	ns.lock.Lock()
	defer ns.lock.Unlock()

	if ns.started {
		panic("state machine already started")
	}
	ns.stateSubs = append(ns.stateSubs, stateSub{ns.stateMask(flags), callback})
}

// SubscribeField adds a node field subscription. Same rules apply as for SubscribeState.
func (ns *NodeStateMachine) SubscribeField(field Field, callback FieldCallback) {
	ns.lock.Lock()
	defer ns.lock.Unlock()

	if ns.started {
		panic("state machine already started")
	}
	f := ns.fields[ns.fieldIndex(field)]
	f.subs = append(f.subs, callback)
}

// newNode creates a new nodeInfo
func (ns *NodeStateMachine) newNode(n *enode.Node) *nodeInfo {
	return &nodeInfo{node: n, fields: make([]interface{}, len(ns.fields))}
}

// checkStarted checks whether the state machine has already been started and panics otherwise.
func (ns *NodeStateMachine) checkStarted() {
	if !ns.started {
		panic("state machine not started yet")
	}
}

// Start starts the state machine, enabling state and field operations and disabling
// further subscriptions.
func (ns *NodeStateMachine) Start() {
	ns.lock.Lock()
	if ns.started {
		panic("state machine already started")
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
	it := ns.db.NewIterator(ns.dbNodeKey, nil)
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

	if enc.Version != ns.setup.Version {
		log.Debug("Removing stored node with unknown version", "current", ns.setup.Version, "stored", enc.Version)
		ns.deleteNode(id)
		return
	}
	if len(enc.Fields) > len(ns.setup.fields) {
		log.Error("Invalid node field count", "id", id, "stored", len(enc.Fields))
		return
	}
	// Resolve persisted node fields
	for i, encField := range enc.Fields {
		if len(encField) == 0 {
			continue
		}
		if decode := ns.fields[i].decode; decode != nil {
			if field, err := decode(encField); err == nil {
				node.fields[i] = field
			} else {
				log.Error("Failed to decode node field", "id", id, "field name", ns.fields[i].name, "error", err)
				return
			}
		} else {
			log.Error("Cannot decode node field", "id", id, "field name", ns.fields[i].name)
			return
		}
	}
	// It's a compatible node record, add it to set.
	ns.nodes[id] = node
	node.state = enc.State
	fields := make([]interface{}, len(node.fields))
	copy(fields, node.fields)
	ns.offlineCallbackList = append(ns.offlineCallbackList, offlineCallback{node.node, node.state, fields})
	log.Debug("Loaded node state", "id", id, "state", Flags{mask: enc.State, setup: ns.setup})
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
		Version: ns.setup.Version,
		State:   storedState,
		Fields:  make([][]byte, len(ns.fields)),
	}
	log.Debug("Saved node state", "id", id, "state", Flags{mask: enc.State, setup: ns.setup})
	lastIndex := -1
	for i, f := range node.fields {
		if f == nil {
			continue
		}
		encode := ns.fields[i].encode
		if encode == nil {
			continue
		}
		blob, err := encode(f)
		if err != nil {
			return err
		}
		enc.Fields[i] = blob
		lastIndex = i
	}
	enc.Fields = enc.Fields[:lastIndex+1]
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

// saveToDb saves the persistent flags and fields of all nodes that have been changed
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

// updateEnode updates the enode entry belonging to the given node if it already exists
func (ns *NodeStateMachine) updateEnode(n *enode.Node) (enode.ID, *nodeInfo) {
	id := n.ID()
	node := ns.nodes[id]
	if node != nil && n.Seq() > node.node.Seq() {
		node.node = n
	}
	return id, node
}

// Persist saves the persistent state and fields of the given node immediately
func (ns *NodeStateMachine) Persist(n *enode.Node) error {
	ns.lock.Lock()
	defer ns.lock.Unlock()

	ns.checkStarted()
	if id, node := ns.updateEnode(n); node != nil && node.dirty {
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
// callbacks) have been processed. If a flag with a timeout is set again, the operation
// removes or replaces the existing timeout.
func (ns *NodeStateMachine) SetState(n *enode.Node, setFlags, resetFlags Flags, timeout time.Duration) {
	ns.lock.Lock()
	ns.checkStarted()
	if ns.stopped {
		ns.lock.Unlock()
		return
	}

	set, reset := ns.stateMask(setFlags), ns.stateMask(resetFlags)
	id, node := ns.updateEnode(n)
	if node == nil {
		if set == 0 {
			ns.lock.Unlock()
			return
		}
		node = ns.newNode(n)
		ns.nodes[id] = node
	}
	oldState := node.state
	newState := (node.state & (^reset)) | set
	changed := oldState ^ newState
	node.state = newState

	// Remove the timeout callbacks for all reset and set flags,
	// even they are not existent(it's noop).
	ns.removeTimeouts(node, set|reset)

	// Register the timeout callback if the new state is not empty
	// and timeout itself is required.
	if timeout != 0 && newState != 0 {
		ns.addTimeout(n, set, timeout)
	}
	if newState == oldState {
		ns.lock.Unlock()
		return
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
			sub.callback(n, Flags{mask: oldState & sub.mask, setup: ns.setup}, Flags{mask: newState & sub.mask, setup: ns.setup})
		}
	}
	if newState == 0 {
		// call field subscriptions for discarded fields
		for i, v := range node.fields {
			if v != nil {
				f := ns.fields[i]
				if len(f.subs) > 0 {
					for _, cb := range f.subs {
						cb(n, Flags{setup: ns.setup}, v, nil)
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
			offState := offlineState & sub.mask
			onState := cb.state & sub.mask
			if offState != onState {
				if start {
					sub.callback(cb.node, Flags{mask: offState, setup: ns.setup}, Flags{mask: onState, setup: ns.setup})
				} else {
					sub.callback(cb.node, Flags{mask: onState, setup: ns.setup}, Flags{mask: offState, setup: ns.setup})
				}
			}
		}
		for i, f := range cb.fields {
			if f != nil && ns.fields[i].subs != nil {
				for _, fsub := range ns.fields[i].subs {
					if start {
						fsub(cb.node, Flags{mask: offlineState, setup: ns.setup}, nil, f)
					} else {
						fsub(cb.node, Flags{mask: offlineState, setup: ns.setup}, f, nil)
					}
				}
			}
		}
	}
	ns.offlineCallbackList = nil
}

// AddTimeout adds a node state timeout associated to the given state flag(s).
// After the specified time interval, the relevant states will be reset.
func (ns *NodeStateMachine) AddTimeout(n *enode.Node, flags Flags, timeout time.Duration) {
	ns.lock.Lock()
	defer ns.lock.Unlock()

	ns.checkStarted()
	if ns.stopped {
		return
	}
	ns.addTimeout(n, ns.stateMask(flags), timeout)
}

// addTimeout adds a node state timeout associated to the given state flag(s).
func (ns *NodeStateMachine) addTimeout(n *enode.Node, mask bitMask, timeout time.Duration) {
	_, node := ns.updateEnode(n)
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
		ns.SetState(n, Flags{}, Flags{mask: t.mask, setup: ns.setup}, 0)
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
func (ns *NodeStateMachine) removeTimeouts(node *nodeInfo, mask bitMask) {
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
func (ns *NodeStateMachine) GetField(n *enode.Node, field Field) interface{} {
	ns.lock.Lock()
	defer ns.lock.Unlock()

	ns.checkStarted()
	if ns.stopped {
		return nil
	}
	if _, node := ns.updateEnode(n); node != nil {
		return node.fields[ns.fieldIndex(field)]
	}
	return nil
}

// SetField sets the given field of the given node
func (ns *NodeStateMachine) SetField(n *enode.Node, field Field, value interface{}) error {
	ns.lock.Lock()
	ns.checkStarted()
	if ns.stopped {
		ns.lock.Unlock()
		return nil
	}
	_, node := ns.updateEnode(n)
	if node == nil {
		ns.lock.Unlock()
		return nil
	}
	fieldIndex := ns.fieldIndex(field)
	f := ns.fields[fieldIndex]
	if value != nil && reflect.TypeOf(value) != f.ftype {
		log.Error("Invalid field type", "type", reflect.TypeOf(value), "required", f.ftype)
		ns.lock.Unlock()
		return errors.New("invalid field type")
	}
	oldValue := node.fields[fieldIndex]
	if value == oldValue {
		ns.lock.Unlock()
		return nil
	}
	node.fields[fieldIndex] = value
	if f.encode != nil {
		node.dirty = true
	}

	state := node.state
	ns.lock.Unlock()
	if len(f.subs) > 0 {
		for _, cb := range f.subs {
			cb(n, Flags{mask: state, setup: ns.setup}, oldValue, value)
		}
	}
	return nil
}

// ForEach calls the callback for each node having all of the required and none of the
// disabled flags set
func (ns *NodeStateMachine) ForEach(requireFlags, disableFlags Flags, cb func(n *enode.Node, state Flags)) {
	ns.lock.Lock()
	ns.checkStarted()
	type callback struct {
		node  *enode.Node
		state bitMask
	}
	require, disable := ns.stateMask(requireFlags), ns.stateMask(disableFlags)
	var callbacks []callback
	for _, node := range ns.nodes {
		if node.state&require == require && node.state&disable == 0 {
			callbacks = append(callbacks, callback{node.node, node.state & (require | disable)})
		}
	}
	ns.lock.Unlock()
	for _, c := range callbacks {
		cb(c.node, Flags{mask: c.state, setup: ns.setup})
	}
}

// GetNode returns the enode currently associated with the given ID
func (ns *NodeStateMachine) GetNode(id enode.ID) *enode.Node {
	ns.lock.Lock()
	defer ns.lock.Unlock()

	ns.checkStarted()
	if node := ns.nodes[id]; node != nil {
		return node.node
	}
	return nil
}

// AddLogMetrics adds logging and/or metrics for nodes entering, exiting and currently
// being in a given set specified by required and disabled state flags
func (ns *NodeStateMachine) AddLogMetrics(requireFlags, disableFlags Flags, name string, inMeter, outMeter metrics.Meter, gauge metrics.Gauge) {
	var count int64
	ns.SubscribeState(requireFlags.Or(disableFlags), func(n *enode.Node, oldState, newState Flags) {
		oldMatch := oldState.HasAll(requireFlags) && oldState.HasNone(disableFlags)
		newMatch := newState.HasAll(requireFlags) && newState.HasNone(disableFlags)
		if newMatch == oldMatch {
			return
		}

		if newMatch {
			count++
			if name != "" {
				log.Debug("Node entered", "set", name, "id", n.ID(), "count", count)
			}
			if inMeter != nil {
				inMeter.Mark(1)
			}
		} else {
			count--
			if name != "" {
				log.Debug("Node left", "set", name, "id", n.ID(), "count", count)
			}
			if outMeter != nil {
				outMeter.Mark(1)
			}
		}
		if gauge != nil {
			gauge.Update(count)
		}
	})
}
