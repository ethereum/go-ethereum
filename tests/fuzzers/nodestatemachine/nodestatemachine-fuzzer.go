package nodestatemachine

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"github.com/ethereum/go-ethereum/common/mclock"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/p2p/enode"
	"github.com/ethereum/go-ethereum/p2p/enr"
	"github.com/ethereum/go-ethereum/p2p/nodestate"
	"github.com/ethereum/go-ethereum/rlp"
	"io"
	"reflect"
	"time"
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

func (f *fuzzer) readSlice(min, max int) []byte {
	var a uint16
	binary.Read(f.input, binary.LittleEndian, &a)
	size := min + int(a)%(max-min)
	out := make([]byte, size)
	if _, err := f.input.Read(out); err != nil {
		f.exhausted = true
	}
	return out
}

func (f *fuzzer) randomBytes(maxlen int) []byte {
	return f.readSlice(0, maxlen)
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

func (f *fuzzer) randomEnode() *enode.Node {
	// 50% chance of using an old enode
	if existing := len(f.enodes); existing > 0 && f.randomBool() {
		index := f.randomInt(existing - 1)
		return f.enodes[index]
	}
	// Create a new one
	node := testNode(f.randomByte())
	f.enodes = append(f.enodes, node)
	return node
}

func (f *fuzzer) randomField() nodestate.Field {
	return f.setup.NewField(string(f.randomBytes(100)), reflect.TypeOf(""))
}

func (f *fuzzer) randomFlags() nodestate.Flags {
	return f.setup.NewFlag(string(f.randomBytes(100)))
}

func (f *fuzzer) randomDuration(maxTime time.Duration) time.Duration {
	var a uint64
	if err := binary.Read(f.input, binary.LittleEndian, &a); err != nil {
		f.exhausted = true
	}
	a = a % uint64(maxTime)
	return time.Duration(a)
}

func (f *fuzzer) fuzz() int {
	mdb, clock := rawdb.NewMemoryDatabase(), &mclock.Simulated{}
	s, _, _ := testSetup([]bool{false, false, false, false}, nil)
	f.setup = s
	ns := nodestate.NewNodeStateMachine(mdb, []byte("-ns"), clock, s)
	ns.Start()
	var stopped bool
	steps := 0
	for !f.exhausted {
		switch f.randomInt(9) {
		case 0:
			ns.SetField(f.randomEnode(), f.randomField(), f.randomBytes(4))
			// The one below panics easily,
			//ns.SetFieldSub(f.randomEnode(), f.randomField(), f.randomBytes(5))
		case 1:
			ns.SetField(f.randomEnode(), f.randomField(), f.randomBytes(4))
		case 2:
			ns.AddTimeout(f.randomEnode(), f.randomFlags(), f.randomDuration(200*time.Millisecond))
		case 3:
			ns.Operation(func() {
				//time.Sleep(f.randomDuration(200 * time.Millisecond))
			})
		case 4:
			ns.Persist(f.randomEnode())
		case 5:
			if !stopped { // double Stop causes panic, because reasons
				ns.Stop()
			}
			stopped = true
		case 6:
			ns.ForEach(f.randomFlags(), f.randomFlags(), func(n *enode.Node, state nodestate.Flags) {})
		case 7:
			ns.Persist(f.randomEnode())
		case 8:
			ns.GetNode(f.randomEnode().ID())
		}
		steps++
	}
	if steps > 2 {
		return 1
	}
	return 0

}
