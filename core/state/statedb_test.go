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

package state

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"maps"
	"math"
	"math/rand"
	"reflect"
	"slices"
	"strings"
	"sync"
	"testing"
	"testing/quick"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/core/state/snapshot"
	"github.com/ethereum/go-ethereum/core/tracing"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/ethereum/go-ethereum/trie"
	"github.com/ethereum/go-ethereum/trie/trienode"
	"github.com/ethereum/go-ethereum/triedb"
	"github.com/ethereum/go-ethereum/triedb/hashdb"
	"github.com/ethereum/go-ethereum/triedb/pathdb"
	"github.com/holiman/uint256"
)

// Tests that updating a state trie does not leak any database writes prior to
// actually committing the state.
func TestUpdateLeaks(t *testing.T) {
	// Create an empty state database
	var (
		db  = rawdb.NewMemoryDatabase()
		tdb = triedb.NewDatabase(db, nil)
		sdb = NewDatabase(tdb, nil)
	)
	state, _ := New(types.EmptyRootHash, sdb)

	// Update it with some accounts
	for i := byte(0); i < 255; i++ {
		addr := common.BytesToAddress([]byte{i})
		state.AddBalance(addr, uint256.NewInt(uint64(11*i)), tracing.BalanceChangeUnspecified)
		state.SetNonce(addr, uint64(42*i))
		if i%2 == 0 {
			state.SetState(addr, common.BytesToHash([]byte{i, i, i}), common.BytesToHash([]byte{i, i, i, i}))
		}
		if i%3 == 0 {
			state.SetCode(addr, []byte{i, i, i, i, i})
		}
	}

	root := state.IntermediateRoot(false)
	if err := tdb.Commit(root, false); err != nil {
		t.Errorf("can not commit trie %v to persistent database", root.Hex())
	}

	// Ensure that no data was leaked into the database
	it := db.NewIterator(nil, nil)
	for it.Next() {
		t.Errorf("State leaked into database: %x -> %x", it.Key(), it.Value())
	}
	it.Release()
}

// Tests that no intermediate state of an object is stored into the database,
// only the one right before the commit.
func TestIntermediateLeaks(t *testing.T) {
	// Create two state databases, one transitioning to the final state, the other final from the beginning
	transDb := rawdb.NewMemoryDatabase()
	finalDb := rawdb.NewMemoryDatabase()
	transNdb := triedb.NewDatabase(transDb, nil)
	finalNdb := triedb.NewDatabase(finalDb, nil)
	transState, _ := New(types.EmptyRootHash, NewDatabase(transNdb, nil))
	finalState, _ := New(types.EmptyRootHash, NewDatabase(finalNdb, nil))

	modify := func(state *StateDB, addr common.Address, i, tweak byte) {
		state.SetBalance(addr, uint256.NewInt(uint64(11*i)+uint64(tweak)), tracing.BalanceChangeUnspecified)
		state.SetNonce(addr, uint64(42*i+tweak))
		if i%2 == 0 {
			state.SetState(addr, common.Hash{i, i, i, 0}, common.Hash{})
			state.SetState(addr, common.Hash{i, i, i, tweak}, common.Hash{i, i, i, i, tweak})
		}
		if i%3 == 0 {
			state.SetCode(addr, []byte{i, i, i, i, i, tweak})
		}
	}

	// Modify the transient state.
	for i := byte(0); i < 255; i++ {
		modify(transState, common.Address{i}, i, 0)
	}
	// Write modifications to trie.
	transState.IntermediateRoot(false)

	// Overwrite all the data with new values in the transient database.
	for i := byte(0); i < 255; i++ {
		modify(transState, common.Address{i}, i, 99)
		modify(finalState, common.Address{i}, i, 99)
	}

	// Commit and cross check the databases.
	transRoot, err := transState.Commit(0, false)
	if err != nil {
		t.Fatalf("failed to commit transition state: %v", err)
	}
	if err = transNdb.Commit(transRoot, false); err != nil {
		t.Errorf("can not commit trie %v to persistent database", transRoot.Hex())
	}

	finalRoot, err := finalState.Commit(0, false)
	if err != nil {
		t.Fatalf("failed to commit final state: %v", err)
	}
	if err = finalNdb.Commit(finalRoot, false); err != nil {
		t.Errorf("can not commit trie %v to persistent database", finalRoot.Hex())
	}

	it := finalDb.NewIterator(nil, nil)
	for it.Next() {
		key, fvalue := it.Key(), it.Value()
		tvalue, err := transDb.Get(key)
		if err != nil {
			t.Errorf("entry missing from the transition database: %x -> %x", key, fvalue)
		}
		if !bytes.Equal(fvalue, tvalue) {
			t.Errorf("value mismatch at key %x: %x in transition database, %x in final database", key, tvalue, fvalue)
		}
	}
	it.Release()

	it = transDb.NewIterator(nil, nil)
	for it.Next() {
		key, tvalue := it.Key(), it.Value()
		fvalue, err := finalDb.Get(key)
		if err != nil {
			t.Errorf("extra entry in the transition database: %x -> %x", key, it.Value())
		}
		if !bytes.Equal(fvalue, tvalue) {
			t.Errorf("value mismatch at key %x: %x in transition database, %x in final database", key, tvalue, fvalue)
		}
	}
}

// TestCopy tests that copying a StateDB object indeed makes the original and
// the copy independent of each other. This test is a regression test against
// https://github.com/ethereum/go-ethereum/pull/15549.
func TestCopy(t *testing.T) {
	// Create a random state test to copy and modify "independently"
	orig, _ := New(types.EmptyRootHash, NewDatabaseForTesting())

	for i := byte(0); i < 255; i++ {
		obj := orig.getOrNewStateObject(common.BytesToAddress([]byte{i}))
		obj.AddBalance(uint256.NewInt(uint64(i)))
		orig.updateStateObject(obj)
	}
	orig.Finalise(false)

	// Copy the state
	copy := orig.Copy()

	// Copy the copy state
	ccopy := copy.Copy()

	// modify all in memory
	for i := byte(0); i < 255; i++ {
		origObj := orig.getOrNewStateObject(common.BytesToAddress([]byte{i}))
		copyObj := copy.getOrNewStateObject(common.BytesToAddress([]byte{i}))
		ccopyObj := ccopy.getOrNewStateObject(common.BytesToAddress([]byte{i}))

		origObj.AddBalance(uint256.NewInt(2 * uint64(i)))
		copyObj.AddBalance(uint256.NewInt(3 * uint64(i)))
		ccopyObj.AddBalance(uint256.NewInt(4 * uint64(i)))

		orig.updateStateObject(origObj)
		copy.updateStateObject(copyObj)
		ccopy.updateStateObject(copyObj)
	}

	// Finalise the changes on all concurrently
	finalise := func(wg *sync.WaitGroup, db *StateDB) {
		defer wg.Done()
		db.Finalise(true)
	}

	var wg sync.WaitGroup
	wg.Add(3)
	go finalise(&wg, orig)
	go finalise(&wg, copy)
	go finalise(&wg, ccopy)
	wg.Wait()

	// Verify that the three states have been updated independently
	for i := byte(0); i < 255; i++ {
		origObj := orig.getOrNewStateObject(common.BytesToAddress([]byte{i}))
		copyObj := copy.getOrNewStateObject(common.BytesToAddress([]byte{i}))
		ccopyObj := ccopy.getOrNewStateObject(common.BytesToAddress([]byte{i}))

		if want := uint256.NewInt(3 * uint64(i)); origObj.Balance().Cmp(want) != 0 {
			t.Errorf("orig obj %d: balance mismatch: have %v, want %v", i, origObj.Balance(), want)
		}
		if want := uint256.NewInt(4 * uint64(i)); copyObj.Balance().Cmp(want) != 0 {
			t.Errorf("copy obj %d: balance mismatch: have %v, want %v", i, copyObj.Balance(), want)
		}
		if want := uint256.NewInt(5 * uint64(i)); ccopyObj.Balance().Cmp(want) != 0 {
			t.Errorf("copy obj %d: balance mismatch: have %v, want %v", i, ccopyObj.Balance(), want)
		}
	}
}

// TestCopyWithDirtyJournal tests if Copy can correct create a equal copied
// stateDB with dirty journal present.
func TestCopyWithDirtyJournal(t *testing.T) {
	db := NewDatabaseForTesting()
	orig, _ := New(types.EmptyRootHash, db)

	// Fill up the initial states
	for i := byte(0); i < 255; i++ {
		obj := orig.getOrNewStateObject(common.BytesToAddress([]byte{i}))
		obj.AddBalance(uint256.NewInt(uint64(i)))
		obj.data.Root = common.HexToHash("0xdeadbeef")
		orig.updateStateObject(obj)
	}
	root, _ := orig.Commit(0, true)
	orig, _ = New(root, db)

	// modify all in memory without finalizing
	for i := byte(0); i < 255; i++ {
		obj := orig.getOrNewStateObject(common.BytesToAddress([]byte{i}))
		amount := uint256.NewInt(uint64(i))
		obj.SetBalance(new(uint256.Int).Sub(obj.Balance(), amount))

		orig.updateStateObject(obj)
	}
	cpy := orig.Copy()

	orig.Finalise(true)
	for i := byte(0); i < 255; i++ {
		root := orig.GetStorageRoot(common.BytesToAddress([]byte{i}))
		if root != (common.Hash{}) {
			t.Errorf("Unexpected storage root %x", root)
		}
	}
	cpy.Finalise(true)
	for i := byte(0); i < 255; i++ {
		root := cpy.GetStorageRoot(common.BytesToAddress([]byte{i}))
		if root != (common.Hash{}) {
			t.Errorf("Unexpected storage root %x", root)
		}
	}
	if cpy.IntermediateRoot(true) != orig.IntermediateRoot(true) {
		t.Error("State is not equal after copy")
	}
}

// TestCopyObjectState creates an original state, S1, and makes a copy S2.
// It then proceeds to make changes to S1. Those changes are _not_ supposed
// to affect S2. This test checks that the copy properly deep-copies the objectstate
func TestCopyObjectState(t *testing.T) {
	db := NewDatabaseForTesting()
	orig, _ := New(types.EmptyRootHash, db)

	// Fill up the initial states
	for i := byte(0); i < 5; i++ {
		obj := orig.getOrNewStateObject(common.BytesToAddress([]byte{i}))
		obj.AddBalance(uint256.NewInt(uint64(i)))
		obj.data.Root = common.HexToHash("0xdeadbeef")
		orig.updateStateObject(obj)
	}
	orig.Finalise(true)
	cpy := orig.Copy()
	for _, op := range cpy.mutations {
		if have, want := op.applied, false; have != want {
			t.Fatalf("Error in test itself, the 'done' flag should not be set before Commit, have %v want %v", have, want)
		}
	}
	orig.Commit(0, true)
	for _, op := range cpy.mutations {
		if have, want := op.applied, false; have != want {
			t.Fatalf("Error: original state affected copy, have %v want %v", have, want)
		}
	}
}

func TestSnapshotRandom(t *testing.T) {
	config := &quick.Config{MaxCount: 1000}
	err := quick.Check((*snapshotTest).run, config)
	if cerr, ok := err.(*quick.CheckError); ok {
		test := cerr.In[0].(*snapshotTest)
		t.Errorf("%v:\n%s", test.err, test)
	} else if err != nil {
		t.Error(err)
	}
}

// A snapshotTest checks that reverting StateDB snapshots properly undoes all changes
// captured by the snapshot. Instances of this test with pseudorandom content are created
// by Generate.
//
// The test works as follows:
//
// A new state is created and all actions are applied to it. Several snapshots are taken
// in between actions. The test then reverts each snapshot. For each snapshot the actions
// leading up to it are replayed on a fresh, empty state. The behaviour of all public
// accessor methods on the reverted state must match the return value of the equivalent
// methods on the replayed state.
type snapshotTest struct {
	addrs     []common.Address // all account addresses
	actions   []testAction     // modifications to the state
	snapshots []int            // actions indexes at which snapshot is taken
	err       error            // failure details are reported through this field
}

type testAction struct {
	name   string
	fn     func(testAction, *StateDB)
	args   []int64
	noAddr bool
}

// newTestAction creates a random action that changes state.
func newTestAction(addr common.Address, r *rand.Rand) testAction {
	actions := []testAction{
		{
			name: "SetBalance",
			fn: func(a testAction, s *StateDB) {
				s.SetBalance(addr, uint256.NewInt(uint64(a.args[0])), tracing.BalanceChangeUnspecified)
			},
			args: make([]int64, 1),
		},
		{
			name: "AddBalance",
			fn: func(a testAction, s *StateDB) {
				s.AddBalance(addr, uint256.NewInt(uint64(a.args[0])), tracing.BalanceChangeUnspecified)
			},
			args: make([]int64, 1),
		},
		{
			name: "SetNonce",
			fn: func(a testAction, s *StateDB) {
				s.SetNonce(addr, uint64(a.args[0]))
			},
			args: make([]int64, 1),
		},
		{
			name: "SetStorage",
			fn: func(a testAction, s *StateDB) {
				var key, val common.Hash
				binary.BigEndian.PutUint16(key[:], uint16(a.args[0]))
				binary.BigEndian.PutUint16(val[:], uint16(a.args[1]))
				s.SetState(addr, key, val)
			},
			args: make([]int64, 2),
		},
		{
			name: "SetCode",
			fn: func(a testAction, s *StateDB) {
				// SetCode can only be performed in case the addr does
				// not already hold code
				if c := s.GetCode(addr); len(c) > 0 {
					// no-op
					return
				}
				code := make([]byte, 16)
				binary.BigEndian.PutUint64(code, uint64(a.args[0]))
				binary.BigEndian.PutUint64(code[8:], uint64(a.args[1]))
				s.SetCode(addr, code)
			},
			args: make([]int64, 2),
		},
		{
			name: "CreateAccount",
			fn: func(a testAction, s *StateDB) {
				if !s.Exist(addr) {
					s.CreateAccount(addr)
				}
			},
		},
		{
			name: "CreateContract",
			fn: func(a testAction, s *StateDB) {
				if !s.Exist(addr) {
					s.CreateAccount(addr)
				}
				contractHash := s.GetCodeHash(addr)
				emptyCode := contractHash == (common.Hash{}) || contractHash == types.EmptyCodeHash
				storageRoot := s.GetStorageRoot(addr)
				emptyStorage := storageRoot == (common.Hash{}) || storageRoot == types.EmptyRootHash
				if s.GetNonce(addr) == 0 && emptyCode && emptyStorage {
					s.CreateContract(addr)
					// We also set some code here, to prevent the
					// CreateContract action from being performed twice in a row,
					// which would cause a difference in state when unrolling
					// the journal. (CreateContact assumes created was false prior to
					// invocation, and the journal rollback sets it to false).
					s.SetCode(addr, []byte{1})
				}
			},
		},
		{
			name: "SelfDestruct",
			fn: func(a testAction, s *StateDB) {
				s.SelfDestruct(addr)
			},
		},
		{
			name: "AddRefund",
			fn: func(a testAction, s *StateDB) {
				s.AddRefund(uint64(a.args[0]))
			},
			args:   make([]int64, 1),
			noAddr: true,
		},
		{
			name: "AddLog",
			fn: func(a testAction, s *StateDB) {
				data := make([]byte, 2)
				binary.BigEndian.PutUint16(data, uint16(a.args[0]))
				s.AddLog(&types.Log{Address: addr, Data: data})
			},
			args: make([]int64, 1),
		},
		{
			name: "AddPreimage",
			fn: func(a testAction, s *StateDB) {
				preimage := []byte{1}
				hash := common.BytesToHash(preimage)
				s.AddPreimage(hash, preimage)
			},
			args: make([]int64, 1),
		},
		{
			name: "AddAddressToAccessList",
			fn: func(a testAction, s *StateDB) {
				s.AddAddressToAccessList(addr)
			},
		},
		{
			name: "AddSlotToAccessList",
			fn: func(a testAction, s *StateDB) {
				s.AddSlotToAccessList(addr,
					common.Hash{byte(a.args[0])})
			},
			args: make([]int64, 1),
		},
		{
			name: "SetTransientState",
			fn: func(a testAction, s *StateDB) {
				var key, val common.Hash
				binary.BigEndian.PutUint16(key[:], uint16(a.args[0]))
				binary.BigEndian.PutUint16(val[:], uint16(a.args[1]))
				s.SetTransientState(addr, key, val)
			},
			args: make([]int64, 2),
		},
	}
	action := actions[r.Intn(len(actions))]
	var nameargs []string
	if !action.noAddr {
		nameargs = append(nameargs, addr.Hex())
	}
	for i := range action.args {
		action.args[i] = rand.Int63n(100)
		nameargs = append(nameargs, fmt.Sprint(action.args[i]))
	}
	action.name += strings.Join(nameargs, ", ")
	return action
}

// Generate returns a new snapshot test of the given size. All randomness is
// derived from r.
func (*snapshotTest) Generate(r *rand.Rand, size int) reflect.Value {
	// Generate random actions.
	addrs := make([]common.Address, 50)
	for i := range addrs {
		addrs[i][0] = byte(i)
	}
	actions := make([]testAction, size)
	for i := range actions {
		addr := addrs[r.Intn(len(addrs))]
		actions[i] = newTestAction(addr, r)
	}
	// Generate snapshot indexes.
	nsnapshots := int(math.Sqrt(float64(size)))
	if size > 0 && nsnapshots == 0 {
		nsnapshots = 1
	}
	snapshots := make([]int, nsnapshots)
	snaplen := len(actions) / nsnapshots
	for i := range snapshots {
		// Try to place the snapshots some number of actions apart from each other.
		snapshots[i] = (i * snaplen) + r.Intn(snaplen)
	}
	return reflect.ValueOf(&snapshotTest{addrs, actions, snapshots, nil})
}

func (test *snapshotTest) String() string {
	out := new(bytes.Buffer)
	sindex := 0
	for i, action := range test.actions {
		if len(test.snapshots) > sindex && i == test.snapshots[sindex] {
			fmt.Fprintf(out, "---- snapshot %d ----\n", sindex)
			sindex++
		}
		fmt.Fprintf(out, "%4d: %s\n", i, action.name)
	}
	return out.String()
}

func (test *snapshotTest) run() bool {
	// Run all actions and create snapshots.
	var (
		state, _     = New(types.EmptyRootHash, NewDatabaseForTesting())
		snapshotRevs = make([]int, len(test.snapshots))
		sindex       = 0
		checkstates  = make([]*StateDB, len(test.snapshots))
	)
	for i, action := range test.actions {
		if len(test.snapshots) > sindex && i == test.snapshots[sindex] {
			snapshotRevs[sindex] = state.Snapshot()
			checkstates[sindex] = state.Copy()
			sindex++
		}
		action.fn(action, state)
	}
	// Revert all snapshots in reverse order. Each revert must yield a state
	// that is equivalent to fresh state with all actions up the snapshot applied.
	for sindex--; sindex >= 0; sindex-- {
		state.RevertToSnapshot(snapshotRevs[sindex])
		if err := test.checkEqual(state, checkstates[sindex]); err != nil {
			test.err = fmt.Errorf("state mismatch after revert to snapshot %d\n%v", sindex, err)
			return false
		}
	}
	return true
}

func forEachStorage(s *StateDB, addr common.Address, cb func(key, value common.Hash) bool) error {
	so := s.getStateObject(addr)
	if so == nil {
		return nil
	}
	tr, err := so.getTrie()
	if err != nil {
		return err
	}
	trieIt, err := tr.NodeIterator(nil)
	if err != nil {
		return err
	}
	var (
		it      = trie.NewIterator(trieIt)
		visited = make(map[common.Hash]bool)
	)

	for it.Next() {
		key := common.BytesToHash(s.trie.GetKey(it.Key))
		visited[key] = true
		if value, dirty := so.dirtyStorage[key]; dirty {
			if !cb(key, value) {
				return nil
			}
			continue
		}

		if len(it.Value) > 0 {
			_, content, _, err := rlp.Split(it.Value)
			if err != nil {
				return err
			}
			if !cb(key, common.BytesToHash(content)) {
				return nil
			}
		}
	}
	return nil
}

// checkEqual checks that methods of state and checkstate return the same values.
func (test *snapshotTest) checkEqual(state, checkstate *StateDB) error {
	for _, addr := range test.addrs {
		var err error
		checkeq := func(op string, a, b interface{}) bool {
			if err == nil && !reflect.DeepEqual(a, b) {
				err = fmt.Errorf("got %s(%s) == %v, want %v", op, addr.Hex(), a, b)
				return false
			}
			return true
		}
		// Check basic accessor methods.
		checkeq("Exist", state.Exist(addr), checkstate.Exist(addr))
		checkeq("HasSelfdestructed", state.HasSelfDestructed(addr), checkstate.HasSelfDestructed(addr))
		checkeq("GetBalance", state.GetBalance(addr), checkstate.GetBalance(addr))
		checkeq("GetNonce", state.GetNonce(addr), checkstate.GetNonce(addr))
		checkeq("GetCode", state.GetCode(addr), checkstate.GetCode(addr))
		checkeq("GetCodeHash", state.GetCodeHash(addr), checkstate.GetCodeHash(addr))
		checkeq("GetCodeSize", state.GetCodeSize(addr), checkstate.GetCodeSize(addr))
		// Check newContract-flag
		if obj := state.getStateObject(addr); obj != nil {
			checkeq("IsNewContract", obj.newContract, checkstate.getStateObject(addr).newContract)
		}
		// Check storage.
		if obj := state.getStateObject(addr); obj != nil {
			forEachStorage(state, addr, func(key, value common.Hash) bool {
				return checkeq("GetState("+key.Hex()+")", checkstate.GetState(addr, key), value)
			})
			forEachStorage(checkstate, addr, func(key, value common.Hash) bool {
				return checkeq("GetState("+key.Hex()+")", checkstate.GetState(addr, key), value)
			})
			other := checkstate.getStateObject(addr)
			// Check dirty storage which is not in trie
			if !maps.Equal(obj.dirtyStorage, other.dirtyStorage) {
				print := func(dirty map[common.Hash]common.Hash) string {
					var keys []common.Hash
					out := new(strings.Builder)
					for key := range dirty {
						keys = append(keys, key)
					}
					slices.SortFunc(keys, common.Hash.Cmp)
					for i, key := range keys {
						fmt.Fprintf(out, "  %d. %v %v\n", i, key, dirty[key])
					}
					return out.String()
				}
				return fmt.Errorf("dirty storage err, have\n%v\nwant\n%v",
					print(obj.dirtyStorage),
					print(other.dirtyStorage))
			}
		}
		// Check transient storage.
		{
			have := state.transientStorage
			want := checkstate.transientStorage
			eq := maps.EqualFunc(have, want,
				func(a Storage, b Storage) bool {
					return maps.Equal(a, b)
				})
			if !eq {
				return fmt.Errorf("transient storage differs ,have\n%v\nwant\n%v",
					have.PrettyPrint(),
					want.PrettyPrint())
			}
		}
		if err != nil {
			return err
		}
	}
	if !checkstate.accessList.Equal(state.accessList) { // Check access lists
		return fmt.Errorf("AccessLists are wrong, have \n%v\nwant\n%v",
			checkstate.accessList.PrettyPrint(),
			state.accessList.PrettyPrint())
	}
	if state.GetRefund() != checkstate.GetRefund() {
		return fmt.Errorf("got GetRefund() == %d, want GetRefund() == %d",
			state.GetRefund(), checkstate.GetRefund())
	}
	if !reflect.DeepEqual(state.GetLogs(common.Hash{}, 0, common.Hash{}), checkstate.GetLogs(common.Hash{}, 0, common.Hash{})) {
		return fmt.Errorf("got GetLogs(common.Hash{}) == %v, want GetLogs(common.Hash{}) == %v",
			state.GetLogs(common.Hash{}, 0, common.Hash{}), checkstate.GetLogs(common.Hash{}, 0, common.Hash{}))
	}
	if !maps.Equal(state.journal.dirties, checkstate.journal.dirties) {
		getKeys := func(dirty map[common.Address]int) string {
			var keys []common.Address
			out := new(strings.Builder)
			for key := range dirty {
				keys = append(keys, key)
			}
			slices.SortFunc(keys, common.Address.Cmp)
			for i, key := range keys {
				fmt.Fprintf(out, "  %d. %v\n", i, key)
			}
			return out.String()
		}
		have := getKeys(state.journal.dirties)
		want := getKeys(checkstate.journal.dirties)
		return fmt.Errorf("dirty-journal set mismatch.\nhave:\n%v\nwant:\n%v\n", have, want)
	}
	return nil
}

func TestTouchDelete(t *testing.T) {
	s := newStateEnv()
	s.state.getOrNewStateObject(common.Address{})
	root, _ := s.state.Commit(0, false)
	s.state, _ = New(root, s.state.db)

	snapshot := s.state.Snapshot()
	s.state.AddBalance(common.Address{}, new(uint256.Int), tracing.BalanceChangeUnspecified)

	if len(s.state.journal.dirties) != 1 {
		t.Fatal("expected one dirty state object")
	}
	s.state.RevertToSnapshot(snapshot)
	if len(s.state.journal.dirties) != 0 {
		t.Fatal("expected no dirty state object")
	}
}

// TestCopyOfCopy tests that modified objects are carried over to the copy, and the copy of the copy.
// See https://github.com/ethereum/go-ethereum/pull/15225#issuecomment-380191512
func TestCopyOfCopy(t *testing.T) {
	state, _ := New(types.EmptyRootHash, NewDatabaseForTesting())
	addr := common.HexToAddress("aaaa")
	state.SetBalance(addr, uint256.NewInt(42), tracing.BalanceChangeUnspecified)

	if got := state.Copy().GetBalance(addr).Uint64(); got != 42 {
		t.Fatalf("1st copy fail, expected 42, got %v", got)
	}
	if got := state.Copy().Copy().GetBalance(addr).Uint64(); got != 42 {
		t.Fatalf("2nd copy fail, expected 42, got %v", got)
	}
}

// Tests a regression where committing a copy lost some internal meta information,
// leading to corrupted subsequent copies.
//
// See https://github.com/ethereum/go-ethereum/issues/20106.
func TestCopyCommitCopy(t *testing.T) {
	tdb := NewDatabaseForTesting()
	state, _ := New(types.EmptyRootHash, tdb)

	// Create an account and check if the retrieved balance is correct
	addr := common.HexToAddress("0xaffeaffeaffeaffeaffeaffeaffeaffeaffeaffe")
	skey := common.HexToHash("aaa")
	sval := common.HexToHash("bbb")

	state.SetBalance(addr, uint256.NewInt(42), tracing.BalanceChangeUnspecified) // Change the account trie
	state.SetCode(addr, []byte("hello"))                                         // Change an external metadata
	state.SetState(addr, skey, sval)                                             // Change the storage trie

	if balance := state.GetBalance(addr); balance.Cmp(uint256.NewInt(42)) != 0 {
		t.Fatalf("initial balance mismatch: have %v, want %v", balance, 42)
	}
	if code := state.GetCode(addr); !bytes.Equal(code, []byte("hello")) {
		t.Fatalf("initial code mismatch: have %x, want %x", code, []byte("hello"))
	}
	if val := state.GetState(addr, skey); val != sval {
		t.Fatalf("initial non-committed storage slot mismatch: have %x, want %x", val, sval)
	}
	if val := state.GetCommittedState(addr, skey); val != (common.Hash{}) {
		t.Fatalf("initial committed storage slot mismatch: have %x, want %x", val, common.Hash{})
	}
	// Copy the non-committed state database and check pre/post commit balance
	copyOne := state.Copy()
	if balance := copyOne.GetBalance(addr); balance.Cmp(uint256.NewInt(42)) != 0 {
		t.Fatalf("first copy pre-commit balance mismatch: have %v, want %v", balance, 42)
	}
	if code := copyOne.GetCode(addr); !bytes.Equal(code, []byte("hello")) {
		t.Fatalf("first copy pre-commit code mismatch: have %x, want %x", code, []byte("hello"))
	}
	if val := copyOne.GetState(addr, skey); val != sval {
		t.Fatalf("first copy pre-commit non-committed storage slot mismatch: have %x, want %x", val, sval)
	}
	if val := copyOne.GetCommittedState(addr, skey); val != (common.Hash{}) {
		t.Fatalf("first copy pre-commit committed storage slot mismatch: have %x, want %x", val, common.Hash{})
	}
	// Copy the copy and check the balance once more
	copyTwo := copyOne.Copy()
	if balance := copyTwo.GetBalance(addr); balance.Cmp(uint256.NewInt(42)) != 0 {
		t.Fatalf("second copy balance mismatch: have %v, want %v", balance, 42)
	}
	if code := copyTwo.GetCode(addr); !bytes.Equal(code, []byte("hello")) {
		t.Fatalf("second copy code mismatch: have %x, want %x", code, []byte("hello"))
	}
	if val := copyTwo.GetState(addr, skey); val != sval {
		t.Fatalf("second copy non-committed storage slot mismatch: have %x, want %x", val, sval)
	}
	if val := copyTwo.GetCommittedState(addr, skey); val != (common.Hash{}) {
		t.Fatalf("second copy committed storage slot mismatch: have %x, want %x", val, sval)
	}
	// Commit state, ensure states can be loaded from disk
	root, _ := state.Commit(0, false)
	state, _ = New(root, tdb)
	if balance := state.GetBalance(addr); balance.Cmp(uint256.NewInt(42)) != 0 {
		t.Fatalf("state post-commit balance mismatch: have %v, want %v", balance, 42)
	}
	if code := state.GetCode(addr); !bytes.Equal(code, []byte("hello")) {
		t.Fatalf("state post-commit code mismatch: have %x, want %x", code, []byte("hello"))
	}
	if val := state.GetState(addr, skey); val != sval {
		t.Fatalf("state post-commit non-committed storage slot mismatch: have %x, want %x", val, sval)
	}
	if val := state.GetCommittedState(addr, skey); val != sval {
		t.Fatalf("state post-commit committed storage slot mismatch: have %x, want %x", val, sval)
	}
}

// Tests a regression where committing a copy lost some internal meta information,
// leading to corrupted subsequent copies.
//
// See https://github.com/ethereum/go-ethereum/issues/20106.
func TestCopyCopyCommitCopy(t *testing.T) {
	state, _ := New(types.EmptyRootHash, NewDatabaseForTesting())

	// Create an account and check if the retrieved balance is correct
	addr := common.HexToAddress("0xaffeaffeaffeaffeaffeaffeaffeaffeaffeaffe")
	skey := common.HexToHash("aaa")
	sval := common.HexToHash("bbb")

	state.SetBalance(addr, uint256.NewInt(42), tracing.BalanceChangeUnspecified) // Change the account trie
	state.SetCode(addr, []byte("hello"))                                         // Change an external metadata
	state.SetState(addr, skey, sval)                                             // Change the storage trie

	if balance := state.GetBalance(addr); balance.Cmp(uint256.NewInt(42)) != 0 {
		t.Fatalf("initial balance mismatch: have %v, want %v", balance, 42)
	}
	if code := state.GetCode(addr); !bytes.Equal(code, []byte("hello")) {
		t.Fatalf("initial code mismatch: have %x, want %x", code, []byte("hello"))
	}
	if val := state.GetState(addr, skey); val != sval {
		t.Fatalf("initial non-committed storage slot mismatch: have %x, want %x", val, sval)
	}
	if val := state.GetCommittedState(addr, skey); val != (common.Hash{}) {
		t.Fatalf("initial committed storage slot mismatch: have %x, want %x", val, common.Hash{})
	}
	// Copy the non-committed state database and check pre/post commit balance
	copyOne := state.Copy()
	if balance := copyOne.GetBalance(addr); balance.Cmp(uint256.NewInt(42)) != 0 {
		t.Fatalf("first copy balance mismatch: have %v, want %v", balance, 42)
	}
	if code := copyOne.GetCode(addr); !bytes.Equal(code, []byte("hello")) {
		t.Fatalf("first copy code mismatch: have %x, want %x", code, []byte("hello"))
	}
	if val := copyOne.GetState(addr, skey); val != sval {
		t.Fatalf("first copy non-committed storage slot mismatch: have %x, want %x", val, sval)
	}
	if val := copyOne.GetCommittedState(addr, skey); val != (common.Hash{}) {
		t.Fatalf("first copy committed storage slot mismatch: have %x, want %x", val, common.Hash{})
	}
	// Copy the copy and check the balance once more
	copyTwo := copyOne.Copy()
	if balance := copyTwo.GetBalance(addr); balance.Cmp(uint256.NewInt(42)) != 0 {
		t.Fatalf("second copy pre-commit balance mismatch: have %v, want %v", balance, 42)
	}
	if code := copyTwo.GetCode(addr); !bytes.Equal(code, []byte("hello")) {
		t.Fatalf("second copy pre-commit code mismatch: have %x, want %x", code, []byte("hello"))
	}
	if val := copyTwo.GetState(addr, skey); val != sval {
		t.Fatalf("second copy pre-commit non-committed storage slot mismatch: have %x, want %x", val, sval)
	}
	if val := copyTwo.GetCommittedState(addr, skey); val != (common.Hash{}) {
		t.Fatalf("second copy pre-commit committed storage slot mismatch: have %x, want %x", val, common.Hash{})
	}
	// Copy the copy-copy and check the balance once more
	copyThree := copyTwo.Copy()
	if balance := copyThree.GetBalance(addr); balance.Cmp(uint256.NewInt(42)) != 0 {
		t.Fatalf("third copy balance mismatch: have %v, want %v", balance, 42)
	}
	if code := copyThree.GetCode(addr); !bytes.Equal(code, []byte("hello")) {
		t.Fatalf("third copy code mismatch: have %x, want %x", code, []byte("hello"))
	}
	if val := copyThree.GetState(addr, skey); val != sval {
		t.Fatalf("third copy non-committed storage slot mismatch: have %x, want %x", val, sval)
	}
	if val := copyThree.GetCommittedState(addr, skey); val != (common.Hash{}) {
		t.Fatalf("third copy committed storage slot mismatch: have %x, want %x", val, sval)
	}
}

// TestCommitCopy tests the copy from a committed state is not fully functional.
func TestCommitCopy(t *testing.T) {
	db := NewDatabaseForTesting()
	state, _ := New(types.EmptyRootHash, db)

	// Create an account and check if the retrieved balance is correct
	addr := common.HexToAddress("0xaffeaffeaffeaffeaffeaffeaffeaffeaffeaffe")
	skey1, skey2 := common.HexToHash("a1"), common.HexToHash("a2")
	sval1, sval2 := common.HexToHash("b1"), common.HexToHash("b2")

	state.SetBalance(addr, uint256.NewInt(42), tracing.BalanceChangeUnspecified) // Change the account trie
	state.SetCode(addr, []byte("hello"))                                         // Change an external metadata
	state.SetState(addr, skey1, sval1)                                           // Change the storage trie

	if balance := state.GetBalance(addr); balance.Cmp(uint256.NewInt(42)) != 0 {
		t.Fatalf("initial balance mismatch: have %v, want %v", balance, 42)
	}
	if code := state.GetCode(addr); !bytes.Equal(code, []byte("hello")) {
		t.Fatalf("initial code mismatch: have %x, want %x", code, []byte("hello"))
	}
	if val := state.GetState(addr, skey1); val != sval1 {
		t.Fatalf("initial non-committed storage slot mismatch: have %x, want %x", val, sval1)
	}
	if val := state.GetCommittedState(addr, skey1); val != (common.Hash{}) {
		t.Fatalf("initial committed storage slot mismatch: have %x, want %x", val, common.Hash{})
	}
	root, _ := state.Commit(0, true)

	state, _ = New(root, db)
	state.SetState(addr, skey2, sval2)
	state.Commit(1, true)

	// Copy the committed state database, the copied one is not fully functional.
	copied := state.Copy()
	if balance := copied.GetBalance(addr); balance.Cmp(uint256.NewInt(42)) != 0 {
		t.Fatalf("unexpected balance: have %v", balance)
	}
	if code := copied.GetCode(addr); !bytes.Equal(code, []byte("hello")) {
		t.Fatalf("unexpected code: have %x", code)
	}
	// Miss slots because of non-functional trie after commit
	if val := copied.GetState(addr, skey1); val != sval1 {
		t.Fatalf("unexpected storage slot: have %x", val)
	}
	if val := copied.GetCommittedState(addr, skey1); val != sval1 {
		t.Fatalf("unexpected storage slot: have %x", val)
	}
	// Slots cached in the stateDB, available after commit
	if val := copied.GetState(addr, skey2); val != sval2 {
		t.Fatalf("unexpected storage slot: have %x", sval1)
	}
	if val := copied.GetCommittedState(addr, skey2); val != sval2 {
		t.Fatalf("unexpected storage slot: have %x", val)
	}
}

// TestDeleteCreateRevert tests a weird state transition corner case that we hit
// while changing the internals of StateDB. The workflow is that a contract is
// self-destructed, then in a follow-up transaction (but same block) it's created
// again and the transaction reverted.
//
// The original StateDB implementation flushed dirty objects to the tries after
// each transaction, so this works ok. The rework accumulated writes in memory
// first, but the journal wiped the entire state object on create-revert.
func TestDeleteCreateRevert(t *testing.T) {
	// Create an initial state with a single contract
	state, _ := New(types.EmptyRootHash, NewDatabaseForTesting())

	addr := common.BytesToAddress([]byte("so"))
	state.SetBalance(addr, uint256.NewInt(1), tracing.BalanceChangeUnspecified)

	root, _ := state.Commit(0, false)
	state, _ = New(root, state.db)

	// Simulate self-destructing in one transaction, then create-reverting in another
	state.SelfDestruct(addr)
	state.Finalise(true)

	id := state.Snapshot()
	state.SetBalance(addr, uint256.NewInt(2), tracing.BalanceChangeUnspecified)
	state.RevertToSnapshot(id)

	// Commit the entire state and make sure we don't crash and have the correct state
	root, _ = state.Commit(0, true)
	state, _ = New(root, state.db)

	if state.getStateObject(addr) != nil {
		t.Fatalf("self-destructed contract came alive")
	}
}

// TestMissingTrieNodes tests that if the StateDB fails to load parts of the trie,
// the Commit operation fails with an error
// If we are missing trie nodes, we should not continue writing to the trie
func TestMissingTrieNodes(t *testing.T) {
	testMissingTrieNodes(t, rawdb.HashScheme)
	testMissingTrieNodes(t, rawdb.PathScheme)
}

func testMissingTrieNodes(t *testing.T, scheme string) {
	// Create an initial state with a few accounts
	var (
		tdb   *triedb.Database
		memDb = rawdb.NewMemoryDatabase()
	)
	if scheme == rawdb.PathScheme {
		tdb = triedb.NewDatabase(memDb, &triedb.Config{PathDB: &pathdb.Config{
			CleanCacheSize:  0,
			WriteBufferSize: 0,
		}}) // disable caching
	} else {
		tdb = triedb.NewDatabase(memDb, &triedb.Config{HashDB: &hashdb.Config{
			CleanCacheSize: 0,
		}}) // disable caching
	}
	db := NewDatabase(tdb, nil)

	var root common.Hash
	state, _ := New(types.EmptyRootHash, db)
	addr := common.BytesToAddress([]byte("so"))
	{
		state.SetBalance(addr, uint256.NewInt(1), tracing.BalanceChangeUnspecified)
		state.SetCode(addr, []byte{1, 2, 3})
		a2 := common.BytesToAddress([]byte("another"))
		state.SetBalance(a2, uint256.NewInt(100), tracing.BalanceChangeUnspecified)
		state.SetCode(a2, []byte{1, 2, 4})
		root, _ = state.Commit(0, false)
		t.Logf("root: %x", root)
		// force-flush
		tdb.Commit(root, false)
	}
	// Create a new state on the old root
	state, _ = New(root, db)
	// Now we clear out the memdb
	it := memDb.NewIterator(nil, nil)
	for it.Next() {
		k := it.Key()
		// Leave the root intact
		if !bytes.Equal(k, root[:]) {
			t.Logf("key: %x", k)
			memDb.Delete(k)
		}
	}
	balance := state.GetBalance(addr)
	// The removed elem should lead to it returning zero balance
	if exp, got := uint64(0), balance.Uint64(); got != exp {
		t.Errorf("expected %d, got %d", exp, got)
	}
	// Modify the state
	state.SetBalance(addr, uint256.NewInt(2), tracing.BalanceChangeUnspecified)
	root, err := state.Commit(0, false)
	if err == nil {
		t.Fatalf("expected error, got root :%x", root)
	}
}

func TestStateDBAccessList(t *testing.T) {
	// Some helpers
	addr := func(a string) common.Address {
		return common.HexToAddress(a)
	}
	slot := func(a string) common.Hash {
		return common.HexToHash(a)
	}

	db := NewDatabaseForTesting()
	state, _ := New(types.EmptyRootHash, db)
	state.accessList = newAccessList()

	verifyAddrs := func(astrings ...string) {
		t.Helper()
		// convert to common.Address form
		var addresses []common.Address
		var addressMap = make(map[common.Address]struct{})
		for _, astring := range astrings {
			address := addr(astring)
			addresses = append(addresses, address)
			addressMap[address] = struct{}{}
		}
		// Check that the given addresses are in the access list
		for _, address := range addresses {
			if !state.AddressInAccessList(address) {
				t.Fatalf("expected %x to be in access list", address)
			}
		}
		// Check that only the expected addresses are present in the access list
		for address := range state.accessList.addresses {
			if _, exist := addressMap[address]; !exist {
				t.Fatalf("extra address %x in access list", address)
			}
		}
	}
	verifySlots := func(addrString string, slotStrings ...string) {
		if !state.AddressInAccessList(addr(addrString)) {
			t.Fatalf("scope missing address/slots %v", addrString)
		}
		var address = addr(addrString)
		// convert to common.Hash form
		var slots []common.Hash
		var slotMap = make(map[common.Hash]struct{})
		for _, slotString := range slotStrings {
			s := slot(slotString)
			slots = append(slots, s)
			slotMap[s] = struct{}{}
		}
		// Check that the expected items are in the access list
		for i, s := range slots {
			if _, slotPresent := state.SlotInAccessList(address, s); !slotPresent {
				t.Fatalf("input %d: scope missing slot %v (address %v)", i, s, addrString)
			}
		}
		// Check that no extra elements are in the access list
		index := state.accessList.addresses[address]
		if index >= 0 {
			stateSlots := state.accessList.slots[index]
			for s := range stateSlots {
				if _, slotPresent := slotMap[s]; !slotPresent {
					t.Fatalf("scope has extra slot %v (address %v)", s, addrString)
				}
			}
		}
	}

	state.AddAddressToAccessList(addr("aa"))          // 1
	state.AddSlotToAccessList(addr("bb"), slot("01")) // 2,3
	state.AddSlotToAccessList(addr("bb"), slot("02")) // 4
	verifyAddrs("aa", "bb")
	verifySlots("bb", "01", "02")

	// Make a copy
	stateCopy1 := state.Copy()
	if exp, got := 4, state.journal.length(); exp != got {
		t.Fatalf("journal length mismatch: have %d, want %d", got, exp)
	}

	// same again, should cause no journal entries
	state.AddSlotToAccessList(addr("bb"), slot("01"))
	state.AddSlotToAccessList(addr("bb"), slot("02"))
	state.AddAddressToAccessList(addr("aa"))
	if exp, got := 4, state.journal.length(); exp != got {
		t.Fatalf("journal length mismatch: have %d, want %d", got, exp)
	}
	// some new ones
	state.AddSlotToAccessList(addr("bb"), slot("03")) // 5
	state.AddSlotToAccessList(addr("aa"), slot("01")) // 6
	state.AddSlotToAccessList(addr("cc"), slot("01")) // 7,8
	state.AddAddressToAccessList(addr("cc"))
	if exp, got := 8, state.journal.length(); exp != got {
		t.Fatalf("journal length mismatch: have %d, want %d", got, exp)
	}

	verifyAddrs("aa", "bb", "cc")
	verifySlots("aa", "01")
	verifySlots("bb", "01", "02", "03")
	verifySlots("cc", "01")

	// now start rolling back changes
	state.journal.revert(state, 7)
	if _, ok := state.SlotInAccessList(addr("cc"), slot("01")); ok {
		t.Fatalf("slot present, expected missing")
	}
	verifyAddrs("aa", "bb", "cc")
	verifySlots("aa", "01")
	verifySlots("bb", "01", "02", "03")

	state.journal.revert(state, 6)
	if state.AddressInAccessList(addr("cc")) {
		t.Fatalf("addr present, expected missing")
	}
	verifyAddrs("aa", "bb")
	verifySlots("aa", "01")
	verifySlots("bb", "01", "02", "03")

	state.journal.revert(state, 5)
	if _, ok := state.SlotInAccessList(addr("aa"), slot("01")); ok {
		t.Fatalf("slot present, expected missing")
	}
	verifyAddrs("aa", "bb")
	verifySlots("bb", "01", "02", "03")

	state.journal.revert(state, 4)
	if _, ok := state.SlotInAccessList(addr("bb"), slot("03")); ok {
		t.Fatalf("slot present, expected missing")
	}
	verifyAddrs("aa", "bb")
	verifySlots("bb", "01", "02")

	state.journal.revert(state, 3)
	if _, ok := state.SlotInAccessList(addr("bb"), slot("02")); ok {
		t.Fatalf("slot present, expected missing")
	}
	verifyAddrs("aa", "bb")
	verifySlots("bb", "01")

	state.journal.revert(state, 2)
	if _, ok := state.SlotInAccessList(addr("bb"), slot("01")); ok {
		t.Fatalf("slot present, expected missing")
	}
	verifyAddrs("aa", "bb")

	state.journal.revert(state, 1)
	if state.AddressInAccessList(addr("bb")) {
		t.Fatalf("addr present, expected missing")
	}
	verifyAddrs("aa")

	state.journal.revert(state, 0)
	if state.AddressInAccessList(addr("aa")) {
		t.Fatalf("addr present, expected missing")
	}
	if got, exp := len(state.accessList.addresses), 0; got != exp {
		t.Fatalf("expected empty, got %d", got)
	}
	if got, exp := len(state.accessList.slots), 0; got != exp {
		t.Fatalf("expected empty, got %d", got)
	}
	// Check the copy
	// Make a copy
	state = stateCopy1
	verifyAddrs("aa", "bb")
	verifySlots("bb", "01", "02")
	if got, exp := len(state.accessList.addresses), 2; got != exp {
		t.Fatalf("expected empty, got %d", got)
	}
	if got, exp := len(state.accessList.slots), 1; got != exp {
		t.Fatalf("expected empty, got %d", got)
	}
}

// Tests that account and storage tries are flushed in the correct order and that
// no data loss occurs.
func TestFlushOrderDataLoss(t *testing.T) {
	// Create a state trie with many accounts and slots
	var (
		memdb    = rawdb.NewMemoryDatabase()
		tdb      = triedb.NewDatabase(memdb, triedb.HashDefaults)
		statedb  = NewDatabase(tdb, nil)
		state, _ = New(types.EmptyRootHash, statedb)
	)
	for a := byte(0); a < 10; a++ {
		state.CreateAccount(common.Address{a})
		for s := byte(0); s < 10; s++ {
			state.SetState(common.Address{a}, common.Hash{a, s}, common.Hash{a, s})
		}
	}
	root, err := state.Commit(0, false)
	if err != nil {
		t.Fatalf("failed to commit state trie: %v", err)
	}
	tdb.Reference(root, common.Hash{})
	if err := tdb.Cap(1024); err != nil {
		t.Fatalf("failed to cap trie dirty cache: %v", err)
	}
	if err := tdb.Commit(root, false); err != nil {
		t.Fatalf("failed to commit state trie: %v", err)
	}
	// Reopen the state trie from flushed disk and verify it
	state, err = New(root, NewDatabase(triedb.NewDatabase(memdb, triedb.HashDefaults), nil))
	if err != nil {
		t.Fatalf("failed to reopen state trie: %v", err)
	}
	for a := byte(0); a < 10; a++ {
		for s := byte(0); s < 10; s++ {
			if have := state.GetState(common.Address{a}, common.Hash{a, s}); have != (common.Hash{a, s}) {
				t.Errorf("account %d: slot %d: state mismatch: have %x, want %x", a, s, have, common.Hash{a, s})
			}
		}
	}
}

func TestStateDBTransientStorage(t *testing.T) {
	db := NewDatabaseForTesting()
	state, _ := New(types.EmptyRootHash, db)

	key := common.Hash{0x01}
	value := common.Hash{0x02}
	addr := common.Address{}

	state.SetTransientState(addr, key, value)
	if exp, got := 1, state.journal.length(); exp != got {
		t.Fatalf("journal length mismatch: have %d, want %d", got, exp)
	}
	// the retrieved value should equal what was set
	if got := state.GetTransientState(addr, key); got != value {
		t.Fatalf("transient storage mismatch: have %x, want %x", got, value)
	}

	// revert the transient state being set and then check that the
	// value is now the empty hash
	state.journal.revert(state, 0)
	if got, exp := state.GetTransientState(addr, key), (common.Hash{}); exp != got {
		t.Fatalf("transient storage mismatch: have %x, want %x", got, exp)
	}

	// set transient state and then copy the statedb and ensure that
	// the transient state is copied
	state.SetTransientState(addr, key, value)
	cpy := state.Copy()
	if got := cpy.GetTransientState(addr, key); got != value {
		t.Fatalf("transient storage mismatch: have %x, want %x", got, value)
	}
}

func TestDeleteStorage(t *testing.T) {
	var (
		disk     = rawdb.NewMemoryDatabase()
		tdb      = triedb.NewDatabase(disk, nil)
		snaps, _ = snapshot.New(snapshot.Config{CacheSize: 10}, disk, tdb, types.EmptyRootHash)
		db       = NewDatabase(tdb, snaps)
		state, _ = New(types.EmptyRootHash, db)
		addr     = common.HexToAddress("0x1")
	)
	// Initialize account and populate storage
	state.SetBalance(addr, uint256.NewInt(1), tracing.BalanceChangeUnspecified)
	state.CreateAccount(addr)
	for i := 0; i < 1000; i++ {
		slot := common.Hash(uint256.NewInt(uint64(i)).Bytes32())
		value := common.Hash(uint256.NewInt(uint64(10 * i)).Bytes32())
		state.SetState(addr, slot, value)
	}
	root, _ := state.Commit(0, true)

	// Init phase done, create two states, one with snap and one without
	fastState, _ := New(root, NewDatabase(tdb, snaps))
	slowState, _ := New(root, NewDatabase(tdb, nil))

	obj := fastState.getOrNewStateObject(addr)
	storageRoot := obj.data.Root

	_, fastNodes, err := fastState.deleteStorage(addr, crypto.Keccak256Hash(addr[:]), storageRoot)
	if err != nil {
		t.Fatal(err)
	}

	_, slowNodes, err := slowState.deleteStorage(addr, crypto.Keccak256Hash(addr[:]), storageRoot)
	if err != nil {
		t.Fatal(err)
	}
	check := func(set *trienode.NodeSet) string {
		var a []string
		set.ForEachWithOrder(func(path string, n *trienode.Node) {
			if n.Hash != (common.Hash{}) {
				t.Fatal("delete should have empty hashes")
			}
			if len(n.Blob) != 0 {
				t.Fatal("delete should have empty blobs")
			}
			a = append(a, fmt.Sprintf("%x", path))
		})
		return strings.Join(a, ",")
	}
	slowRes := check(slowNodes)
	fastRes := check(fastNodes)
	if slowRes != fastRes {
		t.Fatalf("difference found:\nfast: %v\nslow: %v\n", fastRes, slowRes)
	}
}

func TestStorageDirtiness(t *testing.T) {
	var (
		disk       = rawdb.NewMemoryDatabase()
		tdb        = triedb.NewDatabase(disk, nil)
		db         = NewDatabase(tdb, nil)
		state, _   = New(types.EmptyRootHash, db)
		addr       = common.HexToAddress("0x1")
		checkDirty = func(key common.Hash, value common.Hash, dirty bool) {
			obj := state.getStateObject(addr)
			v, exist := obj.dirtyStorage[key]
			if exist != dirty {
				t.Fatalf("Unexpected dirty marker, want: %t, got: %t", dirty, exist)
			}
			if v != value {
				t.Fatalf("Unexpected storage slot, want: %t, got: %t", value, v)
			}
		}
	)
	state.CreateAccount(addr)

	// the storage change is noop, no dirty marker
	state.SetState(addr, common.Hash{0x1}, common.Hash{})
	checkDirty(common.Hash{0x1}, common.Hash{}, false)

	// the storage change is valid, dirty marker is expected
	snap := state.Snapshot()
	state.SetState(addr, common.Hash{0x1}, common.Hash{0x1})
	checkDirty(common.Hash{0x1}, common.Hash{0x1}, true)

	// the storage change is reverted, dirtiness should be revoked
	state.RevertToSnapshot(snap)
	checkDirty(common.Hash{0x1}, common.Hash{}, false)

	// the storage is reset back to its original value, dirtiness should be revoked
	state.SetState(addr, common.Hash{0x1}, common.Hash{0x1})
	snap = state.Snapshot()
	state.SetState(addr, common.Hash{0x1}, common.Hash{})
	checkDirty(common.Hash{0x1}, common.Hash{}, false)

	// the storage change is reverted, dirty value should be set back
	state.RevertToSnapshot(snap)
	checkDirty(common.Hash{0x1}, common.Hash{0x1}, true)
}
