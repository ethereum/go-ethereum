// Copyright 2018 The go-ethereum Authors
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

package pss

import (
	"bytes"
	"context"
	"crypto/ecdsa"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"math/rand"
	"os"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/metrics"
	"github.com/ethereum/go-ethereum/metrics/influxdb"
	"github.com/ethereum/go-ethereum/node"
	"github.com/ethereum/go-ethereum/p2p"
	"github.com/ethereum/go-ethereum/p2p/enode"
	"github.com/ethereum/go-ethereum/p2p/protocols"
	"github.com/ethereum/go-ethereum/p2p/simulations"
	"github.com/ethereum/go-ethereum/p2p/simulations/adapters"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/ethereum/go-ethereum/swarm/network"
	"github.com/ethereum/go-ethereum/swarm/state"
	whisper "github.com/ethereum/go-ethereum/whisper/whisperv5"
)

var (
	initOnce       = sync.Once{}
	debugdebugflag = flag.Bool("vv", false, "veryverbose")
	debugflag      = flag.Bool("v", false, "verbose")
	longrunning    = flag.Bool("longrunning", false, "do run long-running tests")
	w              *whisper.Whisper
	wapi           *whisper.PublicWhisperAPI
	psslogmain     log.Logger
	pssprotocols   map[string]*protoCtrl
	useHandshake   bool
)

func init() {
	flag.Parse()
	rand.Seed(time.Now().Unix())

	adapters.RegisterServices(newServices(false))
	initTest()
}

func initTest() {
	initOnce.Do(
		func() {
			loglevel := log.LvlInfo
			if *debugflag {
				loglevel = log.LvlDebug
			} else if *debugdebugflag {
				loglevel = log.LvlTrace
			}

			psslogmain = log.New("psslog", "*")
			hs := log.StreamHandler(os.Stderr, log.TerminalFormat(true))
			hf := log.LvlFilterHandler(loglevel, hs)
			h := log.CallerFileHandler(hf)
			log.Root().SetHandler(h)

			w = whisper.New(&whisper.DefaultConfig)
			wapi = whisper.NewPublicWhisperAPI(w)

			pssprotocols = make(map[string]*protoCtrl)
		},
	)
}

// test that topic conversion functions give predictable results
func TestTopic(t *testing.T) {

	api := &API{}

	topicstr := strings.Join([]string{PingProtocol.Name, strconv.Itoa(int(PingProtocol.Version))}, ":")

	// bytestotopic is the authoritative topic conversion source
	topicobj := BytesToTopic([]byte(topicstr))

	// string to topic and bytes to topic must match
	topicapiobj, _ := api.StringToTopic(topicstr)
	if topicobj != topicapiobj {
		t.Fatalf("bytes and string topic conversion mismatch; %s != %s", topicobj, topicapiobj)
	}

	// string representation of topichex
	topichex := topicobj.String()

	// protocoltopic wrapper on pingtopic should be same as topicstring
	// check that it matches
	pingtopichex := PingTopic.String()
	if topichex != pingtopichex {
		t.Fatalf("protocol topic conversion mismatch; %s != %s", topichex, pingtopichex)
	}

	// json marshal of topic
	topicjsonout, err := topicobj.MarshalJSON()
	if err != nil {
		t.Fatal(err)
	}
	if string(topicjsonout)[1:len(topicjsonout)-1] != topichex {
		t.Fatalf("topic json marshal mismatch; %s != \"%s\"", topicjsonout, topichex)
	}

	// json unmarshal of topic
	var topicjsonin Topic
	topicjsonin.UnmarshalJSON(topicjsonout)
	if topicjsonin != topicobj {
		t.Fatalf("topic json unmarshal mismatch: %x != %x", topicjsonin, topicobj)
	}
}

// test bit packing of message control flags
func TestMsgParams(t *testing.T) {
	var ctrl byte
	ctrl |= pssControlRaw
	p := newMsgParamsFromBytes([]byte{ctrl})
	m := newPssMsg(p)
	if !m.isRaw() || m.isSym() {
		t.Fatal("expected raw=true and sym=false")
	}
	ctrl |= pssControlSym
	p = newMsgParamsFromBytes([]byte{ctrl})
	m = newPssMsg(p)
	if !m.isRaw() || !m.isSym() {
		t.Fatal("expected raw=true and sym=true")
	}
	ctrl &= 0xff &^ pssControlRaw
	p = newMsgParamsFromBytes([]byte{ctrl})
	m = newPssMsg(p)
	if m.isRaw() || !m.isSym() {
		t.Fatal("expected raw=false and sym=true")
	}
}

// test if we can insert into cache, match items with cache and cache expiry
func TestCache(t *testing.T) {
	var err error
	to, _ := hex.DecodeString("08090a0b0c0d0e0f1011121314150001020304050607161718191a1b1c1d1e1f")
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	keys, err := wapi.NewKeyPair(ctx)
	privkey, err := w.GetPrivateKey(keys)
	if err != nil {
		t.Fatal(err)
	}
	ps := newTestPss(privkey, nil, nil)
	pp := NewPssParams().WithPrivateKey(privkey)
	data := []byte("foo")
	datatwo := []byte("bar")
	datathree := []byte("baz")
	wparams := &whisper.MessageParams{
		TTL:      defaultWhisperTTL,
		Src:      privkey,
		Dst:      &privkey.PublicKey,
		Topic:    whisper.TopicType(PingTopic),
		WorkTime: defaultWhisperWorkTime,
		PoW:      defaultWhisperPoW,
		Payload:  data,
	}
	woutmsg, err := whisper.NewSentMessage(wparams)
	env, err := woutmsg.Wrap(wparams)
	msg := &PssMsg{
		Payload: env,
		To:      to,
	}
	wparams.Payload = datatwo
	woutmsg, err = whisper.NewSentMessage(wparams)
	envtwo, err := woutmsg.Wrap(wparams)
	msgtwo := &PssMsg{
		Payload: envtwo,
		To:      to,
	}
	wparams.Payload = datathree
	woutmsg, err = whisper.NewSentMessage(wparams)
	envthree, err := woutmsg.Wrap(wparams)
	msgthree := &PssMsg{
		Payload: envthree,
		To:      to,
	}

	digest := ps.digest(msg)
	if err != nil {
		t.Fatalf("could not store cache msgone: %v", err)
	}
	digesttwo := ps.digest(msgtwo)
	if err != nil {
		t.Fatalf("could not store cache msgtwo: %v", err)
	}
	digestthree := ps.digest(msgthree)
	if err != nil {
		t.Fatalf("could not store cache msgthree: %v", err)
	}

	if digest == digesttwo {
		t.Fatalf("different msgs return same hash: %d", digesttwo)
	}

	// check the cache
	err = ps.addFwdCache(msg)
	if err != nil {
		t.Fatalf("write to pss expire cache failed: %v", err)
	}

	if !ps.checkFwdCache(msg) {
		t.Fatalf("message %v should have EXPIRE record in cache but checkCache returned false", msg)
	}

	if ps.checkFwdCache(msgtwo) {
		t.Fatalf("message %v should NOT have EXPIRE record in cache but checkCache returned true", msgtwo)
	}

	time.Sleep(pp.CacheTTL + 1*time.Second)
	err = ps.addFwdCache(msgthree)
	if err != nil {
		t.Fatalf("write to pss expire cache failed: %v", err)
	}

	if ps.checkFwdCache(msg) {
		t.Fatalf("message %v should have expired from cache but checkCache returned true", msg)
	}

	if _, ok := ps.fwdCache[digestthree]; !ok {
		t.Fatalf("unexpired message should be in the cache: %v", digestthree)
	}

	if _, ok := ps.fwdCache[digesttwo]; ok {
		t.Fatalf("expired message should have been cleared from the cache: %v", digesttwo)
	}
}

// matching of address hints; whether a message could be or is for the node
func TestAddressMatch(t *testing.T) {

	localaddr := network.RandomAddr().Over()
	copy(localaddr[:8], []byte("deadbeef"))
	remoteaddr := []byte("feedbeef")
	kadparams := network.NewKadParams()
	kad := network.NewKademlia(localaddr, kadparams)
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	keys, err := wapi.NewKeyPair(ctx)
	if err != nil {
		t.Fatalf("Could not generate private key: %v", err)
	}
	privkey, err := w.GetPrivateKey(keys)
	pssp := NewPssParams().WithPrivateKey(privkey)
	ps, err := NewPss(kad, pssp)
	if err != nil {
		t.Fatal(err.Error())
	}

	pssmsg := &PssMsg{
		To:      remoteaddr,
		Payload: &whisper.Envelope{},
	}

	// differ from first byte
	if ps.isSelfRecipient(pssmsg) {
		t.Fatalf("isSelfRecipient true but %x != %x", remoteaddr, localaddr)
	}
	if ps.isSelfPossibleRecipient(pssmsg) {
		t.Fatalf("isSelfPossibleRecipient true but %x != %x", remoteaddr[:8], localaddr[:8])
	}

	// 8 first bytes same
	copy(remoteaddr[:4], localaddr[:4])
	if ps.isSelfRecipient(pssmsg) {
		t.Fatalf("isSelfRecipient true but %x != %x", remoteaddr, localaddr)
	}
	if !ps.isSelfPossibleRecipient(pssmsg) {
		t.Fatalf("isSelfPossibleRecipient false but %x == %x", remoteaddr[:8], localaddr[:8])
	}

	// all bytes same
	pssmsg.To = localaddr
	if !ps.isSelfRecipient(pssmsg) {
		t.Fatalf("isSelfRecipient false but %x == %x", remoteaddr, localaddr)
	}
	if !ps.isSelfPossibleRecipient(pssmsg) {
		t.Fatalf("isSelfPossibleRecipient false but %x == %x", remoteaddr[:8], localaddr[:8])
	}
}

//
func TestHandlerConditions(t *testing.T) {

	t.Skip("Disabled due to probable faulty logic for outbox expectations")
	// setup
	privkey, err := crypto.GenerateKey()
	if err != nil {
		t.Fatal(err.Error())
	}

	addr := make([]byte, 32)
	addr[0] = 0x01
	ps := newTestPss(privkey, network.NewKademlia(addr, network.NewKadParams()), NewPssParams())

	// message should pass
	msg := &PssMsg{
		To:     addr,
		Expire: uint32(time.Now().Add(time.Second * 60).Unix()),
		Payload: &whisper.Envelope{
			Topic: [4]byte{},
			Data:  []byte{0x66, 0x6f, 0x6f},
		},
	}
	if err := ps.handlePssMsg(context.TODO(), msg); err != nil {
		t.Fatal(err.Error())
	}
	tmr := time.NewTimer(time.Millisecond * 100)
	var outmsg *PssMsg
	select {
	case outmsg = <-ps.outbox:
	case <-tmr.C:
	default:
	}
	if outmsg != nil {
		t.Fatalf("expected outbox empty after full address on msg, but had message %s", msg)
	}

	// message should pass and queue due to partial length
	msg.To = addr[0:1]
	msg.Payload.Data = []byte{0x78, 0x79, 0x80, 0x80, 0x79}
	if err := ps.handlePssMsg(context.TODO(), msg); err != nil {
		t.Fatal(err.Error())
	}
	tmr.Reset(time.Millisecond * 100)
	outmsg = nil
	select {
	case outmsg = <-ps.outbox:
	case <-tmr.C:
	}
	if outmsg == nil {
		t.Fatal("expected message in outbox on encrypt fail, but empty")
	}
	outmsg = nil
	select {
	case outmsg = <-ps.outbox:
	default:
	}
	if outmsg != nil {
		t.Fatalf("expected only one queued message but also had message %v", msg)
	}

	// full address mismatch should put message in queue
	msg.To[0] = 0xff
	if err := ps.handlePssMsg(context.TODO(), msg); err != nil {
		t.Fatal(err.Error())
	}
	tmr.Reset(time.Millisecond * 10)
	outmsg = nil
	select {
	case outmsg = <-ps.outbox:
	case <-tmr.C:
	}
	if outmsg == nil {
		t.Fatal("expected message in outbox on address mismatch, but empty")
	}
	outmsg = nil
	select {
	case outmsg = <-ps.outbox:
	default:
	}
	if outmsg != nil {
		t.Fatalf("expected only one queued message but also had message %v", msg)
	}

	// expired message should be dropped
	msg.Expire = uint32(time.Now().Add(-time.Second).Unix())
	if err := ps.handlePssMsg(context.TODO(), msg); err != nil {
		t.Fatal(err.Error())
	}
	tmr.Reset(time.Millisecond * 10)
	outmsg = nil
	select {
	case outmsg = <-ps.outbox:
	case <-tmr.C:
	default:
	}
	if outmsg != nil {
		t.Fatalf("expected empty queue but have message %v", msg)
	}

	// invalid message should return error
	fckedupmsg := &struct {
		pssMsg *PssMsg
	}{
		pssMsg: &PssMsg{},
	}
	if err := ps.handlePssMsg(context.TODO(), fckedupmsg); err == nil {
		t.Fatalf("expected error from processMsg but error nil")
	}

	// outbox full should return error
	msg.Expire = uint32(time.Now().Add(time.Second * 60).Unix())
	for i := 0; i < defaultOutboxCapacity; i++ {
		ps.outbox <- msg
	}
	msg.Payload.Data = []byte{0x62, 0x61, 0x72}
	err = ps.handlePssMsg(context.TODO(), msg)
	if err == nil {
		t.Fatal("expected error when mailbox full, but was nil")
	}
}

// set and generate pubkeys and symkeys
func TestKeys(t *testing.T) {
	// make our key and init pss with it
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	ourkeys, err := wapi.NewKeyPair(ctx)
	if err != nil {
		t.Fatalf("create 'our' key fail")
	}
	ctx, cancel2 := context.WithTimeout(context.Background(), time.Second)
	defer cancel2()
	theirkeys, err := wapi.NewKeyPair(ctx)
	if err != nil {
		t.Fatalf("create 'their' key fail")
	}
	ourprivkey, err := w.GetPrivateKey(ourkeys)
	if err != nil {
		t.Fatalf("failed to retrieve 'our' private key")
	}
	theirprivkey, err := w.GetPrivateKey(theirkeys)
	if err != nil {
		t.Fatalf("failed to retrieve 'their' private key")
	}
	ps := newTestPss(ourprivkey, nil, nil)

	// set up peer with mock address, mapped to mocked publicaddress and with mocked symkey
	addr := make(PssAddress, 32)
	copy(addr, network.RandomAddr().Over())
	outkey := network.RandomAddr().Over()
	topicobj := BytesToTopic([]byte("foo:42"))
	ps.SetPeerPublicKey(&theirprivkey.PublicKey, topicobj, &addr)
	outkeyid, err := ps.SetSymmetricKey(outkey, topicobj, &addr, false)
	if err != nil {
		t.Fatalf("failed to set 'our' outgoing symmetric key")
	}

	// make a symmetric key that we will send to peer for encrypting messages to us
	inkeyid, err := ps.GenerateSymmetricKey(topicobj, &addr, true)
	if err != nil {
		t.Fatalf("failed to set 'our' incoming symmetric key")
	}

	// get the key back from whisper, check that it's still the same
	outkeyback, err := ps.w.GetSymKey(outkeyid)
	if err != nil {
		t.Fatalf(err.Error())
	}
	inkey, err := ps.w.GetSymKey(inkeyid)
	if err != nil {
		t.Fatalf(err.Error())
	}
	if !bytes.Equal(outkeyback, outkey) {
		t.Fatalf("passed outgoing symkey doesnt equal stored: %x / %x", outkey, outkeyback)
	}

	t.Logf("symout: %v", outkeyback)
	t.Logf("symin: %v", inkey)

	// check that the key is stored in the peerpool
	psp := ps.symKeyPool[inkeyid][topicobj]
	if psp.address != &addr {
		t.Fatalf("inkey address does not match; %p != %p", psp.address, &addr)
	}
}

func TestGetPublickeyEntries(t *testing.T) {

	privkey, err := crypto.GenerateKey()
	if err != nil {
		t.Fatal(err)
	}
	ps := newTestPss(privkey, nil, nil)

	peeraddr := network.RandomAddr().Over()
	topicaddr := make(map[Topic]PssAddress)
	topicaddr[Topic{0x13}] = peeraddr
	topicaddr[Topic{0x2a}] = peeraddr[:16]
	topicaddr[Topic{0x02, 0x9a}] = []byte{}

	remoteprivkey, err := crypto.GenerateKey()
	if err != nil {
		t.Fatal(err)
	}
	remotepubkeybytes := crypto.FromECDSAPub(&remoteprivkey.PublicKey)
	remotepubkeyhex := common.ToHex(remotepubkeybytes)

	pssapi := NewAPI(ps)

	for to, a := range topicaddr {
		err = pssapi.SetPeerPublicKey(remotepubkeybytes, to, a)
		if err != nil {
			t.Fatal(err)
		}
	}

	intopic, err := pssapi.GetPeerTopics(remotepubkeyhex)
	if err != nil {
		t.Fatal(err)
	}

OUTER:
	for _, tnew := range intopic {
		for torig, addr := range topicaddr {
			if bytes.Equal(torig[:], tnew[:]) {
				inaddr, err := pssapi.GetPeerAddress(remotepubkeyhex, torig)
				if err != nil {
					t.Fatal(err)
				}
				if !bytes.Equal(addr, inaddr) {
					t.Fatalf("Address mismatch for topic %x; got %x, expected %x", torig, inaddr, addr)
				}
				delete(topicaddr, torig)
				continue OUTER
			}
		}
		t.Fatalf("received topic %x did not match any existing topics", tnew)
	}

	if len(topicaddr) != 0 {
		t.Fatalf("%d topics were not matched", len(topicaddr))
	}
}

// forwarding should skip peers that do not have matching pss capabilities
func TestMismatch(t *testing.T) {

	// create privkey for forwarder node
	privkey, err := crypto.GenerateKey()
	if err != nil {
		t.Fatal(err)
	}

	// initialize kad
	baseaddr := network.RandomAddr()
	kad := network.NewKademlia((baseaddr).Over(), network.NewKadParams())
	rw := &p2p.MsgPipeRW{}

	// one peer has a mismatching version of pss
	wrongpssaddr := network.RandomAddr()
	wrongpsscap := p2p.Cap{
		Name:    pssProtocolName,
		Version: 0,
	}
	nid := enode.ID{0x01}
	wrongpsspeer := network.NewPeer(&network.BzzPeer{
		Peer:    protocols.NewPeer(p2p.NewPeer(nid, common.ToHex(wrongpssaddr.Over()), []p2p.Cap{wrongpsscap}), rw, nil),
		BzzAddr: &network.BzzAddr{OAddr: wrongpssaddr.Over(), UAddr: nil},
	}, kad)

	// one peer doesn't even have pss (boo!)
	nopssaddr := network.RandomAddr()
	nopsscap := p2p.Cap{
		Name:    "nopss",
		Version: 1,
	}
	nid = enode.ID{0x02}
	nopsspeer := network.NewPeer(&network.BzzPeer{
		Peer:    protocols.NewPeer(p2p.NewPeer(nid, common.ToHex(nopssaddr.Over()), []p2p.Cap{nopsscap}), rw, nil),
		BzzAddr: &network.BzzAddr{OAddr: nopssaddr.Over(), UAddr: nil},
	}, kad)

	// add peers to kademlia and activate them
	// it's safe so don't check errors
	kad.Register(wrongpsspeer.BzzAddr)
	kad.On(wrongpsspeer)
	kad.Register(nopsspeer.BzzAddr)
	kad.On(nopsspeer)

	// create pss
	pssmsg := &PssMsg{
		To:      []byte{},
		Expire:  uint32(time.Now().Add(time.Second).Unix()),
		Payload: &whisper.Envelope{},
	}
	ps := newTestPss(privkey, kad, nil)

	// run the forward
	// it is enough that it completes; trying to send to incapable peers would create segfault
	ps.forward(pssmsg)

}

func TestSendRaw(t *testing.T) {
	t.Run("32", testSendRaw)
	t.Run("8", testSendRaw)
	t.Run("0", testSendRaw)
}

func testSendRaw(t *testing.T) {

	var addrsize int64
	var err error

	paramstring := strings.Split(t.Name(), "/")

	addrsize, _ = strconv.ParseInt(paramstring[1], 10, 0)
	log.Info("raw send test", "addrsize", addrsize)

	clients, err := setupNetwork(2, true)
	if err != nil {
		t.Fatal(err)
	}

	topic := "0xdeadbeef"

	var loaddrhex string
	err = clients[0].Call(&loaddrhex, "pss_baseAddr")
	if err != nil {
		t.Fatalf("rpc get node 1 baseaddr fail: %v", err)
	}
	loaddrhex = loaddrhex[:2+(addrsize*2)]
	var roaddrhex string
	err = clients[1].Call(&roaddrhex, "pss_baseAddr")
	if err != nil {
		t.Fatalf("rpc get node 2 baseaddr fail: %v", err)
	}
	roaddrhex = roaddrhex[:2+(addrsize*2)]

	time.Sleep(time.Millisecond * 500)

	// at this point we've verified that symkeys are saved and match on each peer
	// now try sending symmetrically encrypted message, both directions
	lmsgC := make(chan APIMsg)
	lctx, lcancel := context.WithTimeout(context.Background(), time.Second*10)
	defer lcancel()
	lsub, err := clients[0].Subscribe(lctx, "pss", lmsgC, "receive", topic)
	log.Trace("lsub", "id", lsub)
	defer lsub.Unsubscribe()
	rmsgC := make(chan APIMsg)
	rctx, rcancel := context.WithTimeout(context.Background(), time.Second*10)
	defer rcancel()
	rsub, err := clients[1].Subscribe(rctx, "pss", rmsgC, "receive", topic)
	log.Trace("rsub", "id", rsub)
	defer rsub.Unsubscribe()

	// send and verify delivery
	lmsg := []byte("plugh")
	err = clients[1].Call(nil, "pss_sendRaw", loaddrhex, topic, lmsg)
	if err != nil {
		t.Fatal(err)
	}
	select {
	case recvmsg := <-lmsgC:
		if !bytes.Equal(recvmsg.Msg, lmsg) {
			t.Fatalf("node 1 received payload mismatch: expected %v, got %v", lmsg, recvmsg)
		}
	case cerr := <-lctx.Done():
		t.Fatalf("test message (left) timed out: %v", cerr)
	}
	rmsg := []byte("xyzzy")
	err = clients[0].Call(nil, "pss_sendRaw", roaddrhex, topic, rmsg)
	if err != nil {
		t.Fatal(err)
	}
	select {
	case recvmsg := <-rmsgC:
		if !bytes.Equal(recvmsg.Msg, rmsg) {
			t.Fatalf("node 2 received payload mismatch: expected %x, got %v", rmsg, recvmsg.Msg)
		}
	case cerr := <-rctx.Done():
		t.Fatalf("test message (right) timed out: %v", cerr)
	}
}

// send symmetrically encrypted message between two directly connected peers
func TestSendSym(t *testing.T) {
	t.Run("32", testSendSym)
	t.Run("8", testSendSym)
	t.Run("0", testSendSym)
}

func testSendSym(t *testing.T) {

	// address hint size
	var addrsize int64
	var err error
	paramstring := strings.Split(t.Name(), "/")
	addrsize, _ = strconv.ParseInt(paramstring[1], 10, 0)
	log.Info("sym send test", "addrsize", addrsize)

	clients, err := setupNetwork(2, false)
	if err != nil {
		t.Fatal(err)
	}

	var topic string
	err = clients[0].Call(&topic, "pss_stringToTopic", "foo:42")
	if err != nil {
		t.Fatal(err)
	}

	var loaddrhex string
	err = clients[0].Call(&loaddrhex, "pss_baseAddr")
	if err != nil {
		t.Fatalf("rpc get node 1 baseaddr fail: %v", err)
	}
	loaddrhex = loaddrhex[:2+(addrsize*2)]
	var roaddrhex string
	err = clients[1].Call(&roaddrhex, "pss_baseAddr")
	if err != nil {
		t.Fatalf("rpc get node 2 baseaddr fail: %v", err)
	}
	roaddrhex = roaddrhex[:2+(addrsize*2)]

	// retrieve public key from pss instance
	// set this public key reciprocally
	var lpubkeyhex string
	err = clients[0].Call(&lpubkeyhex, "pss_getPublicKey")
	if err != nil {
		t.Fatalf("rpc get node 1 pubkey fail: %v", err)
	}
	var rpubkeyhex string
	err = clients[1].Call(&rpubkeyhex, "pss_getPublicKey")
	if err != nil {
		t.Fatalf("rpc get node 2 pubkey fail: %v", err)
	}

	time.Sleep(time.Millisecond * 500)

	// at this point we've verified that symkeys are saved and match on each peer
	// now try sending symmetrically encrypted message, both directions
	lmsgC := make(chan APIMsg)
	lctx, lcancel := context.WithTimeout(context.Background(), time.Second*10)
	defer lcancel()
	lsub, err := clients[0].Subscribe(lctx, "pss", lmsgC, "receive", topic)
	log.Trace("lsub", "id", lsub)
	defer lsub.Unsubscribe()
	rmsgC := make(chan APIMsg)
	rctx, rcancel := context.WithTimeout(context.Background(), time.Second*10)
	defer rcancel()
	rsub, err := clients[1].Subscribe(rctx, "pss", rmsgC, "receive", topic)
	log.Trace("rsub", "id", rsub)
	defer rsub.Unsubscribe()

	lrecvkey := network.RandomAddr().Over()
	rrecvkey := network.RandomAddr().Over()

	var lkeyids [2]string
	var rkeyids [2]string

	// manually set reciprocal symkeys
	err = clients[0].Call(&lkeyids, "psstest_setSymKeys", rpubkeyhex, lrecvkey, rrecvkey, defaultSymKeySendLimit, topic, roaddrhex)
	if err != nil {
		t.Fatal(err)
	}
	err = clients[1].Call(&rkeyids, "psstest_setSymKeys", lpubkeyhex, rrecvkey, lrecvkey, defaultSymKeySendLimit, topic, loaddrhex)
	if err != nil {
		t.Fatal(err)
	}

	// send and verify delivery
	lmsg := []byte("plugh")
	err = clients[1].Call(nil, "pss_sendSym", rkeyids[1], topic, hexutil.Encode(lmsg))
	if err != nil {
		t.Fatal(err)
	}
	select {
	case recvmsg := <-lmsgC:
		if !bytes.Equal(recvmsg.Msg, lmsg) {
			t.Fatalf("node 1 received payload mismatch: expected %v, got %v", lmsg, recvmsg)
		}
	case cerr := <-lctx.Done():
		t.Fatalf("test message timed out: %v", cerr)
	}
	rmsg := []byte("xyzzy")
	err = clients[0].Call(nil, "pss_sendSym", lkeyids[1], topic, hexutil.Encode(rmsg))
	if err != nil {
		t.Fatal(err)
	}
	select {
	case recvmsg := <-rmsgC:
		if !bytes.Equal(recvmsg.Msg, rmsg) {
			t.Fatalf("node 2 received payload mismatch: expected %x, got %v", rmsg, recvmsg.Msg)
		}
	case cerr := <-rctx.Done():
		t.Fatalf("test message timed out: %v", cerr)
	}
}

// send asymmetrically encrypted message between two directly connected peers
func TestSendAsym(t *testing.T) {
	t.Run("32", testSendAsym)
	t.Run("8", testSendAsym)
	t.Run("0", testSendAsym)
}

func testSendAsym(t *testing.T) {

	// address hint size
	var addrsize int64
	var err error
	paramstring := strings.Split(t.Name(), "/")
	addrsize, _ = strconv.ParseInt(paramstring[1], 10, 0)
	log.Info("asym send test", "addrsize", addrsize)

	clients, err := setupNetwork(2, false)
	if err != nil {
		t.Fatal(err)
	}

	var topic string
	err = clients[0].Call(&topic, "pss_stringToTopic", "foo:42")
	if err != nil {
		t.Fatal(err)
	}

	time.Sleep(time.Millisecond * 250)

	var loaddrhex string
	err = clients[0].Call(&loaddrhex, "pss_baseAddr")
	if err != nil {
		t.Fatalf("rpc get node 1 baseaddr fail: %v", err)
	}
	loaddrhex = loaddrhex[:2+(addrsize*2)]
	var roaddrhex string
	err = clients[1].Call(&roaddrhex, "pss_baseAddr")
	if err != nil {
		t.Fatalf("rpc get node 2 baseaddr fail: %v", err)
	}
	roaddrhex = roaddrhex[:2+(addrsize*2)]

	// retrieve public key from pss instance
	// set this public key reciprocally
	var lpubkey string
	err = clients[0].Call(&lpubkey, "pss_getPublicKey")
	if err != nil {
		t.Fatalf("rpc get node 1 pubkey fail: %v", err)
	}
	var rpubkey string
	err = clients[1].Call(&rpubkey, "pss_getPublicKey")
	if err != nil {
		t.Fatalf("rpc get node 2 pubkey fail: %v", err)
	}

	time.Sleep(time.Millisecond * 500) // replace with hive healthy code

	lmsgC := make(chan APIMsg)
	lctx, lcancel := context.WithTimeout(context.Background(), time.Second*10)
	defer lcancel()
	lsub, err := clients[0].Subscribe(lctx, "pss", lmsgC, "receive", topic)
	log.Trace("lsub", "id", lsub)
	defer lsub.Unsubscribe()
	rmsgC := make(chan APIMsg)
	rctx, rcancel := context.WithTimeout(context.Background(), time.Second*10)
	defer rcancel()
	rsub, err := clients[1].Subscribe(rctx, "pss", rmsgC, "receive", topic)
	log.Trace("rsub", "id", rsub)
	defer rsub.Unsubscribe()

	// store reciprocal public keys
	err = clients[0].Call(nil, "pss_setPeerPublicKey", rpubkey, topic, roaddrhex)
	if err != nil {
		t.Fatal(err)
	}
	err = clients[1].Call(nil, "pss_setPeerPublicKey", lpubkey, topic, loaddrhex)
	if err != nil {
		t.Fatal(err)
	}

	// send and verify delivery
	rmsg := []byte("xyzzy")
	err = clients[0].Call(nil, "pss_sendAsym", rpubkey, topic, hexutil.Encode(rmsg))
	if err != nil {
		t.Fatal(err)
	}
	select {
	case recvmsg := <-rmsgC:
		if !bytes.Equal(recvmsg.Msg, rmsg) {
			t.Fatalf("node 2 received payload mismatch: expected %v, got %v", rmsg, recvmsg.Msg)
		}
	case cerr := <-rctx.Done():
		t.Fatalf("test message timed out: %v", cerr)
	}
	lmsg := []byte("plugh")
	err = clients[1].Call(nil, "pss_sendAsym", lpubkey, topic, hexutil.Encode(lmsg))
	if err != nil {
		t.Fatal(err)
	}
	select {
	case recvmsg := <-lmsgC:
		if !bytes.Equal(recvmsg.Msg, lmsg) {
			t.Fatalf("node 1 received payload mismatch: expected %v, got %v", lmsg, recvmsg.Msg)
		}
	case cerr := <-lctx.Done():
		t.Fatalf("test message timed out: %v", cerr)
	}
}

type Job struct {
	Msg      []byte
	SendNode enode.ID
	RecvNode enode.ID
}

func worker(id int, jobs <-chan Job, rpcs map[enode.ID]*rpc.Client, pubkeys map[enode.ID]string, topic string) {
	for j := range jobs {
		rpcs[j.SendNode].Call(nil, "pss_sendAsym", pubkeys[j.RecvNode], topic, hexutil.Encode(j.Msg))
	}
}

func TestNetwork(t *testing.T) {
	t.Run("16/1000/4/sim", testNetwork)
}

// params in run name:
// nodes/msgs/addrbytes/adaptertype
// if adaptertype is exec uses execadapter, simadapter otherwise
func TestNetwork2000(t *testing.T) {
	//enableMetrics()

	if !*longrunning {
		t.Skip("run with --longrunning flag to run extensive network tests")
	}
	t.Run("3/2000/4/sim", testNetwork)
	t.Run("4/2000/4/sim", testNetwork)
	t.Run("8/2000/4/sim", testNetwork)
	t.Run("16/2000/4/sim", testNetwork)
}

func TestNetwork5000(t *testing.T) {
	//enableMetrics()

	if !*longrunning {
		t.Skip("run with --longrunning flag to run extensive network tests")
	}
	t.Run("3/5000/4/sim", testNetwork)
	t.Run("4/5000/4/sim", testNetwork)
	t.Run("8/5000/4/sim", testNetwork)
	t.Run("16/5000/4/sim", testNetwork)
}

func TestNetwork10000(t *testing.T) {
	//enableMetrics()

	if !*longrunning {
		t.Skip("run with --longrunning flag to run extensive network tests")
	}
	t.Run("3/10000/4/sim", testNetwork)
	t.Run("4/10000/4/sim", testNetwork)
	t.Run("8/10000/4/sim", testNetwork)
}

func testNetwork(t *testing.T) {
	paramstring := strings.Split(t.Name(), "/")
	nodecount, _ := strconv.ParseInt(paramstring[1], 10, 0)
	msgcount, _ := strconv.ParseInt(paramstring[2], 10, 0)
	addrsize, _ := strconv.ParseInt(paramstring[3], 10, 0)
	adapter := paramstring[4]

	log.Info("network test", "nodecount", nodecount, "msgcount", msgcount, "addrhintsize", addrsize)

	nodes := make([]enode.ID, nodecount)
	bzzaddrs := make(map[enode.ID]string, nodecount)
	rpcs := make(map[enode.ID]*rpc.Client, nodecount)
	pubkeys := make(map[enode.ID]string, nodecount)

	sentmsgs := make([][]byte, msgcount)
	recvmsgs := make([]bool, msgcount)
	nodemsgcount := make(map[enode.ID]int, nodecount)

	trigger := make(chan enode.ID)

	var a adapters.NodeAdapter
	if adapter == "exec" {
		dirname, err := ioutil.TempDir(".", "")
		if err != nil {
			t.Fatal(err)
		}
		a = adapters.NewExecAdapter(dirname)
	} else if adapter == "tcp" {
		a = adapters.NewTCPAdapter(newServices(false))
	} else if adapter == "sim" {
		a = adapters.NewSimAdapter(newServices(false))
	}
	net := simulations.NewNetwork(a, &simulations.NetworkConfig{
		ID: "0",
	})
	defer net.Shutdown()

	f, err := os.Open(fmt.Sprintf("testdata/snapshot_%d.json", nodecount))
	if err != nil {
		t.Fatal(err)
	}
	jsonbyte, err := ioutil.ReadAll(f)
	if err != nil {
		t.Fatal(err)
	}
	var snap simulations.Snapshot
	err = json.Unmarshal(jsonbyte, &snap)
	if err != nil {
		t.Fatal(err)
	}
	err = net.Load(&snap)
	if err != nil {
		//TODO: Fix p2p simulation framework to not crash when loading 32-nodes
		//t.Fatal(err)
	}

	time.Sleep(1 * time.Second)

	triggerChecks := func(trigger chan enode.ID, id enode.ID, rpcclient *rpc.Client, topic string) error {
		msgC := make(chan APIMsg)
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		sub, err := rpcclient.Subscribe(ctx, "pss", msgC, "receive", topic)
		if err != nil {
			t.Fatal(err)
		}
		go func() {
			defer sub.Unsubscribe()
			for {
				select {
				case recvmsg := <-msgC:
					idx, _ := binary.Uvarint(recvmsg.Msg)
					if !recvmsgs[idx] {
						log.Debug("msg recv", "idx", idx, "id", id)
						recvmsgs[idx] = true
						trigger <- id
					}
				case <-sub.Err():
					return
				}
			}
		}()
		return nil
	}

	var topic string
	for i, nod := range net.GetNodes() {
		nodes[i] = nod.ID()
		rpcs[nodes[i]], err = nod.Client()
		if err != nil {
			t.Fatal(err)
		}
		if topic == "" {
			err = rpcs[nodes[i]].Call(&topic, "pss_stringToTopic", "foo:42")
			if err != nil {
				t.Fatal(err)
			}
		}
		var pubkey string
		err = rpcs[nodes[i]].Call(&pubkey, "pss_getPublicKey")
		if err != nil {
			t.Fatal(err)
		}
		pubkeys[nod.ID()] = pubkey
		var addrhex string
		err = rpcs[nodes[i]].Call(&addrhex, "pss_baseAddr")
		if err != nil {
			t.Fatal(err)
		}
		bzzaddrs[nodes[i]] = addrhex
		err = triggerChecks(trigger, nodes[i], rpcs[nodes[i]], topic)
		if err != nil {
			t.Fatal(err)
		}
	}

	time.Sleep(1 * time.Second)

	// setup workers
	jobs := make(chan Job, 10)
	for w := 1; w <= 10; w++ {
		go worker(w, jobs, rpcs, pubkeys, topic)
	}

	time.Sleep(1 * time.Second)

	for i := 0; i < int(msgcount); i++ {
		sendnodeidx := rand.Intn(int(nodecount))
		recvnodeidx := rand.Intn(int(nodecount - 1))
		if recvnodeidx >= sendnodeidx {
			recvnodeidx++
		}
		nodemsgcount[nodes[recvnodeidx]]++
		sentmsgs[i] = make([]byte, 8)
		c := binary.PutUvarint(sentmsgs[i], uint64(i))
		if c == 0 {
			t.Fatal("0 byte message")
		}
		if err != nil {
			t.Fatal(err)
		}
		err = rpcs[nodes[sendnodeidx]].Call(nil, "pss_setPeerPublicKey", pubkeys[nodes[recvnodeidx]], topic, bzzaddrs[nodes[recvnodeidx]])
		if err != nil {
			t.Fatal(err)
		}

		jobs <- Job{
			Msg:      sentmsgs[i],
			SendNode: nodes[sendnodeidx],
			RecvNode: nodes[recvnodeidx],
		}
	}

	finalmsgcount := 0
	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()
outer:
	for i := 0; i < int(msgcount); i++ {
		select {
		case id := <-trigger:
			nodemsgcount[id]--
			finalmsgcount++
		case <-ctx.Done():
			log.Warn("timeout")
			break outer
		}
	}

	for i, msg := range recvmsgs {
		if !msg {
			log.Debug("missing message", "idx", i)
		}
	}
	t.Logf("%d of %d messages received", finalmsgcount, msgcount)

	if finalmsgcount != int(msgcount) {
		t.Fatalf("%d messages were not received", int(msgcount)-finalmsgcount)
	}

}

// check that in a network of a -> b -> c -> a
// a doesn't receive a sent message twice
func TestDeduplication(t *testing.T) {
	var err error

	clients, err := setupNetwork(3, false)
	if err != nil {
		t.Fatal(err)
	}

	var addrsize = 32
	var loaddrhex string
	err = clients[0].Call(&loaddrhex, "pss_baseAddr")
	if err != nil {
		t.Fatalf("rpc get node 1 baseaddr fail: %v", err)
	}
	loaddrhex = loaddrhex[:2+(addrsize*2)]
	var roaddrhex string
	err = clients[1].Call(&roaddrhex, "pss_baseAddr")
	if err != nil {
		t.Fatalf("rpc get node 2 baseaddr fail: %v", err)
	}
	roaddrhex = roaddrhex[:2+(addrsize*2)]
	var xoaddrhex string
	err = clients[2].Call(&xoaddrhex, "pss_baseAddr")
	if err != nil {
		t.Fatalf("rpc get node 3 baseaddr fail: %v", err)
	}
	xoaddrhex = xoaddrhex[:2+(addrsize*2)]

	log.Info("peer", "l", loaddrhex, "r", roaddrhex, "x", xoaddrhex)

	var topic string
	err = clients[0].Call(&topic, "pss_stringToTopic", "foo:42")
	if err != nil {
		t.Fatal(err)
	}

	time.Sleep(time.Millisecond * 250)

	// retrieve public key from pss instance
	// set this public key reciprocally
	var rpubkey string
	err = clients[1].Call(&rpubkey, "pss_getPublicKey")
	if err != nil {
		t.Fatalf("rpc get receivenode pubkey fail: %v", err)
	}

	time.Sleep(time.Millisecond * 500) // replace with hive healthy code

	rmsgC := make(chan APIMsg)
	rctx, cancel := context.WithTimeout(context.Background(), time.Second*1)
	defer cancel()
	rsub, err := clients[1].Subscribe(rctx, "pss", rmsgC, "receive", topic)
	log.Trace("rsub", "id", rsub)
	defer rsub.Unsubscribe()

	// store public key for recipient
	// zero-length address means forward to all
	// we have just two peers, they will be in proxbin, and will both receive
	err = clients[0].Call(nil, "pss_setPeerPublicKey", rpubkey, topic, "0x")
	if err != nil {
		t.Fatal(err)
	}

	// send and verify delivery
	rmsg := []byte("xyzzy")
	err = clients[0].Call(nil, "pss_sendAsym", rpubkey, topic, hexutil.Encode(rmsg))
	if err != nil {
		t.Fatal(err)
	}

	var receivedok bool
OUTER:
	for {
		select {
		case <-rmsgC:
			if receivedok {
				t.Fatalf("duplicate message received")
			}
			receivedok = true
		case <-rctx.Done():
			break OUTER
		}
	}
	if !receivedok {
		t.Fatalf("message did not arrive")
	}
}

// symmetric send performance with varying message sizes
func BenchmarkSymkeySend(b *testing.B) {
	b.Run(fmt.Sprintf("%d", 256), benchmarkSymKeySend)
	b.Run(fmt.Sprintf("%d", 1024), benchmarkSymKeySend)
	b.Run(fmt.Sprintf("%d", 1024*1024), benchmarkSymKeySend)
	b.Run(fmt.Sprintf("%d", 1024*1024*10), benchmarkSymKeySend)
	b.Run(fmt.Sprintf("%d", 1024*1024*100), benchmarkSymKeySend)
}

func benchmarkSymKeySend(b *testing.B) {
	msgsizestring := strings.Split(b.Name(), "/")
	if len(msgsizestring) != 2 {
		b.Fatalf("benchmark called without msgsize param")
	}
	msgsize, err := strconv.ParseInt(msgsizestring[1], 10, 0)
	if err != nil {
		b.Fatalf("benchmark called with invalid msgsize param '%s': %v", msgsizestring[1], err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	keys, err := wapi.NewKeyPair(ctx)
	privkey, err := w.GetPrivateKey(keys)
	ps := newTestPss(privkey, nil, nil)
	msg := make([]byte, msgsize)
	rand.Read(msg)
	topic := BytesToTopic([]byte("foo"))
	to := make(PssAddress, 32)
	copy(to[:], network.RandomAddr().Over())
	symkeyid, err := ps.GenerateSymmetricKey(topic, &to, true)
	if err != nil {
		b.Fatalf("could not generate symkey: %v", err)
	}
	symkey, err := ps.w.GetSymKey(symkeyid)
	if err != nil {
		b.Fatalf("could not retrieve symkey: %v", err)
	}
	ps.SetSymmetricKey(symkey, topic, &to, false)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ps.SendSym(symkeyid, topic, msg)
	}
}

// asymmetric send performance with varying message sizes
func BenchmarkAsymkeySend(b *testing.B) {
	b.Run(fmt.Sprintf("%d", 256), benchmarkAsymKeySend)
	b.Run(fmt.Sprintf("%d", 1024), benchmarkAsymKeySend)
	b.Run(fmt.Sprintf("%d", 1024*1024), benchmarkAsymKeySend)
	b.Run(fmt.Sprintf("%d", 1024*1024*10), benchmarkAsymKeySend)
	b.Run(fmt.Sprintf("%d", 1024*1024*100), benchmarkAsymKeySend)
}

func benchmarkAsymKeySend(b *testing.B) {
	msgsizestring := strings.Split(b.Name(), "/")
	if len(msgsizestring) != 2 {
		b.Fatalf("benchmark called without msgsize param")
	}
	msgsize, err := strconv.ParseInt(msgsizestring[1], 10, 0)
	if err != nil {
		b.Fatalf("benchmark called with invalid msgsize param '%s': %v", msgsizestring[1], err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	keys, err := wapi.NewKeyPair(ctx)
	privkey, err := w.GetPrivateKey(keys)
	ps := newTestPss(privkey, nil, nil)
	msg := make([]byte, msgsize)
	rand.Read(msg)
	topic := BytesToTopic([]byte("foo"))
	to := make(PssAddress, 32)
	copy(to[:], network.RandomAddr().Over())
	ps.SetPeerPublicKey(&privkey.PublicKey, topic, &to)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ps.SendAsym(common.ToHex(crypto.FromECDSAPub(&privkey.PublicKey)), topic, msg)
	}
}
func BenchmarkSymkeyBruteforceChangeaddr(b *testing.B) {
	for i := 100; i < 100000; i = i * 10 {
		for j := 32; j < 10000; j = j * 8 {
			b.Run(fmt.Sprintf("%d/%d", i, j), benchmarkSymkeyBruteforceChangeaddr)
		}
		//b.Run(fmt.Sprintf("%d", i), benchmarkSymkeyBruteforceChangeaddr)
	}
}

// decrypt performance using symkey cache, worst case
// (decrypt key always last in cache)
func benchmarkSymkeyBruteforceChangeaddr(b *testing.B) {
	keycountstring := strings.Split(b.Name(), "/")
	cachesize := int64(0)
	var ps *Pss
	if len(keycountstring) < 2 {
		b.Fatalf("benchmark called without count param")
	}
	keycount, err := strconv.ParseInt(keycountstring[1], 10, 0)
	if err != nil {
		b.Fatalf("benchmark called with invalid count param '%s': %v", keycountstring[1], err)
	}
	if len(keycountstring) == 3 {
		cachesize, err = strconv.ParseInt(keycountstring[2], 10, 0)
		if err != nil {
			b.Fatalf("benchmark called with invalid cachesize '%s': %v", keycountstring[2], err)
		}
	}
	pssmsgs := make([]*PssMsg, 0, keycount)
	var keyid string
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	keys, err := wapi.NewKeyPair(ctx)
	privkey, err := w.GetPrivateKey(keys)
	if cachesize > 0 {
		ps = newTestPss(privkey, nil, &PssParams{SymKeyCacheCapacity: int(cachesize)})
	} else {
		ps = newTestPss(privkey, nil, nil)
	}
	topic := BytesToTopic([]byte("foo"))
	for i := 0; i < int(keycount); i++ {
		to := make(PssAddress, 32)
		copy(to[:], network.RandomAddr().Over())
		keyid, err = ps.GenerateSymmetricKey(topic, &to, true)
		if err != nil {
			b.Fatalf("cant generate symkey #%d: %v", i, err)
		}
		symkey, err := ps.w.GetSymKey(keyid)
		if err != nil {
			b.Fatalf("could not retrieve symkey %s: %v", keyid, err)
		}
		wparams := &whisper.MessageParams{
			TTL:      defaultWhisperTTL,
			KeySym:   symkey,
			Topic:    whisper.TopicType(topic),
			WorkTime: defaultWhisperWorkTime,
			PoW:      defaultWhisperPoW,
			Payload:  []byte("xyzzy"),
			Padding:  []byte("1234567890abcdef"),
		}
		woutmsg, err := whisper.NewSentMessage(wparams)
		if err != nil {
			b.Fatalf("could not create whisper message: %v", err)
		}
		env, err := woutmsg.Wrap(wparams)
		if err != nil {
			b.Fatalf("could not generate whisper envelope: %v", err)
		}
		ps.Register(&topic, func(msg []byte, p *p2p.Peer, asymmetric bool, keyid string) error {
			return nil
		})
		pssmsgs = append(pssmsgs, &PssMsg{
			To:      to,
			Payload: env,
		})
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if err := ps.process(pssmsgs[len(pssmsgs)-(i%len(pssmsgs))-1]); err != nil {
			b.Fatalf("pss processing failed: %v", err)
		}
	}
}

func BenchmarkSymkeyBruteforceSameaddr(b *testing.B) {
	for i := 100; i < 100000; i = i * 10 {
		for j := 32; j < 10000; j = j * 8 {
			b.Run(fmt.Sprintf("%d/%d", i, j), benchmarkSymkeyBruteforceSameaddr)
		}
	}
}

// decrypt performance using symkey cache, best case
// (decrypt key always first in cache)
func benchmarkSymkeyBruteforceSameaddr(b *testing.B) {
	var keyid string
	var ps *Pss
	cachesize := int64(0)
	keycountstring := strings.Split(b.Name(), "/")
	if len(keycountstring) < 2 {
		b.Fatalf("benchmark called without count param")
	}
	keycount, err := strconv.ParseInt(keycountstring[1], 10, 0)
	if err != nil {
		b.Fatalf("benchmark called with invalid count param '%s': %v", keycountstring[1], err)
	}
	if len(keycountstring) == 3 {
		cachesize, err = strconv.ParseInt(keycountstring[2], 10, 0)
		if err != nil {
			b.Fatalf("benchmark called with invalid cachesize '%s': %v", keycountstring[2], err)
		}
	}
	addr := make([]PssAddress, keycount)
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	keys, err := wapi.NewKeyPair(ctx)
	privkey, err := w.GetPrivateKey(keys)
	if cachesize > 0 {
		ps = newTestPss(privkey, nil, &PssParams{SymKeyCacheCapacity: int(cachesize)})
	} else {
		ps = newTestPss(privkey, nil, nil)
	}
	topic := BytesToTopic([]byte("foo"))
	for i := 0; i < int(keycount); i++ {
		copy(addr[i], network.RandomAddr().Over())
		keyid, err = ps.GenerateSymmetricKey(topic, &addr[i], true)
		if err != nil {
			b.Fatalf("cant generate symkey #%d: %v", i, err)
		}

	}
	symkey, err := ps.w.GetSymKey(keyid)
	if err != nil {
		b.Fatalf("could not retrieve symkey %s: %v", keyid, err)
	}
	wparams := &whisper.MessageParams{
		TTL:      defaultWhisperTTL,
		KeySym:   symkey,
		Topic:    whisper.TopicType(topic),
		WorkTime: defaultWhisperWorkTime,
		PoW:      defaultWhisperPoW,
		Payload:  []byte("xyzzy"),
		Padding:  []byte("1234567890abcdef"),
	}
	woutmsg, err := whisper.NewSentMessage(wparams)
	if err != nil {
		b.Fatalf("could not create whisper message: %v", err)
	}
	env, err := woutmsg.Wrap(wparams)
	if err != nil {
		b.Fatalf("could not generate whisper envelope: %v", err)
	}
	ps.Register(&topic, func(msg []byte, p *p2p.Peer, asymmetric bool, keyid string) error {
		return nil
	})
	pssmsg := &PssMsg{
		To:      addr[len(addr)-1][:],
		Payload: env,
	}
	for i := 0; i < b.N; i++ {
		if err := ps.process(pssmsg); err != nil {
			b.Fatalf("pss processing failed: %v", err)
		}
	}
}

// setup simulated network with bzz/discovery and pss services.
// connects nodes in a circle
// if allowRaw is set, omission of builtin pss encryption is enabled (see PssParams)
func setupNetwork(numnodes int, allowRaw bool) (clients []*rpc.Client, err error) {
	nodes := make([]*simulations.Node, numnodes)
	clients = make([]*rpc.Client, numnodes)
	if numnodes < 2 {
		return nil, fmt.Errorf("Minimum two nodes in network")
	}
	adapter := adapters.NewSimAdapter(newServices(allowRaw))
	net := simulations.NewNetwork(adapter, &simulations.NetworkConfig{
		ID:             "0",
		DefaultService: "bzz",
	})
	for i := 0; i < numnodes; i++ {
		nodeconf := adapters.RandomNodeConfig()
		nodeconf.Services = []string{"bzz", pssProtocolName}
		nodes[i], err = net.NewNodeWithConfig(nodeconf)
		if err != nil {
			return nil, fmt.Errorf("error creating node 1: %v", err)
		}
		err = net.Start(nodes[i].ID())
		if err != nil {
			return nil, fmt.Errorf("error starting node 1: %v", err)
		}
		if i > 0 {
			err = net.Connect(nodes[i].ID(), nodes[i-1].ID())
			if err != nil {
				return nil, fmt.Errorf("error connecting nodes: %v", err)
			}
		}
		clients[i], err = nodes[i].Client()
		if err != nil {
			return nil, fmt.Errorf("create node 1 rpc client fail: %v", err)
		}
	}
	if numnodes > 2 {
		err = net.Connect(nodes[0].ID(), nodes[len(nodes)-1].ID())
		if err != nil {
			return nil, fmt.Errorf("error connecting first and last nodes")
		}
	}
	return clients, nil
}

func newServices(allowRaw bool) adapters.Services {
	stateStore := state.NewInmemoryStore()
	kademlias := make(map[enode.ID]*network.Kademlia)
	kademlia := func(id enode.ID) *network.Kademlia {
		if k, ok := kademlias[id]; ok {
			return k
		}
		params := network.NewKadParams()
		params.MinProxBinSize = 2
		params.MaxBinSize = 3
		params.MinBinSize = 1
		params.MaxRetries = 1000
		params.RetryExponent = 2
		params.RetryInterval = 1000000
		kademlias[id] = network.NewKademlia(id[:], params)
		return kademlias[id]
	}
	return adapters.Services{
		pssProtocolName: func(ctx *adapters.ServiceContext) (node.Service, error) {
			// execadapter does not exec init()
			initTest()

			ctxlocal, cancel := context.WithTimeout(context.Background(), time.Second)
			defer cancel()
			keys, err := wapi.NewKeyPair(ctxlocal)
			privkey, err := w.GetPrivateKey(keys)
			pssp := NewPssParams().WithPrivateKey(privkey)
			pssp.AllowRaw = allowRaw
			pskad := kademlia(ctx.Config.ID)
			ps, err := NewPss(pskad, pssp)
			if err != nil {
				return nil, err
			}

			ping := &Ping{
				OutC: make(chan bool),
				Pong: true,
			}
			p2pp := NewPingProtocol(ping)
			pp, err := RegisterProtocol(ps, &PingTopic, PingProtocol, p2pp, &ProtocolParams{Asymmetric: true})
			if err != nil {
				return nil, err
			}
			if useHandshake {
				SetHandshakeController(ps, NewHandshakeParams())
			}
			ps.Register(&PingTopic, pp.Handle)
			ps.addAPI(rpc.API{
				Namespace: "psstest",
				Version:   "0.3",
				Service:   NewAPITest(ps),
				Public:    false,
			})
			if err != nil {
				log.Error("Couldnt register pss protocol", "err", err)
				os.Exit(1)
			}
			pssprotocols[ctx.Config.ID.String()] = &protoCtrl{
				C:        ping.OutC,
				protocol: pp,
				run:      p2pp.Run,
			}
			return ps, nil
		},
		"bzz": func(ctx *adapters.ServiceContext) (node.Service, error) {
			addr := network.NewAddr(ctx.Config.Node())
			hp := network.NewHiveParams()
			hp.Discovery = false
			config := &network.BzzConfig{
				OverlayAddr:  addr.Over(),
				UnderlayAddr: addr.Under(),
				HiveParams:   hp,
			}
			return network.NewBzz(config, kademlia(ctx.Config.ID), stateStore, nil, nil), nil
		},
	}
}

func newTestPss(privkey *ecdsa.PrivateKey, kad *network.Kademlia, ppextra *PssParams) *Pss {
	nid := enode.PubkeyToIDV4(&privkey.PublicKey)
	// set up routing if kademlia is not passed to us
	if kad == nil {
		kp := network.NewKadParams()
		kp.MinProxBinSize = 3
		kad = network.NewKademlia(nid[:], kp)
	}

	// create pss
	pp := NewPssParams().WithPrivateKey(privkey)
	if ppextra != nil {
		pp.SymKeyCacheCapacity = ppextra.SymKeyCacheCapacity
	}
	ps, err := NewPss(kad, pp)
	if err != nil {
		return nil
	}
	ps.Start(nil)

	return ps
}

// API calls for test/development use
type APITest struct {
	*Pss
}

func NewAPITest(ps *Pss) *APITest {
	return &APITest{Pss: ps}
}

func (apitest *APITest) SetSymKeys(pubkeyid string, recvsymkey []byte, sendsymkey []byte, limit uint16, topic Topic, to PssAddress) ([2]string, error) {
	recvsymkeyid, err := apitest.SetSymmetricKey(recvsymkey, topic, &to, true)
	if err != nil {
		return [2]string{}, err
	}
	sendsymkeyid, err := apitest.SetSymmetricKey(sendsymkey, topic, &to, false)
	if err != nil {
		return [2]string{}, err
	}
	return [2]string{recvsymkeyid, sendsymkeyid}, nil
}

func (apitest *APITest) Clean() (int, error) {
	return apitest.Pss.cleanKeys(), nil
}

// enableMetrics is starting InfluxDB reporter so that we collect stats when running tests locally
func enableMetrics() {
	metrics.Enabled = true
	go influxdb.InfluxDBWithTags(metrics.DefaultRegistry, 1*time.Second, "http://localhost:8086", "metrics", "admin", "admin", "swarm.", map[string]string{
		"host": "test",
	})
}
