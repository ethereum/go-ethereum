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
	"fmt"
	"math/rand"
	"reflect"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common/mclock"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/p2p/enode"
	"github.com/ethereum/go-ethereum/p2p/enr"
	"github.com/ethereum/go-ethereum/rlp"
)

func indexToMask(index int) NodeStateBitMask {
	return NodeStateBitMask(1) << index
}

func registerTestFlags(ns *NodeStateMachine, n int) []*NodeStateFlag {
	var flags []*NodeStateFlag
	for i := 0; i < n; i++ {
		flag := NewFlag(fmt.Sprintf("flag-%d", i), false, true)
		flags = append(flags, flag)
		ns.RegisterState(flag)
	}
	return flags
}

func registerTestFields(ns *NodeStateMachine, n int, flags []*NodeStateFlag) []*NodeField {
	var fields []*NodeField
	for i := 0; i < n; i++ {
		f := flags[rand.Intn(len(flags))]
		field := NewField(fmt.Sprintf("field-%d", i), reflect.TypeOf(enr.Record{}), []*NodeStateFlag{f}, false, nil, nil)
		fields = append(fields, field)
		ns.RegisterField(field)
	}
	return fields
}

func TestCallback(t *testing.T) {
	mdb, clock := rawdb.NewMemoryDatabase(), &mclock.Simulated{}
	ns := NewNodeStateMachine(mdb, []byte("-ns"), clock)

	// Register order flag 0-2
	f0, _ := ns.RegisterState(NewFlag("flag0", true, true))
	f1, _ := ns.RegisterState(NewFlag("flag1", true, true))
	f2, _ := ns.RegisterState(NewFlag("flag2", true, true))

	set0 := make(chan struct{}, 1)
	set1 := make(chan struct{}, 1)
	set2 := make(chan struct{}, 1)
	ns.SubscribeState(f0, func(id enode.ID, oldState, newState NodeStateBitMask) { set0 <- struct{}{} })
	ns.SubscribeState(f1, func(id enode.ID, oldState, newState NodeStateBitMask) { set1 <- struct{}{} })
	ns.SubscribeState(f2, func(id enode.ID, oldState, newState NodeStateBitMask) { set2 <- struct{}{} })

	ns.Start()

	ns.SetState(enode.ID{0x01}, f0, 0, 0)
	ns.SetState(enode.ID{0x01}, f1, 0, time.Second)
	ns.SetState(enode.ID{0x01}, f2, 0, 2*time.Second)

	for i := 0; i < 3; i++ {
		select {
		case <-set0:
		case <-set1:
		case <-set2:
		case <-time.After(time.Second):
			t.Fatalf("failed to invoke callback")
		}
	}
}

func TestSaveImmediately(t *testing.T) {
	mdb, clock := rawdb.NewMemoryDatabase(), &mclock.Simulated{}
	ns := NewNodeStateMachine(mdb, []byte("-ns"), clock)

	saveNode := make(chan *nodeInfo, 1)
	ns.saveNodeHook = func(node *nodeInfo) {
		saveNode <- node
	}
	// Register order flag 0-2
	flag0 := NewFlag("flag0", true, true)
	flag1 := NewFlag("flag1", true, true)
	flag2 := NewFlag("flag2", true, true)
	f0, _ := ns.RegisterState(flag0)
	f1, _ := ns.RegisterState(flag1)
	f2, _ := ns.RegisterState(flag2)
	fd, _ := ns.RegisterField(NewField("field", reflect.TypeOf(""), []*NodeStateFlag{flag0, flag1, flag2}, true, stringFieldEnc, stringFieldDec))

	ns.Start()

	check := func(expectStatus NodeStateBitMask, expectFields []interface{}) {
		var node *nodeInfo
		select {
		case node = <-saveNode:
		case <-time.After(time.Second):
			t.Fatalf("Timeout")
		}
		if node == nil {
			t.Fatalf("Empty node")
		}
		if node.state != expectStatus {
			t.Fatalf("Status mismatch, want %v, got %v", node.state, expectStatus)
		}
		if !reflect.DeepEqual(node.fields, expectFields) {
			t.Fatalf("Field mismatch, want %v, got %v", expectFields, node.fields)
		}
	}
	// Set status
	ns.SetState(enode.ID{0x01}, f0, 0, 0)
	check(f0, []interface{}{nil})
	ns.SetState(enode.ID{0x01}, f1, 0, 0)
	check(f0|f1, []interface{}{nil})

	// Set fields
	ns.SetField(enode.ID{0x01}, fd, "hello world")
	check(f0|f1, []interface{}{"hello world"})

	ns.SetState(enode.ID{0x01}, f2, 0, 0)
	check(f0|f1|f2, []interface{}{"hello world"})
}

func TestSaveAtShutdown(t *testing.T) {
	mdb, clock := rawdb.NewMemoryDatabase(), &mclock.Simulated{}
	ns := NewNodeStateMachine(mdb, []byte("-ns"), clock)

	saveNode := make(chan *nodeInfo, 2)
	ns.saveNodeHook = func(node *nodeInfo) {
		saveNode <- node
	}
	// Register order flag 0-2
	f0, _ := ns.RegisterState(NewFlag("flag0", false, true))
	f1, _ := ns.RegisterState(NewFlag("flag1", false, true))
	f2, _ := ns.RegisterState(NewFlag("flag2", false, false)) // flag2 shouldn't be saved

	ns.Start()

	ns.SetState(enode.ID{0x01}, f0, 0, time.Second)
	ns.SetState(enode.ID{0x02}, f1, 0, time.Second)
	ns.SetState(enode.ID{0x03}, f2, 0, time.Second)
	ns.Stop()

	for i := 0; i < 2; i++ {
		select {
		case <-saveNode:
		case <-time.After(time.Second):
			t.Fatalf("Timeout")
		}
	}
}

func TestRegistrationProtection(t *testing.T) {
	mdb, clock := rawdb.NewMemoryDatabase(), &mclock.Simulated{}
	ns := NewNodeStateMachine(mdb, []byte("-ns"), clock)
	flags := registerTestFlags(ns, 60)
	fields := registerTestFields(ns, 30, flags)

	// Before initialization, register flags
	var cases = []struct {
		flag      *NodeStateFlag
		mask      NodeStateBitMask
		expectErr error
	}{
		{flags[0], indexToMask(1), nil},
		{flags[59], indexToMask(60), nil},
		{NewFlag("flag-0", false, true), 0, errNameCollision},
		{NewFlag("flag-59", false, true), 0, errNameCollision},
		{NewFlag("flag-60", false, true), indexToMask(61), nil},
	}
	for id, c := range cases {
		mask, err := ns.RegisterState(c.flag)
		if c.expectErr != nil {
			if err == nil {
				t.Fatalf("Expect error => case (%d) %v", id, c.expectErr)
			}
			if err != c.expectErr {
				t.Fatalf("Error mismatch => case (%d), want %v, got %v", id, c.expectErr, err)
			}
			continue
		}
		if mask != c.mask {
			t.Fatalf("Mask mismatch => case (%d), want %v, got %v", id, c.mask, mask)
		}
	}
	// Before initialization, register fields
	var cases2 = []struct {
		field     *NodeField
		fieldId   int
		expectErr error
	}{
		{fields[0], 0, nil},
		{fields[29], 29, nil},
		{NewField("field-0", reflect.TypeOf(enr.Record{}), nil, false, nil, nil), 0, errNameCollision},
		{NewField("field-29", reflect.TypeOf(enr.Record{}), nil, false, nil, nil), 0, errNameCollision},
		{NewField("field-30", reflect.TypeOf(enr.Record{}), nil, false, nil, nil), 30, nil},
	}
	for id, c := range cases2 {
		index, err := ns.RegisterField(c.field)
		if c.expectErr != nil {
			if err == nil {
				t.Fatalf("Expect error => case (%d) %v", id, c.expectErr)
			}
			if err != c.expectErr {
				t.Fatalf("Error mismatch => case (%d), want %v, got %v", id, c.expectErr, err)
			}
			continue
		}
		if index != c.fieldId {
			t.Fatalf("Field id mismatch => case (%d), want %v, got %v", id, c.fieldId, index)
		}
	}

	ns.Start()

	ns.SetState(enode.ID{0x1}, indexToMask(1), 0, 0)
	_, err := ns.RegisterState(NewFlag("flag-61", false, true))
	if err != errAlreadyStarted {
		t.Fatalf("Expect already init error")
	}
	_, err = ns.RegisterField(NewField("field-31", reflect.TypeOf(enr.Record{}), nil, false, nil, nil))
	if err != errAlreadyStarted {
		t.Fatalf("Expect already init error")
	}
	err = ns.SubscribeState(indexToMask(1), nil)
	if err != errAlreadyStarted {
		t.Fatalf("Expect already init error")
	}
}

func TestSetField(t *testing.T) {
	mdb, clock := rawdb.NewMemoryDatabase(), &mclock.Simulated{}
	ns := NewNodeStateMachine(mdb, []byte("-ns"), clock)

	saveNode := make(chan *nodeInfo, 1)
	ns.saveNodeHook = func(node *nodeInfo) {
		saveNode <- node
	}
	flag := NewFlag("flag", false, true)
	f, _ := ns.RegisterState(flag)
	fd, _ := ns.RegisterField(NewField("field", reflect.TypeOf(""), []*NodeStateFlag{flag}, false, nil, nil))

	ns.Start()

	// Set field before setting state
	ns.SetField(enode.ID{0x01}, fd, "hello world")
	field := ns.GetField(enode.ID{0x01}, fd)
	if field != nil {
		t.Fatalf("Field shouldn't be set before setting states")
	}
	// Set field after setting state
	ns.SetState(enode.ID{0x01}, f, 0, time.Second)
	ns.SetField(enode.ID{0x01}, fd, "hello world")
	field = ns.GetField(enode.ID{0x01}, fd)
	if field == nil {
		t.Fatalf("Field should be set after setting states")
	}
	if err := ns.SetField(enode.ID{0x01}, fd, 123); err != errInvalidField {
		t.Fatalf("Invalid field should be rejected")
	}
	// Dirty node should be written back
	ns.Stop()
	select {
	case <-saveNode:
	case <-time.After(time.Second):
		t.Fatalf("Timeout")
	}
}

func TestUnsetField(t *testing.T) {
	mdb, clock := rawdb.NewMemoryDatabase(), &mclock.Simulated{}
	ns := NewNodeStateMachine(mdb, []byte("-ns"), clock)

	flag := NewFlag("flag", false, true)
	f, _ := ns.RegisterState(flag)
	fd, _ := ns.RegisterField(NewField("field", reflect.TypeOf(""), []*NodeStateFlag{flag}, false, nil, nil))

	ns.Start()

	ns.SetState(enode.ID{0x01}, f, 0, time.Second)
	ns.SetField(enode.ID{0x01}, fd, "hello world")

	ns.SetState(enode.ID{0x01}, 0, f, 0)
	if field := ns.GetField(enode.ID{0x01}, fd); field != nil {
		t.Fatalf("Field should be unset")
	}
}

func TestSetState(t *testing.T) {
	mdb, clock := rawdb.NewMemoryDatabase(), &mclock.Simulated{}
	ns := NewNodeStateMachine(mdb, []byte("-ns"), clock)

	f0, _ := ns.RegisterState(NewFlag("flag0", false, false))
	f1, _ := ns.RegisterState(NewFlag("flag1", false, false))
	f2, _ := ns.RegisterState(NewFlag("flag2", false, false))

	type change struct{ old, new NodeStateBitMask }
	set := make(chan change, 1)
	ns.SubscribeState(f0|f1, func(id enode.ID, oldState, newState NodeStateBitMask) {
		set <- change{
			old: oldState,
			new: newState,
		}
	})

	ns.Start()

	check := func(expectOld, expectNew NodeStateBitMask, expectChange bool) {
		if expectChange {
			select {
			case c := <-set:
				if c.old != expectOld {
					t.Fatalf("Old state mismatch")
				}
				if c.new != expectNew {
					t.Fatalf("New state mismatch")
				}
			case <-time.After(time.Second):
			}
			return
		}
		select {
		case <-set:
			t.Fatalf("Unexpected change")
		case <-time.After(time.Millisecond * 100):
			return
		}
	}
	ns.SetState(enode.ID{0x01}, f0, 0, 0)
	check(0, f0, true)

	ns.SetState(enode.ID{0x01}, f1, 0, 0)
	check(f0, f0|f1, true)

	ns.SetState(enode.ID{0x01}, f2, 0, 0)
	check(0, 0, false)

	ns.SetState(enode.ID{0x01}, 0, f0, 0)
	check(f0|f1, f1, true)

	ns.SetState(enode.ID{0x01}, 0, f1, 0)
	check(f1, 0, true)

	ns.SetState(enode.ID{0x01}, 0, f2, 0)
	check(0, 0, false)

	ns.SetState(enode.ID{0x01}, f0|f1, 0, time.Second)
	check(0, f0|f1, true)
	clock.Run(time.Second)
	check(f0|f1, 0, true)
}

func uint64FieldEnc(field interface{}) ([]byte, error) {
	if u, ok := field.(uint64); ok {
		enc, err := rlp.EncodeToBytes(&u)
		return enc, err
	} else {
		return nil, errInvalidField
	}
}

func uint64FieldDec(enc []byte) (interface{}, error) {
	var u uint64
	err := rlp.DecodeBytes(enc, &u)
	return u, err
}

func stringFieldEnc(field interface{}) ([]byte, error) {
	if s, ok := field.(string); ok {
		return []byte(s), nil
	} else {
		return nil, errInvalidField
	}
}

func stringFieldDec(enc []byte) (interface{}, error) {
	return string(enc), nil
}

func TestPersistent(t *testing.T) {
	mdb, clock := rawdb.NewMemoryDatabase(), &mclock.Simulated{}
	ns := NewNodeStateMachine(mdb, []byte("-ns"), clock)

	f0 := NewFlag("flag0", false, true)
	f1 := NewFlag("flag1", false, true)
	s0, _ := ns.RegisterState(f0)
	s1, _ := ns.RegisterState(f1)
	fd0, _ := ns.RegisterField(NewField("field0", reflect.TypeOf(uint64(0)), []*NodeStateFlag{f0}, false, uint64FieldEnc, uint64FieldDec))
	fd1, _ := ns.RegisterField(NewField("field1", reflect.TypeOf(""), []*NodeStateFlag{f1}, false, stringFieldEnc, stringFieldDec))
	ns.Start()
	ns.SetState(enode.ID{0x01}, s0, 0, time.Second)
	ns.SetState(enode.ID{0x01}, s1, 0, time.Second)
	ns.SetField(enode.ID{0x01}, fd0, uint64(100))
	ns.SetField(enode.ID{0x01}, fd1, "hello world")
	ns.Stop()

	ns2 := NewNodeStateMachine(mdb, []byte("-ns"), clock)
	ns2.RegisterState(f0)
	ns2.RegisterState(f1)
	fd0, _ = ns2.RegisterField(NewField("field0", reflect.TypeOf(uint64(0)), []*NodeStateFlag{f0}, false, uint64FieldEnc, uint64FieldDec))
	fd1, _ = ns2.RegisterField(NewField("field1", reflect.TypeOf(""), []*NodeStateFlag{f1}, false, stringFieldEnc, stringFieldDec))
	ns2.Start()
	field0 := ns2.GetField(enode.ID{0x01}, fd0)
	if !reflect.DeepEqual(field0, uint64(100)) {
		t.Fatalf("Field changed")
	}
	field1 := ns2.GetField(enode.ID{0x01}, fd1)
	if !reflect.DeepEqual(field1, "hello world") {
		t.Fatalf("Field changed")
	}

	ns3 := NewNodeStateMachine(mdb, []byte("-ns"), clock)
	// Different order
	ns3.RegisterState(f1)
	ns3.RegisterState(f0)
	fd1, _ = ns3.RegisterField(NewField("field1", reflect.TypeOf(""), []*NodeStateFlag{f1}, false, stringFieldEnc, stringFieldDec))
	fd0, _ = ns3.RegisterField(NewField("field0", reflect.TypeOf(uint64(0)), []*NodeStateFlag{f0}, false, uint64FieldEnc, uint64FieldDec))
	// additional registeration
	ns3.RegisterState(NewFlag("flag2", false, true))
	ns3.RegisterField(NewField("field2", reflect.TypeOf(uint32(0)), []*NodeStateFlag{f0}, false, nil, nil))

	ns3.Start()
	field0 = ns3.GetField(enode.ID{0x01}, fd0)
	if !reflect.DeepEqual(field0, uint64(100)) {
		t.Fatalf("Field changed")
	}
	field1 = ns3.GetField(enode.ID{0x01}, fd1)
	if !reflect.DeepEqual(field1, "hello world") {
		t.Fatalf("Field changed")
	}
}

func TestFieldSub(t *testing.T) {
	mdb, clock := rawdb.NewMemoryDatabase(), &mclock.Simulated{}
	ns := NewNodeStateMachine(mdb, []byte("-ns"), clock)

	f0 := NewFlag("flag0", false, true)
	field0 := NewField("field0", reflect.TypeOf(uint64(0)), []*NodeStateFlag{f0}, false, uint64FieldEnc, uint64FieldDec)
	s0, _ := ns.RegisterState(f0)
	fd0, _ := ns.RegisterField(field0)
	var (
		lastState                  NodeStateBitMask
		lastOldValue, lastNewValue interface{}
	)
	ns.SubscribeField(fd0, func(id enode.ID, state NodeStateBitMask, oldValue, newValue interface{}) {
		lastState, lastOldValue, lastNewValue = state, oldValue, newValue
	})
	check := func(state NodeStateBitMask, oldValue, newValue interface{}) {
		if lastState != state || lastOldValue != oldValue || lastNewValue != newValue {
			t.Fatalf("Incorrect field sub callback (expected [%v %v %v], got [%v %v %v])", state, oldValue, newValue, lastState, lastOldValue, lastNewValue)
		}
	}
	ns.Start()
	ns.SetState(enode.ID{0x01}, s0, 0, 0)
	ns.SetField(enode.ID{0x01}, fd0, uint64(100))
	check(s0, nil, uint64(100))
	ns.Stop()
	check(OfflineState, uint64(100), nil)

	ns2 := NewNodeStateMachine(mdb, []byte("-ns"), clock)
	s0, _ = ns2.RegisterState(f0)
	fd0, _ = ns2.RegisterField(field0)
	ns2.SubscribeField(fd0, func(id enode.ID, state NodeStateBitMask, oldValue, newValue interface{}) {
		lastState, lastOldValue, lastNewValue = state, oldValue, newValue
	})
	ns2.Start()
	check(OfflineState, nil, uint64(100))
	ns2.SetState(enode.ID{0x01}, 0, s0, 0)
	check(0, uint64(100), nil)
	ns2.Stop()
}
