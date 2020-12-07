package nodestatemachine

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"reflect"
	"time"

	"github.com/ethereum/go-ethereum/common/mclock"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/p2p/enode"
	"github.com/ethereum/go-ethereum/p2p/enr"
	"github.com/ethereum/go-ethereum/p2p/nodestate"
	"github.com/ethereum/go-ethereum/rlp"
)

// The function must return
// 1 if the fuzzer should increase priority of the
//    given input during subsequent fuzzing (for example, the input is lexically
//    correct and was parsed successfully);
// -1 if the input must not be added to corpus even if gives new coverage; and
// 0  otherwise
// other values are reserved for future use.
func Fuzz(data []byte) int {
	f := fuzzer{
		input:     bytes.NewReader(data),
		exhausted: false,
	}
	return f.fuzz()
}

type dummyIdentity enode.ID

func (id dummyIdentity) Verify(r *enr.Record, sig []byte) error { return nil }
func (id dummyIdentity) NodeAddr(r *enr.Record) []byte          { return id[:] }

func testNode(b byte) *enode.Node {
	r := &enr.Record{}
	r.SetSig(dummyIdentity{b}, []byte{42})
	n, _ := enode.New(dummyIdentity{b}, r)
	return n
}
func uint64FieldEnc(field interface{}) ([]byte, error) {
	if u, ok := field.(uint64); ok {
		enc, err := rlp.EncodeToBytes(&u)
		return enc, err
	}
	return nil, errors.New("invalid field type")
}

func uint64FieldDec(enc []byte) (interface{}, error) {
	var u uint64
	err := rlp.DecodeBytes(enc, &u)
	return u, err
}

func stringFieldEnc(field interface{}) ([]byte, error) {
	if s, ok := field.(string); ok {
		return []byte(s), nil
	}
	return nil, errors.New("invalid field type")
}

func stringFieldDec(enc []byte) (interface{}, error) {
	return string(enc), nil
}

func testSetup(flagPersist []bool, fieldType []reflect.Type) (*nodestate.Setup, []nodestate.Flags, []nodestate.Field) {
	setup := &nodestate.Setup{}
	flags := make([]nodestate.Flags, len(flagPersist))
	for i, persist := range flagPersist {
		if persist {
			flags[i] = setup.NewPersistentFlag(fmt.Sprintf("flag-%d", i))
		} else {
			flags[i] = setup.NewFlag(fmt.Sprintf("flag-%d", i))
		}
	}
	fields := make([]nodestate.Field, len(fieldType))
	for i, ftype := range fieldType {
		switch ftype {
		case reflect.TypeOf(uint64(0)):
			fields[i] = setup.NewPersistentField(fmt.Sprintf("field-%d", i), ftype, uint64FieldEnc, uint64FieldDec)
		case reflect.TypeOf(""):
			fields[i] = setup.NewPersistentField(fmt.Sprintf("field-%d", i), ftype, stringFieldEnc, stringFieldDec)
		default:
			fields[i] = setup.NewField(fmt.Sprintf("field-%d", i), ftype)
		}
	}
	return setup, flags, fields
}

type fuzzer struct {
	input     io.Reader
	exhausted bool
	debugging bool
	enodes    []*enode.Node
	setup     *nodestate.Setup
}

func (f *fuzzer) read(size int) []byte {
	out := make([]byte, size)
	if _, err := f.input.Read(out); err != nil {
		f.exhausted = true
	}
	return out
}

func (f *fuzzer) randomByte() byte {
	d := f.read(1)
	return d[0]
}
func (f *fuzzer) randomBool() bool {
	d := f.read(1)
	return d[0]&1 == 1
}

func (f *fuzzer) randomInt(max int) int {
	if max == 0 {
		return 0
	}
	var a uint16
	if err := binary.Read(f.input, binary.LittleEndian, &a); err != nil {
		f.exhausted = true
	}
	return int(a % uint16(max))
}

const (
	nodeCount  = 64
	flagCount  = 10
	fieldCount = 10
	cbMin      = 500
	cbMax      = 2500
	cbLimit    = 10000
	opRepeat   = 8
)

type state [nodeCount]struct {
	mask   uint64
	fields [fieldCount]uint16
}

func (f *fuzzer) fuzz() int {
	mdb, clock := rawdb.NewMemoryDatabase(), &mclock.Simulated{}
	fl, fi := make([]bool, flagCount), make([]reflect.Type, fieldCount)
	for i := range fl {
		fl[i] = true
	}
	for i := range fi {
		if f.randomBool() {
			fi[i] = reflect.TypeOf(uint64(0))
		} else {
			fi[i] = reflect.TypeOf("")
		}
	}
	s, flags, fields := testSetup(fl, fi)
	f.setup = s
	ns := nodestate.NewNodeStateMachine(mdb, []byte("-ns"), clock, s)
	var callbackCount int
	var ops [256]func(n *enode.Node, sub bool)

	runsub := func(n *enode.Node, op byte) {
		if fn := ops[op]; fn != nil {
			fn(n, true)
		}
	}

	var allFlags nodestate.Flags
	for _, f := range flags {
		allFlags = allFlags.Or(f)
	}
	var offState state

	var flagSubs [16]struct {
		a, b, c int
		ops     [8]byte
	}
	shift := byte(42)
	for i := range flagSubs {
		x := f.randomInt(flagCount * flagCount * flagCount)
		flagSubs[i].a = x % flagCount
		x /= flagCount
		flagSubs[i].b = x % flagCount
		x /= flagCount
		flagSubs[i].c = x % flagCount
		var subops [8]byte
		for j := range subops {
			flagSubs[i].ops[j] = f.randomByte() + shift
			shift += 77
		}
	}

	var fieldSubs [fieldCount]struct {
		ops [8]byte
	}
	for ii := range fields {
		var subops [8]byte
		for j := range subops {
			fieldSubs[ii].ops[j] = f.randomByte() + shift
			shift += 77
		}
	}

	addSubs := func() {
		ns.SubscribeState(allFlags.Or(s.OfflineFlag()), func(n *enode.Node, oldState, newState nodestate.Flags) {
			o1, o2 := oldState.HasAll(s.OfflineFlag()), newState.HasAll(s.OfflineFlag())
			if o1 != o2 {
				var st nodestate.Flags
				if o1 {
					st = newState
				} else {
					st = oldState
				}
				var m uint64
				for _, f := range flags {
					m += m
					if st.HasAll(f) {
						m++
					}
				}
				offState[int(n.ID()[0])].mask = m
			}
		})

		for _, sub := range flagSubs {
			ns.SubscribeState(nodestate.MergeFlags(s.OfflineFlag(), flags[sub.a], flags[sub.b], flags[sub.c]), func(n *enode.Node, oldState, newState nodestate.Flags) {
				if oldState.Or(newState).HasAll(s.OfflineFlag()) {
					return
				}
				callbackCount++
				if callbackCount > cbLimit {
					return
				}
				j := 0
				if newState.HasAll(flags[sub.a]) {
					j += 1
				}
				if newState.HasAll(flags[sub.b]) {
					j += 2
				}
				if newState.HasAll(flags[sub.c]) {
					j += 4
				}
				runsub(n, sub.ops[j])
			})
		}

		for ii, fi := range fields {
			fieldIdx := ii
			ns.SubscribeField(fi, func(n *enode.Node, state nodestate.Flags, oldValue, newValue interface{}) {
				if state.HasAll(s.OfflineFlag()) {
					v := oldValue
					if v == nil {
						v = newValue
					}
					var vv uint16
					if uv, ok := v.(uint64); ok {
						vv = 256 + uint16(byte(uv))
					}
					if sv, ok := v.(string); ok {
						vv = 512 + uint16(sv[0])
					}
					offState[int(n.ID()[0])].fields[fieldIdx] = vv
					return
				}

				callbackCount++
				if callbackCount > cbLimit {
					return
				}
				var j int
				if newValue != nil {
					if u, ok := newValue.(uint64); ok {
						j = int(u%7) + 1
					} else {
						s := newValue.(string)
						j = int(s[0])%7 + 1
					}
				}
				runsub(n, fieldSubs[fieldIdx].ops[j])
			})
		}
	}
	for i := range ops {
		b := f.randomByte()
		if b+byte(i*137) < 4 {
			break
		}
		b2 := f.randomByte()
		u := uint(b) + uint(b2)<<8
		ops[i] = func(n *enode.Node, sub bool) {
			u := u
			optype := u % 5
			u /= 5
			nn := n
			shift := u % 4
			u /= 4
			if shift < 2 {
				nodeIdx := n.ID()[0]
				if shift == 0 {
					nodeIdx = (nodeIdx + nodeCount - 1) % nodeCount
				} else {
					nodeIdx = (nodeIdx + 1) % nodeCount
				}
				nn = testNode(nodeIdx)
			}
			switch optype {
			case 0: // set state	(set/reset 2 flags)
				idx1 := u % flagCount
				u /= flagCount
				set1 := u%2 == 1
				u /= 2
				idx2 := u % flagCount
				u /= flagCount
				set2 := u%2 == 1
				var set, reset nodestate.Flags
				if set1 {
					set = set.Or(flags[idx1])
				} else {
					reset = reset.Or(flags[idx1])
				}
				if set2 {
					set = set.Or(flags[idx2])
				} else {
					reset = reset.Or(flags[idx2])
				}
				if sub {
					ns.SetStateSub(nn, set, reset, 0)
				} else {
					ns.SetState(nn, set, reset, 0)
				}
			case 1: // set state	(set 1 flag, add timeout)
				set := flags[u%flagCount]
				u /= flagCount
				timeout := time.Second * time.Duration(u%20)
				if sub {
					ns.SetStateSub(nn, set, nodestate.Flags{}, timeout)
				} else {
					ns.SetState(nn, set, nodestate.Flags{}, timeout)
				}
			case 2: // set field (nil or uint64/string value)
				set := fields[u%fieldCount]
				u /= fieldCount
				vtype := u % 4
				u /= 4
				var value interface{}
				switch vtype {
				case 0:
					value = uint64(byte(u))
				case 1:
					s := string([]byte{byte(u)})
					value = s
				default: // use nil with 50% chance
				}
				if sub {
					ns.SetFieldSub(nn, set, value)
				} else {
					ns.SetField(nn, set, value)
				}
			case 3: // get/set (copy) field value
				get := fields[u%fieldCount]
				u /= fieldCount
				set := fields[u%fieldCount]
				value := ns.GetField(nn, get)
				if sub {
					ns.SetFieldSub(nn, set, value)
				} else {
					ns.SetField(nn, set, value)
				}
			case 4: // add timeout to all flags of the node
				timeout := time.Second * time.Duration(u%20)
				ns.AddTimeout(nn, allFlags, timeout)
			}
		}
	}

	var l []byte
	for !f.exhausted && len(l) < 1000 {
		l = append(l, f.randomByte())
	}
	if len(l) == 0 || !f.exhausted {
		return -1
	}
	oplist := make([]byte, len(l)*opRepeat)
	b := byte(0)
	for r := 0; r < opRepeat; r++ {
		for i, o := range l {
			oplist[r*len(l)+i] = o + b
		}
		b += 81
	}
	ll := len(oplist) / 2
	oplist1, oplist2 := oplist[:ll], oplist[ll:]

	timers := clock.ActiveTimers()
	run := func(list []byte, nodeIdx *int) {
		var ptr int
		for ptr < len(list) {
			op := list[ptr]
			ptr++
			if op+byte(*nodeIdx)*111 < 4 {
				// run a batch in an ns.Operation
				ns.Operation(func() {
					for ptr < len(list) {
						op := list[ptr]
						ptr++
						if op+byte(*nodeIdx)*111 < 32 {
							return
						}
						if fn := ops[op]; fn != nil {
							fn(testNode(byte(*nodeIdx)), true)
						}
					}
				})
			} else if fn := ops[op]; fn != nil {
				// run single top-level operation
				fn(testNode(byte(*nodeIdx)), false)
				*nodeIdx = (*nodeIdx + 1) % nodeCount
				clock.Run(time.Second)
			}
		}
		for timers < clock.ActiveTimers() {
			clock.Run(time.Minute) // wait for remaining timeouts in order to ensure consistency
		}
	}

	var nodeIdx int
	addSubs()
	ns.Start()
	run(oplist1, &nodeIdx)
	run(oplist2, &nodeIdx)
	offState = state{}
	ns.Stop()
	endState := offState

	cbc := callbackCount
	if cbc > cbLimit {
		return -1
	}
	callbackCount = 0

	mdb, clock = rawdb.NewMemoryDatabase(), &mclock.Simulated{}
	ns = nodestate.NewNodeStateMachine(mdb, []byte("-ns"), clock, s)
	addSubs()
	nodeIdx = 0

	ns.Start()
	run(oplist1, &nodeIdx)
	offState = state{}
	ns.Stop()
	savedState := offState

	ns = nodestate.NewNodeStateMachine(mdb, []byte("-ns"), clock, s)
	addSubs()
	offState = state{}
	ns.Start()
	loadedState := offState
	if loadedState != savedState {
		panic("saved/loaded state mismatch")
	}
	run(oplist2, &nodeIdx)
	offState = state{}
	ns.Stop()
	endState2 := offState
	if endState != endState2 {
		panic("end state mismatch")
	}

	if cbc >= cbMin && cbc < cbMax {
		return 1
	}
	return 0

}
